package trainer

import (
	"context"
	"fmt"

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
}

type EpochResult struct {
	InitialLoss      float64
	FinalLoss        float64
	SamplesProcessed int
}

type Trainer struct {
	model     transformer.Model
	tokenizer tokenizer.Tokenizer
	encoder   *spatial.Encoder
	tools     []string
	lr        float64
	maxLen    int
	outputDim int
}

func NewTrainer(cfg TrainerConfig) *Trainer {
	return &Trainer{
		model:     cfg.Model,
		tokenizer: cfg.Tokenizer,
		encoder:   cfg.Encoder,
		tools:     cfg.Tools,
		lr:        cfg.LearningRate,
		maxLen:    cfg.ModelConfig.MaxLen,
		outputDim: cfg.ModelConfig.OutputDim,
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
	logits, err := t.model.Forward(tokens, coords)
	if err != nil {
		return nil, fmt.Errorf("trainer: initial forward: %w", err)
	}
	result.InitialLoss = mseLoss(logits[0], targets[0])

	// train on each sample
	for i, s := range samples {
		tokens, coords, targets := t.prepareBatch([]dataloader.Sample{s})
		_, err := t.model.Forward(tokens, coords)
		if err != nil {
			return nil, fmt.Errorf("trainer: forward %d: %w", i, err)
		}
		if err := t.model.BackwardWithTarget(targets[0], t.lr); err != nil {
			return nil, fmt.Errorf("trainer: backward %d: %w", i, err)
		}
	}

	// compute final loss
	tokens, coords, targets = t.prepareBatch(samples[:1])
	logits, err = t.model.Forward(tokens, coords)
	if err != nil {
		return nil, fmt.Errorf("trainer: final forward: %w", err)
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
		targets[i] = t.makeTarget(s.Action)
	}
	return tokens, coords, targets
}

func (t *Trainer) makeTarget(action string) []float64 {
	target := make([]float64, t.outputDim)
	for i, tool := range t.tools {
		if i >= t.outputDim {
			break
		}
		if tool == action {
			target[i] = 1.0
			break
		}
	}
	return target
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
