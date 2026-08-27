package actions

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// validTestPngB64 returns a valid in-memory PNG (width x height) encoded as
// base64, used by capture-surface shape tests so decodePNGB64 and geometry
// detection always succeed.
func validTestPngB64(width, height int) string {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 9), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// TestAnnotateCaptureShape verifies the fused annotated capture keeps its
// structure, geometry metadata, and screen-space coordinate translation even
// when the ML engines are unavailable (best-effort contract). It does not
// depend on live screen capture or ONNX model availability.
func TestAnnotateCaptureShape(t *testing.T) {
	ann := AnnotateCapture(validTestPngB64(10, 10), "", "screen", 100, 200, 3)
	if ann == nil {
		t.Fatal("AnnotateCapture returned nil")
	}
	if ann.Source != "screen" {
		t.Errorf("Source = %q, want screen", ann.Source)
	}
	if ann.OriginX != 100 || ann.OriginY != 200 {
		t.Errorf("Origin = (%d,%d), want (100,200)", ann.OriginX, ann.OriginY)
	}
	if ann.Width != 10 || ann.Height != 10 {
		t.Errorf("geometry = %dx%d, want 10x10", ann.Width, ann.Height)
	}
	if ann.DPIScale <= 0 {
		t.Errorf("dpi_scale missing/zero: %v", ann.DPIScale)
	}
	if ann.VirtualScreen == nil {
		t.Errorf("virtual_screen is nil")
	}
	if ann.TotalMs < 0 {
		t.Errorf("TotalMs negative: %d", ann.TotalMs)
	}
	// Elements may be empty on no-models, but never nil.
	if ann.Elements == nil {
		t.Errorf("Elements is nil, want non-nil slice")
	}
}

// TestAnnotatedElementTrust verifies the per-element click/trust gating logic:
// interactive classes with sufficient confidence are clickable; non-interactive
// classes and very-low-confidence interactive ones are not.
func TestAnnotatedElementTrust(t *testing.T) {
	cases := []struct {
		name string
		el   AnnotatedElement
		want bool
	}{
		{
			name: "interactive high conf",
			el:   AnnotatedElement{Class: "button", YOLOConfidence: 0.9, Classified: []ClassResult{{Label: "button", Confidence: 0.9}}},
			want: true,
		},
		{
			name: "non-interactive class",
			el:   AnnotatedElement{Class: "label", YOLOConfidence: 0.95},
			want: false,
		},
		{
			name: "interactive low conf",
			el:   AnnotatedElement{Class: "button", YOLOConfidence: 0.1},
			want: false,
		},
		{
			name: "coco yolo class with mobile interactive label",
			el:   AnnotatedElement{Class: "person", YOLOConfidence: 0.9, Classified: []ClassResult{{Label: "button", Confidence: 0.7}}},
			want: true,
		},
		{
			name: "coco yolo class with non-interactive mobile label",
			el:   AnnotatedElement{Class: "person", YOLOConfidence: 0.9, Classified: []ClassResult{{Label: "unknown", Confidence: 0.7}}},
			want: false,
		},
	}
	for _, c := range cases {
		c.el.CombinedConfidence = CombineElementConfidence(c.el)
		if got := ElementIsClickable(c.el); got != c.want {
			t.Errorf("%s: ElementIsClickable = %v, want %v (conf=%v)", c.name, got, c.want, c.el.CombinedConfidence)
		}
	}
}
