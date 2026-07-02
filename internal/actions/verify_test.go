package actions

import (
	"encoding/json"
	"testing"
)

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestSmartRegionAround(t *testing.T) {
	rx, ry, rw, rh := SmartRegionAround(500, 500, 400)
	if rw == 0 || rh == 0 {
		t.Errorf("expected non-zero region, got %dx%d", rw, rh)
	}
	if rx < 0 || ry < 0 {
		t.Errorf("region should be clamped to screen, got (%d,%d)", rx, ry)
	}
}

func TestSmartRegionAroundCenter(t *testing.T) {
	screenW, screenH := ScreenSize()
	cx, cy := screenW/2, screenH/2
	rx, ry, rw, rh := SmartRegionAround(cx, cy, 400)
	expectedCX := rx + rw/2
	expectedCY := ry + rh/2
	// Center should be roughly at the click point
	if absInt(int(expectedCX-cx)) > 2 || absInt(int(expectedCY-cy)) > 2 {
		t.Errorf("region not centered at (%d,%d), got center (%d,%d)", cx, cy, expectedCX, expectedCY)
	}
}

func TestSmartRegionAroundEdge(t *testing.T) {
	rx, ry, rw, rh := SmartRegionAround(0, 0, 400)
	if rx != 0 || ry != 0 {
		t.Errorf("edge region should start at (0,0), got (%d,%d)", rx, ry)
	}
	if rw <= 0 || rh <= 0 {
		t.Errorf("edge region should be positive, got %dx%d", rw, rh)
	}
}

func TestComputeTextDiffIdentical(t *testing.T) {
	d := computeTextDiff("hello\nworld", "hello\nworld")
	if d.TotalChangeRatio != 0 {
		t.Errorf("identical text should have 0 change ratio, got %f", d.TotalChangeRatio)
	}
	if len(d.LinesAdded)+len(d.LinesRemoved) != 0 {
		t.Errorf("identical text should have no changes")
	}
}

func TestComputeTextDiffChanged(t *testing.T) {
	d := computeTextDiff("hello\nworld", "hello\nmars")
	if d.TotalChangeRatio <= 0 {
		t.Errorf("changed text should have positive change ratio")
	}
}

func TestComputeTextDiffAdded(t *testing.T) {
	d := computeTextDiff("hello", "hello\nworld")
	if len(d.LinesAdded) != 1 || d.LinesAdded[0] != "world" {
		t.Errorf("expected one added line 'world', got %v", d.LinesAdded)
	}
}

func TestComputeTextDiffRemoved(t *testing.T) {
	d := computeTextDiff("hello\nworld", "hello")
	if len(d.LinesRemoved) != 1 {
		t.Errorf("expected one removed line, got %v", d.LinesRemoved)
	}
}

func TestVerifyResultJSON(t *testing.T) {
	vr := &VerifyResult{
		Passed: true,
		Method: "expected_text",
		Reason: "expected text Submit appeared",
		Diff:   &TextDiff{LinesChanged: 1, TotalChangeRatio: 0.5},
	}
	out, err := json.Marshal(vr)
	if err != nil {
		t.Fatalf("json marshal VerifyResult: %v", err)
	}
	var back VerifyResult
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("json unmarshal VerifyResult: %v", err)
	}
	if back.Passed != true {
		t.Errorf("round-trip: expected Passed=true")
	}
}

func TestVerifyActionExpectedText(t *testing.T) {
	// With an impossible expected text, verification should fail
	cfg := &VerifyConfig{
		ExpectedText: "!@#$%^&*_NONEXISTENT_TEXT_!@#$%^&*",
		AfterWaitMs:  100,
	}
	vr := VerifyAction(cfg)
	if vr.Passed {
		t.Errorf("expected non-existent text to fail verification, got passed=true, reason=%s", vr.Reason)
	}
	if vr.Method != "expected_text" {
		t.Errorf("expected method=expected_text, got %s", vr.Method)
	}
	if vr.BeforeOCR == nil || vr.AfterOCR == nil {
		t.Errorf("expected before and after OCR to be captured")
	}
}

func TestVerifyActionNoChange(t *testing.T) {
	// Without expected text and with minimal wait, should detect as ocr_diff
	cfg := &VerifyConfig{
		AfterWaitMs: 100,
	}
	vr := VerifyAction(cfg)
	if vr.Method != "ocr_diff" {
		t.Errorf("expected method=ocr_diff, got %s", vr.Method)
	}
}

func TestChainVerifyStepJSON(t *testing.T) {
	step := ChainStep{
		Type: "verify",
		Tool: "click",
		Args: map[string]any{"x": 100, "y": 200},
		Verify: &VerifyStepConfig{
			Expected: &ExpConfig{Text: "Submit", WaitMs: 1000},
			Retries:  2,
			WaitMs:   500,
		},
		Capture: "result",
	}
	out, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("marshal chain step: %v", err)
	}
	var back ChainStep
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal chain step: %v", err)
	}
	if back.Type != "verify" {
		t.Errorf("expected type=verify, got %q", back.Type)
	}
	if back.Tool != "click" {
		t.Errorf("expected tool=click, got %q", back.Tool)
	}
	if back.Verify == nil {
		t.Fatal("expected Verify config to survive round-trip")
	}
	if back.Verify.Expected == nil || back.Verify.Expected.Text != "Submit" {
		t.Errorf("expected Expected.Text=Submit, got %v", back.Verify.Expected)
	}
	if back.Verify.Retries != 2 {
		t.Errorf("expected Retries=2, got %d", back.Verify.Retries)
	}
}

func TestChainDetectVerifyType(t *testing.T) {
	s := ChainStep{
		Tool:   "click",
		Args:   map[string]any{"x": 1, "y": 2},
		Verify: &VerifyStepConfig{},
	}
	typ := detectStepType(s)
	if typ != StepVerify {
		t.Errorf("detectStepType should return StepVerify when Verify is set, got %q", typ)
	}
}

func TestExpConfigJSON(t *testing.T) {
	ec := &ExpConfig{
		Text:   "Submit",
		WaitMs: 1000,
	}
	out, err := json.Marshal(ec)
	if err != nil {
		t.Fatalf("json marshal ExpConfig: %v", err)
	}
	var back ExpConfig
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("json unmarshal ExpConfig: %v", err)
	}
	if back.Text != "Submit" {
		t.Errorf("round-trip: expected Text=Submit")
	}
}
