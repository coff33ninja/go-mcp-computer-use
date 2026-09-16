package server

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
)

func TestBuildElementsQueryFilters(t *testing.T) {
	mk := func(class, label string, conf float64, clickable bool, x, y int32) actions.AnnotatedElement {
		ae := actions.AnnotatedElement{
			Class:             class,
			CombinedConfidence: conf,
			Clickable:         clickable,
			ClickPoint:        actions.ElementPoint{X: x, Y: y},
		}
		if label != "" {
			ae.Classified = []actions.ClassResult{{Label: label, Confidence: conf}}
		}
		return ae
	}

	ann := &actions.AnnotatedCapture{
		Source:      "screen",
		WindowTitle: "Test",
		Elements: []actions.AnnotatedElement{
			mk("icon", "button", 0.9, true, 100, 200),
			mk("icon", "text_input", 0.7, true, 300, 400),
			mk("icon", "label", 0.3, false, 500, 600), // low confidence
			mk("icon", "", 0.6, false, 700, 800),
		},
	}

	// Label filter narrows to buttons only.
	r := buildElementsQuery(ann, ElementsQueryArgs{Label: "button"})
	if r.Count != 1 || r.Elements[0].Label != "button" {
		t.Fatalf("label filter: got %d elements, want 1 button", r.Count)
	}

	// MinConfidence floor drops the 0.3 label even without other filters.
	r2 := buildElementsQuery(ann, ElementsQueryArgs{})
	if r2.Count != 3 {
		t.Fatalf("confidence floor: got %d, want 3", r2.Count)
	}

	// ClickableOnly keeps only clickable.
	r3 := buildElementsQuery(ann, ElementsQueryArgs{ClickableOnly: boolPtr(true)})
	if r3.Count != 2 {
		t.Fatalf("clickable only: got %d, want 2", r3.Count)
	}

	// Region containment filters by screen-box click point.
	r4 := buildElementsQuery(ann, ElementsQueryArgs{X: ip32(0), Y: ip32(0), W: ip32(250), H: ip32(250)})
	if r4.Count != 1 || r4.Elements[0].ClickPoint.X != 100 {
		t.Fatalf("region: got %d, want 1 at x=100", r4.Count)
	}

	// Max caps rows.
	r5 := buildElementsQuery(ann, ElementsQueryArgs{Max: 1})
	if r5.Count != 1 {
		t.Fatalf("max: got %d, want 1", r5.Count)
	}

	// OCR text search.
	r6 := buildElementsQuery(&actions.AnnotatedCapture{
		Source: "screen",
		OCR:    &actions.OCRResult{Words: []actions.OCRWord{{Text: "Submit", X: 10, Y: 20, W: 5, H: 5}}},
	}, ElementsQueryArgs{Text: "submit"})
	if len(r6.OcrHits) != 1 || r6.OcrHits[0].Text != "Submit" {
		t.Fatalf("ocr text search: got %d hits, want 1 Submit", len(r6.OcrHits))
	}
}

func ip32(v int32) *int32 { return &v }
