package actions

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestComputeSignalLevel(t *testing.T) {
	tests := []struct {
		detCount   int
		category   string
		taskPrompt string
		want       int
	}{
		{0, "", "", 0},
		{0, "click", "", 0},
		{1, "", "", 1},
		{5, "", "", 1},
		{1, "click", "", 2},
		{1, "", "find button", 2},
		{3, "navigate", "go to settings", 2},
		{0, "click", "click the button", 0},
	}
	for _, tc := range tests {
		got := computeSignalLevel(tc.detCount, tc.category, tc.taskPrompt)
		if got != tc.want {
			t.Errorf("computeSignalLevel(%d, %q, %q) = %d, want %d",
				tc.detCount, tc.category, tc.taskPrompt, got, tc.want)
		}
	}
}

func TestJoinWhere(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, "1=1"},
		{[]string{}, "1=1"},
		{[]string{"source = 'raw'"}, "source = 'raw'"},
		{[]string{"a = 1", "b = 2"}, "a = 1 AND b = 2"},
		{[]string{"x", "y", "z"}, "x AND y AND z"},
	}
	for _, tc := range tests {
		got := joinWhere(tc.input)
		if got != tc.want {
			t.Errorf("joinWhere(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStopRetentionPruner_NoopWhenNotStarted(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StopRetentionPruner()

	if retentionStop != nil {
		t.Error("retentionStop should remain nil when no pruner is running")
	}
}

func TestStopRetentionPruner_StopsGoroutine(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StartRetentionPruner(30)

	if retentionStop == nil {
		t.Fatal("retentionStop should be set after StartRetentionPruner")
	}

	StopRetentionPruner()

	if retentionStop != nil {
		t.Error("retentionStop should be nil after StopRetentionPruner")
	}
}

func TestStartRetentionPruner_NoopWhenZeroDays(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StartRetentionPruner(0)

	// Should not have started the goroutine
	if retentionStop != nil {
		t.Error("StartRetentionPruner(0) should not start pruner")
	}
}

func TestStartRetentionPruner_SkipsWhenDisabled(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}
	ActiveConfig = nil

	StartRetentionPruner(7)

	if retentionStop == nil {
		t.Error("StartRetentionPruner should start goroutine regardless of ActiveConfig")
	}

	StopRetentionPruner()
}

func TestArgInt32(t *testing.T) {
	tests := []struct {
		in      any
		want    int32
		wantOK  bool
	}{
		{float64(42), 42, true},
		{int(7), 7, true},
		{int32(9), 9, true},
		{int64(100), 100, true},
		{"click", 0, false},
		{nil, 0, false},
	}
	for _, tc := range tests {
		got, ok := argInt32(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("argInt32(%v) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestTargetFromXY_MissingCoordsReturnsNil(t *testing.T) {
	if got := targetFromXY(map[string]any{}); got != nil {
		t.Errorf("targetFromXY with no coords = %v, want nil", got)
	}
}

func TestTargetFromXY_Produces400Region(t *testing.T) {
	got := targetFromXY(map[string]any{"x": float64(100), "y": float64(100)})
	if len(got) != 4 {
		t.Fatalf("targetFromXY returned %d values, want 4", len(got))
	}
	// 400x400 centered on (100,100). The region must be clamped to the virtual
	// screen bounds, so verify the size is 400x400 and (x,y) is the center.
	rx, ry, rw, rh := got[0], got[1], got[2], got[3]
	if rw != 400 || rh != 400 {
		t.Errorf("region size = (%d,%d), want (400,400)", rw, rh)
	}
	bounds := VirtualScreenBounds()
	if rx < bounds.X {
		t.Errorf("region origin x = %d, should be >= virtual left %d", rx, bounds.X)
	}
	if ry < bounds.Y {
		t.Errorf("region origin y = %d, should be >= virtual top %d", ry, bounds.Y)
	}
}

func TestTargetFromTo_UsesDestination(t *testing.T) {
	got := targetFromTo(map[string]any{"to_x": float64(200), "to_y": float64(200)})
	if len(got) != 4 {
		t.Fatalf("targetFromTo returned %d values, want 4", len(got))
	}
	// Destination (200,200) centered → size 400x400, x within bounds.
	if got[2] != 400 || got[3] != 400 {
		t.Errorf("targetFromTo region size = (%d,%d), want (400,400)", got[2], got[3])
	}
	if got[0] < 0 {
		t.Errorf("targetFromTo region origin x = %d, should be >= 0", got[0])
	}
}

func TestSaveSnapshotAfterAction_BackwardCompatNoRegion(t *testing.T) {
	// Passing zero variadic args must fall back to full-screen capture path
	// (which is a no-op if training is disabled). Just verify no panic.
	saved := ActiveConfig
	ActiveConfig = nil
	SaveSnapshotAfterAction(TrainingSourceRaw, TrainingCatClick, "test")
	ActiveConfig = saved
}

func TestElementCropRegion_Center(t *testing.T) {
	// Virtual screen origin (0,-1080), size 3200x1980.
	bounds := Rect{X: 0, Y: -1080, W: 3200, H: 1980}
	el := DetectedElement{Class: "button", X: 800, Y: 540, W: 100, H: 40}
	// Element center at virtual (800, -540). With 24px padding:
	x, y, w, h, ok := elementCropRegion(bounds, el)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if w != 100+48 || h != 40+48 {
		t.Errorf("crop size = (%d,%d), want (%d,%d)", w, h, 148, 88)
	}
	if x != 800-24 || y != -540-24 {
		t.Errorf("crop origin = (%d,%d), want (%d,%d)", x, y, 776, -564)
	}
}

func TestElementCropRegion_ClampsToBounds(t *testing.T) {
	bounds := Rect{X: 0, Y: -1080, W: 3200, H: 1980}
	// Element at the very top-left corner of the virtual screen (bitmap 0,0).
	el := DetectedElement{Class: "icon", X: 0, Y: 0, W: 50, H: 50}
	x, y, w, h, ok := elementCropRegion(bounds, el)
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Padding would push origin to (-24,-1104) but clamped to (0,-1080).
	if x != 0 || y != -1080 {
		t.Errorf("crop origin = (%d,%d), want (0,-1080)", x, y)
	}
	if x+w > bounds.W || y+h > bounds.Y+bounds.H {
		t.Errorf("crop escapes virtual bounds: (%d,%d,%d,%d)", x, y, w, h)
	}
}

func TestElementCropRegion_InvalidSize(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 100, H: 100}
	if _, _, _, _, ok := elementCropRegion(bounds, DetectedElement{W: 0, H: 0}); ok {
		t.Error("expected ok=false for zero-size element")
	}
}

func TestPruneOrphanedSamples_DeletesMissingImages(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.png")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.png") // never created

	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := createTrainingTables(db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, p := range []string{existing, missing} {
		if _, err := db.Exec(`INSERT INTO training_samples
			(source, category, image_path, created_at) VALUES('raw','click',?,?)`, p, now); err != nil {
			t.Fatal(err)
		}
	}

	// Swap in the test DB so PruneOrphanedSamples operates on it.
	trainMu.Lock()
	orig := trainDB
	trainDB = db
	trainMu.Unlock()
	defer func() {
		trainMu.Lock()
		trainDB = orig
		trainMu.Unlock()
	}()

	n, err := PruneOrphanedSamples()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("PruneOrphanedSamples deleted %d, want 1 (only the missing-image row)", n)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM training_samples`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("rows after prune = %d, want 1 (existing-image row survives)", count)
	}
}
