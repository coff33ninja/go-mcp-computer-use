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
