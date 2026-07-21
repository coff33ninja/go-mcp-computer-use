package dataloader

import (
	"math/rand"
	"strings"
)

// Augmentor generates synthetic training samples from existing ones.
type Augmentor struct {
	dpiScales    []float64 // DPI scale factors to apply (e.g., 0.75, 1.25, 1.5)
	ocrNoise    bool       // add OCR-like typos
	coordJitter float64    // pixel jitter range for coordinates
}

// NewAugmentor creates an augmentor with sensible defaults.
func NewAugmentor() *Augmentor {
	return &Augmentor{
		dpiScales:    []float64{0.75, 1.25, 1.5},
		ocrNoise:    true,
		coordJitter: 5.0,
	}
}

// Augment generates synthetic variants of the given sample.
// Returns up to N augmented samples (original is NOT included).
func (a *Augmentor) Augment(s Sample, n int) []Sample {
	if n <= 0 {
		return nil
	}
	var out []Sample

	// 1. DPI scale variations
	for _, scale := range a.dpiScales {
		aug := a.scaleSample(s, scale)
		out = append(out, aug)
		if len(out) >= n {
			return out[:n]
		}
	}

	// 2. OCR noise variations
	if a.ocrNoise {
		for i := 0; i < 2 && len(out) < n; i++ {
			aug := a.ocrNoiseSample(s)
			out = append(out, aug)
		}
	}

	// 3. Coordinate jitter
	for i := 0; i < 2 && len(out) < n; i++ {
		aug := a.jitterCoords(s)
		out = append(out, aug)
	}

	if len(out) > n {
		out = out[:n]
	}
	return out
}

// scaleSample scales coordinates by a DPI factor.
func (a *Augmentor) scaleSample(s Sample, scale float64) Sample {
	aug := s
	aug.Context = s.Context // context text stays the same
	if aug.CoordX != 0 || aug.CoordY != 0 {
		aug.CoordX = int(float64(s.CoordX) * scale)
		aug.CoordY = int(float64(s.CoordY) * scale)
	}
	return aug
}

// ocrNoiseSample introduces common OCR errors.
func (a *Augmentor) ocrNoiseSample(s Sample) Sample {
	aug := s
	text := s.Context
	if len(text) == 0 {
		return aug
	}

	// common OCR substitutions
	substitutions := []struct {
		from, to string
	}{
		{"0", "O"},
		{"O", "0"},
		{"1", "l"},
		{"l", "1"},
		{"I", "l"},
		{"5", "S"},
		{"S", "5"},
		{"8", "B"},
		{"rn", "m"},
		{"m", "rn"},
	}

	// pick 1-2 random substitutions
	n := 1 + rand.Intn(2)
	for i := 0; i < n; i++ {
		sub := substitutions[rand.Intn(len(substitutions))]
		if strings.Contains(text, sub.from) {
			idx := strings.Index(text, sub.from)
			text = text[:idx] + sub.to + text[idx+len(sub.from):]
			break // only one substitution per sample
		}
	}

	aug.Context = text
	return aug
}

// jitterCoords adds random pixel offset to coordinates.
func (a *Augmentor) jitterCoords(s Sample) Sample {
	aug := s
	if aug.CoordX != 0 || aug.CoordY != 0 {
		jitter := int(a.coordJitter)
		if jitter > 0 {
			aug.CoordX += rand.Intn(jitter*2+1) - jitter
			aug.CoordY += rand.Intn(jitter*2+1) - jitter
		}
	}
	return aug
}

// AugmentAll augments a slice of samples, generating up to maxAugment per sample.
func (a *Augmentor) AugmentAll(samples []Sample, maxAugment int) []Sample {
	if maxAugment <= 0 {
		return samples
	}
	result := make([]Sample, 0, len(samples)*(maxAugment+1))
	result = append(result, samples...) // keep originals
	for _, s := range samples {
		augmented := a.Augment(s, maxAugment)
		result = append(result, augmented...)
	}
	return result
}
