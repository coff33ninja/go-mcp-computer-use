package dataloader

import "testing"

func TestNormalizeCommandJSON_ProductionNestedStringArgs(t *testing.T) {
	cmd := `{"args":"{\"X\":700,\"Y\":400,\"Button\":\"left\",\"Clicks\":1}","tool":"click"}`
	tool, argsJSON, x, y, _, _, _, _, ok := NormalizeCommandJSON(cmd)
	if !ok {
		t.Fatal("expected ok")
	}
	if tool != "click" {
		t.Fatalf("tool=%q", tool)
	}
	if x != 700 || y != 400 {
		t.Fatalf("coords=(%d,%d) want (700,400)", x, y)
	}
	if argsJSON == "" || argsJSON[0] != '{' {
		t.Fatalf("argsJSON=%q", argsJSON)
	}
	// nested object form
	cmd2 := `{"tool":"click","args":{"X":10,"Y":20}}`
	_, _, x2, y2, _, _, _, _, ok2 := NormalizeCommandJSON(cmd2)
	if !ok2 || x2 != 10 || y2 != 20 {
		t.Fatalf("object args coords=(%d,%d) ok=%v", x2, y2, ok2)
	}
	// lowercase already-normalized
	cmd3 := `{"tool":"click","args":{"x":5,"y":6}}`
	_, _, x3, y3, _, _, _, _, _ := NormalizeCommandJSON(cmd3)
	if x3 != 5 || y3 != 6 {
		t.Fatalf("lowercase coords=(%d,%d)", x3, y3)
	}
}

func TestApplyNormalization_FillsSampleCoords(t *testing.T) {
	s := Sample{Context: "OCR", Action: "click"}
	cmd := `{"args":"{\"X\":2400,\"Y\":2}","tool":"click"}`
	ApplyNormalization(&s, cmd)
	if s.Action != "click" {
		t.Fatalf("action=%q", s.Action)
	}
	if s.CoordX != 2400 || s.CoordY != 2 {
		t.Fatalf("coord=(%d,%d)", s.CoordX, s.CoordY)
	}
}
