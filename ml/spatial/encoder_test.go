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
	if feats[6] != 0.0 {
		t.Errorf("expected isValid=0 for out-of-bounds, got %f", feats[6])
	}
}

func TestEncode_InBounds(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	feats := enc.Encode(100, 100)
	if feats[6] != 1.0 {
		t.Errorf("expected isValid=1 for in-bounds, got %f", feats[6])
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
	if got != cfg {
		t.Errorf("ScreenConfig round-trip failed: got %+v", got)
	}
}

func TestEncode_CornerCoordinates(t *testing.T) {
	enc := NewEncoder(ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	tests := []struct {
		name    string
		x, y    int
		wantNX  float64
		wantNY  float64
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
