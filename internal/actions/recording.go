package actions

import (
	"fmt"
	"time"
)

type RecordedFrame struct {
	Index    int    `json:"index"`
	ImageB64 string `json:"image_b64"`
	TimestampMs int64 `json:"timestamp_ms"`
}

type RecordingResult struct {
	Frames    []RecordedFrame `json:"frames"`
	TotalMs   int64           `json:"total_ms"`
	FPS       float64         `json:"fps"`
}

func RecordScreen(durationMs, intervalMs int32) (*RecordingResult, error) {
	if durationMs <= 0 { durationMs = 5000 }
	if durationMs > 60000 { durationMs = 60000 }
	if intervalMs <= 0 { intervalMs = 500 }
	if intervalMs < 100 { intervalMs = 100 }

	start := time.Now()
	deadline := start.Add(time.Duration(durationMs) * time.Millisecond)
	var frames []RecordedFrame
	index := 0

	for time.Now().Before(deadline) {
		frameStart := time.Now()
		b64, err := CaptureScreen()
		if err != nil {
			return nil, fmt.Errorf("record frame %d: %w", index, err)
		}
		frames = append(frames, RecordedFrame{
			Index:       index,
			ImageB64:    b64,
			TimestampMs: time.Since(start).Milliseconds(),
		})
		index++

		elapsed := time.Since(frameStart)
		sleepFor := time.Duration(intervalMs)*time.Millisecond - elapsed
		if sleepFor > 0 {
			Wait(int32(sleepFor.Milliseconds()))
		}
	}

	totalMs := time.Since(start).Milliseconds()
	fps := float64(len(frames)) / (float64(totalMs) / 1000.0)

	return &RecordingResult{
		Frames:  frames,
		TotalMs: totalMs,
		FPS:     fps,
	}, nil
}
