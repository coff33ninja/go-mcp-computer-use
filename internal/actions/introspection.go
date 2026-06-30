package actions

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type TaskInput struct {
	Description string `json:"description"`
}

type TaskEndInput struct {
	Summary string `json:"summary,omitempty"`
	Success bool   `json:"success,omitempty"`
}

type TaskInfo struct {
	ID           int64  `json:"id"`
	Description  string `json:"description"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	StepCount    int    `json:"step_count"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	DurationMs   int64  `json:"duration_ms"`
	Insights     any    `json:"insights,omitempty"`
}

type TaskInsights struct {
	SlowestTools    []ToolStat   `json:"slowest_tools"`
	MostFailedTools []ToolStat   `json:"most_failed_tools"`
	OCRStats        OCRStats     `json:"ocr_stats"`
	RepeatPatterns  []Pattern    `json:"repeat_patterns"`
	Suggestions     []string     `json:"suggestions"`
	Sequence        []SeqStep    `json:"sequence"`
}

type SeqStep struct {
	Tool       string `json:"tool"`
	Args       string `json:"args"`
	Success    bool   `json:"success"`
	DurationMs int64  `json:"duration_ms"`
}

type cmdEntry struct {
	tool       string
	args       string
	success    bool
	durationMs int64
}

type ToolStat struct {
	Tool         string  `json:"tool"`
	Count        int     `json:"count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	FailCount    int    `json:"fail_count"`
}

type OCRStats struct {
	TotalCalls  int     `json:"total_calls"`
	TotalTextLen int    `json:"total_text_length"`
	AvgTextLen  float64 `json:"avg_text_length"`
}

type Pattern struct {
	Tools  []string `json:"tools"`
	Count  int      `json:"count"`
}

var (
	taskActive   bool
	taskID       int64
	taskDesc     string
	taskStart    time.Time
	taskMu       sync.Mutex
)

func TaskBegin(in TaskInput) (*TaskInfo, error) {
	taskMu.Lock()
	defer taskMu.Unlock()

	if err := InitDataLog(); err != nil {
		return nil, fmt.Errorf("task_begin: %w", err)
	}

	taskActive = true
	taskDesc = in.Description
	taskStart = time.Now()

	now := taskStart.UTC().Format(time.RFC3339)
	dlogMu.Lock()
	res, err := dlogDB.Exec(`INSERT INTO task_log(session_id, description, start_time, step_count, success_count, fail_count, total_duration_ms, created_at)
		VALUES(?, ?, ?, 0, 0, 0, 0, ?)`,
		"", in.Description, now, now)
	if err != nil {
		dlogMu.Unlock()
		return nil, fmt.Errorf("task_begin insert: %w", err)
	}
	taskID, _ = res.LastInsertId()
	dlogMu.Unlock()

	slog.Warn("TaskBegin", "id", taskID, "description", in.Description)
	return &TaskInfo{ID: taskID, Description: in.Description, StartTime: now}, nil
}

func TaskEnd(in TaskEndInput) (*TaskInfo, error) {
	taskMu.Lock()
	taskWasActive := taskActive
	taskWasID := taskID
	taskWasDesc := taskDesc
	taskWasStart := taskStart
	taskActive = false
	taskMu.Unlock()

	if !taskWasActive {
		return nil, fmt.Errorf("no active task to end")
	}

	now := time.Now()
	durationMs := now.Sub(taskWasStart).Milliseconds()
	nowStr := now.UTC().Format(time.RFC3339)
	startStr := taskWasStart.UTC().Format(time.RFC3339)

	dlogMu.Lock()
	if dlogDB == nil {
		dlogMu.Unlock()
		return nil, fmt.Errorf("datalog not initialized")
	}

	insights := mineTaskInsights(dlogDB, startStr, nowStr)

	insightsJSON, _ := json.Marshal(insights)

	stepCount := 0
	successCount := 0
	failCount := 0
	dlogDB.QueryRow("SELECT COUNT(*), COALESCE(SUM(CASE WHEN success=1 THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN success=0 THEN 1 ELSE 0 END),0) FROM command_log WHERE created_at >= ? AND created_at <= ?",
		startStr, nowStr).Scan(&stepCount, &successCount, &failCount)

	_, err := dlogDB.Exec(`UPDATE task_log SET end_time=?, step_count=?, success_count=?, fail_count=?, total_duration_ms=?, insights=?, description=? WHERE id=?`,
		nowStr, stepCount, successCount, failCount, durationMs, string(insightsJSON), taskWasDesc, taskWasID)
	dlogMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("task_end update: %w", err)
	}

	slog.Warn("TaskEnd", "id", taskWasID, "steps", stepCount, "duration_ms", durationMs)
	return &TaskInfo{
		ID: taskWasID, Description: taskWasDesc,
		StartTime: startStr, EndTime: nowStr,
		StepCount: stepCount, SuccessCount: successCount, FailCount: failCount,
		DurationMs: durationMs, Insights: insights,
	}, nil
}

func mineTaskInsights(db *sql.DB, startStr, endStr string) *TaskInsights {
	ins := &TaskInsights{}

	rows, err := db.Query(`SELECT tool, args, success, duration_ms FROM command_log WHERE created_at >= ? AND created_at <= ? ORDER BY id ASC`, startStr, endStr)
	if err != nil {
		slog.Warn("mineTaskInsights query", "error", err)
		return ins
	}
	defer rows.Close()

	var entries []cmdEntry
	toolCounts := map[string]int{}
	toolDurations := map[string]int64{}
	toolFails := map[string]int{}

	for rows.Next() {
		var e cmdEntry
		var successInt int
		if err := rows.Scan(&e.tool, &e.args, &successInt, &e.durationMs); err != nil {
			continue
		}
		e.success = successInt == 1
		entries = append(entries, e)
		toolCounts[e.tool]++
		toolDurations[e.tool] += e.durationMs
		if !e.success {
			toolFails[e.tool]++
		}
	}

	for _, e := range entries {
		ins.Sequence = append(ins.Sequence, SeqStep{
			Tool: e.tool, Args: e.args,
			Success: e.success, DurationMs: e.durationMs,
		})
	}

	for tool, count := range toolCounts {
		avgDur := float64(0)
		if count > 0 {
			avgDur = float64(toolDurations[tool]) / float64(count)
		}
		stat := ToolStat{Tool: tool, Count: count, AvgDurationMs: avgDur, FailCount: toolFails[tool]}
		ins.SlowestTools = append(ins.SlowestTools, stat)
		if toolFails[tool] > 0 {
			ins.MostFailedTools = append(ins.MostFailedTools, stat)
		}
	}

	ins.SlowestTools = sortByDuration(ins.SlowestTools)
	ins.MostFailedTools = sortByFails(ins.MostFailedTools)

	if len(ins.SlowestTools) > 5 {
		ins.SlowestTools = ins.SlowestTools[:5]
	}
	if len(ins.MostFailedTools) > 5 {
		ins.MostFailedTools = ins.MostFailedTools[:5]
	}

	ocrRows, err := db.Query(`SELECT ocr_text FROM ocr_log WHERE created_at >= ? AND created_at <= ?`, startStr, endStr)
	if err == nil {
		defer ocrRows.Close()
		for ocrRows.Next() {
			var text string
			if err := ocrRows.Scan(&text); err == nil {
				ins.OCRStats.TotalCalls++
				ins.OCRStats.TotalTextLen += len(text)
			}
		}
		if ins.OCRStats.TotalCalls > 0 {
			ins.OCRStats.AvgTextLen = float64(ins.OCRStats.TotalTextLen) / float64(ins.OCRStats.TotalCalls)
		}
	}

	ins.RepeatPatterns = findRepeatedPatterns(entries)

	ins.Suggestions = generateSuggestions(ins)

	return ins
}

func findRepeatedPatterns(entries []cmdEntry) []Pattern {
	if len(entries) < 4 {
		return nil
	}
	toolSeq := make([]string, len(entries))
	for i, e := range entries {
		toolSeq[i] = e.tool
	}

	seen := map[string]bool{}
	var patterns []Pattern

	for window := 2; window <= 4; window++ {
		counts := map[string]int{}
		for i := 0; i <= len(toolSeq)-window; i++ {
			key := joinSeq(toolSeq[i : i+window])
			counts[key]++
		}
		for key, count := range counts {
			if count >= 2 && !seen[key] {
				seen[key] = true
				patterns = append(patterns, Pattern{
					Tools: splitSeq(key),
					Count: count,
				})
			}
		}
	}
	return patterns
}

func joinSeq(s []string) string {
	b := make([]byte, 0, len(s)*20)
	for i, v := range s {
		if i > 0 {
			b = append(b, '|')
		}
		b = append(b, v...)
	}
	return string(b)
}

func splitSeq(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func generateSuggestions(ins *TaskInsights) []string {
	var s []string

	for _, ts := range ins.MostFailedTools {
		if ts.FailCount > 1 {
			s = append(s, fmt.Sprintf("Tool '%s' failed %d/%d times — consider pre-validation or fallback", ts.Tool, ts.FailCount, ts.Count))
		} else {
			s = append(s, fmt.Sprintf("Tool '%s' failed once — review args: confidence=<1.0", ts.Tool))
		}
	}

	if len(ins.SlowestTools) > 0 && ins.SlowestTools[0].AvgDurationMs > 2000 {
		s = append(s, fmt.Sprintf("Tool '%s' averaged %.0fms — consider reducing frequency or batching", ins.SlowestTools[0].Tool, ins.SlowestTools[0].AvgDurationMs))
	}

	if ins.OCRStats.TotalCalls > 5 && ins.OCRStats.AvgTextLen < 10 {
		s = append(s, "OCR consistently returns short text — region may be too small or empty")
	}

	if len(ins.Sequence) > 20 {
		s = append(s, fmt.Sprintf("Task used %d steps — consider using chain tool for self-contained sequences", len(ins.Sequence)))
	}

	return s
}

func sortByDuration(stats []ToolStat) []ToolStat {
	n := len(stats)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if stats[j].AvgDurationMs > stats[i].AvgDurationMs {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	return stats
}

func sortByFails(stats []ToolStat) []ToolStat {
	n := len(stats)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if stats[j].FailCount > stats[i].FailCount {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	return stats
}

func IntrospectionAnalyze() ([]TaskInfo, error) {
	if err := InitDataLog(); err != nil {
		return nil, fmt.Errorf("introspection_analyze: %w", err)
	}
	dlogMu.Lock()
	defer dlogMu.Unlock()

	rows, err := dlogDB.Query(`SELECT id, description, start_time, COALESCE(end_time,''), COALESCE(step_count,0), COALESCE(success_count,0), COALESCE(fail_count,0), COALESCE(total_duration_ms,0) FROM task_log ORDER BY id DESC LIMIT 20`)
	if err != nil {
		return nil, fmt.Errorf("query task_log: %w", err)
	}
	defer rows.Close()

	var tasks []TaskInfo
	for rows.Next() {
		var t TaskInfo
		if err := rows.Scan(&t.ID, &t.Description, &t.StartTime, &t.EndTime, &t.StepCount, &t.SuccessCount, &t.FailCount, &t.DurationMs); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func TaskIsActive() bool {
	taskMu.Lock()
	defer taskMu.Unlock()
	return taskActive
}
