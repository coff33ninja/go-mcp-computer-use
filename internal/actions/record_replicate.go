package actions

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type RecordReplicateResult struct {
	EventsRecorded int              `json:"events_recorded"`
	StepsGenerated int              `json:"steps_generated"`
	DurationSecs   int              `json:"duration_secs"`
	LoopCount      int              `json:"loop_count"`
	Slowdown       int              `json:"slowdown"`
	Meta           map[string]any   `json:"meta"`
	ReplayResult   *ChainResult     `json:"replay_result,omitempty"`
	ReplayError    string           `json:"replay_error,omitempty"`
}

func RecordAndReplicate(durationSecs, delayMs, slowdown, loop int) (*RecordReplicateResult, error) {
	if durationSecs < 1 || durationSecs > 60 {
		return nil, fmt.Errorf("duration_secs must be 1-60, got %d", durationSecs)
	}
	if loop < 1 {
		loop = 1
	}
	if slowdown < 1 {
		slowdown = 1
	}
	if delayMs < 0 {
		delayMs = 1000
	}

	slog.Info("record_and_replicate: starting recording", "duration_secs", durationSecs)

	if err := StartKeylogger(); err != nil {
		return nil, fmt.Errorf("failed to start recording: %w", err)
	}

	time.Sleep(time.Duration(durationSecs) * time.Second)

	steps, meta, err := StopKeylogger()
	if err != nil {
		return nil, fmt.Errorf("failed to stop recording: %w", err)
	}

	result := &RecordReplicateResult{
		EventsRecorded: int(meta["total_events"].(int)),
		StepsGenerated: len(steps),
		DurationSecs:   durationSecs,
		LoopCount:      loop,
		Slowdown:       slowdown,
		Meta:           meta,
	}

	if len(steps) == 0 {
		slog.Info("record_and_replicate: no events recorded, skipping replay")
		return result, nil
	}

	chainSteps, err := mapsToChainSteps(steps)
	if err != nil {
		return nil, fmt.Errorf("failed to convert steps: %w", err)
	}

	if slowdown > 1 {
		chainSteps = applySlowdownChain(chainSteps, slowdown)
	}

	if loop > 1 {
		chainSteps = duplicateChainSteps(chainSteps, loop)
	}

	if delayMs > 0 {
		delayStep := ChainStep{
			Tool: "wait",
			Args: map[string]any{"ms": delayMs},
		}
		chainSteps = append([]ChainStep{delayStep}, chainSteps...)
	}

	slog.Info("record_and_replicate: executing replay", "steps", len(chainSteps), "loop", loop)

	chainReq := ChainRequest{
		Steps:     chainSteps,
		TimeoutMs: (durationSecs * loop * slowdown * 1000) + delayMs + 10000,
	}

	chainResult, err := ExecuteChain(chainReq)
	if err != nil {
		result.ReplayError = err.Error()
		return result, fmt.Errorf("replay failed: %w", err)
	}

	result.ReplayResult = chainResult
	return result, nil
}

func mapsToChainSteps(maps []map[string]any) ([]ChainStep, error) {
	var steps []ChainStep
	for _, m := range maps {
		b, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		var step ChainStep
		if err := json.Unmarshal(b, &step); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func applySlowdownChain(steps []ChainStep, factor int) []ChainStep {
	result := make([]ChainStep, len(steps))
	for i, step := range steps {
		cp := step
		if cp.Args != nil {
			cp.Args = make(map[string]any, len(step.Args))
			for k, v := range step.Args {
				cp.Args[k] = v
			}
		}
		if cp.Tool == "wait" && cp.Args != nil {
			if ms, ok := cp.Args["ms"].(int); ok {
				cp.Args["ms"] = ms * factor
			} else if ms, ok := cp.Args["ms"].(float64); ok {
				cp.Args["ms"] = int(ms) * factor
			}
		}
		result[i] = cp
	}
	return result
}

func duplicateChainSteps(steps []ChainStep, count int) []ChainStep {
	total := len(steps) * count
	result := make([]ChainStep, 0, total)
	for i := 0; i < count; i++ {
		result = append(result, steps...)
	}
	return result
}
