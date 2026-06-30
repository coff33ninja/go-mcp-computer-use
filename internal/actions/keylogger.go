package actions

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	eventSystemForeground = 0x0003
	wineventOutOfContext  = 0x0000
	pollInterval          = 50 * time.Millisecond
	vkLButton             = 0x01
	vkRButton             = 0x02
	vkMButton             = 0x04
	vkXButton1            = 0x05
	vkXButton2            = 0x06
)

type recordedEvent struct {
	kind             string
	keyName          string
	button           string
	x, y             int32
	startX, startY   int32
	down             bool
	clicks           int
	scrollDelta      int32
	timestamp        time.Time
}

type winEventHookProc func(hWinEventHook uintptr, event uint32, hwnd uintptr, idObject int32, idChild int32, dwEventThread uint32, dwmsEventTime uint32) uintptr

var (
	klMu              sync.Mutex
	klActive          bool
	klEvents          []recordedEvent
	klStartTime       time.Time
	klLastEventTime   time.Time
	klDone            chan struct{}
	klDownKeys        map[uint32]bool
	klStartWindow     string
	klEndWindow       string
	klMouseDown       struct {
		left, right       bool
		startTime         time.Time
		startX, startY    int32
	}
	klLastMove        struct {
		x, y int32
		time time.Time
	}
	klLogMouseMoves   bool
	klWinEventHandle  windows.Handle
	klEventCallback   uintptr
)

var vkSpecialRev map[uint32]string
var vkModRev map[uint32]string

func init() {
	vkSpecialRev = make(map[uint32]string, len(vkSpecialMap))
	for name, vk := range vkSpecialMap {
		if _, exists := vkSpecialRev[uint32(vk)]; !exists {
			vkSpecialRev[uint32(vk)] = name
		}
	}
	vkModRev = make(map[uint32]string, len(vkModMap))
	for name, vk := range vkModMap {
		if _, exists := vkModRev[uint32(vk)]; !exists {
			vkModRev[uint32(vk)] = name
		}
	}
}

func vkDecode(vk uint32) (string, bool) {
	if name, ok := vkSpecialRev[vk]; ok {
		return name, true
	}
	if name, ok := vkModRev[vk]; ok {
		return name, true
	}
	if vk >= 0x30 && vk <= 0x39 {
		return string(rune('0' + vk - 0x30)), true
	}
	if vk >= 0x41 && vk <= 0x5A {
		return string(rune('A' + vk - 0x41)), true
	}
	return "", false
}

var (
	klUser32         = windows.NewLazySystemDLL("user32.dll")
	klGetAsyncKey    = klUser32.NewProc("GetAsyncKeyState")
	klGetCursorPos   = klUser32.NewProc("GetCursorPos")
	klGetForeground  = klUser32.NewProc("GetForegroundWindow")
	klGetWinText     = klUser32.NewProc("GetWindowTextW")
	klGetWinTextLen  = klUser32.NewProc("GetWindowTextLengthW")
	klWinEventHook   = klUser32.NewProc("SetWinEventHook")
	klUnhookEvent    = klUser32.NewProc("UnhookWinEvent")
)

func klGetState(vk int) bool {
	ret, _, _ := klGetAsyncKey.Call(uintptr(vk))
	return (ret & 0x8000) != 0
}

func klGetMousePos() (int32, int32) {
	var pt point
	klGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt.X, pt.Y
}

func klGetForegroundTitle() string {
	hwnd, _, _ := klGetForeground.Call()
	if hwnd == 0 {
		return ""
	}
	len, _, _ := klGetWinTextLen.Call(hwnd)
	if len == 0 {
		return ""
	}
	buf := make([]uint16, len+1)
	klGetWinText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len+1))
	return windows.UTF16ToString(buf)
}

func klRecord(ev recordedEvent) {
	klMu.Lock()
	defer klMu.Unlock()
	if !klActive {
		return
	}
	ev.timestamp = time.Now()
	klEvents = append(klEvents, ev)
	klLastEventTime = ev.timestamp
}

func StartKeylogger() error {
	klMu.Lock()
	defer klMu.Unlock()

	if klActive {
		return fmt.Errorf("keylogger already active")
	}

	klEvents = nil
	klDownKeys = make(map[uint32]bool)
	klActive = true
	klLogMouseMoves = true
	klDone = make(chan struct{})
	klStartTime = time.Now()
	klLastEventTime = time.Now()
	klStartWindow = klGetForegroundTitle()

	klEventCallback = syscall.NewCallback(func(hWinEventHook uintptr, event uint32, hwnd uintptr, idObject int32, idChild int32, dwEventThread uint32, dwmsEventTime uint32) uintptr {
		if event == eventSystemForeground {
			title := klGetForegroundTitle()
			if title != "" {
				klRecord(recordedEvent{kind: "focus", keyName: title})
			}
		}
		return 0
	})

	winEventHook, _, _ := klWinEventHook.Call(
		eventSystemForeground, eventSystemForeground,
		0, klEventCallback, 0, 0, wineventOutOfContext)
	if winEventHook == 0 {
		klActive = false
		return fmt.Errorf("SetWinEventHook failed")
	}
	klWinEventHandle = windows.Handle(winEventHook)

	go pollLoop()

	return nil
}

func pollLoop() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var prevKeyState [256]bool
	mouseStartTime := time.Time{}
	mouseStartX, mouseStartY := int32(0), int32(0)
	mouseDownButtons := [6]bool{}
	prevX, prevY := klGetMousePos()
	prevFocus := klStartWindow

	for i := 0; i < 256; i++ {
		prevKeyState[i] = klGetState(i)
		if prevKeyState[i] {
			klDownKeys[uint32(i)] = true
		}
	}

	for {
		select {
		case <-ticker.C:
			klPollKeys(&prevKeyState)
			klPollMouse(&prevX, &prevY, &mouseStartTime, &mouseStartX, &mouseStartY, &mouseDownButtons, &prevFocus)
		case <-klDone:
			return
		}
	}
}

func klPollKeys(prev *[256]bool) {
	for vk := 0; vk < 256; vk++ {
		down := klGetState(vk)
		wasDown := prev[vk]
		if down && !wasDown {
			name, ok := vkDecode(uint32(vk))
			if !ok {
				name = fmt.Sprintf("VK_0x%02X", vk)
			}
			klRecord(recordedEvent{kind: "key", keyName: name, down: true})
			klMu.Lock()
			klDownKeys[uint32(vk)] = true
			klMu.Unlock()
		} else if !down && wasDown {
			name, ok := vkDecode(uint32(vk))
			if !ok {
				name = fmt.Sprintf("VK_0x%02X", vk)
			}
			klRecord(recordedEvent{kind: "key", keyName: name, down: false})
			klMu.Lock()
			delete(klDownKeys, uint32(vk))
			klMu.Unlock()
		}
		prev[vk] = down
	}
}

func klPollMouse(prevX, prevY *int32, startTime *time.Time, startX, startY *int32, mouseDown *[6]bool, prevFocus *string) {
	x, y := klGetMousePos()

	if x != *prevX || y != *prevY {
		if klLogMouseMoves {
			dx := x - *prevX
			dy := y - *prevY
			since := time.Since(klLastMove.time)
			if (dx*dx+dy*dy > 25) && since > 100*time.Millisecond {
				klRecord(recordedEvent{kind: "mouse_move", x: x, y: y})
				klMu.Lock()
				klLastMove.x = x
				klLastMove.y = y
				klLastMove.time = time.Now()
				klMu.Unlock()
			}
		}
		*prevX = x
		*prevY = y
	}

	buttonDown := []bool{
		klGetState(vkLButton),
		klGetState(vkRButton),
		klGetState(vkMButton),
		klGetState(vkXButton1),
		klGetState(vkXButton2),
	}
	buttonNames := []string{"left", "right", "middle", "xbutton1", "xbutton2"}

	for i := 0; i < 5; i++ {
		if buttonDown[i] && !mouseDown[i] {
			mouseDown[i] = true
			*startTime = time.Now()
			*startX = x
			*startY = y
		} else if !buttonDown[i] && mouseDown[i] {
			mouseDown[i] = false
			elapsed := time.Since(*startTime)
			dx := x - *startX
			dy := y - *startY
			dist := dx*dx + dy*dy

			if i == 0 || i == 1 {
				if dist > 100 || elapsed > 300*time.Millisecond {
					klRecord(recordedEvent{kind: "drag", button: buttonNames[i],
						startX: *startX, startY: *startY, x: x, y: y})
				} else {
					klRecord(recordedEvent{kind: "click", button: buttonNames[i], x: x, y: y, clicks: 1})
				}
			} else {
				klRecord(recordedEvent{kind: "click", button: buttonNames[i], x: x, y: y, clicks: 1})
			}
		}
	}

	focus := klGetForegroundTitle()
	if focus != "" && focus != *prevFocus {
		klRecord(recordedEvent{kind: "focus", keyName: focus})
		*prevFocus = focus
	}
}

func StopKeylogger() ([]map[string]any, map[string]any, error) {
	klMu.Lock()
	if !klActive {
		klMu.Unlock()
		return nil, nil, fmt.Errorf("keylogger not active")
	}
	klMu.Unlock()

	klEndWindow = klGetForegroundTitle()

	close(klDone)

	klMu.Lock()
	klActive = false
	klUnhookEvent.Call(uintptr(klWinEventHandle))
	events := klEvents
	startWin := klStartWindow
	endWin := klEndWindow
	startTime := klStartTime
	klMu.Unlock()

	steps := klEventsToSteps(events)

	meta := map[string]any{
		"window_start":      startWin,
		"window_end":        endWin,
		"duration_seconds":  int(time.Since(startTime).Seconds()),
		"total_events":      len(events),
	}

	return steps, meta, nil
}

func KeyloggerStatus() (bool, int, string) {
	klMu.Lock()
	defer klMu.Unlock()
	if !klActive {
		return false, 0, ""
	}
	dur := time.Since(klStartTime).Round(time.Second).String()
	return true, len(klEvents), dur
}

func klEventsToSteps(events []recordedEvent) []map[string]any {
	if len(events) == 0 {
		return nil
	}

	var steps []map[string]any
	baseTime := events[0].timestamp

	for _, ev := range events {
		delay := ev.timestamp.Sub(baseTime)
		if delay > 30*time.Millisecond {
			steps = append(steps, map[string]any{
				"tool": "wait",
				"args": map[string]any{"ms": int(delay.Milliseconds())},
			})
		}

		switch ev.kind {
		case "key":
			if ev.down {
				steps = append(steps, map[string]any{
					"tool": "key_down",
					"args": map[string]any{"key": ev.keyName},
				})
			} else {
				steps = append(steps, map[string]any{
					"tool": "key_up",
					"args": map[string]any{"key": ev.keyName},
				})
			}
		case "click":
			args := map[string]any{"x": ev.x, "y": ev.y}
			if ev.button == "right" {
				args["button"] = "right"
			}
			if ev.clicks > 1 {
				args["clicks"] = ev.clicks
			}
			steps = append(steps, map[string]any{
				"tool": "click",
				"args": args,
			})
		case "drag":
			steps = append(steps, map[string]any{
				"tool": "drag",
				"args": map[string]any{"from_x": ev.startX, "from_y": ev.startY, "to_x": ev.x, "to_y": ev.y},
			})
		case "mouse_move":
			steps = append(steps, map[string]any{
				"tool": "move_mouse",
				"args": map[string]any{"x": ev.x, "y": ev.y},
			})
		case "scroll":
			steps = append(steps, map[string]any{
				"tool": "scroll",
				"args": map[string]any{"clicks": ev.scrollDelta},
			})
		case "focus":
			steps = append(steps, map[string]any{
				"tool": "_focus",
				"args": map[string]any{"window": ev.keyName},
			})
		}

		baseTime = ev.timestamp
	}

	return steps
}
