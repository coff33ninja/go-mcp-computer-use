package actions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

const (
	keyEventDown = 0x0000
	keyEventUp   = 0x0002

	inputKeyboard = 1
)

type keyboardInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type inputKbd struct {
	inputType uint32
	ki        keyboardInput
	_         [8]byte
}

type charVK struct {
	vk    uint16
	shift bool
}

var charToVK = map[rune]charVK{
	// Lowercase letters
	'a': {0x41, false}, 'b': {0x42, false}, 'c': {0x43, false},
	'd': {0x44, false}, 'e': {0x45, false}, 'f': {0x46, false},
	'g': {0x47, false}, 'h': {0x48, false}, 'i': {0x49, false},
	'j': {0x4A, false}, 'k': {0x4B, false}, 'l': {0x4C, false},
	'm': {0x4D, false}, 'n': {0x4E, false}, 'o': {0x4F, false},
	'p': {0x50, false}, 'q': {0x51, false}, 'r': {0x52, false},
	's': {0x53, false}, 't': {0x54, false}, 'u': {0x55, false},
	'v': {0x56, false}, 'w': {0x57, false}, 'x': {0x58, false},
	'y': {0x59, false}, 'z': {0x5A, false},
	// Uppercase letters (need Shift)
	'A': {0x41, true}, 'B': {0x42, true}, 'C': {0x43, true},
	'D': {0x44, true}, 'E': {0x45, true}, 'F': {0x46, true},
	'G': {0x47, true}, 'H': {0x48, true}, 'I': {0x49, true},
	'J': {0x4A, true}, 'K': {0x4B, true}, 'L': {0x4C, true},
	'M': {0x4D, true}, 'N': {0x4E, true}, 'O': {0x4F, true},
	'P': {0x50, true}, 'Q': {0x51, true}, 'R': {0x52, true},
	'S': {0x53, true}, 'T': {0x54, true}, 'U': {0x55, true},
	'V': {0x56, true}, 'W': {0x57, true}, 'X': {0x58, true},
	'Y': {0x59, true}, 'Z': {0x5A, true},
	// Digits (no shift) and their shift variants
	'0': {0x30, false}, ')': {0x30, true},
	'1': {0x31, false}, '!': {0x31, true},
	'2': {0x32, false}, '@': {0x32, true},
	'3': {0x33, false}, '#': {0x33, true},
	'4': {0x34, false}, '$': {0x34, true},
	'5': {0x35, false}, '%': {0x35, true},
	'6': {0x36, false}, '^': {0x36, true},
	'7': {0x37, false}, '&': {0x37, true},
	'8': {0x38, false}, '*': {0x38, true},
	'9': {0x39, false}, '(': {0x39, true},
	// OEM keys
	'-': {0xBD, false}, '_': {0xBD, true},
	'=': {0xBB, false}, '+': {0xBB, true},
	'[': {0xDB, false}, '{': {0xDB, true},
	']': {0xDD, false}, '}': {0xDD, true},
	'\\': {0xDC, false}, '|': {0xDC, true},
	';': {0xBA, false}, ':': {0xBA, true},
	'\'': {0xDE, false}, '"': {0xDE, true},
	',': {0xBC, false}, '<': {0xBC, true},
	'.': {0xBE, false}, '>': {0xBE, true},
	'/': {0xBF, false}, '?': {0xBF, true},
	'`': {0xC0, false}, '~': {0xC0, true},
	// Space
	' ': {0x20, false},
}

var vkModMap = map[string]uint16{
	"CTRL":     0x11,
	"CONTROL":  0x11,
	"LCTRL":    0xA2,
	"LCONTROL": 0xA2,
	"RCTRL":    0xA3,
	"RCONTROL": 0xA3,
	"ALT":      0x12,
	"LALT":     0xA4,
	"RALT":     0xA5,
	"SHIFT":    0x10,
	"LSHIFT":   0xA0,
	"RSHIFT":   0xA1,
	"WIN":      0x5B,
	"META":     0x5B,
	"LWIN":     0x5B,
	"SUPER":    0x5B,
	"CMD":      0x5B,
	"COMMAND":  0x5B,
	"RWIN":     0x5C,
}

var vkSpecialMap = map[string]uint16{
	"ENTER":      0x0D,
	"RETURN":     0x0D,
	"BACKSPACE":  0x08,
	"BS":         0x08,
	"TAB":        0x09,
	"ESC":        0x1B,
	"ESCAPE":     0x1B,
	"SPACE":      0x20,
	"DELETE":     0x2E,
	"DEL":        0x2E,
	"INSERT":     0x2D,
	"INS":        0x2D,
	"HOME":       0x24,
	"END":        0x23,
	"PAGEUP":     0x21,
	"PGUP":       0x21,
	"PAGEDOWN":   0x22,
	"PGDN":       0x22,
	"UP":         0x26,
	"DOWN":       0x28,
	"LEFT":       0x25,
	"RIGHT":      0x27,
	"F1":         0x70,
	"F2":         0x71,
	"F3":         0x72,
	"F4":         0x73,
	"F5":         0x74,
	"F6":         0x75,
	"F7":         0x76,
	"F8":         0x77,
	"F9":         0x78,
	"F10":        0x79,
	"F11":        0x7A,
	"F12":        0x7B,
	"CAPSLOCK":   0x14,
	"NUMLOCK":    0x90,
	"SCROLLLOCK": 0x91,
	"PRINTSCREEN": 0x2C,
	"SNAPSHOT":   0x2C,
	"PAUSE":      0x13,
	"MENU":       0x5D,
	"APPS":       0x5D,
	"CONTEXT":    0x5D,
	"CLEAR":      0x0C,
	"EXECUTE":    0x2B,
	"SLEEP":      0x5F,
	// Numpad block
	"NUMPAD0":        0x60,
	"NUMPAD1":        0x61,
	"NUMPAD2":        0x62,
	"NUMPAD3":        0x63,
	"NUMPAD4":        0x64,
	"NUMPAD5":        0x65,
	"NUMPAD6":        0x66,
	"NUMPAD7":        0x67,
	"NUMPAD8":        0x68,
	"NUMPAD9":        0x69,
	"NUMPAD_MULTIPLY":  0x6A,
	"NUMPAD_ADD":       0x6B,
	"NUMPAD_SEPARATOR": 0x6C,
	"NUMPAD_SUBTRACT":  0x6D,
	"NUMPAD_DECIMAL":   0x6E,
	"NUMPAD_DIVIDE":    0x6F,
	// Media / volume
	"VOLUME_MUTE":       0xAD,
	"VOLUME_DOWN":       0xAE,
	"VOLUME_UP":         0xAF,
	"MEDIA_NEXT_TRACK":  0xB0,
	"MEDIA_PREV_TRACK":  0xB1,
	"MEDIA_STOP":        0xB2,
	"MEDIA_PLAY_PAUSE":  0xB3,
}

func sendVK(vk uint16, down bool) {
	var flags uint32 = keyEventDown
	if !down {
		flags = keyEventUp
	}
	i := inputKbd{
		inputType: inputKeyboard,
		ki: keyboardInput{
			wVk:     vk,
			dwFlags: flags,
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
}

const keyPressDelay = 50 * time.Millisecond

func sendVKPress(vk uint16) {
	sendVK(vk, true)
	time.Sleep(keyPressDelay)
	sendVK(vk, false)
}

func sendCharWithVK(r rune) {
	cv, ok := charToVK[r]
	if !ok {
		return
	}
	if cv.shift {
		sendVK(0x10, true)
	}
	sendVKPress(cv.vk)
	if cv.shift {
		sendVK(0x10, false)
	}
}

func keyNameToVK(name string) (uint16, bool) {
	upper := strings.ToUpper(name)
	if vk, ok := vkModMap[upper]; ok {
		return vk, true
	}
	if vk, ok := vkSpecialMap[upper]; ok {
		return vk, true
	}
	if len(name) == 1 {
		ch := name[0]
		if ch >= 'A' && ch <= 'Z' {
			return uint16(ch), true
		}
		if ch >= 'a' && ch <= 'z' {
			return uint16(ch - 32), true
		}
		if ch >= '0' && ch <= '9' {
			return uint16(ch), true
		}
		// Punctuation / single-char keys via the charToVK table
		if cv, ok := charToVK[rune(ch)]; ok {
			return cv.vk, true
		}
	}
	return 0, false
}

func KeyDown(key string) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]string{"key": key})
		LogToolCall("key_down", string(b), err)
		Adaptive.RecordResult("key_down", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("key_down", string(b), err == nil)
	}()
	if err = warnElevated(); err != nil {
		return
	}
	vk, ok := keyNameToVK(key)
	if !ok {
		return fmt.Errorf("key_down: unknown key %q", key)
	}
	sendVK(vk, true)
	return nil
}

func KeyUp(key string) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]string{"key": key})
		LogToolCall("key_up", string(b), err)
		Adaptive.RecordResult("key_up", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("key_up", string(b), err == nil)
	}()
	if err = warnElevated(); err != nil {
		return
	}
	vk, ok := keyNameToVK(key)
	if !ok {
		return fmt.Errorf("key_up: unknown key %q", key)
	}
	sendVK(vk, false)
	return nil
}

func KeyPress(keys []string) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string][]string{"keys": keys})
		LogToolCall("key_press", string(b), err)
		Adaptive.RecordResult("key_press", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("key_press", string(b), err == nil)
	}()
	if err := warnElevated(); err != nil {
		return err
	}
	var pressedMods []uint16
	for _, k := range keys {
		// Normalize to uppercase for map lookups
		ku := strings.ToUpper(k)
		// Check modifier+ prefix (e.g. "CTRL+A", "WIN+R", "ALT+F4")
		// prefix maps any <MOD>+<key> form to a modifier VK.
		if idx := strings.IndexByte(ku, '+'); idx > 0 {
			prefix := ku[:idx]
			if mvk, ok := vkModMap[prefix]; ok {
				rest := k[idx+1:]
				if len(rest) == 1 {
					ruk := strings.ToUpper(rest)
					var vk uint16
					if ruk[0] >= 'A' && ruk[0] <= 'Z' {
						vk = uint16(ruk[0])
					} else if ruk[0] >= '0' && ruk[0] <= '9' {
						vk = uint16(ruk[0])
					} else {
						continue
					}
					sendVK(mvk, true)
					pressedMods = append(pressedMods, mvk)
					sendVKPress(vk)
					continue
				}
			}
		}
		// Modifier keys
		if vk, ok := vkModMap[ku]; ok {
			sendVK(vk, true)
			pressedMods = append(pressedMods, vk)
			continue
		}
		// Special key names (ENTER, BACKSPACE, etc.)
		if vk, ok := vkSpecialMap[ku]; ok {
			sendVKPress(vk)
			continue
		}
		// Single character (letter, digit)
		if vk, ok := keyNameToVK(k); ok {
			sendVKPress(vk)
		}
	}
	for i := len(pressedMods) - 1; i >= 0; i-- {
		sendVK(pressedMods[i], false)
	}
	return nil
}

func TypeText(text string) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]string{"text": text})
		LogToolCall("type", string(b), err)
		Adaptive.RecordResult("type", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("type", string(b), err == nil)
	}()
	if err := warnElevated(); err != nil {
		return err
	}
	for _, r := range text {
		if r == '\n' || r == '\r' {
			sendVKPress(0x0D)
			continue
		}
		if r == '\t' {
			sendVKPress(0x09)
			continue
		}
		if unicode.IsPrint(r) {
			sendCharWithVK(r)
		}
	}
	return nil
}
