package actions

import (
	"testing"
)

func TestUniqueTokens(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "preserves first-seen order",
			input: []string{"a", "b", "a", "c", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "all duplicates",
			input: []string{"x", "x", "x"},
			want:  []string{"x"},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "single element",
			input: []string{"only"},
			want:  []string{"only"},
		},
		{
			name:  "empty strings",
			input: []string{"", "", "hello", ""},
			want:  []string{"", "hello"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := uniqueTokens(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("uniqueTokens(%v) returned %d items, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("uniqueTokens(%v)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestAnalyzePersistedFallbackNoLiveSamples(t *testing.T) {
	e := NewAdaptiveEngine()
	// No live timings/successes recorded — Analyze should fall back to persisted.
	e.persisted = map[string]*PersistedStat{
		"click": {
			SuccessCount:  8,
			FailCount:     2,
			DurationCount: 10,
			DurationSum:   500.0,
			DurationMin:   30.0,
			DurationMax:   80.0,
		},
		"type": {
			SuccessCount:  5,
			FailCount:     0,
			DurationCount: 5,
			DurationSum:   250.0,
			DurationMin:   40.0,
			DurationMax:   60.0,
		},
	}

	analysis := e.Analyze()

	// TimingStats should be filled from persisted.
	ts, ok := analysis.TimingStats["click"]
	if !ok {
		t.Fatal("expected timing_stats for 'click' from persisted")
	}
	if ts.Count != 10 {
		t.Errorf("click.Count = %d, want 10", ts.Count)
	}
	if ts.Mean != 50.0 {
		t.Errorf("click.Mean = %f, want 50.0", ts.Mean)
	}
	if ts.Min != 30.0 {
		t.Errorf("click.Min = %f, want 30.0", ts.Min)
	}
	if ts.Max != 80.0 {
		t.Errorf("click.Max = %f, want 80.0", ts.Max)
	}
	if ts.StdDev != 0 {
		t.Errorf("click.StdDev = %f, want 0 (not available from persisted)", ts.StdDev)
	}

	ts2, ok := analysis.TimingStats["type"]
	if !ok {
		t.Fatal("expected timing_stats for 'type' from persisted")
	}
	if ts2.Mean != 50.0 {
		t.Errorf("type.Mean = %f, want 50.0", ts2.Mean)
	}

	// SuccessRates should be filled from persisted.
	rate, ok := analysis.SuccessRates["click"]
	if !ok {
		t.Fatal("expected success_rates for 'click' from persisted")
	}
	if rate != 0.8 {
		t.Errorf("click success rate = %f, want 0.8", rate)
	}

	rate2, ok := analysis.SuccessRates["type"]
	if !ok {
		t.Fatal("expected success_rates for 'type' from persisted")
	}
	if rate2 != 1.0 {
		t.Errorf("type success rate = %f, want 1.0", rate2)
	}
}

func TestAnalyzePersistedFallbackLiveOverridesPersisted(t *testing.T) {
	e := NewAdaptiveEngine()
	// Persisted data says click has 100ms mean.
	e.persisted = map[string]*PersistedStat{
		"click": {
			SuccessCount:  10,
			FailCount:     0,
			DurationCount: 10,
			DurationSum:   1000.0,
			DurationMin:   50.0,
			DurationMax:   200.0,
		},
	}
	// Record live samples — these should take priority.
	e.RecordTiming("click", 10.0)
	e.RecordTiming("click", 20.0)
	e.RecordSuccess("click", true)

	analysis := e.Analyze()

	ts, ok := analysis.TimingStats["click"]
	if !ok {
		t.Fatal("expected timing_stats for 'click'")
	}
	// Live mean should be 15.0, not 100.0 from persisted.
	if ts.Mean != 15.0 {
		t.Errorf("click.Mean = %f, want 15.0 (live should override persisted)", ts.Mean)
	}
	// Live min/max should be from live samples.
	if ts.Min != 10.0 {
		t.Errorf("click.Min = %f, want 10.0", ts.Min)
	}
	if ts.Max != 20.0 {
		t.Errorf("click.Max = %f, want 20.0", ts.Max)
	}

	rate := analysis.SuccessRates["click"]
	if rate != 1.0 {
		t.Errorf("click success rate = %f, want 1.0 (live)", rate)
	}
}

func TestAnalyzePersistedFallbackPartialCoverage(t *testing.T) {
	e := NewAdaptiveEngine()
	// Only "type" has persisted data, but live only has "click".
	e.persisted = map[string]*PersistedStat{
		"type": {
			SuccessCount:  3,
			FailCount:     1,
			DurationCount: 4,
			DurationSum:   200.0,
			DurationMin:   40.0,
			DurationMax:   60.0,
		},
	}
	e.RecordTiming("click", 50.0)
	e.RecordSuccess("click", true)

	analysis := e.Analyze()

	// "click" should have live data.
	if _, ok := analysis.TimingStats["click"]; !ok {
		t.Error("expected live timing_stats for 'click'")
	}
	if _, ok := analysis.SuccessRates["click"]; !ok {
		t.Error("expected live success_rates for 'click'")
	}

	// "type" should have persisted fallback.
	if _, ok := analysis.TimingStats["type"]; !ok {
		t.Error("expected persisted timing_stats for 'type'")
	}
	if _, ok := analysis.SuccessRates["type"]; !ok {
		t.Error("expected persisted success_rates for 'type'")
	}
}

func TestAnalyzeEmptyEngine(t *testing.T) {
	e := NewAdaptiveEngine()
	analysis := e.Analyze()

	if len(analysis.TimingStats) != 0 {
		t.Errorf("expected empty timing_stats, got %d entries", len(analysis.TimingStats))
	}
	if len(analysis.SuccessRates) != 0 {
		t.Errorf("expected empty success_rates, got %d entries", len(analysis.SuccessRates))
	}
	if analysis.TotalCommands != 0 {
		t.Errorf("expected TotalCommands=0, got %d", analysis.TotalCommands)
	}
}
