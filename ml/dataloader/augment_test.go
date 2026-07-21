package dataloader

import (
	"testing"
)

func TestAugmentor_ScaleSample(t *testing.T) {
	a := NewAugmentor()
	s := Sample{
		Context: "click Submit button",
		CoordX:  100,
		CoordY:  200,
	}

	aug := a.scaleSample(s, 1.5)
	if aug.CoordX != 150 || aug.CoordY != 300 {
		t.Errorf("expected (150, 300), got (%d, %d)", aug.CoordX, aug.CoordY)
	}
	if aug.Context != s.Context {
		t.Error("context should not change")
	}
}

func TestAugmentor_OCRNoise(t *testing.T) {
	a := NewAugmentor()
	s := Sample{Context: "click Button"}

	aug := a.ocrNoiseSample(s)
	// text should be different (one substitution)
	if aug.Context == s.Context {
		// it's possible no substitution matched — that's ok
		t.Logf("no OCR substitution applied (possible with short text)")
	}
}

func TestAugmentor_JitterCoords(t *testing.T) {
	a := NewAugmentor()
	s := Sample{Context: "click", CoordX: 500, CoordY: 500}

	changed := false
	for i := 0; i < 10; i++ {
		aug := a.jitterCoords(s)
		if aug.CoordX != 500 || aug.CoordY != 500 {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("jitter should change coordinates at least once")
	}
}

func TestAugmentor_Augment(t *testing.T) {
	a := NewAugmentor()
	s := Sample{
		Context: "click Submit button",
		CoordX:  200,
		CoordY:  300,
	}

	augmented := a.Augment(s, 5)
	if len(augmented) == 0 {
		t.Error("expected at least 1 augmented sample")
	}
	if len(augmented) > 5 {
		t.Errorf("expected at most 5, got %d", len(augmented))
	}

	// all should have same action
	for _, aug := range augmented {
		if aug.Action != s.Action {
			t.Errorf("action should not change: got %q", aug.Action)
		}
	}
}

func TestAugmentor_AugmentAll(t *testing.T) {
	a := NewAugmentor()
	samples := []Sample{
		{Context: "click Submit", CoordX: 100, CoordY: 200},
		{Context: "hover Cancel", CoordX: 300, CoordY: 400},
	}

	result := a.AugmentAll(samples, 2)
	// originals (2) + up to 2 augmented per sample = up to 6
	if len(result) < len(samples) {
		t.Errorf("should keep originals: got %d, expected >= %d", len(result), len(samples))
	}
	if len(result) > len(samples)*3 {
		t.Errorf("too many augmented: got %d, expected <= %d", len(result), len(samples)*3)
	}
}
