package compress

import (
	"math"
	"sort"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// PruneConfig controls magnitude-based weight pruning.
type PruneConfig struct {
	Sparsity float64 // target fraction of weights to zero (0.0-0.99)
}

// Prune zeros out the smallest-magnitude weights until the target sparsity is reached.
// Returns the model with pruned weights and the actual sparsity achieved.
func Prune(model transformer.Model, cfg PruneConfig) (float64, error) {
	if cfg.Sparsity <= 0 || cfg.Sparsity >= 1 {
		return 0, nil
	}

	params := model.Parameters()
	if len(params) == 0 {
		return 0, nil
	}

	// compute absolute values and sort
	absParams := make([]float64, len(params))
	for i, p := range params {
		absParams[i] = math.Abs(p)
	}
	sort.Float64s(absParams)

	// find threshold: the Nth percentile where N = sparsity
	idx := int(float64(len(absParams)) * cfg.Sparsity)
	if idx >= len(absParams) {
		idx = len(absParams) - 1
	}
	threshold := absParams[idx]

	// zero out weights below threshold
	pruned := 0
	for i, p := range params {
		if math.Abs(p) <= threshold {
			params[i] = 0
			pruned++
		}
	}

	actualSparsity := float64(pruned) / float64(len(params))
	if err := model.LoadParameters(params); err != nil {
		return 0, err
	}

	return actualSparsity, nil
}

// IterativePrune prunes and retrains in cycles for better accuracy at high sparsity.
// Returns final sparsity and loss.
func IterativePrune(model transformer.Model, cfg PruneConfig, retrainFn func(model transformer.Model) float64, rounds int) (float64, float64) {
	var lastLoss float64
	sparsity := 0.0
	for i := 0; i < rounds; i++ {
		// increase sparsity each round
		roundSparsity := cfg.Sparsity * float64(i+1) / float64(rounds)
		s, err := Prune(model, PruneConfig{Sparsity: roundSparsity})
		if err != nil {
			break
		}
		sparsity = s
		if retrainFn != nil {
			lastLoss = retrainFn(model)
		}
	}
	return sparsity, lastLoss
}
