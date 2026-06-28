package actions

import (
	"strings"
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
	"CTRL":    0x11,
	"CONTROL": 0x11,
	"ALT":     0x12,
	"SHIFT":   0x10,
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
	"PAUSE":      0x13,
	"MENU":       0x5D,
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

func sendCharWithVK(r rune) {
	cv, ok := charToVK[r]
	if !ok {
		return
	}
	if cv.shift {
		sendVK(0x10, true)
	}
	sendVK(cv.vk, true)
	sendVK(cv.vk, false)
	if cv.shift {
		sendVK(0x10, false)
	}
}

func keyNameToVK(name string) (uint16, bool) {
	if vk, ok := vkSpecialMap[name]; ok {
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
	}
	return 0, false
}

func KeyPress(keys []string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	var pressedMods []uint16
	for _, k := range keys {
		// Check CTRL+/CONTROL+ prefix (e.g. "CTRL+A")
		if strings.HasPrefix(k, "CTRL+") || strings.HasPrefix(k, "CONTROL+") {
			parts := strings.SplitN(k, "+", 2)
			if len(parts) == 2 && len(parts[1]) == 1 {
				ch := parts[1][0]
				var vk uint16
				if ch >= 'A' && ch <= 'Z' {
					vk = uint16(ch)
				} else if ch >= 'a' && ch <= 'z' {
					vk = uint16(ch - 32)
				} else {
					continue
				}
				sendVK(0x11, true)
				pressedMods = append(pressedMods, 0x11)
				sendVK(vk, true)
				sendVK(vk, false)
				continue
			}
		}
		// Modifier keys
		if vk, ok := vkModMap[k]; ok {
			sendVK(vk, true)
			pressedMods = append(pressedMods, vk)
			continue
		}
		// Special key names (ENTER, BACKSPACE, etc.)
		if vk, ok := vkSpecialMap[k]; ok {
			sendVK(vk, true)
			sendVK(vk, false)
			continue
		}
		// Single character (letter, digit)
		if vk, ok := keyNameToVK(k); ok {
			sendVK(vk, true)
			sendVK(vk, false)
		}
	}
	for i := len(pressedMods) - 1; i >= 0; i-- {
		sendVK(pressedMods[i], false)
	}
	return nil
}

func TypeText(text string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	for _, r := range text {
		if r == '\n' || r == '\r' {
			sendVK(0x0D, true)
			sendVK(0x0D, false)
			continue
		}
		if r == '\t' {
			sendVK(0x09, true)
			sendVK(0x09, false)
			continue
		}
		if unicode.IsPrint(r) {
			sendCharWithVK(r)
		}
	}
	return nil
}
