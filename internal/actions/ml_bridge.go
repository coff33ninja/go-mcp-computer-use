package actions

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/predict"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/trainer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

var mlDefaultTools = []string{
	"click", "move_mouse", "scroll", "drag",
	"type", "key_press", "key_down", "key_up",
	"hover", "focus_window", "launch_and_wait",
}

// MLEngine wraps the Go-native transformer model for UI automation prediction.
// It trains from the same training_pairs SQLite table as the statistical engine
// and provides transformer-based predictions as a primary source, with the
// statistical engine as fallback.
type MLEngine struct {
	model      transformer.Model
	predictor  *predict.Engine
	tok        *tokenizer.SimpleTokenizer
	encoder    *spatial.Encoder
	modelPath  string
	dbPath     string
	tools      []string
	mu         sync.RWMutex
	ready      bool
}

// NewMLEngine creates an ML engine that reads from the same datalog DB
// as the statistical AdaptiveEngine and persists its model alongside it.
func NewMLEngine(dataDir string) *MLEngine {
	dbPath := filepath.Join(dataDir, "datalog.db")
	modelPath := filepath.Join(dataDir, "model.gob")

	enc := spatial.NewEncoder(spatial.ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})

	tok := tokenizer.NewSimpleTokenizer()

	return &MLEngine{
		modelPath: modelPath,
		dbPath:    dbPath,
		tools:     mlDefaultTools,
		encoder:   enc,
		tok:       tok,
	}
}

// Train loads training pairs from the SQLite datalog and trains the
// transformer model. It saves the model to disk on success.
func (m *MLEngine) Train() error {
	loader := dataloader.NewSQLiteLoader(m.dbPath)
	defer loader.Close()

	ctx := context.Background()
	count, err := loader.Count(ctx)
	if err != nil {
		return fmt.Errorf("ml: count training data: %w", err)
	}
	if count < 5 {
		slog.Debug("ml: insufficient training data", "count", count, "min", 5)
		return nil
	}

	cfg := transformer.Config{
		VocabSize: 2000,
		MaxLen:    128,
		EmbedDim:  64,
		NumHeads:  2,
		NumLayers: 2,
		FFNDim:    128,
		CoordDim:  spatial.FeatureDim,
		OutputDim: len(m.tools) + 2,
	}

	// try loading existing model, create new if not found
	var mdl transformer.Model
	if _, statErr := os.Stat(m.modelPath); statErr == nil {
		mdl, err = transformer.New(cfg)
		if err != nil {
			return fmt.Errorf("ml: create model: %w", err)
		}
		if err := mdl.Load(m.modelPath); err != nil {
			slog.Warn("ml: failed to load saved model, retraining", "err", err)
			mdl, err = transformer.New(cfg)
			if err != nil {
				return fmt.Errorf("ml: recreate model: %w", err)
			}
		}
	} else {
		mdl, err = transformer.New(cfg)
		if err != nil {
			return fmt.Errorf("ml: create model: %w", err)
		}
	}

	// build vocabulary from training data
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("ml: load samples: %w", err)
	}

	tok := tokenizer.NewSimpleTokenizer()
	corpus := make([]string, len(samples))
	for i, s := range samples {
		corpus[i] = s.Context
	}
	tok.Fit(corpus)

	enc := spatial.NewEncoder(spatial.ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})

	tr := trainer.NewTrainer(trainer.TrainerConfig{
		Model:        mdl,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        m.tools,
		LearningRate: 0.001,
	})

	result, err := tr.TrainEpoch(loader)
	if err != nil {
		return fmt.Errorf("ml: train epoch: %w", err)
	}

	if result.SamplesProcessed > 0 {
		slog.Info("ml: training complete",
			"loss_start", fmt.Sprintf("%.4f", result.InitialLoss),
			"loss_end", fmt.Sprintf("%.4f", result.FinalLoss),
			"samples", result.SamplesProcessed,
		)
	}

	if err := tr.SaveModel(m.modelPath); err != nil {
		return fmt.Errorf("ml: save model: %w", err)
	}

	// build predictor
	pred := predict.NewEngineWithConfig(mdl, tok, enc, cfg)
	pred.SetTools(m.tools)

	m.mu.Lock()
	m.model = mdl
	m.predictor = pred
	m.tok = tok
	m.ready = true
	m.mu.Unlock()

	return nil
}

// LoadModel loads a previously trained model from disk.
func (m *MLEngine) LoadModel() error {
	cfg := transformer.Config{
		VocabSize: 2000,
		MaxLen:    128,
		EmbedDim:  64,
		NumHeads:  2,
		NumLayers: 2,
		FFNDim:    128,
		CoordDim:  spatial.FeatureDim,
		OutputDim: len(m.tools) + 2,
	}

	mdl, err := transformer.New(cfg)
	if err != nil {
		return fmt.Errorf("ml: create model: %w", err)
	}
	if err := mdl.Load(m.modelPath); err != nil {
		return err
	}

	enc := spatial.NewEncoder(spatial.ScreenConfig{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		DPIScale:     1.0,
	})

	tok := tokenizer.NewSimpleTokenizer()

	pred := predict.NewEngineWithConfig(mdl, tok, enc, cfg)
	pred.SetTools(m.tools)

	m.mu.Lock()
	m.model = mdl
	m.predictor = pred
	m.tok = tok
	m.ready = true
	m.mu.Unlock()

	return nil
}

// IsReady returns true if the ML model is loaded and ready for predictions.
func (m *MLEngine) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// Predict returns transformer-based predictions for the given OCR text.
// Returns nil if the model is not ready or produces no predictions.
func (m *MLEngine) Predict(ocrText string, limit int) []PredictedAction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.ready || m.predictor == nil {
		return nil
	}

	preds, err := m.predictor.Predict(ocrText, limit)
	if err != nil {
		slog.Debug("ml: predict error", "err", err)
		return nil
	}

	var results []PredictedAction
	for _, p := range preds {
		if p.Tool == "" || p.Score < 0.01 {
			continue
		}
		pa := PredictedAction{
			Command:    p.Tool,
			Confidence: math.Round(p.Score*100) / 100,
			SampleSize: 1,
		}
		if p.CoordX != 0 || p.CoordY != 0 {
			pa.Coord = &PredictedCoord{
				X:          p.CoordX,
				Y:          p.CoordY,
				Confidence: p.Score,
				Samples:    1,
			}
		}
		results = append(results, pa)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Reset clears the ML engine state.
func (m *MLEngine) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.model = nil
	m.predictor = nil
	m.ready = false
}
