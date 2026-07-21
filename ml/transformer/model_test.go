package transformer

import (
	"math"
	"testing"
)

func newTestConfig() Config {
	return Config{
		VocabSize: 100,
		MaxLen:    16,
		EmbedDim:  32,
		NumHeads:  2,
		NumLayers: 2,
		FFNDim:    64,
		CoordDim:  7,
		OutputDim: 10,
	}
}

func TestNewModel_ParameterCount(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	params := model.Parameters()
	// should have non-zero parameters
	if len(params) == 0 {
		t.Error("expected non-zero parameter count")
	}
	// should be reasonable (not millions for a test config)
	if len(params) > 1_000_000 {
		t.Errorf("parameter count too large: %d", len(params))
	}
	t.Logf("parameter count: %d", len(params))
}

func TestForward_OutputShape(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	tokens := [][]int{
		{1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	coords := [][]float64{make([]float64, cfg.CoordDim)}
	logits, err := model.Forward(tokens, coords)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}
	if len(logits) != 1 {
		t.Errorf("expected 1 row, got %d", len(logits))
	}
	if len(logits[0]) != cfg.OutputDim {
		t.Errorf("expected %d output dims, got %d", cfg.OutputDim, len(logits[0]))
	}
}

func TestForward_AllZeros_ProducesValidOutput(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	tokens := make([][]int, 1)
	tokens[0] = make([]int, cfg.MaxLen)
	coords := make([][]float64, 1)
	coords[0] = make([]float64, cfg.CoordDim)
	logits, err := model.Forward(tokens, coords)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}
	for i, v := range logits[0] {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Errorf("logit[%d] is NaN/Inf: %f", i, v)
		}
	}
}

func TestBackward_UpdatesWeights(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	before := make([]float64, len(model.Parameters()))
	copy(before, model.Parameters())

	tokens := [][]int{{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	coords := [][]float64{make([]float64, cfg.CoordDim)}
	_, err = model.Forward(tokens, coords)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}
	if err := model.Backward(1.0, 0.01); err != nil {
		t.Fatalf("Backward failed: %v", err)
	}

	after := model.Parameters()
	changed := false
	for i := range before {
		if before[i] != after[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("parameters did not change after Backward")
	}
}

func TestLoadParameters_RoundTrip(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	params := model.Parameters()

	model2, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := model2.LoadParameters(params); err != nil {
		t.Fatalf("LoadParameters failed: %v", err)
	}
	params2 := model2.Parameters()
	if len(params) != len(params2) {
		t.Fatalf("parameter count mismatch: %d vs %d", len(params), len(params2))
	}
	for i := range params {
		if math.Abs(params[i]-params2[i]) > 1e-6 {
			t.Errorf("param[%d] mismatch: %f vs %f", i, params[i], params2[i])
		}
	}
}

func TestLoadParameters_WrongSize(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	err = model.LoadParameters([]float64{1, 2, 3})
	if err == nil {
		t.Error("expected error for wrong-sized parameter vector")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	dir := t.TempDir()
	path := dir + "/model.bin"
	if err := model.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	model2, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := model2.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	params1 := model.Parameters()
	params2 := model2.Parameters()
	if len(params1) != len(params2) {
		t.Fatalf("loaded param count mismatch: %d vs %d", len(params1), len(params2))
	}
}

func TestForward_BatchConsistency(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	tok := []int{1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	coord := make([]float64, cfg.CoordDim)

	// call forward twice with same input — outputs should match
	logits1, _ := model.Forward([][]int{tok}, [][]float64{coord})
	logits2, _ := model.Forward([][]int{tok}, [][]float64{coord})

	for i := range logits1[0] {
		if math.Abs(logits1[0][i]-logits2[0][i]) > 1e-6 {
			t.Errorf("inconsistency at dim %d: %f vs %f", i, logits1[0][i], logits2[0][i])
		}
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"zero vocab", Config{VocabSize: 0, EmbedDim: 32, NumHeads: 2, NumLayers: 1, FFNDim: 64, MaxLen: 16, OutputDim: 10}},
		{"zero embed", Config{VocabSize: 100, EmbedDim: 0, NumHeads: 2, NumLayers: 1, FFNDim: 64, MaxLen: 16, OutputDim: 10}},
		{"zero layers", Config{VocabSize: 100, EmbedDim: 32, NumHeads: 2, NumLayers: 0, FFNDim: 64, MaxLen: 16, OutputDim: 10}},
		{"zero heads", Config{VocabSize: 100, EmbedDim: 32, NumHeads: 0, NumLayers: 1, FFNDim: 64, MaxLen: 16, OutputDim: 10}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.cfg)
			if err == nil {
				t.Error("expected error for invalid config")
			}
		})
	}
}

func TestForward_MismatchedBatchSize(t *testing.T) {
	cfg := newTestConfig()
	model, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	tokens := [][]int{{1, 2, 3}}
	coords := [][]float64{make([]float64, cfg.CoordDim), make([]float64, cfg.CoordDim)}
	_, err = model.Forward(tokens, coords)
	if err == nil {
		t.Error("expected error for mismatched batch/token length")
	}
}

func TestParameters_Deterministic(t *testing.T) {
	cfg := newTestConfig()
	m1, _ := New(cfg)
	m2, _ := New(cfg)
	p1 := m1.Parameters()
	p2 := m2.Parameters()
	// same config should produce same parameter count
	if len(p1) != len(p2) {
		t.Errorf("parameter count not deterministic: %d vs %d", len(p1), len(p2))
	}
}
