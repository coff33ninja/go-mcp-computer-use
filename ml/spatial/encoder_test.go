package spatial

import (
	"math"
	"testing"
)

func TestEncode_BasicFeatures(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(960, 540)
	if len(feats) != FeatureDim {
		t.Fatalf("expected %d features, got %d", FeatureDim, len(feats))
	}
	// center of screen: normX ≈ 0.5, normY ≈ 0.5
	if math.Abs(feats[0]-0.5) > 0.001 {
		t.Errorf("normX: expected ≈0.5, got %f", feats[0])
	}
	if math.Abs(feats[1]-0.5) > 0.001 {
		t.Errorf("normY: expected ≈0.5, got %f", feats[1])
	}
}

func TestEncode_DPIAware(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.5,
	})
	feats := enc.Encode(960, 540)
	// dpiAdj = norm * scale = 0.5 * 1.5 = 0.75
	if math.Abs(feats[2]-0.75) > 0.001 {
		t.Errorf("dpiAdjX: expected ≈0.75, got %f", feats[2])
	}
	if math.Abs(feats[3]-0.75) > 0.001 {
		t.Errorf("dpiAdjY: expected ≈0.75, got %f", feats[3])
	}
}

func TestEncode_RelativeToWindow(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		WindowX:      100,
		WindowY:      50,
		WindowWidth:  800,
		WindowHeight: 600,
	})
	feats := enc.Encode(500, 350)
	// relX = (500 - 100) / 800 = 0.5, relY = (350 - 50) / 600 = 0.5
	if math.Abs(feats[4]-0.5) > 0.001 {
		t.Errorf("relX: expected ≈0.5, got %f", feats[4])
	}
	if math.Abs(feats[5]-0.5) > 0.001 {
		t.Errorf("relY: expected ≈0.5, got %f", feats[5])
	}
}

func TestEncode_OutOfBounds(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(2000, 500)
	if feats[8] != 0.0 {
		t.Errorf("expected isValid=0 for out-of-bounds, got %f", feats[8])
	}
}

func TestEncode_InBounds(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(100, 100)
	if feats[8] != 1.0 {
		t.Errorf("expected isValid=1 for in-bounds, got %f", feats[8])
	}
}

func TestDecode_RoundTrip(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(960, 540)
	x, y := enc.Decode(feats)
	if x != 960 || y != 540 {
		t.Errorf("round-trip failed: expected (960,540), got (%d,%d)", x, y)
	}
}

func TestDecode_RoundTrip_OffCenter(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(333, 777)
	x, y := enc.Decode(feats)
	// allow ±1 for rounding
	if math.Abs(float64(x-333)) > 1 || math.Abs(float64(y-777)) > 1 {
		t.Errorf("round-trip off-center: expected ~(333,777), got (%d,%d)", x, y)
	}
}

func TestEncode_DefaultDPI(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     0, // should default to 1.0
	})
	feats := enc.Encode(960, 540)
	if math.Abs(feats[2]-0.5) > 0.001 {
		t.Errorf("default DPI scale should be 1.0, got dpiAdjX=%f", feats[2])
	}
}

func TestEncode_DefaultResolution(t *testing.T) {
	enc := NewEncoder(ScreenConfig{}) // all zeros → defaults
	feats := enc.Encode(960, 540)
	// defaults to 1920×1080
	if math.Abs(feats[0]-0.5) > 0.001 {
		t.Errorf("default resolution: normX expected ≈0.5, got %f", feats[0])
	}
}

func TestEncode_NoWindow(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		WindowWidth:  0,
		WindowHeight: 0,
	})
	feats := enc.Encode(500, 500)
	// relX and relY should be 0 when window size is 0
	if feats[4] != 0 || feats[5] != 0 {
		t.Errorf("expected relX=relY=0 for zero window, got (%f,%f)", feats[4], feats[5])
	}
}

func TestFeatureDim(t *testing.T) {
	enc := NewEncoder(ScreenConfig{})
	if enc.FeatureDimValue() != FeatureDim {
		t.Errorf("FeatureDimValue() = %d, want %d", enc.FeatureDimValue(), FeatureDim)
	}
}

func TestScreenConfig_RoundTrip(t *testing.T) {
	cfg := ScreenConfig{
		ScreenWidth:  2560,
		ScreenHeight: 1440,
		DPIScale:     1.25,
		WindowX:      100,
		WindowY:      200,
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	enc := NewEncoder(cfg)
	got := enc.ScreenConfig()
	// ScreenConfig holds a slice (Monitors), so compare field-by-field instead of ==/!=.
	if got.ScreenWidth != cfg.ScreenWidth || got.ScreenHeight != cfg.ScreenHeight ||
		got.DPIScale != cfg.DPIScale || got.WindowX != cfg.WindowX || got.WindowY != cfg.WindowY ||
		got.WindowWidth != cfg.WindowWidth || got.WindowHeight != cfg.WindowHeight ||
		len(got.Monitors) != len(cfg.Monitors) {
		t.Errorf("ScreenConfig round-trip failed: got %+v, want %+v", got, cfg)
	}
}

func TestEncode_MultiMonitor_PerMonitorDPI(t *testing.T) {
	// Two 1920x1080 monitors side by side on the virtual desktop: primary at
	// 100% scale, secondary (to the right) at 150% scale -- a realistic dual
	// monitor setup with mismatched DPI.
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  3840,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: true},
			{X: 1920, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.5, Primary: false},
		},
	})

	// Point on the primary monitor, dead center.
	primary := enc.Encode(960, 540)
	if math.Abs(primary[2]-0.25*1.0) > 0.001 {
		t.Errorf("primary dpiAdjX: expected normX(0.25)*1.0=0.25, got %f", primary[2])
	}
	if math.Abs(primary[6]-0.5) > 0.001 || math.Abs(primary[7]-0.5) > 0.001 {
		t.Errorf("primary monRel: expected (0.5,0.5), got (%f,%f)", primary[6], primary[7])
	}

	// Point on the secondary monitor, dead center -- same on-screen relative
	// position as above, but a different monitor with a different DPI.
	secondary := enc.Encode(1920+960, 540)
	wantNormX := (1920.0 + 960.0) / 3840.0
	wantDax := wantNormX * 1.5
	if math.Abs(secondary[2]-wantDax) > 0.001 {
		t.Errorf("secondary dpiAdjX: expected %f (using monitor's own 1.5x DPI), got %f", wantDax, secondary[2])
	}
	if math.Abs(secondary[6]-0.5) > 0.001 || math.Abs(secondary[7]-0.5) > 0.001 {
		t.Errorf("secondary monRel: expected (0.5,0.5) within its own monitor, got (%f,%f)", secondary[6], secondary[7])
	}
}

func TestEncode_CornerCoordinates(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	tests := []struct {
		name   string
		x, y   int
		wantNX float64
		wantNY float64
	}{
		{"top-left", 0, 0, 0, 0},
		{"top-right", 1919, 0, 1919.0 / 1920.0, 0},
		{"bottom-left", 0, 1079, 0, 1079.0 / 1080.0},
		{"bottom-right", 1919, 1079, 1919.0 / 1920.0, 1079.0 / 1080.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			feats := enc.Encode(tc.x, tc.y)
			if math.Abs(feats[0]-tc.wantNX) > 0.001 {
				t.Errorf("normX: got %f, want %f", feats[0], tc.wantNX)
			}
			if math.Abs(feats[1]-tc.wantNY) > 0.001 {
				t.Errorf("normY: got %f, want %f", feats[1], tc.wantNY)
			}
		})
	}
}

func TestEncode_LargeDPI(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  3840,
		ScreenHeight: 2160,
		DPIScale:     2.0,
	})
	feats := enc.Encode(1920, 1080)
	// normX = 0.5, dpiAdjX = 0.5 * 2.0 = 1.0
	if math.Abs(feats[2]-1.0) > 0.001 {
		t.Errorf("high-DPI dpiAdjX: expected ≈1.0, got %f", feats[2])
	}
}

// --- Multi-monitor tests ---

func TestEncode_ThreeMonitors_LShape(t *testing.T) {
	// L-shape: primary at (0,0), secondary to the right, tertiary below primary.
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  3840,
		ScreenHeight: 2160,
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: true},
			{X: 1920, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.5, Primary: false},
			{X: 0, Y: 1080, Width: 1920, Height: 1080, DPIScale: 1.25, Primary: false},
		},
	})

	// Point on primary (center)
	f1 := enc.Encode(960, 540)
	if math.Abs(f1[6]-0.5) > 0.001 || math.Abs(f1[7]-0.5) > 0.001 {
		t.Errorf("L primary monRel: expected (0.5,0.5), got (%f,%f)", f1[6], f1[7])
	}

	// Point on secondary (center)
	f2 := enc.Encode(2880, 540)
	if math.Abs(f2[6]-0.5) > 0.001 || math.Abs(f2[7]-0.5) > 0.001 {
		t.Errorf("L secondary monRel: expected (0.5,0.5), got (%f,%f)", f2[6], f2[7])
	}
	// secondary DPI is 1.5
	if math.Abs(f2[2]-((1920+960)/3840.0)*1.5) > 0.001 {
		t.Errorf("L secondary dpiAdjX: expected normX*1.5, got %f", f2[2])
	}

	// Point on tertiary (center)
	f3 := enc.Encode(960, 1620)
	if math.Abs(f3[6]-0.5) > 0.001 || math.Abs(f3[7]-0.5) > 0.001 {
		t.Errorf("L tertiary monRel: expected (0.5,0.5), got (%f,%f)", f3[6], f3[7])
	}
	if math.Abs(f3[2]-0.25*1.25) > 0.001 {
		t.Errorf("L tertiary dpiAdjX: expected 0.25*1.25, got %f", f3[2])
	}
}

func TestEncode_ThreeMonitors_Stacked_Vertical(t *testing.T) {
	// Three monitors stacked: primary at (0,0), one below it, one above (negative Y).
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 3240, // spans all three vertically
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: true},    // middle
			{X: 0, Y: 1080, Width: 1920, Height: 1080, DPIScale: 1.25, Primary: false}, // bottom
			{X: 0, Y: -1080, Width: 1920, Height: 1080, DPIScale: 1.5, Primary: false}, // top (above)
		},
	})

	// Point on the top monitor (above primary) — center is at virtual (960, -540)
	f1 := enc.Encode(960, -540)
	// normX = 960/1920 = 0.5, monitor DPI = 1.5 → dpiAdjX = 0.5 * 1.5 = 0.75
	if math.Abs(f1[6]-0.5) > 0.001 || math.Abs(f1[7]-0.5) > 0.001 {
		t.Errorf("stacked top monRel: expected (0.5,0.5), got (%f,%f)", f1[6], f1[7])
	}
	if math.Abs(f1[2]-0.75) > 0.001 {
		t.Errorf("stacked top dpiAdjX: expected 0.75, got %f", f1[2])
	}
	// normY = -540/3240 is negative → isValid = 0
	if f1[8] != 0.0 {
		t.Errorf("stacked top: expected isValid=0 for negative normY, got %f", f1[8])
	}

	// Point on the bottom monitor center (960, 1620)
	f2 := enc.Encode(960, 1620)
	// normX = 960/1920 = 0.5, monitor DPI = 1.25 → dpiAdjX = 0.5 * 1.25 = 0.625
	if math.Abs(f2[6]-0.5) > 0.001 || math.Abs(f2[7]-0.5) > 0.001 {
		t.Errorf("stacked bottom monRel: expected (0.5,0.5), got (%f,%f)", f2[6], f2[7])
	}
	if math.Abs(f2[2]-0.625) > 0.001 {
		t.Errorf("stacked bottom dpiAdjX: expected 0.625, got %f", f2[2])
	}
}

func TestEncode_MonitorAt_Fallback(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  5760,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: true},
			{X: 1920, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.5, Primary: false},
		},
	})

	// Point outside both monitors — falls back to synthetic 5760x1080 screen
	f := enc.Encode(5000, 500)
	// monRel is relative to fallback monitor (0,0)-(5760,1080): 5000/5760 ≈ 0.868056
	wantMonRelX := 5000.0 / 5760.0
	if math.Abs(f[6]-wantMonRelX) > 0.001 {
		t.Errorf("fallback monRelX: expected %f, got %f", wantMonRelX, f[6])
	}
	// DPI should be the encoder's fallback DPI (1.0)
	wantDax := (5000.0 / 5760.0) * 1.0
	if math.Abs(f[2]-wantDax) > 0.001 {
		t.Errorf("fallback dpiAdjX: expected %f, got %f", wantDax, f[2])
	}
}

func TestEncode_NoMonitors_DefaultsToWholeScreen(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.5,
	})
	f := enc.Encode(960, 540)
	// no monitors → monitorAt returns whole-screen with fallback DPI
	if math.Abs(f[2]-(960.0/1920.0)*1.5) > 0.001 {
		t.Errorf("no-monitors dpiAdjX: expected 0.5*1.5=0.75, got %f", f[2])
	}
	if math.Abs(f[6]-0.5) > 0.001 || math.Abs(f[7]-0.5) > 0.001 {
		t.Errorf("no-monitors monRel: expected (0.5,0.5), got (%f,%f)", f[6], f[7])
	}
}

func TestEncode_MonitorNegativeCoords(t *testing.T) {
	// Virtual desktop with negative origin: monitors at (-1920,0) and (0,0).
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  3840,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: -1920, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: false},
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.25, Primary: true},
		},
	})

	// Point on the left monitor (negative virtual coords)
	f := enc.Encode(-960, 540)
	if math.Abs(f[6]-0.5) > 0.001 || math.Abs(f[7]-0.5) > 0.001 {
		t.Errorf("neg-coord monRel: expected (0.5,0.5), got (%f,%f)", f[6], f[7])
	}
	// normX = -960/3840 = -0.25, dpiAdjX = -0.25 * 1.0 = -0.25
	if math.Abs(f[2]-(-0.25)) > 0.001 {
		t.Errorf("neg-coord dpiAdjX: expected -0.25, got %f", f[2])
	}
}

func TestEncode_ThreeMonitors_AllPrimary(t *testing.T) {
	// Only one should be primary
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  5760,
		ScreenHeight: 1080,
		DPIScale:     1.0,
		Monitors: []MonitorInfo{
			{X: 0, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: true},
			{X: 1920, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: false},
			{X: 3840, Y: 0, Width: 1920, Height: 1080, DPIScale: 1.0, Primary: false},
		},
	})

	// Each monitor's center should have monRel (0.5, 0.5)
	for _, tc := range []struct {
		x, y int
		name string
	}{
		{960, 540, "monitor-0"},
		{2880, 540, "monitor-1"},
		{4800, 540, "monitor-2"},
	} {
		f := enc.Encode(tc.x, tc.y)
		if math.Abs(f[6]-0.5) > 0.001 || math.Abs(f[7]-0.5) > 0.001 {
			t.Errorf("%s monRel: expected (0.5,0.5), got (%f,%f)", tc.name, f[6], f[7])
		}
	}
}
