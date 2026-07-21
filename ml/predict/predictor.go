package predict

import (
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// Prediction represents a single predicted action.
type Prediction struct {
	Tool           string  `json:"tool"`
	Confidence     float64 `json:"confidence"`
	ArgsJSON       string  `json:"args_json,omitempty"`
	CoordX         int     `json:"coord_x,omitempty"`          // destination (to_x)
	CoordY         int     `json:"coord_y,omitempty"`          // destination (to_y)
	FromCoordX     int     `json:"from_coord_x,omitempty"`     // source (from_x) — for drag operations
	FromCoordY     int     `json:"from_coord_y,omitempty"`     // source (from_y) — for drag operations
	WindowCategory string  `json:"window_category,omitempty"`  // predicted target window type
	WindowConf     float64 `json:"window_confidence,omitempty"` // confidence for window category
	Score          float64 `json:"score"`
	Args           *ArgPrediction `json:"args,omitempty"`
}

// SequencePrediction holds the primary prediction plus future action predictions.
type SequencePrediction struct {
	Primary Prediction   `json:"primary"`
	Next    []Prediction `json:"next,omitempty"` // future actions (length = SequenceLen)
}

// ArgPrediction holds predicted argument values.
type ArgPrediction struct {
	ScrollDir  string  `json:"scroll_dir,omitempty"`  // up, down, left, right
	KeyCategory string `json:"key_category,omitempty"` // modifier, navigation, function, alpha, numeric, special
	Confidence float64 `json:"confidence"`
}

// ArgCategories defines the argument class labels.
// Scroll directions (indices 0-3), key categories (indices 4-9).
var ArgCategories = []string{
	"scroll_up", "scroll_down", "scroll_left", "scroll_right",
	"key_modifier", "key_navigation", "key_function", "key_alpha", "key_numeric", "key_special",
}

// SeqConfidenceThreshold is the minimum confidence for a sequence step to be included.
// Steps below this threshold truncate the sequence (fallback to shorter prediction).
var SeqConfidenceThreshold = 0.15

// WindowCategories defines the window type labels for window-target prediction.
// The model predicts which category of window to focus before performing the action.
var WindowCategories = []string{
	"browser", "editor", "terminal", "file_manager", "dialog", "other",
}

// Predictor wraps a trained transformer model for inference.
type Predictor interface {
	Predict(ocrText string, topK int) ([]Prediction, error)
	PredictWithContext(ocrText string, history []string, topK int) ([]Prediction, error)
	PredictSequence(ocrText string, history []string) (*SequencePrediction, error)
	PredictWithWindow(ocrText string, windowTitle string, topK int) ([]Prediction, error)
	PredictSequenceWithWindow(ocrText string, history []string, windowTitle string) (*SequencePrediction, error)
	PredictWithWindows(ocrText string, windows []WindowInfo, topK int) ([]Prediction, error)
	PredictSequenceWithWindows(ocrText string, history []string, windows []WindowInfo) (*SequencePrediction, error)
	LoadModel(path string) error
	SetTools(tools []string)
}

// WindowInfo describes a visible window for multi-window context.
type WindowInfo struct {
	Title    string `json:"title"`
	Category string `json:"category,omitempty"` // browser, editor, terminal, etc.
	Active   bool   `json:"active"`             // is this the foreground window?
}

// Engine is the default Predictor implementation.
type Engine struct {
	model       transformer.Model
	tokenizer   tokenizer.Tokenizer
	encoder     *spatial.Encoder
	tools       []string
	maxLen      int
	argDim      int
	windowDim   int  // window category dimensions (0 = disabled)
	sequenceLen int  // number of future actions to predict (0 = disabled)
	loaded      bool
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
		model:       model,
		tokenizer:   tok,
		encoder:     enc,
		maxLen:      cfg.MaxLen,
		argDim:      cfg.ArgDim,
		windowDim:   cfg.WindowDim,
		sequenceLen: cfg.SequenceLen,
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

	// encode history actions into token sequences
	var historyTokens [][]int
	if len(history) > 0 {
		for _, action := range history {
			historyTokens = append(historyTokens, e.tokenizer.Encode(action, e.maxLen))
		}
	}

	logits, err := e.model.Forward([][]int{tokens}, [][]float64{coordFeatures}, historyTokens)
	if err != nil {
		return nil, err
	}
	if len(logits) == 0 || len(logits[0]) == 0 {
		return nil, nil
	}

	raw := logits[0]
	numTools := len(e.tools)

	// split output: tool logits | from_xy(2) | to_xy(2) | arg logits | window logits
	toolLogits := raw
	coordX, coordY := 0, 0
	fromX, fromY := 0, 0
	var argLogits []float64
	var windowLogits []float64
	argStart := numTools + 4
	windowStart := argStart + e.argDim
	if numTools+2 <= len(raw) {
		toolLogits = raw[:numTools]
		normFromX := transformer.Sigmoid(raw[numTools])
		normFromY := transformer.Sigmoid(raw[numTools+1])
		fromX, fromY = e.encoder.Decode([]float64{normFromX, normFromY})
	}
	if numTools+4 <= len(raw) {
		normToX := transformer.Sigmoid(raw[numTools+2])
		normToY := transformer.Sigmoid(raw[numTools+3])
		coordX, coordY = e.encoder.Decode([]float64{normToX, normToY})
	}
	if e.argDim > 0 && argStart+e.argDim <= len(raw) {
		argLogits = raw[argStart : argStart+e.argDim]
	}
	if e.windowDim > 0 && windowStart+e.windowDim <= len(raw) {
		windowLogits = raw[windowStart : windowStart+e.windowDim]
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
		pred := Prediction{
			Tool:       scores[i].tool,
			Score:      scores[i].score,
			Confidence: scores[i].score,
			CoordX:     coordX,
			CoordY:     coordY,
			FromCoordX: fromX,
			FromCoordY: fromY,
		}
		// decode arg predictions if available
		if argLogits != nil && len(ArgCategories) == len(argLogits) {
			argProbs := transformer.Softmax(argLogits)
			bestArgIdx := 0
			for j, p := range argProbs {
				if p > argProbs[bestArgIdx] {
					bestArgIdx = j
				}
			}
			pred.Args = &ArgPrediction{
				Confidence: argProbs[bestArgIdx],
			}
			if bestArgIdx < 4 {
				// scroll direction
				pred.Args.ScrollDir = ArgCategories[bestArgIdx][7:] // strip "scroll_"
			} else if bestArgIdx < len(ArgCategories) {
				// key category
				pred.Args.KeyCategory = ArgCategories[bestArgIdx][4:] // strip "key_"
			}
		}
		// decode window category if available
		if windowLogits != nil && len(WindowCategories) == len(windowLogits) {
			winProbs := transformer.Softmax(windowLogits)
			bestWinIdx := 0
			for j, p := range winProbs {
				if p > winProbs[bestWinIdx] {
					bestWinIdx = j
				}
			}
			pred.WindowCategory = WindowCategories[bestWinIdx]
			pred.WindowConf = winProbs[bestWinIdx]
		}
		preds[i] = pred
	}
	return preds, nil
}

// PredictWithWindow includes the active window title as context for the prediction.
// The window title is prepended to the OCR text before tokenization.
func (e *Engine) PredictWithWindow(ocrText string, windowTitle string, topK int) ([]Prediction, error) {
	combined := windowTitle + " " + ocrText
	return e.Predict(combined, topK)
}

// PredictSequenceWithWindow includes the active window title as context for sequence prediction.
// The window title is prepended to the OCR text before tokenization.
func (e *Engine) PredictSequenceWithWindow(ocrText string, history []string, windowTitle string) (*SequencePrediction, error) {
	combined := windowTitle + " " + ocrText
	return e.PredictSequence(combined, history)
}

// PredictWithWindows includes top-3 visible window titles as context for multi-window awareness.
// Windows are encoded as: "[WIN]title1 [WIN]title2 [WIN]title3 OCR_TEXT".
func (e *Engine) PredictWithWindows(ocrText string, windows []WindowInfo, topK int) ([]Prediction, error) {
	combined := encodeWindowContext(windows) + " " + ocrText
	return e.Predict(combined, topK)
}

// PredictSequenceWithWindows includes top-3 visible window titles for sequence prediction.
func (e *Engine) PredictSequenceWithWindows(ocrText string, history []string, windows []WindowInfo) (*SequencePrediction, error) {
	combined := encodeWindowContext(windows) + " " + ocrText
	return e.PredictSequence(combined, history)
}

// encodeWindowContext builds a prefix string from up to 3 visible windows.
func encodeWindowContext(windows []WindowInfo) string {
	if len(windows) == 0 {
		return ""
	}
	// limit to top 3
	n := len(windows)
	if n > 3 {
		n = 3
	}
	prefix := ""
	for i := 0; i < n; i++ {
		w := windows[i]
		if w.Active {
			prefix += "[ACTIVE]"
		}
		if w.Category != "" {
			prefix += "[" + w.Category + "]"
		}
		prefix += w.Title
		if i < n-1 {
			prefix += " "
		}
	}
	return prefix
}

// LoadModel loads a trained model checkpoint.
func (e *Engine) LoadModel(path string) error {
	return e.model.Load(path)
}

// PredictSequence returns the primary prediction plus future action predictions.
// The primary is the same as Predict(). Next contains sequenceLen future actions.
func (e *Engine) PredictSequence(ocrText string, history []string) (*SequencePrediction, error) {
	tokens := e.tokenizer.Encode(ocrText, e.maxLen)
	coordFeatures := make([]float64, e.encoder.FeatureDimValue())

	var historyTokens [][]int
	if len(history) > 0 {
		for _, action := range history {
			historyTokens = append(historyTokens, e.tokenizer.Encode(action, e.maxLen))
		}
	}

	logits, err := e.model.Forward([][]int{tokens}, [][]float64{coordFeatures}, historyTokens)
	if err != nil {
		return nil, err
	}
	if len(logits) == 0 || len(logits[0]) == 0 {
		return nil, nil
	}

	raw := logits[0]
	numTools := len(e.tools)

	// decode primary prediction
	primary := e.decodePrimary(raw, numTools)

	// decode sequence predictions (if SequenceLen > 0 and output is large enough)
	var next []Prediction
	if e.sequenceLen > 0 && numTools > 0 {
		slotDim := numTools + 2 + e.argDim // tool + to_xy + arg
		primaryDim := numTools + 4 + e.argDim
		for i := 0; i < e.sequenceLen; i++ {
			start := primaryDim + i*slotDim
			end := start + slotDim
			if end <= len(raw) {
				slotPred := e.decodeSlot(raw[start:end], numTools)
				if slotPred.Confidence < SeqConfidenceThreshold {
					break // confidence gate: truncate sequence
				}
				next = append(next, slotPred)
			}
		}
	}

	return &SequencePrediction{
		Primary: primary,
		Next:    next,
	}, nil
}

// decodePrimary extracts the primary prediction from the full output vector.
func (e *Engine) decodePrimary(raw []float64, numTools int) Prediction {
	pred := Prediction{}

	toolLogits := raw
	if numTools <= len(raw) {
		toolLogits = raw[:numTools]
	}

	// from_xy
	if numTools+2 <= len(raw) {
		normFromX := transformer.Sigmoid(raw[numTools])
		normFromY := transformer.Sigmoid(raw[numTools+1])
		pred.FromCoordX, pred.FromCoordY = e.encoder.Decode([]float64{normFromX, normFromY})
	}
	// to_xy
	if numTools+4 <= len(raw) {
		normToX := transformer.Sigmoid(raw[numTools+2])
		normToY := transformer.Sigmoid(raw[numTools+3])
		pred.CoordX, pred.CoordY = e.encoder.Decode([]float64{normToX, normToY})
	}

	// arg logits
	argStart := numTools + 4
	if e.argDim > 0 && argStart+e.argDim <= len(raw) {
		argLogits := raw[argStart : argStart+e.argDim]
		if len(ArgCategories) == len(argLogits) {
			argProbs := transformer.Softmax(argLogits)
			bestArgIdx := 0
			for j, p := range argProbs {
				if p > argProbs[bestArgIdx] {
					bestArgIdx = j
				}
			}
			pred.Args = &ArgPrediction{Confidence: argProbs[bestArgIdx]}
			if bestArgIdx < 4 {
				pred.Args.ScrollDir = ArgCategories[bestArgIdx][7:]
			} else if bestArgIdx < len(ArgCategories) {
				pred.Args.KeyCategory = ArgCategories[bestArgIdx][4:]
			}
		}
	}

	// tool probabilities
	probs := transformer.Softmax(toolLogits)
	bestIdx := 0
	for i, p := range probs {
		if p > probs[bestIdx] {
			bestIdx = i
		}
	}
	if bestIdx < numTools {
		pred.Tool = e.tools[bestIdx]
	}
	pred.Score = probs[bestIdx]
	pred.Confidence = probs[bestIdx]

	return pred
}

// decodeSlot decodes a single sequence slot from raw logits.
// Slot layout: tool(numTools) + to_xy(2) + arg(argDim).
func (e *Engine) decodeSlot(slot []float64, numTools int) Prediction {
	pred := Prediction{}

	toolLogits := slot
	if numTools <= len(slot) {
		toolLogits = slot[:numTools]
	}

	// to_xy
	if numTools+2 <= len(slot) {
		normToX := transformer.Sigmoid(slot[numTools])
		normToY := transformer.Sigmoid(slot[numTools+1])
		pred.CoordX, pred.CoordY = e.encoder.Decode([]float64{normToX, normToY})
	}

	// arg logits
	argStart := numTools + 2
	if e.argDim > 0 && argStart+e.argDim <= len(slot) {
		argLogits := slot[argStart : argStart+e.argDim]
		if len(ArgCategories) == len(argLogits) {
			argProbs := transformer.Softmax(argLogits)
			bestArgIdx := 0
			for j, p := range argProbs {
				if p > argProbs[bestArgIdx] {
					bestArgIdx = j
				}
			}
			pred.Args = &ArgPrediction{Confidence: argProbs[bestArgIdx]}
			if bestArgIdx < 4 {
				pred.Args.ScrollDir = ArgCategories[bestArgIdx][7:]
			} else if bestArgIdx < len(ArgCategories) {
				pred.Args.KeyCategory = ArgCategories[bestArgIdx][4:]
			}
		}
	}

	// tool probabilities
	probs := transformer.Softmax(toolLogits)
	bestIdx := 0
	for i, p := range probs {
		if p > probs[bestIdx] {
			bestIdx = i
		}
	}
	if bestIdx < numTools {
		pred.Tool = e.tools[bestIdx]
	}
	pred.Score = probs[bestIdx]
	pred.Confidence = probs[bestIdx]

	return pred
}
