package actions

import "testing"

func TestNormalizeVirtualDesktopPointUsesVirtualOrigin(t *testing.T) {
	bounds := Rect{X: -1600, Y: -1080, W: 3200, H: 1980}
	x, y := normalizeVirtualDesktopPoint(0, 0, bounds)

	wantX := int32((1600 * 65535) / 3199)
	wantY := int32((1080 * 65535) / 1979)
	if x != wantX || y != wantY {
		t.Fatalf("normalizeVirtualDesktopPoint(0,0) = (%d,%d), want (%d,%d)", x, y, wantX, wantY)
	}
}
