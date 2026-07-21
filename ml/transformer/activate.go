package transformer

import "math"

// Softmax applies the softmax function to a slice of logits, returning probabilities.
func Softmax(logits []float64) []float64 {
	if len(logits) == 0 {
		return logits
	}
	max := logits[0]
	for _, v := range logits[1:] {
		if v > max {
			max = v
		}
	}
	out := make([]float64, len(logits))
	var sum float64
	for i, v := range logits {
		out[i] = math.Exp(v - max)
		sum += out[i]
	}
	if sum > 0 {
		for i := range out {
			out[i] /= sum
		}
	}
	return out
}

// Sigmoid applies the sigmoid function to a value.
func Sigmoid(x float64) float64 {
	if x >= 0 {
		return 1.0 / (1.0 + math.Exp(-x))
	}
	ez := math.Exp(x)
	return ez / (1.0 + ez)
}

// SigmoidSlice applies sigmoid to each element.
func SigmoidSlice(xs []float64) []float64 {
	out := make([]float64, len(xs))
	for i, v := range xs {
		out[i] = Sigmoid(v)
	}
	return out
}
