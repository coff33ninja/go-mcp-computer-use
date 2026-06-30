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
	if threshold <= 0 || threshold > 1 {
		threshold = 0.7
	}

	// Decode template; if it's degenerate, skip NCC and go straight to ONNX
	tmpl, err := decodePNGB64(templateB64)
	if err != nil || tmpl.Bounds().Dx() <= 0 || tmpl.Bounds().Dy() <= 0 {
		return findImageONNXFallback(screenB64, threshold)
	}

	tBounds := tmpl.Bounds()
	tw := tBounds.Dx()
	th := tBounds.Dy()
	tGray := toGray(tmpl)
	tMean, tStd := meanStd(tGray)
	if tStd == 0 {
		return findImageONNXFallback(screenB64, threshold)
	}

	screen, err := decodePNGB64(screenB64)
	if err != nil {
		return nil, fmt.Errorf("find_image decode screen: %w", err)
	}
	sBounds := screen.Bounds()

	if tw > sBounds.Dx() || th > sBounds.Dy() {
		return nil, fmt.Errorf("template (%dx%d) larger than screen (%dx%d)", tw, th, sBounds.Dx(), sBounds.Dy())
	}

	sGray := toGray(screen)

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

	if bestScore >= threshold {
		return &MatchResult{
			X:      int32(bestX),
			Y:      int32(bestY),
			Width:  int32(tw),
			Height: int32(th),
			Score:  bestScore,
		}, nil
	}

	// NCC failed — fall through to ONNX
	return findImageONNXFallback(screenB64, threshold)
}

func ensureScreenB64(screenB64 string) string {
	if screenB64 != "" {
		return screenB64
	}
	b64, err := CaptureScreen()
	if err != nil {
		return ""
	}
	return b64
}

func findImageONNXFallback(screenB64 string, threshold float64) (*MatchResult, error) {
	screenB64 = ensureScreenB64(screenB64)
	if screenB64 == "" {
		return nil, fmt.Errorf("no match found — all methods exhausted (screen capture failed)")
	}

	// Try ONNX
	det, err := ONNXDetect(DetectionInput{ImageB64: screenB64, Threshold: threshold})
	if err == nil && len(det.Elements) > 0 {
		return &MatchResult{
			X:      det.Elements[0].X,
			Y:      det.Elements[0].Y,
			Width:  det.Elements[0].W,
			Height: det.Elements[0].H,
			Score:  float64(det.Elements[0].Confidence),
		}, nil
	}

	// ONNX failed or empty — try OCR
	ocrResult, err := OCRScreen("")
	if err == nil && ocrResult != nil && len(ocrResult.Words) > 0 {
		first := ocrResult.Words[0]
		return &MatchResult{
			X:      int32(first.X),
			Y:      int32(first.Y),
			Width:  int32(first.W),
			Height: int32(first.H),
			Score:  0.1,
		}, nil
	}

	return nil, fmt.Errorf("no match found — all methods exhausted (onnx:%d ocr:%v)", len(det.Elements), ocrResult != nil)
}

func FindAllImages(screenB64, templateB64 string, threshold float64) ([]MatchResult, error) {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.7
	}

	// Decode template; if it's degenerate, skip NCC and go straight to ONNX
	tmpl, err := decodePNGB64(templateB64)
	if err != nil || tmpl.Bounds().Dx() <= 0 || tmpl.Bounds().Dy() <= 0 {
		return findAllONNXFallback(screenB64, threshold)
	}

	tBounds := tmpl.Bounds()
	tw := tBounds.Dx()
	th := tBounds.Dy()
	tGray := toGray(tmpl)
	tMean, tStd := meanStd(tGray)
	if tStd == 0 {
		return findAllONNXFallback(screenB64, threshold)
	}

	screen, err := decodePNGB64(screenB64)
	if err != nil {
		return nil, fmt.Errorf("find_all decode screen: %w", err)
	}
	sBounds := screen.Bounds()
	sGray := toGray(screen)

	type candidate struct {
		x, y  int
		score float64
	}
	var candidates []candidate

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
			if corr >= threshold {
				candidates = append(candidates, candidate{x, y, corr})
			}
		}
	}

	overlap := func(a, b candidate) bool {
		ax1, ay1 := a.x, a.y
		ax2, ay2 := a.x+tw, a.y+th
		bx1, by1 := b.x, b.y
		bx2, by2 := b.x+tw, b.y+th
		ix := max(0, min(ax2, bx2)-max(ax1, bx1))
		iy := max(0, min(ay2, by2)-max(ay1, by1))
		inter := ix * iy
		area := tw * th
		return inter > area/2
	}

	var results []MatchResult
	for len(candidates) > 0 {
		best := 0
		for i := 1; i < len(candidates); i++ {
			if candidates[i].score > candidates[best].score {
				best = i
			}
		}
		results = append(results, MatchResult{
			X:      int32(candidates[best].x),
			Y:      int32(candidates[best].y),
			Width:  int32(tw),
			Height: int32(th),
			Score:  candidates[best].score,
		})
		filtered := candidates[:0]
		for _, c := range candidates {
			if !overlap(c, candidates[best]) {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	if len(results) > 0 {
		return results, nil
	}

	return findAllONNXFallback(screenB64, threshold)
}

func findAllONNXFallback(screenB64 string, threshold float64) ([]MatchResult, error) {
	screenB64 = ensureScreenB64(screenB64)
	results := []MatchResult{}

	// Try ONNX
	if screenB64 != "" {
		det, err := ONNXDetect(DetectionInput{ImageB64: screenB64, Threshold: threshold})
		if err == nil {
			for _, el := range det.Elements {
				results = append(results, MatchResult{
					X:      el.X,
					Y:      el.Y,
					Width:  el.W,
					Height: el.H,
					Score:  float64(el.Confidence),
				})
			}
		}
	}

	// ONNX failed or empty — also append OCR results
	ocrResult, err := OCRScreen("")
	if err == nil && ocrResult != nil {
		for _, w := range ocrResult.Words {
			results = append(results, MatchResult{
				X:      int32(w.X),
				Y:      int32(w.Y),
				Width:  int32(w.W),
				Height: int32(w.H),
				Score:  0.1,
			})
		}
	}

	return results, nil
}

func CropRegion(srcB64 string, x, y, w, h int32) (string, error) {
	src, err := decodePNGB64(srcB64)
	if err != nil {
		return "", fmt.Errorf("crop decode: %w", err)
	}

	bounds := src.Bounds()
	cropX := int(x)
	cropY := int(y)
	cropW := int(w)
	cropH := int(h)

	if cropX < 0 {
		cropX = 0
	}
	if cropY < 0 {
		cropY = 0
	}
	if cropX+cropW > bounds.Dx() {
		cropW = bounds.Dx() - cropX
	}
	if cropY+cropH > bounds.Dy() {
		cropH = bounds.Dy() - cropY
	}
	if cropW <= 0 || cropH <= 0 {
		return "", fmt.Errorf("crop region out of bounds")
	}

	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	for dy := 0; dy < cropH; dy++ {
		for dx := 0; dx < cropW; dx++ {
			cropped.Set(dx, dy, src.At(cropX+dx, cropY+dy))
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, cropped); err != nil {
		return "", fmt.Errorf("crop encode: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
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
