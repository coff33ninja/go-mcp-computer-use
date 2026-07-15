package main

// credit-audit measures the JSON payload size each MCP tool would hand back
// to the calling AI, estimates the token cost, and ranks tools by how much
// context/credit they burn per call. Complements cmd/benchmark (which times
// latency, not payload size). Only calls read-only tools — nothing here
// clicks, types, or mutates state on the machine.
//
// Usage:
//   go run ./cmd/credit-audit
//   go run ./cmd/credit-audit -sessions 20   // project N-call session cost
//   go run ./cmd/credit-audit -json         // machine-readable JSON output
//
// Token estimate is bytes/4 (rough BPE approximation for JSON/base64 text).
// Edit pricePerMillionTokens below to match your actual model/plan rate —
// this file intentionally does not hardcode a "real" number since pricing
// changes and varies by model tier.

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
)

const pricePerMillionTokens = 3.00

type probe struct {
	Name     string
	Category string
	Fn       func() (any, error)
}

type probeResult struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Group      string `json:"group"`
	Bytes      int    `json:"bytes"`
	EstTokens  int    `json:"est_tokens"`
	Ms         int64  `json:"ms"`
	Err        string `json:"err,omitempty"`
}

func estTokens(b int) int {
	return (b + 3) / 4
}

func main() {
	sessions := flag.Int("sessions", 1, "simulate N calls per hog tool to project session cost")
	jsonOut := flag.Bool("json", false, "output JSON instead of a formatted table")
	flag.Parse()

	actions.ActiveConfig = config.Default()
	actions.SetDPIAware()

	if err := actions.CheckScreenshotPermission(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}

	actions.InitMemoryStore()
	actions.InitTrainingStore()
	actions.InitDataLog()

	activeWin, err := actions.GetActiveWindowInfo()
	var hwnd uintptr
	if err == nil && activeWin != nil {
		hwnd = activeWin.Handle
	}

	probes := []probe{
		// ── System Info (16 tools) ──
		{"get_system_info", "cheap", func() (any, error) { return actions.GetSystemInfo() }},
		{"get_screen_size", "cheap", func() (any, error) { w, h := actions.ScreenSize(); return map[string]int32{"width": w, "height": h}, nil }},
		{"get_uptime", "cheap", func() (any, error) { d, e := actions.GetUptime(); return map[string]any{"uptime_ms": e, "uptime": d.String()}, nil }},
		{"get_idle_time", "cheap", func() (any, error) { d, e := actions.GetIdleTime(); return map[string]any{"idle_ms": e, "idle": d.String()}, nil }},
		{"get_battery", "cheap", func() (any, error) { return actions.GetBattery() }},
		{"get_volume", "cheap", func() (any, error) { return actions.GetVolume() }},
		{"get_brightness", "cheap", func() (any, error) { return actions.GetBrightness() }},
		{"get_disk_usage", "cheap", func() (any, error) { return actions.GetDiskUsage() }},
		{"get_network_info", "cheap", func() (any, error) { return actions.GetNetworkInfo() }},
		{"get_keyboard_layout", "cheap", func() (any, error) { return actions.GetKeyboardLayout() }},
		{"get_clipboard", "cheap", func() (any, error) { return actions.GetClipboardText() }},
		{"get_cursor_position", "cheap", func() (any, error) { x, y, e := actions.GetCursorPosition(); return map[string]int32{"x": x, "y": y}, e }},
		{"get_pixel_color", "cheap", func() (any, error) { return actions.GetPixelColor(100, 100) }},
		{"get_working_directory", "cheap", func() (any, error) { return actions.GetWorkingDirectory(), nil }},
		{"get_file_info", "cheap", func() (any, error) { return actions.GetFileInfo(".") }},
		{"ping", "cheap", func() (any, error) { return actions.PingHost("127.0.0.1") }},

		// ── Display / DPI (6 tools) ──
		{"list_displays", "cheap", func() (any, error) { return actions.ListDisplays() }},
		{"get_display_modes", "cheap", func() (any, error) { disps, _ := actions.ListDisplays(); if len(disps) > 0 { return actions.GetDisplayModes(disps[0].Name) }; return nil, nil }},
		{"get_screen_dpi", "cheap", func() (any, error) { return actions.GetScreenDPI() }},
		{"list_monitor_dpis", "cheap", func() (any, error) { return actions.ListMonitorDPIs() }},
		{"get_dpi_for_point", "cheap", func() (any, error) { return actions.GetDPIScaleForPoint(0, 0) }},
		{"get_dpi_for_window", "cheap", func() (any, error) { return actions.GetDPIScaleForWindow(hwnd), nil }},

		// ── Audio (1 tool) ──
		{"list_audio_devices", "cheap", func() (any, error) { return actions.ListAudioDevices() }},

		// ── Window / Process (7 tools) ──
		{"list_windows", "cheap", func() (any, error) { return actions.ListWindows() }},
		{"list_processes", "cheap", func() (any, error) { return actions.ListProcesses() }},
		{"get_active_window", "cheap", func() (any, error) { return actions.GetActiveWindowInfo() }},
		{"find_window_by_title", "cheap", func() (any, error) { h := actions.FindWindowByTitle(""); return map[string]uintptr{"handle": h}, nil }},
		{"get_window_state", "cheap", func() (any, error) { return actions.GetWindowState(hwnd) }},
		{"get_window_rect", "cheap", func() (any, error) { return actions.GetWindowRectByHandle(hwnd) }},
		{"get_cached_detections", "cheap", func() (any, error) { return actions.GetCachedDetections(), nil }},

		// ── File System (3 tools) ──
		{"list_directory", "cheap", func() (any, error) { return actions.ListDirectory(".") }},
		{"find_files", "cheap", func() (any, error) { return actions.FindFiles(".", "*.go") }},
		{"read_file", "cheap", func() (any, error) { return actions.ReadFile("VERSION", 0, 0) }},

		// ── Perception — the expensive ones ──
		{"screenshot_full", "hog", func() (any, error) { return actions.CaptureScreen() }},
		{"screenshot_region_400x400", "hog", func() (any, error) { return actions.CaptureRegion(0, 0, 400, 400) }},
		{"screenshot_element", "hog", func() (any, error) { return actions.ScreenshotElement(hwnd) }},
		{"ocr_full_screen", "hog", func() (any, error) { return actions.OCRScreen("en-US") }},
		{"ocr_region_400x400", "hog", func() (any, error) { return actions.OCRRegion(0, 0, 400, 400, "en-US") }},
		{"ocr_window", "hog", func() (any, error) { return actions.OCRWindow(hwnd, "en-US") }},
		{"ocr_languages", "cheap", func() (any, error) { return actions.OcrLanguages() }},


		// ── ONNX / Watcher (5 tools) ──
		{"onnx_status", "cheap", func() (any, error) { return actions.ONNXStatus(), nil }},
		{"watcher_status", "cheap", func() (any, error) { return actions.GetWatcherStatus(), nil }},
		{"onnx_detect_empty", "hog", func() (any, error) { return actions.ONNXDetect(actions.DetectionInput{}) }},
		{"record_screen_1s", "hog", func() (any, error) { return actions.RecordScreen(1000, 500) }},
		{"keylogger_status", "cheap", func() (any, error) { active, count, dur := actions.KeyloggerStatus(); return map[string]any{"active": active, "event_count": count, "duration": dur}, nil }},
		{"bridge_debug", "cheap", func() (any, error) { return actions.BridgeDebugInfo(), nil }},

		// ── UIA (5 tools) ──
		{"uia_find_by_name", "cheap", func() (any, error) { return actions.UIAFindElement(actions.UIAFindOpts{Name: "Taskbar"}) }},
		{"uia_find_by_type", "cheap", func() (any, error) { return actions.UIAFindElement(actions.UIAFindOpts{ControlType: "Button"}) }},
		{"uia_get_element_at_point", "cheap", func() (any, error) { return actions.UIAFindElement(actions.UIAFindOpts{Name: "Start"}) }},
		{"uia_get_all_elements", "hog", func() (any, error) { return actions.UIAGetAllElements(hwnd, 500) }},
		{"find_ui_element", "cheap", func() (any, error) { return actions.FindUIElement(actions.FindUIElementInput{}) }},

		// ── Memory / Template (6 tools) ──
		{"memory_list", "cheap", func() (any, error) { return actions.MemoryList(actions.MemoryListInput{}) }},
		{"memory_search", "cheap", func() (any, error) { return actions.MemorySearch(actions.MemorySearchInput{Query: "test"}) }},
		{"template_list", "cheap", func() (any, error) { return actions.TemplateList(actions.TemplateListInput{}) }},
		{"template_find", "cheap", func() (any, error) { return actions.TemplateFind(actions.TemplateFindInput{ElementKey: "nonexistent"}) }},
		{"priors_stats", "cheap", func() (any, error) { return actions.GetPriorStats(0) }},

		// ── Training (2 tools) ──
		{"training_stats", "cheap", func() (any, error) { return actions.TrainingStatsReport() }},
		{"training_list_samples_limited", "cheap", func() (any, error) { return actions.TrainingSampleList(actions.TrainingListInput{Limit: 5}) }},

		// ── Datalog / Introspection (4 tools) ──
		{"datalog_status", "cheap", func() (any, error) { return actions.DataLogStatsReport() }},
		{"datalog_query", "cheap", func() (any, error) { return actions.QueryDataLog(actions.DataLogQuery{Table: "commands", Limit: 10}) }},
		{"introspection_analyze", "cheap", func() (any, error) { return actions.IntrospectionAnalyze() }},
		{"export_training_data", "cheap", func() (any, error) { return actions.ExportTrainingData("", 10) }},

		// ── Adaptive / Agent (3 tools) ──
		{"agent_analyze", "cheap", func() (any, error) { actions.EnsureAdaptive(); return actions.Adaptive.Analyze(), nil }},
		{"agent_suggest", "cheap", func() (any, error) { actions.EnsureAdaptive(); return actions.Adaptive.PredictActions("current screen text", 5), nil }},
		{"agent_train", "cheap", func() (any, error) { actions.EnsureAdaptive(); return nil, actions.Adaptive.TrainFromDatalog() }},

		// ── Layout validation (1 tool) ──
		{"layout_validate", "cheap", func() (any, error) {
			return actions.ValidateLayout(actions.LayoutValidateInput{
				Elements: []actions.LayoutElement{
					{ID: "test", StoredCoord: actions.Coord{X: 100, Y: 100}},
				},
				DriftTolerance: 50,
			})
		}},

	}

	var results []probeResult
	probeTimeout := 60 * time.Second
	for _, p := range probes {
		start := time.Now()
		type probeOut struct {
			val any
			err error
		}
		ch := make(chan probeOut, 1)
		go func(fn func() (any, error)) {
			v, e := fn()
			ch <- probeOut{v, e}
		}(p.Fn)
		var out any
		var err error
		select {
		case res := <-ch:
			out, err = res.val, res.err
		case <-time.After(probeTimeout):
			err = fmt.Errorf("TIMEOUT after %v", probeTimeout)
		}
		elapsed := time.Since(start).Milliseconds()
		r := probeResult{Name: p.Name, Category: p.Category, Ms: elapsed}
		if err != nil {
			r.Err = err.Error()
			results = append(results, r)
			continue
		}
		b, merr := json.Marshal(out)
		if merr != nil {
			r.Err = merr.Error()
			results = append(results, r)
			continue
		}
		r.Bytes = len(b)
		r.EstTokens = estTokens(len(b))
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].EstTokens > results[j].EstTokens })

	// Assign groups for reporting
	groupOf := map[string]string{
		"get_system_info": "System", "get_screen_size": "System",
		"get_uptime": "System", "get_idle_time": "System",
		"get_battery": "System", "get_volume": "System",
		"get_brightness": "System", "get_disk_usage": "System",
		"get_network_info": "System", "get_keyboard_layout": "System",
		"get_clipboard": "System", "get_cursor_position": "System",
		"get_pixel_color": "System", "get_working_directory": "System",
		"get_file_info": "System", "ping": "System",
		"onnx_detect_empty": "ONNX", "record_screen_1s": "System",
		"keylogger_status": "System",

		"list_displays": "Display", "get_display_modes": "Display",
		"get_screen_dpi": "Display", "list_monitor_dpis": "Display",
		"get_dpi_for_point": "Display", "get_dpi_for_window": "Display",
		"list_audio_devices": "Audio",

		"list_windows": "Window", "list_processes": "Window",
		"get_active_window": "Window", "find_window_by_title": "Window",
		"get_window_state": "Window", "get_window_rect": "Window",
		"get_cached_detections": "Window",

		"list_directory": "File", "find_files": "File", "read_file": "File",

		"screenshot_full": "Perception", "screenshot_region_400x400": "Perception",
		"screenshot_element": "Perception",
		"ocr_full_screen": "Perception", "ocr_region_400x400": "Perception",
		"ocr_window": "Perception", "ocr_languages": "Perception",
		"find_image_full_screen": "Perception", "find_all_images_full_screen": "Perception",
		"detect_on_screen": "Perception",

		"onnx_status": "ONNX", "watcher_status": "ONNX", "bridge_debug": "ONNX",

		"uia_find_by_name": "UIA", "uia_find_by_type": "UIA",
		"uia_get_element_at_point": "UIA",
		"uia_get_all_elements": "UIA", "find_ui_element": "UIA",

		"memory_list": "Memory", "memory_search": "Memory",
		"template_list": "Template", "template_find": "Template",
		"priors_stats": "Template",

		"training_stats": "Training", "training_list_samples_limited": "Training",

		"datalog_status": "Datalog", "datalog_query": "Datalog",
		"introspection_analyze": "Datalog", "export_training_data": "Datalog",

		"agent_analyze": "Agent", "agent_suggest": "Agent", "agent_train": "Agent",

		"layout_validate": "Layout",
	}
	for i := range results {
		if g, ok := groupOf[results[i].Name]; ok {
			results[i].Group = g
		} else {
			results[i].Group = "Other"
		}
	}

	if *jsonOut {
		summary := buildJSONSummary(results, *sessions)
		b, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(b))
		return
	}

	printTable(results, *sessions)
}

func printTable(results []probeResult, sessions int) {
	fmt.Println("=== MCP TOOL CREDIT AUDIT (single call per tool) ===")
	fmt.Printf("%-40s %-6s %10s %12s %8s\n", "TOOL", "CAT", "BYTES", "EST. TOKENS", "TIME")
	fmt.Println("--------------------------------------------------------------------------------")
	var totalBytes, totalTokens int
	var hogBytes, hogTokens int
	var failCount int
	for _, r := range results {
		if r.Err != "" {
			fmt.Printf("%-40s %-6s %10s %12s %s  (error: %s)\n", r.Name, r.Category, "-", "-", "-", r.Err)
			failCount++
			continue
		}
		var timeStr string
		if r.Ms >= 1000 {
			timeStr = fmt.Sprintf("%5.1fs", float64(r.Ms)/1000)
		} else {
			timeStr = fmt.Sprintf("%5dms", r.Ms)
		}
		fmt.Printf("%-40s %-6s %10d %12d %s\n", r.Name, r.Category, r.Bytes, r.EstTokens, timeStr)
		totalBytes += r.Bytes
		totalTokens += r.EstTokens
		if r.Category == "hog" {
			hogBytes += r.Bytes
			hogTokens += r.EstTokens
		}
	}

	fmt.Println()
	fmt.Printf("Tools called: %d (failed: %d)\n", len(results), failCount)
	fmt.Printf("Total payload: %s (%d bytes)\n", humanBytes(totalBytes), totalBytes)
	fmt.Printf("Total est. tokens: %d (~$%.4f at $%.2f/M)\n",
		totalTokens, float64(totalTokens)*pricePerMillionTokens/1e6, pricePerMillionTokens)
	fmt.Printf("Hog tools alone:   %s (%d bytes, %d tokens, %.1f%% of total)\n",
		humanBytes(hogBytes), hogBytes, hogTokens, 100*float64(hogTokens)/float64(totalTokens))

	// ── Top 10 hogs ──
	fmt.Println()
	fmt.Println("── Top 10 Credit Hogs ──")
	var shown int
	for _, r := range results {
		if r.Err != "" {
			continue
		}
		fmt.Printf("%3d. %-37s %s %7d tok (%s)\n", shown+1, r.Name, fmt.Sprintf("%8s", humanBytes(r.Bytes)), r.EstTokens, r.Group)
		shown++
		if shown >= 10 {
			break
		}
	}

	// ── By group ──
	fmt.Println()
	fmt.Println("── By Group ──")
	groups := make(map[string][]probeResult)
	for _, r := range results {
		if r.Err != "" {
			continue
		}
		groups[r.Group] = append(groups[r.Group], r)
	}
	var gnames []string
	for g := range groups {
		gnames = append(gnames, g)
	}
	sort.Strings(gnames)
	for _, g := range gnames {
		rs := groups[g]
		var gBytes, gTok int
		for _, r := range rs {
			gBytes += r.Bytes
			gTok += r.EstTokens
		}
		fmt.Printf("  %-15s %3d tools  %10s ~%8dtok\n", g, len(rs), humanBytes(gBytes), gTok)
	}

	if sessions > 1 {
		fmt.Println()
		fmt.Printf("── PROJECTED SESSION: %d calls to EACH hog tool ──\n", sessions)
		var projected int
		for _, r := range results {
			if r.Category != "hog" || r.Err != "" {
				continue
			}
			projected += r.EstTokens * sessions
		}
		fmt.Printf("Projected hog tokens: %d (~$%.4f)\n",
			projected, float64(projected)*pricePerMillionTokens/1e6)
		fmt.Println("Note: assumes every hog call returns a similarly-sized payload;")
		fmt.Println("actual OCR/screenshot size varies heavily with screen content.")
	}
}

func buildJSONSummary(results []probeResult, sessions int) map[string]any {
	var totalBytes, totalTokens, failCount int
	for _, r := range results {
		totalBytes += r.Bytes
		totalTokens += r.EstTokens
		if r.Err != "" {
			failCount++
		}
	}

	top10 := make([]probeResult, 0, 10)
	for _, r := range results {
		if r.Err != "" {
			continue
		}
		top10 = append(top10, r)
		if len(top10) >= 10 {
			break
		}
	}

	groups := make(map[string]map[string]any)
	for _, r := range results {
		if r.Err != "" {
			continue
		}
		if _, ok := groups[r.Group]; !ok {
			groups[r.Group] = map[string]any{"tools": 0, "bytes": 0, "est_tokens": 0}
		}
		g := groups[r.Group]
		g["tools"] = g["tools"].(int) + 1
		g["bytes"] = g["bytes"].(int) + r.Bytes
		g["est_tokens"] = g["est_tokens"].(int) + r.EstTokens
	}

	return map[string]any{
		"total_tools":       len(results),
		"failed":            failCount,
		"total_bytes":       totalBytes,
		"total_tokens":      totalTokens,
		"total_human":       humanBytes(totalBytes),
		"estimated_cost_usd": math.Round(float64(totalTokens)*pricePerMillionTokens/1e6*10000) / 10000,
		"top10_hogs":        top10,
		"by_group":          groups,
		"sessions":          sessions,
	}
}

func humanBytes(b int) string {
	if b >= 1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
	if b >= 1024 {
		return fmt.Sprintf("%.1fkB", float64(b)/1024)
	}
	return fmt.Sprintf("%dB", b)
}
