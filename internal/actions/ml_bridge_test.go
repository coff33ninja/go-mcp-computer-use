package actions

import (
	"testing"
)

func TestNewMLEngine(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	if engine == nil {
		t.Fatal("NewMLEngine returned nil")
	}
	if engine.IsReady() {
		t.Error("expected not ready before training")
	}
}

func TestMLEngine_Predict_NotReady(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	preds := engine.Predict("hello world", 5, nil)
	if preds != nil {
		t.Error("expected nil predictions when not ready")
	}
}

func TestMLEngine_Reset(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	engine.Reset()
	if engine.IsReady() {
		t.Error("expected not ready after reset")
	}
}

func TestMLEngine_LoadModel_NoFile(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	err := engine.LoadModel()
	if err == nil {
		t.Error("expected error loading non-existent model")
	}
}

func TestMLEngine_Train_InsufficientData(t *testing.T) {
	dir := t.TempDir()
	// create empty datalog.db (no training data)
	engine := NewMLEngine(dir)
	err := engine.Train()
	if err != nil {
		t.Errorf("expected nil for insufficient data, got: %v", err)
	}
	if engine.IsReady() {
		t.Error("expected not ready with insufficient data")
	}
}

func TestMLEngine_Predict_LimitZero(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	preds := engine.Predict("test", 0, nil)
	if preds != nil {
		t.Error("expected nil predictions when not ready")
	}
}

func TestMLEngine_ConcurrentPredict(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_ = engine.Predict("test text", 3, nil)
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestNormalizeAppName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Google Chrome", "google_chrome"},
		{"Visual Studio Code", "visual_studio_code"},
		{"Minecraft", "minecraft"},
		{"  lots   of   spaces  ", "lots_of_spaces"},
		{"UPPER CASE", "upper_case"},
		{"ctrl+c", "ctrl_c"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeAppName(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeAppName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMLEngine_FinetuneForApp_NoData(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	_, err := engine.FinetuneForApp("TestApp", 1)
	if err == nil {
		t.Error("expected error for empty app name or no data")
	}
}

func TestMLEngine_FinetuneForApp_EmptyName(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	_, err := engine.FinetuneForApp("", 1)
	if err == nil {
		t.Error("expected error for empty app name")
	}
}

func TestMLEngine_PredictForApp_NoAppModel(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	// no base model either, so should return nil
	preds := engine.PredictForApp("test", "SomeApp", 5, nil)
	if preds != nil {
		t.Error("expected nil when no model is ready")
	}
}

func TestMLEngine_LoadAppModels_NoDir(t *testing.T) {
	dir := t.TempDir()
	engine := NewMLEngine(dir)
	count := engine.LoadAppModels()
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestPredictedArgs_InPredictedAction(t *testing.T) {
	pa := PredictedAction{
		Command:    "scroll",
		Confidence: 0.8,
		Args: &PredictedArgs{
			ScrollDir: "down",
		},
	}
	if pa.Args == nil || pa.Args.ScrollDir != "down" {
		t.Error("PredictedArgs not set correctly")
	}
}
