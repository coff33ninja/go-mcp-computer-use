package actions

import (
	"fmt"
	"strings"
)

type FindUIElementInput struct {
	Label       string `json:"label"`
	WindowTitle string `json:"window_title,omitempty"`
	UseOCR      bool   `json:"use_ocr,omitempty"`
}

type FindUIElementResult struct {
	Found       bool             `json:"found"`
	Element     *DetectedElement `json:"element,omitempty"`
	Source      string           `json:"source"`
	WindowTitle string           `json:"window_title,omitempty"`
	OCRText     string           `json:"ocr_text,omitempty"`
}

func FindUIElement(in FindUIElementInput) (*FindUIElementResult, error) {
	winTitle := in.WindowTitle
	if winTitle == "" {
		if info, err := GetActiveWindowInfo(); err == nil {
			winTitle = info.Title
		}
	}

	// 1. Check memory first
	memKey := fmt.Sprintf("ui:%s:%s", winTitle, in.Label)
	fact, err := MemoryGet(memKey, "ui")
	if err == nil && fact != nil {
		if el, ok := fact.Value.(map[string]any); ok {
			if x, xok := el["x"].(float64); xok {
				if y, yok := el["y"].(float64); yok {
					if w, wok := el["w"].(float64); wok {
						if h, hok := el["h"].(float64); hok {
							return &FindUIElementResult{
								Found:       true,
								Source:      "memory",
								WindowTitle: winTitle,
								Element: &DetectedElement{
									Class:      in.Label,
									X:          int32(x),
									Y:          int32(y),
									W:          int32(w),
									H:          int32(h),
								},
							}, nil
						}
					}
				}
			}
		}
	}

	// 2. Run ONNX detection
	b64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("find_ui_element screenshot: %w", err)
	}

	detResult, err := ONNXDetect(DetectionInput{ImageB64: b64})
	if err == nil {
		labelLower := strings.ToLower(in.Label)
		for _, el := range detResult.Elements {
			if strings.Contains(strings.ToLower(el.Class), labelLower) ||
				strings.Contains(labelLower, strings.ToLower(el.Class)) {
				SaveTrainingSample(SaveTrainingSampleInput{
					Source:      TrainingSourceRaw,
					Category:    TrainingCatGeneral,
					TaskPrompt:  fmt.Sprintf("find %s on screen", in.Label),
					ImageB64:    b64,
					WindowTitle: winTitle,
				})
				return &FindUIElementResult{
					Found:       true,
					Source:      "onnx",
					WindowTitle: winTitle,
					Element:     &el,
				}, nil
			}
		}
	}

	// 3. Fallback: try OCR if text-based search
	if in.UseOCR || in.Label != "" {
		ocrResult, err := OCRScreen("")
		if err == nil {
			labelLower := strings.ToLower(in.Label)
			for _, word := range ocrResult.Words {
				if strings.Contains(strings.ToLower(word.Text), labelLower) {
					result := &FindUIElementResult{
						Found:       true,
						Source:      "ocr",
						WindowTitle: winTitle,
						OCRText:     word.Text,
						Element: &DetectedElement{
							Class: in.Label,
							X:     int32(word.X),
							Y:     int32(word.Y),
							W:     int32(word.W),
							H:     int32(word.H),
						},
					}
					MemoryStoreDetectionElements([]DetectedElement{*result.Element}, winTitle)
					return result, nil
				}
			}
			for _, line := range ocrResult.Lines {
				if strings.Contains(strings.ToLower(line.Text), labelLower) {
					result := &FindUIElementResult{
						Found:       true,
						Source:      "ocr",
						WindowTitle: winTitle,
						OCRText:     line.Text,
						Element: &DetectedElement{
							Class: in.Label,
							X:     int32(line.X),
							Y:     int32(line.Y),
							W:     int32(line.W),
							H:     int32(line.H),
						},
					}
					MemoryStoreDetectionElements([]DetectedElement{*result.Element}, winTitle)
					return result, nil
				}
			}
		}
	}

	// 4. Not found anywhere — save as negative training sample
	SaveTrainingSample(SaveTrainingSampleInput{
		Source:      TrainingSourceRaw,
		Category:    TrainingCatGeneral,
		TaskPrompt:  fmt.Sprintf("could not find %s on screen", in.Label),
		ImageB64:    b64,
		WindowTitle: winTitle,
	})

	return &FindUIElementResult{
		Found:       false,
		Source:      "none",
		WindowTitle: winTitle,
	}, nil
}
