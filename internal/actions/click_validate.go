package actions

import (
	"sync"
	"time"
)

// ClickValidation is the advisory validation signal attached to every click
// result so the AI can confirm what element it actually clicked. It is
// best-effort: any of the sub-blocks may be absent if the corresponding
// signal could not be produced (model missing, capture failure, no priors,
// no ML memory). It never fails or modifies the click itself.
type ClickValidation struct {
	// Top is the MobileNet top-N classification of a tight crop around the
	// click point (label + confidence).
	Top []ClassResult `json:"top,omitempty"`
	// Priors describes what the element-priors DB knows about the top
	// classified element in the current window (samples + learned position
	// confidence), when a matching prior entry exists.
	Priors *ClickValidationPriors `json:"priors,omitempty"`
	// MLMemory is a cross-reference into the adaptive/ML engine answering "have
	// I clicked here before in this window context?" Only populated when a
	// prior entry exists (avoids a blind full-query on every click).
	MLMemory *ClickValidationML `json:"ml_memory,omitempty"`
	// WindowTitle is the active window context at validation time.
	WindowTitle string `json:"window_title,omitempty"`
	// Reason is a short note when validation was skipped (no priors, etc.).
	Reason  string `json:"reason,omitempty"`
	TotalMs int64  `json:"total_ms"`
}

// ClickValidationPriors is the priors-DB contribution to a click validation.
type ClickValidationPriors struct {
	Class string `json:"class"`
	// Samples is how many priors samples exist for this class in the window.
	Samples int `json:"samples"`
	// PriorConfidence is the MobileNet confidence after AdjustConfidenceWithPriors.
	PriorConfidence float64 `json:"prior_confidence"`
	// KnownConfidence reports whether the (class, window, location) pair is
	// confidently known per ElementKnownConfidently.
	KnownConfidence bool `json:"known_confidence"`
	// Frequency is the learned occurrence frequency of this class in the window.
	Frequency float64 `json:"frequency,omitempty"`
	// AvgX/AvgY are the learned average normalized position (0-100) of the class.
	AvgX float64 `json:"avg_x,omitempty"`
	AvgY float64 `json:"avg_y,omitempty"`
}

// ClickValidationML is the adaptive/ML memory contribution to a click validation.
type ClickValidationML struct {
	Match      bool    `json:"match"`
	Confidence float64 `json:"confidence,omitempty"`
	X          int     `json:"x,omitempty"`
	Y          int     `json:"y,omitempty"`
	Samples    int     `json:"samples,omitempty"`
}

// lastClickValidation holds the most recent click's advisory validation so the
// server handlers can attach it to the tool result. Server processing is
// single-threaded per request, so a mutex-protected value is more than enough.
var (
	lastClickValMu sync.RWMutex
	lastClickVal   *ClickValidation
)

// classifyClickTarget runs the advisory post-click validation for a click at
// (x, y): MobileNet classification of a tight crop around the point, then a
// priors-based confidence adjustment + known-location check, and an ML-memory
// cross-reference when a prior entry exists. Best-effort: returns a validation
// object with the classify block at minimum, or (if even classification is
// unavailable) a minimal validation carrying a reason — it never returns a hard
// error that could fail the click.
func classifyClickTarget(x, y int32) *ClickValidation {
	start := time.Now()
	title := ""
	if info, err := GetActiveWindowInfo(); err == nil {
		title = info.Title
	}

	rx, ry, rw, rh := SmartRegionAround(x, y, 96)
	out, err := ClassifyRegion(rx, ry, rw, rh, 3)
	if err != nil || out == nil || len(out.Top) == 0 {
		return &ClickValidation{
			WindowTitle: title,
			Reason:      "classify unavailable",
			TotalMs:     time.Since(start).Milliseconds(),
		}
	}

	v := &ClickValidation{
		Top:         out.Top,
		WindowTitle: title,
		TotalMs:     time.Since(start).Milliseconds(),
	}

	topClass := out.Top[0].Label
	samples := PriorSampleCount(title, topClass)
	if samples > 0 {
		priorConf := AdjustConfidenceWithPriors(topClass, title, out.Top[0].Confidence, float64(x), float64(y))
		freq, avgX, avgY := priorStats(title, topClass)
		v.Priors = &ClickValidationPriors{
			Class:           topClass,
			Samples:         samples,
			PriorConfidence: priorConf,
			KnownConfidence: ElementKnownConfidently(title, topClass, 0, 0, out.Top[0].Confidence, 3, watcherLocTolerance),
			Frequency:       freq,
			AvgX:            avgX,
			AvgY:            avgY,
		}
		// ML-memory cross-reference only when we have a priors entry, so we
		// don't run a blind full query on every click.
		if q := Adaptive.MLQuery("click "+topClass, "", 3); q != nil && len(q.Matches) > 0 {
			best := q.Matches[0]
			ml := &ClickValidationML{
				Match:      best.Coord != nil,
				Confidence: best.Confidence,
				Samples:    best.Samples,
			}
			if best.Coord != nil {
				ml.X = best.Coord.X
				ml.Y = best.Coord.Y
			}
			v.MLMemory = ml
		}
	} else {
		reason := "no priors"
		if title == "" {
			reason = "no window context"
		}
		v.Reason = reason
	}

	v.TotalMs = time.Since(start).Milliseconds()
	return v
}

// priorStats returns the learned frequency and average normalized position for a
// (windowTitle, className) prior pair, or zeroes when none exists.
func priorStats(windowTitle, className string) (frequency, avgX, avgY float64) {
	loadPriorsOnce.Do(loadPriorsFromDB)
	elementPriors.mu.RLock()
	defer elementPriors.mu.RUnlock()
	normalized := normalizeWindowTitle(windowTitle)
	for i := range elementPriors.priors {
		p := &elementPriors.priors[i]
		if p.Class == className && p.WindowTitle == normalized {
			return p.Frequency, p.AvgX, p.AvgY
		}
	}
	return 0, 0, 0
}

// RecordClickValidation stores the advisory validation for the most recent click
// so a server handler can attach it to the click tool result.
func RecordClickValidation(v *ClickValidation) {
	lastClickValMu.Lock()
	lastClickVal = v
	lastClickValMu.Unlock()
}

// LastClickValidation returns (and clears) the most recent click validation.
func LastClickValidation() *ClickValidation {
	lastClickValMu.Lock()
	v := lastClickVal
	lastClickVal = nil
	lastClickValMu.Unlock()
	return v
}
