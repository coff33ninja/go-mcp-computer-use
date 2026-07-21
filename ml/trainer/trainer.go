package trainer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

type TrainerConfig struct {
	Model        transformer.Model
	ModelConfig  transformer.Config
	Tokenizer    tokenizer.Tokenizer
	Encoder      *spatial.Encoder
	Tools        []string
	LearningRate float64
	BatchSize    int // mini-batch size (0 or 1 = online, >1 = mini-batch)
}

type EpochResult struct {
	InitialLoss      float64
	FinalLoss        float64
	SamplesProcessed int
}

type Trainer struct {
	model        transformer.Model
	tokenizer    tokenizer.Tokenizer
	encoder      *spatial.Encoder
	tools        []string
	lr           float64
	maxLen       int
	outputDim    int
	argDim       int
	windowDim    int // window category dimensions (0 = disabled)
	fromCoordDim int // 0 = single-coord tools, 2 = drag (from_x, from_y)
	sequenceLen  int // number of future actions to predict (0 = disabled)
	batchSize    int // mini-batch size (0 or 1 = online, >1 = mini-batch)
	toolStart    int // where coord+arg dims start in output (numTools + fromCoordDim + 2)
	primaryDim   int // total dims for primary prediction head
}

func NewTrainer(cfg TrainerConfig) *Trainer {
	argDim := cfg.ModelConfig.ArgDim
	windowDim := cfg.ModelConfig.WindowDim
	numTools := len(cfg.Tools)
	fromCoordDim := cfg.ModelConfig.FromCoordDim
	primaryDim := numTools + fromCoordDim + 2 + argDim + windowDim
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	return &Trainer{
		model:        cfg.Model,
		tokenizer:    cfg.Tokenizer,
		encoder:      cfg.Encoder,
		tools:        cfg.Tools,
		lr:           cfg.LearningRate,
		maxLen:       cfg.ModelConfig.MaxLen,
		outputDim:    cfg.ModelConfig.OutputDim,
		argDim:       argDim,
		windowDim:    windowDim,
		fromCoordDim: fromCoordDim,
		sequenceLen:  cfg.ModelConfig.SequenceLen,
		batchSize:    batchSize,
		toolStart:    numTools + fromCoordDim + 2,
		primaryDim:   primaryDim,
	}
}

func (t *Trainer) TrainEpoch(loader *dataloader.SQLiteLoader) (*EpochResult, error) {
	if loader == nil {
		return nil, fmt.Errorf("trainer: nil loader")
	}
	ctx := context.Background()
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("trainer: load: %w", err)
	}

	result := &EpochResult{SamplesProcessed: len(samples)}
	if len(samples) == 0 {
		return result, nil
	}

	// compute initial loss
	tokens, coords, targets := t.prepareBatch(samples[:1])
	logits, err := t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("trainer: initial forward: %w", err)
	}
	result.InitialLoss = mseLoss(logits[0], targets[0])

	// mini-batch training
	bs := t.batchSize
	if bs <= 0 {
		bs = 1
	}
	for i := 0; i < len(samples); i += bs {
		end := i + bs
		if end > len(samples) {
			end = len(samples)
		}
		batch := samples[i:end]

		// accumulate gradients over mini-batch
		for j, s := range batch {
			tokens, coords, targets := t.prepareBatch([]dataloader.Sample{s})
			_, err := t.model.Forward(tokens, coords, nil)
			if err != nil {
				return nil, fmt.Errorf("trainer: forward %d: %w", i+j, err)
			}
			if err := t.model.ForwardBackward(targets[0]); err != nil {
				return nil, fmt.Errorf("trainer: forward-backward %d: %w", i+j, err)
			}
		}
		// step solver with accumulated gradients
		if err := t.model.Step(t.lr); err != nil {
			return nil, fmt.Errorf("trainer: step %d: %w", i, err)
		}
	}

	// compute final loss
	tokens, coords, targets = t.prepareBatch(samples[:1])
	logits, err = t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("trainer: final forward: %w", err)
	}
	result.FinalLoss = mseLoss(logits[0], targets[0])

	return result, nil
}

// FinetuneEpoch trains on pre-filtered samples (for app-specific transfer learning).
// Lower learning rate recommended (0.0001) to avoid catastrophic forgetting.
func (t *Trainer) FinetuneEpoch(samples []dataloader.Sample, lr float64) (*EpochResult, error) {
	result := &EpochResult{SamplesProcessed: len(samples)}
	if len(samples) == 0 {
		return result, nil
	}

	// initial loss
	tokens, coords, targets := t.prepareBatch(samples[:1])
	logits, err := t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("finetune: initial forward: %w", err)
	}
	result.InitialLoss = mseLoss(logits[0], targets[0])

	for i, s := range samples {
		tokens, coords, targets := t.prepareBatch([]dataloader.Sample{s})
		_, err := t.model.Forward(tokens, coords, nil)
		if err != nil {
			return nil, fmt.Errorf("finetune: forward %d: %w", i, err)
		}
		if err := t.model.BackwardWithTarget(targets[0], lr); err != nil {
			return nil, fmt.Errorf("finetune: backward %d: %w", i, err)
		}
	}

	tokens, coords, targets = t.prepareBatch(samples[:1])
	logits, err = t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("finetune: final forward: %w", err)
	}
	result.FinalLoss = mseLoss(logits[0], targets[0])
	return result, nil
}

func (t *Trainer) prepareBatch(samples []dataloader.Sample) ([][]int, [][]float64, [][]float64) {
	tokens := make([][]int, len(samples))
	coords := make([][]float64, len(samples))
	targets := make([][]float64, len(samples))

	for i, s := range samples {
		tokens[i] = t.tokenizer.Encode(s.Context, t.maxLen)
		coords[i] = t.encoder.Encode(0, 0)
		targets[i] = t.makeTarget(s.Action, s.ArgsJSON)
	}
	return tokens, coords, targets
}

func (t *Trainer) makeTarget(action string, argsJSON string) []float64 {
	target := make([]float64, t.outputDim)
	// tool one-hot
	for i, tool := range t.tools {
		if i >= t.toolStart {
			break
		}
		if tool == action {
			target[i] = 1.0
			break
		}
	}
	// decode coords from ArgsJSON and normalize to 0-1
	fromX, fromY, toX, toY := decodeCoords(action, argsJSON)
	numTools := len(t.tools)
	// from_xy (only set if FromCoordDim > 0)
	if t.fromCoordDim > 0 {
		fromFeatures := t.encoder.Encode(int(fromX), int(fromY))
		target[numTools] = fromFeatures[0]
		target[numTools+1] = fromFeatures[1]
	}
	// to_xy (always set)
	toFeatures := t.encoder.Encode(int(toX), int(toY))
	target[numTools+t.fromCoordDim] = toFeatures[0]
	target[numTools+t.fromCoordDim+1] = toFeatures[1]
	// arg one-hot (scroll direction, key category)
	if t.argDim > 0 && t.toolStart+t.argDim <= len(target) {
		argIdx := decodeArgIndex(action, argsJSON)
		if argIdx >= 0 && argIdx < t.argDim {
			target[t.toolStart+argIdx] = 1.0
		}
	}
	// window category one-hot (default "other" = last category)
	if t.windowDim > 0 {
		windowStart := t.toolStart + t.argDim
		if windowStart+t.windowDim <= len(target) {
			target[windowStart+t.windowDim-1] = 1.0 // "other" = last index
		}
	}
	return target
}

// makeSequenceTargets fills the sequence section of the target vector with future action targets.
// actions is a slice of (action, argsJSON) pairs for the next N actions.
// The sequence section starts at primaryDim and each slot has slotDim dims.
func (t *Trainer) makeSequenceTargets(target []float64, actions []struct {
	Action  string
	ArgsJSON string
}) {
	if t.sequenceLen <= 0 || t.primaryDim >= len(target) {
		return
	}
	numTools := len(t.tools)
	slotDim := numTools + 2 + t.argDim // tool + to_xy + arg (no from_xy in sequences)
	for i := 0; i < t.sequenceLen && i < len(actions); i++ {
		slotStart := t.primaryDim + i*slotDim
		if slotStart+slotDim > len(target) {
			break
		}
		action := actions[i]
		// tool one-hot in slot
		for j, tool := range t.tools {
			if tool == action.Action {
				target[slotStart+j] = 1.0
				break
			}
		}
		// to_xy in slot
		_, _, toX, toY := decodeCoords(action.Action, action.ArgsJSON)
		toFeatures := t.encoder.Encode(int(toX), int(toY))
		target[slotStart+numTools] = toFeatures[0]
		target[slotStart+numTools+1] = toFeatures[1]
		// arg in slot
		if t.argDim > 0 {
			argIdx := decodeArgIndex(action.Action, action.ArgsJSON)
			if argIdx >= 0 && argIdx < t.argDim {
				target[slotStart+numTools+2+argIdx] = 1.0
			}
		}
	}
}

// decodeCoords extracts coordinate values from ArgsJSON for different action types.
// Returns (fromX, fromY, toX, toY). For single-coord actions, from == to.
func decodeCoords(action string, argsJSON string) (float64, float64, float64, float64) {
	var args struct {
		X     int `json:"x"`
		Y     int `json:"y"`
		FromX int `json:"from_x"`
		FromY int `json:"from_y"`
		ToX   int `json:"to_x"`
		ToY   int `json:"to_y"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return 0, 0, 0, 0
	}
	switch action {
	case "drag", "drag_and_drop":
		// drag uses from_x/from_y + to_x/to_y
		return float64(args.FromX), float64(args.FromY), float64(args.ToX), float64(args.ToY)
	default:
		// click, hover, scroll, etc. use x/y as destination
		return float64(args.X), float64(args.Y), float64(args.X), float64(args.Y)
	}
}

// decodeArgIndex returns the index into ArgCategories for the given action+args.
// Returns -1 if no arg mapping is available.
func decodeArgIndex(action string, argsJSON string) int {
	switch action {
	case "scroll":
		var args struct {
			Clicks int `json:"clicks"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
			if args.Clicks > 0 {
				return 0 // scroll_up
			}
			return 1 // scroll_down
		}
		return 1 // default down
	case "key_press":
		var args struct {
			Keys []string `json:"keys"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err == nil && len(args.Keys) > 0 {
			return classifyKey(args.Keys[0])
		}
	}
	return -1
}

// classifyKey maps a key name to one of the ArgCategories key indices (4-9).
func classifyKey(key string) int {
	k := strings.ToLower(key)
	switch {
	case k == "ctrl" || k == "alt" || k == "shift" || k == "win" || k == "meta" ||
		strings.HasPrefix(k, "ctrl+") || strings.HasPrefix(k, "alt+") || strings.HasPrefix(k, "shift+"):
		return 4 // modifier
	case k == "up" || k == "down" || k == "left" || k == "right" ||
		k == "home" || k == "end" || k == "pageup" || k == "pagedown" || k == "tab":
		return 5 // navigation
	case k == "f1" || k == "f2" || k == "f3" || k == "f4" || k == "f5" || k == "f6" ||
		k == "f7" || k == "f8" || k == "f9" || k == "f10" || k == "f11" || k == "f12":
		return 6 // function
	case len(k) == 1 && k >= "a" && k <= "z":
		return 7 // alpha
	case len(k) == 1 && k >= "0" && k <= "9":
		return 8 // numeric
	default:
		return 9 // special (enter, space, escape, backspace, delete, etc.)
	}
}

func (t *Trainer) SaveModel(path string) error {
	return t.model.Save(path)
}

func (t *Trainer) LoadModel(path string) error {
	return t.model.Load(path)
}

func mseLoss(pred, target []float64) float64 {
	n := len(pred)
	if len(target) < n {
		n = len(target)
	}
	var sum float64
	for i := 0; i < n; i++ {
		d := pred[i] - target[i]
		sum += d * d
	}
	return sum / float64(n)
}
