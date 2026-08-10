package actions

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteChainTimeoutStopsLaterSteps(t *testing.T) {
	var markerCalls atomic.Int32
	originalDelay, hadDelay := toolDispatch["test_delay"]
	originalMarker, hadMarker := toolDispatch["test_marker"]
	t.Cleanup(func() {
		if hadDelay {
			toolDispatch["test_delay"] = originalDelay
		} else {
			delete(toolDispatch, "test_delay")
		}
		if hadMarker {
			toolDispatch["test_marker"] = originalMarker
		} else {
			delete(toolDispatch, "test_marker")
		}
	})

	toolDispatch["test_delay"] = func(map[string]any) (any, error) {
		time.Sleep(25 * time.Millisecond)
		return nil, nil
	}
	toolDispatch["test_marker"] = func(map[string]any) (any, error) {
		markerCalls.Add(1)
		return nil, nil
	}

	_, err := ExecuteChain(ChainRequest{
		TimeoutMs: 1,
		Steps: []ChainStep{
			{Tool: "test_delay"},
			{Tool: "test_marker"},
		},
	})
	if err == nil {
		t.Fatal("expected the chain to time out")
	}

	time.Sleep(50 * time.Millisecond)
	if got := markerCalls.Load(); got != 0 {
		t.Fatalf("steps after a timeout must not run; marker ran %d time(s)", got)
	}
}
