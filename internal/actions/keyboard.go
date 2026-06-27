package actions

import "unsafe"

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

var vkMap = map[string]uint16{
	"CTRL":     0x11,
	"CONTROL":  0x11,
	"ALT":      0x12,
	"SHIFT":    0x10,
	"TAB":      0x09,
	"ENTER":    0x0D,
	"ESC":      0x1B,
	"ESCAPE":   0x1B,
	"BACKSPACE": 0x08,
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
	"SPACE":    0x20,
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
	"A":        0x41,
	"B":        0x42,
	"C":        0x43,
	"D":        0x44,
	"E":        0x45,
	"F":        0x46,
	"G":        0x47,
	"H":        0x48,
	"I":        0x49,
	"J":        0x4A,
	"K":        0x4B,
	"L":        0x4C,
	"M":        0x4D,
	"N":        0x4E,
	"O":        0x4F,
	"P":        0x50,
	"Q":        0x51,
	"R":        0x52,
	"S":        0x53,
	"T":        0x54,
	"U":        0x55,
	"V":        0x56,
	"W":        0x57,
	"X":        0x58,
	"Y":        0x59,
	"Z":        0x5A,
	"0":        0x30,
	"1":        0x31,
	"2":        0x32,
	"3":        0x33,
	"4":        0x34,
	"5":        0x35,
	"6":        0x36,
	"7":        0x37,
	"8":        0x38,
	"9":        0x39,
}

func sendKey(vk uint16, down bool) {
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

func KeyPress(keys []string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	for _, k := range keys {
		vk, ok := vkMap[k]
		if !ok {
			continue
		}
		sendKey(vk, true)
	}

	for i := len(keys) - 1; i >= 0; i-- {
		vk, ok := vkMap[keys[i]]
		if !ok {
			continue
		}
		sendKey(vk, false)
	}

	return nil
}

func TypeText(text string) error {
	if err := warnElevated(); err != nil {
		return err
	}
	for _, r := range text {
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
	return nil
}
