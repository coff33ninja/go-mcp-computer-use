package actions

import (
	"math"
	"testing"
)

func TestNormalizeDenormalizeRoundTrip(t *testing.T) {
	wn := &WindowNormalizer{
		WindowRect: WindowRect{Left: 100, Top: 50, Right: 1100, Bottom: 650, Width: 1000, Height: 600},
		DPIScale:   1.0,
	}

	tests := []struct {
		x, y, w, h int32
	}{
		{100, 50, 200, 100},
		{600, 350, 100, 50},
		{150, 75, 50, 25},
		{100, 50, 1000, 600},
	}

	for _, tc := range tests {
		norm := wn.Normalize(tc.x, tc.y, tc.w, tc.h)
		rx, ry, rw, rh := wn.Denormalize(norm)

		if rx != tc.x || ry != tc.y || rw != tc.w || rh != tc.h {
			t.Errorf("round trip (%d,%d,%d,%d): got (%d,%d,%d,%d)", tc.x, tc.y, tc.w, tc.h, rx, ry, rw, rh)
		}
	}
}

func TestNormalizeBounds(t *testing.T) {
	wn := &WindowNormalizer{
		WindowRect: WindowRect{Left: 200, Top: 100, Right: 1000, Bottom: 700, Width: 800, Height: 600},
		DPIScale:   1.0,
	}

	norm := wn.Normalize(200, 100, 0, 0)
	if norm.X != 0.0 || norm.Y != 0.0 {
		t.Errorf("top-left corner: got (%.4f,%.4f), want (0,0)", norm.X, norm.Y)
	}

	norm = wn.Normalize(1000, 700, 800, 600)
	if math.Abs(norm.X-1.0) > 0.001 || math.Abs(norm.Y-1.0) > 0.001 {
		t.Errorf("bottom-right corner: got (%.4f,%.4f), want (1,1)", norm.X, norm.Y)
	}

	norm = wn.Normalize(600, 400, 400, 300)
	if math.Abs(norm.X-0.5) > 0.001 || math.Abs(norm.Y-0.5) > 0.001 {
		t.Errorf("center: got (%.4f,%.4f), want (0.5,0.5)", norm.X, norm.Y)
	}
	if math.Abs(norm.W-0.5) > 0.001 || math.Abs(norm.H-0.5) > 0.001 {
		t.Errorf("center size: got (%.4f,%.4f), want (0.5,0.5)", norm.W, norm.H)
	}
}

func TestProportionalRegion(t *testing.T) {
	wn := &WindowNormalizer{
		WindowRect: WindowRect{Left: 0, Top: 0, Right: 1920, Bottom: 1080, Width: 1920, Height: 1080},
		DPIScale:   1.0,
	}

	x, y, w, h := wn.ProportionalRegion(0.05, 0.05, 0.95, 0.95)
	if x != 96 || y != 54 || w != 1728 || h != 972 {
		t.Errorf("5-95%% region on 1920x1080: got (%d,%d,%d,%d), want (96,54,1728,972)", x, y, w, h)
	}

	x, y, w, h = wn.ProportionalRegion(0, 0, 0.5, 0.5)
	if x != 0 || y != 0 || w != 960 || h != 540 {
		t.Errorf("0-50%% region: got (%d,%d,%d,%d), want (0,0,960,540)", x, y, w, h)
	}
}

func TestNormalizeElementRoundTrip(t *testing.T) {
	wn := &WindowNormalizer{
		WindowRect: WindowRect{Left: 100, Top: 50, Right: 1100, Bottom: 650, Width: 1000, Height: 600},
		DPIScale:   1.0,
	}

	orig := DetectedElement{
		Class:      "button",
		Confidence: 0.95,
		X:          300, Y: 200, W: 100, H: 50,
	}

	norm := wn.NormalizeElement(orig)
	back := wn.DenormalizeElement(norm)

	if orig.Class != back.Class || orig.Confidence != back.Confidence {
		t.Errorf("class/confidence mismatch: (%q,%.2f) vs (%q,%.2f)", orig.Class, orig.Confidence, back.Class, back.Confidence)
	}
	if orig.X != back.X || orig.Y != back.Y || orig.W != back.W || orig.H != back.H {
		t.Errorf("coord mismatch: (%d,%d,%d,%d) vs (%d,%d,%d,%d)", orig.X, orig.Y, orig.W, orig.H, back.X, back.Y, back.W, back.H)
	}
}

func TestWindowNormalizerInvalidRect(t *testing.T) {
	wn := &WindowNormalizer{
		WindowRect: WindowRect{Left: 0, Top: 0, Right: 0, Bottom: 0, Width: 0, Height: 0},
	}
	n := wn.Normalize(100, 100, 50, 50)
	if !math.IsInf(n.X, 1) || !math.IsInf(n.Y, 1) {
		t.Error("normalizing into zero-size window should produce +Inf")
	}

	_, _, _, _ = wn.Denormalize(NormalizedElement{X: 0.5, Y: 0.5, W: 0.1, H: 0.1})
}
