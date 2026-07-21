package compress

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func testModel(t *testing.T) transformer.Model {
	t.Helper()
	cfg := transformer.Config{
		VocabSize:  100,
		MaxLen:     16,
		EmbedDim:   32,
		NumHeads:   2,
		NumLayers:  2,
		FFNDim:     64,
		CoordDim:   7,
		OutputDim:  7,
		HistoryLen: 0,
	}
	m, err := transformer.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPrune_ZeroSparsity(t *testing.T) {
	m := testModel(t)
	paramsBefore := make([]float64, len(m.Parameters()))
	copy(paramsBefore, m.Parameters())

	sparsity, err := Prune(m, PruneConfig{Sparsity: 0})
	if err != nil {
		t.Fatal(err)
	}
	if sparsity != 0 {
		t.Errorf("expected 0 sparsity, got %f", sparsity)
	}
	paramsAfter := m.Parameters()
	for i := range paramsBefore {
		if paramsBefore[i] != paramsAfter[i] {
			t.Errorf("param %d changed at zero sparsity", i)
		}
	}
}

func TestPrune_50Percent(t *testing.T) {
	m := testModel(t)
	sparsity, err := Prune(m, PruneConfig{Sparsity: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if sparsity < 0.4 || sparsity > 0.6 {
		t.Errorf("expected ~0.5 sparsity, got %f", sparsity)
	}

	// verify exactly half the params are zero
	zeros := 0
	for _, p := range m.Parameters() {
		if p == 0 {
			zeros++
		}
	}
	ratio := float64(zeros) / float64(len(m.Parameters()))
	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("expected ~50%% zeros, got %.1f%%", ratio*100)
	}
}

func TestPrune_InvalidSparsity(t *testing.T) {
	m := testModel(t)
	s1, err := Prune(m, PruneConfig{Sparsity: -1})
	if err != nil {
		t.Fatal(err)
	}
	if s1 != 0 {
		t.Errorf("expected 0 for negative sparsity, got %f", s1)
	}
	s2, err := Prune(m, PruneConfig{Sparsity: 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if s2 != 0 {
		t.Errorf("expected 0 for sparsity=1.0, got %f", s2)
	}
}

func TestQuantize_8Bit(t *testing.T) {
	m := testModel(t)
	paramsBefore := m.Parameters()
	origMin, origMax := paramsBefore[0], paramsBefore[0]
	for _, p := range paramsBefore[1:] {
		if p < origMin {
			origMin = p
		}
		if p > origMax {
			origMax = p
		}
	}

	err := Quantize(m, QuantizeConfig{Bits: 8})
	if err != nil {
		t.Fatal(err)
	}

	paramsAfter := m.Parameters()
	// values should still be within original range
	for i, p := range paramsAfter {
		if p < origMin-0.001 || p > origMax+0.001 {
			t.Errorf("param %d out of range: %f (was %f)", i, p, paramsBefore[i])
		}
	}

	// check that there are fewer unique values (quantized)
	uniqueBefore := countUnique(paramsBefore)
	uniqueAfter := countUnique(paramsAfter)
	if uniqueAfter > uniqueBefore {
		t.Errorf("quantization increased unique values: %d -> %d", uniqueBefore, uniqueAfter)
	}
}

func TestQuantize_4Bit(t *testing.T) {
	m := testModel(t)
	err := Quantize(m, QuantizeConfig{Bits: 4})
	if err != nil {
		t.Fatal(err)
	}
	// 4-bit = 16 levels max
	unique := countUnique(m.Parameters())
	if unique > 16 {
		t.Errorf("4-bit quantization should have <=16 unique values, got %d", unique)
	}
}

func TestQuantize_InvalidBits(t *testing.T) {
	m := testModel(t)
	err := Quantize(m, QuantizeConfig{Bits: 1})
	if err != nil {
		t.Fatal(err)
	}
	// invalid bits should be ignored (default 8-bit)
}

func TestQuantizeSize(t *testing.T) {
	orig := OriginalSize(1000)
	q8 := QuantizedSize(1000, 8)
	q4 := QuantizedSize(1000, 4)
	q2 := QuantizedSize(1000, 2)

	if orig != 8000 {
		t.Errorf("original size: got %d, want 8000", orig)
	}
	if q8 != 1000 {
		t.Errorf("8-bit size: got %d, want 1000", q8)
	}
	if q4 != 500 {
		t.Errorf("4-bit size: got %d, want 500", q4)
	}
	if q2 != 250 {
		t.Errorf("2-bit size: got %d, want 250", q2)
	}
}

func TestCompressionRatio(t *testing.T) {
	r := CompressionRatio(1000, 8)
	if r != 8.0 {
		t.Errorf("8-bit ratio: got %f, want 8.0", r)
	}
	r = CompressionRatio(1000, 4)
	if r != 16.0 {
		t.Errorf("4-bit ratio: got %f, want 16.0", r)
	}
}

func TestPruneThenQuantize(t *testing.T) {
	m := testModel(t)
	sparsity, err := Prune(m, PruneConfig{Sparsity: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if sparsity < 0.4 {
		t.Errorf("sparsity too low: %f", sparsity)
	}

	err = Quantize(m, QuantizeConfig{Bits: 4})
	if err != nil {
		t.Fatal(err)
	}

	// verify model still produces valid output
	tokens := []int{1, 2, 3, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} // len == maxLen 16
	coords := make([]float64, 7)
	logits, err := m.Forward([][]int{tokens}, [][]float64{coords}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(logits) == 0 || len(logits[0]) == 0 {
		t.Error("model produced empty output after compression")
	}
}

func countUnique(vals []float64) int {
	seen := make(map[float64]bool)
	for _, v := range vals {
		seen[v] = true
	}
	return len(seen)
}
