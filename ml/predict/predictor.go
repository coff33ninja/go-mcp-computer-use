package predict

import (
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// Prediction represents a single predicted action.
type Prediction struct {
	Tool       string  `json:"tool"`
	Confidence float64 `json:"confidence"`
	ArgsJSON   string  `json:"args_json,omitempty"`
	CoordX     int     `json:"coord_x,omitempty"`
	CoordY     int     `json:"coord_y,omitempty"`
	Score      float64 `json:"score"`
}

// Predictor wraps a trained transformer model for inference.
type Predictor interface {
	Predict(ocrText string, topK int) ([]Prediction, error)
	PredictWithContext(ocrText string, history []string, topK int) ([]Prediction, error)
	LoadModel(path string) error
	SetTools(tools []string)
}

// Engine is the default Predictor implementation.
type Engine struct {
	model     transformer.Model
	tokenizer tokenizer.Tokenizer
	encoder   *spatial.Encoder
	tools     []string
	maxLen    int
	loaded    bool
}

// NewEngine creates a new prediction engine.
func NewEngine(
	model transformer.Model,
	tok tokenizer.Tokenizer,
	enc *spatial.Encoder,
) *Engine {
	return &Engine{
		model:     model,
		tokenizer: tok,
		encoder:   enc,
		maxLen:    128,
	}
}

// NewEngineWithConfig creates a new prediction engine with the given model config.
func NewEngineWithConfig(
	model transformer.Model,
	tok tokenizer.Tokenizer,
	enc *spatial.Encoder,
	cfg transformer.Config,
) *Engine {
	return &Engine{
		model:     model,
		tokenizer: tok,
		encoder:   enc,
		maxLen:    cfg.MaxLen,
	}
}

// SetTools defines the mapping from output indices to tool names.
func (e *Engine) SetTools(tools []string) {
	e.tools = tools
}

// Predict returns the top-k predicted actions given OCR text.
func (e *Engine) Predict(ocrText string, topK int) ([]Prediction, error) {
	return e.PredictWithContext(ocrText, nil, topK)
}

// PredictWithContext includes preceding action history for better predictions.
func (e *Engine) PredictWithContext(ocrText string, history []string, topK int) ([]Prediction, error) {
	tokens := e.tokenizer.Encode(ocrText, e.maxLen)
	coordFeatures := make([]float64, e.encoder.FeatureDimValue())
	logits, err := e.model.Forward([][]int{tokens}, [][]float64{coordFeatures})
	if err != nil {
		return nil, err
	}
	if len(logits) == 0 || len(logits[0]) == 0 {
		return nil, nil
	}

	raw := logits[0]
	numTools := len(e.tools)

	// split output: tool logits → softmax → probabilities; coord logits → sigmoid → pixels
	toolLogits := raw
	coordX, coordY := 0, 0
	if numTools+2 <= len(raw) {
		toolLogits = raw[:numTools]
		normX := transformer.Sigmoid(raw[numTools])
		normY := transformer.Sigmoid(raw[numTools+1])
		coordX, coordY = e.encoder.Decode([]float64{normX, normY})
	}

	probs := transformer.Softmax(toolLogits)

	type scored struct {
		tool  string
		score float64
		idx   int
	}
	var scores []scored
	for i, p := range scores {
		_ = i
		_ = p
	}
	scores = scores[:0]
	for i, p := range probs {
		tool := ""
		if i < numTools {
			tool = e.tools[i]
		}
		scores = append(scores, scored{tool: tool, score: p, idx: i})
	}

	// sort by score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if topK > len(scores) {
		topK = len(scores)
	}
	preds := make([]Prediction, topK)
	for i := 0; i < topK; i++ {
		preds[i] = Prediction{
			Tool:       scores[i].tool,
			Score:      scores[i].score,
			Confidence: scores[i].score,
			CoordX:     coordX,
			CoordY:     coordY,
		}
	}
	return preds, nil
}

// LoadModel loads a trained model checkpoint.
func (e *Engine) LoadModel(path string) error {
	return e.model.Load(path)
}
