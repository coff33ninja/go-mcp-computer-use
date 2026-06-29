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

type PredictedAction struct {
	Command    string  `json:"command"`
	Confidence float64 `json:"confidence"`
	SampleSize int     `json:"sample_size"`
	SuccessRate float64 `json:"success_rate"`
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

func (e *AdaptiveEngine) TrainFromDatalog() error {
	q := DataLogQuery{Table: "pairs", Limit: 2000}
	rows, err := QueryDataLog(q)
	if err != nil {
		return fmt.Errorf("train from datalog: %w", err)
	}
	e.mu.Lock()
	e.wordToCmds = make(map[string]map[string]*cmdFreq)
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

		tokens := tokenize(ocrBefore)
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
		results = append(results, PredictedAction{
			Command:     cmd,
			Confidence:  math.Round(conf*100) / 100,
			SampleSize:  cs.samples,
			SuccessRate: math.Round(sr*100) / 100,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
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
