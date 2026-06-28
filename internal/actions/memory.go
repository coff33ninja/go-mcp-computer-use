package actions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	memDB    *sql.DB
	memMu    sync.Mutex
	memInit  sync.Once
	memError error
)

func InitMemoryStore() error {
	memInit.Do(func() {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			memError = fmt.Errorf("APPDATA not set")
			return
		}
		dir := filepath.Join(appData, "go-mcp-computer-use")
		if err := os.MkdirAll(dir, 0755); err != nil {
			memError = fmt.Errorf("create memory dir: %w", err)
			return
		}
		dbPath := filepath.Join(dir, "memory.db")
		db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
		if err != nil {
			memError = fmt.Errorf("open memory db: %w", err)
			return
		}
		if err := createMemoryTables(db); err != nil {
			db.Close()
			memError = fmt.Errorf("create tables: %w", err)
			return
		}
		memDB = db
	})
	return memError
}

func createMemoryTables(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS facts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			ttl INTEGER DEFAULT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_facts_key_scope ON facts(key, scope)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS facts_fts USING fts5(
			key, value, scope, tags, content='facts', content_rowid='id'
		)`,
		`CREATE TRIGGER IF NOT EXISTS facts_ai AFTER INSERT ON facts BEGIN
			INSERT INTO facts_fts(rowid, key, value, scope, tags)
			VALUES (new.id, new.key, new.value, new.scope, new.tags);
		END`,
		`CREATE TRIGGER IF NOT EXISTS facts_ad AFTER DELETE ON facts BEGIN
			INSERT INTO facts_fts(facts_fts, rowid, key, value, scope, tags)
			VALUES('delete', old.id, old.key, old.value, old.scope, old.tags);
		END`,
		`CREATE TRIGGER IF NOT EXISTS facts_au AFTER UPDATE ON facts BEGIN
			INSERT INTO facts_fts(facts_fts, rowid, key, value, scope, tags)
			VALUES('delete', old.id, old.key, old.value, old.scope, old.tags);
			INSERT INTO facts_fts(rowid, key, value, scope, tags)
			VALUES (new.id, new.key, new.value, new.scope, new.tags);
		END`,
		`CREATE TABLE IF NOT EXISTS element_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			element_key TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT '',
			template_b64 TEXT NOT NULL,
			template_width INTEGER NOT NULL DEFAULT 48,
			template_height INTEGER NOT NULL DEFAULT 48,
			stored_coord_x INTEGER NOT NULL DEFAULT 0,
			stored_coord_y INTEGER NOT NULL DEFAULT 0,
			window_title TEXT NOT NULL DEFAULT '',
			signature_keywords TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			hit_count INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tmpl_key_scope ON element_templates(element_key, scope)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

type MemoryFact struct {
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	Scope     string    `json:"scope"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TTL       int       `json:"ttl"`
	Confidence string   `json:"confidence,omitempty"`
}

type MemorySetInput struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
	TTL   int    `json:"ttl,omitempty"`
}

func MemorySet(in MemorySetInput) error {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.Key == "" {
		return fmt.Errorf("key is required")
	}

	scope := in.Scope
	if scope == "" {
		scope = "default"
	}

	valueJSON, err := json.Marshal(in.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err = memDB.Exec(`INSERT INTO facts(key, value, scope, tags, created_at, updated_at, ttl)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key, scope) DO UPDATE SET
			value = excluded.value,
			tags = excluded.tags,
			updated_at = excluded.updated_at,
			ttl = excluded.ttl`,
		in.Key, string(valueJSON), scope, in.Tags, now, now, ttlOrNil(in.TTL))
	return err
}

func MemoryGet(key, scope string) (*MemoryFact, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if scope == "" {
		scope = "default"
	}

	row := memDB.QueryRow(`SELECT key, value, scope, tags, created_at, updated_at, ttl
		FROM facts WHERE key = ? AND scope = ?`, key, scope)

	var k, v, s, tagsStr, createdAtStr, updatedAtStr string
	var ttl sql.NullInt64
	if err := row.Scan(&k, &v, &s, &tagsStr, &createdAtStr, &updatedAtStr, &ttl); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan fact: %w", err)
	}

	if ttl.Valid && ttl.Int64 > 0 {
		createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
		if time.Since(createdAt) > time.Duration(ttl.Int64)*time.Second {
			return nil, nil
		}
	}

	fact := &MemoryFact{
		Key:       k,
		Scope:     s,
		Tags:      splitTags(tagsStr),
		CreatedAt: parseTime(createdAtStr),
		UpdatedAt: parseTime(updatedAtStr),
	}
	if ttl.Valid {
		fact.TTL = int(ttl.Int64)
	}

	if err := json.Unmarshal([]byte(v), &fact.Value); err != nil {
		fact.Value = v
	}

	fact.Confidence = "ok"
	return fact, nil
}

type MemorySearchInput struct {
	Query string `json:"query"`
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type MemorySearchResult struct {
	Key       string   `json:"key"`
	Value     any      `json:"value"`
	Scope     string   `json:"scope"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	TTL       int      `json:"ttl"`
	Confidence string  `json:"confidence,omitempty"`
}

func MemorySearch(in MemorySearchInput) ([]MemorySearchResult, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 20
	}

	ftsQuery := strings.ReplaceAll(in.Query, "\"", "\"\"")
	ftsQuery = fmt.Sprintf(`"%s"`, ftsQuery)

	var rows *sql.Rows
	var err error
	if in.Scope != "" {
		rows, err = memDB.Query(`SELECT f.key, f.value, f.scope, f.tags, f.created_at, f.updated_at, f.ttl
			FROM facts_fts ft JOIN facts f ON ft.rowid = f.id
			WHERE facts_fts MATCH ? AND f.scope = ?
			ORDER BY rank LIMIT ?`, ftsQuery, in.Scope, in.Limit)
	} else {
		rows, err = memDB.Query(`SELECT f.key, f.value, f.scope, f.tags, f.created_at, f.updated_at, f.ttl
			FROM facts_fts ft JOIN facts f ON ft.rowid = f.id
			WHERE facts_fts MATCH ?
			ORDER BY rank LIMIT ?`, ftsQuery, in.Limit)
	}
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []MemorySearchResult
	for rows.Next() {
		var k, v, s, tagsStr, createdAtStr, updatedAtStr string
		var ttl sql.NullInt64
		if err := rows.Scan(&k, &v, &s, &tagsStr, &createdAtStr, &updatedAtStr, &ttl); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}

		if ttl.Valid && ttl.Int64 > 0 {
			createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
			if time.Since(createdAt) > time.Duration(ttl.Int64)*time.Second {
				continue
			}
		}

		r := MemorySearchResult{
			Key:       k,
			Scope:     s,
			Tags:      splitTags(tagsStr),
			CreatedAt: createdAtStr,
			UpdatedAt: updatedAtStr,
		}
		if ttl.Valid {
			r.TTL = int(ttl.Int64)
		}
		if err := json.Unmarshal([]byte(v), &r.Value); err != nil {
			r.Value = v
		}
		r.Confidence = "ok"
		results = append(results, r)
	}
	return results, rows.Err()
}

type MemoryListInput struct {
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func MemoryList(in MemoryListInput) ([]MemorySearchResult, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 50
	}

	var where []string
	var args []any

	if in.Scope != "" {
		where = append(where, "scope = ?")
		args = append(args, in.Scope)
	}
	if in.Tags != "" {
		tagList := splitTags(in.Tags)
		for _, tag := range tagList {
			where = append(where, "',' || tags || ',' LIKE '%' || ? || '%'")
			args = append(args, ","+tag+",")
		}
	}

	query := `SELECT key, value, scope, tags, created_at, updated_at, ttl FROM facts`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, in.Limit)

	rows, err := memDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	var results []MemorySearchResult
	for rows.Next() {
		var k, v, s, tagsStr, createdAtStr, updatedAtStr string
		var ttl sql.NullInt64
		if err := rows.Scan(&k, &v, &s, &tagsStr, &createdAtStr, &updatedAtStr, &ttl); err != nil {
			return nil, fmt.Errorf("scan list result: %w", err)
		}

		if ttl.Valid && ttl.Int64 > 0 {
			createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
			if time.Since(createdAt) > time.Duration(ttl.Int64)*time.Second {
				continue
			}
		}

		r := MemorySearchResult{
			Key:       k,
			Scope:     s,
			Tags:      splitTags(tagsStr),
			CreatedAt: createdAtStr,
			UpdatedAt: updatedAtStr,
		}
		if ttl.Valid {
			r.TTL = int(ttl.Int64)
		}
		if err := json.Unmarshal([]byte(v), &r.Value); err != nil {
			r.Value = v
		}
		r.Confidence = "ok"
		results = append(results, r)
	}
	return results, rows.Err()
}

type MemoryForgetInput struct {
	Key   string `json:"key,omitempty"`
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
}

func MemoryForget(in MemoryForgetInput) (int64, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return 0, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.Key == "" && in.Scope == "" && in.Tags == "" {
		return 0, fmt.Errorf("at least one filter (key, scope, or tags) is required")
	}

	var where []string
	var args []any

	if in.Key != "" {
		where = append(where, "key = ?")
		args = append(args, in.Key)
	}
	if in.Scope != "" {
		where = append(where, "scope = ?")
		args = append(args, in.Scope)
	}
	if in.Tags != "" {
		tagList := splitTags(in.Tags)
		for _, tag := range tagList {
			where = append(where, "',' || tags || ',' LIKE '%' || ? || '%'")
			args = append(args, ","+tag+",")
		}
	}

	query := "DELETE FROM facts"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	result, err := memDB.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("forget: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func splitTags(tagsStr string) []string {
	if tagsStr == "" {
		return nil
	}
	parts := strings.Split(tagsStr, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func ttlOrNil(ttl int) any {
	if ttl <= 0 {
		return nil
	}
	return ttl
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ── Element Templates ──

type TemplateInfo struct {
	ElementKey        string   `json:"element_key"`
	Scope             string   `json:"scope"`
	TemplateB64       string   `json:"template_b64,omitempty"`
	TemplateWidth     int      `json:"template_width"`
	TemplateHeight    int      `json:"template_height"`
	StoredCoordX      int32    `json:"stored_coord_x"`
	StoredCoordY      int32    `json:"stored_coord_y"`
	WindowTitle       string   `json:"window_title"`
	SignatureKeywords []string `json:"signature_keywords,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	HitCount          int      `json:"hit_count"`
}

type TemplateStoreInput struct {
	ElementKey        string   `json:"element_key"`
	Scope             string   `json:"scope,omitempty"`
	CenterX           int32    `json:"center_x"`
	CenterY           int32    `json:"center_y"`
	CropSize          int      `json:"crop_size,omitempty"`
	WindowTitle       string   `json:"window_title,omitempty"`
	SignatureKeywords []string `json:"signature_keywords,omitempty"`
}

func TemplateStore(in TemplateStoreInput) (*TemplateInfo, error) {
	cropSize := in.CropSize
	if cropSize <= 0 {
		cropSize = 48
	}

	screenB64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("template_store screenshot: %w", err)
	}

	cropB64, err := CropRegion(screenB64, in.CenterX-int32(cropSize/2), in.CenterY-int32(cropSize/2), int32(cropSize), int32(cropSize))
	if err != nil {
		return nil, fmt.Errorf("template_store crop: %w", err)
	}

	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.ElementKey == "" {
		return nil, fmt.Errorf("element_key is required")
	}

	scope := in.Scope
	if scope == "" {
		scope = "default"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	kw := strings.Join(in.SignatureKeywords, ",")

	_, err = memDB.Exec(`INSERT INTO element_templates(
		element_key, scope, template_b64, template_width, template_height,
		stored_coord_x, stored_coord_y, window_title, signature_keywords,
		created_at, updated_at, hit_count
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
	ON CONFLICT(element_key, scope) DO UPDATE SET
		template_b64 = excluded.template_b64,
		template_width = excluded.template_width,
		template_height = excluded.template_height,
		stored_coord_x = excluded.stored_coord_x,
		stored_coord_y = excluded.stored_coord_y,
		window_title = excluded.window_title,
		signature_keywords = excluded.signature_keywords,
		updated_at = excluded.updated_at,
		hit_count = hit_count + 1`,
		in.ElementKey, scope, cropB64, cropSize, cropSize,
		in.CenterX, in.CenterY, in.WindowTitle, kw, now, now)
	if err != nil {
		return nil, fmt.Errorf("store template: %w", err)
	}

	return &TemplateInfo{
		ElementKey:        in.ElementKey,
		Scope:             scope,
		TemplateB64:       cropB64,
		TemplateWidth:     cropSize,
		TemplateHeight:    cropSize,
		StoredCoordX:      in.CenterX,
		StoredCoordY:      in.CenterY,
		WindowTitle:       in.WindowTitle,
		SignatureKeywords: in.SignatureKeywords,
		CreatedAt:         now,
		UpdatedAt:         now,
		HitCount:          1,
	}, nil
}

type TemplateFindInput struct {
	ElementKey string  `json:"element_key"`
	Scope      string  `json:"scope,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
}

type TemplateFindResult struct {
	ElementKey   string  `json:"element_key"`
	Scope        string  `json:"scope"`
	Found        bool    `json:"found"`
	X            int32   `json:"x,omitempty"`
	Y            int32   `json:"y,omitempty"`
	Width        int32   `json:"width,omitempty"`
	Height       int32   `json:"height,omitempty"`
	Score        float64 `json:"score,omitempty"`
	StoredCoordX int32   `json:"stored_coord_x,omitempty"`
	StoredCoordY int32   `json:"stored_coord_y,omitempty"`
	DriftX       int32   `json:"drift_x,omitempty"`
	DriftY       int32   `json:"drift_y,omitempty"`
}

func TemplateFind(in TemplateFindInput) (*TemplateFindResult, error) {
	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	memMu.Lock()

	scope := in.Scope
	if scope == "" {
		scope = "default"
	}

	var b64 string
	var tw, th, sx, sy int32
	var hitCount int
	err := memDB.QueryRow(`SELECT template_b64, template_width, template_height,
		stored_coord_x, stored_coord_y, hit_count
		FROM element_templates WHERE element_key = ? AND scope = ?`,
		in.ElementKey, scope).Scan(&b64, &tw, &th, &sx, &sy, &hitCount)
	memMu.Unlock()

	if err == sql.ErrNoRows {
		return &TemplateFindResult{ElementKey: in.ElementKey, Scope: scope, Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("template_find read: %w", err)
	}

	threshold := in.Threshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.7
	}

	result, err := FindImageOnScreen(b64, threshold)
	if err != nil {
		return &TemplateFindResult{
			ElementKey: in.ElementKey, Scope: scope, Found: false,
			StoredCoordX: sx, StoredCoordY: sy,
		}, nil
	}

	// Update hit count
	memMu.Lock()
	memDB.Exec(`UPDATE element_templates SET hit_count = hit_count + 1, updated_at = ?
		WHERE element_key = ? AND scope = ?`,
		time.Now().UTC().Format(time.RFC3339), in.ElementKey, scope)
	memMu.Unlock()

	return &TemplateFindResult{
		ElementKey:   in.ElementKey,
		Scope:        scope,
		Found:        true,
		X:            result.X,
		Y:            result.Y,
		Width:        result.Width,
		Height:       result.Height,
		Score:        result.Score,
		StoredCoordX: sx,
		StoredCoordY: sy,
		DriftX:       result.X - sx,
		DriftY:       result.Y - sy,
	}, nil
}

type TemplateListInput struct {
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func TemplateList(in TemplateListInput) ([]TemplateInfo, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return nil, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 50
	}

	var rows *sql.Rows
	var err error
	if in.Scope != "" {
		rows, err = memDB.Query(`SELECT element_key, scope, template_width, template_height,
			stored_coord_x, stored_coord_y, window_title, signature_keywords,
			created_at, updated_at, hit_count
			FROM element_templates WHERE scope = ? ORDER BY updated_at DESC LIMIT ?`,
			in.Scope, in.Limit)
	} else {
		rows, err = memDB.Query(`SELECT element_key, scope, template_width, template_height,
			stored_coord_x, stored_coord_y, window_title, signature_keywords,
			created_at, updated_at, hit_count
			FROM element_templates ORDER BY updated_at DESC LIMIT ?`, in.Limit)
	}
	if err != nil {
		return nil, fmt.Errorf("template_list: %w", err)
	}
	defer rows.Close()

	var results []TemplateInfo
	for rows.Next() {
		var t TemplateInfo
		var kw string
		if err := rows.Scan(&t.ElementKey, &t.Scope, &t.TemplateWidth, &t.TemplateHeight,
			&t.StoredCoordX, &t.StoredCoordY, &t.WindowTitle, &kw,
			&t.CreatedAt, &t.UpdatedAt, &t.HitCount); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		t.SignatureKeywords = splitTags(kw)
		results = append(results, t)
	}
	return results, rows.Err()
}

func MemoryStoreDetectionElements(elements []DetectedElement, windowTitle string) {
	for _, el := range elements {
		key := fmt.Sprintf("ui:%s:%s", windowTitle, el.Class)
		MemorySet(MemorySetInput{
			Key:   key,
			Value: el,
			Scope: "ui",
			Tags:  fmt.Sprintf("ui,element,%s", el.Class),
			TTL:   3600,
		})
	}
}

func TemplateForget(elementKey, scope string) (int64, error) {
	memMu.Lock()
	defer memMu.Unlock()

	if memDB == nil {
		if err := InitMemoryStore(); err != nil {
			return 0, fmt.Errorf("memory store not initialized: %w", err)
		}
	}
	if elementKey == "" {
		return 0, fmt.Errorf("element_key is required")
	}
	if scope == "" {
		scope = "default"
	}

	result, err := memDB.Exec(`DELETE FROM element_templates WHERE element_key = ? AND scope = ?`,
		elementKey, scope)
	if err != nil {
		return 0, fmt.Errorf("template_forget: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}
