package actions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	dlogDB   *sql.DB
	dlogMu   sync.Mutex
	dlogOnce sync.Once
	dlogRoot string
)

type DataLogConfig struct {
	LogOCR   bool
	LogChain bool
	LogKeys  bool
}

var DataLog = &DataLogConfig{
	LogOCR:   true,
	LogChain: true,
	LogKeys:  true,
}

func ForegroundWindowHandle() uintptr {
	hwnd, _, _ := getForegroundWindow.Call()
	return hwnd
}

func bridgeBufferSize() int {
	pairMu.Lock()
	defer pairMu.Unlock()
	return len(recentOCR)
}

func getDataLogRoot() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "go-mcp-computer-use", "datalog")
}

func InitDataLog() error {
	var initErr error
	dlogOnce.Do(func() {
		dlogRoot = getDataLogRoot()
		if dlogRoot == "" {
			initErr = fmt.Errorf("APPDATA not set")
			return
		}
		if err := os.MkdirAll(dlogRoot, 0755); err != nil {
			initErr = fmt.Errorf("create datalog dir: %w", err)
			return
		}
		dbPath := filepath.Join(dlogRoot, "datalog.db")
		db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
		if err != nil {
			initErr = fmt.Errorf("open datalog db: %w", err)
			return
		}
		if err := createDataLogTables(db); err != nil {
			db.Close()
			initErr = fmt.Errorf("create datalog tables: %w", err)
			return
		}
		dlogDB = db
	})
	return initErr
}

func createDataLogTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS command_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'chain',
			tool TEXT NOT NULL,
			args TEXT NOT NULL DEFAULT '{}',
			success INTEGER NOT NULL DEFAULT 1,
			error_text TEXT NOT NULL DEFAULT '',
			window_title TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			ocr_before TEXT NOT NULL DEFAULT '',
			ocr_after TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_session ON command_log(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_tool ON command_log(tool)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_created ON command_log(created_at)`,

		`CREATE TABLE IF NOT EXISTS chain_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'chain',
			step_count INTEGER NOT NULL DEFAULT 0,
			success_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			chain_json TEXT NOT NULL DEFAULT '',
			result_json TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chain_session ON chain_log(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chain_created ON chain_log(created_at)`,

		`CREATE TABLE IF NOT EXISTS ocr_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'tool',
			image_path TEXT NOT NULL DEFAULT '',
			ocr_text TEXT NOT NULL DEFAULT '',
			ocr_json TEXT NOT NULL DEFAULT '{}',
			text_length INTEGER NOT NULL DEFAULT 0,
			word_count INTEGER NOT NULL DEFAULT 0,
			window_title TEXT NOT NULL DEFAULT '',
			triggered_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ocr_session ON ocr_log(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ocr_created ON ocr_log(created_at)`,

		`CREATE TABLE IF NOT EXISTS training_pairs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			ocr_before_text TEXT NOT NULL DEFAULT '',
			command_json TEXT NOT NULL DEFAULT '',
			ocr_after_text TEXT NOT NULL DEFAULT '',
			window_title TEXT NOT NULL DEFAULT '',
			success INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pair_session ON training_pairs(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pair_created ON training_pairs(created_at)`,

		`CREATE TABLE IF NOT EXISTS adaptive_stats (
			tool TEXT PRIMARY KEY,
			success_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			duration_count INTEGER NOT NULL DEFAULT 0,
			duration_sum REAL NOT NULL DEFAULT 0,
			duration_min REAL NOT NULL DEFAULT 0,
			duration_max REAL NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS task_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL DEFAULT '',
			step_count INTEGER NOT NULL DEFAULT 0,
			success_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			total_duration_ms INTEGER NOT NULL DEFAULT 0,
			insights TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_session ON task_log(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_created ON task_log(created_at)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func saveScreenshotForDataLog() string {
	if dlogRoot == "" {
		return ""
	}
	b64, err := CaptureScreen()
	if err != nil {
		return ""
	}
	ts := time.Now().UnixMilli()
	imgPath := filepath.Join(dlogRoot, fmt.Sprintf("screen_%d.png", ts))
	img, err := decodePNGB64(b64)
	if err != nil {
		return ""
	}
	if err := savePNG(imgPath, img); err != nil {
		return ""
	}
	return imgPath
}

// LogToolCall logs an action function call with its args. Used by action
// functions (Click, KeyPress, etc.) to log every MCP tool invocation.
// Bridge logic (OCR before/after pairing) runs synchronously to avoid races
// with subsequent OCR calls. Only the DB insert is sent to a goroutine.
func LogToolCall(tool string, argsJSON string, err error) {
	if !DataLog.LogKeys {
		return
	}
	errVal := err

	// Note: timing/success stats for the adaptive engine are recorded by
	// the caller (Click, TypeText, etc.) with the real elapsed duration
	// via Adaptive.RecordResult. Recording here too would double-count
	// every action with a phantom 0ms sample, so this function leaves
	// the engine's timing/success maps alone.

	// Bridge: find recent OCR and set pending command synchronously so
	// the next OCR call will find the pending pair immediately. The OCR
	// context is scoped to text near the command's target coordinates
	// when available, instead of the entire screen, so generic on-screen
	// words don't dominate the learned associations.
	ocrBefore := findRecentOCRBefore(tool, argsJSON)
	bufSize := bridgeBufferSize()
	slog.Warn("LogToolCall bridge", "tool", tool, "found_ocr", ocrBefore != "", "buffer_size", bufSize)
	if ocrBefore != "" {
		cmdJSON := argsJSON
		if tool != "" {
			cmdMap := map[string]any{"tool": tool, "args": argsJSON}
			if b, jErr := json.Marshal(cmdMap); jErr == nil {
				cmdJSON = string(b)
			}
		}
		pairMu.Lock()
		pendingCmd = &TrainingPairInput{
			OCRBefore: ocrBefore,
			Command:   cmdJSON,
			Success:   errVal == nil,
		}
		pendingTime = time.Now()
		slog.Warn("LogToolCall set pending", "ocr_before", ocrBefore[:min(len(ocrBefore), 40)])
		pairMu.Unlock()
	}

	// Always refresh the OCR buffer after every action, regardless of
	// whether ocrBefore was found above. This is what completes the
	// pending pair (if any) via tryCompletePair, AND seeds recentOCR for
	// the *next* action. Previously this only ran inside the
	// `ocrBefore != ""` branch, so a single missed bridgeWindow meant no
	// further training pairs would ever be logged until something called
	// an OCR tool manually again — one miss silently killed the pipeline
	// for the rest of the session. Now every action re-arms the bridge.
	if result, ocrErr := OCRScreen(""); ocrErr != nil {
		slog.Warn("LogToolCall OCR auto-capture failed", "tool", tool, "error", ocrErr)
	} else if result != nil {
		slog.Warn("LogToolCall OCR auto-capture", "tool", tool, "ocr_after_len", len(result.Text))
	}

	if ocrBefore != "" {
		// Learn coordinates with OCR context for spatial predictions.
		Adaptive.LearnFromCommandWithContext(tool, argsJSON, ocrBefore, errVal == nil)
	}

	go func() {
		errText := ""
		if errVal != nil {
			errText = errVal.Error()
		}
		LogCommand(tool, argsJSON, errVal == nil, errText, "", 0)
	}()
}

type ocrSnap struct {
	text      string
	words     []OCRWord
	timestamp time.Time
}

var (
	recentOCR    []ocrSnap
	pendingCmd   *TrainingPairInput
	pendingTime  time.Time
	pairMu       sync.Mutex
)

// bridgeWindow was 30s, which is tight for agent-driven workflows where
// there's real round-trip/thinking latency between an OCR call and the
// action that follows it. Combined with the self-healing refresh in
// LogToolCall above, 60s gives realistic headroom without letting stale
// context linger indefinitely.
const bridgeWindow = 60 * time.Second
const maxRecentOCR = 5

// nearbyWordRadius / nearbyWordLimit bound how much OCR context gets
// attributed to a coordinate-based action (click/move_mouse/drag/hover).
// Without this, ocr_before captured the ENTIRE screen's text, so every
// common word on screen ("the", "with", bullet points, unrelated news
// headlines, etc.) got associated with whatever command followed —
// producing counts that vastly exceeded total_commands and predictions
// dominated by noise instead of the actual clicked/typed element.
const nearbyWordRadius = 200.0
const nearbyWordLimit = 20

// fallbackWordLimit caps the OCR context used for non-coordinate tools
// (type, key_press, etc.) where we have no target location to scope
// against. It's a safety net, not the primary fix — capAndDedupeText
// still trims a full-screen dump down to a bounded, deduped word list.
const fallbackWordLimit = 40

func pushRecentOCR(result *OCRResult) {
	if result == nil {
		return
	}
	pairMu.Lock()
	defer pairMu.Unlock()
	recentOCR = append(recentOCR, ocrSnap{text: result.Text, words: result.Words, timestamp: time.Now()})
	slog.Warn("pushRecentOCR", "buffer_size", len(recentOCR), "text_preview", result.Text[:min(len(result.Text), 40)])
	if len(recentOCR) > maxRecentOCR {
		recentOCR = recentOCR[len(recentOCR)-maxRecentOCR:]
	}
}

// findRecentOCRBefore returns the OCR context to associate with the given
// command. For coordinate-based tools it scopes the context to words near
// the actual target (x, y) instead of the whole screen. Otherwise, or if no
// words are found nearby, it falls back to a capped, deduped slice of the
// full screen text rather than the raw dump.
func findRecentOCRBefore(tool, argsJSON string) string {
	pairMu.Lock()
	var snap *ocrSnap
	now := time.Now()
	for i := len(recentOCR) - 1; i >= 0; i-- {
		age := now.Sub(recentOCR[i].timestamp)
		slog.Warn("findRecentOCRBefore check", "index", i, "age_ms", age.Milliseconds(), "bridge_window_ms", bridgeWindow.Milliseconds())
		if age <= bridgeWindow {
			snap = &recentOCR[i]
			break
		}
	}
	pairMu.Unlock()
	if snap == nil {
		slog.Warn("findRecentOCRBefore found nothing", "buffer_size", bridgeBufferSize())
		return ""
	}

	if coordTools[tool] {
		if coords := extractCoordsFromArgs(tool, argsJSON); len(coords) > 0 {
			cp := coords[0]
			if text := nearbyOCRText(snap.words, float64(cp.x), float64(cp.y), nearbyWordRadius, nearbyWordLimit); text != "" {
				return text
			}
		}
	}

	return capAndDedupeText(snap.text, fallbackWordLimit)
}

// nearbyOCRText returns the deduped text of the OCR words closest to
// (x, y) within radius, ordered nearest-first, capped at maxWords.
func nearbyOCRText(words []OCRWord, x, y, radius float64, maxWords int) string {
	type scored struct {
		text string
		dist float64
	}
	var candidates []scored
	for _, w := range words {
		cx := w.X + w.W/2
		cy := w.Y + w.H/2
		dx := cx - x
		dy := cy - y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= radius {
			candidates = append(candidates, scored{text: w.Text, dist: dist})
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].dist < candidates[j].dist })
	if len(candidates) > maxWords {
		candidates = candidates[:maxWords]
	}
	seen := make(map[string]bool, len(candidates))
	var parts []string
	for _, c := range candidates {
		t := strings.TrimSpace(c.text)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		parts = append(parts, t)
	}
	return strings.Join(parts, " ")
}

// capAndDedupeText trims text down to at most maxWords unique words,
// preserving first-seen order, instead of passing a raw full-screen dump
// through to the training pair.
func capAndDedupeText(text string, maxWords int) string {
	words := strings.Fields(text)
	seen := make(map[string]bool, len(words))
	out := make([]string, 0, maxWords)
	for _, w := range words {
		key := strings.ToLower(w)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, w)
		if len(out) >= maxWords {
			break
		}
	}
	return strings.Join(out, " ")
}

func tryCompletePair(afterText, windowTitle string) {
	pairMu.Lock()
	p := pendingCmd
	pt := pendingTime
	pendingCmd = nil
	pairMu.Unlock()
	slog.Warn("tryCompletePair", "has_pending", p != nil)
	if p == nil {
		return
	}
	age := time.Since(pt)
	slog.Warn("tryCompletePair pending age", "age_ms", age.Milliseconds())
	if age > bridgeWindow {
		slog.Warn("tryCompletePair expired")
		return
	}
	p.OCRAfter = afterText
	if p.WindowTitle == "" {
		p.WindowTitle = windowTitle
	}
	if p.OCRBefore != "" && p.Command != "" {
		slog.Warn("tryCompletePair logging pair", "ocr_before", p.OCRBefore[:min(len(p.OCRBefore), 30)])
		LogTrainingPair(*p)
	}
}

func BridgeDebugInfo() map[string]any {
	pairMu.Lock()
	defer pairMu.Unlock()
	info := map[string]any{
		"recent_ocr_count": len(recentOCR),
		"has_pending":      pendingCmd != nil,
	}
	if len(recentOCR) > 0 {
		last := recentOCR[len(recentOCR)-1]
		info["last_ocr_text"] = last.text[:min(len(last.text), 40)]
		info["last_ocr_age_ms"] = time.Since(last.timestamp).Milliseconds()
	}
	if pendingCmd != nil {
		info["pending_ocr_before"] = pendingCmd.OCRBefore[:min(len(pendingCmd.OCRBefore), 40)]
		info["pending_command"] = pendingCmd.Command
		info["pending_age_ms"] = time.Since(pendingTime).Milliseconds()
	}
	return info
}

func LogCommand(tool, args string, success bool, errText, windowTitle string, durationMs int64) {
	if !DataLog.LogKeys {
		return
	}
	dlogMu.Lock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			dlogMu.Unlock()
			return
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	successInt := 0
	if success {
		successInt = 1
	}
	dlogDB.Exec(`INSERT INTO command_log(session_id, source, tool, args, success, error_text, window_title, duration_ms, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"", "chain", tool, args, successInt, errText, windowTitle, durationMs, now)
	dlogMu.Unlock()
}

func LogChain(sessionID string, steps []ChainStep, result *ChainResult, durationMs int64) {
	if !DataLog.LogChain {
		return
	}
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return
		}
	}
	chainJSON, _ := json.Marshal(steps)
	resultJSON, _ := json.Marshal(result)
	successCount := 0
	failCount := 0
	for _, r := range result.Results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dlogDB.Exec(`INSERT INTO chain_log(session_id, source, step_count, success_count, fail_count, duration_ms, chain_json, result_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, "chain", len(steps), successCount, failCount, durationMs, string(chainJSON), string(resultJSON), now)
}

func LogOCRSnapshot(source, triggeredBy, windowTitle string, result *OCRResult) {
	if !DataLog.LogOCR {
		return
	}
	if result == nil {
		return
	}

	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return
		}
	}
	imgPath := saveScreenshotForDataLog()
	ocrJSON, _ := json.Marshal(result)
	now := time.Now().UTC().Format(time.RFC3339)
	dlogDB.Exec(`INSERT INTO ocr_log(session_id, source, image_path, ocr_text, ocr_json, text_length, word_count, window_title, triggered_by, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"", source, imgPath, result.Text, string(ocrJSON), len(result.Text), len(result.Words), windowTitle, triggeredBy, now)
}

type TrainingPairInput struct {
	OCRBefore string
	Command   string
	OCRAfter  string
	WindowTitle string
	Success   bool
}

func LogTrainingPair(in TrainingPairInput) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return
		}
	}
	successInt := 0
	if in.Success {
		successInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dlogDB.Exec(`INSERT INTO training_pairs(session_id, ocr_before_text, command_json, ocr_after_text, window_title, success, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		"", in.OCRBefore, in.Command, in.OCRAfter, in.WindowTitle, successInt, now)
}

// SaveAdaptiveStat persists per-tool timing/success aggregates so that
// agent_analyze's timing_stats and success_rates survive server restarts.
// AdaptiveEngine.RecordResult calls this on every recorded action; the
// in-memory ring buffers still provide fine-grained recent stddev/min/max
// for a live session, while this table is the durable long-term record.
func SaveAdaptiveStat(tool string, durationMs float64, success bool) {
	if tool == "" {
		return
	}
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return
		}
	}
	successInc, failInc := 0, 0
	if success {
		successInc = 1
	} else {
		failInc = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dlogDB.Exec(`INSERT INTO adaptive_stats(tool, success_count, fail_count, duration_count, duration_sum, duration_min, duration_max, updated_at)
		VALUES(?, ?, ?, 1, ?, ?, ?, ?)
		ON CONFLICT(tool) DO UPDATE SET
			success_count = adaptive_stats.success_count + excluded.success_count,
			fail_count = adaptive_stats.fail_count + excluded.fail_count,
			duration_count = adaptive_stats.duration_count + excluded.duration_count,
			duration_sum = adaptive_stats.duration_sum + excluded.duration_sum,
			duration_min = MIN(adaptive_stats.duration_min, excluded.duration_min),
			duration_max = MAX(adaptive_stats.duration_max, excluded.duration_max),
			updated_at = excluded.updated_at`,
		tool, successInc, failInc, durationMs, durationMs, durationMs, now)
}

// LoadPersistedStats reads the durable per-tool aggregates back out of the
// datalog DB so AdaptiveEngine can seed timing_stats/success_rates
// immediately after a restart, before any new live commands arrive.
func LoadPersistedStats() (map[string]*PersistedStat, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return nil, err
		}
	}
	rows, err := dlogDB.Query(`SELECT tool, success_count, fail_count, duration_count, duration_sum, duration_min, duration_max FROM adaptive_stats`)
	if err != nil {
		return nil, fmt.Errorf("load adaptive stats: %w", err)
	}
	defer rows.Close()
	result := make(map[string]*PersistedStat)
	for rows.Next() {
		var tool string
		p := &PersistedStat{}
		if err := rows.Scan(&tool, &p.SuccessCount, &p.FailCount, &p.DurationCount, &p.DurationSum, &p.DurationMin, &p.DurationMax); err != nil {
			continue
		}
		result[tool] = p
	}
	return result, nil
}

type DataLogQuery struct {
	Table     string `json:"table"`
	Source    string `json:"source,omitempty"`
	Tool      string `json:"tool,omitempty"`
	Success   *bool  `json:"success,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

type DataLogRow map[string]any

func QueryDataLog(q DataLogQuery) ([]DataLogRow, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return nil, err
		}
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 5000 {
		limit = 5000
	}
	var table string
	var cols string
	switch q.Table {
	case "commands", "command_log":
		table = "command_log"
		cols = "id, session_id, source, tool, args, success, error_text, window_title, duration_ms, created_at"
	case "chains", "chain_log":
		table = "chain_log"
		cols = "id, session_id, source, step_count, success_count, fail_count, duration_ms, created_at"
	case "ocr", "ocr_log":
		table = "ocr_log"
		cols = "id, session_id, source, image_path, ocr_text, text_length, word_count, window_title, triggered_by, created_at"
	case "pairs", "training_pairs":
		table = "training_pairs"
		cols = "id, session_id, ocr_before_text, command_json, ocr_after_text, window_title, success, created_at"
	default:
		table = "command_log"
		cols = "id, session_id, source, tool, args, success, error_text, window_title, duration_ms, created_at"
	}
	var where []string
	var args []any
	if q.Source != "" {
		where = append(where, "source = ?")
		args = append(args, q.Source)
	}
	if q.Tool != "" {
		where = append(where, "tool = ?")
		args = append(args, q.Tool)
	}
	if q.Success != nil {
		val := 0
		if *q.Success {
			val = 1
		}
		where = append(where, "success = ?")
		args = append(args, val)
	}
	query := fmt.Sprintf("SELECT %s FROM %s", cols, table)
	if len(where) > 0 {
		query += " WHERE " + joinWhere(where)
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, q.Offset)
	rows, err := dlogDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query datalog: %w", err)
	}
	defer rows.Close()
	var results []DataLogRow
	colNames, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(colNames))
		valPtrs := make([]any, len(colNames))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, fmt.Errorf("scan datalog: %w", err)
		}
		row := make(DataLogRow)
		for i, name := range colNames {
			row[name] = vals[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

type ExportTrainingOutput struct {
	Pairs []TrainingPairExport `json:"pairs"`
	Count int                  `json:"count"`
}

type TrainingPairExport struct {
	OCRBefore   string `json:"ocr_before"`
	Command     string `json:"command"`
	OCRAfter    string `json:"ocr_after"`
	WindowTitle string `json:"window_title"`
	Success     bool   `json:"success"`
	CreatedAt   string `json:"created_at"`
}

func ExportTrainingData(sessionID string, limit int) (*ExportTrainingOutput, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return nil, err
		}
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if sessionID != "" {
		rows, err = dlogDB.Query(`SELECT ocr_before_text, command_json, ocr_after_text, window_title, success, created_at
			FROM training_pairs WHERE session_id = ? ORDER BY id DESC LIMIT ?`, sessionID, limit)
	} else {
		rows, err = dlogDB.Query(`SELECT ocr_before_text, command_json, ocr_after_text, window_title, success, created_at
			FROM training_pairs ORDER BY id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("export training data: %w", err)
	}
	defer rows.Close()
	out := &ExportTrainingOutput{}
	for rows.Next() {
		var p TrainingPairExport
		var successInt int
		if err := rows.Scan(&p.OCRBefore, &p.Command, &p.OCRAfter, &p.WindowTitle, &successInt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan export: %w", err)
		}
		p.Success = successInt == 1
		out.Pairs = append(out.Pairs, p)
	}
	out.Count = len(out.Pairs)
	return out, rows.Err()
}

type DataLogStats struct {
	CommandCount    int `json:"command_count"`
	ChainCount      int `json:"chain_count"`
	OCRCount        int `json:"ocr_count"`
	TrainingPairs   int `json:"training_pairs"`
	TaskCount       int `json:"task_count"`
}

func DataLogStatsReport() (*DataLogStats, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB == nil {
		if err := InitDataLog(); err != nil {
			return nil, err
		}
	}
	s := &DataLogStats{}
	dlogDB.QueryRow("SELECT COUNT(*) FROM command_log").Scan(&s.CommandCount)
	dlogDB.QueryRow("SELECT COUNT(*) FROM chain_log").Scan(&s.ChainCount)
	dlogDB.QueryRow("SELECT COUNT(*) FROM ocr_log").Scan(&s.OCRCount)
	dlogDB.QueryRow("SELECT COUNT(*) FROM training_pairs").Scan(&s.TrainingPairs)
	dlogDB.QueryRow("SELECT COUNT(*) FROM task_log").Scan(&s.TaskCount)
	return s, nil
}
