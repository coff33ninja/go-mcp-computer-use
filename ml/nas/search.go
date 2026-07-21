package nas

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/coff33ninja/go-mcp-computer-use/ml/dataloader"
	"github.com/coff33ninja/go-mcp-computer-use/ml/spatial"
	"github.com/coff33ninja/go-mcp-computer-use/ml/tokenizer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/trainer"
	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// SearchSpace defines the architecture dimensions to explore.
type SearchSpace struct {
	EmbedDims  []int
	NumHeads   []int
	NumLayers  []int
	FFNDims    []int
}

// DefaultSearchSpace returns a reasonable search space for UI automation.
func DefaultSearchSpace() SearchSpace {
	return SearchSpace{
		EmbedDims: []int{32, 48, 64, 96},
		NumHeads:  []int{2, 3, 4},
		NumLayers: []int{2, 3},
		FFNDims:   []int{64, 128, 192, 256},
	}
}

// SearchResult holds one architecture evaluation.
type SearchResult struct {
	Config    transformer.Config
	ParamCount int
	Loss      float64
	EPOCHS    int
}

// Search runs a grid search over the search space and returns results sorted by loss.
func Search(
	dbPath string,
	tools []string,
	screenCfg spatial.ScreenConfig,
	epochs int,
	maxConfigs int,
) ([]SearchResult, error) {
	loader := dataloader.NewSQLiteLoader(dbPath)
	defer loader.Close()

	ctx := context.Background()
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("nas: load data: %w", err)
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("nas: no training data")
	}

	ss := DefaultSearchSpace()
	var results []SearchResult
	evaluated := 0

	for _, embedDim := range ss.EmbedDims {
		for _, numHeads := range ss.NumHeads {
			for _, numLayers := range ss.NumLayers {
				for _, ffDim := range ss.FFNDims {
					// skip invalid combos (ffn must be >= embed_dim)
					if ffDim < embedDim {
						continue
					}
					// numHeads must divide embedDim
					if embedDim%numHeads != 0 {
						continue
					}

					cfg := transformer.Config{
						VocabSize:  2000,
						MaxLen:     128,
						EmbedDim:   embedDim,
						NumHeads:   numHeads,
						NumLayers:  numLayers,
						FFNDim:     ffDim,
						CoordDim:   spatial.FeatureDim,
						OutputDim:  len(tools) + 2 + 10,
						ArgDim:     10,
						HistoryLen: 5,
					}

					loss := evalConfig(cfg, tools, samples, loader, screenCfg, epochs)
					pc := paramCount(cfg)

					results = append(results, SearchResult{
						Config:     cfg,
						ParamCount: pc,
						Loss:       loss,
						EPOCHS:     epochs,
					})
					evaluated++

					if maxConfigs > 0 && evaluated >= maxConfigs {
						break
					}
				}
				if maxConfigs > 0 && evaluated >= maxConfigs {
					break
				}
			}
			if maxConfigs > 0 && evaluated >= maxConfigs {
				break
			}
		}
		if maxConfigs > 0 && evaluated >= maxConfigs {
			break
		}
	}

	// sort by loss ascending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Loss < results[i].Loss {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	slog.Info("nas: search complete",
		"evaluated", evaluated,
		"best_loss", fmt.Sprintf("%.6f", results[0].Loss),
		"best_params", results[0].ParamCount,
	)

	return results, nil
}

// evalConfig trains a fresh model with the given config and returns final loss.
func evalConfig(
	cfg transformer.Config,
	tools []string,
	samples []dataloader.Sample,
	loader *dataloader.SQLiteLoader,
	screenCfg spatial.ScreenConfig,
	epochs int,
) float64 {
	mdl, err := transformer.New(cfg)
	if err != nil {
		return math.MaxFloat64
	}

	tok := tokenizer.NewSimpleTokenizer()
	tok.Fit([]string{"click button Submit hover Cancel type text scroll up down"})

	enc := spatial.NewEncoder(screenCfg)

	tr := trainer.NewTrainer(trainer.TrainerConfig{
		Model:        mdl,
		ModelConfig:  cfg,
		Tokenizer:    tok,
		Encoder:      enc,
		Tools:        tools,
		LearningRate: 0.005,
	})

	var lastLoss float64
	for i := 0; i < epochs; i++ {
		result, err := tr.TrainEpoch(loader)
		if err != nil {
			return math.MaxFloat64
		}
		lastLoss = result.FinalLoss
		if result.SamplesProcessed == 0 {
			return math.MaxFloat64
		}
	}
	return lastLoss
}

// paramCount returns the approximate number of trainable parameters.
func paramCount(cfg transformer.Config) int {
	d := cfg.EmbedDim
	// embedding: VocabSize * EmbedDim + pos encoding (MaxLen * EmbedDim)
	emb := cfg.VocabSize*d + cfg.MaxLen*d
	// per layer: QKV + output projections + FFN
	perLayer := 3*d*d + d + d*cfg.FFNDim + cfg.FFNDim + cfg.FFNDim*d + d
	layers := perLayer * cfg.NumLayers
	// output head
	out := d*cfg.OutputDim + cfg.OutputDim
	return emb + layers + out
}

// BestConfig returns the best config from search results under a parameter budget.
func BestConfig(results []SearchResult, maxParams int) *SearchResult {
	for _, r := range results {
		if maxParams <= 0 || r.ParamCount <= maxParams {
			return &r
		}
	}
	if len(results) > 0 {
		return &results[len(results)-1]
	}
	return nil
}
