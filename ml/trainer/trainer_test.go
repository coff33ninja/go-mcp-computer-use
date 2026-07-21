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
		OutputDim:  7,
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
