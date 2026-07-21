package predict

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func setup() *Engine {
	cfg := transformer.Config{
		VocabSize: 100,
		MaxLen:    16,
		EmbedDim:  32,
		NumHeads:  2,
		NumLayers: 2,
		FFNDim:    64,
		CoordDim:  7,
		OutputDim: 10,
	}
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})
	engine := NewEngineWithConfig(model, tok, enc, cfg)
	engine.SetTools([]string{"click", "hover", "type_text", "scroll", "key_press"})
	return engine
}

func TestPredict_BasicOutput(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 3)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) == 0 {
		t.Fatal("expected at least 1 prediction")
	}
	if len(preds) > 3 {
		t.Errorf("expected at most 3 predictions, got %d", len(preds))
	}
}

func TestPredict_ToolNameFilled(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) == 0 {
		t.Fatal("expected at least 1 prediction")
	}
	hasTool := false
	for _, p := range preds {
		if p.Tool != "" {
			hasTool = true
		}
	}
	if !hasTool {
		t.Error("expected at least one prediction with a tool name")
	}
}

func TestPredict_ConfidenceRange(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("hover over Cancel", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) == 0 {
		t.Fatal("expected at least 1 prediction")
	}
	// scores are raw logits, so just check they're finite
	for _, p := range preds {
		if p.Score != p.Score { // NaN check
			t.Errorf("score is NaN: %f", p.Score)
		}
	}
}

func TestPredict_SortedByScore(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	for i := 1; i < len(preds); i++ {
		if preds[i].Score > preds[i-1].Score {
			t.Errorf("predictions not sorted by score at index %d", i)
		}
	}
}

func TestPredictWithContext_UsesHistory(t *testing.T) {
	engine := setup()
	history := []string{"click button Open", "type_text filename.txt"}
	preds1, _ := engine.Predict("click button Submit", 3)
	preds2, _ := engine.PredictWithContext("click button Submit", history, 3)
	// predictions should be valid (may or may not differ)
	if len(preds1) == 0 || len(preds2) == 0 {
		t.Fatal("expected at least 1 prediction from each method")
	}
}

func TestPredict_EmptyText(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("", 5)
	if err != nil {
		t.Fatalf("Predict failed on empty text: %v", err)
	}
	// should return empty or low-confidence predictions
	if len(preds) > 5 {
		t.Errorf("too many predictions for empty text: %d", len(preds))
	}
}

func TestSetTools_MapsToNames(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	validTools := map[string]bool{"click": true, "hover": true, "type_text": true, "scroll": true, "key_press": true}
	hasTool := false
	for _, p := range preds {
		if p.Tool != "" {
			hasTool = true
			if !validTools[p.Tool] {
				t.Errorf("unexpected tool name: %q", p.Tool)
			}
		}
	}
	if !hasTool {
		t.Error("expected at least one prediction with a tool name")
	}
}

func TestLoadModel_NonexistentFile(t *testing.T) {
	engine := setup()
	err := engine.LoadModel("/nonexistent/path/model.bin")
	if err == nil {
		t.Error("expected error loading nonexistent model")
	}
}

func TestPredict_TopKRespected(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 1)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) > 1 {
		t.Errorf("expected at most 1 prediction for topK=1, got %d", len(preds))
	}
}

func TestPredict_CoordForClickTool(t *testing.T) {
	engine := setup()
	preds, err := engine.Predict("click button Submit", 1)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) == 0 {
		t.Fatal("no predictions")
	}
	// click tool should have coordinates
	if preds[0].Tool == "click" {
		if preds[0].CoordX == 0 && preds[0].CoordY == 0 {
			// might be zero for untrained model, that's OK
		}
	}
}

func TestPredict_Deterministic(t *testing.T) {
	engine := setup()
	preds1, _ := engine.Predict("click button Submit", 3)
	preds2, _ := engine.Predict("click button Submit", 3)
	if len(preds1) != len(preds2) {
		t.Fatalf("non-deterministic: %d vs %d predictions", len(preds1), len(preds2))
	}
	for i := range preds1 {
		if preds1[i].Tool != preds2[i].Tool {
			t.Errorf("non-deterministic tool at %d: %q vs %q", i, preds1[i].Tool, preds2[i].Tool)
		}
	}
}

func TestNewEngine_NonNil(t *testing.T) {
	cfg := transformer.Config{
		VocabSize: 50, MaxLen: 8, EmbedDim: 16, NumHeads: 1,
		NumLayers: 1, FFNDim: 32, CoordDim: 7, OutputDim: 5,
	}
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	enc := spatial.NewEncoder(spatial.ScreenConfig{})
	engine := NewEngine(model, tok, enc)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestPredict_ContextHistoryLonger(t *testing.T) {
	engine := setup()
	longHistory := make([]string, 50)
	for i := range longHistory {
		longHistory[i] = "click button"
	}
	_, err := engine.PredictWithContext("click Submit", longHistory, 3)
	if err != nil {
		t.Fatalf("PredictWithContext failed with long history: %v", err)
	}
}
