package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ElementsQueryArgs filters the fused capture (YOLO + MobileNet + OCR) down to a
// compact, text-LLM-friendly result. The AI asks "where is the submit button"
// (class/label filter + region) and gets a handful of flat rows with screen
// coordinates — instead of a multi-hundred-KB element dump to grep through.
type ElementsQueryArgs struct {
	// Source is the capture source: screen (default), window, or region.
	Source string `json:"source,omitempty"`
	// Handle is the window handle when Source == "window".
	Handle uintptr `json:"handle,omitempty"`
	// Region bounds when Source == "region" (or a containment filter for any source).
	X *int32 `json:"x,omitempty"`
	Y *int32 `json:"y,omitempty"`
	W *int32 `json:"w,omitempty"`
	H *int32 `json:"h,omitempty"`
	// Class filters elements by YOLO class (case-insensitive substring).
	Class string `json:"class,omitempty"`
	// Label filters elements by MobileNet top-1 label (case-insensitive substring).
	Label string `json:"label,omitempty"`
	// MinConfidence drops elements below this fused combined-confidence floor.
	MinConfidence *float64 `json:"min_confidence,omitempty"`
	// ClickableOnly returns only elements the clickable gate recommends acting on.
	ClickableOnly *bool `json:"clickable_only,omitempty"`
	// Text searches the OCR words of the same frame (case-insensitive substring)
	// and returns matching text with screen coordinates. Useful for locating
	// labels/buttons that OCR read but YOLO/MobileNet did not classify.
	Text string `json:"text,omitempty"`
	// Max caps the number of element rows returned (default 25, 0 = default).
	Max int `json:"max,omitempty"`
	// IncludeImage overrides the global config include_image for this call.
	IncludeImage *bool `json:"include_image,omitempty"`
}

// ElementsQueryResult is the compact, flat projection a text-based LLM can act
// on directly. It deliberately omits the bulky fused AnnotatedElement struct.
type ElementsQueryResult struct {
	Source       string             `json:"source"`
	WindowTitle  string             `json:"window_title,omitempty"`
	Count        int                `json:"count"`
	FilteredFrom int                `json:"filtered_from"`
	Elements     []ElementQueryRow  `json:"elements"`
	OcrHits      []OCRQueryHit      `json:"ocr_hits,omitempty"`
	TotalMs      int64              `json:"total_ms"`
}

// ElementQueryRow is one flat, screen-actionable element row.
type ElementQueryRow struct {
	Index      int32          `json:"index"`
	Class      string         `json:"class"`
	Label      string         `json:"label,omitempty"`
	Confidence float64        `json:"confidence"`
	Clickable  bool           `json:"clickable"`
	ScreenBox  actions.ElementBox `json:"screen_box"`
	ImageBox   actions.ElementBox `json:"image_box"`
	ClickPoint actions.ElementPoint `json:"click_point"`
}

// OCRQueryHit is one OCR word/text match with screen coordinates.
type OCRQueryHit struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

func elementsQueryHandler(ctx context.Context, req *mcp.CallToolRequest, args ElementsQueryArgs) (*mcp.CallToolResult, any, error) {
	var ann *actions.AnnotatedCapture
	switch strings.ToLower(args.Source) {
	case "window":
		if args.Handle == 0 {
			return nil, nil, fmt.Errorf("elements_query window: handle required")
		}
		ann = actions.AnnotateWindow(args.Handle, "", 3)
	case "region":
		if args.X == nil || args.Y == nil || args.W == nil || args.H == nil {
			return nil, nil, fmt.Errorf("elements_query region: x,y,w,h required")
		}
		ann = actions.AnnotateRegion(*args.X, *args.Y, *args.W, *args.H, "", 3)
	default:
		ann = actions.AnnotateScreen("", 3)
	}

	res := buildElementsQuery(ann, args)

	// Return without the bulky base64 image unless explicitly requested.
	return &mcp.CallToolResult{},
		map[string]any{"query_result": res, "source": res.Source}, nil
}

func buildElementsQuery(ann *actions.AnnotatedCapture, args ElementsQueryArgs) *ElementsQueryResult {
	res := &ElementsQueryResult{
		Source:   ann.Source,
		Elements: []ElementQueryRow{},
		OcrHits:  []OCRQueryHit{},
	}
	if ann == nil {
		return res
	}
	res.WindowTitle = ann.WindowTitle
	res.TotalMs = ann.TotalMs

	res.FilteredFrom = len(ann.Elements)

	minConf := 0.40
	if args.MinConfidence != nil {
		minConf = *args.MinConfidence
	}
	max := 25
	if args.Max > 0 {
		max = args.Max
	}
	clickableOnly := args.ClickableOnly != nil && *args.ClickableOnly

	classF := strings.ToLower(strings.TrimSpace(args.Class))
	labelF := strings.ToLower(strings.TrimSpace(args.Label))
	region := args.X != nil && args.Y != nil && args.W != nil && args.H != nil

	for i, e := range ann.Elements {
		if max > 0 && len(res.Elements) >= max {
			break
		}
		if e.CombinedConfidence < minConf {
			continue
		}
		if clickableOnly && !e.Clickable {
			continue
		}
		mobileLabel := ""
		if len(e.Classified) > 0 {
			mobileLabel = e.Classified[0].Label
		}
		if classF != "" && !strings.Contains(strings.ToLower(e.Class), classF) {
			continue
		}
		if labelF != "" && !strings.Contains(strings.ToLower(mobileLabel), labelF) {
			continue
		}
		// Region containment test on the screen-box center.
		if region {
			cx, cy := e.ClickPoint.X, e.ClickPoint.Y
			if cx < *args.X || cx > *args.X+*args.W || cy < *args.Y || cy > *args.Y+*args.H {
				continue
			}
		}
		res.Elements = append(res.Elements, ElementQueryRow{
			Index:      int32(i),
			Class:      e.Class,
			Label:      mobileLabel,
			Confidence: e.CombinedConfidence,
			Clickable:  e.Clickable,
			ScreenBox:  e.ScreenBox,
			ImageBox:   e.ImageBox,
			ClickPoint: e.ClickPoint,
		})
	}

	if txt := strings.ToLower(strings.TrimSpace(args.Text)); txt != "" && ann.OCR != nil {
		seen := map[string]bool{}
		for _, w := range ann.OCR.Words {
			if max > 0 && len(res.OcrHits) >= max {
				break
			}
			if !strings.Contains(strings.ToLower(w.Text), txt) {
				continue
			}
			key := fmt.Sprintf("%s|%.0f|%.0f", w.Text, w.X, w.Y)
			if seen[key] {
				continue
			}
			seen[key] = true
			res.OcrHits = append(res.OcrHits, OCRQueryHit{Text: w.Text, X: w.X, Y: w.Y, W: w.W, H: w.H})
		}
	}

	res.Count = len(res.Elements)
	return res
}
