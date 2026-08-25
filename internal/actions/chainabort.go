package actions

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

var (
	abortMu     sync.Mutex
	abortCh     chan struct{}
	abortStop   chan struct{}
	abortKeys   []uint32
	abortPollMs int
)

// ParseHotkeyString converts a hotkey string like "Ctrl+Shift+Escape" to VK codes.
// Uses vkModMap and vkSpecialMap from keyboard.go.
func ParseHotkeyString(hotkey string) []uint32 {
	var codes []uint32
	parts := strings.Split(hotkey, "+")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)

		if vk, ok := vkModMap[upper]; ok {
			codes = append(codes, uint32(vk))
			continue
		}
		if vk, ok := vkSpecialMap[upper]; ok {
			codes = append(codes, uint32(vk))
			continue
		}
		slog.Warn("ParseHotkeyString: unknown key, skipping", "key", part)
	}
	if len(codes) == 0 {
		slog.Warn("ParseHotkeyString: no valid keys, falling back to Ctrl+Shift+Escape")
		return []uint32{0x11, 0x10, 0x1B}
	}
	return codes
}

// abortGetAsyncKeyState wraps the Win32 GetAsyncKeyState call.
var abortAsyncKeyProc = user32.NewProc("GetAsyncKeyState")

func abortGetAsyncKeyState(vk uint32) bool {
	ret, _, _ := abortAsyncKeyProc.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}

// StartAbortPoller launches the global hotkey polling goroutine.
// Safe to call multiple times — stops previous poller first.
func StartAbortPoller(keys []uint32, pollMs int) {
	StopAbortPoller()

	if len(keys) == 0 {
		return
	}
	if pollMs < 10 {
		pollMs = 50
	}

	abortMu.Lock()
	abortKeys = keys
	abortPollMs = pollMs
	abortStop = make(chan struct{})
	abortMu.Unlock()

	go abortPollLoop()
	slog.Info("chain abort poller started", "keys", keys, "poll_ms", pollMs)
}

// StopAbortPoller stops the polling goroutine.
func StopAbortPoller() {
	abortMu.Lock()
	defer abortMu.Unlock()
	if abortStop != nil {
		close(abortStop)
		abortStop = nil
	}
}

// GetAbortChannel returns the shared abort channel.
// Creates a new one if nil or already closed.
func GetAbortChannel() <-chan struct{} {
	abortMu.Lock()
	defer abortMu.Unlock()
	if abortCh == nil {
		abortCh = make(chan struct{})
	}
	select {
	case <-abortCh:
		abortCh = make(chan struct{})
	default:
	}
	return abortCh
}

// ResetAbortChannel creates a fresh abort channel.
func ResetAbortChannel() {
	abortMu.Lock()
	defer abortMu.Unlock()
	abortCh = make(chan struct{})
}

// IsChainAborted non-blocking check.
func IsChainAborted() bool {
	abortMu.Lock()
	ch := abortCh
	abortMu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func abortPollLoop() {
	ticker := time.NewTicker(time.Duration(abortPollMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		abortMu.Lock()
		stop := abortStop
		keys := abortKeys
		abortMu.Unlock()

		if stop == nil {
			return
		}

		select {
		case <-ticker.C:
			allDown := true
			for _, vk := range keys {
				if !abortGetAsyncKeyState(vk) {
					allDown = false
					break
				}
			}
			if allDown {
				abortMu.Lock()
				ch := abortCh
				abortMu.Unlock()
				if ch != nil {
					select {
					case <-ch:
					default:
						close(ch)
						slog.Info("chain abort triggered by global hotkey")
					}
				}
				return
			}
		case <-stop:
			return
		}
	}
}

// InitAbortFromConfig starts the abort poller based on config values.
// It is a convenience wrapper that parses the hotkey string and starts the poller.
func InitAbortFromConfig(enabled bool, keys string, pollMs int) {
	if !enabled {
		return
	}
	parsed := ParseHotkeyString(keys)
	StartAbortPoller(parsed, pollMs)
}
