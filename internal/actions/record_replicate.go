package actions

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"
)

// ── Menu position memory cache ──

// menuCacheEntry stores a context menu item's relative position from the click point.
type menuCacheEntry struct {
	ItemText   string `json:"item_text"`
	RelativeX  int32  `json:"relative_x"`
	RelativeY  int32  `json:"relative_y"`
	HitCount   int    `json:"hit_count"`
}

// menuCacheKey identifies a right-click context by window title + click position bucket.
type menuCacheKey struct {
	Window string
	BucketX int32
	BucketY int32
}

var (
	menuCache   = map[menuCacheKey][]menuCacheEntry{}
	menuCacheMu sync.RWMutex

	// windowsAtRecordStart holds the window snapshot captured at Record() start.
	windowsAtRecordStart []WindowStateInfo
)

// cacheMenuItems stores observed menu items for a right-click context.
func cacheMenuItems(window string, clickX, clickY int32, items []string) {
	if len(items) == 0 {
		return
	}
	// Bucket positions to 50px grid for fuzzy matching
	key := menuCacheKey{
		Window:  window,
		BucketX: (clickX / 50) * 50,
		BucketY: (clickY / 50) * 50,
	}
	entries := make([]menuCacheEntry, len(items))
	for i, item := range items {
		entries[i] = menuCacheEntry{
			ItemText:  item,
			RelativeX: 0,
			RelativeY: int32(i) * 24, // typical menu item height ~24px
			HitCount:  1,
		}
	}
	menuCacheMu.Lock()
	defer menuCacheMu.Unlock()
	// Merge with existing entries
	existing := menuCache[key]
	for _, new := range entries {
		found := false
		for i := range existing {
			if existing[i].ItemText == new.ItemText {
				existing[i].HitCount++
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, new)
		}
	}
	menuCache[key] = existing
}

// lookupMenuItems returns cached menu items for a right-click context, or nil.
func lookupMenuItems(window string, clickX, clickY int32) []menuCacheEntry {
	key := menuCacheKey{
		Window:  window,
		BucketX: (clickX / 50) * 50,
		BucketY: (clickY / 50) * 50,
	}
	menuCacheMu.RLock()
	defer menuCacheMu.RUnlock()
	return menuCache[key]
}

// ── Enriched recording ──

// EnrichedEvent extends a raw recordedEvent with contextual information
// captured at the time of the event (OCR text, UIA element, window title).
type EnrichedEvent struct {
	Kind        string            `json:"kind"`
	Button      string            `json:"button,omitempty"`
	KeyName     string            `json:"key_name,omitempty"`
	Modifiers   []string          `json:"modifiers,omitempty"`
	X           int32             `json:"x,omitempty"`
	Y           int32             `json:"y,omitempty"`
	StartX      int32             `json:"start_x,omitempty"`
	StartY      int32             `json:"start_y,omitempty"`
	ScrollDelta int32             `json:"scroll_delta,omitempty"`
	Text        string            `json:"text,omitempty"`
	OCRText     string            `json:"ocr_text,omitempty"`
	UIAElement  map[string]any    `json:"uia_element,omitempty"`
	MLCoords    *MLPrediction     `json:"ml_coords,omitempty"`
	Window      string            `json:"window,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	DelayMs     int64             `json:"delay_ms,omitempty"`
	ElapsedMs   int64             `json:"elapsed_ms,omitempty"`
	Down        bool              `json:"down,omitempty"`
	Clicks      int               `json:"clicks,omitempty"`
	Source      string            `json:"source,omitempty"`
	MenuItems   []string          `json:"menu_items,omitempty"` // context menu items observed after right-click
}

// MLPrediction stores ML-predicted coordinates for a click target.
type MLPrediction struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Confidence float64 `json:"confidence"`
}

// RecordSession is the full output of a recording session.
type RecordSession struct {
	Events          []EnrichedEvent  `json:"events"`
	Meta            map[string]any   `json:"meta"`
	RecordedAt      time.Time        `json:"recorded_at"`
	DurationSec     int              `json:"duration_secs"`
	WindowsAtStart  []WindowStateInfo `json:"windows_at_start,omitempty"`
}

// ── Record ──

// Record starts recording and optionally waits for durationSecs before stopping.
// If durationSecs <= 0, recording starts and the caller must call RecordStop() later.
// Returns the enriched session with OCR, UIA, and window context at each click.
func Record(durationSecs int) (*RecordSession, error) {
	if err := StartKeylogger(); err != nil {
		return nil, fmt.Errorf("failed to start recording: %w", err)
	}

	// Capture window state at recording start for accurate restoration later
	windowsAtRecordStart = nil
	if wins, err := ListWindows(); err == nil {
		for _, w := range wins {
			if state, err := GetWindowState(w.Handle); err == nil {
				windowsAtRecordStart = append(windowsAtRecordStart, *state)
			}
		}
	}

	if durationSecs <= 0 {
		slog.Info("record: started (manual stop required)")
		return nil, nil
	}

	slog.Info("record: started, recording for", "duration_secs", durationSecs)
	go func() {
		time.Sleep(time.Duration(durationSecs) * time.Second)
		if _, err := RecordStop(); err != nil {
			slog.Error("record: auto-stop failed", "err", err)
		}
	}()
	return nil, nil
}

// RecordStop stops an active recording and returns the enriched session.
// Use this after Record(0) or keylogger_start when ready to stop.
func RecordStop() (*RecordSession, error) {
	// Use the window snapshot captured at Record() start
	windowsAtStart := windowsAtRecordStart
	windowsAtRecordStart = nil

	rawSteps, meta, err := StopKeylogger()
	if err != nil {
		return nil, fmt.Errorf("failed to stop recording: %w", err)
	}

	events := enrichEvents(rawSteps, meta)

	duration := 0
	if d, ok := meta["duration_seconds"].(int); ok {
		duration = d
	}

	session := &RecordSession{
		Events:         events,
		Meta:           meta,
		RecordedAt:     time.Now(),
		DurationSec:    duration,
		WindowsAtStart: windowsAtStart,
	}

	slog.Info("record: complete", "events", len(events), "duration", duration, "windows", len(windowsAtStart))

	// Feed enrichment-level patterns (OCR, UIA, ML payloads) back to the ML engine
	// Run async to avoid blocking the MCP response
	go LogEnrichPatternsFromSession(session, true)

	return session, nil
}

// enrichEvents takes raw keylogger steps (maps) and enriches click/drag
// events with OCR, UIA, and window context. Also accumulates consecutive
// character key events into type steps, and detects modifier combos.
func enrichEvents(rawSteps []map[string]any, meta map[string]any) []EnrichedEvent {
	if len(rawSteps) == 0 {
		return nil
	}

	var events []EnrichedEvent
	var lastWindow string

	if w, ok := meta["window_start"].(string); ok {
		lastWindow = w
	}

	// Text accumulation state
	var textBuf []rune
	modifiers := map[string]bool{
		"CTRL": false, "CONTROL": false, "ALT": false,
		"SHIFT": false, "LWIN": false, "RWIN": false,
	}
	hasModifier := func() bool {
		return modifiers["CTRL"] || modifiers["CONTROL"] || modifiers["ALT"] ||
			modifiers["LWIN"] || modifiers["RWIN"]
	}
	activeModifiers := func() []string {
		var mods []string
		for name, held := range modifiers {
			if held {
				mods = append(mods, name)
			}
		}
		return mods
	}
	flushText := func() {
		if len(textBuf) > 0 {
			events = append(events, EnrichedEvent{
				Kind:    "type",
				Text:    string(textBuf),
				Window:  lastWindow,
				Source:  "keylogger",
			})
			textBuf = nil
		}
	}

	for _, raw := range rawSteps {
		tool, _ := raw["tool"].(string)
		args, _ := raw["args"].(map[string]any)

		switch tool {
		case "key_down":
			keyName, _ := args["key"].(string)
			ku := strings.ToUpper(keyName)

			// Track modifier state
			if isModifierName(ku) {
				modifiers[ku] = true
				continue
			}

			// If modifier held, this is a combo — flush text and emit key_combo
			if hasModifier() {
				flushText()
				events = append(events, EnrichedEvent{
					Kind:      "key_combo",
					KeyName:   keyName,
					Modifiers: activeModifiers(),
					Window:    lastWindow,
					Source:    "keylogger",
					Down:      true,
				})
				continue
			}

			// Try to resolve VK code to printable character
			vk, vkOK := resolveVK(ku)
			if vkOK {
				if modifiers["SHIFT"] {
					// Shift held: prefer shifted variant (e.g. !, @, _, +), fall back to uppercase letter
					if shifted, ok := vkToShiftedChar[vk]; ok {
						textBuf = append(textBuf, shifted)
					} else if ch, ok := vkToChar[vk]; ok {
						if ch >= 'a' && ch <= 'z' {
							ch = ch - 'a' + 'A'
						}
						textBuf = append(textBuf, ch)
					}
				} else if ch, ok := vkToChar[vk]; ok {
					textBuf = append(textBuf, ch)
				}
				continue
			}

			// Non-printable key (Enter, Tab, Backspace, F-keys, arrows, etc.)
			flushText()
			events = append(events, EnrichedEvent{
				Kind:    "key_down",
				KeyName: keyName,
				Window:  lastWindow,
				Source:  "keylogger",
				Down:    true,
			})

		case "key_up":
			keyName, _ := args["key"].(string)
			ku := strings.ToUpper(keyName)
			if isModifierName(ku) {
				modifiers[ku] = false
				continue
			}
			if hasModifier() {
				events = append(events, EnrichedEvent{
					Kind:      "key_combo",
					KeyName:   keyName,
					Modifiers: activeModifiers(),
					Window:    lastWindow,
					Source:    "keylogger",
					Down:      false,
				})
				continue
			}
			events = append(events, EnrichedEvent{
				Kind:    "key_up",
				KeyName: keyName,
				Window:  lastWindow,
				Source:  "keylogger",
				Down:    false,
			})

		case "click":
			flushText()
			ev := EnrichedEvent{
				Kind:   "click",
				Source: "keylogger",
			}
			ev.X = toInt32(args["x"])
			ev.Y = toInt32(args["y"])
			if b, ok := args["button"].(string); ok {
				ev.Button = b
			}
			if c, ok := args["clicks"].(int); ok {
				ev.Clicks = c
			}
			if e, ok := args["elapsed_ms"].(int64); ok {
				ev.ElapsedMs = e
			} else if e, ok := args["elapsed_ms"].(float64); ok {
				ev.ElapsedMs = int64(e)
			}
			ev.Window = lastWindow
			enrichClick(&ev)
			events = append(events, ev)

		case "move_mouse":
			flushText()
			ev := EnrichedEvent{
				Kind:   "move_mouse",
				Source: "keylogger",
			}
			ev.X = toInt32(args["x"])
			ev.Y = toInt32(args["y"])
			ev.Window = lastWindow
			events = append(events, ev)

		case "drag":
			flushText()
			ev := EnrichedEvent{
				Kind:   "drag",
				Source: "keylogger",
			}
			ev.StartX = toInt32(args["from_x"])
			ev.StartY = toInt32(args["from_y"])
			ev.X = toInt32(args["to_x"])
			ev.Y = toInt32(args["to_y"])
			if b, ok := args["button"].(string); ok {
				ev.Button = b
			}
			if e, ok := args["elapsed_ms"].(int64); ok {
				ev.ElapsedMs = e
			} else if e, ok := args["elapsed_ms"].(float64); ok {
				ev.ElapsedMs = int64(e)
			}
			ev.Window = lastWindow
			enrichClick(&ev)
			events = append(events, ev)

		case "scroll":
			flushText()
			ev := EnrichedEvent{
				Kind:        "scroll",
				ScrollDelta: toInt32(args["clicks"]),
				Window:      lastWindow,
				Source:      "keylogger",
			}
			events = append(events, ev)

		case "_focus":
			flushText()
			focusName, _ := args["window"].(string)
			if focusName != "" {
				lastWindow = focusName
			}
			events = append(events, EnrichedEvent{
				Kind:    "focus",
				KeyName: focusName,
				Window:  lastWindow,
				Source:  "keylogger",
			})
		}
	}

	flushText()

	// Post-process: merge double-clicks (2 rapid left-clicks within 400ms at same position)
	events = mergeDoubleClicks(events, 400)

	// Post-process: detect long-press (hold > 500ms without movement)
	events = detectLongPress(events, 500)

	return events
}

// detectLongPress converts click events with elapsedMs > threshold into long_press events.
func detectLongPress(events []EnrichedEvent, thresholdMs int64) []EnrichedEvent {
	for i := range events {
		ev := &events[i]
		if ev.Kind == "click" && ev.ElapsedMs > thresholdMs {
			slog.Info("detectLongPress: converting click to long_press",
				"position", fmt.Sprintf("(%d,%d)", ev.X, ev.Y),
				"elapsed_ms", ev.ElapsedMs)
			ev.Kind = "long_press"
		}
	}
	return events
}

// mergeDoubleClicks scans for two consecutive left-click events within the
// threshold at the same position and merges them into a single double_click.
func mergeDoubleClicks(events []EnrichedEvent, thresholdMs int64) []EnrichedEvent {
	merged := make([]EnrichedEvent, 0, len(events))
	i := 0
	for i < len(events) {
		ev := events[i]
		if ev.Kind == "click" && (ev.Button == "" || ev.Button == "left") {
			// Look ahead for a second click close enough in time and space
			if i+1 < len(events) {
				next := events[i+1]
				if next.Kind == "click" && (next.Button == "" || next.Button == "left") {
					timeDiff := next.Timestamp.Sub(ev.Timestamp)
					if timeDiff >= 0 && timeDiff <= time.Duration(thresholdMs)*time.Millisecond {
						dx := int32(next.X) - int32(ev.X)
						dy := int32(next.Y) - int32(ev.Y)
						if dx < 0 {
							dx = -dx
						}
						if dy < 0 {
							dy = -dy
						}
						if dx <= 8 && dy <= 8 {
							// Merge: keep first click's position + OCR/UIA, set Kind to double_click
							ev.Kind = "double_click"
							ev.Button = "left"
							if next.Timestamp.After(ev.Timestamp) {
								ev.Timestamp = next.Timestamp
							}
							if ev.OCRText == "" {
								ev.OCRText = next.OCRText
							}
							if ev.UIAElement == nil {
								ev.UIAElement = next.UIAElement
							}
							merged = append(merged, ev)
							i += 2
							continue
						}
					}
				}
			}
		}
		merged = append(merged, ev)
		i++
	}
	return merged
}

// LogEnrichPattern captures an enrichment-level event (double_click, long_press,
// context_menu) and feeds it into the ML engine with OCR context so the AI learns
// these higher-level patterns, not just the individual chain steps.
func LogEnrichPattern(kind string, x, y int32, elapsedMs int64, window string, success bool) {
	if kind != "double_click" && kind != "long_press" && kind != "context_menu" {
		return
	}

	// Capture OCR context at the event coordinates
	ocrText := ""
	if ocrResult, err := OCRScreen(""); err == nil && ocrResult != nil {
		ocrText = nearbyOCRText(ocrResult.Words, float64(x), float64(y), 200, 5)
	}

	argsJSON := fmt.Sprintf(`{"x":%d,"y":%d}`, x, y)
	if elapsedMs > 0 {
		argsJSON = fmt.Sprintf(`{"x":%d,"y":%d,"elapsed_ms":%d}`, x, y, elapsedMs)
	}

	// Feed to ML engine: coordIndex + timing stats
	if ocrText != "" {
		Adaptive.LearnFromCommandWithContext(kind, argsJSON, ocrText, success)
	}
	Adaptive.LearnFromCommand(kind, argsJSON, success)
	Adaptive.RecordResult(kind, float64(elapsedMs), success)

	slog.Info("LogEnrichPattern",
		"kind", kind,
		"position", fmt.Sprintf("(%d,%d)", x, y),
		"ocr_len", len(ocrText),
		"elapsed_ms", elapsedMs,
		"success", success)
}

// LogEnrichPatternsFromSession logs all enrichment-level events from a recording
// session after chain execution, so the ML learns the higher-level patterns.
func LogEnrichPatternsFromSession(session *RecordSession, success bool) {
	if session == nil || len(session.Events) == 0 {
		return
	}
	for _, ev := range session.Events {
		switch ev.Kind {
		case "double_click":
			LogEnrichPattern("double_click", ev.X, ev.Y, 0, ev.Window, success)
		case "long_press":
			LogEnrichPattern("long_press", ev.X, ev.Y, ev.ElapsedMs, ev.Window, success)
		}
		// Context menu: right-click events
		if ev.Kind == "click" && ev.Button == "right" {
			LogEnrichPattern("context_menu", ev.X, ev.Y, 0, ev.Window, success)
		}
	}
}

// isModifierName returns true if the key name is a modifier.
func isModifierName(ku string) bool {
	return ku == "CTRL" || ku == "CONTROL" || ku == "ALT" ||
		ku == "SHIFT" || ku == "LWIN" || ku == "RWIN"
}

// resolveVK resolves a key name to its VK code using the existing maps.
func resolveVK(ku string) (uint32, bool) {
	if vk, ok := vkModMap[ku]; ok {
		return uint32(vk), true
	}
	if vk, ok := vkSpecialMap[ku]; ok {
		return uint32(vk), true
	}
	if len(ku) == 1 {
		ch := ku[0]
		if ch >= 'A' && ch <= 'Z' {
			return uint32(ch), true
		}
		if ch >= '0' && ch <= '9' {
			return uint32(ch), true
		}
	}
	return 0, false
}

// enrichClick captures OCR text and UIA element at the click coordinates.
func enrichClick(ev *EnrichedEvent) {
	// Capture OCR text near click point
	if ocrResult, err := OCRScreen(""); err == nil {
		ev.OCRText = nearbyOCRText(ocrResult.Words, float64(ev.X), float64(ev.Y), 100, 5)
	}

	// Capture UIA element at click point
	if el, err := UIAElementFromPoint(ev.X, ev.Y); err == nil && el != nil {
		ev.UIAElement = map[string]any{
			"name":          el.Name,
			"automation_id": el.AutomationID,
			"control_type":  el.ControlType,
			"is_enabled":    el.IsEnabled,
		}
	}

	// Try ML prediction for this coordinate
	if pred := predictAtCoord(ev.X, ev.Y); pred != nil {
		ev.MLCoords = pred
	}
}

// predictAtCoord uses the Adaptive engine to find the best predicted coordinates
// for a click near the given position.
func predictAtCoord(x, y int32) *MLPrediction {
	ocrText := ""
	if ocrResult, err := OCRScreen(""); err == nil {
		ocrText = nearbyOCRText(ocrResult.Words, float64(x), float64(y), 200, 3)
	}
	if ocrText == "" {
		return nil
	}

	preds := Adaptive.PredictActions(ocrText, 1)
	if len(preds) == 0 || preds[0].Coord == nil {
		return nil
	}

	p := preds[0]
	return &MLPrediction{
		X:          p.Coord.X,
		Y:          p.Coord.Y,
		Confidence: p.Confidence,
	}
}

func toInt32(v any) int32 {
	switch val := v.(type) {
	case int32:
		return val
	case int:
		return int32(val)
	case float64:
		return int32(val)
	default:
		return 0
	}
}

// ── Replicate ──

// ReplicateResult is the output of the Replicate function.
type ReplicateResult struct {
	Steps       []ChainStep `json:"steps"`
	StepCount   int         `json:"step_count"`
	Meta        map[string]any `json:"meta"`
}

// Replicate takes a recorded session and generates intelligent chain steps.
// Priority for click events: UIA invoke > OCR find_text_and_click > ML coords > raw coords.
// Mouse movements are preserved as move_mouse steps.
func Replicate(session *RecordSession, slowdown, loop int) (*ReplicateResult, error) {
	if session == nil || len(session.Events) == 0 {
		return nil, fmt.Errorf("empty session")
	}
	if slowdown < 1 {
		slowdown = 1
	}
	if loop < 1 {
		loop = 1
	}

	slog.Info("replicate: generating chain", "events", len(session.Events), "slowdown", slowdown, "loop", loop)

	chainSteps := eventsToSmartSteps(session.Events)

	// Add window state restoration steps for ALL windows that were open at recording start.
	// Uses find_window to get current handles (stale handles from recording won't work),
	// then restores position, size, min/max state, and focus.
	if len(session.WindowsAtStart) > 0 {
		var preamble []ChainStep
		for i, w := range session.WindowsAtStart {
			if w.Title == "" {
				continue
			}
			varName := fmt.Sprintf("win_%d", i)

			// Step 1: Find window by title, capture handle
			preamble = append(preamble, ChainStep{
				Tool:    "find_window",
				Args:    map[string]any{"title": w.Title},
				Capture: varName,
			})

			// Step 2: Restore position and size
			if w.Rect != nil {
				preamble = append(preamble, ChainStep{
					Tool: "move_window",
					Args: map[string]any{
						"handle": fmt.Sprintf("${%s.handle}", varName),
						"x":      w.Rect.Left,
						"y":      w.Rect.Top,
						"width":  w.Rect.Width,
						"height": w.Rect.Height,
					},
				})
			}

			// Step 3: Restore minimized/maximized state
			if w.Minimized {
				preamble = append(preamble, ChainStep{
					Tool: "restore_window",
					Args: map[string]any{
						"handle": fmt.Sprintf("${%s.handle}", varName),
					},
				})
			} else if w.Maximized {
				preamble = append(preamble, ChainStep{
					Tool: "maximize_window",
					Args: map[string]any{
						"handle": fmt.Sprintf("${%s.handle}", varName),
					},
				})
			}

			// Step 4: Focus the window (by title, not handle)
			preamble = append(preamble, ChainStep{
				FocusWindow: w.Title,
			})
		}
		chainSteps = append(preamble, chainSteps...)
	}

	if slowdown > 1 {
		chainSteps = applySlowdownChain(chainSteps, slowdown)
	}
	if loop > 1 {
		chainSteps = duplicateChainSteps(chainSteps, loop)
	}

	result := &ReplicateResult{
		Steps:     chainSteps,
		StepCount: len(chainSteps),
		Meta: map[string]any{
			"original_events": len(session.Events),
			"slowdown":        slowdown,
			"loop":            loop,
		},
	}

	slog.Info("replicate: done", "chain_steps", len(chainSteps))
	return result, nil
}

// eventsToSmartSteps converts enriched events to intelligent chain steps.
func eventsToSmartSteps(events []EnrichedEvent) []ChainStep {
	var steps []ChainStep

	for _, ev := range events {
		if ev.DelayMs > 0 {
			steps = append(steps, ChainStep{
				Tool: "wait",
				Args: map[string]any{"ms": ev.DelayMs},
			})
		}

		switch ev.Kind {
		case "click":
			switch ev.Button {
			case "right":
				// Right-click: click + wait for menu + cache-aware menu item detection
				steps = append(steps, ChainStep{
					Tool: "click",
					Args: map[string]any{"x": ev.X, "y": ev.Y, "button": "right"},
				})
				steps = append(steps, ChainStep{
					Tool: "wait",
					Args: map[string]any{"ms": 350},
				})
				// Check menu position cache for known items
				cached := lookupMenuItems(ev.Window, ev.X, ev.Y)
				if len(cached) > 0 {
					// Use first most-hit cached menu item
					best := cached[0]
					for _, e := range cached[1:] {
						if e.HitCount > best.HitCount {
							best = e
						}
					}
					steps = append(steps, ChainStep{
						Tool: "find_text_and_click",
						Args: map[string]any{"text": best.ItemText, "skip_system_find": true},
					})
				} else {
					// No cache: use UIA MenuItem scan + OCR fallback
					steps = append(steps, ChainStep{
						Tool: "uia_find",
						Args: map[string]any{"control_type": "MenuItem"},
					})
					if ev.OCRText != "" {
						steps = append(steps, ChainStep{
							Tool: "find_text_and_click",
							Args: map[string]any{"text": ev.OCRText, "skip_system_find": true},
						})
					}
				}

			case "middle":
				// Middle-click → paste (common convention)
				steps = append(steps, ChainStep{
					Tool: "click",
					Args: map[string]any{"x": ev.X, "y": ev.Y, "button": "middle"},
				})

			case "xbutton1":
				// X1 → browser back (Alt+Left)
				steps = append(steps, ChainStep{
					Tool: "key_press",
					Args: map[string]any{"keys": []string{"Alt", "Left"}},
				})

			case "xbutton2":
				// X2 → browser forward (Alt+Right)
				steps = append(steps, ChainStep{
					Tool: "key_press",
					Args: map[string]any{"keys": []string{"Alt", "Right"}},
				})

			default:
				// Left-click: smart priority UIA > OCR > ML > raw
				steps = append(steps, smartClickSteps(ev)...)
			}

		case "double_click":
			// Double-click: smart click with clicks:2, or UIA invoke (single step is enough)
			baseSteps := smartClickSteps(ev)
			if len(baseSteps) == 1 && baseSteps[0].Tool == "uia_invoke" {
				steps = append(steps, baseSteps...)
			} else if len(baseSteps) == 1 && baseSteps[0].Tool == "click" {
				if baseSteps[0].Args == nil {
					baseSteps[0].Args = map[string]any{}
				}
				baseSteps[0].Args["clicks"] = 2
				steps = append(steps, baseSteps...)
			} else {
				// find_text_and_click or multi-step: emit steps + raw double-click
				steps = append(steps, baseSteps...)
				steps = append(steps, ChainStep{
					Tool: "click",
					Args: map[string]any{"x": ev.X, "y": ev.Y, "clicks": 2},
				})
			}

		case "long_press":
			// Long-press: move → mouse_down → wait → mouse_up (actual hold)
			holdMs := ev.ElapsedMs
			if holdMs < 500 {
				holdMs = 500
			}
			button := ev.Button
			if button == "" {
				button = "left"
			}
			steps = append(steps, ChainStep{
				Tool: "move_mouse",
				Args: map[string]any{"x": ev.X, "y": ev.Y},
			})
			steps = append(steps, ChainStep{
				Tool: "mouse_down",
				Args: map[string]any{"x": ev.X, "y": ev.Y, "button": button},
			})
			steps = append(steps, ChainStep{
				Tool: "wait",
				Args: map[string]any{"ms": holdMs},
			})
			steps = append(steps, ChainStep{
				Tool: "mouse_up",
				Args: map[string]any{"x": ev.X, "y": ev.Y, "button": button},
			})

		case "drag":
			dragArgs := map[string]any{
				"from_x": ev.StartX, "from_y": ev.StartY,
				"to_x": ev.X, "to_y": ev.Y,
			}
			if ev.Button != "" && ev.Button != "left" {
				dragArgs["button"] = ev.Button
			}
			steps = append(steps, ChainStep{
				Tool: "drag",
				Args: dragArgs,
			})

		case "move_mouse":
			steps = append(steps, ChainStep{
				Tool: "move_mouse",
				Args: map[string]any{"x": ev.X, "y": ev.Y},
			})

		case "scroll":
			steps = append(steps, ChainStep{
				Tool: "scroll",
				Args: map[string]any{"clicks": ev.ScrollDelta},
			})

		case "type":
			if ev.Text != "" {
				steps = append(steps, ChainStep{
					Tool: "type",
					Args: map[string]any{"text": ev.Text},
				})
			}

		case "key_combo":
			// Build full key list: modifiers first, then the key
			var keys []string
			for _, m := range ev.Modifiers {
				switch m {
				case "CTRL", "CONTROL":
					keys = append(keys, "Ctrl")
				case "ALT":
					keys = append(keys, "Alt")
				case "SHIFT":
					keys = append(keys, "Shift")
				case "LWIN", "RWIN":
					keys = append(keys, "Win")
				}
			}
			keys = append(keys, ev.KeyName)
			steps = append(steps, ChainStep{
				Tool: "key_press",
				Args: map[string]any{"keys": keys},
			})

		case "key_down":
			steps = append(steps, ChainStep{
				Tool: "key_down",
				Args: map[string]any{"key": ev.KeyName},
			})

		case "key_up":
			steps = append(steps, ChainStep{
				Tool: "key_up",
				Args: map[string]any{"key": ev.KeyName},
			})

		case "focus":
			if ev.KeyName != "" {
				steps = append(steps, ChainStep{
					FocusWindow: ev.KeyName,
				})
			}
		}
	}

	return steps
}

// smartClickSteps generates the best chain steps for a click event.
// Priority: UIA invoke > OCR find_text_and_click > ML predicted coords > raw coords.
func smartClickSteps(ev EnrichedEvent) []ChainStep {
	var steps []ChainStep

	// 1. If UIA element was found, use uia_invoke (most reliable)
	if ev.UIAElement != nil {
		name, _ := ev.UIAElement["name"].(string)
		aid, _ := ev.UIAElement["automation_id"].(string)
		if name != "" || aid != "" {
			args := map[string]any{}
			if name != "" {
				args["name"] = name
			}
			if aid != "" {
				args["automation_id"] = aid
			}
			steps = append(steps, ChainStep{
				Tool: "uia_invoke",
				Args: args,
			})
			return steps
		}
	}

	// 2. If OCR text was found, use find_text_and_click (reliable)
	if ev.OCRText != "" {
		args := map[string]any{"text": ev.OCRText}
		if ev.Window != "" {
			args["window_title"] = ev.Window
		}
		steps = append(steps, ChainStep{
			Tool: "find_text_and_click",
			Args: args,
		})
		return steps
	}

	// 3. If ML prediction available with good confidence, use predicted coords
	if ev.MLCoords != nil && ev.MLCoords.Confidence > 0.5 {
		args := map[string]any{"x": ev.MLCoords.X, "y": ev.MLCoords.Y}
		if ev.Button != "" && ev.Button != "left" {
			args["button"] = ev.Button
		}
		if ev.Clicks > 1 {
			args["clicks"] = ev.Clicks
		}
		steps = append(steps, ChainStep{
			Tool: "click",
			Args: args,
		})
		return steps
	}

	// 4. Fallback: raw coordinates
	args := map[string]any{"x": ev.X, "y": ev.Y}
	if ev.Button != "" && ev.Button != "left" {
		args["button"] = ev.Button
	}
	if ev.Clicks > 1 {
		args["clicks"] = ev.Clicks
	}
	steps = append(steps, ChainStep{
		Tool: "click",
		Args: args,
	})
	return steps
}

// ── Combined record_and_replicate ──

// RecordReplicateResult is the output of the combined RecordAndReplicate.
type RecordReplicateResult struct {
	Session       *RecordSession   `json:"session,omitempty"`
	EventsRecorded int             `json:"events_recorded"`
	StepsGenerated int             `json:"steps_generated"`
	DurationSecs   int             `json:"duration_secs"`
	LoopCount      int             `json:"loop_count"`
	Slowdown       int             `json:"slowdown"`
	Meta           map[string]any  `json:"meta"`
	ReplayResult   *ChainResult    `json:"replay_result,omitempty"`
	ReplayError    string          `json:"replay_error,omitempty"`
}

// RecordAndReplicate records input for durationSecs then replays it.
// If durationSecs <= 0, uses manual stop via keylogger_stop (no auto-replay).
func RecordAndReplicate(durationSecs, delayMs, slowdown, loop int) (*RecordReplicateResult, error) {
	if durationSecs < 0 {
		return nil, fmt.Errorf("duration_secs must be >= 0, got %d", durationSecs)
	}
	if loop < 1 {
		loop = 1
	}
	if slowdown < 1 {
		slowdown = 1
	}
	if delayMs < 0 {
		delayMs = 1000
	}

	session, err := Record(durationSecs)
	if err != nil {
		return nil, err
	}

	// Manual mode: Record(0) returns nil session, caller must use record_stop
	if session == nil {
		return &RecordReplicateResult{
			DurationSecs: durationSecs,
			LoopCount:    loop,
			Slowdown:     slowdown,
		}, nil
	}

	result := &RecordReplicateResult{
		Session:        session,
		EventsRecorded: len(session.Events),
		DurationSecs:   durationSecs,
		LoopCount:      loop,
		Slowdown:       slowdown,
		Meta:           session.Meta,
	}

	if len(session.Events) == 0 {
		slog.Info("record_and_replicate: no events recorded, skipping replay")
		return result, nil
	}

	repl, err := Replicate(session, slowdown, loop)
	if err != nil {
		return nil, fmt.Errorf("replicate failed: %w", err)
	}

	result.StepsGenerated = repl.StepCount

	chainSteps := repl.Steps
	if delayMs > 0 {
		delayStep := ChainStep{
			Tool: "wait",
			Args: map[string]any{"ms": delayMs},
		}
		chainSteps = append([]ChainStep{delayStep}, chainSteps...)
	}

	slog.Info("record_and_replicate: executing replay", "steps", len(chainSteps), "loop", loop)

	chainReq := ChainRequest{
		Steps:     chainSteps,
		TimeoutMs: (durationSecs * loop * slowdown * 1000) + delayMs + 10000,
	}

	chainResult, err := ExecuteChain(chainReq)
	if err != nil {
		result.ReplayError = err.Error()
		return result, fmt.Errorf("replay failed: %w", err)
	}

	result.ReplayResult = chainResult

	// Feed enrichment-level patterns (double_click, long_press, context_menu)
	// into the ML engine so the AI learns these as semantic units, not just
	// the decomposed chain steps.
	chainSuccess := err == nil && (chainResult == nil || chainResult.Success)
	LogEnrichPatternsFromSession(session, chainSuccess)

	return result, nil
}

// ── Helpers ──

func mapsToChainSteps(maps []map[string]any) ([]ChainStep, error) {
	var steps []ChainStep
	for _, m := range maps {
		b, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		var step ChainStep
		if err := json.Unmarshal(b, &step); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func applySlowdownChain(steps []ChainStep, factor int) []ChainStep {
	result := make([]ChainStep, len(steps))
	for i, step := range steps {
		cp := step
		if cp.Args != nil {
			cp.Args = make(map[string]any, len(step.Args))
			for k, v := range step.Args {
				cp.Args[k] = v
			}
		}
		if cp.Tool == "wait" && cp.Args != nil {
			if ms, ok := cp.Args["ms"].(int); ok {
				cp.Args["ms"] = ms * factor
			} else if ms, ok := cp.Args["ms"].(float64); ok {
				cp.Args["ms"] = int(ms) * factor
			}
		}
		result[i] = cp
	}
	return result
}

func duplicateChainSteps(steps []ChainStep, count int) []ChainStep {
	total := len(steps) * count
	result := make([]ChainStep, 0, total)
	for i := 0; i < count; i++ {
		result = append(result, steps...)
	}
	return result
}

func distanceBetween(x1, y1, x2, y2 int32) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}


