package actions

import (
	"database/sql"
	"testing"
	"unsafe"
)

// setupTestDB replaces dlogDB with a fresh in-memory SQLite database for
// the duration of a test, then restores the original.
func setupTestDB(t *testing.T) {
	t.Helper()
	dlogMu.Lock()
	defer dlogMu.Unlock()
	origDB := dlogDB
	// Reset dlogOnce so InitDataLog can run again if needed.
	// unsafe.Pointer avoids the sync.Once copy vet warning.
	*(*[2]uint64)(unsafe.Pointer(&dlogOnce)) = [2]uint64{}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := createDataLogTables(db); err != nil {
		db.Close()
		t.Fatalf("create tables: %v", err)
	}
	dlogDB = db
	t.Cleanup(func() {
		dlogMu.Lock()
		defer dlogMu.Unlock()
		dlogDB.Close()
		dlogDB = origDB
	})
}

// --- nearbyOCRText tests ---

func TestNearbyOCRTextWordsWithinRadius(t *testing.T) {
	words := []OCRWord{
		{Text: "Hello", X: 100, Y: 100, W: 50, H: 20},
		{Text: "World", X: 200, Y: 200, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 120, 110, 50, 10)
	if got != "Hello" {
		t.Errorf("nearbyOCRText() = %q, want %q", got, "Hello")
	}
}

func TestNearbyOCRTextWordsOutsideRadius(t *testing.T) {
	words := []OCRWord{
		{Text: "Far", X: 500, Y: 500, W: 50, H: 20},
		{Text: "AlsoFar", X: 600, Y: 600, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 100, 100, 30, 10)
	if got != "" {
		t.Errorf("nearbyOCRText() = %q, want empty", got)
	}
}

func TestNearbyOCRTextSortedByDistance(t *testing.T) {
	words := []OCRWord{
		{Text: "Far", X: 300, Y: 300, W: 50, H: 20},
		{Text: "Near", X: 110, Y: 110, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 120, 120, 300, 10)
	if got != "Near Far" {
		t.Errorf("nearbyOCRText() = %q, want %q", got, "Near Far")
	}
}

func TestNearbyOCRTextMaxWordsCap(t *testing.T) {
	words := []OCRWord{
		{Text: "A", X: 100, Y: 100, W: 10, H: 10},
		{Text: "B", X: 110, Y: 100, W: 10, H: 10},
		{Text: "C", X: 200, Y: 200, W: 10, H: 10},
		{Text: "D", X: 300, Y: 300, W: 10, H: 10},
	}
	got := nearbyOCRText(words, 115, 100, 500, 2)
	// All 4 within radius. Sort by distance: B=5.0, A=11.18, C=..., D=...
	// Cap at 2 keeps B and A (nearest first).
	if got != "B A" {
		t.Errorf("nearbyOCRText(maxWords=2) = %q, want %q", got, "B A")
	}
}

func TestNearbyOCRTextDedup(t *testing.T) {
	words := []OCRWord{
		{Text: "Submit", X: 100, Y: 100, W: 50, H: 20},
		{Text: "Submit", X: 110, Y: 105, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 120, 110, 200, 10)
	if got != "Submit" {
		t.Errorf("nearbyOCRText() = %q, want %q", got, "Submit")
	}
}

func TestNearbyOCRTextEmptyWords(t *testing.T) {
	got := nearbyOCRText(nil, 100, 100, 50, 10)
	if got != "" {
		t.Errorf("nearbyOCRText(nil) = %q, want empty", got)
	}
}

func TestNearbyOCRTextTrimsWhitespace(t *testing.T) {
	words := []OCRWord{
		{Text: "  hello  ", X: 100, Y: 100, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 120, 110, 50, 10)
	if got != "hello" {
		t.Errorf("nearbyOCRText() = %q, want %q", got, "hello")
	}
}

func TestNearbyOCRTextSkipsBlankAfterTrim(t *testing.T) {
	words := []OCRWord{
		{Text: "   ", X: 100, Y: 100, W: 50, H: 20},
		{Text: "real", X: 110, Y: 100, W: 50, H: 20},
	}
	got := nearbyOCRText(words, 120, 110, 200, 10)
	if got != "real" {
		t.Errorf("nearbyOCRText() = %q, want %q", got, "real")
	}
}

// --- capAndDedupeText tests ---

func TestCapAndDedupeTextBasic(t *testing.T) {
	got := capAndDedupeText("hello world foo bar", 3)
	if got != "hello world foo" {
		t.Errorf("capAndDedupeText() = %q, want %q", got, "hello world foo")
	}
}

func TestCapAndDedupeTextCaseInsensitiveDedup(t *testing.T) {
	got := capAndDedupeText("Hello hello HELLO", 10)
	if got != "Hello" {
		t.Errorf("capAndDedupeText() = %q, want %q", got, "Hello")
	}
}

func TestCapAndDedupeTextMaxWords(t *testing.T) {
	got := capAndDedupeText("a b c d e", 3)
	if got != "a b c" {
		t.Errorf("capAndDedupeText(max=3) = %q, want %q", got, "a b c")
	}
}

func TestCapAndDedupeTextEmptyInput(t *testing.T) {
	got := capAndDedupeText("", 5)
	if got != "" {
		t.Errorf("capAndDedupeText(empty) = %q, want empty", got)
	}
}

func TestCapAndDedupeTextWhitespaceOnly(t *testing.T) {
	got := capAndDedupeText("   ", 5)
	if got != "" {
		t.Errorf("capAndDedupeText(whitespace) = %q, want empty", got)
	}
}

func TestCapAndDedupeTextPreservesFirstSeenOrder(t *testing.T) {
	got := capAndDedupeText("c a b a c b", 10)
	if got != "c a b" {
		t.Errorf("capAndDedupeText() = %q, want %q", got, "c a b")
	}
}

func TestCapAndDedupeTextMaxOne(t *testing.T) {
	got := capAndDedupeText("first second third", 1)
	if got != "first" {
		t.Errorf("capAndDedupeText(max=1) = %q, want %q", got, "first")
	}
}

// --- SaveAdaptiveStat / LoadPersistedStats round-trip ---

func TestSaveAndLoadAdaptiveStatsRoundTrip(t *testing.T) {
	setupTestDB(t)

	SaveAdaptiveStat("click", 50.0, true)
	SaveAdaptiveStat("click", 70.0, true)
	SaveAdaptiveStat("click", 90.0, false)
	SaveAdaptiveStat("type", 30.0, true)

	stats, err := LoadPersistedStats()
	if err != nil {
		t.Fatalf("LoadPersistedStats: %v", err)
	}

	click, ok := stats["click"]
	if !ok {
		t.Fatal("expected 'click' in loaded stats")
	}
	if click.SuccessCount != 2 {
		t.Errorf("click.SuccessCount = %d, want 2", click.SuccessCount)
	}
	if click.FailCount != 1 {
		t.Errorf("click.FailCount = %d, want 1", click.FailCount)
	}
	if click.DurationCount != 3 {
		t.Errorf("click.DurationCount = %d, want 3", click.DurationCount)
	}
	if click.DurationSum != 210.0 {
		t.Errorf("click.DurationSum = %f, want 210.0", click.DurationSum)
	}
	if click.DurationMin != 50.0 {
		t.Errorf("click.DurationMin = %f, want 50.0", click.DurationMin)
	}
	if click.DurationMax != 90.0 {
		t.Errorf("click.DurationMax = %f, want 90.0", click.DurationMax)
	}

	type_, ok := stats["type"]
	if !ok {
		t.Fatal("expected 'type' in loaded stats")
	}
	if type_.SuccessCount != 1 {
		t.Errorf("type.SuccessCount = %d, want 1", type_.SuccessCount)
	}
	if type_.DurationSum != 30.0 {
		t.Errorf("type.DurationSum = %f, want 30.0", type_.DurationSum)
	}
}

func TestSaveAdaptiveStatEmptyToolIgnored(t *testing.T) {
	setupTestDB(t)

	SaveAdaptiveStat("", 100.0, true)

	stats, err := LoadPersistedStats()
	if err != nil {
		t.Fatalf("LoadPersistedStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected no stats after SaveAdaptiveStat with empty tool, got %d", len(stats))
	}
}

func TestSaveAdaptiveStatMinMaxAccumulation(t *testing.T) {
	setupTestDB(t)

	SaveAdaptiveStat("scroll", 100.0, true)
	SaveAdaptiveStat("scroll", 200.0, true)
	SaveAdaptiveStat("scroll", 50.0, true)
	SaveAdaptiveStat("scroll", 150.0, true)

	stats, err := LoadPersistedStats()
	if err != nil {
		t.Fatalf("LoadPersistedStats: %v", err)
	}

	s := stats["scroll"]
	if s == nil {
		t.Fatal("expected 'scroll' in stats")
	}
	if s.DurationMin != 50.0 {
		t.Errorf("scroll.DurationMin = %f, want 50.0", s.DurationMin)
	}
	if s.DurationMax != 200.0 {
		t.Errorf("scroll.DurationMax = %f, want 200.0", s.DurationMax)
	}
	if s.DurationCount != 4 {
		t.Errorf("scroll.DurationCount = %d, want 4", s.DurationCount)
	}
}

func TestLoadPersistedStatsEmptyDB(t *testing.T) {
	setupTestDB(t)

	stats, err := LoadPersistedStats()
	if err != nil {
		t.Fatalf("LoadPersistedStats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats from fresh DB, got %d entries", len(stats))
	}
}
