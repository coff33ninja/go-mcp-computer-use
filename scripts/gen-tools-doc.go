package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Tool struct {
	Name        string
	Description string
}

// categoryForTool returns the category label for a given tool name.
// Keep this map sorted by category name, then tool name.
var categoryForTool = map[string]string{
	// Screenshot & Vision (10)
	"screenshot":        "Screenshot & Vision",
	"get_screen_size":   "Screenshot & Vision",
	"get_pixel_color":   "Screenshot & Vision",
	"get_screen_dpi":    "Screenshot & Vision",
	"get_display_modes": "Screenshot & Vision",
	"ocr":               "Screenshot & Vision",
	"ocr_languages":     "Screenshot & Vision",
	"find_image":        "Screenshot & Vision",
	"find_all_images":   "Screenshot & Vision",
	"record_screen":     "Screenshot & Vision",

	// Mouse (6)
	"click":              "Mouse",
	"move_mouse":         "Mouse",
	"scroll":             "Mouse",
	"drag":               "Mouse",
	"hover":              "Mouse",
	"get_cursor_position": "Mouse",

	// Keyboard (9 incl. keylogger)
	"type":                "Keyboard",
	"key_press":           "Keyboard",
	"type_and_submit":     "Keyboard",
	"select_all_and_type": "Keyboard",
	"key_down":            "Keyboard",
	"key_up":              "Keyboard",
	"keylogger_start":     "Keyboard",
	"keylogger_stop":      "Keyboard",
	"keylogger_status":    "Keyboard",

	// Window Management (13)
	"list_windows":        "Window Management",
	"focus_window":        "Window Management",
	"focus_window_by_title": "Window Management",
	"find_window":         "Window Management",
	"wait_for_window":     "Window Management",
	"move_window":         "Window Management",
	"minimize_window":     "Window Management",
	"maximize_window":     "Window Management",
	"restore_window":      "Window Management",
	"close_window":        "Window Management",
	"get_window_state":    "Window Management",
	"screenshot_element":  "Window Management",
	"get_active_window": "Window Management",
	"set_window_lock":   "Window Management",
	"clear_window_lock": "Window Management",

	// Chained / Composite (4) — note: type_and_submit, select_all_and_type,
	// and launch_and_wait are dual-listed in Keyboard / Process Management
	"find_text_and_click": "Chained / Composite",
	"wait_for_text":       "Chained / Composite",
	"click_menu_item":     "Chained / Composite",
	"launch_and_wait":     "Chained / Composite",

	// Chain Automation (2)
	"chain":       "Chain Automation",
	"chain_abort": "Chain Automation",

	// UI Automation (3)
	"uia_find":    "UI Automation",
	"uia_get_text": "UI Automation",
	"uia_invoke":  "UI Automation",

	// Browser Automation (4)
	"browser_navigate":     "Browser Automation",
	"browser_search":       "Browser Automation",
	"browser_new_tab":      "Browser Automation",
	"browser_focus_url_bar": "Browser Automation",

	// File Explorer (4)
	"explorer_focus":      "File Explorer",
	"explorer_open_path":  "File Explorer",
	"open_file_explorer":  "File Explorer",
	"open_file_location":  "File Explorer",

	// Audio (2)
	"list_audio_devices":    "Audio",
	"set_default_audio_device": "Audio",

	// Memory & Templates (10)
	"memory_set":      "Memory & Templates",
	"memory_get":      "Memory & Templates",
	"memory_search":   "Memory & Templates",
	"memory_list":     "Memory & Templates",
	"memory_forget":   "Memory & Templates",
	"template_store":  "Memory & Templates",
	"template_find":   "Memory & Templates",
	"template_list":   "Memory & Templates",
	"template_forget": "Memory & Templates",
	"layout_validate": "Memory & Templates",

	// ONNX ML (7)
	"onnx_detect":       "ONNX ML",
	"onnx_status":       "ONNX ML",
	"onnx_download":     "ONNX ML",
	"onnx_watch_start":  "ONNX ML",
	"onnx_watch_stop":   "ONNX ML",
	"onnx_watch_status": "ONNX ML",
	"onnx_watch_cache":  "ONNX ML",

	// Priors & Statistics (1)
	"priors_stats": "Priors & Statistics",

	// Training Pipeline (6)
	"training_save_sample":   "Training Pipeline",
	"training_list_samples":  "Training Pipeline",
	"training_stats":         "Training Pipeline",
	"training_mark_used":     "Training Pipeline",
	"find_ui_element":        "Training Pipeline",
	"training_cleanup_noise": "Training Pipeline",

	// Data Export (1)
	"export_yolo_dataset": "Data Export",

	// Data Logging (3)
	"datalog_query":  "Data Logging",
	"datalog_export": "Data Logging",
	"datalog_status": "Data Logging",

	// Adaptive Agent (3)
	"agent_analyze": "Adaptive Agent",
	"agent_suggest": "Adaptive Agent",
	"agent_train":   "Adaptive Agent",

	// Introspection & Debugging (4)
	"task_begin":            "Introspection & Debugging",
	"task_end":              "Introspection & Debugging",
	"introspection_analyze": "Introspection & Debugging",
	"bridge_debug":          "Introspection & Debugging",

	// Runtime Config (1)
	"set_config": "Runtime Config",

	// System (25)
	"get_volume":         "System",
	"set_volume":         "System",
	"set_mute":           "System",
	"get_clipboard":      "System",
	"set_clipboard":      "System",
	"get_brightness":     "System",
	"set_brightness":     "System",
	"get_battery":        "System",
	"get_disk_usage":     "System",
	"get_keyboard_layout": "System",
	"set_keyboard_layout": "System",
	"get_network_info":   "System",
	"ping":               "System",
	"get_system_info":    "System",
	"get_uptime":         "System",
	"get_idle_time":      "System",
	"list_displays":      "System",
	"open_url":           "System",
	"show_notification":  "System",
	"lock_workstation":   "System",
	"shutdown":           "System",
	"restart":            "System",
	"sleep":              "System",
	"hibernate":          "System",
	"wait":               "System",
	// Process Management (3) — launch_and_wait is in Chained / Composite
	"list_processes": "Process Management",
	"kill_process":   "Process Management",
	"launch_app":     "Process Management",
}

var categoryOrder = []string{
	"Screenshot & Vision",
	"Mouse",
	"Keyboard",
	"Window Management",
	"Chained / Composite",
	"Chain Automation",
	"UI Automation",
	"Browser Automation",
	"File Explorer",
	"Audio",
	"Memory & Templates",
	"ONNX ML",
	"Priors & Statistics",
	"Training Pipeline",
	"Data Export",
	"Data Logging",
	"Adaptive Agent",
	"Introspection & Debugging",
	"Runtime Config",
	"System",
	"Process Management",
}

func main() {
	// Find server.go relative to script location or CWD
	serverPath := findServerGo()
	if serverPath == "" {
		fmt.Fprintf(os.Stderr, "error: cannot find internal/server/server.go\n")
		os.Exit(1)
	}

	tools := parseTools(serverPath)
	if len(tools) == 0 {
		fmt.Fprintf(os.Stderr, "error: no tools found in %s\n", serverPath)
		os.Exit(1)
	}

	// Group by category
	byCategory := make(map[string][]Tool)
	seenCategories := make(map[string]bool)
	for _, t := range tools {
		cat := categoryForTool[t.Name]
		if cat == "" {
			cat = "Uncategorized"
		}
		byCategory[cat] = append(byCategory[cat], t)
		seenCategories[cat] = true
	}

	// Build output
	var b strings.Builder
	b.WriteString("# Tools (")
	b.WriteString(fmt.Sprintf("%d", len(tools)))
	b.WriteString(")\n\n")
	b.WriteString("Auto-generated from `internal/server/server.go`. ")
	b.WriteString(fmt.Sprintf("Total: **%d tools**.\n\n", len(tools)))

	// Print categories in order
	seen := make(map[string]bool)
	for _, cat := range categoryOrder {
		tools, ok := byCategory[cat]
		if !ok {
			continue
		}
		seen[cat] = true
		writeCategory(&b, cat, tools)
	}

	// Print any uncategorized or extra categories
	for cat, tools := range byCategory {
		if seen[cat] {
			continue
		}
		writeCategory(&b, cat, tools)
	}

	b.WriteString("<!--\n")
	b.WriteString(fmt.Sprintf("Generated by scripts/gen-tools-doc.go — %d tools found\n", len(tools)))
	b.WriteString("-->\n")

	// Write to docs/
	outputPath := filepath.Join("docs", "reference", "tools.md")
	if err := os.MkdirAll("docs/reference", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating docs/reference: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outputPath, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d tools)\n", outputPath, len(tools))

	// Patch hardcoded tool counts in other docs
	patchDocCounts(len(tools))
}

func writeCategory(b *strings.Builder, cat string, tools []Tool) {
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	b.WriteString(fmt.Sprintf("## %s (%d)\n\n", cat, len(tools)))
	for _, t := range tools {
		b.WriteString(fmt.Sprintf("- `%s` — %s\n", t.Name, t.Description))
	}
	b.WriteString("\n")
}

// patchDocCounts rewrites hardcoded tool counts in docs that are NOT auto-generated.
// Each entry is (filename, compiled regex, replacement template where $1 = group).
var docCountPatches = []struct {
	file  string
	re    *regexp.Regexp
	tmpl  string
}{
	{
		file: filepath.Join("README.md"),
		re:   regexp.MustCompile(`(?m)- \*\*\d+ MCP tools\*\*`),
		tmpl: "- **%d MCP tools**",
	},
	{
		file: filepath.Join("docs", "architecture.md"),
		re:   regexp.MustCompile(`(?m)\(\d+ tools\)`),
		tmpl: "(%d tools)",
	},
	{
		file: filepath.Join("docs", "comparison-vs-alternatives.md"),
		// Landscape table: "| 140 | — | ✅ MIT |" — match the cell before the stars/licence columns
		re: regexp.MustCompile(`(?m)(go-mcp-computer-use[^|]*\|[^|]*\|[^|]*\|[^|]*\|[^|]*\|[^|]*\|[^|]*\| )\d+( \| — \| ✅ MIT \|)`),
		tmpl: "${1}%d${2}",
	},
	{
		file: filepath.Join("docs", "comparison-vs-alternatives.md"),
		// Side-by-side tables: "| **Tool count** | 140 | ~30 |"
		re: regexp.MustCompile(`(?m)(\*\*Tool count\*\* \| )\d+( \| ~\d+ \|)`),
		tmpl: "${1}%d${2}",
	},
	{
		file: filepath.Join("docs", "comparison-vs-alternatives.md"),
		// "(140 tools)" references in DesktopCtl section
		re: regexp.MustCompile(`\(\d+ tools\)`),
		tmpl: "(%d tools)",
	},
	{
		file: filepath.Join("docs", "comparison-vs-alternatives.md"),
		// bare "N tools" at end of table cell: "memory, 132 tools |"
		re: regexp.MustCompile(`\d+ tools \|`),
		tmpl: "%d tools |",
	},
	{
		file: filepath.Join("docs", "comparison-vs-alternatives.md"),
		// bare "N tools)" in prose: "memory, 132 tools)"
		re: regexp.MustCompile(`, \d+ tools\)`),
		tmpl: ", %d tools)",
	},
	{
		file: filepath.Join("docs", "meta", "backlog.md"),
		// "~130 tools" strategy line
		re: regexp.MustCompile(`~\d+ tools\)`),
		tmpl: "~%d tools)",
	},
	{
		file: filepath.Join("docs", "reference", "codebase-map.md"),
		// "lists all 140 tools"
		re: regexp.MustCompile(`lists all \d+ tools`),
		tmpl: "lists all %d tools",
	},
	{
		file: filepath.Join("docs", "guides", "agent-guides.md"),
		// "140-tool set"
		re: regexp.MustCompile(`\d+-tool set`),
		tmpl: "%d-tool set",
	},
	{
		file: filepath.Join("docs", "guides", "computer-use-guide-for-ai-agents.md"),
		// "**140 tools**"
		re: regexp.MustCompile(`\*\*\d+ tools\*\*`),
		tmpl: "**%d tools**",
	},
	{
		file: filepath.Join("docs", "meta", "known-issues.md"),
		// "exposes 140 tools"
		re: regexp.MustCompile(`exposes \d+ tools`),
		tmpl: "exposes %d tools",
	},
	{
		file: filepath.Join("docs", "reference", "scripts.md"),
		// "all 140 MCP tool definitions"
		re: regexp.MustCompile(`all \d+ MCP tool`),
		tmpl: "all %d MCP tool",
	},
}

func patchDocCounts(count int) {
	for _, p := range docCountPatches {
		data, err := os.ReadFile(p.file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: cannot read %s: %v\n", p.file, err)
			continue
		}
		orig := string(data)
		var result string
		if strings.Contains(p.tmpl, "${1}") {
			// Group-reference replacement: substitute %d first, then let regex handle ${1}/${2}
			tmpl := strings.Replace(p.tmpl, "%d", strconv.Itoa(count), 1)
			result = p.re.ReplaceAllString(orig, tmpl)
		} else {
			// Simple count replacement
			result = p.re.ReplaceAllString(orig, fmt.Sprintf(p.tmpl, count))
		}
		if result != orig {
			if err := os.WriteFile(p.file, []byte(result), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot write %s: %v\n", p.file, err)
				continue
			}
			fmt.Printf("patched tool count in %s\n", p.file)
		}
	}
}

func findServerGo() string {
	candidates := []string{
		filepath.Join("internal", "server", "server.go"),
		filepath.Join("..", "internal", "server", "server.go"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func parseTools(path string) []Tool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", path, err)
		os.Exit(1)
	}

	var tools []Tool

	// Walk all function calls looking for mcp.AddTool
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "AddTool" {
				return true
			}
			// mcp.AddTool(server, &mcp.Tool{Name: "...", Description: "..."}, handler)
			if len(call.Args) < 2 {
				return true
			}
			// Args[1] is &mcp.Tool{...} — a UnaryExpr wrapping CompositeLit
			arg := call.Args[1]
			if unary, ok := arg.(*ast.UnaryExpr); ok {
				arg = unary.X
			}
			compLit, ok := arg.(*ast.CompositeLit)
			if !ok {
				return true
			}
			var t Tool
			for _, elt := range compLit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Name":
					t.Name = stringValue(kv.Value)
				case "Description":
					t.Description = stringValue(kv.Value)
				}
			}
			if t.Name != "" {
				tools = append(tools, t)
			}
			return true
		})
	}

	return tools
}

func stringValue(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return strings.Trim(lit.Value, "\"")
	}
	return s
}
