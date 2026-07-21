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
