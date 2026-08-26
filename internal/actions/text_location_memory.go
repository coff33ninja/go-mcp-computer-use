package actions

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"
)

var (
	textLocDB    *sql.DB
	textLocMu    sync.Mutex
	textLocInit  sync.Once
	textLocError error
)

type TextLocation struct {
	ID          int64     `json:"id"`
	TextHash    string    `json:"text_hash"`
	TextContent string    `json:"text_content"`
	WindowTitle string    `json:"window_title"`
	X           int32     `json:"x"`
	Y           int32     `json:"y"`
	W           int32     `json:"w"`
	H           int32     `json:"h"`
	ZOrder      int       `json:"z_order"`
	Confidence  float64   `json:"confidence"`
	LastSeen    time.Time `json:"last_seen"`
	HitCount    int       `json:"hit_count"`
}

func InitTextLocationStore() error {
	textLocInit.Do(func() {
		if memDB == nil {
			if err := InitMemoryStore(); err != nil {
				textLocError = fmt.Errorf("init memory store: %w", err)
				return
			}
		}
		textLocDB = memDB
		if err := createTextLocationTables(textLocDB); err != nil {
			textLocError = fmt.Errorf("create text_location tables: %w", err)
			return
		}
		// Prune stale entries with coords outside virtual screen bounds
		// (leftover from pre-0.3.4 bitmap-space coordinates)
		pruneStaleTextLocations()
	})
	return textLocError
}

func pruneStaleTextLocations() {
	bounds := VirtualScreenBounds()
	if textLocDB == nil {
		return
	}
	textLocMu.Lock()
	defer textLocMu.Unlock()
	textLocDB.Exec(
		`DELETE FROM text_locations WHERE x < ? OR x >= ? OR y < ? OR y >= ?`,
		bounds.X, bounds.X+bounds.W, bounds.Y, bounds.Y+bounds.H,
	)
}

func createTextLocationTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS text_locations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text_hash TEXT NOT NULL,
			text_content TEXT NOT NULL,
			window_title TEXT NOT NULL DEFAULT '',
			x INTEGER NOT NULL DEFAULT 0,
			y INTEGER NOT NULL DEFAULT 0,
			w INTEGER NOT NULL DEFAULT 0,
			h INTEGER NOT NULL DEFAULT 0,
			z_order INTEGER NOT NULL DEFAULT 0,
			confidence REAL NOT NULL DEFAULT 1.0,
			last_seen TEXT NOT NULL,
			hit_count INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tl_hash ON text_locations(text_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_tl_window ON text_locations(window_title)`,
		`CREATE INDEX IF NOT EXISTS idx_tl_hash_window ON text_locations(text_hash, window_title)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS text_locations_fts USING fts5(
			text_content, window_title, content='text_locations', content_rowid='id'
		)`,
		`CREATE TRIGGER IF NOT EXISTS tl_ai AFTER INSERT ON text_locations BEGIN
			INSERT INTO text_locations_fts(rowid, text_content, window_title)
			VALUES (new.id, new.text_content, new.window_title);
		END`,
		`CREATE TRIGGER IF NOT EXISTS tl_ad AFTER DELETE ON text_locations BEGIN
			INSERT INTO text_locations_fts(text_locations_fts, rowid, text_content, window_title)
			VALUES('delete', old.id, old.text_content, old.window_title);
		END`,
		`CREATE TRIGGER IF NOT EXISTS tl_au AFTER UPDATE ON text_locations BEGIN
			INSERT INTO text_locations_fts(text_locations_fts, rowid, text_content, window_title)
			VALUES('delete', old.id, old.text_content, old.window_title);
			INSERT INTO text_locations_fts(rowid, text_content, window_title)
			VALUES (new.id, new.text_content, new.window_title);
		END`,
		`ALTER TABLE text_locations ADD COLUMN z_order INTEGER NOT NULL DEFAULT 0`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			// ALTER TABLE fails silently if column already exists
			if !strings.Contains(err.Error(), "duplicate column") {
				return err
			}
		}
	}
	return nil
}

func textHash(text string) string {
	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(text))))
	return fmt.Sprintf("%x", h.Sum32())
}

func StoreTextLocation(textContent, windowTitle string, x, y, w, h, zOrder int32) {
	if textLocDB == nil {
		return
	}
	trimmed := strings.TrimSpace(textContent)
	if trimmed == "" || len(trimmed) > 200 {
		return
	}
	th := textHash(trimmed)
	now := time.Now().UTC().Format(time.RFC3339)

	textLocMu.Lock()
	defer textLocMu.Unlock()

	var existingID int64
	err := textLocDB.QueryRow(
		`SELECT id FROM text_locations WHERE text_hash=? AND window_title=?`,
		th, windowTitle,
	).Scan(&existingID)

	if err == nil {
		textLocDB.Exec(
			`UPDATE text_locations SET x=?, y=?, w=?, h=?, z_order=?, last_seen=?, hit_count=hit_count+1, confidence=MIN(confidence+0.1, 2.0) WHERE id=?`,
			x, y, w, h, zOrder, now, existingID,
		)
	} else {
		textLocDB.Exec(
			`INSERT INTO text_locations (text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1.0, ?, 1)`,
			th, trimmed, windowTitle, x, y, w, h, zOrder, now,
		)
	}
}

func StoreOCRResultLocations(result *OCRResult, windowTitle string, zOrder int32) {
	if textLocDB == nil || result == nil {
		return
	}
	for _, word := range result.Words {
		t := strings.TrimSpace(word.Text)
		if t != "" && len(t) <= 200 {
			StoreTextLocation(t, windowTitle, int32(word.X), int32(word.Y), int32(word.W), int32(word.H), zOrder)
		}
	}
}

func FindTextLocation(text, windowTitle string) *TextLocation {
	if textLocDB == nil {
		return nil
	}
	th := textHash(strings.TrimSpace(text))

	textLocMu.Lock()
	defer textLocMu.Unlock()

	var loc TextLocation
	var lastSeen string
	err := textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=? AND window_title=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th, windowTitle,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err != nil {
		return nil
	}
	loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return &loc
}

// FindTextLocationMatch tries to find text matching the given z_order first (topmost layer),
// then falls back to any z_order. Returns the best match.
func FindTextLocationMatch(text, windowTitle string, zOrder int) *TextLocation {
	if textLocDB == nil {
		return nil
	}
	th := textHash(strings.TrimSpace(text))

	textLocMu.Lock()
	defer textLocMu.Unlock()

	// Try exact z_order match first (topmost layer preference)
	var loc TextLocation
	var lastSeen string
	err := textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=? AND window_title=? AND z_order=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th, windowTitle, zOrder,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err == nil {
		loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		return &loc
	}

	// Fallback: any z_order for same text+window
	err = textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=? AND window_title=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th, windowTitle,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err != nil {
		return nil
	}
	loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return &loc
}

// FindTextLocationAnyMatch tries to find text matching z_order first, then any.
func FindTextLocationAnyMatch(text string, zOrder int) *TextLocation {
	if textLocDB == nil {
		return nil
	}
	th := textHash(strings.TrimSpace(text))

	textLocMu.Lock()
	defer textLocMu.Unlock()

	var loc TextLocation
	var lastSeen string
	err := textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=? AND z_order=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th, zOrder,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err == nil {
		loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		return &loc
	}

	// Fallback: any z_order
	err = textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err != nil {
		return nil
	}
	loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return &loc
}

func FindTextLocationAny(text string) *TextLocation {
	if textLocDB == nil {
		return nil
	}
	th := textHash(strings.TrimSpace(text))

	textLocMu.Lock()
	defer textLocMu.Unlock()

	var loc TextLocation
	var lastSeen string
	err := textLocDB.QueryRow(
		`SELECT id, text_hash, text_content, window_title, x, y, w, h, z_order, confidence, last_seen, hit_count
		 FROM text_locations WHERE text_hash=?
		 ORDER BY hit_count DESC, last_seen DESC LIMIT 1`,
		th,
	).Scan(&loc.ID, &loc.TextHash, &loc.TextContent, &loc.WindowTitle,
		&loc.X, &loc.Y, &loc.W, &loc.H, &loc.ZOrder, &loc.Confidence, &lastSeen, &loc.HitCount)
	if err != nil {
		return nil
	}
	loc.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
	return &loc
}

func PruneTextLocations(maxAgeDays int) int {
	if textLocDB == nil || maxAgeDays <= 0 {
		return 0
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays).UTC().Format(time.RFC3339)
	textLocMu.Lock()
	defer textLocMu.Unlock()
	res, err := textLocDB.Exec(`DELETE FROM text_locations WHERE last_seen < ? AND hit_count <= 1`, cutoff)
	if err != nil {
		return 0
	}
	n, _ := res.RowsAffected()
	return int(n)
}
