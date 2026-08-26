package actions

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestTextLocDB(t *testing.T) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		textLocMu.Lock()
		textLocDB = nil
		textLocMu.Unlock()
	})
	if err := createTextLocationTables(db); err != nil {
		t.Fatal(err)
	}
	textLocMu.Lock()
	textLocDB = db
	textLocMu.Unlock()
}

func insertOldTextLocation(t *testing.T, content, window string, hitCount int, daysOld int) {
	t.Helper()
	insertOldTextLocationZOrder(t, content, window, hitCount, daysOld, 0)
}

func insertOldTextLocationZOrder(t *testing.T, content, window string, hitCount int, daysOld int, zOrder int) {
	t.Helper()
	th := textHash(content)
	ts := time.Now().AddDate(0, 0, -daysOld).UTC().Format(time.RFC3339)
	textLocMu.Lock()
	_, err := textLocDB.Exec(
		`INSERT INTO text_locations (text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count)
		 VALUES (?, ?, ?, 0, 0, 10, 10, ?, 1.0, ?, ?)`,
		th, content, window, zOrder, ts, hitCount,
	)
	textLocMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPruneTextLocations_NilDB(t *testing.T) {
	textLocMu.Lock()
	textLocDB = nil
	textLocMu.Unlock()

	got := PruneTextLocations(30)
	if got != 0 {
		t.Errorf("PruneTextLocations(nil DB) = %d, want 0", got)
	}
}

func TestPruneTextLocations_InvalidMaxAge(t *testing.T) {
	setupTestTextLocDB(t)

	got := PruneTextLocations(0)
	if got != 0 {
		t.Errorf("PruneTextLocations(0) = %d, want 0", got)
	}
	got = PruneTextLocations(-5)
	if got != 0 {
		t.Errorf("PruneTextLocations(-5) = %d, want 0", got)
	}
}

func TestPruneTextLocations_PrunesOldSingleHit(t *testing.T) {
	setupTestTextLocDB(t)
	insertOldTextLocation(t, "old text", "TestWindow", 1, 31)

	n := PruneTextLocations(30)
	if n != 1 {
		t.Errorf("PruneTextLocations(30) = %d, want 1", n)
	}

	loc := FindTextLocation("old text", "TestWindow")
	if loc != nil {
		t.Error("expected old text to be pruned, but it still exists")
	}
}

func TestPruneTextLocations_KeepsRecent(t *testing.T) {
	setupTestTextLocDB(t)
	insertOldTextLocation(t, "recent text", "TestWindow", 1, 1)

	n := PruneTextLocations(30)
	if n != 0 {
		t.Errorf("PruneTextLocations(30) = %d, want 0 (recent entry should survive)", n)
	}

	loc := FindTextLocation("recent text", "TestWindow")
	if loc == nil {
		t.Error("expected recent text to survive, but it was pruned")
	}
}

func TestPruneTextLocations_KeepsHighHitCount(t *testing.T) {
	setupTestTextLocDB(t)
	insertOldTextLocation(t, "popular text", "TestWindow", 5, 31)

	n := PruneTextLocations(30)
	if n != 0 {
		t.Errorf("PruneTextLocations(30) = %d, want 0 (high hit_count should survive)", n)
	}

	loc := FindTextLocation("popular text", "TestWindow")
	if loc == nil {
		t.Error("expected popular text to survive, but it was pruned")
	}
}

func TestPruneTextLocations_MixedEntries(t *testing.T) {
	setupTestTextLocDB(t)

	insertOldTextLocation(t, "old single-hit", "TestWindow", 1, 60)
	insertOldTextLocation(t, "old multi-hit", "TestWindow", 5, 60)
	insertOldTextLocation(t, "recent single-hit", "TestWindow", 1, 1)
	insertOldTextLocation(t, "recent multi-hit", "TestWindow", 3, 1)

	n := PruneTextLocations(30)
	if n != 1 {
		t.Errorf("PruneTextLocations(30) = %d, want 1 (only old single-hit pruned)", n)
	}

	for _, want := range []string{"old multi-hit", "recent single-hit", "recent multi-hit"} {
		if FindTextLocation(want, "TestWindow") == nil {
			t.Errorf("expected %q to survive, but it was pruned", want)
		}
	}
	if FindTextLocation("old single-hit", "TestWindow") != nil {
		t.Error("expected 'old single-hit' to be pruned, but it still exists")
	}
}

func TestFindTextLocationMatch_PrefersZOrder(t *testing.T) {
	setupTestTextLocDB(t)
	insertOldTextLocationZOrder(t, "layered text", "TestWindow", 5, 1, 2)
	insertOldTextLocationZOrder(t, "layered text", "TestWindow", 3, 1, 0)

	// z_order=0 should prefer the z_order=0 entry
	loc := FindTextLocationMatch("layered text", "TestWindow", 0)
	if loc == nil {
		t.Fatal("FindTextLocationMatch returned nil")
	}
	if loc.ZOrder != 0 {
		t.Errorf("expected z_order=0, got %d", loc.ZOrder)
	}

	// z_order=2 should prefer the z_order=2 entry
	loc = FindTextLocationMatch("layered text", "TestWindow", 2)
	if loc == nil {
		t.Fatal("FindTextLocationMatch returned nil")
	}
	if loc.ZOrder != 2 {
		t.Errorf("expected z_order=2, got %d", loc.ZOrder)
	}

	// z_order=5 (no exact match) should fall back to any
	loc = FindTextLocationMatch("layered text", "TestWindow", 5)
	if loc == nil {
		t.Fatal("FindTextLocationMatch fallback returned nil")
	}
}

func TestFindTextLocationAnyMatch_PrefersZOrder(t *testing.T) {
	setupTestTextLocDB(t)
	insertOldTextLocationZOrder(t, "any layered", "WindowA", 5, 1, 1)
	insertOldTextLocationZOrder(t, "any layered", "WindowB", 3, 1, 0)

	loc := FindTextLocationAnyMatch("any layered", 1)
	if loc == nil {
		t.Fatal("FindTextLocationAnyMatch returned nil")
	}
	if loc.ZOrder != 1 {
		t.Errorf("expected z_order=1, got %d", loc.ZOrder)
	}
}
