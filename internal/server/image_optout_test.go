package server

import (
	"testing"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func boolPtr(b bool) *bool { return &b }

func TestStripImageIfExcluded(t *testing.T) {
	orig := actions.ActiveConfig
	defer func() { actions.ActiveConfig = orig }()

	cases := []struct {
		name         string
		global       bool
		perCall      *bool
		wantStripped bool
	}{
		{"global-true-no-override", true, nil, false},
		{"global-false-no-override", false, nil, true},
		{"global-false-call-true", false, boolPtr(true), false},
		{"global-true-call-false", true, boolPtr(false), true},
		{"global-true-call-true", true, boolPtr(true), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actions.ActiveConfig = config.Default()
			actions.ActiveConfig.IncludeImage = tc.global

			ann := &actions.AnnotatedCapture{ImageB64: "AAAA"}
			result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "AAAA"}}}
			args := OCRArgs{IncludeImage: tc.perCall}

			stripImageIfExcluded(args, result, ann)

			if tc.wantStripped {
				if ann.ImageB64 != "" {
					t.Fatalf("expected ImageB64 stripped, got %q", ann.ImageB64)
				}
				if len(result.Content) != 0 {
					t.Fatalf("expected primary image content removed, got %d items", len(result.Content))
				}
			} else {
				if ann.ImageB64 != "AAAA" {
					t.Fatalf("expected ImageB64 preserved, got %q", ann.ImageB64)
				}
				if len(result.Content) != 1 {
					t.Fatalf("expected image content preserved, got %d items", len(result.Content))
				}
			}
		})
	}
}

func TestStripImageNonCapturePayloadUntouched(t *testing.T) {
	orig := actions.ActiveConfig
	defer func() { actions.ActiveConfig = orig }()
	actions.ActiveConfig = config.Default()
	actions.ActiveConfig.IncludeImage = false

	payload := map[string]any{"ok": true}
	result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "hello"}}}
	stripImageIfExcluded(ClickArgs{}, result, payload)

	if len(result.Content) != 1 {
		t.Fatalf("non-capture payload content should be untouched, got %d items", len(result.Content))
	}
}

func TestStripElementsIfExcluded(t *testing.T) {
	orig := actions.ActiveConfig
	defer func() { actions.ActiveConfig = orig }()

	cases := []struct {
		name         string
		global       bool
		perCall      *bool
		wantExcluded bool
	}{
		{"default-off", false, nil, false},
		{"global-on", true, nil, true},
		{"global-on-call-off", true, boolPtr(false), false},
		{"global-off-call-on", false, boolPtr(true), true},
		{"call-on", false, boolPtr(true), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actions.ActiveConfig = config.Default()
			actions.ActiveConfig.ExcludeElements = tc.global

			ann := &actions.AnnotatedCapture{
				ImageB64: "AAAA",
				Elements: []actions.AnnotatedElement{{Class: "icon"}},
			}
			result := &mcp.CallToolResult{}
			args := OCRArgs{ExcludeElements: tc.perCall}

			stripImageIfExcluded(args, result, ann)

			if tc.wantExcluded {
				if len(ann.Elements) != 0 {
					t.Fatalf("expected elements excluded, got %d", len(ann.Elements))
				}
			} else {
				if len(ann.Elements) != 1 {
					t.Fatalf("expected elements preserved, got %d", len(ann.Elements))
				}
			}
		})
	}
}
