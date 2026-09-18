package actions

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Prediction outcome statuses. UNKNOWN is first-class: never train on it.
// Only hit and miss (plus recovered-from-miss) are supervised signals.
const (
	PredStatusPending   = "pending"
	PredStatusHit       = "hit"
	PredStatusMiss      = "miss"
	PredStatusUnknown   = "unknown"
	PredStatusRecovered = "recovered"
)

const defaultHitRadiusPx = 80
const pendingPredictionWindow = 2 * time.Minute
const unknownAfter = 10 * time.Minute

// MLPredictionInput is one engine guess. Prediction fields are immutable
// after insert; only resolution columns may change later.
type MLPredictionInput struct {
	Engine       string
	Query        string
	OCRText      string
	WindowTitle  string
	PredTool     string
	PredX        int
	PredY        int
	Confidence   float64
	ModelVersion string
	// ExpectedTarget is optional intent (e.g. OCR word / element label).
	ExpectedTarget string
}

// MLPredictionOutcomeStats is reported on ml_status.
type MLPredictionOutcomeStats struct {
	Pending         int     `json:"pending"`
	Hit             int     `json:"hit"`
	Miss            int     `json:"miss"`
	Unknown         int     `json:"unknown"`
	Recovered       int     `json:"recovered"`
	Total           int     `json:"total"`
	HitRate         float64 `json:"hit_rate"`          // (hit+recovered)/(hit+miss+recovered)
	RecoveryRate    float64 `json:"recovery_rate"`     // recovered/(miss+recovered) among initially-miss
	UnknownRate     float64 `json:"unknown_rate"`      // unknown/total
	LastHourHitRate float64 `json:"last_hour_hit_rate"`
	EligibleTrain   int     `json:"eligible_train"`    // hit+miss+recovered
}

func shortHash(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func ensureLedgerDB() (*sql.DB, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	if dlogDB != nil {
		return dlogDB, nil
	}
	if err := InitDataLog(); err != nil {
		return nil, err
	}
	if dlogDB == nil {
		return nil, fmt.Errorf("datalog db not initialized")
	}
	// Older DBs may predate ml_predictions.
	if err := ensureMLPredictionsSchema(dlogDB); err != nil {
		return nil, err
	}
	return dlogDB, nil
}

func ensureMLPredictionsSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ml_predictions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			engine TEXT NOT NULL DEFAULT '',
			query TEXT NOT NULL DEFAULT '',
			query_hash TEXT NOT NULL DEFAULT '',
			ocr_hash TEXT NOT NULL DEFAULT '',
			window_hash TEXT NOT NULL DEFAULT '',
			window_title TEXT NOT NULL DEFAULT '',
			pred_tool TEXT NOT NULL DEFAULT '',
			pred_x INTEGER NOT NULL DEFAULT 0,
			pred_y INTEGER NOT NULL DEFAULT 0,
			confidence REAL NOT NULL DEFAULT 0,
			model_version TEXT NOT NULL DEFAULT '',
			expected_target TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			outcome_source TEXT NOT NULL DEFAULT '',
			actual_tool TEXT NOT NULL DEFAULT '',
			actual_x INTEGER NOT NULL DEFAULT 0,
			actual_y INTEGER NOT NULL DEFAULT 0,
			verification_data TEXT NOT NULL DEFAULT '{}',
			latency_ms INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			resolved_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mlpred_status ON ml_predictions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_mlpred_created ON ml_predictions(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_mlpred_tool ON ml_predictions(pred_tool)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// Best-effort ALTERs for DBs created with the v0.3.9-preview schema.
	alters := []string{
		`ALTER TABLE ml_predictions ADD COLUMN query_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE ml_predictions ADD COLUMN ocr_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE ml_predictions ADD COLUMN window_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE ml_predictions ADD COLUMN expected_target TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE ml_predictions ADD COLUMN verification_data TEXT NOT NULL DEFAULT '{}'`,
	}
	for _, s := range alters {
		_, _ = db.Exec(s) // ignore duplicate column
	}
	return nil
}

// LogMLPrediction inserts an immutable prediction row (pending).
func LogMLPrediction(in MLPredictionInput) int64 {
	db, err := ensureLedgerDB()
	if err != nil {
		slog.Debug("ml ledger: init failed", "err", err)
		return 0
	}
	if in.Engine == "" {
		in.Engine = MLSourceStatistical
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dlogMu.Lock()
	defer dlogMu.Unlock()
	res, err := db.Exec(`INSERT INTO ml_predictions(
		engine, query, query_hash, ocr_hash, window_hash, window_title,
		pred_tool, pred_x, pred_y, confidence, model_version,
		expected_target, status, verification_data, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		in.Engine,
		truncateStr(in.Query, 500),
		shortHash(in.Query),
		shortHash(in.OCRText),
		shortHash(in.WindowTitle),
		in.WindowTitle,
		in.PredTool,
		in.PredX,
		in.PredY,
		in.Confidence,
		in.ModelVersion,
		in.ExpectedTarget,
		PredStatusPending,
		`{}`,
		now,
	)
	if err != nil {
		slog.Debug("ml ledger: insert failed", "err", err)
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}

// LogMLPredictionsFromActions logs up to topN predicted actions.
func LogMLPredictionsFromActions(engine, query, ocrText, windowTitle, modelVersion string, preds []PredictedAction) {
	for i, p := range preds {
		if i >= 3 {
			break
		}
		eng := p.Source
		if eng == "" {
			eng = engine
		}
		x, y := 0, 0
		if p.Coord != nil {
			x, y = p.Coord.X, p.Coord.Y
		}
		LogMLPrediction(MLPredictionInput{
			Engine:       eng,
			Query:        query,
			OCRText:      ocrText,
			WindowTitle:  windowTitle,
			PredTool:     p.Command,
			PredX:        x,
			PredY:        y,
			Confidence:   p.Confidence,
			ModelVersion: modelVersion,
		})
	}
}

type pendingPred struct {
	id     int64
	tool   string
	x, y   int
	hasXY  bool
	engine string
	at     time.Time
}

type resolutionPatch struct {
	status       string
	outcomeSrc   string
	actualTool   string
	actualX      int
	actualY      int
	verification map[string]any
	latencyMs    int64
}

// ResolveMLPredictionsForAction updates ONLY resolution columns after a real
// desktop action. Prediction geometry/identity stays immutable.
//
// Invariants:
//   - unknown ≠ miss (no evidence → do not train)
//   - success far from preds → leave pending (agent may have ignored ML)
//   - miss then quick successful retry → recovered (task can still succeed)
func ResolveMLPredictionsForAction(tool string, x, y int32, success bool, windowTitle string, evidence string) {
	db, err := ensureLedgerDB()
	if err != nil {
		return
	}
	expireStalePredictions(db)

	now := time.Now().UTC()
	cutoff := now.Add(-pendingPredictionWindow).UTC().Format(time.RFC3339)
	rows, err := queryPending(db, cutoff)
	if err != nil || len(rows) == 0 {
		return
	}

	radius := hitRadius()
	actualX, actualY := int(x), int(y)
	src := "action"
	if evidence != "" {
		src = evidence
	}

	var near []pendingPred
	var sameTool []pendingPred
	for _, p := range rows {
		if toolMatches(p.tool, tool) {
			sameTool = append(sameTool, p)
		}
		if p.hasXY && dist(p.x, p.y, actualX, actualY) <= float64(radius) {
			near = append(near, p)
		}
	}

	resolvedAt := now.UTC().Format(time.RFC3339)
	_ = resolvedAt

	verif := map[string]any{
		"evidence":    src,
		"success":     success,
		"hit_radius":  radius,
		"window":      windowTitle,
		"actual_tool": tool,
	}

	if !success {
		targets := near
		if len(targets) == 0 {
			targets = filterSameToolRecent(sameTool, float64(actualX), float64(actualY), float64(radius*2))
		}
		for _, p := range targets {
			applyResolution(db, p.id, resolutionPatch{
				status:       PredStatusMiss,
				outcomeSrc:   src,
				actualTool:   tool,
				actualX:      actualX,
				actualY:      actualY,
				verification: verif,
			})
		}
		return
	}

	hitCount := 0
	for _, p := range near {
		applyResolution(db, p.id, resolutionPatch{
			status:       PredStatusHit,
			outcomeSrc:   src,
			actualTool:   tool,
			actualX:      actualX,
			actualY:      actualY,
			verification: verif,
		})
		hitCount++
		if Adaptive != nil && p.engine != MLSourceTransformer {
			Adaptive.MLTeach(truncateStr(fmt.Sprintf("%s %s", tool, windowTitle), 80), "", tool, actualX, actualY, true)
		}
	}
	if hitCount == 0 {
		return
	}

	// Recovery: earlier miss on same tool (and window if known) → recovered.
	// Task-level success is separate from initial prediction quality.
	recoverRecentMisses(db, tool, windowTitle, actualX, actualY, radius, now)
}

func toolMatches(predTool, actualTool string) bool {
	if predTool == actualTool {
		return true
	}
	// click-family predictions can be confirmed by a click
	switch actualTool {
	case "click":
		return predTool == "click" || predTool == "hover" || predTool == "move_mouse" ||
			predTool == "double_click" || predTool == "find_text_and_click"
	}
	return false
}

func queryPending(db *sql.DB, cutoff string) ([]pendingPred, error) {
	dlogMu.Lock()
	defer dlogMu.Unlock()
	rows, err := db.Query(`SELECT id, pred_tool, pred_x, pred_y, engine, created_at
		FROM ml_predictions WHERE status=? AND created_at >= ?`, PredStatusPending, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pendingPred
	for rows.Next() {
		var p pendingPred
		var created string
		if err := rows.Scan(&p.id, &p.tool, &p.x, &p.y, &p.engine, &created); err != nil {
			continue
		}
		p.hasXY = p.x != 0 || p.y != 0
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			p.at = t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func filterSameToolRecent(same []pendingPred, x, y, maxDist float64) []pendingPred {
	var out []pendingPred
	for _, p := range same {
		if !p.hasXY {
			continue
		}
		if dist(p.x, p.y, int(x), int(y)) <= maxDist {
			out = append(out, p)
		}
	}
	return out
}

// applyResolution mutates resolution columns only (prediction stays immutable).
func applyResolution(db *sql.DB, id int64, patch resolutionPatch) {
	verifJSON := `{}`
	if len(patch.verification) > 0 {
		if b, err := json.Marshal(patch.verification); err == nil {
			verifJSON = string(b)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dlogMu.Lock()
	defer dlogMu.Unlock()
	_, err := db.Exec(`UPDATE ml_predictions SET
		status=?, outcome_source=?, actual_tool=?, actual_x=?, actual_y=?,
		verification_data=?, latency_ms=?, resolved_at=?
		WHERE id=? AND status IN (?,?)`,
		patch.status, patch.outcomeSrc, patch.actualTool, patch.actualX, patch.actualY,
		verifJSON, patch.latencyMs, now,
		id, PredStatusPending, PredStatusMiss, // allow miss→recovered
	)
	if err != nil {
		slog.Debug("ml ledger: resolve failed", "err", err, "id", id)
	}
}

func expireStalePredictions(db *sql.DB) {
	cutoff := time.Now().UTC().Add(-unknownAfter).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	dlogMu.Lock()
	defer dlogMu.Unlock()
	db.Exec(`UPDATE ml_predictions SET status=?, outcome_source='timeout', resolved_at=?
		WHERE status=? AND created_at < ?`,
		PredStatusUnknown, now, PredStatusPending, cutoff)
}

func recoverRecentMisses(db *sql.DB, tool, windowTitle string, x, y int, radius int, now time.Time) {
	since := now.Add(-45 * time.Second).UTC().Format(time.RFC3339)
	dlogMu.Lock()
	defer dlogMu.Unlock()
	rows, err := db.Query(`SELECT id, pred_x, pred_y, window_title FROM ml_predictions
		WHERE status=? AND pred_tool=? AND created_at >= ?`, PredStatusMiss, tool, since)
	if err != nil {
		return
	}
	defer rows.Close()
	type missRow struct {
		id     int64
		x, y   int
		win    string
		hasXY  bool
	}
	var misses []missRow
	for rows.Next() {
		var m missRow
		if err := rows.Scan(&m.id, &m.x, &m.y, &m.win); err != nil {
			continue
		}
		m.hasXY = m.x != 0 || m.y != 0
		misses = append(misses, m)
	}
	nowStr := now.UTC().Format(time.RFC3339)
	for _, m := range misses {
		if windowTitle != "" && m.win != "" && m.win != windowTitle {
			continue
		}
		near := false
		if m.hasXY {
			near = dist(m.x, m.y, x, y) <= float64(radius*3)
		} else {
			// no coords on the miss — same tool retry shortly after still counts
			near = true
		}
		if !near {
			continue
		}
		verif, _ := json.Marshal(map[string]any{
			"recovery": true, "actual_x": x, "actual_y": y, "window": windowTitle,
		})
		db.Exec(`UPDATE ml_predictions SET status=?, outcome_source='recovery',
			actual_x=?, actual_y=?, verification_data=?, resolved_at=?
			WHERE id=? AND status=?`,
			PredStatusRecovered, x, y, string(verif), nowStr, m.id, PredStatusMiss)
	}
}

// MLPredictionOutcomeStatsFromDB aggregates ledger outcomes for ml_status.
func MLPredictionOutcomeStatsFromDB() MLPredictionOutcomeStats {
	var st MLPredictionOutcomeStats
	db, err := ensureLedgerDB()
	if err != nil {
		return st
	}
	expireStalePredictions(db)

	dlogMu.Lock()
	defer dlogMu.Unlock()
	rows, err := db.Query(`SELECT status, COUNT(*) FROM ml_predictions GROUP BY status`)
	if err != nil {
		return st
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			continue
		}
		st.Total += n
		switch status {
		case PredStatusPending:
			st.Pending = n
		case PredStatusHit:
			st.Hit = n
		case PredStatusMiss:
			st.Miss = n
		case PredStatusUnknown:
			st.Unknown = n
		case PredStatusRecovered:
			st.Recovered = n
		}
	}
	labeled := st.Hit + st.Miss + st.Recovered
	st.EligibleTrain = labeled
	if labeled > 0 {
		st.HitRate = float64(st.Hit+st.Recovered) / float64(labeled)
	}
	initialMisses := st.Miss + st.Recovered
	if initialMisses > 0 {
		st.RecoveryRate = float64(st.Recovered) / float64(initialMisses)
	}
	if st.Total > 0 {
		st.UnknownRate = float64(st.Unknown) / float64(st.Total)
	}
	since := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	var hHit, hMiss, hRec int
	row := db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN status='hit' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status='miss' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status='recovered' THEN 1 ELSE 0 END),0)
		FROM ml_predictions WHERE created_at >= ? AND status IN ('hit','miss','recovered')`, since)
	_ = row.Scan(&hHit, &hMiss, &hRec)
	hl := hHit + hMiss + hRec
	if hl > 0 {
		st.LastHourHitRate = float64(hHit+hRec) / float64(hl)
	}
	return st
}

// SupervisedLedgerSamples returns ledger rows eligible for supervised learning.
// Only hit/miss/recovered — never unknown/pending.
func SupervisedLedgerSamples(limit int) []map[string]any {
	db, err := ensureLedgerDB()
	if err != nil || limit <= 0 {
		return nil
	}
	dlogMu.Lock()
	defer dlogMu.Unlock()
	rows, err := db.Query(`SELECT id, engine, query, pred_tool, pred_x, pred_y, confidence,
		status, actual_tool, actual_x, actual_y, created_at
		FROM ml_predictions WHERE status IN (?,?,?) ORDER BY id DESC LIMIT ?`,
		PredStatusHit, PredStatusMiss, PredStatusRecovered, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var engine, query, predTool, status, actualTool, created string
		var px, py, ax, ay int
		var conf float64
		if err := rows.Scan(&id, &engine, &query, &predTool, &px, &py, &conf,
			&status, &actualTool, &ax, &ay, &created); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"id": id, "engine": engine, "query": query,
			"pred_tool": predTool, "pred_x": px, "pred_y": py,
			"confidence": conf, "status": status,
			"actual_tool": actualTool, "actual_x": ax, "actual_y": ay,
			"created_at": created,
		})
	}
	return out
}

func hitRadius() int {
	sw, _ := ScreenSize()
	if sw <= 0 {
		return defaultHitRadiusPx
	}
	r := int(float64(sw) * 0.03)
	if r < defaultHitRadiusPx {
		r = defaultHitRadiusPx
	}
	if r > 200 {
		r = 200
	}
	return r
}

func dist(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}

func truncateStr(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
