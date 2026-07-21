package transformer

import (
	"math"
	"testing"
)

func TestSoftmax_ProducesProbabilities(t *testing.T) {
	logits := []float64{1.0, 2.0, 3.0, 4.0}
	probs := Softmax(logits)
	if len(probs) != len(logits) {
		t.Fatalf("length mismatch: %d vs %d", len(probs), len(logits))
	}
	var sum float64
	for _, p := range probs {
		sum += p
		if p < 0 || p > 1 {
			t.Errorf("probability out of range: %f", p)
		}
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("probabilities don't sum to 1: %f", sum)
	}
}

func TestSoftmax_OrderPreserved(t *testing.T) {
	logits := []float64{0.1, 0.5, 0.9}
	probs := Softmax(logits)
	if probs[2] <= probs[1] || probs[1] <= probs[0] {
		t.Error("softmax should preserve ordering")
	}
}

func TestSoftmax_AllZeros(t *testing.T) {
	logits := []float64{0, 0, 0}
	probs := Softmax(logits)
	for _, p := range probs {
		if math.Abs(p-1.0/3.0) > 1e-6 {
			t.Errorf("equal logits should give equal probs, got %f", p)
		}
	}
}

func TestSoftmax_Empty(t *testing.T) {
	result := Softmax(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestSoftmax_LargeValues(t *testing.T) {
	logits := []float64{1000, 1001, 1002}
	probs := Softmax(logits)
	var sum float64
	for _, p := range probs {
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-4 {
		t.Errorf("sum off for large values: %f", sum)
	}
}

func TestSigmoid_Range(t *testing.T) {
	vals := []float64{-10, -1, 0, 1, 10}
	for _, v := range vals {
		s := Sigmoid(v)
		if s < 0 || s > 1 {
			t.Errorf("sigmoid(%f) = %f out of range", v, s)
		}
	}
}

func TestSigmoid_Monotone(t *testing.T) {
	if Sigmoid(0) >= Sigmoid(1) {
		t.Error("sigmoid should be monotone")
	}
	if Sigmoid(-1) >= Sigmoid(0) {
		t.Error("sigmoid should be monotone")
	}
}

func TestSigmoidSlice_Length(t *testing.T) {
	in := []float64{-1, 0, 1}
	out := SigmoidSlice(in)
	if len(out) != len(in) {
		t.Fatalf("length mismatch: %d vs %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != Sigmoid(in[i]) {
			t.Errorf("mismatch at %d", i)
		}
	}
}
