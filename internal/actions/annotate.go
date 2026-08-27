package actions

import (
	"image"
	"time"
)

// ElementBox is a bounding box in one coordinate space.
type ElementBox struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
	W int32 `json:"w"`
	H int32 `json:"h"`
}

// AnnotatedElement is a fully-composed, actionable UI element: the YOLO-detected
// box fused with MobileNet classification, the element-priors DB, and ML-memory
// cross-reference, exposed in BOTH bitmap-image space and virtual-screen space
// so any AI (vision-based or text-bound) knows exactly what is on screen and
// where to interact.
type AnnotatedElement struct {
	// Class is the YOLO-detected element class.
	Class string `json:"class"`
	// YOLOConfidence is the (prior-adjusted) YOLO detection confidence.
	YOLOConfidence float64 `json:"yolo_confidence"`
	// ImageBox is the element box in the captured image's pixel space (matches
	// the OCR bboxes and the raw image).
	ImageBox ElementBox `json:"image_box"`
	// ScreenBox is the same element box translated to virtual-screen
	// coordinates — the coordinates to click / type into.
	ScreenBox ElementBox `json:"screen_box"`
	// Classified is the MobileNet top-N classification of the element crop.
	Classified []ClassResult `json:"classified,omitempty"`
	// Priors is what the element-priors DB knows about this element class in
	// the current window, when a prior entry exists.
	Priors *ClickValidationPriors `json:"priors,omitempty"`
	// MLMemory is the adaptive/ML-engine memory cross-reference ("have I seen
	// this element/click before"), when available.
	MLMemory *ClickValidationML `json:"ml_memory,omitempty"`
	// ClickPoint is a screen-space {X, Y} centroid (center of ScreenBox) — the
	// exact coordinates the AI should click/type into for this element.
	ClickPoint ElementPoint `json:"click_point"`
	// CombinedConfidence is a single 0-1 trust score fusing YOLO + MobileNet
	// (top-1) + priors-known + ML-memory. Higher = more trustworthy to act on.
	CombinedConfidence float64 `json:"combined_confidence"`
	// Clickable recommends whether the AI should interact with this element
	// based on confidence AND whether the element class is a known interactive
	// type. Low-confidence or non-interactive boxes are flagged false to
	// prevent false-positive over-clicks.
	Clickable bool `json:"clickable"`
}

// ElementPoint is a single coordinate (screen space).
type ElementPoint struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

// AnnotatedCapture is the always-on fused result returned by AI-facing capture
// tools (screenshot, screenshot_element, ocr, ocr_window, ocr_active_window,
// get_viewport). All signals run on the SAME captured frame so their geometry
// shares one coordinate space. It composes every classifier the server has —
// OCR (text), YOLO (element boxes), MobileNet (element type), the element-priors
// DB and the adaptive ML engine — into one structured, actionable view.
//
// Best-effort: any sub-block may be empty if its engine is unavailable, but the
// capture itself never fails because a classifier is missing.
type AnnotatedCapture struct {
	// ImageB64 is the raw captured image (unchanged, for vision models).
	ImageB64 string `json:"image_b64,omitempty"`
	// OCR is the text-recognition signal for the same frame.
	OCR *OCRResult `json:"ocr,omitempty"`
	// Elements are the composed, actionable UI elements with screen-space boxes.
	Elements []AnnotatedElement `json:"elements,omitempty"`
	// Source describes what was captured (screen / window / region / element).
	Source string `json:"source,omitempty"`
	// WindowTitle is the active window context.
	WindowTitle string `json:"window_title,omitempty"`
	// OriginX/OriginY is the virtual-screen origin of the captured image
	// (image (0,0) maps to screen (OriginX, OriginY)).
	OriginX int32 `json:"origin_x,omitempty"`
	OriginY int32 `json:"origin_y,omitempty"`
	// Width/Height are the captured image dimensions in pixels (bitmap space).
	Width  int32 `json:"width,omitempty"`
	Height int32 `json:"height,omitempty"`
	// DPIScale is the DPI scaling factor (1.0 = 96 DPI) at the capture origin.
	DPIScale float64 `json:"dpi_scale,omitempty"`
	// VirtualScreen is the full virtual desktop bounds (physical pixels). Lets
	// the AI reason about multi-monitor layout and which part of the screen the
	// capture represents.
	VirtualScreen *Rect `json:"virtual_screen,omitempty"`
	// TotalMs is the end-to-end annotation time.
	TotalMs int64 `json:"total_ms"`
	// Errors notes which sub-signals were unavailable (for diagnosis), if any.
	Errors []string `json:"errors,omitempty"`
}

// AnnotateCapture runs OCR + YOLO-detection + MobileNet classification + priors
// + ML-memory over a single base64 PNG frame and returns the fused
// AnnotatedCapture. originX/originY is the virtual-screen position of the
// image's top-left corner (used to build actionable ScreenBox coords). It never
// fails the capture; unavailable engines yield empty sub-blocks noted in Errors.
// topN limits the per-element classification depth.
func AnnotateCapture(b64, language, source string, originX, originY int32, topN int) *AnnotatedCapture {
	return annotateCaptureOpts(b64, language, source, originX, originY, topN, 0, 0, "")
}

// annotateCaptureOpts is the core fused-annotation producer. It additionally
// accepts YOLO detection threshold/IOU overrides (0 = model defaults) and an
// optional provided-image source tag used by onnx_detect when a caller passes an
// explicit image_b64 rather than a live capture.
func annotateCaptureOpts(b64, language, source string, originX, originY int32, topN int, threshold, iou float64, providedSource string) *AnnotatedCapture {
	start := time.Now()
	ac := &AnnotatedCapture{
		ImageB64: b64,
		Source:   source,
		OriginX:  originX,
		OriginY:  originY,
		TotalMs:  0,
		Elements: []AnnotatedElement{},
		VirtualScreen: func() *Rect {
			b := VirtualScreenBounds()
			return &b
		}(),
	}
	if providedSource != "" {
		ac.Source = providedSource
	}

	// Capture dimension metadata from the decoded bitmap (also reused below for
	// cropping). DPI scale is sampled at the capture origin.
	{
		var dimsW, dimsH int32
		if img, derr := decodePNGB64(b64); derr == nil {
			b := img.Bounds()
			dimsW, dimsH = int32(b.Dx()), int32(b.Dy())
		}
		ac.Width, ac.Height = dimsW, dimsH
		if dpi, derr := GetDPIScaleForPoint(originX, originY); derr == nil {
			ac.DPIScale = float64(dpi) / 96.0
		}
	}

	if title := getActiveWindowTitle(); title != "" {
		ac.WindowTitle = title
	}

	// OCR (text + bboxes on the same frame).
	if ocr, err := ocrFromBase64(b64, language); err == nil && ocr != nil {
		ac.OCR = ocr
	} else if err != nil {
		ac.Errors = append(ac.Errors, "ocr: "+err.Error())
	}

	// YOLO detection (element boxes on the same frame).
	var yoloEls []DetectedElement
	if det, err := ONNXDetect(DetectionInput{ImageB64: b64, Threshold: threshold, IOUThreshold: iou}); err != nil {
		ac.Errors = append(ac.Errors, "detect: "+err.Error())
	} else if det != nil {
		yoloEls = det.Elements
	}

	// Compose each element with MobileNet + priors + ML memory, and translate
	// its box into screen space.
	if len(yoloEls) > 0 {
		var img image.Image
		if decoded, derr := decodePNGB64(b64); derr == nil {
			img = decoded
		} else {
			ac.Errors = append(ac.Errors, "decode: "+derr.Error())
		}
		for _, el := range yoloEls {
			sbx, sby := el.X+originX, el.Y+originY
			ae := AnnotatedElement{
				Class:          el.Class,
				YOLOConfidence: el.Confidence,
				ImageBox:       ElementBox{X: el.X, Y: el.Y, W: el.W, H: el.H},
				ScreenBox: ElementBox{
					X: sbx,
					Y: sby,
					W: el.W,
					H: el.H,
				},
				ClickPoint: ElementPoint{X: sbx + el.W/2, Y: sby + el.H/2},
			}
			// MobileNet classification of this element's crop from the SAME frame.
			if img != nil {
				if top, cerr := classifyElementFromImage(img, el, topN); cerr == nil && len(top) > 0 {
					ae.Classified = top
				}
			}
			// Priors + ML memory for this element class in this window.
			if priors, ml := clickContextForClass(ac.WindowTitle, el.Class, el.X, el.Y); priors != nil {
				ae.Priors = priors
				ae.MLMemory = ml
			}
			// Fuse every signal into one trust score and a click recommendation.
			ae.CombinedConfidence = CombineElementConfidence(ae)
			ae.Clickable = ElementIsClickable(ae)
			ac.Elements = append(ac.Elements, ae)
		}
	}

	ac.TotalMs = time.Since(start).Milliseconds()
	return ac
}

// AnnotateDetectImage annotates a caller-supplied base64 image with explicit
// YOLO detection threshold/IOU overrides. Origin defaults to (0,0) (image-relative
// coords), so screen_box reflects the image's own pixel space.
func AnnotateDetectImage(b64, language string, topN int, threshold, iou float64) *AnnotatedCapture {
	return annotateCaptureOpts(b64, language, "detect_image", 0, 0, topN, threshold, iou, "detect_image")
}

// AnnotateDetectScreen captures the live screen and annotates it with explicit
// YOLO detection threshold/IOU overrides, using the virtual-screen origin so the
// screen_box coords are directly actionable.
func AnnotateDetectScreen(language string, topN int, threshold, iou float64) *AnnotatedCapture {
	b64, err := CaptureScreen()
	if err != nil {
		return &AnnotatedCapture{Source: "detect_screen", Errors: []string{"capture: " + err.Error()}, ImageB64: ""}
	}
	b := VirtualScreenBounds()
	return annotateCaptureOpts(b64, language, "detect_screen", b.X, b.Y, topN, threshold, iou, "detect_screen")
}

// AnnotateScreen captures the full virtual screen and annotates it. The origin
// is the virtual-screen top-left, so ScreenBox coords are directly actionable.
func AnnotateScreen(language string, topN int) *AnnotatedCapture {
	b64, err := CaptureScreen()
	if err != nil {
		return &AnnotatedCapture{Source: "screen", Errors: []string{"capture: " + err.Error()}, ImageB64: ""}
	}
	b := VirtualScreenBounds()
	return AnnotateCapture(b64, language, "screen", b.X, b.Y, topN)
}

// AnnotateRegion captures a screen region and annotates it. The origin is the
// requested region's top-left, so ScreenBox coords map back to the exact
// requested capture position.
func AnnotateRegion(x, y, w, h int32, language string, topN int) *AnnotatedCapture {
	b64, err := CaptureRegion(x, y, w, h)
	if err != nil {
		return &AnnotatedCapture{Source: "region", Errors: []string{"capture: " + err.Error()}, ImageB64: ""}
	}
	return AnnotateCapture(b64, language, "region", x, y, topN)
}

// AnnotateWindow captures a window (clamped to visible screen bounds, matching
// ScreenshotElement) and annotates it. The origin is the clamped on-screen
// top-left, so ScreenBox coords are actionable screen coordinates.
func AnnotateWindow(handle uintptr, language string, topN int) *AnnotatedCapture {
	b64, err := ScreenshotElement(handle)
	if err != nil {
		return &AnnotatedCapture{Source: "window", Errors: []string{"capture: " + err.Error()}, ImageB64: ""}
	}
	var ox, oy int32
	if state, sErr := GetWindowState(handle); sErr == nil && state.Rect != nil {
		ox, oy = state.Rect.Left, state.Rect.Top
		if ox < 0 {
			ox = 0
		}
		if oy < 0 {
			oy = 0
		}
	}
	return AnnotateCapture(b64, language, "window", ox, oy, topN)
}

// classifyElementFromImage classifies a single detected element's crop extracted
// directly from the provided decoded image (image-space box, exact alignment).
func classifyElementFromImage(img image.Image, el DetectedElement, topN int) ([]ClassResult, error) {
	bounds := img.Bounds()
	x0, y0 := int(el.X), int(el.Y)
	if x0 < bounds.Min.X {
		x0 = bounds.Min.X
	}
	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	x1, y1 := x0+int(el.W), y0+int(el.H)
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}
	if x1 <= x0 || y1 <= y0 {
		return nil, image.ErrFormat
	}
	crop := cropImage(img, x0, y0, x1, y1)
	return ClassifyImage(crop, topN)
}

// clickContextForClass builds the priors + ML-memory advisory context for an
// element class at (x, y) in the given window. It returns nil priors when the
// priors DB has no entries for the (class, window) pair, in which case no
// ML-memory cross-reference is attempted either (avoiding a blind full query).
func clickContextForClass(windowTitle, className string, x, y int32) (*ClickValidationPriors, *ClickValidationML) {
	samples := PriorSampleCount(windowTitle, className)
	if samples <= 0 {
		return nil, nil
	}
	priorConf := AdjustConfidenceWithPriors(className, windowTitle, 0.5, float64(x), float64(y))
	freq, avgX, avgY := priorStats(windowTitle, className)
	priors := &ClickValidationPriors{
		Class:           className,
		Samples:         samples,
		PriorConfidence: priorConf,
		KnownConfidence: ElementKnownConfidently(windowTitle, className, 0, 0, 0.5, 3, watcherLocTolerance),
		Frequency:       freq,
		AvgX:            avgX,
		AvgY:            avgY,
	}
	var ml *ClickValidationML
	if q := Adaptive.MLQuery("click "+className, "", 3); q != nil && len(q.Matches) > 0 {
		best := q.Matches[0]
		ml = &ClickValidationML{
			Match:      best.Coord != nil,
			Confidence: best.Confidence,
			Samples:    best.Samples,
		}
		if best.Coord != nil {
			ml.X = best.Coord.X
			ml.Y = best.Coord.Y
		}
	}
	return priors, ml
}

// interactiveClasses are the element types the AI should actually interact with
// (click / type / drag). Non-interactive classes (label, image, container,
// unknown, etc.) are excluded so the AI does not waste clicks on passive chrome.
var interactiveClasses = map[string]bool{
	"button": true, "checkbox": true, "dropdown": true, "icon_button": true,
	"link": true, "menu_item": true, "scrollbar": true, "slider": true,
	"tab": true, "text_input": true, "toggle": true,
}

// CombineElementConfidence fuses YOLO + MobileNet + priors + ML-memory into a
// single 0-1 trust score for acting on an element. Each signal that exists
// contributes; the score favors signals the system has actually "seen before"
// (priors samples / ML samples) to combat false positives on unfamiliar UI.
func CombineElementConfidence(ae AnnotatedElement) float64 {
	weights := []struct {
		v float64
		w float64
	}{}
	// YOLO detection confidence (prior-adjusted already by ONNXDetect).
	if ae.YOLOConfidence > 0 {
		weights = append(weights, struct {
			v float64
			w float64
		}{ae.YOLOConfidence, 0.4})
	}
	// MobileNet top-1 label+confidence for the crop. This is the AUTHORITATIVE
	// UI-class signal (it emits real controls, unlike the COCO YOLO class), so we
	// weight it more heavily whenever it names an interactive type — whether or
	// not it happens to agree with the YOLO proposal class.
	if len(ae.Classified) > 0 && ae.Classified[0].Confidence > 0 {
		w := 0.3
		if interactiveClasses[ae.Classified[0].Label] {
			w = 0.4
		}
		weights = append(weights, struct {
			v float64
			w float64
		}{ae.Classified[0].Confidence, w})
	}
	// Priors known-location + frequency (the strongest anti-false-positive
	// signal: the system has physically seen this element in this window).
	if ae.Priors != nil {
		known := 0.0
		if ae.Priors.KnownConfidence {
			known = 0.9
		} else if ae.Priors.Frequency > 0 {
			known = min64(0.5+ae.Priors.Frequency*0.5, 0.9)
		}
		weights = append(weights, struct {
			v float64
			w float64
		}{known, 0.35})
	}
	// ML-memory confidence ("have I clicked here before in this context").
	if ae.MLMemory != nil && ae.MLMemory.Match {
		weights = append(weights, struct {
			v float64
			w float64
		}{ae.MLMemory.Confidence, 0.2})
	}
	if len(weights) == 0 {
		// No classifiers produced signal — do not trust this box.
		return 0
	}
	var sum, wsum float64
	for _, e := range weights {
		sum += e.v * e.w
		wsum += e.w
	}
	if wsum == 0 {
		return 0
	}
	return sum / wsum
}

// ElementIsClickable decides whether the AI should act on an element. It
// requires (a) an interactive UI class from the whitelist AND (b) a minimum
// combined confidence, discouraging blind clicks on low-trust boxes.
//
// The authoritative UI-class signal is the MobileNet top-1 classified label
// (the ui-element-classifier emits real controls like button/text_input/link),
// whereas the YOLO proposal class is a general COCO object label (person, vase,
// ...) and is rarely interactive. We therefore accept the element when either
// the MobileNet top label OR the YOLO class names an interactive type.
func ElementIsClickable(ae AnnotatedElement) bool {
	kind := ae.Class
	if len(ae.Classified) > 0 && interactiveClasses[ae.Classified[0].Label] {
		kind = ae.Classified[0].Label
	}
	if !interactiveClasses[kind] {
		return false
	}
	return ae.CombinedConfidence >= 0.35
}

// min64 returns the smaller of two float64 values.
func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// cropImage returns a sub-image bounded by (x0,y0)-(x1,y1). When the source
// implements SubImage we return a zero-copy windowed view; otherwise we fall
// back to a copied crop via a fresh RGBA.
func cropImage(src image.Image, x0, y0, x1, y1 int) image.Image {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := src.(subImager); ok {
		return si.SubImage(image.Rect(x0, y0, x1, y1))
	}
	dst := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dst.Set(x-x0, y-y0, src.At(x, y))
		}
	}
	return dst
}
