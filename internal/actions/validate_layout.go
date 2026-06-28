package actions

type LayoutElement struct {
	ID                string      `json:"id"`
	StoredCoord       Coord       `json:"stored_coord"`
	WindowRect        WindowRect  `json:"window_rect,omitempty"`
	SignatureKeywords []string    `json:"signature_keywords,omitempty"`
	ExpectedOCRRegion *WindowRect `json:"expected_ocr_region,omitempty"`
}

type Coord struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type LayoutValidateInput struct {
	Elements       []LayoutElement `json:"elements"`
	WindowTitle    string          `json:"window_title,omitempty"`
	DriftTolerance int32           `json:"drift_tolerance,omitempty"`
	Language       string          `json:"language,omitempty"`
}

type LayoutValidateResult struct {
	Elements []LayoutElementResult `json:"elements"`
	Summary  LayoutSummary         `json:"summary"`
}

type LayoutElementResult struct {
	ID            string `json:"id"`
	Valid         bool   `json:"valid"`
	DriftX        int32  `json:"drift_x"`
	DriftY        int32  `json:"drift_y"`
	KeywordFound  bool   `json:"keyword_found"`
	AdjustedCoord Coord  `json:"adjusted_coord"`
	Confidence    string `json:"confidence"`
}

type LayoutSummary struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Drifted int `json:"drifted"`
	Stale   int `json:"stale"`
}

func ValidateLayout(in LayoutValidateInput) (*LayoutValidateResult, error) {
	tolerance := in.DriftTolerance
	if tolerance <= 0 {
		tolerance = 20
	}

	var currentRect *WindowRect

	if in.WindowTitle != "" {
		hwnd := FindWindowByTitle(in.WindowTitle)
		if hwnd == 0 {
			return staleAll(in.Elements), nil
		}
		state, err := GetWindowState(hwnd)
		if err != nil {
			return staleAll(in.Elements), nil
		}
		if state != nil {
			r := state.Rect
			currentRect = &WindowRect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}
		}
	}

	results := make([]LayoutElementResult, 0, len(in.Elements))
	summary := LayoutSummary{Total: len(in.Elements)}

	for _, el := range in.Elements {
		res := LayoutElementResult{
			ID:            el.ID,
			AdjustedCoord: el.StoredCoord,
			KeywordFound:  false,
		}

		if currentRect != nil && el.WindowRect.Right > el.WindowRect.Left {
			dx := (currentRect.Left + currentRect.Right - el.WindowRect.Left - el.WindowRect.Right) / 2
			dy := (currentRect.Top + currentRect.Bottom - el.WindowRect.Top - el.WindowRect.Bottom) / 2
			res.DriftX = dx
			res.DriftY = dy

			if abs32(dx) > tolerance || abs32(dy) > tolerance {
				res.Confidence = "stale"
				res.Valid = false
				summary.Stale++
				results = append(results, res)
				continue
			}
			if dx != 0 || dy != 0 {
				res.Confidence = "drifted"
				res.AdjustedCoord = Coord{X: el.StoredCoord.X + dx, Y: el.StoredCoord.Y + dy}
			} else {
				res.Confidence = "ok"
			}
		} else {
			res.Confidence = "ok"
		}

		if len(el.SignatureKeywords) > 0 {
			region := el.ExpectedOCRRegion
			if region == nil {
				region = &WindowRect{
					Left: el.StoredCoord.X - 40, Top: el.StoredCoord.Y - 20,
					Right: el.StoredCoord.X + 40, Bottom: el.StoredCoord.Y + 20,
				}
			}
			ocrResult, err := OCRRegion(
				region.Left, region.Top,
				region.Right-region.Left,
				region.Bottom-region.Top,
				in.Language,
			)
			if err == nil && ocrResult != nil {
				for _, kw := range el.SignatureKeywords {
					if containsFold(ocrResult.Text, kw) {
						res.KeywordFound = true
						break
					}
				}
			}
		} else {
			res.KeywordFound = true
		}

		res.Valid = res.KeywordFound
		if res.Valid && res.Confidence == "ok" {
			summary.Valid++
		} else if res.Valid && res.Confidence == "drifted" {
			summary.Drifted++
		} else if !res.Valid && res.Confidence == "ok" {
			res.Confidence = "stale"
			summary.Stale++
		} else if !res.Valid {
			summary.Stale++
		}

		results = append(results, res)
	}

	return &LayoutValidateResult{Elements: results, Summary: summary}, nil
}

func staleAll(elements []LayoutElement) *LayoutValidateResult {
	results := make([]LayoutElementResult, 0, len(elements))
	for _, el := range elements {
		results = append(results, LayoutElementResult{
			ID: el.ID, Valid: false,
			AdjustedCoord: el.StoredCoord, Confidence: "stale",
		})
	}
	return &LayoutValidateResult{
		Elements: results,
		Summary:  LayoutSummary{Total: len(elements), Stale: len(elements)},
	}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return false
	}
	lowerS := toLowerASCII(s)
	lowerSub := toLowerASCII(substr)
	for i := 0; i <= len(lowerS)-len(lowerSub); i++ {
		if lowerS[i:i+len(lowerSub)] == lowerSub {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}
