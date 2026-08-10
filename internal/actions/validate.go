package actions

import (
	"fmt"

	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
)

var ActiveConfig *config.Config

type Rect struct {
	X int32
	Y int32
	W int32
	H int32
}

func ValidateClickCoord(x, y int32) error {
	if ActiveConfig != nil && !ActiveConfig.VerifyBounds {
		return nil
	}
	bounds := VirtualScreenBounds()
	if x < bounds.X || x >= bounds.X+bounds.W {
		return fmt.Errorf("x=%d out of bounds (virtual screen x=%d width=%d)", x, bounds.X, bounds.W)
	}
	if y < bounds.Y || y >= bounds.Y+bounds.H {
		return fmt.Errorf("y=%d out of bounds (virtual screen y=%d height=%d)", y, bounds.Y, bounds.H)
	}
	return nil
}

func ValidateRegion(x, y, w, h int32) error {
	if ActiveConfig != nil && !ActiveConfig.VerifyBounds {
		return nil
	}
	bounds := VirtualScreenBounds()
	if w <= 0 {
		return fmt.Errorf("width=%d must be positive", w)
	}
	if h <= 0 {
		return fmt.Errorf("height=%d must be positive", h)
	}
	if x < bounds.X || x >= bounds.X+bounds.W {
		return fmt.Errorf("x=%d out of bounds (virtual screen x=%d width=%d)", x, bounds.X, bounds.W)
	}
	if y < bounds.Y || y >= bounds.Y+bounds.H {
		return fmt.Errorf("y=%d out of bounds (virtual screen y=%d height=%d)", y, bounds.Y, bounds.H)
	}
	if x+w > bounds.X+bounds.W {
		return fmt.Errorf("x+w=%d exceeds virtual screen right=%d", x+w, bounds.X+bounds.W)
	}
	if y+h > bounds.Y+bounds.H {
		return fmt.Errorf("y+h=%d exceeds virtual screen bottom=%d", y+h, bounds.Y+bounds.H)
	}
	return nil
}

func CheckScreenshotPermission() error {
	hdc := GetDesktopDC()
	if hdc == 0 {
		return fmt.Errorf("screenshot permission denied: cannot get desktop DC; run in a user session with GUI access")
	}
	ReleaseDesktopDC(hdc)
	return nil
}
