package actions

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WatcherState int

const (
	WatcherStopped WatcherState = iota
	WatcherRunning
)

type CachedDetection struct {
	Timestamp       int64                   `json:"ts"`
	WindowTitle     string                  `json:"window_title,omitempty"`
	Elements        []DetectedElement       `json:"elements"`
	Normalized      []NormalizedElement     `json:"normalized,omitempty"`
	Classifications []ElementClassification `json:"classifications,omitempty"`
	SavedRef        string                  `json:"saved_ref,omitempty"`
	TotalMs         int64                   `json:"total_ms"`
}

type WatcherStatus struct {
	Running    bool  `json:"running"`
	IntervalMs int64 `json:"interval_ms"`
	LastRun    int64 `json:"last_run,omitempty"`
	CacheSize  int   `json:"cache_size"`
}

type watcher struct {
	state    atomic.Int32
	interval time.Duration
	stopCh   chan struct{}

	mu       sync.RWMutex
	lastRun  time.Time
	cache    []CachedDetection
	maxCache int
}

var bgWatcher = &watcher{
	maxCache: 20,
}

func StartWatcher(intervalSeconds int) error {
	if intervalSeconds < 1 {
		intervalSeconds = 5
	}
	bgWatcher.mu.Lock()
	if bgWatcher.state.Load() == int32(WatcherRunning) {
		bgWatcher.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}
	bgWatcher.interval = time.Duration(intervalSeconds) * time.Second
	bgWatcher.stopCh = make(chan struct{})
	bgWatcher.state.Store(int32(WatcherRunning))
	bgWatcher.mu.Unlock()

	go bgWatcher.loop()
	return nil
}

func StopWatcher() {
	bgWatcher.mu.Lock()
	defer bgWatcher.mu.Unlock()
	if bgWatcher.state.Load() != int32(WatcherRunning) {
		return
	}
	bgWatcher.state.Store(int32(WatcherStopped))
	close(bgWatcher.stopCh)
}

func GetWatcherStatus() *WatcherStatus {
	bgWatcher.mu.RLock()
	defer bgWatcher.mu.RUnlock()
	s := &WatcherStatus{
		Running:    bgWatcher.state.Load() == int32(WatcherRunning),
		IntervalMs: bgWatcher.interval.Milliseconds(),
		CacheSize:  len(bgWatcher.cache),
	}
	if !bgWatcher.lastRun.IsZero() {
		s.LastRun = bgWatcher.lastRun.UnixMilli()
	}
	return s
}

func GetCachedDetections() []CachedDetection {
	bgWatcher.mu.RLock()
	defer bgWatcher.mu.RUnlock()
	out := make([]CachedDetection, len(bgWatcher.cache))
	copy(out, bgWatcher.cache)
	return out
}

func (w *watcher) loop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.runOnce()
		}
	}
}

func (w *watcher) runOnce() {
	start := time.Now()

	b64, err := CaptureScreen()
	if err != nil {
		return
	}

	result, err := ONNXDetect(DetectionInput{ImageB64: b64})
	if err != nil {
		return
	}

	title := result.WindowTitle
	if title == "" {
		title = getActiveWindowTitle()
	}

	// Hard guard: never harvest training crops from the desktop/wallpaper —
	// YOLO fires spurious object detections there with no task context, which
	// floods disk with garbage. This applies regardless of the lock state.
	if isDesktopWindow(title) {
		det := CachedDetection{
			Timestamp:   start.UnixMilli(),
			WindowTitle: title,
			Elements:    result.Elements,
			Normalized:  result.Normalized,
			SavedRef:    "",
			TotalMs:     result.TotalMs,
		}
		w.mu.Lock()
		w.lastRun = start
		w.cache = append([]CachedDetection{det}, w.cache...)
		if len(w.cache) > w.maxCache {
			w.cache = w.cache[:w.maxCache]
		}
		w.mu.Unlock()
		return
	}

	// The watcher is achievement-locked until the AI has proven competence
	// through real interactive actions (click/type/record) and explicitly
	// unlocked it. While locked, the watcher still detects/caches for
	// reference but never persists training crops.
	locked := ActiveConfig != nil && ActiveConfig.WatcherLocked

	// Advisory MobileNet classification tier: classify each detected element
	// crop so the AI can consult type+confidence alongside the YOLO box. This is
	// best-effort and never blocks detection or training. We run this BEFORE
	// saving so the authoritative UI label can be threaded into the priors/
	// dedup gate (which keys on MobileNet label, not the detector's "icon").
	var classifications []ElementClassification
	if len(result.Elements) > 0 && !isDesktopWindow(title) {
		classifications = classifyElementList(result.Elements, 3)
		// Fold the MobileNet top label back onto each element so downstream
		// consumers (priors, clickable gate) key on the real UI control type.
		for i := range classifications {
			lc := &classifications[i]
			if len(lc.Top) > 0 {
				lc.Element.MobileNetLabel = lc.Top[0].Label
			}
		}
	}

	// When elements are found, save a cropped sample per detected element so
	// the ML sees the element's local context instead of a full-screen image.
	// This keeps training data small and focused. When nothing is detected we
	// skip saving entirely. MobileNet labels (not the detector class) drive the
	// priors/dedup key.
	var savedRef string
	if (ActiveConfig == nil || ActiveConfig.TrainingEnabled) && len(result.Elements) > 0 && !locked && !isDesktopWindow(title) {
		savedRef = saveElementRegionSamples(title, result.Elements, &classifications)
	}

	det := CachedDetection{
		Timestamp:       start.UnixMilli(),
		WindowTitle:     title,
		Elements:        result.Elements,
		Normalized:      result.Normalized,
		Classifications: classifications,
		SavedRef:        savedRef,
		TotalMs:         result.TotalMs,
	}

	w.mu.Lock()
	w.lastRun = start
	w.cache = append([]CachedDetection{det}, w.cache...)
	if len(w.cache) > w.maxCache {
		w.cache = w.cache[:w.maxCache]
	}
	w.mu.Unlock()
}

func getActiveWindowTitle() string {
	info, err := GetActiveWindowInfo()
	if err != nil {
		return ""
	}
	return info.Title
}

// watcherDedupThreshold is the number of times an (element class, window) pair
// must be observed before the watcher stops saving new crops of it. This
// prevents over-training (the ML already knows the element) and disk flooding
// from near-identical full screenshots every cycle.
const watcherDedupThreshold = 5

// watcherLocTolerance is the normalized (0-1) spatial tolerance used by the
// watcher's novelty gate. An element detected within this distance of a
// known prior's average location is treated as already-known and skipped.
const watcherLocTolerance = 0.15

// watcherCropPadding is added around each detected element's bounding box when
// capturing its focused training crop, giving the ML a little local context.
const watcherCropPadding = 24

// watcherDesktopTitles are window titles that indicate no real application is
// in the foreground (the desktop / wallpaper / shell). The watcher must never
// harvest training samples from these: YOLO fires spurious "person"/object
// detections on a wallpaper with no task context, flooding disk with garbage.
// Even when the watcher is unlocked, crops from the desktop are discarded.
var watcherDesktopTitles = map[string]bool{
	"program manager": true,
	"":                true,
}

// isDesktopWindow reports whether the given window title represents the
// desktop/wallpaper rather than a real application window that the AI acted
// on. Used as a hard guard so the watcher only learns from real app contexts.
func isDesktopWindow(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	if watcherDesktopTitles[t] {
		return true
	}
	// Shell/desktop viewer windows.
	if strings.Contains(t, "shell_") || strings.Contains(t, "windows shell experience") {
		return true
	}
	return false
}

// elementCropRegion converts an ONNX-detected element's bitmap-space bounding
// box (relative to a full-screen capture whose origin is the virtual screen
// top-left) into a padded, bounds-clamped virtual-screen CaptureRegion rect.
// Returns ok=false when the resulting crop is empty or invalid.
func elementCropRegion(bounds Rect, el DetectedElement) (x, y, w, h int32, ok bool) {
	if el.W <= 0 || el.H <= 0 {
		return 0, 0, 0, 0, false
	}

	vx := bounds.X + el.X
	vy := bounds.Y + el.Y

	px := vx - int32(watcherCropPadding)
	py := vy - int32(watcherCropPadding)
	pw := el.W + int32(2*watcherCropPadding)
	ph := el.H + int32(2*watcherCropPadding)

	if px < bounds.X {
		pw += px - bounds.X
		px = bounds.X
	}
	if py < bounds.Y {
		ph += py - bounds.Y
		py = bounds.Y
	}
	if px+pw > bounds.X+bounds.W {
		pw = bounds.X + bounds.W - px
	}
	if py+ph > bounds.Y+bounds.H {
		ph = bounds.Y + bounds.H - py
	}
	if pw <= 0 || ph <= 0 {
		return 0, 0, 0, 0, false
	}
	return px, py, pw, ph, true
}

// saveElementRegionSamples captures and saves focused, deduplicated training
// crops around each detected element. It returns the path of the first saved
// sample (for the detection cache) or "" if nothing was saved.
func saveElementRegionSamples(windowTitle string, elements []DetectedElement, classifications *[]ElementClassification) string {
	if len(elements) == 0 {
		return ""
	}

	// Build a lookup from element index -> MobileNet UI label so priors/dedup
	// key on the authoritative control type rather than the detector's "icon".
	uiClass := make(map[int]string)
	if classifications != nil {
		uiClass = make(map[int]string, len(*classifications))
		for i := range *classifications {
			el := (*classifications)[i].Element
			if el.MobileNetLabel != "" {
				uiClass[i] = el.MobileNetLabel
			} else if top := (*classifications)[i].Top; len(top) > 0 {
				uiClass[i] = top[0].Label
			}
		}
	}

	bounds := VirtualScreenBounds()
	var savedFirst string
	for i, el := range elements {
		// The priors key is the MobileNet UI control label when available,
		// falling back to the detector class for unclassified elements.
		key := el.Class
		if lbl, ok := uiClass[i]; ok && lbl != "" {
			key = lbl
		}

		// Novelty gate: if the ML already knows this element at this location
		// with high confidence, skip saving another near-identical crop. This is
		// what makes the watcher snapshot less and less as it gains confidence —
		// it only records truly new/moved/uncertain elements instead of flooding.
		normX, normY := 0.0, 0.0
		if bounds.W > 0 && bounds.H > 0 {
			normX = float64(el.X+el.W/2) / float64(bounds.W)
			normY = float64(el.Y+el.H/2) / float64(bounds.H)
		}
		if ElementKnownConfidently(windowTitle, key, normX, normY, el.Confidence,
			watcherDedupThreshold, watcherLocTolerance) {
			continue
		}

		px, py, pw, ph, ok := elementCropRegion(bounds, el)
		if !ok {
			continue
		}

		cat := TrainingCatElementsFound
		task := fmt.Sprintf("find UI element %s in window: %s", key, windowTitle)
		sample, err := SaveRegionTrainingSample(
			TrainingSourceWatcher, cat, task, windowTitle, "",
			px, py, pw, ph,
		)
		if err != nil || sample == nil {
			continue
		}
		if savedFirst == "" {
			savedFirst = sample.ImagePath
		}
	}
	return savedFirst
}
