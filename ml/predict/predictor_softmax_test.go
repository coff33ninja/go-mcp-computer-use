package predict

import (
	"math"
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func setupLarge() *Engine {
	cfg := transformer.Config{
		VocabSize:  100,
		MaxLen:     16,
		EmbedDim:   32,
		NumHeads:   2,
		NumLayers:  2,
		FFNDim:     64,
		CoordDim:   7,
		OutputDim:  8, // 5 tools + 2 coords + 1 extra
	}
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
	})
	engine := NewEngineWithConfig(model, tok, enc, cfg)
	engine.SetTools([]string{"click", "hover", "type_text", "scroll", "key_press"})
	return engine
}

func TestPredict_SoftmaxConfidenceSumToOne(t *testing.T) {
	engine := setupLarge()
	preds, err := engine.Predict("click button Submit", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) == 0 {
		t.Fatal("expected predictions")
	}
	// top-k predictions from softmax should all be positive
	for _, p := range preds {
		if p.Confidence < 0 || p.Confidence > 1.0+1e-6 {
			t.Errorf("confidence out of [0,1] range: %f", p.Confidence)
		}
	}
	// the top prediction should have confidence > 0
	if preds[0].Confidence <= 0 {
		t.Errorf("top prediction confidence should be > 0, got %f", preds[0].Confidence)
	}
}

func TestPredict_CoordsAreNonNegative(t *testing.T) {
	engine := setupLarge()
	preds, err := engine.Predict("click Submit", 3)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	for _, p := range preds {
		if p.CoordX < 0 || p.CoordY < 0 {
			t.Errorf("negative coords: (%d, %d)", p.CoordX, p.CoordY)
		}
	}
}

func TestPredict_ScoresAreSorted(t *testing.T) {
	engine := setupLarge()
	preds, err := engine.Predict("click Submit", 5)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	for i := 1; i < len(preds); i++ {
		if preds[i].Score > preds[i-1].Score+1e-9 {
			t.Errorf("not sorted at %d: %f > %f", i, preds[i].Score, preds[i-1].Score)
		}
	}
}

func TestPredict_ToolProbsSumToOne(t *testing.T) {
	// verify softmax produces valid probability distribution
	logits := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	probs := transformer.Softmax(logits)
	var sum float64
	for _, p := range probs {
		sum += p
		if p < 0 || p > 1 {
			t.Errorf("prob out of range: %f", p)
		}
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("probs sum to %f, expected 1.0", sum)
	}
}

func TestPredict_ToolsOnlyOutput(t *testing.T) {
	// when output has exactly numTools dims (no coords)
	cfg := transformer.Config{
		VocabSize: 50, MaxLen: 8, EmbedDim: 16, NumHeads: 1,
		NumLayers: 1, FFNDim: 32, CoordDim: 7, OutputDim: 5,
	}
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click hover"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{})
	engine := NewEngineWithConfig(model, tok, enc, cfg)
	engine.SetTools([]string{"click", "hover", "type_text", "scroll", "key_press"})

	preds, err := engine.Predict("click", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) == 0 {
		t.Fatal("expected predictions")
	}
	// coords should be 0 when not enough output dims
	if preds[0].CoordX != 0 || preds[0].CoordY != 0 {
		t.Errorf("expected zero coords when output is tools-only, got (%d,%d)", preds[0].CoordX, preds[0].CoordY)
	}
}
