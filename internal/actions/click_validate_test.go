package actions

import "testing"

func TestClickValidationRoundTrip(t *testing.T) {
	// LastClickValidation should be nil when nothing was recorded.
	if v := LastClickValidation(); v != nil {
		t.Fatalf("expected nil last validation, got %+v", v)
	}

	in := &ClickValidation{
		Top: []ClassResult{{Label: "button", Confidence: 0.9, Index: 0}},
		Priors: &ClickValidationPriors{
			Class: "button", Samples: 4, PriorConfidence: 0.92,
			KnownConfidence: true, Frequency: 0.5,
		},
		MLMemory:    &ClickValidationML{Match: true, Confidence: 0.6, Samples: 3},
		WindowTitle: "Test Window",
		TotalMs:     42,
	}
	RecordClickValidation(in)
	got := LastClickValidation()
	if got == nil {
		t.Fatal("expected a validation after RecordClickValidation")
	}
	if got.Top[0].Label != "button" || got.WindowTitle != "Test Window" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Priors == nil || got.Priors.Class != "button" {
		t.Fatalf("priors not preserved: %+v", got.Priors)
	}
	if got.MLMemory == nil || !got.MLMemory.Match {
		t.Fatalf("ml_memory not preserved: %+v", got.MLMemory)
	}

	// Reading again should clear the slot.
	if v := LastClickValidation(); v != nil {
		t.Fatalf("expected nil after consuming, got %+v", v)
	}
}

func TestClassifyClickTargetShape(t *testing.T) {
	// classifyClickTarget is best-effort; on a live machine the classify block
	// should be non-empty, but under any failure it must still return a
	// non-nil validation rather than nil or an error.
	v := classifyClickTarget(200, 200)
	if v == nil {
		t.Fatal("classifyClickTarget must never return nil")
	}
	_ = v
}
