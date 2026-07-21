package compress

import (
	"math"

	"github.com/coff33ninja/go-mcp-computer-use/ml/transformer"
)

// QuantizeConfig controls weight quantization.
type QuantizeConfig struct {
	Bits int // number of bits per weight (2-8), default 8
}

// Quantize buckets each weight into 2^bits evenly-spaced levels.
// Reduces precision but keeps all weights (no sparsity change).
func Quantize(model transformer.Model, cfg QuantizeConfig) error {
	if cfg.Bits < 2 || cfg.Bits > 8 {
		cfg.Bits = 8
	}

	params := model.Parameters()
	if len(params) == 0 {
		return nil
	}

	// find range
	minVal, maxVal := params[0], params[0]
	for _, p := range params[1:] {
		if p < minVal {
			minVal = p
		}
		if p > maxVal {
			maxVal = p
		}
	}

	nLevels := 1 << cfg.Bits
	levels := float64(nLevels - 1)
	if levels == 0 {
		return nil
	}
	scale := (maxVal - minVal) / levels
	if scale == 0 {
		return nil // all weights equal
	}

	// quantize each weight to nearest level
	for i, p := range params {
		level := math.Round((p-minVal)/scale)
		params[i] = minVal + level*scale
	}

	return model.LoadParameters(params)
}

// QuantizedSize returns the compressed size in bytes for the given parameter count and bit depth.
func QuantizedSize(paramCount int, bits int) int {
	return (paramCount * bits) / 8
}

// OriginalSize returns the uncompressed size in bytes (float64 = 8 bytes each).
func OriginalSize(paramCount int) int {
	return paramCount * 8
}

// CompressionRatio returns the ratio of original to quantized size.
func CompressionRatio(paramCount int, bits int) float64 {
	return float64(OriginalSize(paramCount)) / float64(QuantizedSize(paramCount, bits))
}
