package dataloader

import (
	"context"
	"testing"
)

func TestLoadAll_EmptyDB(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("expected 0 samples from empty DB, got %d", len(samples))
	}
}

func TestCount_EmptyDB(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	count, err := loader.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 count from empty DB, got %d", count)
	}
}

func TestStats_EmptyDB(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	stats, err := loader.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats from empty DB, got %v", stats)
	}
}

func TestLoadRecent_ZeroN(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	samples, err := loader.LoadRecent(ctx, 0)
	if err != nil {
		t.Fatalf("LoadRecent failed: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("expected 0 samples for n=0, got %d", len(samples))
	}
}

func TestLoadByTool_NoMatch(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	samples, err := loader.LoadByTool(ctx, "nonexistent_tool")
	if err != nil {
		t.Fatalf("LoadByTool failed: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("expected 0 samples for nonexistent tool, got %d", len(samples))
	}
}

func TestLoadAll_WithContext(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := loader.LoadAll(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestNewSQLiteLoader_InMemory(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestLoadRecent_NegativeN(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	samples, err := loader.LoadRecent(ctx, -1)
	if err != nil {
		t.Fatalf("LoadRecent failed: %v", err)
	}
	if len(samples) != 0 {
		t.Errorf("expected 0 samples for negative n, got %d", len(samples))
	}
}

func TestCount_Concurrent(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	ctx := context.Background()
	// run multiple counts concurrently
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := loader.Count(ctx)
			done <- err
		}()
	}
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent Count failed: %v", err)
		}
	}
}
