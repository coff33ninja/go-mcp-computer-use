package actions

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"
)

type ExpConfig struct {
	Text    string `json:"text,omitempty"`
	Change  *bool  `json:"change,omitempty"`
	NotText string `json:"not_text,omitempty"`
	WaitMs  int    `json:"wait_ms,omitempty"`
}

type VerifyConfig struct {
	BeforeOCR                   *OCRResult
	AfterWaitMs                 int
	ExpectedText, NotText, Lang string
	RegionX, RegionY            *int32
	RegionW, RegionH            *int32
}

type VerifyResult struct {
	Passed      bool       `json:"passed"`
	Method      string     `json:"method"`
	Reason      string     `json:"reason"`
	BeforeOCR   *OCRResult `json:"before_ocr"`
	AfterOCR    *OCRResult `json:"after_ocr"`
	MatchedText string     `json:"matched_text,omitempty"`
	Position    *OCRWord   `json:"position,omitempty"`
	Diff        *TextDiff  `json:"diff,omitempty"`
}

type TextDiff struct {
	LinesAdded       []string `json:"lines_added,omitempty"`
	LinesRemoved     []string `json:"lines_removed,omitempty"`
	LinesChanged     int      `json:"lines_changed"`
	TotalChangeRatio float64  `json:"total_change_ratio"`
}

func VerifyAction(cfg *VerifyConfig) *VerifyResult {
	before := cfg.BeforeOCR
	if before == nil {
		var err error
		before, err = captureOCR(cfg.RegionX, cfg.RegionY, cfg.RegionW, cfg.RegionH, cfg.Lang)
		if err != nil {
			return &VerifyResult{Method: "error", Reason: fmt.Sprintf("before OCR failed: %v", err)}
		}
	}

	waitMs := cfg.AfterWaitMs
	if waitMs <= 0 {
		waitMs = 500
	}
	time.Sleep(time.Duration(waitMs) * time.Millisecond)

	after, err := captureOCR(cfg.RegionX, cfg.RegionY, cfg.RegionW, cfg.RegionH, cfg.Lang)
	if err != nil {
		return &VerifyResult{BeforeOCR: before, Method: "error", Reason: fmt.Sprintf("after OCR failed: %v", err)}
	}
	return verifyOCRResults(before, after, cfg)
}

func verifyOCRResults(before, after *OCRResult, cfg *VerifyConfig) *VerifyResult {
	r := &VerifyResult{BeforeOCR: before, AfterOCR: after}
	r.Diff = computeTextDiff(before.Text, after.Text)

	if cfg.ExpectedText != "" {
		r.Method = "expected_text"
		r.MatchedText = cfg.ExpectedText
		lower := strings.ToLower(cfg.ExpectedText)
		for _, w := range after.Words {
			if strings.Contains(strings.ToLower(w.Text), lower) {
				r.Passed = true
				r.Position = &w
				r.Reason = fmt.Sprintf("expected text %q appeared", cfg.ExpectedText)
				return r
			}
		}
		for _, ln := range after.Lines {
			if strings.Contains(strings.ToLower(ln.Text), lower) {
				r.Passed = true
				r.Reason = fmt.Sprintf("expected text %q appeared in line", cfg.ExpectedText)
				return r
			}
		}
		r.Reason = fmt.Sprintf("expected text %q not found", cfg.ExpectedText)
		return r
	}

	if cfg.NotText != "" {
		r.Method = "not_text"
		lower := strings.ToLower(cfg.NotText)
		for _, w := range after.Words {
			if strings.Contains(strings.ToLower(w.Text), lower) {
				r.Reason = fmt.Sprintf("text %q still present", cfg.NotText)
				return r
			}
		}
		for _, ln := range after.Lines {
			if strings.Contains(strings.ToLower(ln.Text), lower) {
				r.Reason = fmt.Sprintf("text %q still present in line", cfg.NotText)
				return r
			}
		}
		r.Passed = true
		r.Reason = fmt.Sprintf("text %q disappeared", cfg.NotText)
		return r
	}

	r.Method = "ocr_diff"
	if r.BeforeOCR.Text != after.Text {
		r.Passed = true
		r.Reason = "screen content changed"
	} else {
		r.Reason = "no visible change"
	}
	return r
}

func captureOCR(rx, ry, rw, rh *int32, lang string) (*OCRResult, error) {
	if rw != nil && rh != nil {
		x := int32(0)
		y := int32(0)
		if rx != nil {
			x = *rx
		}
		if ry != nil {
			y = *ry
		}
		return OCRRegion(x, y, *rw, *rh, lang)
	}
	return OCRScreen(lang)
}

func computeTextDiff(before, after string) *TextDiff {
	if before == after {
		return &TextDiff{}
	}
	bLines := strings.Split(before, "\n")
	aLines := strings.Split(after, "\n")

	bSet := make(map[string]int)
	for _, l := range bLines {
		t := strings.TrimSpace(l)
		if t != "" {
			bSet[t]++
		}
	}
	aSet := make(map[string]int)
	for _, l := range aLines {
		t := strings.TrimSpace(l)
		if t != "" {
			aSet[t]++
		}
	}

	d := &TextDiff{}
	for _, l := range aLines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if bSet[t] <= 0 {
			d.LinesAdded = append(d.LinesAdded, t)
		} else {
			bSet[t]--
		}
	}
	for _, l := range bLines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if aSet[t] <= 0 {
			d.LinesRemoved = append(d.LinesRemoved, t)
		} else {
			aSet[t]--
		}
	}

	total := len(bLines) + len(aLines)
	changed := len(d.LinesAdded) + len(d.LinesRemoved)
	if total > 0 {
		d.TotalChangeRatio = float64(changed) / float64(total)
	}
	d.LinesChanged = changed / 2
	return d
}

func SmartRegionAround(x, y, size int32) (int32, int32, int32, int32) {
	if size <= 0 {
		size = 400
	}
	half := size / 2
	bounds := VirtualScreenBounds()
	rx := x - half
	ry := y - half
	if rx < bounds.X {
		rx = bounds.X
	}
	if ry < bounds.Y {
		ry = bounds.Y
	}
	rw := size
	rh := size
	if rx+rw > bounds.X+bounds.W {
		rw = bounds.X + bounds.W - rx
	}
	if ry+rh > bounds.Y+bounds.H {
		rh = bounds.Y + bounds.H - ry
	}
	return rx, ry, rw, rh
}

// ── Image diff (pixel-level screenshot comparison) ──

type ImageDiffResult struct {
	ChangedPixels int     `json:"changed_pixels"`
	TotalPixels   int     `json:"total_pixels"`
	ChangeRatio   float64 `json:"change_ratio"`
	MeanDiff      float64 `json:"mean_diff"`
	MaxDiff       int     `json:"max_diff"`
	Same          bool    `json:"same"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	DiffImageB64  string  `json:"diff_image,omitempty"`
}

type ImageDiffOpts struct {
	Threshold     int
	GenerateImage bool
}

func ImageDiff(beforeB64, afterB64 string, opts ImageDiffOpts) (*ImageDiffResult, error) {
	beforeImg, err := decodePNGB64(beforeB64)
	if err != nil {
		return nil, fmt.Errorf("image_diff: failed to decode 'before' image: %w", err)
	}
	afterImg, err := decodePNGB64(afterB64)
	if err != nil {
		return nil, fmt.Errorf("image_diff: failed to decode 'after' image: %w", err)
	}

	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 30
	}

	bounds := beforeImg.Bounds()
	aBounds := afterImg.Bounds()
	w := minInt(bounds.Dx(), aBounds.Dx())
	h := minInt(bounds.Dy(), aBounds.Dy())
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("image_diff: images have no overlapping region (%dx%d vs %dx%d)",
			bounds.Dx(), bounds.Dy(), aBounds.Dx(), aBounds.Dy())
	}

	var diffImg *image.RGBA
	if opts.GenerateImage {
		diffImg = image.NewRGBA(image.Rect(0, 0, w, h))
	}

	var totalDiff float64
	var maxDiff int
	changedPixels := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := beforeImg.At(x, y).RGBA()
			r2, g2, b2, _ := afterImg.At(x, y).RGBA()

			dr := absInt(int(r1>>8) - int(r2>>8))
			dg := absInt(int(g1>>8) - int(g2>>8))
			db := absInt(int(b1>>8) - int(b2>>8))
			dMax := maxInt(dr, maxInt(dg, db))
			dAvg := (dr + dg + db) / 3

			totalDiff += float64(dAvg)
			if dMax > maxDiff {
				maxDiff = dMax
			}

			isChanged := dMax >= threshold
			if isChanged {
				changedPixels++
			}

			if diffImg != nil {
				if isChanged {
					highlight := uint8(minInt(255, dMax*3))
					diffImg.Set(x, y, color.RGBA{R: highlight, G: 0, B: 0, A: 255})
				} else {
					diffImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
				}
			}
		}
	}

	totalPixels := w * h
	result := &ImageDiffResult{
		ChangedPixels: changedPixels,
		TotalPixels:   totalPixels,
		ChangeRatio:   float64(changedPixels) / float64(totalPixels),
		MeanDiff:      totalDiff / float64(totalPixels),
		MaxDiff:       maxDiff,
		Same:          changedPixels == 0,
		Width:         w,
		Height:        h,
	}

	if diffImg != nil {
		var buf bytes.Buffer
		if err := png.Encode(&buf, diffImg); err != nil {
			return nil, fmt.Errorf("image_diff: failed to encode diff image: %w", err)
		}
		result.DiffImageB64 = base64.StdEncoding.EncodeToString(buf.Bytes())
	}

	return result, nil
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
