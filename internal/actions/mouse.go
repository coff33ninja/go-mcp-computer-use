package actions

import (
	"encoding/json"
	"math"
	"syscall"
	"time"
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

func Click(args ClickInput) (err error) {
	if args.Button == "" {
		args.Button = "left"
	}
	if args.Clicks == 0 {
		args.Clicks = 1
	}

	start := time.Now()
	defer func() {
		b, _ := json.Marshal(args)
		LogToolCall("click", string(b), err)
		Adaptive.RecordResult("click", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("click", string(b), err == nil)
	}()

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

func MoveMouse(x, y int32) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]int32{"x": x, "y": y})
		LogToolCall("move_mouse", string(b), err)
		Adaptive.RecordResult("move_mouse", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("move_mouse", string(b), err == nil)
	}()
	if err = ValidateClickCoord(x, y); err != nil {
		return
	}
	ret, _, _ := setCursorPos.Call(uintptr(x), uintptr(y))
	if ret == 0 {
		err = syscall.GetLastError()
	}
	return
}

func GetCursorPosition() (int32, int32, error) {
	var pt point
	ret, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, syscall.GetLastError()
	}
	return pt.X, pt.Y, nil
}

func Scroll(clicks int32) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]int32{"clicks": clicks})
		LogToolCall("scroll", string(b), err)
		Adaptive.RecordResult("scroll", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("scroll", string(b), err == nil)
	}()
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

func Drag(fromX, fromY, toX, toY int32) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]int32{"from_x": fromX, "from_y": fromY, "to_x": toX, "to_y": toY})
		LogToolCall("drag", string(b), err)
		Adaptive.RecordResult("drag", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("drag", string(b), err == nil)
	}()
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

	down := input{
		inputType: inputMouse,
		mi: mouseInput{
			dx:      fx,
			dy:      fy,
			dwFlags: mouseEventLeftDown | mouseEventAbsolute | mouseEventMove,
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&down)), unsafe.Sizeof(down))

	dx := tx - fx
	dy := ty - fy
	dist := int32(math.Sqrt(float64(dx*dx + dy*dy)))
	steps := dist / 400
	if steps < 5 {
		steps = 5
	}
	if steps > 50 {
		steps = 50
	}

	for i := int32(1); i <= steps; i++ {
		px := fx + dx*i/steps
		py := fy + dy*i/steps
		move := input{
			inputType: inputMouse,
			mi: mouseInput{
				dx:      px,
				dy:      py,
				dwFlags: mouseEventMove | mouseEventAbsolute,
			},
		}
		sendInput.Call(1, uintptr(unsafe.Pointer(&move)), unsafe.Sizeof(move))
		time.Sleep(5 * time.Millisecond)
	}

	up := input{
		inputType: inputMouse,
		mi: mouseInput{
			dx:      tx,
			dy:      ty,
			dwFlags: mouseEventLeftUp | mouseEventAbsolute | mouseEventMove,
		},
	}
	sendInput.Call(1, uintptr(unsafe.Pointer(&up)), unsafe.Sizeof(up))

	return nil
}
