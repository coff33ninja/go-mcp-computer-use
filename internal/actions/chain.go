package actions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ── Step types ──

const (
	StepTool = "tool"
	StepWait = "wait"
	StepPoll = "poll"
	StepIf   = "if"
	StepLoop = "loop"
)

// ── Poll / If / Loop config ──

type PollConfig struct {
	EveryMs     int    `json:"every_ms"`
	TimeoutMs   int    `json:"timeout_ms"`
	OCRContains string `json:"ocr_contains"`
}

type IfConfig struct {
	OCRContains string `json:"ocr_contains"`
	Then        []any  `json:"then,omitempty"`
	Else        []any  `json:"else,omitempty"`
}

type LoopConfig struct {
	Times int   `json:"times"`
	Steps []any `json:"steps,omitempty"`
}

// ── Data structures ──

type ChainRequest struct {
	Steps     []ChainStep `json:"steps"`
	TimeoutMs int         `json:"timeout_ms,omitempty"`
	OnError   string      `json:"on_error,omitempty"`
}

type ChainStep struct {
	Type    string         `json:"type,omitempty"`
	Capture string         `json:"capture,omitempty"`
	Tool    string         `json:"tool,omitempty"`
	Args    map[string]any `json:"args,omitempty"`
	WaitMs  int            `json:"wait_ms,omitempty"`
	Poll    *PollConfig    `json:"poll,omitempty"`
	If      *IfConfig      `json:"if,omitempty"`
	Loop    *LoopConfig    `json:"loop,omitempty"`
}

type ChainResult struct {
	Success   bool                   `json:"success"`
	StepCount int                    `json:"step_count"`
	Results   []StepResult           `json:"results"`
	Variables map[string]any         `json:"variables,omitempty"`
}

type StepResult struct {
	Index   int          `json:"index"`
	Tool    string       `json:"tool,omitempty"`
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
	Output  any          `json:"output,omitempty"`
	Steps   []StepResult `json:"steps,omitempty"`
}

// ── Tool dispatch ──

type toolFunc func(args map[string]any) (any, error)

var toolDispatch map[string]toolFunc

func init() {
	toolDispatch = map[string]toolFunc{
		"click":               chainClick,
		"move_mouse":          chainMoveMouse,
		"get_cursor_position": chainGetCursorPos,
		"scroll":              chainScroll,
		"drag":                chainDrag,
		"type":                chainType,
		"key_press":           chainKeyPress,
		"type_and_submit":     chainTypeAndSubmit,
		"select_all_and_type": chainSelectAllAndType,
		"screenshot":          chainScreenshot,
		"get_screen_size":     chainGetScreenSize,
		"get_pixel_color":     chainGetPixelColor,
		"ocr":                 chainOCR,
		"wait":                chainWaitTool,
		"hover":               chainHover,
		"list_windows":        chainListWindows,
		"get_system_info":     chainGetSystemInfo,
		"get_clipboard":       chainGetClipboard,
		"set_clipboard":       chainSetClipboard,
		"get_volume":          chainGetVolume,
		"set_volume":          chainSetVolume,
		"set_mute":            chainSetMute,
		"get_battery":         chainGetBattery,
		"get_uptime":          chainGetUptime,
		"get_idle_time":       chainGetIdleTime,
		"get_network_info":    chainGetNetworkInfo,
		"ping":                chainPing,
		"get_disk_usage":      chainGetDiskUsage,
		"open_url":            chainOpenURL,
		"show_notification":   chainShowNotification,
		"list_displays":       chainListDisplays,
		"launch_app":          chainLaunchApp,
		"kill_process":        chainKillProcess,
		"list_processes":      chainListProcesses,
		"focus_window":        chainFocusWindow,
		"minimize_window":     chainMinimizeWindow,
		"maximize_window":     chainMaximizeWindow,
		"restore_window":      chainRestoreWindow,
		"close_window":        chainCloseWindow,
		"get_window_state":    chainGetWindowState,
		"find_text_and_click": chainFindTextAndClick,
		"wait_for_text":       chainWaitForText,
		"wait_for_window":     chainWaitForWindow,
		"launch_and_wait":     chainLaunchAndWait,
		"find_window":         chainFindWindow,
	}
}

// ── Chain execution ──

func ExecuteChain(req ChainRequest) (*ChainResult, error) {
	state := &chainState{
		variables: make(map[string]any),
		onError:   req.OnError,
	}

	if state.onError == "" {
		state.onError = "stop"
	}

	globalTimeout := defaultTimeout(req.TimeoutMs)

	result := &ChainResult{
		Variables: state.variables,
	}

	done := make(chan bool, 1)
	go func() {
		results, sc := execSteps(req.Steps, state)
		result.Results = results
		result.StepCount = sc
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(globalTimeout):
		return nil, fmt.Errorf("chain timed out after %v", globalTimeout)
	}

	for _, r := range result.Results {
		if !r.Success {
			result.Success = false
			return result, nil
		}
	}
	result.Success = true
	return result, nil
}

// execSteps executes a slice of steps and collects results. Returns results and count.
// Used recursively by if/loop steps.
func execSteps(steps []ChainStep, state *chainState) ([]StepResult, int) {
	var results []StepResult
	var stepCount int
	for i, step := range steps {
		stepType := step.Type
		if stepType == "" {
			stepType = detectStepType(step)
		}

		stepArgs := substituteVars(step.Args, state.variables)
		var stepResult StepResult

		switch stepType {
		case StepWait:
			stepResult = execWait(step, state)
		case StepPoll:
			stepResult = execPoll(step, state)
		case StepIf:
			stepResult = execIf(step, state)
		case StepLoop:
			stepResult = execLoop(step, state)
		default:
			stepResult = execTool(step, stepArgs, state)
		}
		stepResult.Index = i

		if step.Capture != "" && stepResult.Success {
			state.variables[step.Capture] = stepResult.Output
		}

		results = append(results, stepResult)
		stepCount++

		if !stepResult.Success && state.onError == "stop" {
			break
		}
	}
	return results, stepCount
}

type chainState struct {
	variables map[string]any
	onError   string
}

func defaultTimeout(ms int) time.Duration {
	if ms <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(ms) * time.Millisecond
}

// ── Wait step ──

func execWait(step ChainStep, _ *chainState) StepResult {
	if step.WaitMs <= 0 {
		return StepResult{Success: true}
	}
	time.Sleep(time.Duration(step.WaitMs) * time.Millisecond)
	return StepResult{Success: true}
}

// ── Tool step ──

func execTool(step ChainStep, args map[string]any, _ *chainState) StepResult {
	fn, ok := toolDispatch[step.Tool]
	if !ok {
		return StepResult{
			Tool:    step.Tool,
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", step.Tool),
		}
	}
	output, err := fn(args)
	if err != nil {
		return StepResult{
			Tool:    step.Tool,
			Success: false,
			Error:   err.Error(),
		}
	}
	return StepResult{
		Tool:    step.Tool,
		Success: true,
		Output:  output,
	}
}

// ── Poll step ──

func execPoll(step ChainStep, state *chainState) StepResult {
	cfg := step.Poll
	if cfg == nil {
		return StepResult{Tool: "poll", Success: false, Error: "missing poll config"}
	}
	if cfg.OCRContains == "" {
		return StepResult{Tool: "poll", Success: false, Error: "poll: ocr_contains required"}
	}

	everyMs := cfg.EveryMs
	if everyMs <= 0 {
		everyMs = 500
	}
	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	lowerText := strings.ToLower(cfg.OCRContains)

	for time.Now().Before(deadline) {
		result, err := OCRScreen("")
		if err == nil {
			for _, word := range result.Words {
				if strings.Contains(strings.ToLower(word.Text), lowerText) {
					return StepResult{
						Tool:    "poll",
						Success: true,
						Output:  map[string]any{"found": true, "text": word.Text},
					}
				}
			}
			for _, line := range result.Lines {
				if strings.Contains(strings.ToLower(line.Text), lowerText) {
					return StepResult{
						Tool:    "poll",
						Success: true,
						Output:  map[string]any{"found": true, "text": line.Text},
					}
				}
			}
		}
		time.Sleep(time.Duration(everyMs) * time.Millisecond)
	}

	return StepResult{
		Tool:    "poll",
		Success: false,
		Error:   fmt.Sprintf("poll: text %q not found within %dms", cfg.OCRContains, timeoutMs),
	}
}

// ── If step ──

func execIf(step ChainStep, state *chainState) StepResult {
	cfg := step.If
	if cfg == nil {
		return StepResult{Tool: "if", Success: false, Error: "missing if config"}
	}
	if cfg.OCRContains == "" {
		return StepResult{Tool: "if", Success: false, Error: "if: ocr_contains required"}
	}

	// Check condition via OCR
	lowerText := strings.ToLower(cfg.OCRContains)
	conditionMet := false

	result, err := OCRScreen("")
	if err == nil {
		for _, word := range result.Words {
			if strings.Contains(strings.ToLower(word.Text), lowerText) {
				conditionMet = true
				break
			}
		}
		if !conditionMet {
			for _, line := range result.Lines {
				if strings.Contains(strings.ToLower(line.Text), lowerText) {
					conditionMet = true
					break
				}
			}
		}
	}

	var branch string
	var subSteps []ChainStep
	if conditionMet {
		branch = "then"
		subSteps = rawToSteps(cfg.Then)
	} else {
		branch = "else"
		subSteps = rawToSteps(cfg.Else)
	}

	subResults, _ := execSteps(subSteps, state)
	return StepResult{
		Tool:    "if",
		Success: true,
		Output:  map[string]any{"condition": fmt.Sprintf("ocr_contains: %s", cfg.OCRContains), "branch": branch},
		Steps:   subResults,
	}
}

// ── Loop step ──

func execLoop(step ChainStep, state *chainState) StepResult {
	cfg := step.Loop
	if cfg == nil {
		return StepResult{Tool: "loop", Success: false, Error: "missing loop config"}
	}
	if cfg.Times <= 0 {
		return StepResult{Tool: "loop", Success: true}
	}

	subSteps := rawToSteps(cfg.Steps)
	var allResults []StepResult
	for iter := 0; iter < cfg.Times; iter++ {
		subResults, _ := execSteps(subSteps, state)
		allResults = append(allResults, subResults...)

		// Stop early if on_error=stop and a sub-step failed
		if state.onError == "stop" {
			for _, r := range subResults {
				if !r.Success {
					return StepResult{
						Tool:    "loop",
						Success: false,
						Error:   fmt.Sprintf("loop iteration %d failed", iter),
						Steps:   allResults,
					}
				}
			}
		}
	}

	return StepResult{
		Tool:    "loop",
		Success: true,
		Output:  map[string]any{"iterations": cfg.Times},
		Steps:   allResults,
	}
}

// ── Step type detection ──

func detectStepType(s ChainStep) string {
	if s.WaitMs > 0 {
		return StepWait
	}
	if s.Poll != nil {
		return StepPoll
	}
	if s.If != nil {
		return StepIf
	}
	if s.Loop != nil {
		return StepLoop
	}
	if s.Tool != "" {
		return StepTool
	}
	return StepTool
}

// ── Variable substitution ──

var varPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

func substituteVars(args map[string]any, vars map[string]any) map[string]any {
	if args == nil || vars == nil {
		return args
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		s, ok := v.(string)
		if ok {
			out[k] = varPattern.ReplaceAllStringFunc(s, func(match string) string {
				path := varPattern.FindStringSubmatch(match)[1]
				return resolveVarPath(path, vars)
			})
		} else {
			out[k] = v
		}
	}
	return out
}

// resolveVarPath resolves "var.field" or "var" paths against vars map.
func resolveVarPath(path string, vars map[string]any) string {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		if val, exists := vars[parts[0]]; exists {
			return fmt.Sprintf("%v", val)
		}
		return path // not found, return unresolved placeholder
	}
	// parts[0]=varName, parts[1]=fieldName
	if val, exists := vars[parts[0]]; exists {
		if m, ok := val.(map[string]any); ok {
			if field, exists := m[parts[1]]; exists {
				return fmt.Sprintf("%v", field)
			}
		}
	}
	return path
}

// rawToSteps converts a []any (from JSON unmarshaling) to []ChainStep.
// Used to break the circular type chain in IfConfig/LoopConfig (MCP SDK
// schema generator panics on recursive types).
func rawToSteps(raw []any) []ChainStep {
	if raw == nil {
		return nil
	}
	steps := make([]ChainStep, len(raw))
	for i, r := range raw {
		b, err := json.Marshal(r)
		if err != nil {
			continue
		}
		json.Unmarshal(b, &steps[i])
	}
	return steps
}

// ── Arg helpers ──

func getFloat(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, _ := val.Float64()
		return f, true
	}
	return 0, false
}

func getInt(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case json.Number:
		n, _ := val.Int64()
		return int(n), true
	}
	return 0, false
}

func getString(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func getStringSlice(m map[string]any, key string) ([]string, bool) {
	v, ok := m[key]
	if !ok {
		return nil, false
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, len(raw))
	for i, r := range raw {
		out[i], _ = r.(string)
	}
	return out, true
}

func getBool(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// ── Tool implementations ──

func chainClick(args map[string]any) (any, error) {
	x, _ := getInt(args, "x")
	y, _ := getInt(args, "y")
	button, _ := getString(args, "button")
	clicks, _ := getInt(args, "clicks")
	if button == "" {
		button = "left"
	}
	if clicks <= 0 {
		clicks = 1
	}
	err := Click(ClickInput{X: int32(x), Y: int32(y), Button: button, Clicks: clicks})
	return nil, err
}

func chainMoveMouse(args map[string]any) (any, error) {
	x, _ := getInt(args, "x")
	y, _ := getInt(args, "y")
	return nil, MoveMouse(int32(x), int32(y))
}

func chainGetCursorPos(_ map[string]any) (any, error) {
	x, y, err := GetCursorPosition()
	if err != nil {
		return nil, err
	}
	return map[string]any{"x": x, "y": y}, nil
}

func chainScroll(args map[string]any) (any, error) {
	clicks, ok := getInt(args, "clicks")
	if !ok {
		return nil, fmt.Errorf("scroll: clicks required")
	}
	return nil, Scroll(int32(clicks))
}

func chainDrag(args map[string]any) (any, error) {
	fromX, _ := getInt(args, "from_x")
	fromY, _ := getInt(args, "from_y")
	toX, _ := getInt(args, "to_x")
	toY, _ := getInt(args, "to_y")
	return nil, Drag(int32(fromX), int32(fromY), int32(toX), int32(toY))
}

func chainType(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("type: text required")
	}
	return nil, TypeText(text)
}

func chainKeyPress(args map[string]any) (any, error) {
	keys, ok := getStringSlice(args, "keys")
	if !ok {
		return nil, fmt.Errorf("key_press: keys required")
	}
	return nil, KeyPress(keys)
}

func chainTypeAndSubmit(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("type_and_submit: text required")
	}
	return nil, TypeAndSubmit(text)
}

func chainSelectAllAndType(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("select_all_and_type: text required")
	}
	return nil, SelectAllAndType(text)
}

func chainScreenshot(args map[string]any) (any, error) {
	if x, ok := getInt(args, "x"); ok {
		y, _ := getInt(args, "y")
		w, _ := getInt(args, "w")
		h, _ := getInt(args, "h")
		return CaptureRegion(int32(x), int32(y), int32(w), int32(h))
	}
	return CaptureScreen()
}

func chainGetScreenSize(_ map[string]any) (any, error) {
	w, h := ScreenSize()
	return map[string]any{"width": w, "height": h}, nil
}

func chainGetPixelColor(args map[string]any) (any, error) {
	x, _ := getInt(args, "x")
	y, _ := getInt(args, "y")
	return GetPixelColor(int32(x), int32(y))
}

func chainOCR(args map[string]any) (any, error) {
	lang, _ := getString(args, "language")
	if x, ok := getInt(args, "x"); ok {
		y, _ := getInt(args, "y")
		w, _ := getInt(args, "w")
		h, _ := getInt(args, "h")
		return OCRRegion(int32(x), int32(y), int32(w), int32(h), lang)
	}
	return OCRScreen(lang)
}

func chainWaitTool(args map[string]any) (any, error) {
	ms, ok := getInt(args, "ms")
	if ok {
		Wait(int32(ms))
	}
	return nil, nil
}

func chainHover(args map[string]any) (any, error) {
	x, _ := getInt(args, "x")
	y, _ := getInt(args, "y")
	return nil, Hover(int32(x), int32(y))
}

func chainListWindows(_ map[string]any) (any, error) {
	return ListWindows()
}

func chainGetSystemInfo(_ map[string]any) (any, error) {
	return GetSystemInfo()
}

func chainGetClipboard(_ map[string]any) (any, error) {
	return GetClipboardText()
}

func chainSetClipboard(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("set_clipboard: text required")
	}
	return nil, SetClipboardText(text)
}

func chainGetVolume(_ map[string]any) (any, error) {
	vol, err := GetVolume()
	if err != nil {
		return nil, err
	}
	return map[string]any{"volume": vol}, nil
}

func chainSetVolume(args map[string]any) (any, error) {
	pct, ok := getInt(args, "percent")
	if !ok {
		pct, ok = getInt(args, "volume")
	}
	if !ok {
		return nil, fmt.Errorf("set_volume: percent required")
	}
	return nil, SetVolume(uint32(pct))
}

func chainSetMute(args map[string]any) (any, error) {
	mute, ok := getBool(args, "mute")
	if !ok {
		return nil, fmt.Errorf("set_mute: mute required")
	}
	return nil, SetMute(mute)
}

func chainGetBattery(_ map[string]any) (any, error) {
	return GetBattery()
}

func chainGetUptime(_ map[string]any) (any, error) {
	d, err := GetUptime()
	if err != nil {
		return nil, err
	}
	return map[string]any{"uptime_ms": d.Milliseconds()}, nil
}

func chainGetIdleTime(_ map[string]any) (any, error) {
	d, err := GetIdleTime()
	if err != nil {
		return nil, err
	}
	return map[string]any{"idle_ms": d.Milliseconds()}, nil
}

func chainGetNetworkInfo(_ map[string]any) (any, error) {
	return GetNetworkInfo()
}

func chainPing(args map[string]any) (any, error) {
	host, ok := getString(args, "host")
	if !ok {
		return nil, fmt.Errorf("ping: host required")
	}
	return PingHost(host)
}

func chainGetDiskUsage(_ map[string]any) (any, error) {
	return GetDiskUsage()
}

func chainOpenURL(args map[string]any) (any, error) {
	url, ok := getString(args, "url")
	if !ok {
		return nil, fmt.Errorf("open_url: url required")
	}
	return nil, OpenURL(url)
}

func chainShowNotification(args map[string]any) (any, error) {
	title, _ := getString(args, "title")
	msg, _ := getString(args, "message")
	return nil, ShowNotification(title, msg)
}

func chainListDisplays(_ map[string]any) (any, error) {
	return ListDisplays()
}

func chainLaunchApp(args map[string]any) (any, error) {
	path, ok := getString(args, "path")
	if !ok {
		return nil, fmt.Errorf("launch_app: path required")
	}
	return nil, LaunchApp(path)
}

func chainKillProcess(args map[string]any) (any, error) {
	pid, ok := getInt(args, "pid")
	if !ok {
		return nil, fmt.Errorf("kill_process: pid required")
	}
	return nil, KillProcess(uint32(pid))
}

func chainListProcesses(_ map[string]any) (any, error) {
	return ListProcesses()
}

func chainFocusWindow(args map[string]any) (any, error) {
	handle, ok := getFloat(args, "handle")
	if !ok {
		h, ok2 := getInt(args, "handle")
		if !ok2 {
			return nil, fmt.Errorf("focus_window: handle required")
		}
		handle = float64(h)
	}
	return nil, FocusWindow(uintptr(handle))
}

func chainMinimizeWindow(args map[string]any) (any, error) {
	h, _ := getFloat(args, "handle")
	return nil, MinimizeWindow(uintptr(h))
}

func chainMaximizeWindow(args map[string]any) (any, error) {
	h, _ := getFloat(args, "handle")
	return nil, MaximizeWindow(uintptr(h))
}

func chainRestoreWindow(args map[string]any) (any, error) {
	h, _ := getFloat(args, "handle")
	return nil, RestoreWindow(uintptr(h))
}

func chainCloseWindow(args map[string]any) (any, error) {
	h, _ := getFloat(args, "handle")
	return nil, CloseWindow(uintptr(h))
}

func chainGetWindowState(args map[string]any) (any, error) {
	h, _ := getFloat(args, "handle")
	return GetWindowState(uintptr(h))
}

func chainFindTextAndClick(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("find_text_and_click: text required")
	}
	opts := FindTextOpts{Text: text}
	if lang, ok := getString(args, "language"); ok {
		opts.Language = lang
	}
	if x, ok := getInt(args, "x"); ok {
		v := int32(x)
		opts.RegionX = &v
	}
	if y, ok := getInt(args, "y"); ok {
		v := int32(y)
		opts.RegionY = &v
	}
	if w, ok := getInt(args, "w"); ok {
		v := int32(w)
		opts.RegionW = &v
	}
	if h, ok := getInt(args, "h"); ok {
		v := int32(h)
		opts.RegionH = &v
	}
	return nil, FindTextAndClick(opts)
}

func chainWaitForText(args map[string]any) (any, error) {
	text, ok := getString(args, "text")
	if !ok {
		return nil, fmt.Errorf("wait_for_text: text required")
	}
	timeoutMs, _ := getInt(args, "timeout_ms")
	lang, _ := getString(args, "language")
	return WaitForText(text, int32(timeoutMs), lang)
}

func chainWaitForWindow(args map[string]any) (any, error) {
	title, ok := getString(args, "title")
	if !ok {
		return nil, fmt.Errorf("wait_for_window: title required")
	}
	timeoutMs, _ := getInt(args, "timeout_ms")
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	hwnd, err := WaitForWindow(title, int32(timeoutMs))
	if err != nil {
		return map[string]any{"found": false}, nil
	}
	return map[string]any{"handle": hwnd, "found": true}, nil
}

func chainLaunchAndWait(args map[string]any) (any, error) {
	path, ok := getString(args, "path")
	if !ok {
		return nil, fmt.Errorf("launch_and_wait: path required")
	}
	title, ok := getString(args, "window_title")
	if !ok {
		return nil, fmt.Errorf("launch_and_wait: window_title required")
	}
	timeoutMs, _ := getInt(args, "timeout_ms")
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	hwnd, err := LaunchAndWait(path, title, int32(timeoutMs))
	if err != nil {
		return map[string]any{"found": false}, nil
	}
	return map[string]any{"handle": hwnd, "found": true}, nil
}

func chainFindWindow(args map[string]any) (any, error) {
	title, ok := getString(args, "title")
	if !ok {
		return nil, fmt.Errorf("find_window: title required")
	}
	hwnd := FindWindowByTitle(title)
	return map[string]any{"handle": hwnd, "found": hwnd != 0}, nil
}

// ── ChainFromJSON ──

func ChainFromJSON(jsonStr string) (*ChainResult, error) {
	var req ChainRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return nil, fmt.Errorf("chain: invalid JSON: %w", err)
	}
	return ExecuteChain(req)
}


