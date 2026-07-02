package actions

import (
	"fmt"
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
	BeforeOCR                    *OCRResult
	AfterWaitMs                  int
	ExpectedText, NotText, Lang string
	RegionX, RegionY             *int32
	RegionW, RegionH             *int32
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
	LinesAdded      []string `json:"lines_added,omitempty"`
	LinesRemoved    []string `json:"lines_removed,omitempty"`
	LinesChanged    int      `json:"lines_changed"`
	TotalChangeRatio float64 `json:"total_change_ratio"`
}

func VerifyAction(cfg *VerifyConfig) *VerifyResult {
	r := &VerifyResult{}

	oc, err := captureOCR(cfg.RegionX, cfg.RegionY, cfg.RegionW, cfg.RegionH, cfg.Lang)
	if err != nil {
		r.Reason = fmt.Sprintf("before OCR failed: %v", err)
		r.Method = "error"
		return r
	}
	if cfg.BeforeOCR != nil {
		r.BeforeOCR = cfg.BeforeOCR
	} else {
		r.BeforeOCR = oc
	}

	waitMs := cfg.AfterWaitMs
	if waitMs <= 0 {
		waitMs = 500
	}
	time.Sleep(time.Duration(waitMs) * time.Millisecond)

	after, err := captureOCR(cfg.RegionX, cfg.RegionY, cfg.RegionW, cfg.RegionH, cfg.Lang)
	if err != nil {
		r.Reason = fmt.Sprintf("after OCR failed: %v", err)
		r.Method = "error"
		return r
	}
	r.AfterOCR = after
	r.Diff = computeTextDiff(r.BeforeOCR.Text, after.Text)

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
	sw, sh := ScreenSize()
	rx := x - half
	ry := y - half
	if rx < 0 {
		rx = 0
	}
	if ry < 0 {
		ry = 0
	}
	rw := size
	rh := size
	if rx+rw > sw {
		rw = sw - rx
	}
	if ry+rh > sh {
		rh = sh - ry
	}
	return rx, ry, rw, rh
}
