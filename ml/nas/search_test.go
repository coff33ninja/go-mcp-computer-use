package nas

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func TestDefaultSearchSpace(t *testing.T) {
	ss := DefaultSearchSpace()
	if len(ss.EmbedDims) == 0 || len(ss.NumHeads) == 0 || len(ss.NumLayers) == 0 || len(ss.FFNDims) == 0 {
		t.Error("default search space has empty dimensions")
	}
}

func TestParamCount(t *testing.T) {
	tests := []struct {
		cfg transformer.Config
	}{
		{transformer.Config{VocabSize: 100, MaxLen: 16, EmbedDim: 32, NumHeads: 2, NumLayers: 2, FFNDim: 64, OutputDim: 7, ArgDim: 10}},
		{transformer.Config{VocabSize: 2000, MaxLen: 128, EmbedDim: 64, NumHeads: 2, NumLayers: 2, FFNDim: 128, OutputDim: 50, ArgDim: 10}},
		{transformer.Config{VocabSize: 2000, MaxLen: 128, EmbedDim: 128, NumHeads: 4, NumLayers: 4, FFNDim: 256, OutputDim: 50, ArgDim: 10}},
	}
	for _, tt := range tests {
		pc := transformer.ParamCount(tt.cfg)
		if pc <= 0 {
			t.Errorf("transformer.ParamCount returned %d for EmbedDim=%d", pc, tt.cfg.EmbedDim)
		}
	}
}

func TestBestConfig_Empty(t *testing.T) {
	r := BestConfig(nil, 10000)
	if r != nil {
		t.Error("expected nil for empty results")
	}
}

func TestBestConfig_UnderBudget(t *testing.T) {
	results := []SearchResult{
		{ParamCount: 5000, Loss: 0.1},
		{ParamCount: 3000, Loss: 0.2},
		{ParamCount: 1000, Loss: 0.3},
	}
	r := BestConfig(results, 4000)
	if r == nil || r.ParamCount != 3000 {
		t.Errorf("expected 3000-param model, got %v", r)
	}
}

func TestBestConfig_OverBudget(t *testing.T) {
	results := []SearchResult{
		{ParamCount: 5000, Loss: 0.1},
		{ParamCount: 3000, Loss: 0.2},
	}
	r := BestConfig(results, 2000)
	if r == nil || r.ParamCount != 3000 {
		t.Errorf("expected fallback model, got %v", r)
	}
}

func TestSearch_NoData(t *testing.T) {
	_, err := Search(":memory:", []string{"click"}, spatial.ScreenConfig{}, 2, 5)
	if err == nil {
		t.Error("expected error for empty database")
	}
}
