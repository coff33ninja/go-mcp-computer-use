package spatial

import "math"

// MonitorInfo describes one physical display: its rect in virtual-desktop
// pixel coordinates (origins can be negative/offset on multi-monitor setups)
// and its own DPI scale. Monitors can run at different DPI from each other,
// so a single screen-wide DPIScale is not enough once more than one is in play.
type MonitorInfo struct {
	X, Y          int
	Width, Height int
	DPIScale      float64 // 1.0 = 100%, 1.5 = 150%, etc.
	Primary       bool
}

// Encoder transforms raw pixel coordinates into DPI-aware normalized
// features suitable for transformer input.
type Encoder struct {
	screenW    float64
	screenH    float64
	dpiScale   float64
	windowX    float64
	windowY    float64
	windowW    float64
	windowH    float64
	monitors   []MonitorInfo
	featureDim int
}

// ScreenConfig describes the display geometry for coordinate normalization.
type ScreenConfig struct {
	ScreenWidth  int
	ScreenHeight int
	DPIScale     float64 // fallback DPI; used when Monitors is empty or a point falls outside all of them
	WindowX      int     // window origin X on screen
	WindowY      int     // window origin Y on screen
	WindowWidth  int
	WindowHeight int
	Monitors     []MonitorInfo // optional per-monitor rects+DPI; enables correct DPI and monitor-relative features on multi-monitor setups
}

// NewEncoder creates a spatial encoder from the current display configuration.
func NewEncoder(cfg ScreenConfig) *Encoder {
	if cfg.DPIScale == 0 {
		cfg.DPIScale = 1.0
	}
	if cfg.ScreenWidth == 0 {
		cfg.ScreenWidth = 1920
	}
	if cfg.ScreenHeight == 0 {
		cfg.ScreenHeight = 1080
	}
	return &Encoder{
		screenW:    float64(cfg.ScreenWidth),
		screenH:    float64(cfg.ScreenHeight),
		dpiScale:   cfg.DPIScale,
		windowX:    float64(cfg.WindowX),
		windowY:    float64(cfg.WindowY),
		windowW:    float64(cfg.WindowWidth),
		windowH:    float64(cfg.WindowHeight),
		monitors:   cfg.Monitors,
		featureDim: FeatureDim,
	}
}

// monitorAt returns the monitor containing (x, y). If no monitor list was
// configured, or none contains the point, it falls back to a synthetic
// monitor spanning the whole configured screen with the encoder's single
// DPIScale — this reproduces the old single-screen behavior exactly.
func (e *Encoder) monitorAt(x, y int) MonitorInfo {
	for _, m := range e.monitors {
		if x >= m.X && x < m.X+m.Width && y >= m.Y && y < m.Y+m.Height {
			return m
		}
	}
	return MonitorInfo{X: 0, Y: 0, Width: int(e.screenW), Height: int(e.screenH), DPIScale: e.dpiScale}
}

// FeatureDim is the number of output features per coordinate.
// [normX, normY, dpiAdjX, dpiAdjY, relX, relY, monRelX, monRelY, isValid, distFromCenter, isCenter, isEdge, windowAspect, isDialog]
const FeatureDim = 14

// Encode converts a raw pixel coordinate into a normalized feature vector.
// Returns [normX, normY, dpiAdjX, dpiAdjY, relX, relY, monRelX, monRelY, isValid, distFromCenter, isCenter, isEdge, windowAspect, isDialog].
//
// Four distinct positional encodings feed the model: normX/Y (position on
// the whole virtual desktop), dpiAdjX/Y (DPI-scaled position, using the
// specific monitor the point falls on rather than one desktop-wide DPI),
// relX/Y (position within the target window), and monRelX/Y (position
// within the specific monitor the point falls on, distinct from the window).
func (e *Encoder) Encode(x, y int) []float64 {
	nx := float64(x) / e.screenW
	ny := float64(y) / e.screenH

	mon := e.monitorAt(x, y)
	dax := nx * mon.DPIScale
	day := ny * mon.DPIScale

	var rx, ry float64
	if e.windowW > 0 && e.windowH > 0 {
		rx = (float64(x) - e.windowX) / e.windowW
		ry = (float64(y) - e.windowY) / e.windowH
	}

	var monRelX, monRelY float64
	if mon.Width > 0 && mon.Height > 0 {
		monRelX = (float64(x) - float64(mon.X)) / float64(mon.Width)
		monRelY = (float64(y) - float64(mon.Y)) / float64(mon.Height)
	}

	isValid := 0.0
	if nx >= 0 && nx <= 1 && ny >= 0 && ny <= 1 {
		isValid = 1.0
	}

	// distance from window center (0 = at center, 1 = at corner)
	distFromCenter := 0.0
	if e.windowW > 0 && e.windowH > 0 {
		cx := e.windowX + e.windowW/2
		cy := e.windowY + e.windowH/2
		dx := (float64(x) - cx) / e.windowW
		dy := (float64(y) - cy) / e.windowH
		distFromCenter = math.Sqrt(dx*dx+dy*dy) * 2 // normalize to 0-1 range
		if distFromCenter > 1.0 {
			distFromCenter = 1.0
		}
	}

	// isCenter: 1.0 if within 20% of window center
	isCenter := 0.0
	if rx >= 0.3 && rx <= 0.7 && ry >= 0.3 && ry <= 0.7 {
		isCenter = 1.0
	}

	// isEdge: 1.0 if within 10% of window edge
	isEdge := 0.0
	if rx < 0.1 || rx > 0.9 || ry < 0.1 || ry > 0.9 {
		isEdge = 1.0
	}

	// windowAspect: aspect ratio normalized to 0-1 (0.5 = square, <0.5 = tall, >0.5 = wide)
	windowAspect := 0.5
	if e.windowH > 0 {
		windowAspect = e.windowW / (e.windowW + e.windowH) // normalize to [0,1]
	}

	// isDialog: 1.0 if window is small (likely a dialog)
	isDialog := 0.0
	if e.windowW > 0 && e.windowH > 0 {
		if e.windowW < e.screenW*0.5 && e.windowH < e.screenH*0.5 {
			isDialog = 1.0
		}
	}

	return []float64{nx, ny, dax, day, rx, ry, monRelX, monRelY, isValid, distFromCenter, isCenter, isEdge, windowAspect, isDialog}
}

// Decode reverses the encoding to recover approximate pixel coordinates.
// Uses the normX/normY components.
func (e *Encoder) Decode(features []float64) (x, y int) {
	if len(features) < 2 {
		return 0, 0
	}
	x = int(math.Round(features[0] * e.screenW))
	y = int(math.Round(features[1] * e.screenH))
	return
}

// ScreenConfig returns the stored screen configuration.
func (e *Encoder) ScreenConfig() ScreenConfig {
	return ScreenConfig{
		ScreenWidth:  int(e.screenW),
		ScreenHeight: int(e.screenH),
		DPIScale:     e.dpiScale,
		WindowX:      int(e.windowX),
		WindowY:      int(e.windowY),
		WindowWidth:  int(e.windowW),
		WindowHeight: int(e.windowH),
		Monitors:     e.monitors,
	}
}

// FeatureDim returns the dimensionality of the output feature vector.
func (e *Encoder) FeatureDimValue() int {
	return e.featureDim
}
