package transformer

import (
	"math"
	"testing"
)

func TestRealTransformer_ForwardProducesNonZeroOutput(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	tokens := [][]int{{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	coords := [][]float64{make([]float64, cfg.CoordDim)}
	logits, err := m.Forward(tokens, coords)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}
	nonZero := 0
	for _, v := range logits[0] {
		if math.Abs(v) > 1e-8 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("expected at least one non-zero output from Forward")
	}
}

func TestRealTransformer_DifferentInputsProduceDifferentOutputs(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	tok1 := [][]int{{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	tok2 := [][]int{{50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	coord := [][]float64{make([]float64, cfg.CoordDim)}

	out1, _ := m.Forward(tok1, coord)
	out2, _ := m.Forward(tok2, coord)

	same := true
	for i := range out1[0] {
		if math.Abs(out1[0][i]-out2[0][i]) > 1e-8 {
			same = false
			break
		}
	}
	if same {
		t.Error("different token inputs should produce different outputs")
	}
}

func TestRealTransformer_TrainingReducesLoss(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}

	tokens := [][]int{{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	coords := [][]float64{make([]float64, cfg.CoordDim)}
	target := make([]float64, cfg.OutputDim)
	target[0] = 1.0 // target one-hot

	// measure initial loss
	logits, _ := m.Forward(tokens, coords)
	initialLoss := mseLoss(logits[0], target)

	// train for several steps
	for i := 0; i < 20; i++ {
		logits, err = m.Forward(tokens, coords)
		if err != nil {
			t.Fatalf("Forward failed: %v", err)
		}
		err = m.BackwardWithTarget(target, 0.01)
		if err != nil {
			t.Fatalf("Backward failed at step %d: %v", i, err)
		}
	}

	// measure final loss
	logits, _ = m.Forward(tokens, coords)
	finalLoss := mseLoss(logits[0], target)

	if finalLoss >= initialLoss {
		t.Errorf("loss did not decrease: initial=%.6f final=%.6f", initialLoss, finalLoss)
	}
	t.Logf("loss: %.6f -> %.6f", initialLoss, finalLoss)
}

func TestRealTransformer_ParametersMatchConfig(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	params := m.Parameters()
	if len(params) == 0 {
		t.Error("expected non-zero parameter count")
	}
	if len(params) > 1_000_000 {
		t.Errorf("parameter count too large: %d", len(params))
	}
	t.Logf("real transformer params: %d", len(params))
}

func TestRealTransformer_SaveLoadRoundTrip(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}

	// train a few steps so weights are meaningful
	tokens := [][]int{{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	coords := [][]float64{make([]float64, cfg.CoordDim)}
	target := make([]float64, cfg.OutputDim)
	target[0] = 1.0
	for i := 0; i < 5; i++ {
		m.Forward(tokens, coords)
		m.BackwardWithTarget(target, 0.01)
	}

	dir := t.TempDir()
	path := dir + "/model_real.bin"
	if err := m.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	m2, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	if err := m2.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// outputs should match
	out1, _ := m.Forward(tokens, coords)
	out2, _ := m2.Forward(tokens, coords)
	for i := range out1[0] {
		if math.Abs(out1[0][i]-out2[0][i]) > 1e-6 {
			t.Errorf("output mismatch at dim %d after Save/Load: %f vs %f", i, out1[0][i], out2[0][i])
		}
	}
}

func TestRealTransformer_LoadParametersRoundTrip(t *testing.T) {
	cfg := newTestConfig()
	m, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	params := m.Parameters()

	m2, err := NewReal(cfg)
	if err != nil {
		t.Fatalf("NewReal failed: %v", err)
	}
	if err := m2.LoadParameters(params); err != nil {
		t.Fatalf("LoadParameters failed: %v", err)
	}
	params2 := m2.Parameters()
	if len(params) != len(params2) {
		t.Fatalf("param count mismatch: %d vs %d", len(params), len(params2))
	}
	for i := range params {
		if math.Abs(params[i]-params2[i]) > 1e-6 {
			t.Errorf("param[%d] mismatch: %f vs %f", i, params[i], params2[i])
		}
	}
}

func mseLoss(pred, target []float64) float64 {
	var sum float64
	for i := range pred {
		d := pred[i] - target[i]
		sum += d * d
	}
	return sum / float64(len(pred))
}
