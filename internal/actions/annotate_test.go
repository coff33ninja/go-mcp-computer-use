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

// TestAnnotateWindowTitle verifies the E1 fix: window-scoped captures must be
// keyed on the CAPTURED window's title, not the foreground window's. The
// explicit windowTitle passed to annotateCaptureOpts must win; an empty title
// falls back to the active-window title (or stays empty when none is available).
func TestAnnotateWindowTitle(t *testing.T) {
	b64 := validTestPngB64(10, 10)

	// Explicit title must be used verbatim (E1: ocr_window/screenshot_element
	// reported the foreground window even when a different handle was captured).
	ann := annotateCaptureOpts(b64, "", "window", 0, 0, 3, 0, 0, "", "Mozilla Firefox")
	if ann.WindowTitle != "Mozilla Firefox" {
		t.Fatalf("WindowTitle = %q, want %q (explicit title must override foreground)", ann.WindowTitle, "Mozilla Firefox")
	}

	// Empty title falls back to the foreground window's title (unchanged legacy
	// behavior for screen/region/detect_image paths).
	ann2 := annotateCaptureOpts(b64, "", "screen", 0, 0, 3, 0, 0, "", "")
	if ann2.WindowTitle == "" {
		// If no foreground window, empty is acceptable; when one exists it must
		// be reported. Just ensure the field is populated from some source.
		return
	}
	active := getActiveWindowTitle()
	if active != "" && ann2.WindowTitle != active {
		t.Fatalf("empty-title fallback = %q, want active window %q", ann2.WindowTitle, active)
	}
}

// TestFilterAndCapElements verifies the fused-capture element array stays
// bounded: low-confidence degenerate boxes are dropped to a trust floor and the
// survivors are sorted by confidence and capped to the limit (the OCR-dump fix).
func TestFilterAndCapElements(t *testing.T) {
	mk := func(conf float64, class string) AnnotatedElement {
		return AnnotatedElement{Class: class, CombinedConfidence: conf}
	}
	in := []AnnotatedElement{
		mk(0.95, "icon"),
		mk(0.10, "icon"), // degenerate, below floor
		mk(0.50, "icon"),
		mk(0.30, "icon"), // below floor (0.40)
		mk(0.80, "icon"),
	}

	// No cap: drops the two low boxes, sorts high-first, keeps 3.
	got := FilterAndCapElements(in, 0, 0.40)
	if len(got) != 3 {
		t.Fatalf("filter: got %d elements, want 3", len(got))
	}
	if got[0].CombinedConfidence != 0.95 || got[1].CombinedConfidence != 0.80 || got[2].CombinedConfidence != 0.50 {
		t.Fatalf("filter: wrong sort order: %v", got)
	}

	// Cap of 2 truncates to highest two.
	got2 := FilterAndCapElements(in, 2, 0.40)
	if len(got2) != 2 || got2[0].CombinedConfidence != 0.95 || got2[1].CombinedConfidence != 0.80 {
		t.Fatalf("cap: got %d elements with wrong values", len(got2))
	}

	// All-below-floor yields empty non-nil slice (JSON [] not null).
	got3 := FilterAndCapElements([]AnnotatedElement{mk(0.1, "icon")}, 0, 0.40)
	if got3 == nil || len(got3) != 0 {
		t.Fatalf("all-filtered: expected empty non-nil slice, got %#v", got3)
	}

	// Nil input yields empty non-nil slice.
	got4 := FilterAndCapElements(nil, 0, 0.40)
	if got4 == nil || len(got4) != 0 {
		t.Fatalf("nil input: expected empty non-nil slice, got %#v", got4)
	}
}
