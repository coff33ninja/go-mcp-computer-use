package actions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// ── Step types ──

const (
	StepTool = "tool"
	StepWait = "wait"
)

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
}

type ChainResult struct {
	Success   bool                   `json:"success"`
	StepCount int                    `json:"step_count"`
	Results   []StepResult           `json:"results"`
	Variables map[string]any         `json:"variables,omitempty"`
}

type StepResult struct {
	Index   int    `json:"index"`
	Tool    string `json:"tool,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Output  any    `json:"output,omitempty"`
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
		for i, step := range req.Steps {
			stepType := step.Type
			if stepType == "" {
				stepType = StepTool
			}

			stepArgs := substituteVars(step.Args, state.variables)
			stepResult := StepResult{Index: i}

			switch stepType {
			case StepWait:
				stepResult = execWait(step, state)
			default:
				stepResult = execTool(step, stepArgs, state)
			}

			result.Results = append(result.Results, stepResult)
			result.StepCount++

			if step.Capture != "" && stepResult.Success {
				state.variables[step.Capture] = stepResult.Output
			}

			if !stepResult.Success && state.onError == "stop" {
				break
			}
		}
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

// ── Variable substitution ──

var varPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

func substituteVars(args map[string]any, vars map[string]any) map[string]any {
	if args == nil || vars == nil {
		return args
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		s, ok := v.(string)
		if ok {
			out[k] = varPattern.ReplaceAllStringFunc(s, func(match string) string {
				name := varPattern.FindStringSubmatch(match)[1]
				if val, exists := vars[name]; exists {
					return fmt.Sprintf("%v", val)
				}
				return match
			})
		} else {
			out[k] = v
		}
	}
	return out
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

// ── ChainFromJSON ──

func ChainFromJSON(jsonStr string) (*ChainResult, error) {
	var req ChainRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return nil, fmt.Errorf("chain: invalid JSON: %w", err)
	}
	return ExecuteChain(req)
}


