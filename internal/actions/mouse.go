package actions

import (
	"syscall"
	"unsafe"
)

const (
	inputMouse = 0

	mouseEventMove      = 0x0001
	mouseEventLeftDown  = 0x0002
	mouseEventLeftUp    = 0x0004
	mouseEventRightDown = 0x0008
	mouseEventRightUp   = 0x0010
	mouseEventWheel     = 0x0800
	mouseEventAbsolute  = 0x8000
)

type mouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type input struct {
	inputType uint32
	mi        mouseInput
}

type point struct {
	X, Y int32
}

type ClickInput struct {
	X, Y   int32
	Button string
	Clicks int
}

func Click(args ClickInput) error {
	if args.Button == "" {
		args.Button = "left"
	}
	if args.Clicks == 0 {
		args.Clicks = 1
	}

	if err := ValidateClickCoord(args.X, args.Y); err != nil {
		return err
	}

	setCursorPos.Call(uintptr(args.X), uintptr(args.Y))

	var downFlag, upFlag uint32
	switch args.Button {
	case "right":
		downFlag = mouseEventRightDown
		upFlag = mouseEventRightUp
	default:
		downFlag = mouseEventLeftDown
		upFlag = mouseEventLeftUp
	}

	in := func(flags uint32) {
		i := input{
			inputType: inputMouse,
			mi: mouseInput{dwFlags: flags},
		}
		sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
	}

	for range args.Clicks {
		in(downFlag)
		in(upFlag)
	}

	return nil
}

func MoveMouse(x, y int32) error {
	if err := ValidateClickCoord(x, y); err != nil {
		return err
	}
	ret, _, _ := setCursorPos.Call(uintptr(x), uintptr(y))
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func GetCursorPosition() (int32, int32, error) {
	var pt point
	ret, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, syscall.GetLastError()
	}
	return pt.X, pt.Y, nil
}

func Scroll(clicks int32) error {
	i := input{
		inputType: inputMouse,
		mi: mouseInput{
			dwFlags:   mouseEventWheel,
			mouseData: uint32(clicks * 120),
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
	return nil
}

func sendMouseEvent(flags uint32, dx, dy int32) {
	i := input{
		inputType: inputMouse,
		mi: mouseInput{
			dx:      dx,
			dy:      dy,
			dwFlags: flags,
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&i)), unsafe.Sizeof(i))
}

func Drag(fromX, fromY, toX, toY int32) error {
	if err := ValidateClickCoord(fromX, fromY); err != nil {
		return err
	}
	if err := ValidateClickCoord(toX, toY); err != nil {
		return err
	}

	sw, sh := ScreenSize()
	norm := func(x, y int32) (int32, int32) {
		return int32((int(x) * 65535) / int(sw-1)), int32((int(y) * 65535) / int(sh-1))
	}

	fx, fy := norm(fromX, fromY)
	tx, ty := norm(toX, toY)

	// 1. Move to start position (absolute)
	sendMouseEvent(mouseEventMove|mouseEventAbsolute, fx, fy)
	// 2. Press left button (relative — at current position)
	sendMouseEvent(mouseEventLeftDown, 0, 0)
	// 3. Move to end position while holding (absolute)
	sendMouseEvent(mouseEventMove|mouseEventAbsolute, tx, ty)
	// 4. Release left button (relative — at current position)
	sendMouseEvent(mouseEventLeftUp, 0, 0)

	return nil
}
