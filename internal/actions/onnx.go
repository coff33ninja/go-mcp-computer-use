package actions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
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
)

const (
	yoloInputSize      = 640
	yoloConfThresh     = 0.25
	yoloNMSThresh      = 0.45
	yoloNumClasses     = 7
	yoloModelURL       = "https://huggingface.co/IndextDataLab/windows-ui-locator/resolve/main/best.pt"
	yoloModelFile      = "yolov11s_windows_ui.onnx"
	mobilenetModelURL  = "https://huggingface.co/diogoneno/gui-element-classifier/resolve/main/mobilenetv3_small.onnx"
	mobilenetModelFile = "mobilenetv3_small.onnx"
	onnxDLLURL         = "https://github.com/microsoft/onnxruntime/releases/download/v1.20.1/onnxruntime-win-x64-1.20.1.zip"
	onnxDLLFile        = "onnxruntime.dll"
)

var yoloLabels = []string{
	"button", "textbox", "checkbox", "dropdown", "icon", "tab", "menu_item",
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
	YoloModel     string `json:"yolo_model"`
	Mobilenet     string `json:"mobilenet"`
	YoloFormat    string `json:"yolo_format,omitempty"`
	RuntimeDLL    string `json:"runtime_dll,omitempty"`
}

func checkYoloModel(dir string) (string, string) {
	onnxPath := filepath.Join(dir, "yolov11s_windows_ui.onnx")
	if _, err := os.Stat(onnxPath); err == nil {
		return "present", "onnx"
	}
	ptPath := filepath.Join(dir, "best.pt")
	if _, err := os.Stat(ptPath); err == nil {
		return "present_pt", "pytorch"
	}
	return "missing", ""
}

func ONNXStatus() *ONNXModelStatus {
	s := &ONNXModelStatus{}
	if modelsDir != "" {
		s.YoloModel, s.YoloFormat = checkYoloModel(modelsDir)
		mobPath := filepath.Join(modelsDir, "mobilenetv3_small.onnx")
		if _, err := os.Stat(mobPath); err == nil {
			s.Mobilenet = "present"
		} else {
			s.Mobilenet = "missing"
		}
		rtPath := findONNXRuntime()
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
	ImageB64 string  `json:"image_b64"`
	Threshold float64 `json:"threshold,omitempty"`
	IOUThreshold float64 `json:"iou_threshold,omitempty"`
}

type DetectedElement struct {
	Class      string  `json:"class"`
	Confidence float64 `json:"confidence"`
	X          int32   `json:"x"`
	Y          int32   `json:"y"`
	W          int32   `json:"w"`
	H          int32   `json:"h"`
}

type DetectionOutput struct {
	Elements    []DetectedElement `json:"elements"`
	TotalMs     int64             `json:"total_ms"`
	ModelInput  string            `json:"model_input,omitempty"`
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

	yoloPath := filepath.Join(modelsDir, yoloModelFile)
	if _, err := os.Stat(yoloPath); os.IsNotExist(err) {
		// ONNX not available — try Python/Ultralytics fallback with best.pt
		ptPath := filepath.Join(modelsDir, "best.pt")
		if _, ptErr := os.Stat(ptPath); ptErr == nil {
			out, pyErr := detectWithPython(img, ptPath, in.Threshold, in.IOUThreshold)
			if pyErr == nil {
				out.TotalMs = time.Since(start).Milliseconds()
				return out, nil
			}
		}
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
	boxes := parseYOLOOutput(outputData, yoloInputSize, img.Bounds().Dx(), img.Bounds().Dy())

	thresh := in.Threshold
	if thresh <= 0 {
		thresh = yoloConfThresh
	}
	iouThresh := in.IOUThreshold
	if iouThresh <= 0 {
		iouThresh = yoloNMSThresh
	}

	filtered := nms(filterBoxes(boxes, float32(thresh)), float32(iouThresh))

	elements := make([]DetectedElement, 0, len(filtered))
	for _, b := range filtered {
		elements = append(elements, DetectedElement{
			Class:      yoloLabels[b.classID],
			Confidence: float64(b.confidence),
			X:          int32(b.x),
			Y:          int32(b.y),
			W:          int32(b.w),
			H:          int32(b.h),
		})
	}

	return &DetectionOutput{
		Elements:   elements,
		TotalMs:    time.Since(start).Milliseconds(),
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

func parseYOLOOutput(data []float32, inputSize, imgW, imgH int) []yoloBox {
	numDetections := 8400
	rowStride := 4 + yoloNumClasses
	scaleX := float32(imgW) / float32(inputSize)
	scaleY := float32(imgH) / float32(inputSize)

	boxes := make([]yoloBox, 0, numDetections)
	for i := 0; i < numDetections; i++ {
		offset := i * rowStride
		cx := data[offset] * scaleX
		cy := data[offset+1] * scaleY
		w := data[offset+2] * scaleX
		h := data[offset+3] * scaleY

		bestClass := 0
		bestConf := float32(0)
		for c := 0; c < yoloNumClasses; c++ {
			conf := sigmoid(data[offset+4+c])
			if conf > bestConf {
				bestConf = conf
				bestClass = c
			}
		}

		if bestConf > 0 {
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

func filterBoxes(boxes []yoloBox, threshold float32) []yoloBox {
	filtered := make([]yoloBox, 0, len(boxes))
	for _, b := range boxes {
		if b.confidence >= threshold {
			filtered = append(filtered, b)
		}
	}
	return filtered
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

type ONNXDownloadResult struct {
	YoloModel      string `json:"yolo_model"`
	Mobilenet      string `json:"mobilenet"`
	RuntimeDLL     string `json:"runtime_dll"`
	YoloBytes      int64  `json:"yolo_bytes,omitempty"`
	MobilenetBytes int64  `json:"mobilenet_bytes,omitempty"`
	RuntimeStatus  string `json:"runtime_status,omitempty"`
}

type pythonDetectResult struct {
	Elements []struct {
		Class      string  `json:"class"`
		Confidence float64 `json:"confidence"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
		W          float64 `json:"w"`
		H          float64 `json:"h"`
	} `json:"elements"`
}

func detectWithPython(img image.Image, ptPath string, threshold, iouThreshold float64) (*DetectionOutput, error) {
	// Find Python interpreter
	python, err := exec.LookPath("python")
	if err != nil {
		python, err = exec.LookPath("python3")
		if err != nil {
			return nil, fmt.Errorf("python not found")
		}
	}

	// Save image to temp file
	tmpDir := os.TempDir()
	imgPath := filepath.Join(tmpDir, "yolo_detect_input.png")
	if err := savePNG(imgPath, img); err != nil {
		return nil, fmt.Errorf("save temp image: %w", err)
	}
	defer os.Remove(imgPath)

	// Choose label mapping for this model (7 UI classes from best.pt)
	var labelsJSON bytes.Buffer
	labelsJSON.WriteString("[")
	for i, l := range yoloLabels {
		if i > 0 {
			labelsJSON.WriteString(",")
		}
		b, _ := json.Marshal(l)
		labelsJSON.Write(b)
	}
	labelsJSON.WriteString("]")

	// Python script that loads the model and runs detection
	script := fmt.Sprintf(`
import json, sys, warnings
warnings.filterwarnings("ignore")
try:
    from ultralytics import YOLO
except ImportError:
    print(json.dumps({"error": "ultralytics not installed"}))
    sys.exit(0)

model = YOLO(%q)
results = model.predict(%q, conf=%f, iou=%f, verbose=False)
labels = %s
elements = []
for r in results:
    if r.boxes is None:
        continue
    for box, cls, conf in zip(r.boxes.xywh, r.boxes.cls, r.boxes.conf):
        cx, cy, w, h = box.tolist()
        elements.append({
            "class": labels[int(cls.item())],
            "confidence": round(float(conf), 4),
            "x": round(cx - w/2, 1),
            "y": round(cy - h/2, 1),
            "w": round(w, 1),
            "h": round(h, 1),
        })
print(json.dumps({"elements": elements}))
`, ptPath, imgPath, threshold, iouThreshold, labelsJSON.String())

	scriptPath := filepath.Join(tmpDir, "yolo_detect_runner.py")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return nil, fmt.Errorf("write script: %w", err)
	}
	defer os.Remove(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(python, scriptPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python exec: %w\nstderr: %s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		// Non-fatal, log but continue
	}

	var result pythonDetectResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// Check if it's a known import error
		var errResult struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(stdout.Bytes(), &errResult) == nil && errResult.Error != "" {
			return nil, fmt.Errorf("python: %s", errResult.Error)
		}
		return nil, fmt.Errorf("parse python output: %w\noutput: %s", err, stdout.String())
	}

	elements := make([]DetectedElement, 0, len(result.Elements))
	for _, e := range result.Elements {
		elements = append(elements, DetectedElement{
			Class:      e.Class,
			Confidence: e.Confidence,
			X:          int32(e.X),
			Y:          int32(e.Y),
			W:          int32(e.W),
			H:          int32(e.H),
		})
	}

	return &DetectionOutput{Elements: elements, ModelInput: "pytorch"}, nil
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
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

func ONNXDownload() (*ONNXDownloadResult, error) {
	dir := getModelsDir()
	if dir == "" {
		return nil, fmt.Errorf("APPDATA not set")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create models dir: %w", err)
	}

	result := &ONNXDownloadResult{}

	// YOLO model: HF only provides PyTorch (.pt), not ONNX
	yoloPath := filepath.Join(dir, "best.pt")
	if _, err := os.Stat(yoloPath); os.IsNotExist(err) {
		n, err := downloadFile(yoloModelURL, yoloPath)
		if err != nil {
			result.YoloModel = fmt.Sprintf("download_failed: %s", err)
		} else {
			result.YoloModel = "downloaded_pt"
			result.YoloBytes = n
		}
	} else {
		result.YoloModel = "present_pt"
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
