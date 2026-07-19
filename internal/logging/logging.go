package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// LogEntry represents a single parsed log line.
type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"msg"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// RotatingFile handles log rotation for a single file.
type RotatingFile struct {
	mu       sync.Mutex
	f        *os.File
	path     string
	size     int64
	maxBytes int64
	keep     int
}

// NewRotatingFile creates or opens a rotating log file.
func NewRotatingFile(path string, maxMB, keep int) (*RotatingFile, error) {
	if maxMB < 1 {
		maxMB = 10
	}
	if keep < 1 {
		keep = 7
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	info, _ := f.Stat()
	var size int64
	if info != nil {
		size = info.Size()
	}
	return &RotatingFile{
		f:        f,
		path:     path,
		size:     size,
		maxBytes: int64(maxMB) * 1024 * 1024,
		keep:     keep,
	}, nil
}

func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size+int64(len(p)) > r.maxBytes {
		r.rotate()
	}

	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *RotatingFile) rotate() {
	r.f.Close()

	for i := r.keep; i >= 1; i-- {
		src := r.path
		if i > 1 {
			src = fmt.Sprintf("%s.%d", r.path, i-1)
		}
		dst := fmt.Sprintf("%s.%d", r.path, i)
		os.Remove(dst)
		os.Rename(src, dst)
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	r.f = f
	r.size = 0
}

func (r *RotatingFile) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f != nil {
		r.f.Close()
		r.f = nil
	}
}

// FileHandler is a slog.Handler that writes JSON to a RotatingFile.
type FileHandler struct {
	inner slog.Handler
	file  *RotatingFile
}

// NewFileHandler creates a new file-based slog handler.
func NewFileHandler(file *RotatingFile, level slog.Level) *FileHandler {
	return &FileHandler{
		inner: slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level}),
		file:  file,
	}
}

func (h *FileHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *FileHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.inner.Handle(ctx, r)
}

func (h *FileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &FileHandler{inner: h.inner.WithAttrs(attrs), file: h.file}
}

func (h *FileHandler) WithGroup(name string) slog.Handler {
	return &FileHandler{inner: h.inner.WithGroup(name), file: h.file}
}

func (h *FileHandler) Close() {
	h.file.Close()
}

// SafeHandler wraps a handler function with panic recovery.
func SafeHandler(name string, fn func(ctx context.Context, req any, args any) (any, any, error)) func(ctx context.Context, req any, args any) (any, any, error) {
	return func(ctx context.Context, req any, args any) (result any, payload any, err error) {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])

				slog.Error("panic in tool handler",
					"tool", name,
					"panic", fmt.Sprintf("%v", r),
					"stack", stack,
				)

				result = nil
				payload = nil
				err = fmt.Errorf("panic in %s: %v", name, r)
			}
		}()
		return fn(ctx, req, args)
	}
}

// ReadLogs reads log lines from a file, returning the last N matching lines.
func ReadLogs(path string, maxLines int, levelFilter, search string, sinceMinutes int) ([]LogEntry, int, bool) {
	if maxLines < 1 {
		maxLines = 50
	}
	if maxLines > 500 {
		maxLines = 500
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, false
	}
	defer f.Close()

	// Read lines from end for efficiency
	lines := readLinesFromEnd(f, maxLines*3) // over-read for filtering

	var entries []LogEntry
	since := time.Time{}
	if sinceMinutes > 0 {
		since = time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Filter by level
		if levelFilter != "" && !levelMatch(entry.Level, levelFilter) {
			continue
		}

		// Filter by search keyword
		if search != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(search)) {
			// Also check attrs
			found := false
			for _, v := range entry.Attrs {
				if strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), strings.ToLower(search)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by time
		if !since.IsZero() {
			t, err := time.Parse(time.RFC3339Nano, entry.Time)
			if err != nil {
				t, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", entry.Time)
			}
			if err == nil && t.Before(since) {
				continue
			}
		}

		entries = append(entries, entry)
		if len(entries) >= maxLines {
			break
		}
	}

	totalLines := countLines(path)
	truncated := len(entries) >= maxLines && totalLines > maxLines

	return entries, totalLines, truncated
}

func readLinesFromEnd(f *os.File, maxLines int) []string {
	stat, _ := f.Stat()
	if stat == nil {
		return nil
	}
	size := stat.Size()
	if size == 0 {
		return nil
	}

	// Read last 256KB max
	readSize := int64(256 * 1024)
	if readSize > size {
		readSize = size
	}

	buf := make([]byte, readSize)
	_, err := f.ReadAt(buf, size-readSize)
	if err != nil && err != io.EOF {
		return nil
	}

	lines := strings.Split(string(buf), "\n")

	// If we didn't read from the start, first line may be partial
	if readSize < size {
		// Remove first (partial) line unless it's empty
		if len(lines) > 0 && lines[0] != "" {
			lines = lines[1:]
		}
	}

	// Return last maxLines
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return lines
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count
}

func levelMatch(entryLevel, filter string) bool {
	levels := map[string]int{
		"DEBUG": -4, "INFO": 0, "WARN": 4, "ERROR": 8,
		"debug": -4, "info": 0, "warn": 4, "error": 8,
	}
	entryVal, ok1 := levels[entryLevel]
	filterVal, ok2 := levels[filter]
	if !ok1 || !ok2 {
		return strings.EqualFold(entryLevel, filter)
	}
	return entryVal >= filterVal
}

// ReportIssue generates a markdown issue body and optionally submits via gh CLI.
type ReportIssue struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	IssueURL string `json:"issue_url,omitempty"`
	LogLines int    `json:"log_lines_included"`
}

func GenerateIssue(title, userBody, logPath string, autoLogs bool) (*ReportIssue, error) {
	info := getSystemInfo()

	body := fmt.Sprintf("## System Info\n\n")
	body += fmt.Sprintf("- **OS**: %s %s\n", info.OS, info.Arch)
	body += fmt.Sprintf("- **Hostname**: %s\n", info.Hostname)
	body += fmt.Sprintf("- **RAM**: %d GB\n", info.RAMGB)
	body += fmt.Sprintf("- **Go**: %s\n", info.GoVersion)
	body += fmt.Sprintf("- **CWD**: %s\n", info.WorkDir)
	body += fmt.Sprintf("- **Time**: %s\n\n", time.Now().Format(time.RFC3339))

	if userBody != "" {
		body += fmt.Sprintf("## Description\n\n%s\n\n", userBody)
	}

	logLines := 0
	if autoLogs && logPath != "" {
		entries, _, _ := ReadLogs(logPath, 100, "warn", "", 0)
		if len(entries) == 0 {
			entries, _, _ = ReadLogs(logPath, 50, "", "", 0)
		}
		if len(entries) > 0 {
			body += "## Recent Logs\n\n```\n"
			for i := len(entries) - 1; i >= 0; i-- {
				e := entries[i]
				body += fmt.Sprintf("[%s] %s %s\n", e.Time, e.Level, e.Message)
				for k, v := range e.Attrs {
					body += fmt.Sprintf("  %s=%v\n", k, v)
				}
			}
			body += "```\n\n"
			logLines = len(entries)
		}
	}

	issue := &ReportIssue{
		Title:    title,
		Body:     body,
		LogLines: logLines,
	}

	// Try gh CLI
	if url, err := submitViaGH(title, body); err == nil && url != "" {
		issue.IssueURL = url
	}

	return issue, nil
}

func submitViaGH(title, body string) (string, error) {
	path, err := exec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("gh not found: %w", err)
	}

	cmd := exec.Command(path, "issue", "create",
		"--title", title,
		"--body", body,
		"--label", "bug",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh issue create: %w", err)
	}

	url := strings.TrimSpace(string(out))
	if !strings.HasPrefix(url, "http") {
		return "", fmt.Errorf("unexpected gh output: %s", url)
	}
	return url, nil
}

type systemInfo struct {
	OS       string
	Arch     string
	Hostname string
	RAMGB    int
	GoVersion string
	WorkDir  string
}

func getSystemInfo() systemInfo {
	hostname, _ := os.Hostname()
	workdir, _ := os.Getwd()

	return systemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Hostname:  hostname,
		RAMGB:     getRAMGB(),
		GoVersion: runtime.Version(),
		WorkDir:   workdir,
	}
}

func getRAMGB() int {
	var memStatus memoryStatus
	memStatus.cb = uint32(unsafe.Sizeof(memStatus))
	k32 := syscall.NewLazyDLL("kernel32.dll")
	proc := k32.NewProc("GlobalMemoryStatusEx")
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&memStatus)))
	if r != 0 {
		return int(memStatus.ullTotalPhys / (1024 * 1024 * 1024))
	}
	return 0
}

type memoryStatus struct {
	cb               uint32
	dwMemoryLoad     uint32
	ullTotalPhys     uint64
	ullAvailPhys     uint64
	ullTotalPageFile uint64
	ullAvailPageFile uint64
	ullTotalVirtual  uint64
	ullAvailVirtual  uint64
	ullAvailExtended uint64
}

// Init sets up file-based logging and returns the file handler for cleanup.
func Init(logDir string, maxMB, keep, levelSlog int) (*FileHandler, error) {
	logPath := filepath.Join(logDir, "server.log")
	rf, err := NewRotatingFile(logPath, maxMB, keep)
	if err != nil {
		return nil, err
	}

	fh := NewFileHandler(rf, slog.Level(levelSlog))

	// Create multi-handler: stderr + file
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(levelSlog)})
	mh := &multiHandler{handlers: []slog.Handler{stderrHandler, fh}}
	slog.SetDefault(slog.New(mh))

	return fh, nil
}

// multiHandler broadcasts to multiple handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, sh := range h.handlers {
		if sh.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, sh := range h.handlers {
		if sh.Enabled(ctx, r.Level) {
			if err := sh.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, sh := range h.handlers {
		newHandlers[i] = sh.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, sh := range h.handlers {
		newHandlers[i] = sh.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

// LogPath returns the default log file path.
func LogPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "go-mcp-computer-use", "logs", "server.log")
}

// LogDir returns the default log directory.
func LogDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	return filepath.Join(appData, "go-mcp-computer-use", "logs")
}
