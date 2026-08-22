package actions

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
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
	// mouse & keyboard (11)
	"click", "move_mouse", "scroll", "drag",
	"type", "key_press", "key_down", "key_up",
	"hover", "focus_window", "launch_and_wait",
	// window management (6)
	"close_window", "minimize_window", "maximize_window",
	"restore_window", "move_window", "find_window",
	// clipboard & text (4)
	"set_clipboard", "get_clipboard", "type_and_submit", "select_all_and_type",
	// utility (4)
	"ocr", "screenshot", "click_menu_item", "launch_app",
}

// MLEngine wraps the Go-native transformer model for UI automation prediction.
// It trains from the same training_pairs SQLite table as the statistical engine
// and provides transformer-based predictions as a primary source, with the
// statistical engine as fallback.
type MLEngine struct {
	model     transformer.Model
	predictor *predict.Engine
	tok       *tokenizer.SimpleTokenizer
	encoder   *spatial.Encoder
	modelPath string
	dbPath    string
	screenCfg spatial.ScreenConfig
	tools     []string
	replayBuf *online.ReplayBuffer
	versioner *versioning.ModelVersion
	trainDone chan struct{}
	mu        sync.RWMutex
	ready     bool
	// app-specific models for transfer learning
	appModels map[string]*predict.Engine // keyed by normalized app name
	appDir    string                     // directory for app model checkpoints
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

	// Enumerate all monitors with per-monitor DPI
	displays, _ := ListDisplays()
	monDPIs, _ := ListMonitorDPIs()
	if len(displays) > 0 {
		// Build a map from device name to DPI
		dpiMap := make(map[string]float64)
		for _, md := range monDPIs {
			dpiMap[md.Name] = float64(md.DPI) / 96.0
		}
		monitors := make([]spatial.MonitorInfo, 0, len(displays))
		for _, d := range displays {
			monDPI := dpiMap[d.Name]
			if monDPI == 0 {
				monDPI = scale // fallback to desktop-wide DPI
			}
			monitors = append(monitors, spatial.MonitorInfo{
				X:        int(d.PositionX),
				Y:        int(d.PositionY),
				Width:    int(d.Width),
				Height:   int(d.Height),
				DPIScale: monDPI,
				Primary:  d.Primary,
			})
		}
		cfg.Monitors = monitors
		slog.Debug("ml: detected monitors", "count", len(monitors))
		for i, m := range monitors {
			slog.Debug("ml: monitor",
				"index", i,
				"rect", fmt.Sprintf("%dx%d@(%d,%d)", m.Width, m.Height, m.X, m.Y),
				"dpi_scale", fmt.Sprintf("%.2f", m.DPIScale),
				"primary", m.Primary)
		}
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
		appModels: make(map[string]*predict.Engine),
		appDir:    filepath.Join(dataDir, "app_models"),
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
		VocabSize:    2000,
		MaxLen:       128,
		EmbedDim:     64,
		NumHeads:     2,
		NumLayers:    2,
		FFNDim:       128,
		CoordDim:     spatial.FeatureDim,
		OutputDim:    len(m.tools) + 4 + 10 + 6, // tool + from_xy(2) + to_xy(2) + arg(10) + window(6)
		FromCoordDim: 2,
		WindowDim:    6,
		HistoryLen:   5,
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

	// hold out 10% for evaluation before augmenting so eval stays honest
	nTest := len(samples) / 10
	if nTest == 0 {
		nTest = 1
	}
	testSamples := samples[:nTest]
	trainSamples := samples[nTest:]

	aug := dataloader.NewAugmentor()
	augmented := aug.AugmentAll(trainSamples, 2) // 2 augmented per original

	tok := tokenizer.NewSimpleTokenizer()
	corpus := make([]string, 0, len(augmented)+len(testSamples))
	for _, s := range augmented {
		corpus = append(corpus, s.Context)
	}
	for _, s := range testSamples {
		corpus = append(corpus, s.Context)
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

	// multiple passes — a single epoch barely moves the loss
	const epochs = 5
	var result *trainer.EpochResult
	for e := 1; e <= epochs; e++ {
		result, err = tr.TrainSamples(augmented)
		if err != nil {
			return fmt.Errorf("ml: train epoch %d: %w", e, err)
		}
	}

	evalLoss := tr.Evaluate(testSamples)
	accuracy := tr.Accuracy(testSamples)
	if result.SamplesProcessed > 0 {
		slog.Info("ml: training complete",
			"epochs", epochs,
			"loss_train_final", fmt.Sprintf("%.4f", result.FinalLoss),
			"loss_eval", fmt.Sprintf("%.4f", evalLoss),
			"accuracy", fmt.Sprintf("%.2f%%", accuracy*100),
			"samples_train", result.SamplesProcessed,
			"samples_eval", len(testSamples),
		)
	}

	if err := tr.SaveModel(m.modelPath); err != nil {
		return fmt.Errorf("ml: save model: %w", err)
	}

	// save versioned checkpoint with real eval numbers so rollback logic works
	if _, err := m.versioner.SaveCheckpoint(mdl, evalLoss, accuracy, result.SamplesProcessed); err != nil {
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
		VocabSize:    2000,
		MaxLen:       128,
		EmbedDim:     64,
		NumHeads:     2,
		NumLayers:    2,
		FFNDim:       128,
		CoordDim:     spatial.FeatureDim,
		OutputDim:    len(m.tools) + 4 + 10 + 6, // tool + from_xy(2) + to_xy(2) + arg(10) + window(6)
		FromCoordDim: 2,
		WindowDim:    6,
		HistoryLen:   5,
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
		if p.FromCoordX != 0 || p.FromCoordY != 0 {
			pa.FromCoord = &PredictedCoord{
				X:          p.FromCoordX,
				Y:          p.FromCoordY,
				Confidence: p.Score,
				Samples:    1,
			}
		}
		if p.Args != nil {
			pa.Args = &PredictedArgs{
				ScrollDir:   p.Args.ScrollDir,
				KeyCategory: p.Args.KeyCategory,
			}
		}
		results = append(results, pa)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// SequencePredictionResult holds the primary prediction plus future actions.
type SequencePredictionResult struct {
	Primary PredictedAction   `json:"primary"`
	Next    []PredictedAction `json:"next,omitempty"`
}

// PredictSequence returns the primary prediction plus predicted future actions.
// The primary is the same as Predict(). Next contains SequenceLen future actions.
func (m *MLEngine) PredictSequence(ocrText string, history []string) *SequencePredictionResult {
	if !m.IsReady() {
		return nil
	}

	m.mu.RLock()
	pred := m.predictor
	m.mu.RUnlock()
	if pred == nil {
		return nil
	}

	mlPred, err := pred.PredictSequence(ocrText, history)
	if err != nil {
		slog.Debug("ml: predict_sequence error", "err", err)
		return nil
	}
	if mlPred == nil {
		return nil
	}

	result := &SequencePredictionResult{
		Primary: mlPredictedAction(mlPred.Primary),
	}
	for _, np := range mlPred.Next {
		result.Next = append(result.Next, mlPredictedAction(np))
	}
	return result
}

// PredictWithWindow includes the active window title as context for the prediction.
func (m *MLEngine) PredictWithWindow(ocrText string, windowTitle string, limit int, history []string) []PredictedAction {
	if !m.IsReady() {
		return nil
	}

	m.mu.RLock()
	pred := m.predictor
	m.mu.RUnlock()
	if pred == nil {
		return nil
	}

	predictions, err := pred.PredictWithWindow(ocrText, windowTitle, limit)
	if err != nil {
		slog.Debug("ml: predict_with_window error", "err", err)
		return nil
	}

	result := make([]PredictedAction, 0, len(predictions))
	for _, p := range predictions {
		result = append(result, mlPredictedAction(p))
	}
	return result
}

// PredictSequenceWithWindow includes the active window title as context for sequence prediction.
func (m *MLEngine) PredictSequenceWithWindow(ocrText string, history []string, windowTitle string) *SequencePredictionResult {
	if !m.IsReady() {
		return nil
	}

	m.mu.RLock()
	pred := m.predictor
	m.mu.RUnlock()
	if pred == nil {
		return nil
	}

	mlPred, err := pred.PredictSequenceWithWindow(ocrText, history, windowTitle)
	if err != nil {
		slog.Debug("ml: predict_sequence_with_window error", "err", err)
		return nil
	}
	if mlPred == nil {
		return nil
	}

	result := &SequencePredictionResult{
		Primary: mlPredictedAction(mlPred.Primary),
	}
	for _, np := range mlPred.Next {
		result.Next = append(result.Next, mlPredictedAction(np))
	}
	return result
}

// WindowInfoForPrediction describes a visible window for multi-window prediction context.
type WindowInfoForPrediction struct {
	Title    string `json:"title"`
	Category string `json:"category,omitempty"`
	Active   bool   `json:"active"`
}

// PredictWithWindows includes top-3 visible window titles as context for ML predictions.
func (m *MLEngine) PredictWithWindows(ocrText string, windows []WindowInfoForPrediction, limit int, history []string) []PredictedAction {
	if !m.IsReady() {
		return nil
	}

	m.mu.RLock()
	pred := m.predictor
	m.mu.RUnlock()
	if pred == nil {
		return nil
	}

	// convert to predict.WindowInfo
	pw := make([]predict.WindowInfo, len(windows))
	for i, w := range windows {
		pw[i] = predict.WindowInfo{Title: w.Title, Category: w.Category, Active: w.Active}
	}

	predictions, err := pred.PredictWithWindows(ocrText, pw, limit)
	if err != nil {
		slog.Debug("ml: predict_with_windows error", "err", err)
		return nil
	}

	result := make([]PredictedAction, 0, len(predictions))
	for _, p := range predictions {
		result = append(result, mlPredictedAction(p))
	}
	return result
}

// PredictSequenceWithWindows includes top-3 visible window titles for sequence prediction.
func (m *MLEngine) PredictSequenceWithWindows(ocrText string, history []string, windows []WindowInfoForPrediction) *SequencePredictionResult {
	if !m.IsReady() {
		return nil
	}

	m.mu.RLock()
	pred := m.predictor
	m.mu.RUnlock()
	if pred == nil {
		return nil
	}

	pw := make([]predict.WindowInfo, len(windows))
	for i, w := range windows {
		pw[i] = predict.WindowInfo{Title: w.Title, Category: w.Category, Active: w.Active}
	}

	mlPred, err := pred.PredictSequenceWithWindows(ocrText, history, pw)
	if err != nil {
		slog.Debug("ml: predict_sequence_with_windows error", "err", err)
		return nil
	}
	if mlPred == nil {
		return nil
	}

	result := &SequencePredictionResult{
		Primary: mlPredictedAction(mlPred.Primary),
	}
	for _, np := range mlPred.Next {
		result.Next = append(result.Next, mlPredictedAction(np))
	}
	return result
}

func mlPredictedAction(p predict.Prediction) PredictedAction {
	return PredictedAction{
		Command:    p.Tool,
		Confidence: math.Round(p.Score*100) / 100,
		SampleSize: 1,
		Coord:      predictedCoordFromPrediction(p.CoordX, p.CoordY, p.Score),
		FromCoord:  predictedCoordFromPrediction(p.FromCoordX, p.FromCoordY, p.Score),
		Args:       predictedArgsFromPrediction(p.Args),
	}
}

func predictedCoordFromPrediction(x, y int, score float64) *PredictedCoord {
	if x == 0 && y == 0 {
		return nil
	}
	return &PredictedCoord{
		X:          x,
		Y:          y,
		Confidence: score,
		Samples:    1,
	}
}

func predictedArgsFromPrediction(args *predict.ArgPrediction) *PredictedArgs {
	if args == nil {
		return nil
	}
	return &PredictedArgs{
		ScrollDir:   args.ScrollDir,
		KeyCategory: args.KeyCategory,
	}
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
		Context: ctx,
		Action:  action,
		Success: success,
		CoordX:  x,
		CoordY:  y,
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
		VocabSize:    2000,
		MaxLen:       128,
		EmbedDim:     64,
		NumHeads:     2,
		NumLayers:    2,
		FFNDim:       128,
		CoordDim:     spatial.FeatureDim,
		OutputDim:    len(m.tools) + 4 + 10 + 6, // tool + from_xy(2) + to_xy(2) + arg(10) + window(6)
		FromCoordDim: 2,
		WindowDim:    6,
		HistoryLen:   5,
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

// FinetuneForApp loads the base model, fine-tunes it on app-specific samples
// (filtered by window title), and saves as app_model_{appName}.gob.
// Returns the number of samples used or an error.
func (m *MLEngine) FinetuneForApp(appName string, epochs int) (int, error) {
	if appName == "" {
		return 0, fmt.Errorf("ml: empty app name")
	}
	normalized := normalizeAppName(appName)

	m.mu.RLock()
	model := m.model
	tok := m.tok
	enc := m.encoder
	m.mu.RUnlock()

	if model == nil || tok == nil || enc == nil {
		return 0, fmt.Errorf("ml: base model not ready")
	}

	// load base model fresh
	cfg := transformer.Config{
		VocabSize:  2000,
		MaxLen:     128,
		EmbedDim:   64,
		NumHeads:   2,
		NumLayers:  2,
		FFNDim:     128,
		CoordDim:   spatial.FeatureDim,
		OutputDim:  len(m.tools) + 2 + 10,
		HistoryLen: 5,
	}
	freshModel, err := transformer.New(cfg)
	if err != nil {
		return 0, fmt.Errorf("ml: create finetune model: %w", err)
	}
	if err := freshModel.Load(m.modelPath); err != nil {
		return 0, fmt.Errorf("ml: load base model: %w", err)
	}

	// load app-specific samples
	loader := dataloader.NewSQLiteLoader(m.dbPath)
	defer loader.Close()
	ctx := context.Background()
	allSamples, err := loader.LoadAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("ml: load samples: %w", err)
	}

	var appSamples []dataloader.Sample
	for _, s := range allSamples {
		if normalizeAppName(s.WindowTitle) == normalized {
			appSamples = append(appSamples, s)
		}
	}
	if len(appSamples) == 0 {
		return 0, fmt.Errorf("ml: no samples for app %q", appName)
	}

	// finetune with lower learning rate to avoid catastrophic forgetting
	tr := trainer.NewTrainer(trainer.TrainerConfig{
		Model:        freshModel,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        m.tools,
		LearningRate: 0.0001,
	})

	var lastResult *trainer.EpochResult
	for i := 0; i < epochs; i++ {
		result, err := tr.FinetuneEpoch(appSamples, 0.0001)
		if err != nil {
			return 0, fmt.Errorf("ml: finetune epoch %d: %w", i, err)
		}
		lastResult = result
	}

	// save app model
	if err := os.MkdirAll(m.appDir, 0755); err != nil {
		return 0, fmt.Errorf("ml: create app dir: %w", err)
	}
	appPath := filepath.Join(m.appDir, "model_"+normalized+".gob")
	if err := freshModel.Save(appPath); err != nil {
		return 0, fmt.Errorf("ml: save app model: %w", err)
	}

	slog.Info("ml: finetuned app model",
		"app", appName, "samples", len(appSamples), "epochs", epochs,
		"loss", fmt.Sprintf("%.4f", lastResult.FinalLoss))

	// load into predictor cache
	appPred := predict.NewEngineWithConfig(freshModel, tok, enc, cfg)
	appPred.SetTools(m.tools)

	m.mu.Lock()
	m.appModels[normalized] = appPred
	m.mu.Unlock()

	return len(appSamples), nil
}

// PredictForApp uses the app-specific model if available, else falls back to base.
func (m *MLEngine) PredictForApp(ocrText, appName string, limit int, history []string) []PredictedAction {
	normalized := normalizeAppName(appName)

	m.mu.RLock()
	appPred, ok := m.appModels[normalized]
	m.mu.RUnlock()

	if !ok {
		return m.Predict(ocrText, limit, history)
	}

	preds, err := appPred.PredictWithContext(ocrText, history, limit)
	if err != nil {
		return m.Predict(ocrText, limit, history)
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
		if p.Args != nil {
			pa.Args = &PredictedArgs{
				ScrollDir:   p.Args.ScrollDir,
				KeyCategory: p.Args.KeyCategory,
			}
		}
		results = append(results, pa)
	}

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// LoadAppModels loads all app-specific models from the app_models directory.
func (m *MLEngine) LoadAppModels() int {
	entries, err := os.ReadDir(m.appDir)
	if err != nil {
		return 0
	}

	cfg := transformer.Config{
		VocabSize:  2000,
		MaxLen:     128,
		EmbedDim:   64,
		NumHeads:   2,
		NumLayers:  2,
		FFNDim:     128,
		CoordDim:   spatial.FeatureDim,
		OutputDim:  len(m.tools) + 2 + 10,
		HistoryLen: 5,
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "model_") || !strings.HasSuffix(name, ".gob") {
			continue
		}
		normalized := strings.TrimPrefix(name, "model_")
		normalized = strings.TrimSuffix(normalized, ".gob")

		mdl, err := transformer.New(cfg)
		if err != nil {
			slog.Debug("ml: failed to create app model", "app", normalized, "err", err)
			continue
		}
		path := filepath.Join(m.appDir, name)
		if err := mdl.Load(path); err != nil {
			slog.Debug("ml: failed to load app model", "app", normalized, "path", path, "err", err)
			continue
		}

		pred := predict.NewEngineWithConfig(mdl, m.tok, m.encoder, cfg)
		pred.SetTools(m.tools)

		m.mu.Lock()
		m.appModels[normalized] = pred
		m.mu.Unlock()
		count++
		slog.Debug("ml: loaded app model", "app", normalized, "path", path)
	}
	return count
}

// normalizeAppName converts an app/window title to a safe filename key.
func normalizeAppName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	// collapse multiple underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.Trim(name, "_")
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}
