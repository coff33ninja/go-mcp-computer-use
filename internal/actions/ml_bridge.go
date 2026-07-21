package actions

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/online"
	"github.com/coff33ninja/go-mcp-computer-use/ml/predict"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/trainer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/versioning"
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
	screenCfg  spatial.ScreenConfig
	tools      []string
	replayBuf  *online.ReplayBuffer
	versioner  *versioning.ModelVersion
	trainDone  chan struct{}
	mu         sync.RWMutex
	ready      bool
}

// detectScreenConfig reads the actual screen dimensions and DPI scale
// using Windows APIs. Falls back to 1920x1080/1.0 if detection fails.
func detectScreenConfig() spatial.ScreenConfig {
	sw, sh := ScreenSize()
	dpi, err := GetDPIScaleForPoint(0, 0)
	scale := 1.0
	if err == nil && dpi > 0 {
		scale = float64(dpi) / 96.0
	}
	cfg := spatial.ScreenConfig{
		ScreenWidth:  int(sw),
		ScreenHeight: int(sh),
		DPIScale:     scale,
	}
	if cfg.ScreenWidth <= 0 {
		cfg.ScreenWidth = 1920
	}
	if cfg.ScreenHeight <= 0 {
		cfg.ScreenHeight = 1080
	}
	slog.Debug("ml: detected screen config",
		"width", cfg.ScreenWidth, "height", cfg.ScreenHeight,
		"dpi_scale", fmt.Sprintf("%.2f", cfg.DPIScale))
	return cfg
}

// NewMLEngine creates an ML engine that reads from the same datalog DB
// as the statistical AdaptiveEngine and persists its model alongside it.
func NewMLEngine(dataDir string) *MLEngine {
	dbPath := filepath.Join(dataDir, "datalog.db")
	modelPath := filepath.Join(dataDir, "model.gob")

	screenCfg := detectScreenConfig()
	enc := spatial.NewEncoder(screenCfg)

	tok := tokenizer.NewSimpleTokenizer()

	return &MLEngine{
		modelPath: modelPath,
		dbPath:    dbPath,
		screenCfg: screenCfg,
		tools:     mlDefaultTools,
		encoder:   enc,
		tok:       tok,
		replayBuf: online.NewReplayBuffer(10000),
		versioner: versioning.New(dataDir),
		trainDone: make(chan struct{}),
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

	// build vocabulary from training data (with augmentation)
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("ml: load samples: %w", err)
	}

	aug := dataloader.NewAugmentor()
	samples = aug.AugmentAll(samples, 2) // 2 augmented per original

	tok := tokenizer.NewSimpleTokenizer()
	corpus := make([]string, len(samples))
	for i, s := range samples {
		corpus[i] = s.Context
	}
	tok.Fit(corpus)

	enc := spatial.NewEncoder(m.screenCfg)

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

	// save initial versioned checkpoint
	if _, err := m.versioner.SaveCheckpoint(mdl, result.FinalLoss, 0.0, result.SamplesProcessed); err != nil {
		slog.Debug("ml: failed to save initial checkpoint", "err", err)
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

	m.StartOnlineTraining()

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

	enc := spatial.NewEncoder(m.screenCfg)

	tok := tokenizer.NewSimpleTokenizer()

	pred := predict.NewEngineWithConfig(mdl, tok, enc, cfg)
	pred.SetTools(m.tools)

	m.mu.Lock()
	m.model = mdl
	m.predictor = pred
	m.tok = tok
	m.ready = true
	m.mu.Unlock()

	m.StartOnlineTraining()

	return nil
}

// IsReady returns true if the ML model is loaded and ready for predictions.
func (m *MLEngine) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// Predict returns transformer-based predictions for the given OCR text.
// history is a list of recent action tool names for sequence context.
// Returns nil if the model is not ready or produces no predictions.
func (m *MLEngine) Predict(ocrText string, limit int, history []string) []PredictedAction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.ready || m.predictor == nil {
		return nil
	}

	preds, err := m.predictor.PredictWithContext(ocrText, history, limit)
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
	m.StopOnlineTraining()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.model = nil
	m.predictor = nil
	m.ready = false
}

// RecordExperience stores a completed action for online learning.
func (m *MLEngine) RecordExperience(ctx, action string, success bool, x, y int) {
	m.replayBuf.Store(online.Experience{
		Context:  ctx,
		Action:   action,
		Success:  success,
		CoordX:   x,
		CoordY:   y,
	})
}

// StartOnlineTraining launches a background goroutine that periodically
// retrains the model from the replay buffer. Safe to call multiple times.
func (m *MLEngine) StartOnlineTraining() {
	m.mu.RLock()
	ready := m.ready
	m.mu.RUnlock()
	if !ready {
		return
	}
	// prevent duplicate goroutines
	select {
	case <-m.trainDone:
		// previous goroutine exited, can start new one
	default:
		if m.trainDone != nil {
			return // already running
		}
	}
	m.trainDone = make(chan struct{})

	go func() {
		defer close(m.trainDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.trainFromBuffer()
			case <-m.trainDone:
				return
			}
		}
	}()
}

// StopOnlineTraining stops the background training goroutine.
func (m *MLEngine) StopOnlineTraining() {
	m.mu.RLock()
	done := m.trainDone
	m.mu.RUnlock()
	if done != nil {
		close(done)
	}
}

func (m *MLEngine) trainFromBuffer() {
	m.mu.RLock()
	model := m.model
	tok := m.tok
	enc := m.encoder
	m.mu.RUnlock()

	if model == nil || tok == nil || enc == nil {
		return
	}
	if m.replayBuf.Size() < 32 {
		return
	}

	batch := m.replayBuf.Sample(32)
	var corpus []string
	for _, exp := range batch {
		if exp.Context != "" {
			corpus = append(corpus, exp.Context)
		}
	}
	if len(corpus) == 0 {
		return
	}

	tok.Fit(corpus)

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

	loader := dataloader.NewSQLiteLoader(m.dbPath)
	defer loader.Close()

	tr := trainer.NewTrainer(trainer.TrainerConfig{
		Model:        model,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        m.tools,
		LearningRate: 0.0005,
	})

	result, err := tr.TrainEpoch(loader)
	if err != nil {
		slog.Debug("ml: online training failed", "err", err)
		return
	}
	if result.SamplesProcessed > 0 {
		slog.Debug("ml: online training complete",
			"loss", fmt.Sprintf("%.4f", result.FinalLoss),
			"samples", result.SamplesProcessed,
		)

		// save versioned checkpoint
		vid, err := m.versioner.SaveCheckpoint(model, result.FinalLoss, 0.0, result.SamplesProcessed)
		if err != nil {
			slog.Debug("ml: failed to save checkpoint", "err", err)
		}

		// check for regression and rollback if needed
		if rollbackPath := m.versioner.CheckAndRollback(result.FinalLoss, 0.0); rollbackPath != "" {
			slog.Info("ml: rolling back to best checkpoint", "path", rollbackPath)
			if err := model.Load(rollbackPath); err != nil {
				slog.Warn("ml: rollback failed", "err", err)
			}
			// re-save as current version
			model.Save(m.modelPath)
		} else if vid >= 0 {
			// save as current model
			model.Save(m.modelPath)
		}
	}
}
