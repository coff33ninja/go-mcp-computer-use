package actions

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type TimingStat struct {
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stddev"`
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

type ToolTiming struct {
	durations []float64
	mu        sync.RWMutex
}

const maxTimingSamples = 100

func (t *ToolTiming) Add(durMs float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.durations = append(t.durations, durMs)
	if len(t.durations) > maxTimingSamples {
		t.durations = t.durations[len(t.durations)-maxTimingSamples:]
	}
}

func (t *ToolTiming) Stat() *TimingStat {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.durations) == 0 {
		return nil
	}
	s := &TimingStat{Min: t.durations[0], Max: t.durations[0]}
	sum := 0.0
	for _, d := range t.durations {
		sum += d
		if d < s.Min {
			s.Min = d
		}
		if d > s.Max {
			s.Max = d
		}
	}
	s.Count = len(t.durations)
	s.Mean = sum / float64(s.Count)
	vsum := 0.0
	for _, d := range t.durations {
		vsum += (d - s.Mean) * (d - s.Mean)
	}
	s.StdDev = math.Sqrt(vsum / float64(s.Count))
	return s
}

type ToolSuccess struct {
	Success int `json:"success"`
	Fail    int `json:"fail"`
	mu      sync.RWMutex
}

func (s *ToolSuccess) Record(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if success {
		s.Success++
	} else {
		s.Fail++
	}
}

func (s *ToolSuccess) Rate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := s.Success + s.Fail
	if total == 0 {
		return 0
	}
	return float64(s.Success) / float64(total)
}

type WordIndex struct {
	word  string
	count int
}

type SequenceExample struct {
	OCRBefore string `json:"ocr_before"`
	Command   string `json:"command"`
	Success   bool   `json:"success"`
	Count     int    `json:"count"`
	Freq      float64 `json:"freq"`
}

type PredictedCoord struct {
	X          int     `json:"x,omitempty"`
	Y          int     `json:"y,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Samples    int     `json:"samples,omitempty"`
}

type coordPoint struct {
	x int
	y int
}

type coordSample struct {
	xSum    int64
	ySum    int64
	count   int
	success int
}

type PredictedAction struct {
	Command     string           `json:"command"`
	Confidence  float64          `json:"confidence"`
	SampleSize  int              `json:"sample_size"`
	SuccessRate float64          `json:"success_rate"`
	Coord       *PredictedCoord  `json:"coord,omitempty"`
}

type EngineAnalysis struct {
	TimingStats     map[string]*TimingStat     `json:"timing_stats"`
	SuccessRates    map[string]float64         `json:"success_rates"`
	TopSequences    []SequenceExample          `json:"top_sequences"`
	TotalCommands   int                        `json:"total_commands"`
	TotalSequences  int                        `json:"total_sequences"`
	LastTrained     string                     `json:"last_trained"`
}

type AdaptiveEngine struct {
	timings    map[string]*ToolTiming
	successes  map[string]*ToolSuccess
	sequences  []SequenceExample
	wordToCmds map[string]map[string]*cmdFreq
	coordIndex map[string]map[string]*coordSample
	totalCmds  int
	totalSeqs  int
	lastTrain  time.Time
	mu         sync.RWMutex
}

type cmdFreq struct {
	success int
	fail    int
}

var (
	Adaptive     = NewAdaptiveEngine()
	adaptiveOnce sync.Once
)

func NewAdaptiveEngine() *AdaptiveEngine {
	return &AdaptiveEngine{
		timings:    make(map[string]*ToolTiming),
		successes:  make(map[string]*ToolSuccess),
		wordToCmds: make(map[string]map[string]*cmdFreq),
		coordIndex: make(map[string]map[string]*coordSample),
	}
}

func (e *AdaptiveEngine) RecordTiming(tool string, durationMs float64) {
	e.mu.Lock()
	tt, ok := e.timings[tool]
	if !ok {
		tt = &ToolTiming{}
		e.timings[tool] = tt
	}
	e.mu.Unlock()
	tt.Add(durationMs)
}

func (e *AdaptiveEngine) RecordSuccess(tool string, success bool) {
	e.mu.Lock()
	ts, ok := e.successes[tool]
	if !ok {
		ts = &ToolSuccess{}
		e.successes[tool] = ts
	}
	e.mu.Unlock()
	ts.Record(success)
}

func (e *AdaptiveEngine) RecordResult(tool string, durationMs float64, success bool) {
	e.RecordTiming(tool, durationMs)
	e.RecordSuccess(tool, success)
}

func (e *AdaptiveEngine) GetTiming(tool string) *TimingStat {
	e.mu.RLock()
	tt, ok := e.timings[tool]
	e.mu.RUnlock()
	if !ok {
		return nil
	}
	return tt.Stat()
}

func (e *AdaptiveEngine) GetSuccessRate(tool string) float64 {
	e.mu.RLock()
	ts, ok := e.successes[tool]
	e.mu.RUnlock()
	if !ok {
		return 0
	}
	return ts.Rate()
}

func (e *AdaptiveEngine) SuggestDelay(tool string) float64 {
	base := 50.0
	stat := e.GetTiming(tool)
	rate := e.GetSuccessRate(tool)
	if stat == nil || stat.Count < 3 {
		return base
	}
	multiplier := 1.5
	if rate > 0 && rate < 0.5 {
		multiplier = 3.0
	} else if rate < 0.8 {
		multiplier = 2.0
	}
	delay := stat.Mean + multiplier*stat.StdDev
	if delay < base {
		return base
	}
	if delay > 5000 {
		return 5000
	}
	return math.Round(delay)
}

func extractToolFromJSON(cmdJSON string) string {
	var cmdData map[string]any
	if err := json.Unmarshal([]byte(cmdJSON), &cmdData); err != nil {
		return ""
	}
	if tool, ok := cmdData["tool"].(string); ok && tool != "" {
		return tool
	}
	return ""
}

func tokenize(text string) []string {
	var tokens []string
	for _, w := range strings.Fields(text) {
		w = strings.Trim(w, ".,:;!?\"'()[]{}/\\<>@#$%^&*+=~`|")
		w = strings.ToLower(w)
		if len(w) > 2 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

var coordTools = map[string]bool{
	"click":      true,
	"move_mouse": true,
	"hover":      true,
	"drag":       true,
}

func extractArgsFromJSON(cmdJSON string) string {
	var cmdData map[string]any
	if err := json.Unmarshal([]byte(cmdJSON), &cmdData); err != nil {
		return ""
	}
	switch a := cmdData["args"].(type) {
	case string:
		return a
	default:
		return ""
	}
}

func getIntArg(args map[string]any, key string) (int, bool) {
	// Try exact match first, then case-insensitive fallback
	v, ok := args[key]
	if !ok {
		for k, val := range args {
			if strings.EqualFold(k, key) {
				v = val
				ok = true
				break
			}
		}
		if !ok {
			return 0, false
		}
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func extractCoordsFromArgs(tool string, argsJSON string) []coordPoint {
	if !coordTools[tool] {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}
	switch tool {
	case "click", "move_mouse", "hover":
		x, xOK := getIntArg(args, "x")
		y, yOK := getIntArg(args, "y")
		if xOK && yOK {
			return []coordPoint{{x: x, y: y}}
		}
	case "drag":
		fx, fxOK := getIntArg(args, "from_x")
		fy, fyOK := getIntArg(args, "from_y")
		tx, txOK := getIntArg(args, "to_x")
		ty, tyOK := getIntArg(args, "to_y")
		var pts []coordPoint
		if fxOK && fyOK {
			pts = append(pts, coordPoint{x: fx, y: fy})
		}
		if txOK && tyOK {
			pts = append(pts, coordPoint{x: tx, y: ty})
		}
		return pts
	}
	return nil
}

func (e *AdaptiveEngine) TrainFromDatalog() error {
	q := DataLogQuery{Table: "pairs", Limit: 2000}
	rows, err := QueryDataLog(q)
	if err != nil {
		return fmt.Errorf("train from datalog: %w", err)
	}
	e.mu.Lock()
	e.wordToCmds = make(map[string]map[string]*cmdFreq)
	e.coordIndex = make(map[string]map[string]*coordSample)
	e.sequences = nil
	e.totalCmds = 0
	e.totalSeqs = 0
	e.mu.Unlock()

	for _, row := range rows {
		ocrBefore, _ := row["ocr_before_text"].(string)
		cmdJSON, _ := row["command_json"].(string)
		successInt, _ := row["success"].(int64)
		success := successInt == 1

		tool := extractToolFromJSON(cmdJSON)
		if tool == "" {
			tool = cmdJSON
		}
		if tool == "" {
			continue
		}

		argsRaw := extractArgsFromJSON(cmdJSON)
		tokens := tokenize(ocrBefore)
		coords := extractCoordsFromArgs(tool, argsRaw)
		e.mu.Lock()
		for _, tok := range tokens {
			cmds, ok := e.wordToCmds[tok]
			if !ok {
				cmds = make(map[string]*cmdFreq)
				e.wordToCmds[tok] = cmds
			}
			cf, ok := cmds[tool]
			if !ok {
				cf = &cmdFreq{}
				cmds[tool] = cf
			}
			if success {
				cf.success++
			} else {
				cf.fail++
			}

			for _, cp := range coords {
				toolMap, ok := e.coordIndex[tool]
				if !ok {
					toolMap = make(map[string]*coordSample)
					e.coordIndex[tool] = toolMap
				}
				cs, ok := toolMap[tok]
				if !ok {
					cs = &coordSample{}
					toolMap[tok] = cs
				}
				cs.xSum += int64(cp.x)
				cs.ySum += int64(cp.y)
				cs.count++
				if success {
					cs.success++
				}
			}
		}
		// Aggregate all coords for this tool under __learned__ key so
		// coordinate predictions survive restart even when per-token
		// samples are below the threshold.
		if len(coords) > 0 {
			for _, cp := range coords {
				toolMap, ok := e.coordIndex[tool]
				if !ok {
					toolMap = make(map[string]*coordSample)
					e.coordIndex[tool] = toolMap
				}
				cs, ok := toolMap["__learned__"]
				if !ok {
					cs = &coordSample{}
					toolMap["__learned__"] = cs
				}
				cs.xSum += int64(cp.x)
				cs.ySum += int64(cp.y)
				cs.count++
				if success {
					cs.success++
				}
			}
		}
		e.totalSeqs++
		e.totalCmds++
		e.mu.Unlock()
	}

	e.mu.Lock()
	e.lastTrain = time.Now()
	e.mu.Unlock()
	e.rebuildSequences()
	return nil
}

func (e *AdaptiveEngine) rebuildSequences() {
	e.mu.Lock()
	defer e.mu.Unlock()
	type aggKey struct {
		ocr   string
		cmd   string
	}
	agg := make(map[aggKey]*SequenceExample)
	for word, cmds := range e.wordToCmds {
		for cmd, cf := range cmds {
			key := aggKey{ocr: word, cmd: cmd}
			if existing, ok := agg[key]; ok {
				existing.Count += cf.success + cf.fail
				if cf.success > 0 {
					existing.Count = cf.success + cf.fail
				}
			} else {
				total := cf.success + cf.fail
				freq := float64(cf.success) / float64(total)
				agg[key] = &SequenceExample{
					OCRBefore: word,
					Command:   cmd,
					Success:   cf.success > cf.fail,
					Count:     total,
					Freq:      freq,
				}
			}
		}
	}
	e.sequences = nil
	for _, seq := range agg {
		e.sequences = append(e.sequences, *seq)
	}
	sort.Slice(e.sequences, func(i, j int) bool {
		return e.sequences[i].Count > e.sequences[j].Count
	})
	if len(e.sequences) > 100 {
		e.sequences = e.sequences[:100]
	}
}

func (e *AdaptiveEngine) PredictActions(ocrText string, limit int) []PredictedAction {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if limit <= 0 {
		limit = 5
	}
	tokens := tokenize(ocrText)
	if len(tokens) == 0 {
		return nil
	}
	cmdScores := make(map[string]*struct {
		score     float64
		samples   int
		successes int
	})
	for _, tok := range tokens {
		cmds, ok := e.wordToCmds[tok]
		if !ok {
			continue
		}
		for cmd, cf := range cmds {
			cs, ok := cmdScores[cmd]
			if !ok {
				cs = &struct {
					score     float64
					samples   int
					successes int
				}{}
				cmdScores[cmd] = cs
			}
			total := cf.success + cf.fail
			cs.score += float64(cf.success) / float64(total)
			cs.samples += total
			cs.successes += cf.success
		}
	}
	if len(cmdScores) == 0 {
		return nil
	}
	var results []PredictedAction
	for cmd, cs := range cmdScores {
		conf := cs.score / float64(len(tokens))
		if conf > 1.0 {
			conf = 1.0
		}
		sr := 0.0
		if cs.samples > 0 {
			sr = float64(cs.successes) / float64(cs.samples)
		}
		pa := PredictedAction{
			Command:     cmd,
			Confidence:  math.Round(conf*100) / 100,
			SampleSize:  cs.samples,
			SuccessRate: math.Round(sr*100) / 100,
		}
		if cmd == "click" || cmd == "move_mouse" || cmd == "hover" {
			pa.Coord = e.predictCoord(cmd, tokens)
		}
		results = append(results, pa)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func (e *AdaptiveEngine) predictCoord(tool string, tokens []string) *PredictedCoord {
	var txSum, tySum int64
	var tCount, tSuccess int

	// First, try matching against OCR-context tokens (per-token coord index).
	toolMap, toolOK := e.coordIndex[tool]
	if toolOK {
		for _, tok := range tokens {
			cs, ok := toolMap[tok]
			if !ok {
				continue
			}
			txSum += cs.xSum
			tySum += cs.ySum
			tCount += cs.count
			tSuccess += cs.success
		}
	}

	// If insufficient token-specific samples, fall back to the runtime-learned
	// __learned__ aggregate which accumulates across all invocations.
	if tCount < 3 && toolOK {
		if cs, ok := toolMap["__learned__"]; ok && cs.count >= 3 {
			return &PredictedCoord{
				X:          int(cs.xSum / int64(cs.count)),
				Y:          int(cs.ySum / int64(cs.count)),
				Confidence: float64(cs.success) / float64(cs.count),
				Samples:    cs.count,
			}
		}
	}

	if tCount < 3 {
		return nil
	}
	conf := float64(tSuccess) / float64(tCount)
	return &PredictedCoord{
		X:          int(txSum / int64(tCount)),
		Y:          int(tySum / int64(tCount)),
		Confidence: math.Round(conf*100) / 100,
		Samples:    tCount,
	}
}

func (e *AdaptiveEngine) LearnFromCommand(tool, argsJSON string, success bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	coords := extractCoordsFromArgs(tool, argsJSON)
	if len(coords) == 0 {
		return
	}
	for _, cp := range coords {
		toolMap, ok := e.coordIndex[tool]
		if !ok {
			toolMap = make(map[string]*coordSample)
			e.coordIndex[tool] = toolMap
		}
		key := "__learned__"
		cs, ok := toolMap[key]
		if !ok {
			cs = &coordSample{}
			toolMap[key] = cs
		}
		cs.xSum += int64(cp.x)
		cs.ySum += int64(cp.y)
		cs.count++
		if success {
			cs.success++
		}
	}
}

func (e *AdaptiveEngine) LearnFromCommandWithContext(tool, argsJSON, ocrBefore string, success bool) {
	if ocrBefore == "" {
		return
	}
	tokens := tokenize(ocrBefore)
	if len(tokens) == 0 {
		return
	}
	coords := extractCoordsFromArgs(tool, argsJSON)
	if len(coords) == 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, tok := range tokens {
		toolMap, ok := e.coordIndex[tool]
		if !ok {
			toolMap = make(map[string]*coordSample)
			e.coordIndex[tool] = toolMap
		}
		for _, cp := range coords {
			cs, ok := toolMap[tok]
			if !ok {
				cs = &coordSample{}
				toolMap[tok] = cs
			}
			cs.xSum += int64(cp.x)
			cs.ySum += int64(cp.y)
			cs.count++
			if success {
				cs.success++
			}
		}
	}
}

func (e *AdaptiveEngine) Analyze() *EngineAnalysis {
	e.mu.RLock()
	defer e.mu.RUnlock()
	analysis := &EngineAnalysis{
		TimingStats:    make(map[string]*TimingStat),
		SuccessRates:   make(map[string]float64),
		TotalCommands:  e.totalCmds,
		TotalSequences: e.totalSeqs,
		TopSequences:   e.sequences,
	}
	if !e.lastTrain.IsZero() {
		analysis.LastTrained = e.lastTrain.Format(time.RFC3339)
	}
	for tool, tt := range e.timings {
		if stat := tt.Stat(); stat != nil {
			analysis.TimingStats[tool] = stat
		}
	}
	for tool, ts := range e.successes {
		analysis.SuccessRates[tool] = ts.Rate()
	}
	return analysis
}

func (e *AdaptiveEngine) RecordCommand(tool, argsJSON string, success bool, durationMs int64) {
	if !DataLog.LogKeys {
		return
	}
	e.RecordResult(tool, float64(durationMs), success)
	LogCommand(tool, argsJSON, success, "", "", durationMs)
}

func EnsureAdaptive() {
	adaptiveOnce.Do(func() {
		go func() {
			if err := Adaptive.TrainFromDatalog(); err != nil {
				return
			}
		}()
	})
}
