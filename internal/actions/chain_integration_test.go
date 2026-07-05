//go:build integration

package actions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "mcp-chain-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "mcp-server.test.exe")
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/mcp-server/")
	buildCmd.Dir = projectRoot
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type mcpClient struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
}

func startMCPServer(t *testing.T) *mcpClient {
	t.Helper()
	cmd := exec.Command(binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	c := &mcpClient{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewScanner(stdout),
	}
	c.stdout.Buffer(make([]byte, 65536), 65536*4)
	return c
}

func (c *mcpClient) send(t *testing.T, req any) string {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := c.stdin.Write([]byte("\n")); err != nil {
		t.Fatalf("write newline: %v", err)
	}
	c.stdin.Flush()
	return c.readResp(t)
}

func (c *mcpClient) readResp(t *testing.T) string {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for response")
		default:
		}
		if c.stdout.Scan() {
			return c.stdout.Text()
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (c *mcpClient) initialize(t *testing.T) {
	t.Helper()
	resp := c.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "chain-test", "version": "1.0"},
		},
	})
	if !strings.Contains(resp, `"id":1`) {
		t.Fatalf("initialize failed: %s", resp)
	}
}

func (c *mcpClient) callTool(t *testing.T, name string, args map[string]any) map[string]any {
	t.Helper()
	resp := c.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": args,
		},
	})
	var parsed struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StructuredContent json.RawMessage `json:"structuredContent"`
		} `json:"result"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, resp)
	}
	if parsed.Error.Message != "" {
		t.Fatalf("RPC error: %s", parsed.Error.Message)
	}
	var result map[string]any
	if len(parsed.Result.StructuredContent) > 0 {
		json.Unmarshal(parsed.Result.StructuredContent, &result)
	}
	if result == nil && len(parsed.Result.Content) > 0 {
		json.Unmarshal([]byte(parsed.Result.Content[0].Text), &result)
	}
	return result
}

func TestChain_SimpleSteps(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{"tool": "get_cursor_position"},
			{"tool": "get_screen_size"},
			{"tool": "get_system_info"},
		},
	})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("chain not successful: %v", result)
	}
	stepCount, _ := result["step_count"].(float64)
	if stepCount != 3 {
		t.Fatalf("expected 3 steps, got %v", stepCount)
	}
}

func TestChain_Capture(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{"tool": "get_cursor_position", "capture": "pos"},
		},
	})
	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("chain not successful: %v", result)
	}
	vars, ok := result["variables"].(map[string]any)
	if !ok {
		t.Fatalf("expected variables in result")
	}
	pos, ok := vars["pos"].(map[string]any)
	if !ok {
		t.Fatalf("expected pos variable, got %v", vars)
	}
	if pos["x"] == nil || pos["y"] == nil {
		t.Fatalf("expected x/y in pos, got %v", pos)
	}
}

func TestChain_Loop(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{
				"loop": map[string]any{"times": float64(3)},
				"steps": []map[string]any{
					{"tool": "get_cursor_position"},
				},
			},
		},
	})
	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("chain not successful: %v", result)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) < 1 {
		t.Fatalf("expected results array")
	}
	loopResult, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected loop step result")
	}
	if loopResult["tool"] != "loop" {
		t.Fatalf("expected tool 'loop', got %v", loopResult["tool"])
	}
	iterations, _ := loopResult["output"].(map[string]any)["iterations"].(float64)
	if iterations != 3 {
		t.Fatalf("expected 3 iterations, got %v", iterations)
	}
}

func TestChain_IfElse(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{
				"if": map[string]any{"ocr_contains": "NONEXISTENT_TEXT_XYZZY_DEADBEEF"},
				"then": []map[string]any{
					{"tool": "get_screen_size"},
				},
				"else": []map[string]any{
					{"tool": "get_system_info"},
				},
			},
		},
	})
	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("chain not successful: %v", result)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) < 1 {
		t.Fatalf("expected results array")
	}
	ifResult, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected if step result")
	}
	if ifResult["tool"] != "if" {
		t.Fatalf("expected tool 'if', got %v", ifResult["tool"])
	}
	branch, _ := ifResult["output"].(map[string]any)["branch"].(string)
	if branch != "else" {
		t.Fatalf("expected branch 'else' for non-matching text, got %q", branch)
	}
}

func TestChain_UnknownTool(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{"tool": "this_tool_does_not_exist"},
		},
	})
	success, _ := result["success"].(bool)
	if success {
		t.Fatalf("expected chain to fail on unknown tool")
	}
}

func TestChain_Timeout(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"timeout_ms": float64(1),
		"steps": []map[string]any{
			{"tool": "wait", "args": map[string]any{"ms": float64(5000)}},
		},
	})
	success, _ := result["success"].(bool)
	if success {
		t.Fatalf("expected chain to fail with timeout")
	}
}

func TestChain_StructuredData(t *testing.T) {
	c := startMCPServer(t)
	c.initialize(t)
	result := c.callTool(t, "chain", map[string]any{
		"steps": []map[string]any{
			{"tool": "get_cursor_position"},
		},
	})
	pos, ok := result["results"].([]any)[0].(map[string]any)["output"].(map[string]any)
	if !ok {
		t.Fatalf("expected structured output with x/y, got %v", result)
	}
	if pos["x"] == nil {
		t.Fatalf("expected x field in output")
	}
}
