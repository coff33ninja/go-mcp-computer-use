package trainer

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

func TestDecodeCoords_ProductionNestedArgs(t *testing.T) {
	fromX, fromY, toX, toY := decodeCoords("click", `{"args":"{\"X\":700,\"Y\":400}","tool":"click"}`)
	if toX != 700 || toY != 400 {
		t.Fatalf("nested string args -> (%v,%v) want (700,400)", toX, toY)
	}
	if fromX != 700 || fromY != 400 {
		t.Fatalf("from should mirror to for click, got (%v,%v)", fromX, fromY)
	}

	_, _, x2, y2 := decodeCoords("click", `{"tool":"click","args":{"X":11,"Y":22}}`)
	if x2 != 11 || y2 != 22 {
		t.Fatalf("nested object args -> (%v,%v)", x2, y2)
	}

	_, _, x3, y3 := decodeCoords("click", `{"x":3,"y":4}`)
	if x3 != 3 || y3 != 4 {
		t.Fatalf("flat lowercase -> (%v,%v)", x3, y3)
	}
}

func TestMakeTargetFromSample_UsesNormalizedCoords(t *testing.T) {
	cfg := transformer.Config{
		VocabSize: 64, MaxLen: 16, EmbedDim: 32, NumHeads: 2, NumLayers: 1, FFNDim: 64,
		CoordDim: spatial.FeatureDim, ArgDim: 10, FromCoordDim: 2, WindowDim: 6,
		OutputDim: 5 + 2 + 2 + 10 + 6, HistoryLen: 0,
	}
	model, err := transformer.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click Submit button"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080, DPIScale: 1})
	tools := []string{"click", "hover", "type", "scroll", "key_press"}
	tr := NewTrainer(TrainerConfig{
		Model: model, ModelConfig: cfg, Tokenizer: tok, Encoder: enc, Tools: tools, LearningRate: 0.01,
	})

	s := dataloader.Sample{
		Context:  "click Submit",
		Action:   "click",
		ArgsJSON: `{"x":960,"y":540}`,
		CoordX:   960,
		CoordY:   540,
	}
	target := tr.makeTargetFromSample(s)
	numTools := len(tools)
	// to_xy slots are at numTools+fromCoordDim .. +2; features[0]=normX
	normX := target[numTools+2]
	normY := target[numTools+3]
	wantX := 960.0 / 1920.0
	wantY := 540.0 / 1080.0
	if abs(normX-wantX) > 1e-6 || abs(normY-wantY) > 1e-6 {
		t.Fatalf("target to_xy norm=(%v,%v) want (~%v,~%v)", normX, normY, wantX, wantY)
	}

	// production-shaped args without Sample coords still decode
	s2 := dataloader.Sample{Action: "click", ArgsJSON: `{"args":"{\"X\":100,\"Y\":200}","tool":"click"}`}
	target2 := tr.makeTargetFromSample(s2)
	nx2 := target2[numTools+2]
	ny2 := target2[numTools+3]
	if abs(nx2-100.0/1920.0) > 1e-5 || abs(ny2-200.0/1080.0) > 1e-5 {
		t.Fatalf("production args target=(%v,%v)", nx2, ny2)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
