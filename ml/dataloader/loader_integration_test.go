package dataloader

import (
	"context"
	"testing"
)

func setupLoaderWithData(t *testing.T) *SQLiteLoader {
	t.Helper()
	loader := NewSQLiteLoader(":memory:")
	db, err := loader.openDB()
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('click Submit button', '{"tool":"click","args":"{\"x\":100,\"y\":200}"}', 'Test Window', 1, '2026-07-21T10:00:00Z')`)
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('hover Cancel', '{"tool":"hover","args":"{\"x\":50,\"y\":50}"}', 'Test Window', 1, '2026-07-21T10:01:00Z')`)
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('type text filename', 'type_text', 'Test Window', 0, '2026-07-21T10:02:00Z')`)
	return loader
}

func TestLoadAll_WithInsertedData(t *testing.T) {
	loader := setupLoaderWithData(t)
	ctx := context.Background()
	samples, err := loader.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(samples))
	}
}

func TestCount_WithData(t *testing.T) {
	loader := setupLoaderWithData(t)
	count, err := loader.Count(context.Background())
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestStats_WithData(t *testing.T) {
	loader := setupLoaderWithData(t)
	stats, err := loader.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats["click"] != 1 {
		t.Errorf("expected click=1, got %d", stats["click"])
	}
	if stats["hover"] != 1 {
		t.Errorf("expected hover=1, got %d", stats["hover"])
	}
	if stats["type_text"] != 1 {
		t.Errorf("expected type_text=1, got %d", stats["type_text"])
	}
}

func TestLoadByTool_WithData(t *testing.T) {
	loader := setupLoaderWithData(t)
	samples, err := loader.LoadByTool(context.Background(), "click")
	if err != nil {
		t.Fatalf("LoadByTool failed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 click sample, got %d", len(samples))
	}
	if samples[0].Action != "click" {
		t.Errorf("expected action=click, got %q", samples[0].Action)
	}
}

func TestLoadRecent_WithData(t *testing.T) {
	loader := setupLoaderWithData(t)
	samples, err := loader.LoadRecent(context.Background(), 2)
	if err != nil {
		t.Fatalf("LoadRecent failed: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 recent samples, got %d", len(samples))
	}
	// most recent first (ordered by id DESC)
	if samples[0].Context != "type text filename" {
		t.Errorf("expected most recent first, got %q", samples[0].Context)
	}
}

func TestQueryDB_PlainTextTool(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	db, _ := loader.openDB()
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, success, created_at)
		VALUES ('click button', 'click', 1, '2026-07-21')`)

	samples, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(samples))
	}
	if samples[0].Action != "click" {
		t.Errorf("expected action=click, got %q", samples[0].Action)
	}
}

func TestLoadAll_ParsesCommandJSON(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	db, _ := loader.openDB()
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, window_title, success, created_at)
		VALUES ('click Submit', '{"tool":"click","args":"{\"x\":100,\"y\":200}"}', 'App', 1, '2026-07-21')`)

	samples, _ := loader.LoadAll(context.Background())
	if len(samples) != 1 {
		t.Fatal("expected 1 sample")
	}
	s := samples[0]
	if s.Action != "click" {
		t.Errorf("action: got %q", s.Action)
	}
	if s.Context != "click Submit" {
		t.Errorf("context: got %q", s.Context)
	}
	if s.WindowTitle != "App" {
		t.Errorf("window: got %q", s.WindowTitle)
	}
	if !s.Success {
		t.Error("expected success=true")
	}
}

func TestLoadAll_FailedActions(t *testing.T) {
	loader := NewSQLiteLoader(":memory:")
	db, _ := loader.openDB()
	db.Exec(`INSERT INTO training_pairs (ocr_before_text, command_json, success, created_at)
		VALUES ('click', '{"tool":"click"}', 0, '2026-07-21')`)

	samples, _ := loader.LoadAll(context.Background())
	if len(samples) != 1 {
		t.Fatal("expected 1 sample")
	}
	if samples[0].Success {
		t.Error("expected success=false")
	}
}
