package dataloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// SQLiteLoader implements Loader using a SQLite database.
type SQLiteLoader struct {
	dbPath string
	db     *sql.DB
}

// NewSQLiteLoader creates a new SQLite-backed loader.
func NewSQLiteLoader(dbPath string) *SQLiteLoader {
	return &SQLiteLoader{dbPath: dbPath}
}

func (l *SQLiteLoader) openDB() (*sql.DB, error) {
	if l.db != nil {
		return l.db, nil
	}
	dsn := l.dbPath
	if l.dbPath != ":memory:" {
		dsn += "?_journal_mode=WAL&_busy_timeout=5000"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS training_pairs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL DEFAULT '',
		ocr_before_text TEXT NOT NULL DEFAULT '',
		command_json TEXT NOT NULL DEFAULT '',
		ocr_after_text TEXT NOT NULL DEFAULT '',
		window_title TEXT NOT NULL DEFAULT '',
		success INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	l.db = db
	return db, nil
}

func (l *SQLiteLoader) Close() error {
	if l.db != nil {
		err := l.db.Close()
		l.db = nil
		return err
	}
	return nil
}

func (l *SQLiteLoader) LoadAll(ctx context.Context) ([]Sample, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("dataloader: %w", err)
	}
	db, err := l.openDB()
	if err != nil {
		return nil, fmt.Errorf("dataloader: open: %w", err)
	}
	return l.queryDB(ctx, db, "SELECT ocr_before_text, command_json, ocr_after_text, window_title, success, created_at FROM training_pairs ORDER BY id ASC", nil)
}

func (l *SQLiteLoader) LoadRecent(ctx context.Context, n int) ([]Sample, error) {
	if n <= 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("dataloader: %w", err)
	}
	db, err := l.openDB()
	if err != nil {
		return nil, fmt.Errorf("dataloader: open: %w", err)
	}
	return l.queryDB(ctx, db, "SELECT ocr_before_text, command_json, ocr_after_text, window_title, success, created_at FROM training_pairs ORDER BY id DESC LIMIT ?", []interface{}{n})
}

func (l *SQLiteLoader) LoadByTool(ctx context.Context, tool string) ([]Sample, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("dataloader: %w", err)
	}
	db, err := l.openDB()
	if err != nil {
		return nil, fmt.Errorf("dataloader: open: %w", err)
	}
	return l.queryDB(ctx, db, "SELECT ocr_before_text, command_json, ocr_after_text, window_title, success, created_at FROM training_pairs WHERE command_json LIKE ?", []interface{}{"%\"tool\":\"" + tool + "\"%"})
}

func (l *SQLiteLoader) Count(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("dataloader: %w", err)
	}
	db, err := l.openDB()
	if err != nil {
		return 0, fmt.Errorf("dataloader: open: %w", err)
	}
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM training_pairs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("dataloader: count: %w", err)
	}
	return count, nil
}

func (l *SQLiteLoader) Stats(ctx context.Context) (map[string]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("dataloader: %w", err)
	}
	db, err := l.openDB()
	if err != nil {
		return nil, fmt.Errorf("dataloader: open: %w", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT command_json FROM training_pairs")
	if err != nil {
		return nil, fmt.Errorf("dataloader: stats query: %w", err)
	}
	defer rows.Close()
	stats := make(map[string]int)
	for rows.Next() {
		var cmdJSON string
		if err := rows.Scan(&cmdJSON); err != nil {
			continue
		}
		tool := extractTool(cmdJSON)
		if tool != "" {
			stats[tool]++
		}
	}
	return stats, rows.Err()
}

func (l *SQLiteLoader) queryDB(ctx context.Context, db *sql.DB, query string, args []interface{}) ([]Sample, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("dataloader: query: %w", err)
	}
	defer rows.Close()
	var samples []Sample
	for rows.Next() {
		var s Sample
		var cmdJSON string
		var successInt int
		if err := rows.Scan(&s.Context, &cmdJSON, &s.ArgsJSON, &s.WindowTitle, &successInt, &s.CreatedAt); err != nil {
			continue
		}
		s.Success = successInt == 1
		s.Action = extractTool(cmdJSON)
		s.ArgsJSON = cmdJSON
		samples = append(samples, s)
	}
	return samples, rows.Err()
}

func extractTool(cmdJSON string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(cmdJSON), &data); err != nil {
		if !strings.HasPrefix(cmdJSON, "{") {
			return strings.TrimSpace(cmdJSON)
		}
		return ""
	}
	if tool, ok := data["tool"].(string); ok {
		return tool
	}
	return ""
}
