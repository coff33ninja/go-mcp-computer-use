package actions

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
	"bytes"
)

type MatchResult struct {
	X      int32   `json:"x"`
	Y      int32   `json:"y"`
	Width  int32   `json:"width"`
	Height int32   `json:"height"`
	Score  float64 `json:"score"`
}

func FindImageOnScreen(templateB64 string, threshold float64) (*MatchResult, error) {
	screenB64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("find_image screenshot: %w", err)
	}
	return FindImage(screenB64, templateB64, threshold)
}

func FindImageInRegion(screenB64, templateB64 string, threshold float64) (*MatchResult, error) {
	return FindImage(screenB64, templateB64, threshold)
}

func FindImage(screenB64, templateB64 string, threshold float64) (*MatchResult, error) {
	if threshold <= 0 {
		threshold = 0.7
	}

	screen, err := decodePNGB64(screenB64)
	if err != nil {
		return nil, fmt.Errorf("find_image decode screen: %w", err)
	}
	tmpl, err := decodePNGB64(templateB64)
	if err != nil {
		return nil, fmt.Errorf("find_image decode template: %w", err)
	}

	sBounds := screen.Bounds()
	tBounds := tmpl.Bounds()
	tw := tBounds.Dx()
	th := tBounds.Dy()

	if tw > sBounds.Dx() || th > sBounds.Dy() {
		return nil, fmt.Errorf("template (%dx%d) larger than screen (%dx%d)", tw, th, sBounds.Dx(), sBounds.Dy())
	}

	sGray := toGray(screen)
	tGray := toGray(tmpl)

	tMean, tStd := meanStd(tGray)
	if tStd == 0 {
		return nil, fmt.Errorf("template has no variance")
	}

	bestX, bestY := 0, 0
	bestScore := math.Inf(-1)

	maxX := sBounds.Dx() - tw
	maxY := sBounds.Dy() - th

	for y := 0; y <= maxY; y++ {
		for x := 0; x <= maxX; x++ {
			var sSum, sSumSq, stSum float64
			for ty := 0; ty < th; ty++ {
				for tx := 0; tx < tw; tx++ {
					sv := float64(sGray[(y+ty)*sBounds.Dx()+(x+tx)])
					tv := float64(tGray[ty*tw+tx])
					sSum += sv
					sSumSq += sv * sv
					stSum += sv * tv
				}
			}
			n := float64(tw * th)
			sMean := sSum / n
			sStd := math.Sqrt(sSumSq/n - sMean*sMean)
			if sStd == 0 {
				continue
			}
			cov := stSum - n*sMean*tMean
			corr := cov / (n * sStd * tStd)
			if corr > bestScore {
				bestScore = corr
				bestX = x
				bestY = y
			}
		}
	}

	if bestScore < threshold {
		return nil, fmt.Errorf("no match found (best score %.3f below threshold %.2f)", bestScore, threshold)
	}

	return &MatchResult{
		X:      int32(bestX),
		Y:      int32(bestY),
		Width:  int32(tw),
		Height: int32(th),
		Score:  bestScore,
	}, nil
}

func decodePNGB64(b64 string) (image.Image, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func toGray(img image.Image) []uint8 {
	b := img.Bounds()
	gray := make([]uint8, b.Dx()*b.Dy())
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bv, _ := img.At(x, y).RGBA()
			gray[i] = uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bv>>8))
			i++
		}
	}
	return gray
}

func meanStd(data []uint8) (mean, std float64) {
	n := float64(len(data))
	if n == 0 {
		return 0, 0
	}
	var sum, sumSq float64
	for _, v := range data {
		f := float64(v)
		sum += f
		sumSq += f * f
	}
	mean = sum / n
	std = math.Sqrt(sumSq/n - mean*mean)
	return
}
