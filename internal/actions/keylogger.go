package actions

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	whKeyboardLL   = 13
	whMouseLL      = 14
	wmKeyDown      = 0x0100
	wmKeyUp        = 0x0101
	wmSysKeyDown   = 0x0104
	wmSysKeyUp     = 0x0105
	wmQuit         = 0x0012
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmLButtonDblClk = 0x0203
	wmRButtonDown  = 0x0204
	wmRButtonUp    = 0x0205
	wmRButtonDblClk = 0x0206
	wmMouseWheel   = 0x020A
	wmMouseHwheel  = 0x020E
	llkhfUp        = 0x80
	eventSystemForeground = 0x0003
	wineventOutOfContext  = 0x0000
)

type kbdLLHookStruct struct {
	vkCode      uint32
	scanCode    uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type msLLHookStruct struct {
	ptX         int32
	ptY         int32
	mouseData   uint32
	flags       uint32
	time        uint32
	dwExtraInfo uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
}

type recordedEvent struct {
	kind         string
	keyName      string
	button       string
	x, y         int32
	startX, startY int32
	down         bool
	clicks       int
	scrollDelta  int32
	timestamp    time.Time
}

var (
	klMu             sync.Mutex
	klActive         bool
	klEvents         []recordedEvent
	klStartTime      time.Time
	klLastEventTime  time.Time
	klKbdHook        windows.Handle
	klMouseHook      windows.Handle
	klWinEventHandle windows.Handle
	klThreadID       uint32
	klDone           chan struct{}
	klDownKeys       map[uint32]bool
	klStartWindow    string
	klEndWindow      string
	klMouseDown      struct {
		left  bool
		right bool
		startTime time.Time
		startX, startY int32
	}
	klLastMove       struct {
		x, y int32
		time time.Time
	}
	klLogMouseMoves  bool
)

var (
	klUser32        = windows.NewLazySystemDLL("user32.dll")
	klSetHook       = klUser32.NewProc("SetWindowsHookExW")
	klUnhook        = klUser32.NewProc("UnhookWindowsHookEx")
	klCallNext      = klUser32.NewProc("CallNextHookEx")
	klGetMsg        = klUser32.NewProc("GetMessageW")
	klPostThread    = klUser32.NewProc("PostThreadMessageW")
	klWinEventHook  = klUser32.NewProc("SetWinEventHook")
	klUnhookEvent   = klUser32.NewProc("UnhookWinEvent")
	klGetForeground = klUser32.NewProc("GetForegroundWindow")
	klGetWinText    = klUser32.NewProc("GetWindowTextW")
	klGetWinTextLen = klUser32.NewProc("GetWindowTextLengthW")
)

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

func kbdHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == 0 {
		kbd := (*kbdLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))
		isUp := (kbd.flags & llkhfUp) != 0
		down := (wParam == wmKeyDown || wParam == wmSysKeyDown) && !isUp

		if down || isUp || wParam == wmKeyUp || wParam == wmSysKeyUp {
			klMu.Lock()
			alreadyDown := klDownKeys[kbd.vkCode]
			if down && !alreadyDown {
				klDownKeys[kbd.vkCode] = true
				klMu.Unlock()
				name, ok := vkDecode(kbd.vkCode)
				if !ok {
					name = fmt.Sprintf("VK_0x%02X", kbd.vkCode)
				}
				klRecord(recordedEvent{kind: "key", keyName: name, down: true})
			} else if !down && alreadyDown {
				delete(klDownKeys, kbd.vkCode)
				klMu.Unlock()
				name, ok := vkDecode(kbd.vkCode)
				if !ok {
					name = fmt.Sprintf("VK_0x%02X", kbd.vkCode)
				}
				klRecord(recordedEvent{kind: "key", keyName: name, down: false})
			} else {
				klMu.Unlock()
			}
		}
	}
	ret, _, _ := klCallNext.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode == 0 {
		ms := (*msLLHookStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&lParam)))

		switch wParam {
		case wmMouseMove:
			if klLogMouseMoves {
				klMu.Lock()
				dx := ms.ptX - klLastMove.x
				dy := ms.ptY - klLastMove.y
				since := time.Since(klLastMove.time)
				sigMove := dx*dx+dy*dy > 100
				klMu.Unlock()
				if sigMove && since > 40*time.Millisecond {
					klMu.Lock()
					klLastMove.x = ms.ptX
					klLastMove.y = ms.ptY
					klLastMove.time = time.Now()
					klMu.Unlock()
					klRecord(recordedEvent{kind: "mouse_move", x: ms.ptX, y: ms.ptY})
				}
			}

		case wmLButtonDown, wmLButtonDblClk:
			klMu.Lock()
			klMouseDown.left = true
			klMouseDown.startTime = time.Now()
			klMouseDown.startX = ms.ptX
			klMouseDown.startY = ms.ptY
			klMu.Unlock()

		case wmLButtonUp:
			klMu.Lock()
			if klMouseDown.left {
				elapsed := time.Since(klMouseDown.startTime)
				dx := ms.ptX - klMouseDown.startX
				dy := ms.ptY - klMouseDown.startY
				dist := dx*dx + dy*dy
				klMouseDown.left = false
				klMu.Unlock()

				if dist > 100 || elapsed > 300*time.Millisecond {
					klRecord(recordedEvent{kind: "drag", button: "left",
						startX: klMouseDown.startX, startY: klMouseDown.startY,
						x: ms.ptX, y: ms.ptY})
				} else {
					clicks := 1
					if wParam == wmLButtonDblClk {
						clicks = 2
					}
					klRecord(recordedEvent{kind: "click", button: "left", x: ms.ptX, y: ms.ptY, clicks: clicks})
				}
			} else {
				klMu.Unlock()
			}

		case wmRButtonDown, wmRButtonDblClk:
			klMu.Lock()
			klMouseDown.right = true
			klMouseDown.startTime = time.Now()
			klMouseDown.startX = ms.ptX
			klMouseDown.startY = ms.ptY
			klMu.Unlock()

		case wmRButtonUp:
			klMu.Lock()
			if klMouseDown.right {
				klMouseDown.right = false
				klMu.Unlock()
				klRecord(recordedEvent{kind: "click", button: "right", x: ms.ptX, y: ms.ptY, clicks: 1})
			} else {
				klMu.Unlock()
			}

		case wmMouseWheel:
			delta := int32(int16(ms.mouseData >> 16))
			clicks := delta / 120
			if clicks != 0 {
				klRecord(recordedEvent{kind: "scroll", scrollDelta: clicks})
			}
		}
	}
	ret, _, _ := klCallNext.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

func winEventHookProc(hWinEventHook uintptr, event uint32, hwnd uintptr, idObject int32, idChild int32, dwEventThread uint32, dwmsEventTime uint32) uintptr {
	if event == eventSystemForeground {
		title := klGetForegroundTitle()
		if title != "" {
			klRecord(recordedEvent{kind: "focus", keyName: title})
		}
	}
	return 0
}

var klKbdCallback uintptr
var klMouseCallback uintptr
var klWinEventCallback uintptr

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

	errCh := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer func() {
			klMu.Lock()
			klActive = false
			klMu.Unlock()
			close(klDone)
		}()

		klKbdCallback = syscall.NewCallback(kbdHookProc)
		klMouseCallback = syscall.NewCallback(mouseHookProc)
		klWinEventCallback = syscall.NewCallback(winEventHookProc)

		kbdHook, _, _ := klSetHook.Call(whKeyboardLL, klKbdCallback, 0, 0)
		if kbdHook == 0 {
			errCh <- fmt.Errorf("SetWindowsHookEx(WH_KEYBOARD_LL) failed")
			return
		}

		mouseHook, _, _ := klSetHook.Call(whMouseLL, klMouseCallback, 0, 0)
		if mouseHook == 0 {
			klUnhook.Call(kbdHook)
			errCh <- fmt.Errorf("SetWindowsHookEx(WH_MOUSE_LL) failed")
			return
		}

		winEventHook, _, _ := klWinEventHook.Call(
			eventSystemForeground, eventSystemForeground,
			0, klWinEventCallback, 0, 0, wineventOutOfContext)
		if winEventHook == 0 {
			klUnhook.Call(kbdHook)
			klUnhook.Call(mouseHook)
			errCh <- fmt.Errorf("SetWinEventHook failed")
			return
		}

		klMu.Lock()
		klKbdHook = windows.Handle(kbdHook)
		klMouseHook = windows.Handle(mouseHook)
		klWinEventHandle = windows.Handle(winEventHook)
		klThreadID = windows.GetCurrentThreadId()
		klMu.Unlock()

		errCh <- nil

		var m msg
		for {
			ret, _, _ := klGetMsg.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if ret == 0 {
				break
			}
		}
	}()

	return <-errCh
}

func StopKeylogger() ([]map[string]any, map[string]any, error) {
	klMu.Lock()
	if !klActive {
		klMu.Unlock()
		return nil, nil, fmt.Errorf("keylogger not active")
	}
	tid := klThreadID
	klMu.Unlock()

	klEndWindow = klGetForegroundTitle()

	klPostThread.Call(uintptr(tid), wmQuit, 0, 0)
	<-klDone

	klMu.Lock()
	klUnhook.Call(uintptr(klKbdHook))
	klUnhook.Call(uintptr(klMouseHook))
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

	_ = baseTime
	return steps
}
