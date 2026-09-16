package actions

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSoftmaxTopN(t *testing.T) {
	logits := []float32{1.0, 2.0, 3.0, 0.5, -1.0, 4.0, 2.5, 0.0, 0.1, -2.0, 3.5, 1.5, 0.8, 0.9, 2.0}
	labels := mobilenetLabels
	res := softmaxTopN(logits, labels, 3)
	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}
	if res[0].Index != 5 { // 4.0 is the max logit -> index 5 = "image"
		t.Fatalf("expected index 5 first, got %d (%s)", res[0].Index, res[0].Label)
	}
	// Confidence should be a probability in (0,1]
	sum := 0.0
	for _, r := range res {
		sum += r.Confidence
		if r.Confidence <= 0 || r.Confidence > 1 {
			t.Fatalf("confidence out of range: %v", r.Confidence)
		}
	}
	if sum > 1.0001 {
		t.Fatalf("sum of all top concurrency exceeds 1: %v", sum)
	}
	// Verify monotonic
	if res[0].Confidence < res[1].Confidence || res[1].Confidence < res[2].Confidence {
		t.Fatalf("results not sorted desc: %v", res)
	}
}

func TestPreprocessMobileNet(t *testing.T) {
	// A 100x50 image should be padded to 100x100 (gray border) then resized to 224.
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	blob := preprocessMobileNet(img)
	want := 1 * 3 * 224 * 224
	if len(blob) != want {
		t.Fatalf("blob len = %d, want %d", len(blob), want)
	}
	// Red channel at center of the original image area: (255/255 - 0.485)/0.229
	centerIdx := 0*224*224 + 112*224 + 112
	wantVal := (255.0/255.0 - 0.485) / 0.229
	if math.Abs(float64(blob[centerIdx])-wantVal) > 0.02 {
		t.Fatalf("center red normalized = %v, want ~%v", blob[centerIdx], wantVal)
	}
	// Gray border (padded region) for a channel where pad=128: (128/255-mean)/std
	// Corner (0,0) of the padded square is gray since original is 100x50 centered.
	padVal := (128.0/255.0 - 0.485) / 0.229
	if math.Abs(float64(blob[0])-padVal) > 0.02 {
		t.Fatalf("corner red = %v, want gray-ish ~%v", blob[0], padVal)
	}
	_ = color.RGBA{}
}

func TestClassifyImage_Integration(t *testing.T) {
	dir := filepath.Join(os.Getenv("APPDATA"), "go-mcp-computer-use", "models")
	if _, err := os.Stat(filepath.Join(dir, mobilenetModelFile)); os.IsNotExist(err) {
		t.Skip("mobilenet model not present; skipping integration test")
	}
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	// Solid gray image — model should still produce a sane 15-class output.
	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	res, err := ClassifyImage(img, 5)
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if len(res) != 5 {
		t.Fatalf("expected 5 results, got %d", len(res))
	}
	for _, r := range res {
		if r.Label == "" {
			t.Fatalf("empty label in result: %+v", r)
		}
		if r.Confidence <= 0 || r.Confidence > 1 {
			t.Fatalf("bad confidence: %+v", r)
		}
	}
	t.Logf("top results: %+v", res)
}

// TestPreprocessYOLO_LetterboxAspect verifies that a non-square source image is
// fit into the 640x640 model input with a SINGLE uniform scale (aspect
// preserved) plus centered padding — the fix for the degenerate-box flood,
// where X and Y were being rescaled independently and grossly distorted
// non-square inputs like the 3200x1980 virtual desktop.
func TestPreprocessYOLO_LetterboxAspect(t *testing.T) {
	srcW, srcH := 3200, 1980
	img := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	blob, lb := preprocessYOLO(img, yoloInputSize)
	if len(blob) != 3*yoloInputSize*yoloInputSize {
		t.Fatalf("blob len = %d, want %d", len(blob), 3*yoloInputSize*yoloInputSize)
	}
	// Uniform scale: min(640/3200, 640/1980) = 640/3200 = 0.2
	wantScale := float64(yoloInputSize) / float64(srcW)
	if math.Abs(lb.scale-wantScale) > 1e-9 {
		t.Fatalf("scale = %v, want %v", lb.scale, wantScale)
	}
	// newH = 1980*0.2 = 396; vertical padding = (640-396)/2 = 122
	wantNewH := int(math.Round(float64(srcH) * lb.scale))
	wantPadY := (yoloInputSize - wantNewH) / 2
	if lb.padY != wantPadY {
		t.Fatalf("padY = %d, want %d", lb.padY, wantPadY)
	}
	if lb.padX != 0 {
		t.Fatalf("padX = %d, want 0 (X scale == target, no X padding)", lb.padX)
	}
	// Original content region: blob[0] (red) inside the scaled area should be
	// source color 200/255; the padded border should be gray 128/255.
	cx := lb.padX + int(math.Round(float64(srcW)/2*lb.scale))
	cy := lb.padY + int(math.Round(float64(srcH)/2*lb.scale))
	inner := blob[cy*yoloInputSize+cx]
	if math.Abs(float64(inner)-200.0/255.0) > 0.01 {
		t.Fatalf("inner red = %v, want ~%v", inner, 200.0/255.0)
	}
	border := blob[0]
	if math.Abs(float64(border)-128.0/255.0) > 0.01 {
		t.Fatalf("border pad red = %v, want gray ~%v", border, 128.0/255.0)
	}
}

// TestParseYOLOOutput_RoundTrip verifies the inverse mapping: a source-image box
// pushed forward into 640-letterbox space by the model comes back to the same
// source-image coordinates after parseYOLOOutput.
func TestParseYOLOOutput_RoundTrip(t *testing.T) {
	srcW, srcH := 3200, 1980
	scale := math.Min(float64(yoloInputSize)/float64(srcW), float64(yoloInputSize)/float64(srcH))
	padX := (yoloInputSize - int(math.Round(float64(srcW)*scale))) / 2
	padY := (yoloInputSize - int(math.Round(float64(srcH)*scale))) / 2
	lb := yoloLetterbox{scale: scale, padX: padX, padY: padY}

	// A source-image box (cx, cy, w, h).
	src := struct{ cx, cy, w, h float32 }{1600.0, 990.0, 200.0, 120.0}
	// Forward: what the model emits in 640 input space (already box-space).
	fwCX := float32(float64(src.cx)*scale + float64(padX))
	fwCY := float32(float64(src.cy)*scale + float64(padY))
	fwW := float32(float64(src.w) * scale)
	fwH := float32(float64(src.h) * scale)

	// Build a YOLO output buffer with one strong detection. Zero-fill would
	// give every anchor a sigmoid(0)=0.5 class logit (above the 0.25 threshold),
	// so set all class logits very negative to silence the noise floor.
	rowStride := 4 + yoloNumClasses
	data := make([]float32, 8400*rowStride)
	for i := range data {
		data[i] = -10.0
	}
	idx := 0 * rowStride
	data[idx] = fwCX
	data[idx+1] = fwCY
	data[idx+2] = fwW
	data[idx+3] = fwH
	data[idx+4+0] = 10.0 // class logit -> sigmoid(10) ≈ 1.0 (well above 0.25)

	boxes := parseYOLOOutput(data, yoloInputSize, lb, 0.25)
	if len(boxes) != 1 {
		t.Fatalf("expected 1 box, got %d", len(boxes))
	}
	b := boxes[0]
	const tol = 1.5 // allow rounding in the integer letterbox fit
	if math.Abs(float64(b.x-(float32(src.cx-src.w/2)))) > tol ||
		math.Abs(float64(b.y-(float32(src.cy-src.h/2)))) > tol ||
		math.Abs(float64(b.w-src.w)) > tol ||
		math.Abs(float64(b.h-src.h)) > tol {
		t.Fatalf("round-trip mismatch: got box %+v, want source (cx=%v cy=%v w=%v h=%v)",
			b, src.cx, src.cy, src.w, src.h)
	}
}

// TestClipElementToImage_FiltersPaddingFalsePositives reproduces the garbage-
// coordinate bug: a detector box centered in the letterbox padding band maps to
// negative source y (e.g. {526,-150,518,574} on a 1608x860 window — the exact
// degenerate elements live elements_query returned). The center gate must drop
// it before it ever reaches the AI as an actionable click point.
func TestClipElementToImage_FiltersPaddingFalsePositives(t *testing.T) {
	const imgW, imgH = 1608, 860

	cases := []struct {
		name string
		el   DetectedElement
		want bool // keep (true) / drop (false)
	}{
		// Padding-band false positives: center above/below/left/right of the
		// image. These produced the reported negative + oversized boxes.
		{"top-padding", DetectedElement{X: 100, Y: -351, W: 390, H: 44}, false},
		{"far-top", DetectedElement{X: 500, Y: -500, W: 50, H: 60}, false},
		{"bottom-padding", DetectedElement{X: 948, Y: 842, W: 839, H: 236}, false},
		{"left-padding", DetectedElement{X: -400, Y: 300, W: 390, H: 44}, false},
		{"fully-right", DetectedElement{X: 1600, Y: 100, W: 200, H: 60}, false},
		// Degenerate geometry.
		{"zero-size", DetectedElement{X: 10, Y: 10, W: 0, H: 10}, false},
		{"negative-size", DetectedElement{X: 10, Y: 10, W: -5, H: 10}, false},
		// Legitimate on-screen boxes: center inside, kept (possibly clipped).
		{"intact", DetectedElement{X: 400, Y: 300, W: 100, H: 50}, true},
		{"top-straddle", DetectedElement{X: 526, Y: -150, W: 518, H: 574}, true},
		{"right-straddle", DetectedElement{X: 1500, Y: 100, W: 200, H: 60}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			el := tc.el
			got := clipElementToImage(&el, imgW, imgH)
			if got != tc.want {
				t.Fatalf("clipElementToImage(%+v) = %v, want %v", tc.el, got, tc.want)
			}
		})
	}
}

// TestClipElementToImage_ClipsOffImageBoxes verifies that straddling boxes are
// clipped to the image bounds and their width/height recomputed, so a click
// point derived from the clipped box is always on-screen.
func TestClipElementToImage_ClipsOffImageBoxes(t *testing.T) {
	const imgW, imgH = 1608, 860

	// 200x60 box hanging off the right edge by 42px and past the bottom by 10px,
	// with its center still on-screen.
	el := DetectedElement{X: 1450, Y: 810, W: 200, H: 60}
	if !clipElementToImage(&el, imgW, imgH) {
		t.Fatalf("straddling box should be kept")
	}
	if el.X != 1450 || el.W != 158 || el.Y != 810 || el.H != 50 {
		t.Fatalf("unexpected clip: got %+v, want X=1450 W=158 Y=810 H=50", el)
	}

	// Box sticking out the top keeps the visible portion and recenters the box.
	el2 := DetectedElement{X: 100, Y: -20, W: 120, H: 60}
	if !clipElementToImage(&el2, imgW, imgH) {
		t.Fatalf("top-straddling box should be kept")
	}
	if el2.X != 100 || el2.W != 120 || el2.Y != 0 || el2.H != 40 {
		t.Fatalf("unexpected top clip: got %+v, want X=100 W=120 Y=0 H=40", el2)
	}
}
