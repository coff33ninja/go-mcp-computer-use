package actions

import (
	"encoding/json"
	"fmt"
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
	Timestamp   int64               `json:"ts"`
	WindowTitle string              `json:"window_title,omitempty"`
	Elements    []DetectedElement   `json:"elements"`
	Normalized  []NormalizedElement `json:"normalized,omitempty"`
	SavedRef    string              `json:"saved_ref,omitempty"`
	TotalMs     int64               `json:"total_ms"`
}

type WatcherStatus struct {
	Running    bool   `json:"running"`
	IntervalMs int64  `json:"interval_ms"`
	LastRun    int64  `json:"last_run,omitempty"`
	CacheSize  int    `json:"cache_size"`
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

	cat := TrainingCatNoElements
	if len(result.Elements) > 0 {
		cat = TrainingCatElementsFound
	}

	det := CachedDetection{
		Timestamp:   start.UnixMilli(),
		WindowTitle: title,
		Elements:    result.Elements,
		Normalized:  result.Normalized,
		TotalMs:     result.TotalMs,
	}

	windowRect := ""
	if info, err := GetActiveWindowInfo(); err == nil && info != nil && info.Handle != 0 {
		if rect, err := GetWindowRectByHandle(info.Handle); err == nil {
			b, _ := json.Marshal(rect)
			windowRect = string(b)
		}
	}

	if ActiveConfig == nil || ActiveConfig.TrainingEnabled {
		if sample, err := saveTrainingSampleDirect(
			TrainingSourceWatcher, cat,
			fmt.Sprintf("find UI elements in window: %s", title),
			b64, title, "", result.Elements, result.Normalized, windowRect,
		); err == nil && sample != nil {
			det.SavedRef = sample.ImagePath
		}
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
