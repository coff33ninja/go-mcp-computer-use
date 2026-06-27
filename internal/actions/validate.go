package actions

import (
	"fmt"

	"github.com/user/go-mcp-computer-use/internal/config"
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
	sw, sh := ScreenSize()
	if x < 0 || x >= sw {
		return fmt.Errorf("x=%d out of bounds (screen width=%d)", x, sw)
	}
	if y < 0 || y >= sh {
		return fmt.Errorf("y=%d out of bounds (screen height=%d)", y, sh)
	}
	return nil
}

func ValidateRegion(x, y, w, h int32) error {
	if ActiveConfig != nil && !ActiveConfig.VerifyBounds {
		return nil
	}
	sw, sh := ScreenSize()
	if w <= 0 {
		return fmt.Errorf("width=%d must be positive", w)
	}
	if h <= 0 {
		return fmt.Errorf("height=%d must be positive", h)
	}
	if x < 0 || x >= sw {
		return fmt.Errorf("x=%d out of bounds (screen width=%d)", x, sw)
	}
	if y < 0 || y >= sh {
		return fmt.Errorf("y=%d out of bounds (screen height=%d)", y, sh)
	}
	if x+w > sw {
		return fmt.Errorf("x+w=%d exceeds screen width=%d", x+w, sw)
	}
	if y+h > sh {
		return fmt.Errorf("y+h=%d exceeds screen height=%d", y+h, sh)
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
