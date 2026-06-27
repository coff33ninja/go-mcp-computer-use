package actions

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"syscall"
	"unsafe"
)

var (
	gdi32 = syscall.NewLazyDLL("gdi32.dll")

	createCompatibleDC    = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	selectObject          = gdi32.NewProc("SelectObject")
	bitBlt                = gdi32.NewProc("BitBlt")
	getDIBits             = gdi32.NewProc("GetDIBits")
	deleteDC              = gdi32.NewProc("DeleteDC")
	deleteObject          = gdi32.NewProc("DeleteObject")
)

const (
	SRCCOPY       = 0x00CC0020
	DIB_RGB_COLORS = 0
)

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
}

func CaptureScreen() (string, error) {
	w, h := ScreenSize()
	return CaptureRegion(0, 0, w, h)
}

func CaptureRegion(x, y, w, h int32) (string, error) {
	if err := ValidateRegion(x, y, w, h); err != nil {
		return "", err
	}
	hdc := GetDesktopDC()
	if hdc == 0 {
		return "", syscall.GetLastError()
	}
	defer ReleaseDesktopDC(hdc)

	memDC, _, _ := createCompatibleDC.Call(hdc)
	if memDC == 0 {
		return "", syscall.GetLastError()
	}
	defer deleteDC.Call(memDC)

	hbitmap, _, _ := createCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	if hbitmap == 0 {
		return "", syscall.GetLastError()
	}
	defer deleteObject.Call(hbitmap)

	selectObject.Call(memDC, hbitmap)
	bitBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h), hdc, uintptr(x), uintptr(y), SRCCOPY)

	header := BITMAPINFOHEADER{
		Size:     uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
		Width:    w,
		Height:   h,
		Planes:   1,
		BitCount: 32,
	}

	pixels := make([]byte, w*h*4)
	ret, _, _ := getDIBits.Call(hdc, hbitmap, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&BITMAPINFO{Header: header})),
		DIB_RGB_COLORS)
	if ret == 0 {
		return "", syscall.GetLastError()
	}

	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
	rowSize := int(w) * 4
	for row := 0; row < int(h); row++ {
		srcRow := (int(h) - 1 - row) * rowSize
		dstRow := row * img.Stride
		for col := 0; col < rowSize; col += 4 {
			img.Pix[dstRow+col+0] = pixels[srcRow+col+2]
			img.Pix[dstRow+col+1] = pixels[srcRow+col+1]
			img.Pix[dstRow+col+2] = pixels[srcRow+col+0]
			img.Pix[dstRow+col+3] = 255
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
