package versioning

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModelVersion_SaveAndList(t *testing.T) {
	dir := t.TempDir()
	mv := New(dir)

	id, err := mv.SaveCheckpoint(nil, 0.5, 0.7, 100)
	if err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	if id != 0 {
		t.Errorf("expected ID 0, got %d", id)
	}

	id2, err := mv.SaveCheckpoint(nil, 0.3, 0.85, 200)
	if err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}
	if id2 != 1 {
		t.Errorf("expected ID 1, got %d", id2)
	}

	versions := mv.List()
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}

	latest := mv.Latest()
	if latest == nil || latest.ID != 1 {
		t.Errorf("latest should be ID 1, got %v", latest)
	}

	best := mv.Best()
	if best == nil || best.ID != 1 {
		t.Errorf("best should be ID 1 (0.85 accuracy), got %v", best)
	}
}

func TestModelVersion_Rollback(t *testing.T) {
	dir := t.TempDir()
	mv := New(dir)
	mv.SetThreshold(0.05)

	// save v0 with good accuracy
	mv.SaveCheckpoint(nil, 0.5, 0.8, 100)
	// save v1 with slightly worse accuracy (within threshold)
	path := mv.CheckAndRollback(0.6, 0.77)
	if path != "" {
		t.Errorf("should not rollback for 3%% regression, got path: %s", path)
	}

	// save v2 with much worse accuracy (exceeds threshold)
	mv.SaveCheckpoint(nil, 0.7, 0.6, 100)
	path = mv.CheckAndRollback(0.8, 0.6)
	if path == "" {
		t.Error("should rollback for 20%% regression")
	}

	if mv.Rollbacks() != 1 {
		t.Errorf("expected 1 rollback, got %d", mv.Rollbacks())
	}
}

func TestModelVersion_Persistence(t *testing.T) {
	dir := t.TempDir()

	// save versions
	mv1 := New(dir)
	mv1.SaveCheckpoint(nil, 0.5, 0.7, 100)
	mv1.SaveCheckpoint(nil, 0.3, 0.85, 200)

	// reload
	mv2 := New(dir)
	versions := mv2.List()
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions after reload, got %d", len(versions))
	}
	if mv2.Latest().ID != 1 {
		t.Errorf("latest ID should be 1, got %d", mv2.Latest().ID)
	}
}

func TestModelVersion_CleanupOldVersions(t *testing.T) {
	dir := t.TempDir()
	mv := New(dir)

	for i := 0; i < 5; i++ {
		// create dummy model files
		p := filepath.Join(dir, "model_v"+string(rune('0'+i))+".gob")
		os.WriteFile(p, []byte("dummy"), 0644)
		mv.SaveCheckpoint(nil, 0.5, 0.7, 100)
	}

	removed := mv.CleanupOldVersions(2)
	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}
	if len(mv.List()) != 2 {
		t.Errorf("expected 2 versions remaining, got %d", len(mv.List()))
	}
}

func TestModelVersion_EmptyList(t *testing.T) {
	dir := t.TempDir()
	mv := New(dir)

	if mv.Latest() != nil {
		t.Error("Latest should be nil for empty list")
	}
	if mv.Best() != nil {
		t.Error("Best should be nil for empty list")
	}
	if len(mv.List()) != 0 {
		t.Error("List should be empty for empty version list")
	}
}
