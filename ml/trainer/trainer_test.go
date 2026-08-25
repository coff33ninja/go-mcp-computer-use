package trainer

import (
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
	_ "modernc.org/sqlite"
)

func testCfg() transformer.Config {
	return transformer.Config{
		VocabSize:  100,
		MaxLen:     16,
		EmbedDim:   32,
		NumHeads:   2,
		NumLayers:  2,
		FFNDim:     64,
		CoordDim:   7,
		OutputDim:  5 + 2 + 10 + 6, // 5 tools + 2 to_xy + 10 args + 6 window (FromCoordDim=0)
		ArgDim:     10,
		WindowDim:  6,
		HistoryLen: 0,
	}
}

func seedLoader(t *testing.T) *dataloader.SQLiteLoader {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE training_pairs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT DEFAULT '',
		ocr_before_text TEXT DEFAULT '', command_json TEXT DEFAULT '',
		ocr_after_text TEXT DEFAULT '', window_title TEXT DEFAULT '',
		success INTEGER DEFAULT 1, created_at TEXT DEFAULT '')`)
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('click Submit', '{"tool":"click","args":"{\"x\":100,\"y\":200}"}', 'App', 1, '2026-07-21')`)
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('hover Cancel', '{"tool":"hover","args":"{\"x\":50,\"y\":50}"}', 'App', 1, '2026-07-21')`)
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('type text filename', '{"tool":"type_text","args":"{\"text\":\"hello\"}"}', 'App', 1, '2026-07-21')`)
	db.Close()
	loader := dataloader.NewSQLiteLoader(dbPath)
	t.Cleanup(func() { loader.Close() })
	return loader
}

func TestTrainer_EpochReducesLoss(t *testing.T) {
	cfg := testCfg()
	model, err := transformer.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text scroll"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080})
	tools := []string{"click", "hover", "type_text", "scroll", "key_press"}

	loader := seedLoader(t)
	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.01,
	})

	result, err := tr.TrainEpoch(loader)
	if err != nil {
		t.Fatalf("TrainEpoch failed: %v", err)
	}
	if result.SamplesProcessed == 0 {
		t.Error("expected samples to be processed")
	}
	if result.FinalLoss >= result.InitialLoss {
		t.Errorf("loss did not decrease: %.6f -> %.6f", result.InitialLoss, result.FinalLoss)
	}
	t.Logf("loss: %.6f -> %.6f (samples: %d)", result.InitialLoss, result.FinalLoss, result.SamplesProcessed)
}

func TestTrainer_EmptyDataset(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click hover"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{})

	loader := dataloader.NewSQLiteLoader(":memory:")
	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click", "hover"},
		LearningRate: 0.01,
	})

	result, err := tr.TrainEpoch(loader)
	if err != nil {
		t.Fatalf("TrainEpoch failed: %v", err)
	}
	if result.SamplesProcessed != 0 {
		t.Errorf("expected 0 samples, got %d", result.SamplesProcessed)
	}
}

func TestTrainer_MultipleEpochs(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080})

	loader := seedLoader(t)
	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click", "hover", "type_text"},
		LearningRate: 0.01,
	})

	var prevLoss float64
	for i := 0; i < 3; i++ {
		result, err := tr.TrainEpoch(loader)
		if err != nil {
			t.Fatalf("epoch %d failed: %v", i, err)
		}
		// just verify training runs and produces finite loss
		if math.IsNaN(result.FinalLoss) || math.IsInf(result.FinalLoss, 0) {
			t.Errorf("epoch %d: loss is not finite: %f", i, result.FinalLoss)
		}
		if i > 0 && result.FinalLoss >= prevLoss {
			// allow small regression — just log it
			t.Logf("epoch %d: loss not strictly decreasing (%.6f >= %.6f)", i, result.FinalLoss, prevLoss)
		}
		prevLoss = result.FinalLoss
	}
}

func TestTrainer_SaveLoadTrainedModel(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080})

	loader := seedLoader(t)
	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click", "hover"},
		LearningRate: 0.01,
	})
	tr.TrainEpoch(loader)

	dir := t.TempDir()
	if err := tr.SaveModel(dir + "/model.bin"); err != nil {
		t.Fatalf("SaveModel failed: %v", err)
	}

	model2, _ := transformer.New(cfg)
	tr2 := NewTrainer(TrainerConfig{
		Model:        model2,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click", "hover"},
		LearningRate: 0.01,
	})
	if err := tr2.LoadModel(dir + "/model.bin"); err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	tokens := tok.Encode("click Submit", cfg.MaxLen)
	coords := make([]float64, cfg.CoordDim)
	out1, _ := model.Forward([][]int{tokens}, [][]float64{coords}, nil)
	out2, _ := model2.Forward([][]int{tokens}, [][]float64{coords}, nil)
	for i := range out1[0] {
		diff := out1[0][i] - out2[0][i]
		if diff < 0 {
			diff = -diff
		}
		if diff > 1e-6 {
			t.Errorf("output mismatch at %d: %f vs %f", i, out1[0][i], out2[0][i])
		}
	}
}

func TestTrainer_NilLoader(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	enc := spatial.NewEncoder(spatial.ScreenConfig{})

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click"},
		LearningRate: 0.01,
	})

	_, err := tr.TrainEpoch(nil)
	if err == nil {
		t.Error("expected error for nil loader")
	}
}

func TestTrainer_ZeroLearningRate(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{})
	loader := seedLoader(t)

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click"},
		LearningRate: 0,
	})

	result, err := tr.TrainEpoch(loader)
	if err != nil {
		t.Fatal(err)
	}
	if result.SamplesProcessed == 0 {
		t.Error("expected samples processed")
	}
	// zero lr with Adam may still change loss due to internal momentum,
	// but should not panic or error
	t.Logf("zero lr: %.6f -> %.6f", result.InitialLoss, result.FinalLoss)
}

func TestTrainer_TempDirCleanup(t *testing.T) {
	dir, _ := os.MkdirTemp("", "trainer-test-*")
	defer os.RemoveAll(dir)

	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	enc := spatial.NewEncoder(spatial.ScreenConfig{})

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click"},
		LearningRate: 0.01,
	})

	if err := tr.SaveModel(filepath.Join(dir, "model.bin")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.bin")); os.IsNotExist(err) {
		t.Error("model file not created")
	}
}

func TestDecodeArgIndex_Scroll(t *testing.T) {
	tests := []struct {
		argsJSON string
		expected int
	}{
		{`{"clicks":5}`, 0},   // up
		{`{"clicks":-3}`, 1},  // down
		{`{"clicks":0}`, 1},   // default down
		{`{"clicks":100}`, 0}, // up
	}
	for _, tt := range tests {
		idx := decodeArgIndex("scroll", tt.argsJSON)
		if idx != tt.expected {
			t.Errorf("scroll %s: got %d, want %d", tt.argsJSON, idx, tt.expected)
		}
	}
}

func TestDecodeArgIndex_KeyPress(t *testing.T) {
	tests := []struct {
		argsJSON string
		expected int
	}{
		{`{"keys":["Ctrl"]}`, 4},         // modifier
		{`{"keys":["Shift"]}`, 4},         // modifier
		{`{"keys":["Up"]}`, 5},            // navigation
		{`{"keys":["Tab"]}`, 5},           // navigation
		{`{"keys":["F5"]}`, 6},            // function
		{`{"keys":["a"]}`, 7},             // alpha
		{`{"keys":["Z"]}`, 7},             // alpha
		{`{"keys":["3"]}`, 8},             // numeric
		{`{"keys":["Enter"]}`, 9},         // special
		{`{"keys":["Escape"]}`, 9},        // special
		{`{"keys":["Backspace"]}`, 9},     // special
	}
	for _, tt := range tests {
		idx := decodeArgIndex("key_press", tt.argsJSON)
		if idx != tt.expected {
			t.Errorf("key_press %s: got %d, want %d", tt.argsJSON, idx, tt.expected)
		}
	}
}

func TestDecodeArgIndex_OtherTool(t *testing.T) {
	idx := decodeArgIndex("click", `{"x":100,"y":200}`)
	if idx != -1 {
		t.Errorf("click should return -1, got %d", idx)
	}
	idx = decodeArgIndex("hover", `{"x":50,"y":50}`)
	if idx != -1 {
		t.Errorf("hover should return -1, got %d", idx)
	}
}

func TestMakeTarget_ArgDim(t *testing.T) {
	cfg := testCfg()
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	enc := spatial.NewEncoder(spatial.ScreenConfig{})

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        []string{"click", "scroll", "key_press", "hover", "type_text"},
		LearningRate: 0.01,
	})

	// scroll down → tool index 1 = 1.0, arg index 1 (scroll_down) = 1.0
	target := tr.makeTarget("scroll", `{"clicks":-5}`)
	numTools := 5
	if target[1] != 1.0 {
		t.Errorf("scroll tool: got %.1f, want 1.0", target[1])
	}
	if target[numTools+2+1] != 1.0 {
		t.Errorf("scroll_down arg: got %.1f, want 1.0 at index %d", target[numTools+2+1], numTools+2+1)
	}

	// key_press Enter → tool index 2 = 1.0, arg index 9 (special) = 1.0
	target = tr.makeTarget("key_press", `{"keys":["Enter"]}`)
	if target[2] != 1.0 {
		t.Errorf("key_press tool: got %.1f, want 1.0", target[2])
	}
	if target[numTools+2+9] != 1.0 {
		t.Errorf("key_special arg: got %.1f, want 1.0 at index %d", target[numTools+2+9], numTools+2+9)
	}

	// click → no arg set
	target = tr.makeTarget("click", `{"x":100,"y":200}`)
	if target[0] != 1.0 {
		t.Errorf("click tool: got %.1f, want 1.0", target[0])
	}
	// all arg dims should be 0
	for i := numTools + 2; i < numTools+2+10; i++ {
		if target[i] != 0.0 {
			t.Errorf("click arg[%d] should be 0, got %.1f", i, target[i])
		}
	}
}

func TestClassifyKey_Comprehensive(t *testing.T) {
	tests := []struct {
		key      string
		expected int
	}{
		{"ctrl", 4}, {"alt", 4}, {"shift", 4}, {"win", 4},
		{"Ctrl+C", 4}, {"Alt+Tab", 4}, {"Shift+A", 4},
		{"up", 5}, {"down", 5}, {"left", 5}, {"right", 5},
		{"home", 5}, {"end", 5}, {"pageup", 5}, {"pagedown", 5}, {"tab", 5},
		{"f1", 6}, {"f12", 6},
		{"a", 7}, {"z", 7}, {"m", 7},
		{"0", 8}, {"9", 8},
		{"enter", 9}, {"space", 9}, {"escape", 9}, {"backspace", 9}, {"delete", 9},
	}
	for _, tt := range tests {
		got := classifyKey(tt.key)
		if got != tt.expected {
			t.Errorf("classifyKey(%q) = %d, want %d", tt.key, got, tt.expected)
		}
	}
}

func TestMakeSequenceTargets_FillsSlots(t *testing.T) {
	cfg := testCfg()
	cfg.SequenceLen = 2
	cfg.OutputDim = cfg.PrimaryDim(5) + 2*cfg.SeqSlotDim(5)
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text scroll"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080})
	tools := []string{"click", "hover", "type_text", "scroll", "key_press"}

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.01,
	})

	target := make([]float64, cfg.OutputDim)
	actions := []struct {
		Action   string
		ArgsJSON string
	}{
		{Action: "hover", ArgsJSON: `{"x":50,"y":50}`},
		{Action: "click", ArgsJSON: `{"x":100,"y":200}`},
	}
	tr.makeSequenceTargets(target, actions)

	// slot 0: hover → tool index 1 = 1.0
	slotDim := len(tools) + 2 + cfg.ArgDim
	if target[tr.primaryDim+1] != 1.0 {
		t.Errorf("seq slot 0 hover tool: got %.1f, want 1.0", target[tr.primaryDim+1])
	}
	// slot 1: click → tool index 0 = 1.0
	if target[tr.primaryDim+slotDim+0] != 1.0 {
		t.Errorf("seq slot 1 click tool: got %.1f, want 1.0", target[tr.primaryDim+slotDim+0])
	}
}

func TestMakeSequenceTargets_ZeroLenSkips(t *testing.T) {
	cfg := testCfg()
	cfg.SequenceLen = 0
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	enc := spatial.NewEncoder(spatial.ScreenConfig{})
	tools := []string{"click", "hover"}

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.01,
	})

	target := make([]float64, cfg.OutputDim)
	tr.makeSequenceTargets(target, []struct {
		Action   string
		ArgsJSON string
	}{
		{Action: "click", ArgsJSON: `{"x":10,"y":20}`},
	})
	// should remain all zeros
	for i, v := range target {
		if v != 0.0 {
			t.Errorf("target[%d] = %f, want 0 (sequenceLen=0)", i, v)
		}
	}
}

func TestPrepareBatch_FillsSequenceTargets(t *testing.T) {
	cfg := testCfg()
	cfg.SequenceLen = 2
	cfg.OutputDim = cfg.PrimaryDim(5) + 2*cfg.SeqSlotDim(5)
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text scroll"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{ScreenWidth: 1920, ScreenHeight: 1080})
	tools := []string{"click", "hover", "type_text", "scroll", "key_press"}

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.01,
	})

	samples := []dataloader.Sample{
		{Action: "click", ArgsJSON: `{"x":100,"y":200}`},
		{Action: "hover", ArgsJSON: `{"x":50,"y":50}`},
		{Action: "type_text", ArgsJSON: `{"text":"hello"}`},
	}

	// pass full samples slice so sequence look-ahead works
	_, _, targets := tr.prepareBatch(samples, 0)
	// sequence section should be filled from samples[0..1]
	slotDim := len(tools) + 2 + cfg.ArgDim
	// slot 0: samples[0].Action = "click" → tool index 0
	if targets[0][tr.primaryDim+0] != 1.0 {
		t.Errorf("seq slot 0 click tool: got %.1f, want 1.0", targets[0][tr.primaryDim+0])
	}
	// slot 1: samples[1].Action = "hover" → tool index 1
	if targets[0][tr.primaryDim+slotDim+1] != 1.0 {
		t.Errorf("seq slot 1 hover tool: got %.1f, want 1.0", targets[0][tr.primaryDim+slotDim+1])
	}
}

func TestPrepareBatch_NoSequenceWhenDisabled(t *testing.T) {
	cfg := testCfg()
	cfg.SequenceLen = 0
	model, _ := transformer.New(cfg)
	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click hover"})
	enc := spatial.NewEncoder(spatial.ScreenConfig{})
	tools := []string{"click", "hover"}

	tr := NewTrainer(TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.01,
	})

	samples := []dataloader.Sample{
		{Action: "click", ArgsJSON: `{"x":10,"y":20}`},
	}

	_, _, targets := tr.prepareBatch(samples, 0)
	// target should only have primary dims, no sequence section
	if len(targets[0]) != cfg.OutputDim {
		t.Errorf("target dim: got %d, want %d", len(targets[0]), cfg.OutputDim)
	}
}
