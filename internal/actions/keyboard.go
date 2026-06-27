package actions

import (
	"strings"
	"unsafe"
)

const (
	keyEventDown = 0x0000
	keyEventUp   = 0x0002
	keyEventUnicode = 0x0004

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

// Unicode control-character mappings for keys that can be sent via KEYEVENTF_UNICODE
var unicodeKeyMap = map[string]rune{
	"ENTER":     0x0D,
	"BACKSPACE": 0x08,
	"TAB":       0x09,
	"ESC":      0x1B,
	"ESCAPE":   0x1B,
	"SPACE":    0x20,
}

// VK-coded modifier keys (no Unicode equivalent)
var vkModMap = map[string]uint16{
	"CTRL":    0x11,
	"CONTROL": 0x11,
	"ALT":     0x12,
	"SHIFT":   0x10,
}

// VK-coded non-character keys (no Unicode equivalent)
var vkSpecialMap = map[string]uint16{
	"DELETE":   0x2E,
	"DEL":      0x2E,
	"INSERT":   0x2D,
	"HOME":     0x24,
	"END":      0x23,
	"PAGEUP":   0x21,
	"PAGEDOWN": 0x22,
	"UP":       0x26,
	"DOWN":     0x28,
	"LEFT":     0x25,
	"RIGHT":    0x27,
	"F1":       0x70,
	"F2":       0x71,
	"F3":       0x72,
	"F4":       0x73,
	"F5":       0x74,
	"F6":       0x75,
	"F7":       0x76,
	"F8":       0x77,
	"F9":       0x78,
	"F10":      0x79,
	"F11":      0x7A,
	"F12":      0x7B,
}

// send a Unicode character via KEYEVENTF_UNICODE
func sendUnicode(r rune) {
	i := inputKbd{
		inputType: inputKeyboard,
		ki: keyboardInput{
			wScan:   uint16(r),
			dwFlags: keyEventUnicode,
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
	i.ki.dwFlags = keyEventUnicode | keyEventUp
	sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
}

// send a VK-coded key (for non-printable keys without Unicode equivalents)
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

func keyNameToUnicode(name string) (rune, bool) {
	// Check unicodeKeyMap first (ENTER, BACKSPACE, TAB, ESC, SPACE)
	if r, ok := unicodeKeyMap[name]; ok {
		return r, true
	}
	// Ctrl+letter combos → control characters 0x01-0x1A
	if strings.HasPrefix(name, "CTRL+") || strings.HasPrefix(name, "CONTROL+") {
		parts := strings.SplitN(name, "+", 2)
		if len(parts) == 2 && len(parts[1]) == 1 {
			ch := parts[1][0]
			if ch >= 'A' && ch <= 'Z' {
				return rune(ch - 'A' + 1), true
			}
		}
	}
	// Single letter or digit → its Unicode value
	if len(name) == 1 {
		ch := name[0]
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == ' ' {
			return rune(ch), true
		}
	}
	return 0, false
}

func KeyPress(keys []string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	var unicodeKeys []rune
	var vkDown []uint16
	var vkUp []uint16
	for _, k := range keys {
		if r, ok := keyNameToUnicode(k); ok {
			unicodeKeys = append(unicodeKeys, r)
		} else if vk, ok := vkModMap[k]; ok {
			vkDown = append(vkDown, vk)
			vkUp = append([]uint16{vk}, vkUp...)
		} else if vk, ok := vkSpecialMap[k]; ok {
			vkDown = append(vkDown, vk)
			vkUp = append([]uint16{vk}, vkUp...)
		}
	}
	// Send all keys: Unicode chars first, then VK down, then VK up in reverse
	for _, r := range unicodeKeys {
		sendUnicode(r)
	}
	for _, vk := range vkDown {
		sendVK(vk, true)
	}
	for _, vk := range vkUp {
		sendVK(vk, false)
	}
	return nil
}

func TypeText(text string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	for _, r := range text {
		sendUnicode(r)
	}
	return nil
}
