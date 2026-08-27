package actions

import (
	"archive/zip"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	ort "github.com/yalue/onnxruntime_go"
)

type ONNXState int

const (
	ONNXUninitialized ONNXState = iota
	ONNXReady
	ONNXNoRuntime
	ONNXNoModels
	ONNXError
)

var (
	onnxState   ONNXState
	onnxStateMu sync.Mutex
	modelsDir   string

	// yoloAutoDLMu serializes automatic model downloads so concurrent detection
	// paths (watcher goroutine + tool calls) don't race on the same file.
	// yoloAutoDLDone records whether we already attempted download this process
	// so we don't hammer the network on every detection when it is unavailable.
	yoloAutoDLMu   sync.Mutex
	yoloAutoDLDone bool
)

const (
	yoloInputSize  = 640
	yoloConfThresh = 0.25
	yoloNMSThresh  = 0.45
	// The GUI detector is a single-class ("icon") ONNX export of Salesforce
	// GPA-GUI-Detector (MIT) fine-tuned from OmniParser. It proposes element
	// boxes; the authoritative per-control UI class comes from MobileNetV3.
	// The .onnx ships alongside the binary. yoloModelURL uses GitHub's
	// releases/latest/download/... form, which always redirects to the newest
	// non-draft non-prerelease release's asset — so the URL needs no hardcoded
	// version tag and follows every version bump. ensureYoloModel fetches it
	// automatically when the model file is absent.
	yoloNumClasses     = 1
	yoloModelURL       = "https://github.com/coff33ninja/go-mcp-computer-use/releases/latest/download/gpa_gui_detector.onnx"
	yoloModelFile      = "gpa_gui_detector.onnx"
	mobilenetModelURL  = "https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx"
	mobilenetModelFile = "mobilenetv3_small.onnx"
	onnxDLLURL         = "https://github.com/microsoft/onnxruntime/releases/download/v1.26.0/onnxruntime-win-x64-1.26.0.zip"
	onnxDLLFile        = "onnxruntime.dll"
)

// yoloLabels corresponds to the detector's single output class. GPA-GUI is a
// single-class ("icon") detector: it proposes interactive-element boxes but
// does not distinguish control types. The 15-class MobileNet tier is the
// authoritative UI control label (see mobilenetLabels). Keeping exactly one
// label matches the exported (1, 5, 8400) output layout.
var yoloLabels = []string{
	"icon",
}

// mobilenetLabels is the 15-class label set of the gui-element-classifier
// MobileNetV3-small model (diogoneno/gui-element-classifier). Indices 0..14
// match the model output in alphabetical order.
var mobilenetLabels = []string{
	"button", "checkbox", "container", "dropdown", "icon_button",
	"image", "label", "link", "menu_item", "scrollbar",
	"slider", "tab", "text_input", "toggle", "unknown",
}

func getModelsDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "go-mcp-computer-use", "models")
}

func InitONNX() error {
	onnxStateMu.Lock()
	defer onnxStateMu.Unlock()

	// Already initialized
	if onnxState == ONNXReady {
		return nil
	}

	modelsDir = getModelsDir()
	if modelsDir == "" {
		onnxState = ONNXNoRuntime
		return fmt.Errorf("APPDATA not set")
	}
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		onnxState = ONNXNoRuntime
		return fmt.Errorf("create models dir: %w", err)
	}

	// Re-find runtime DLL (may have been downloaded since last attempt)
	rtPath := findONNXRuntime()
	ort.SetSharedLibraryPath(rtPath)
	if err := ort.InitializeEnvironment(); err != nil {
		onnxState = ONNXNoRuntime
		return fmt.Errorf("onnx runtime init: %w", err)
	}

	onnxState = ONNXReady
	return nil
}

func findONNXRuntime() string {
	candidates := []string{
		filepath.Join(modelsDir, "onnxruntime.dll"),
		filepath.Join(modelsDir, onnxDLLFile),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fallback: check working dir and system32
	if _, err := os.Stat(onnxDLLFile); err == nil {
		return onnxDLLFile
	}
	sysPath := filepath.Join(os.Getenv("WINDIR"), "System32", "onnxruntime.dll")
	if _, err := os.Stat(sysPath); err == nil {
		return sysPath
	}
	return filepath.Join(modelsDir, onnxDLLFile)
}

type ONNXModelStatus struct {
	YoloModel  string `json:"yolo_model"`
	Mobilenet  string `json:"mobilenet"`
	RuntimeDLL string `json:"runtime_dll,omitempty"`
}

func checkYoloModel(dir string) string {
	onnxPath := filepath.Join(dir, yoloModelFile)
	if _, err := os.Stat(onnxPath); err == nil {
		return "present"
	}
	return "missing"
}

func ONNXStatus() *ONNXModelStatus {
	s := &ONNXModelStatus{}
	dir := getModelsDir()
	if dir != "" {
		s.YoloModel = checkYoloModel(dir)
		mobPath := filepath.Join(dir, "mobilenetv3_small.onnx")
		if _, err := os.Stat(mobPath); err == nil {
			s.Mobilenet = "present"
		} else {
			s.Mobilenet = "missing"
		}
		rtPath := filepath.Join(dir, onnxDLLFile)
		if _, err := os.Stat(rtPath); err == nil {
			s.RuntimeDLL = rtPath
		}
	} else {
		s.YoloModel = "unknown"
		s.Mobilenet = "unknown"
	}
	return s
}

type DetectionInput struct {
	ImageB64     string  `json:"image_b64"`
	Threshold    float64 `json:"threshold,omitempty"`
	IOUThreshold float64 `json:"iou_threshold,omitempty"`
}

type DetectedElement struct {
	Class      string  `json:"class"`
	Confidence float64 `json:"confidence"`
	X          int32   `json:"x"`
	Y          int32   `json:"y"`
	W          int32   `json:"w"`
	H          int32   `json:"h"`
	// MobileNetLabel is the authoritative 15-class UI control label assigned
	// by the MobileNet classifier to this element's crop. The detector class
	// (single-class "icon") is only a box proposer; priors/dedup and the
	// clickable gate key on this label when present.
	MobileNetLabel string `json:"mobile_net_label,omitempty"`
}

type DetectionOutput struct {
	Elements    []DetectedElement   `json:"elements"`
	TotalMs     int64               `json:"total_ms"`
	ModelInput  string              `json:"model_input,omitempty"`
	SavedRef    string              `json:"saved_ref,omitempty"`
	WindowTitle string              `json:"window_title,omitempty"`
	Normalized  []NormalizedElement `json:"normalized,omitempty"`
}

func ONNXDetect(in DetectionInput) (*DetectionOutput, error) {
	start := time.Now()

	if err := InitONNX(); err != nil {
		// Runtime unavailable — return empty gracefully
	}

	onnxStateMu.Lock()
	state := onnxState
	onnxStateMu.Unlock()

	if state == ONNXNoRuntime {
		return &DetectionOutput{
			Elements:   []DetectedElement{},
			TotalMs:    time.Since(start).Milliseconds(),
			ModelInput: "runtime_not_found",
		}, nil
	}

	img, err := decodePNGB64(in.ImageB64)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	yoloPath, err := ensureYoloModel(modelsDir)
	if err != nil || yoloPath == "" {
		return &DetectionOutput{
			Elements:   []DetectedElement{},
			TotalMs:    time.Since(start).Milliseconds(),
			ModelInput: "model_not_found",
		}, nil
	}

	blob := preprocessYOLO(img, yoloInputSize)
	inputShape := ort.NewShape(1, 3, yoloInputSize, yoloInputSize)
	inputTensor, err := ort.NewTensor(inputShape, blob)
	if err != nil {
		return nil, fmt.Errorf("create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	outputShape := ort.NewShape(1, 4+yoloNumClasses, 8400)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, fmt.Errorf("create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(yoloPath,
		[]string{"images"}, []string{"output0"},
		[]ort.Value{inputTensor}, []ort.Value{outputTensor}, nil)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("run inference: %w", err)
	}

	outputData := outputTensor.GetData()

	thresh := float32(in.Threshold)
	if thresh <= 0 {
		thresh = yoloConfThresh
	}
	iouThresh := float32(in.IOUThreshold)
	if iouThresh <= 0 {
		iouThresh = yoloNMSThresh
	}

	boxes := parseYOLOOutput(outputData, yoloInputSize, img.Bounds().Dx(), img.Bounds().Dy(), thresh)
	filtered := nms(boxes, iouThresh)

	elements := make([]DetectedElement, 0, len(filtered))
	winTitle := ""
	var winHandle uintptr
	if info, err := GetActiveWindowInfo(); err == nil && info != nil {
		winTitle = info.Title
		winHandle = info.Handle
	}
	for _, b := range filtered {
		el := DetectedElement{
			Class:      yoloLabels[b.classID],
			Confidence: float64(b.confidence),
			X:          int32(b.x),
			Y:          int32(b.y),
			W:          int32(b.w),
			H:          int32(b.h),
		}
		if winTitle != "" {
			el.Confidence = AdjustConfidenceWithPriors(el.Class, winTitle, el.Confidence, float64(el.X), float64(el.Y))
		}
		elements = append(elements, el)
	}

	var normalized []NormalizedElement
	if winHandle != 0 && len(elements) > 0 {
		if wn, err := NewWindowNormalizer(winHandle); err == nil {
			normalized = make([]NormalizedElement, len(elements))
			for i, el := range elements {
				normalized[i] = wn.NormalizeElement(el)
			}
		}
		// Store detections in memory for AI reuse
		MemoryStoreDetectionElements(elements, winTitle)
	}

	return &DetectionOutput{
		Elements:    elements,
		TotalMs:     time.Since(start).Milliseconds(),
		WindowTitle: winTitle,
		Normalized:  normalized,
	}, nil
}

type yoloBox struct {
	classID    int
	confidence float32
	x, y, w, h float32
}

func preprocessYOLO(img image.Image, targetSize int) []float32 {
	bounds := img.Bounds()
	blob := make([]float32, 3*targetSize*targetSize)

	rScale := float64(targetSize) / float64(bounds.Dx())
	cScale := float64(targetSize) / float64(bounds.Dy())

	for y := 0; y < targetSize; y++ {
		for x := 0; x < targetSize; x++ {
			srcX := int(float64(x) / rScale)
			srcY := int(float64(y) / cScale)
			if srcX >= bounds.Dx() {
				srcX = bounds.Dx() - 1
			}
			if srcY >= bounds.Dy() {
				srcY = bounds.Dy() - 1
			}
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			idx := y*targetSize + x
			blob[idx] = float32(r>>8) / 255.0
			blob[targetSize*targetSize+idx] = float32(g>>8) / 255.0
			blob[2*targetSize*targetSize+idx] = float32(b>>8) / 255.0
		}
	}
	return blob
}

func parseYOLOOutput(data []float32, inputSize, imgW, imgH int, confThresh float32) []yoloBox {
	numDetections := 8400
	rowStride := 4 + yoloNumClasses
	scaleX := float32(imgW) / float32(inputSize)
	scaleY := float32(imgH) / float32(inputSize)

	boxes := make([]yoloBox, 0, 256)
	for i := 0; i < numDetections; i++ {
		offset := i * rowStride
		cx := data[offset] * scaleX
		cy := data[offset+1] * scaleY
		w := data[offset+2] * scaleX
		h := data[offset+3] * scaleY

		bestClass := 0
		bestConf := float32(0)
		off := offset + 4
		offEnd := off + yoloNumClasses
		for c := off; c < offEnd; c++ {
			conf := sigmoid(data[c])
			if conf > bestConf {
				bestConf = conf
				bestClass = c - off
			}
		}

		if bestConf >= confThresh {
			boxes = append(boxes, yoloBox{
				classID:    bestClass,
				confidence: bestConf,
				x:          cx - w/2,
				y:          cy - h/2,
				w:          w,
				h:          h,
			})
		}
	}
	return boxes
}

func sigmoid(x float32) float32 {
	return 1.0 / (1.0 + float32(math.Exp(float64(-x))))
}

func nms(boxes []yoloBox, iouThreshold float32) []yoloBox {
	if len(boxes) == 0 {
		return boxes
	}

	sort.Slice(boxes, func(i, j int) bool {
		return boxes[i].confidence > boxes[j].confidence
	})

	selected := make([]yoloBox, 0, len(boxes))
	removed := make([]bool, len(boxes))

	for i := 0; i < len(boxes); i++ {
		if removed[i] {
			continue
		}
		selected = append(selected, boxes[i])
		for j := i + 1; j < len(boxes); j++ {
			if removed[j] {
				continue
			}
			if boxes[i].classID != boxes[j].classID {
				continue
			}
			if iou(boxes[i], boxes[j]) >= iouThreshold {
				removed[j] = true
			}
		}
	}
	return selected
}

func iou(a, b yoloBox) float32 {
	x1 := max32(a.x, b.x)
	y1 := max32(a.y, b.y)
	x2 := min32(a.x+a.w, b.x+b.w)
	y2 := min32(a.y+a.h, b.y+b.h)
	intersection := max32(0, x2-x1) * max32(0, y2-y1)
	areaA := a.w * a.h
	areaB := b.w * b.h
	union := areaA + areaB - intersection
	if union <= 0 {
		return 0
	}
	return intersection / union
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

const mobilenetInputSize = 224

// modest model mean/std (ImageNet stats, required by the gui-element-classifier).
var (
	mobilenetMean = []float32{0.485, 0.456, 0.406}
	mobilenetStd  = []float32{0.229, 0.224, 0.225}
)

// ClassResult is a single (label, confidence) prediction for a classified image
// or element crop. Index is the model output class index (0..14).
type ClassResult struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	Index      int     `json:"index"`
}

// ClassificationOutput is the result of classifying a screen/window/region/crop.
type ClassificationOutput struct {
	Top          []ClassResult           `json:"top,omitempty"`
	Elements     []ElementClassification `json:"elements,omitempty"`
	Model        string                  `json:"model"`
	TotalMs      int64                   `json:"total_ms"`
	Source       string                  `json:"source,omitempty"`
	WindowTitle  string                  `json:"window_title,omitempty"`
	ElementCount int                     `json:"element_count,omitempty"`
	SavedRef     string                  `json:"saved_ref,omitempty"`
}

// ElementClassification pairs a detected element with its classified crop.
type ElementClassification struct {
	Element DetectedElement `json:"element"`
	Top     []ClassResult   `json:"top"`
	CropX   int32           `json:"crop_x"`
	CropY   int32           `json:"crop_y"`
	CropW   int32           `json:"crop_w"`
	CropH   int32           `json:"crop_h"`
}

// ClassifyInput describes what to classify and how many top results to return.
type ClassifyInput struct {
	// Source is one of: "screen", "window", "region", "elements", "crop".
	Source string `json:"source,omitempty"`
	// ImageB64 is an optional raw PNG to classify directly (source "crop").
	ImageB64 string `json:"image_b64,omitempty"`
	// X, Y, W, H select a region when source is "region".
	X int32 `json:"x,omitempty"`
	Y int32 `json:"y,omitempty"`
	W int32 `json:"w,omitempty"`
	H int32 `json:"h,omitempty"`
	// TopN limits the number of results returned (default 3, max 15).
	TopN int `json:"top_n,omitempty"`
	// Elements classifies each YOLO-detected element crop when source is "elements".
	Elements []DetectedElement `json:"elements,omitempty"`
}

// cached MobileNet session — created lazily and reused across calls (classify is
// called frequently by the watcher and AI validation tiers).
var (
	mobilenetSessionMu sync.Mutex
	mobilenetSession   struct {
		loaded bool
		sess   *ort.AdvancedSession
		input  *ort.Tensor[float32]
		output *ort.Tensor[float32]
	}
)

// getMobilenetSession lazily creates and caches the MobileNet session. Callers
// must hold mobilenetSessionMu.
func getMobilenetSession() (*ort.AdvancedSession, *ort.Tensor[float32], *ort.Tensor[float32], error) {
	if mobilenetSession.loaded {
		return mobilenetSession.sess, mobilenetSession.input, mobilenetSession.output, nil
	}
	modelPath := filepath.Join(modelsDir, mobilenetModelFile)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("mobilenet model not found")
	}
	inElems := 1 * 3 * mobilenetInputSize * mobilenetInputSize
	inputShape := ort.NewShape(1, 3, mobilenetInputSize, mobilenetInputSize)
	inputTensor, err := ort.NewTensor[float32](inputShape, make([]float32, inElems))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create mobilenet input tensor: %w", err)
	}
	outputShape := ort.NewShape(1, int64(len(mobilenetLabels)))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		inputTensor.Destroy()
		return nil, nil, nil, fmt.Errorf("create mobilenet output tensor: %w", err)
	}
	sess, err := ort.NewAdvancedSession(modelPath,
		[]string{"input"}, []string{"output"},
		[]ort.Value{inputTensor}, []ort.Value{outputTensor}, nil)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, nil, nil, fmt.Errorf("create mobilenet session: %w", err)
	}
	mobilenetSession.sess = sess
	mobilenetSession.input = inputTensor
	mobilenetSession.output = outputTensor
	mobilenetSession.loaded = true
	return sess, inputTensor, outputTensor, nil
}

// ClassifyImage runs the MobileNet classifier on a single image and returns the
// top-N (label, confidence) results. Preprocessing matches the model's required
// pipeline: PadToSquare(gray 128) -> BILINEAR resize to 224 -> /255 -> ImageNet
// normalize.
func ClassifyImage(img image.Image, topN int) ([]ClassResult, error) {
	if err := InitONNX(); err != nil {
		return nil, fmt.Errorf("onnx runtime unavailable: %w", err)
	}
	if topN <= 0 {
		topN = 3
	}
	if topN > len(mobilenetLabels) {
		topN = len(mobilenetLabels)
	}

	mobilenetSessionMu.Lock()
	sess, inputTensor, outputTensor, err := getMobilenetSession()
	if err != nil {
		mobilenetSessionMu.Unlock()
		return nil, err
	}
	blob := preprocessMobileNet(img)
	copy(inputTensor.GetData(), blob)
	if err := sess.Run(); err != nil {
		mobilenetSessionMu.Unlock()
		return nil, fmt.Errorf("run mobilenet inference: %w", err)
	}
	logits := make([]float32, len(outputTensor.GetData()))
	copy(logits, outputTensor.GetData())
	mobilenetSessionMu.Unlock()

	return softmaxTopN(logits, mobilenetLabels, topN), nil
}

// ClassifyB64 decodes a base64 PNG and classifies it.
func ClassifyB64(imageB64 string, topN int) ([]ClassResult, error) {
	img, err := decodePNGB64(imageB64)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return ClassifyImage(img, topN)
}

// ClassifyScreen captures the full virtual screen and classifies it.
func ClassifyScreen(topN int) (*ClassificationOutput, error) {
	start := time.Now()
	b64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("capture screen: %w", err)
	}
	top, err := ClassifyB64(b64, topN)
	if err != nil {
		return nil, err
	}
	return &ClassificationOutput{
		Top:     top,
		Model:   "mobilenetv3_small",
		TotalMs: time.Since(start).Milliseconds(),
		Source:  "screen",
	}, nil
}

// ClassifyRegion captures a screen region and classifies it.
func ClassifyRegion(x, y, w, h int32, topN int) (*ClassificationOutput, error) {
	start := time.Now()
	b64, err := CaptureRegion(x, y, w, h)
	if err != nil {
		return nil, fmt.Errorf("capture region: %w", err)
	}
	top, err := ClassifyB64(b64, topN)
	if err != nil {
		return nil, err
	}
	return &ClassificationOutput{
		Top:     top,
		Model:   "mobilenetv3_small",
		TotalMs: time.Since(start).Milliseconds(),
		Source:  "region",
	}, nil
}

// ClassifyWindow captures the active/foreground window's on-screen region and
// classifies it.
func ClassifyWindow(topN int) (*ClassificationOutput, error) {
	start := time.Now()
	info, err := GetActiveWindowInfo()
	if err != nil || info == nil {
		return nil, fmt.Errorf("no active window: %w", err)
	}
	rect, ok := windowScreenRect(info)
	if !ok {
		return nil, fmt.Errorf("could not determine window region")
	}
	b64, err := CaptureRegion(rect.X, rect.Y, rect.W, rect.H)
	if err != nil {
		return nil, fmt.Errorf("capture window: %w", err)
	}
	top, err := ClassifyB64(b64, topN)
	if err != nil {
		return nil, err
	}
	out := &ClassificationOutput{
		Top:     top,
		Model:   "mobilenetv3_small",
		TotalMs: time.Since(start).Milliseconds(),
		Source:  "window",
	}
	out.WindowTitle = info.Title
	return out, nil
}

// ClassifyElements classifies each detected element crop against the live
// screen and returns per-element classifications in a combined output.
func ClassifyElements(elements []DetectedElement, topN int) (*ClassificationOutput, error) {
	start := time.Now()
	perEl := classifyElementList(elements, topN)
	return &ClassificationOutput{
		Elements:     perEl,
		Model:        "mobilenetv3_small",
		TotalMs:      time.Since(start).Milliseconds(),
		Source:       "elements",
		ElementCount: len(perEl),
	}, nil
}

// classifyElementList classifies each detected element crop against the live
// screen and returns per-element classifications. Best-effort: elements that
// fail to capture/classify are skipped.
func classifyElementList(elements []DetectedElement, topN int) []ElementClassification {
	bounds := VirtualScreenBounds()
	perEl := make([]ElementClassification, 0, len(elements))
	for _, el := range elements {
		px, py, pw, ph, ok := elementCropRegion(bounds, el)
		if !ok {
			continue
		}
		b64, err := CaptureRegion(px, py, pw, ph)
		if err != nil {
			continue
		}
		top, err := ClassifyB64(b64, topN)
		if err != nil {
			continue
		}
		perEl = append(perEl, ElementClassification{
			Element: el,
			Top:     top,
			CropX:   px,
			CropY:   py,
			CropW:   pw,
			CropH:   ph,
		})
	}
	return perEl
}

// windowScreenRect converts an ActiveWindowInfo into a screen Rect for capture.
func windowScreenRect(info *ActiveWindowInfo) (Rect, bool) {
	if info == nil || info.Width <= 0 || info.Height <= 0 {
		return Rect{}, false
	}
	return Rect{X: info.X, Y: info.Y, W: info.Width, H: info.Height}, true
}

// preprocessMobileNet implements the exact preprocessing required by the
// gui-element-classifier model: PadToSquare with gray (128,128,128) on the
// shorter axis, BILINEAR resize to 224x224, divide by 255, ImageNet normalize,
// then HWC->CHW into a [1,3,224,224] float32 blob.
func preprocessMobileNet(img image.Image) []float32 {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	side := w
	if h > side {
		side = h
	}

	// 1) Pad-to-square with gray background (centered), sample source with
	//    bilinear into a square of size `side`.
	square := make([]uint8, side*side*3)
	offX := (side - w) / 2
	offY := (side - h) / 2
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			sx := x - offX
			sy := y - offY
			dst := (y*side + x) * 3
			if sx < 0 || sy < 0 || sx >= w || sy >= h {
				square[dst] = 128
				square[dst+1] = 128
				square[dst+2] = 128
				continue
			}
			r, g, b, _ := img.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
			square[dst] = uint8(r >> 8)
			square[dst+1] = uint8(g >> 8)
			square[dst+2] = uint8(b >> 8)
		}
	}

	// 2) BILINEAR resize square -> 224x224 with normalization folded in.
	srcStride := side * 3
	scale := float64(side) / float64(mobilenetInputSize)
	blob := make([]float32, 3*mobilenetInputSize*mobilenetInputSize)
	for dy := 0; dy < mobilenetInputSize; dy++ {
		srcY := float64(dy) * scale
		y0 := int(srcY)
		if y0 > side-1 {
			y0 = side - 1
		}
		y1 := y0 + 1
		if y1 > side-1 {
			y1 = side - 1
		}
		fy := srcY - float64(y0)
		for dx := 0; dx < mobilenetInputSize; dx++ {
			srcX := float64(dx) * scale
			x0 := int(srcX)
			if x0 > side-1 {
				x0 = side - 1
			}
			x1 := x0 + 1
			if x1 > side-1 {
				x1 = side - 1
			}
			fx := srcX - float64(x0)

			idx00 := (y0*srcStride + x0*3)
			idx10 := (y0*srcStride + x1*3)
			idx01 := (y1*srcStride + x0*3)
			idx11 := (y1*srcStride + x1*3)

			for c := 0; c < 3; c++ {
				top := float64(square[idx00+c])*(1-fx) + float64(square[idx10+c])*fx
				bot := float64(square[idx01+c])*(1-fx) + float64(square[idx11+c])*fx
				val := top*(1-fy) + bot*fy
				normalized := (val/255.0 - float64(mobilenetMean[c])) / float64(mobilenetStd[c])
				// CHW: channel c, row dy, col dx
				blob[c*mobilenetInputSize*mobilenetInputSize+dy*mobilenetInputSize+dx] = float32(normalized)
			}
		}
	}
	return blob
}

// softmaxTopN computes the softmax of logits and returns the top-N labels with
// their probabilities.
func softmaxTopN(logits []float32, labels []string, n int) []ClassResult {
	if len(logits) == 0 {
		return nil
	}
	maxLogit := logits[0]
	for _, l := range logits {
		if l > maxLogit {
			maxLogit = l
		}
	}
	sum := 0.0
	probs := make([]float64, len(logits))
	for i, l := range logits {
		probs[i] = math.Exp(float64(l) - float64(maxLogit))
		sum += probs[i]
	}
	for i := range probs {
		probs[i] /= sum
	}

	type scored struct {
		idx int
		p   float64
	}
	sc := make([]scored, len(probs))
	for i := range probs {
		sc[i] = scored{i, probs[i]}
	}
	sort.Slice(sc, func(i, j int) bool { return sc[i].p > sc[j].p })
	if n > len(sc) {
		n = len(sc)
	}
	res := make([]ClassResult, 0, n)
	for i := 0; i < n; i++ {
		label := "class_" + string(rune('0'+sc[i].idx))
		if sc[i].idx < len(labels) {
			label = labels[sc[i].idx]
		}
		res = append(res, ClassResult{
			Label:      label,
			Confidence: round64(sc[i].p, 4),
			Index:      sc[i].idx,
		})
	}
	return res
}

func round64(v float64, places int) float64 {
	pow := math.Pow10(places)
	return math.Round(v*pow) / pow
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

type ONNXDownloadResult struct {
	YoloModel      string `json:"yolo_model"`
	Mobilenet      string `json:"mobilenet"`
	RuntimeDLL     string `json:"runtime_dll"`
	YoloBytes      int64  `json:"yolo_bytes,omitempty"`
	MobilenetBytes int64  `json:"mobilenet_bytes,omitempty"`
	RuntimeStatus  string `json:"runtime_status,omitempty"`
}

func downloadFile(url, dest string) (int64, error) {
	tmp := dest + ".tmp"
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http %s: %s", url, resp.Status)
	}
	f, err := os.Create(tmp)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", tmp, err)
	}
	n, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return 0, fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return 0, fmt.Errorf("rename %s -> %s: %w", tmp, dest, err)
	}
	return n, nil
}

func downloadAndExtractZip(url, destDir, extractFile string) (int64, error) {
	tmpZip := filepath.Join(destDir, extractFile+".download.zip")
	_, err := downloadFile(url, tmpZip)
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpZip)

	r, err := zip.OpenReader(tmpZip)
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == extractFile {
			rc, err := f.Open()
			if err != nil {
				return 0, fmt.Errorf("open %s in zip: %w", f.Name, err)
			}
			defer rc.Close()

			outPath := filepath.Join(destDir, extractFile)
			out, err := os.Create(outPath)
			if err != nil {
				return 0, fmt.Errorf("create %s: %w", outPath, err)
			}
			defer out.Close()

			written, err := io.Copy(out, rc)
			if err != nil {
				return 0, fmt.Errorf("extract %s: %w", f.Name, err)
			}
			return written, nil
		}
	}
	return 0, fmt.Errorf("%s not found in zip", extractFile)
}

// ensureYoloModel returns the path to the detector ONNX, auto-downloading it
// from the latest release on first use when it is missing. It is best-effort:
// a failed download degrades gracefully (caller proceeds as model-not-found)
// and is only retried on the next process start, so we never hammer the network
// on every detection when the model is unavailable.
func ensureYoloModel(dir string) (string, error) {
	yoloPath := filepath.Join(dir, yoloModelFile)
	if _, err := os.Stat(yoloPath); err == nil {
		return yoloPath, nil
	}

	yoloAutoDLMu.Lock()
	defer yoloAutoDLMu.Unlock()

	// Re-check under the lock in case another goroutine just fetched it.
	if _, err := os.Stat(yoloPath); err == nil {
		return yoloPath, nil
	}
	if yoloAutoDLDone {
		return yoloPath, fmt.Errorf("detector model not present and auto-download already attempted")
	}
	yoloAutoDLDone = true

	if err := os.MkdirAll(dir, 0755); err != nil {
		return yoloPath, fmt.Errorf("create models dir: %w", err)
	}
	if _, err := downloadFile(yoloModelURL, yoloPath); err != nil {
		return yoloPath, fmt.Errorf("auto-download detector model: %w", err)
	}
	return yoloPath, nil
}

func ONNXDownload() (*ONNXDownloadResult, error) {
	dir := getModelsDir()
	if dir == "" {
		return nil, fmt.Errorf("APPDATA not set")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create models dir: %w", err)
	}

	result := &ONNXDownloadResult{}

	// YOLO model: download pre-exported ONNX directly (no Python/Ultralytics needed)
	yoloPath := filepath.Join(dir, yoloModelFile)
	if _, err := os.Stat(yoloPath); os.IsNotExist(err) {
		n, err := downloadFile(yoloModelURL, yoloPath)
		if err != nil {
			result.YoloModel = fmt.Sprintf("download_failed: %s", err)
		} else {
			result.YoloModel = "downloaded"
			result.YoloBytes = n
			// A manual download succeeded; clear the auto-download guard so the
			// next ensureYoloModel does not short-circuit on a prior failure.
			yoloAutoDLMu.Lock()
			yoloAutoDLDone = false
			yoloAutoDLMu.Unlock()
		}
	} else {
		result.YoloModel = "present"
	}

	// MobileNetV3-small: ONNX format available
	mobPath := filepath.Join(dir, mobilenetModelFile)
	if _, err := os.Stat(mobPath); os.IsNotExist(err) {
		n, err := downloadFile(mobilenetModelURL, mobPath)
		if err != nil {
			result.Mobilenet = fmt.Sprintf("download_failed: %s", err)
		} else {
			result.Mobilenet = "downloaded"
			result.MobilenetBytes = n
		}
	} else {
		result.Mobilenet = "present"
	}

	// ONNX Runtime DLL: download compatible version if not in models dir
	rtLocalPath := filepath.Join(dir, onnxDLLFile)
	if _, err := os.Stat(rtLocalPath); os.IsNotExist(err) {
		_, err := downloadAndExtractZip(onnxDLLURL, dir, onnxDLLFile)
		if err != nil {
			result.RuntimeStatus = fmt.Sprintf("download_failed: %s", err)
		} else {
			result.RuntimeStatus = "downloaded"
			result.RuntimeDLL = rtLocalPath
		}
	} else {
		result.RuntimeStatus = "present"
		result.RuntimeDLL = rtLocalPath
	}

	return result, nil
}
