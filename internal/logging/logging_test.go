package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	rf, err := NewRotatingFile(path, 1, 3)
	if err != nil {
		t.Fatalf("NewRotatingFile: %v", err)
	}
	defer rf.Close()

	n, err := rf.Write([]byte("hello world\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 12 {
		t.Errorf("expected 12 bytes written, got %d", n)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello world\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestRotatingFileRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	rf, err := NewRotatingFile(path, 1, 3)
	if err != nil {
		t.Fatalf("NewRotatingFile: %v", err)
	}
	defer rf.Close()

	// Write enough to trigger rotation (1MB = 1024*1024 bytes)
	line := strings.Repeat("x", 100) + "\n"
	for i := 0; i < 12000; i++ {
		rf.Write([]byte(line))
	}

	// Should have rotated at least once
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("main log file should exist")
	}

	// Check that rotated file exists
	rotated := path + ".1"
	if _, err := os.Stat(rotated); os.IsNotExist(err) {
		t.Error("rotated log file should exist")
	}
}

func TestReadLogsBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Write some log entries
	f, _ := os.Create(path)
	for i := 0; i < 10; i++ {
		entry := map[string]any{
			"time":    "2025-01-15T10:00:0" + fmt.Sprintf("%d", i) + "Z",
			"level":   "INFO",
			"msg":     fmt.Sprintf("test message %d", i),
			"tool":    "click",
		}
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
	f.Close()

	entries, total, truncated := ReadLogs(path, 5, "", "", 0)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
	if total != 10 {
		t.Errorf("expected 10 total lines, got %d", total)
	}
	if truncated != true {
		t.Error("expected truncated=true (10 lines in file, requested 5)")
	}
}

func TestReadLogsLevelFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	f, _ := os.Create(path)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "INFO", "WARN"}
	for i, level := range levels {
		entry := map[string]any{
			"time":  fmt.Sprintf("2025-01-15T10:00:%02dZ", i),
			"level": level,
			"msg":   fmt.Sprintf("message %d", i),
		}
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
	f.Close()

	// Filter: WARN and above
	entries, _, _ := ReadLogs(path, 50, "warn", "", 0)
	if len(entries) != 3 {
		t.Errorf("expected 3 warn+ entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Level != "WARN" && e.Level != "ERROR" {
			t.Errorf("unexpected level in results: %s", e.Level)
		}
	}
}

func TestReadLogsSearchFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	f, _ := os.Create(path)
	messages := []string{"click failed", "scroll ok", "click retry", "type done"}
	for i, msg := range messages {
		entry := map[string]any{
			"time":  fmt.Sprintf("2025-01-15T10:00:%02dZ", i),
			"level": "INFO",
			"msg":   msg,
		}
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
	f.Close()

	entries, _, _ := ReadLogs(path, 50, "", "click", 0)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries matching 'click', got %d", len(entries))
	}
}

func TestReadLogsSinceMinutes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	f, _ := os.Create(path)
	// Write one old entry and one recent entry
	oldEntry := map[string]any{
		"time":  "2020-01-01T00:00:00Z",
		"level": "INFO",
		"msg":   "old message",
	}
	data, _ := json.Marshal(oldEntry)
	f.Write(append(data, '\n'))

	recentEntry := map[string]any{
		"time":  "2099-01-01T00:00:00Z",
		"level": "INFO",
		"msg":   "future message",
	}
	data, _ = json.Marshal(recentEntry)
	f.Write(append(data, '\n'))
	f.Close()

	entries, _, _ := ReadLogs(path, 50, "", "", 5)
	if len(entries) != 1 {
		t.Errorf("expected 1 recent entry, got %d", len(entries))
	}
	if len(entries) == 1 && entries[0].Message != "future message" {
		t.Errorf("expected 'future message', got %q", entries[0].Message)
	}
}

func TestReadLogsFileNotFound(t *testing.T) {
	entries, total, truncated := ReadLogs("/nonexistent/path.log", 10, "", "", 0)
	if entries != nil {
		t.Error("expected nil entries for nonexistent file")
	}
	if total != 0 {
		t.Error("expected 0 total for nonexistent file")
	}
	if truncated {
		t.Error("expected truncated=false for nonexistent file")
	}
}

func TestLevelMatch(t *testing.T) {
	tests := []struct {
		entry  string
		filter string
		want   bool
	}{
		{"INFO", "info", true},
		{"INFO", "debug", true},
		{"INFO", "warn", false},
		{"ERROR", "warn", true},
		{"DEBUG", "info", false},
		{"WARN", "error", false},
	}
	for _, tt := range tests {
		got := levelMatch(tt.entry, tt.filter)
		if got != tt.want {
			t.Errorf("levelMatch(%q, %q) = %v, want %v", tt.entry, tt.filter, got, tt.want)
		}
	}
}

func TestSafeHandlerSuccess(t *testing.T) {
	called := false
	fn := SafeHandler("test_tool", func(ctx context.Context, req any, args any) (any, any, error) {
		called = true
		return nil, map[string]any{"ok": true}, nil
	})

	result, payload, err := fn(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if payload == nil {
		t.Error("expected payload")
	}
	_ = result
}

func TestSafeHandlerPanic(t *testing.T) {
	fn := SafeHandler("panic_tool", func(ctx context.Context, req any, args any) (any, any, error) {
		panic("something went wrong")
	})

	result, payload, err := fn(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error from panic")
	}
	if result != nil {
		t.Error("expected nil result from panic")
	}
	if payload != nil {
		t.Error("expected nil payload from panic")
	}
	if !strings.Contains(err.Error(), "panic in panic_tool") {
		t.Errorf("error should mention tool name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should mention panic message, got: %v", err)
	}
}

func TestReadLogsMaxLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	f, _ := os.Create(path)
	for i := 0; i < 100; i++ {
		entry := map[string]any{
			"time":  fmt.Sprintf("2025-01-15T10:%02d:00Z", i%60),
			"level": "INFO",
			"msg":   fmt.Sprintf("message %d", i),
		}
		data, _ := json.Marshal(entry)
		f.Write(append(data, '\n'))
	}
	f.Close()

	// Request 500 (max), should get 100
	entries, _, _ := ReadLogs(path, 500, "", "", 0)
	if len(entries) != 100 {
		t.Errorf("expected 100 entries, got %d", len(entries))
	}

	// Request 10
	entries, _, _ = ReadLogs(path, 10, "", "", 0)
	if len(entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(entries))
	}
}

func TestGenerateIssueBody(t *testing.T) {
	issue, err := GenerateIssue("test error", "something broke", "", false)
	if err != nil {
		t.Fatalf("GenerateIssue: %v", err)
	}
	if issue.Title != "test error" {
		t.Errorf("expected title 'test error', got %q", issue.Title)
	}
	if !strings.Contains(issue.Body, "something broke") {
		t.Error("body should contain user text")
	}
	if !strings.Contains(issue.Body, "System Info") {
		t.Error("body should contain system info")
	}
}
