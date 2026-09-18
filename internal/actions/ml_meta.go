package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// MLStatus reports whether the transformer path is actually usable.
type MLStatus struct {
	Ready            bool    `json:"ready"`
	ModelLoaded      bool    `json:"model_loaded"`
	VocabLoaded      bool    `json:"vocab_loaded"`
	VocabSize        int     `json:"vocab_size"`
	ModelPath        string  `json:"model_path"`
	VocabPath        string  `json:"vocab_path"`
	LastError        string  `json:"last_error,omitempty"`
	LastEvalLoss     float64 `json:"last_eval_loss"`
	LastEvalAccuracy float64 `json:"last_eval_accuracy"`
	MajorityBaseline float64 `json:"majority_baseline"`
	TrainSamples     int     `json:"train_samples"`
	LastTrainAt      string  `json:"last_train_at,omitempty"`
	Source           string  `json:"source"` // transformer | statistical | unavailable
	Notes            string  `json:"notes,omitempty"`
}

type mlMeta struct {
	VocabSize        int     `json:"vocab_size"`
	ArgDim           int     `json:"arg_dim"`
	FromCoordDim     int     `json:"from_coord_dim"`
	WindowDim        int     `json:"window_dim"`
	HistoryLen       int     `json:"history_len"`
	NumTools         int     `json:"num_tools"`
	LastEvalLoss     float64 `json:"last_eval_loss"`
	LastEvalAccuracy float64 `json:"last_eval_accuracy"`
	MajorityBaseline float64 `json:"majority_baseline"`
	TrainSamples     int     `json:"train_samples"`
	LastTrainAt      string  `json:"last_train_at"`
	Tools            []string `json:"tools"`
}

// modelConfigFor builds a consistent transformer config.
// VocabSize is raised to cover the fitted tokenizer so embeddings are not dropped.
func modelConfigFor(tools []string, vocabSize int) transformer.Config {
	if vocabSize < 2048 {
		vocabSize = 2048
	}
	// leave headroom for online tokens
	vocabSize = ((vocabSize + 255) / 256) * 256
	argDim := len(predictArgCategories)
	windowDim := len(predictWindowCategories)
	fromCoordDim := 2
	return transformer.Config{
		VocabSize:    vocabSize,
		MaxLen:       128,
		EmbedDim:     64,
		NumHeads:     2,
		NumLayers:    2,
		FFNDim:       128,
		CoordDim:     spatial.FeatureDim,
		ArgDim:       argDim,
		FromCoordDim: fromCoordDim,
		WindowDim:    windowDim,
		HistoryLen:   5,
		OutputDim:    len(tools) + fromCoordDim + 2 + argDim + windowDim,
	}
}

var (
	predictArgCategories    = []string{"scroll_up", "scroll_down", "scroll_left", "scroll_right", "key_modifier", "key_navigation", "key_function", "key_alpha", "key_numeric", "key_special"}
	predictWindowCategories = []string{"browser", "editor", "terminal", "file_manager", "dialog", "other"}
)

func (m *MLEngine) vocabPath() string {
	return filepath.Join(filepath.Dir(m.modelPath), "vocab.bin")
}

func (m *MLEngine) metaPath() string {
	return filepath.Join(filepath.Dir(m.modelPath), "ml_meta.json")
}

func (m *MLEngine) saveMeta(meta mlMeta) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.metaPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.metaPath())
}

func (m *MLEngine) loadMeta() (mlMeta, bool) {
	var meta mlMeta
	b, err := os.ReadFile(m.metaPath())
	if err != nil {
		return meta, false
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return meta, false
	}
	return meta, true
}

// Status returns a truthful snapshot of transformer usability.
func (m *MLEngine) Status() MLStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st := MLStatus{
		Ready:            m.ready,
		ModelLoaded:      m.model != nil,
		VocabLoaded:      m.vocabLoaded,
		ModelPath:        m.modelPath,
		VocabPath:        m.vocabPath(),
		LastError:        m.lastErr,
		LastEvalLoss:     m.lastEvalLoss,
		LastEvalAccuracy: m.lastEvalAcc,
		MajorityBaseline: m.lastMajority,
		TrainSamples:     m.lastTrainSamples,
	}
	if m.tok != nil {
		st.VocabSize = m.tok.VocabSize()
	}
	if meta, ok := m.loadMeta(); ok {
		if st.LastEvalLoss == 0 {
			st.LastEvalLoss = meta.LastEvalLoss
		}
		if st.LastEvalAccuracy == 0 {
			st.LastEvalAccuracy = meta.LastEvalAccuracy
		}
		if st.MajorityBaseline == 0 {
			st.MajorityBaseline = meta.MajorityBaseline
		}
		if st.TrainSamples == 0 {
			st.TrainSamples = meta.TrainSamples
		}
		st.LastTrainAt = meta.LastTrainAt
	}
	switch {
	case m.ready && m.vocabLoaded && m.model != nil:
		st.Source = "transformer"
		if st.MajorityBaseline > 0 && st.LastEvalAccuracy < st.MajorityBaseline {
			st.Source = "statistical_preferred"
			st.Notes = fmt.Sprintf(
				"transformer holdout acc %.2f%% < majority baseline %.2f%% — statistical ml_query/agent_suggest fallback is preferred until more/better training_pairs exist",
				st.LastEvalAccuracy*100, st.MajorityBaseline*100,
			)
		} else if st.LastEvalAccuracy > 0 && st.MajorityBaseline > 0 && st.LastEvalAccuracy >= st.MajorityBaseline {
			st.Notes = fmt.Sprintf(
				"transformer holdout acc %.2f%% vs majority %.2f%%",
				st.LastEvalAccuracy*100, st.MajorityBaseline*100,
			)
		}
	case m.model != nil && !m.vocabLoaded:
		st.Source = "unavailable"
		st.Notes = "model weights present but tokenizer vocab missing — run agent_train or scripts/eval-ml.ps1 -Train"
	default:
		st.Source = "unavailable"
		st.Notes = "transformer not ready — statistical adaptive engine still available via ml_query / agent_suggest"
	}
	return st
}

// MLSourceTag labels a prediction origin.
const (
	MLSourceTransformer  = "transformer"
	MLSourceStatistical  = "statistical"
	MLSourceUnavailable  = "unavailable"
)

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }
