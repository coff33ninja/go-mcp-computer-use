package spatial

import "math"

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
	featureDim int
}

// ScreenConfig describes the display geometry for coordinate normalization.
type ScreenConfig struct {
	ScreenWidth  int
	ScreenHeight int
	DPIScale     float64 // 1.0 = 100%, 1.5 = 150%, etc.
	WindowX      int     // window origin X on screen
	WindowY      int     // window origin Y on screen
	WindowWidth  int
	WindowHeight int
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
		featureDim: FeatureDim,
	}
}

// FeatureDim is the number of output features per coordinate.
// [normX, normY, dpiAdjX, dpiAdjY, relX, relY, isValid]
const FeatureDim = 7

// Encode converts a raw pixel coordinate into a normalized feature vector.
// Returns [normX, normY, dpiAdjX, dpiAdjY, relX, relY, isValid].
func (e *Encoder) Encode(x, y int) []float64 {
	nx := float64(x) / e.screenW
	ny := float64(y) / e.screenH
	dax := nx * e.dpiScale
	day := ny * e.dpiScale

	var rx, ry float64
	if e.windowW > 0 && e.windowH > 0 {
		rx = (float64(x) - e.windowX) / e.windowW
		ry = (float64(y) - e.windowY) / e.windowH
	}

	isValid := 0.0
	if nx >= 0 && nx <= 1 && ny >= 0 && ny <= 1 {
		isValid = 1.0
	}

	return []float64{nx, ny, dax, day, rx, ry, isValid}
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
	}
}

// FeatureDim returns the dimensionality of the output feature vector.
func (e *Encoder) FeatureDimValue() int {
	return e.featureDim
}
