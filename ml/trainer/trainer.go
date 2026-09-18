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
	return t.TrainSamples(samples)
}

// TrainSamples runs one training pass over an in-memory sample set,
// letting callers pre-filter or augment samples before training.
func (t *Trainer) TrainSamples(samples []dataloader.Sample) (*EpochResult, error) {
	result := &EpochResult{SamplesProcessed: len(samples)}
	if len(samples) == 0 {
		return result, nil
	}

	// compute initial loss
	tokens, coords, targets := t.prepareBatch(samples, 0)
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
		for j := range batch {
			tokens, coords, targets := t.prepareBatch(samples, i+j)
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
	tokens, coords, targets = t.prepareBatch(samples, 0)
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
	tokens, coords, targets := t.prepareBatch(samples, 0)
	logits, err := t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("finetune: initial forward: %w", err)
	}
	result.InitialLoss = mseLoss(logits[0], targets[0])

	for i := range samples {
		tokens, coords, targets := t.prepareBatch(samples, i)
		_, err := t.model.Forward(tokens, coords, nil)
		if err != nil {
			return nil, fmt.Errorf("finetune: forward %d: %w", i, err)
		}
		if err := t.model.BackwardWithTarget(targets[0], lr); err != nil {
			return nil, fmt.Errorf("finetune: backward %d: %w", i, err)
		}
	}

	tokens, coords, targets = t.prepareBatch(samples, 0)
	logits, err = t.model.Forward(tokens, coords, nil)
	if err != nil {
		return nil, fmt.Errorf("finetune: final forward: %w", err)
	}
	result.FinalLoss = mseLoss(logits[0], targets[0])
	return result, nil
}

func (t *Trainer) prepareBatch(samples []dataloader.Sample, idx int) ([][]int, [][]float64, [][]float64) {
	s := samples[idx]
	tokens := [][]int{t.tokenizer.Encode(s.Context, t.maxLen)}
	// Feed real click/drag coordinates into the spatial encoder.
	// Previously this was always Encode(0,0), so the model never saw position.
	sx, sy := s.CoordX, s.CoordY
	fx, fy := s.FromCoordX, s.FromCoordY
	if sx == 0 && sy == 0 {
		_, _, dx, dy := decodeCoords(s.Action, s.ArgsJSON)
		sx, sy = int(dx), int(dy)
	}
	coords := [][]float64{t.encoder.Encode(sx, sy)}
	_ = fx
	_ = fy
	target := t.makeTargetFromSample(s)

	// fill sequence section if sequence training is enabled
	if t.sequenceLen > 0 && idx+t.sequenceLen <= len(samples) {
		future := make([]struct {
			Action   string
			ArgsJSON string
		}, t.sequenceLen)
		for k := 0; k < t.sequenceLen; k++ {
			future[k].Action = samples[idx+k].Action
			future[k].ArgsJSON = samples[idx+k].ArgsJSON
		}
		t.makeSequenceTargets(target, future)
	}

	return tokens, coords, [][]float64{target}
}

func (t *Trainer) makeTargetFromSample(s dataloader.Sample) []float64 {
	// Prefer loader-normalized coords; fall back to decoding args JSON.
	fromX, fromY, toX, toY := float64(s.FromCoordX), float64(s.FromCoordY), float64(s.CoordX), float64(s.CoordY)
	if toX == 0 && toY == 0 {
		fromX, fromY, toX, toY = decodeCoords(s.Action, s.ArgsJSON)
	}
	if s.Action == "drag" || s.Action == "drag_and_drop" {
		if fromX == 0 && fromY == 0 && s.CoordX != 0 {
			fromX, fromY = float64(s.CoordX), float64(s.CoordY)
		}
	} else if fromX == 0 && fromY == 0 {
		fromX, fromY = toX, toY
	}
	return t.makeTargetWithCoords(s.Action, s.ArgsJSON, fromX, fromY, toX, toY)
}

func (t *Trainer) makeTarget(action string, argsJSON string) []float64 {
	fromX, fromY, toX, toY := decodeCoords(action, argsJSON)
	return t.makeTargetWithCoords(action, argsJSON, fromX, fromY, toX, toY)
}

func (t *Trainer) makeTargetWithCoords(action string, argsJSON string, fromX, fromY, toX, toY float64) []float64 {
	target := make([]float64, t.outputDim)
	// tool one-hot (amplified so MSE is not drowned by zero coord/arg dims)
	for i, tool := range t.tools {
		if i >= t.toolStart {
			break
		}
		if tool == action {
			target[i] = 2.0
			break
		}
	}
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
	Action   string
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
// Accepts production shapes:
//   - normalized object: {"x":700,"y":400}
//   - nested string args: {"tool":"click","args":"{\"X\":700,\"Y\":400}"}
//   - nested object args: {"tool":"click","args":{"X":700,"Y":400}}
//
// Returns (fromX, fromY, toX, toY). For single-coord actions, from == to.
func decodeCoords(action string, argsJSON string) (float64, float64, float64, float64) {
	args := decodeArgsMap(argsJSON)
	if args == nil {
		return 0, 0, 0, 0
	}
	x := mapFloat(args, "x", "X")
	y := mapFloat(args, "y", "Y")
	fromX := mapFloat(args, "from_x", "fromX", "FromX")
	fromY := mapFloat(args, "from_y", "fromY", "FromY")
	toX := mapFloat(args, "to_x", "toX", "ToX")
	toY := mapFloat(args, "to_y", "toY", "ToY")

	switch strings.ToLower(action) {
	case "drag", "drag_and_drop":
		if toX == 0 && toY == 0 {
			toX, toY = x, y
		}
		return fromX, fromY, toX, toY
	default:
		if x == 0 && y == 0 {
			x, y = toX, toY
		}
		if fromX == 0 && fromY == 0 {
			fromX, fromY = x, y
		}
		return fromX, fromY, x, y
	}
}

func decodeArgsMap(argsJSON string) map[string]any {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" || argsJSON[0] != '{' {
		return nil
	}
	var top map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &top); err != nil {
		return nil
	}
	if raw, ok := top["args"]; ok {
		switch a := raw.(type) {
		case string:
			var inner map[string]any
			if err := json.Unmarshal([]byte(a), &inner); err == nil {
				return inner
			}
		case map[string]any:
			return a
		}
	}
	return top
}

func mapFloat(m map[string]any, keys ...string) float64 {
	for _, want := range keys {
		for k, v := range m {
			if !strings.EqualFold(k, want) {
				continue
			}
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			case json.Number:
				f, err := n.Float64()
				if err == nil {
					return f
				}
			}
		}
	}
	return 0
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

// Evaluate returns mean MSE loss over the given samples without updating weights.
func (t *Trainer) Evaluate(samples []dataloader.Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for i := range samples {
		tokens, coords, targets := t.prepareBatch(samples, i)
		logits, err := t.model.Forward(tokens, coords, nil)
		if err != nil {
			continue
		}
		sum += mseLoss(logits[0], targets[0])
	}
	return sum / float64(len(samples))
}

// Accuracy returns tool-classification accuracy (argmax over tool logits only).
// Previously this used toolStart (tools+coords+args), which mixed continuous
// coord dims into the argmax and inflated scores.
func (t *Trainer) Accuracy(samples []dataloader.Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	numTools := len(t.tools)
	if numTools == 0 {
		return 0
	}
	correct := 0
	evaluated := 0
	for i := range samples {
		s := samples[i]
		// skip rows whose tool is not in the label set
		gold := -1
		for j, tool := range t.tools {
			if tool == s.Action {
				gold = j
				break
			}
		}
		if gold < 0 {
			continue
		}
		tokens, coords, targets := t.prepareBatch(samples, i)
		logits, err := t.model.Forward(tokens, coords, nil)
		if err != nil {
			continue
		}
		if len(logits) == 0 || len(logits[0]) < numTools || len(targets[0]) < numTools {
			continue
		}
		evaluated++
		if argmax(logits[0][:numTools]) == gold {
			correct++
		}
	}
	if evaluated == 0 {
		return 0
	}
	return float64(correct) / float64(evaluated)
}

func argmax(v []float64) int {
	best := 0
	for i := 1; i < len(v); i++ {
		if v[i] > v[best] {
			best = i
		}
	}
	return best
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
