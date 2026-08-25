package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	jsonschema "github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/coff33ninja/go-mcp-computer-use/internal/actions"
	"github.com/coff33ninja/go-mcp-computer-use/internal/config"
	"github.com/coff33ninja/go-mcp-computer-use/internal/logging"
)

func shouldVerify(autoVerify *bool, expected *actions.ExpConfig) bool {
	return (autoVerify != nil && *autoVerify) || expected != nil
}

// cleanNullableTypes recursively strips nullable union types from a JSON Schema.
// The Go jsonschema library generates "type": ["null", "X"] for pointer types (*int32, *bool, etc.).
// The opencode MCP client cannot serialize values for nullable union types, producing truncated JSON.
// This function converts ["null", "X"] → "X" for all properties recursively.
func cleanNullableTypes(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	if len(s.Types) > 0 {
		nonNull := make([]string, 0, len(s.Types))
		for _, t := range s.Types {
			if t != "null" {
				nonNull = append(nonNull, t)
			}
		}
		if len(nonNull) == 1 {
			s.Type = nonNull[0]
			s.Types = nil
		} else if len(nonNull) > 1 {
			s.Types = nonNull
		} else {
			s.Types = nil
		}
	}
	for _, p := range s.Properties {
		cleanNullableTypes(p)
	}
	if s.Items != nil {
		cleanNullableTypes(s.Items)
	}
	if s.AdditionalProperties != nil {
		cleanNullableTypes(s.AdditionalProperties)
	}
	for _, v := range s.Defs {
		cleanNullableTypes(v)
	}
}

// addToolClean is a wrapper around mcp.AddTool that auto-generates the JSON Schema
// from the Go In type and strips nullable union types before registering.
// This fixes the opencode MCP client bug where ["null", "integer"] etc. cause
// truncated JSON serialization (e.g. {"x":  → {"x": .
func addToolClean[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	rt := reflect.TypeFor[In]()
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() == reflect.Struct && tool.InputSchema == nil {
		schema, err := jsonschema.ForType(rt, &jsonschema.ForOptions{})
		if err == nil {
			schema.Schema = ""
			cleanNullableTypes(schema)
			tool.InputSchema = schema
		}
	}
	mcp.AddTool(server, tool, handler)
}

func safeHandler[Args any](name string, fn func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error)) func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args Args) (result *mcp.CallToolResult, payload any, err error) {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])
				slog.Error("panic in tool handler", "tool", name, "panic", fmt.Sprintf("%v", r), "stack", stack)
				result = nil
				payload = nil
				err = fmt.Errorf("panic in %s: %v", name, r)
			}
		}()
		return fn(ctx, req, args)
	}
}

func verifyCfg(ec *actions.ExpConfig, rx, ry, rw, rh *int32) *actions.VerifyConfig {
	if ec == nil {
		return &actions.VerifyConfig{RegionX: rx, RegionY: ry, RegionW: rw, RegionH: rh, AfterWaitMs: 300}
	}
	return &actions.VerifyConfig{
		ExpectedText: ec.Text, NotText: ec.NotText,
		RegionX: rx, RegionY: ry, RegionW: rw, RegionH: rh,
		AfterWaitMs: ec.WaitMs,
	}
}

func preVerifyCheck(ec *actions.ExpConfig, rx, ry, rw, rh *int32) error {
	if ec == nil {
		return nil
	}
	vr := actions.VerifyAction(verifyCfg(ec, rx, ry, rw, rh))
	if !vr.Passed {
		return fmt.Errorf("precondition: %s", vr.Reason)
	}
	return nil
}

func verifiedResult(extra any, vr *actions.VerifyResult) (*mcp.CallToolResult, any, error) {
	if vr == nil {
		if extra == nil {
			return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
		}
		return &mcp.CallToolResult{}, extra, nil
	}
	out := map[string]any{"ok": true, "verified": vr.Passed, "verification": vr}
	if extra != nil {
		out["extra"] = extra
	}
	return &mcp.CallToolResult{}, out, nil
}

type VerifyArgs struct {
	AutoVerify  *bool              `json:"auto_verify,omitempty"`
	Expected    *actions.ExpConfig `json:"expected,omitempty"`
	PreExpected *actions.ExpConfig `json:"pre_expected,omitempty"`
}

type ScreenshotArgs struct {
	X *int32 `json:"x,omitempty"`
	Y *int32 `json:"y,omitempty"`
	W *int32 `json:"w,omitempty"`
	H *int32 `json:"h,omitempty"`
}

type ClickArgs struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Button string `json:"button,omitempty"`
	Clicks int    `json:"clicks,omitempty"`
	VerifyArgs
}

type MoveMouseArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

	type ScrollArgs struct {
		Clicks     int32  `json:"clicks"`
		Direction  string `json:"direction,omitempty"`
		Horizontal bool   `json:"horizontal,omitempty"`
		VerifyArgs
	}

type KeyPressArgs struct {
	Keys []string `json:"keys"`
	VerifyArgs
}

type KeyEventArgs struct {
	Key string `json:"key"`
}

type KeyloggerStartArgs struct{}

type KeyloggerStatusArgs struct{}

type RecordReplicateArgs struct {
	DurationSecs int `json:"duration_secs"`
	DelayMs      int `json:"delay_ms,omitempty"`
	Slowdown     int `json:"slowdown,omitempty"`
	Loop         int `json:"loop,omitempty"`
}

type TypeArgs struct {
	Text string `json:"text"`
	VerifyArgs
}

type ScreenSizeResult struct {
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type CursorPosResult struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type DragArgs struct {
	FromX int32 `json:"from_x"`
	FromY int32 `json:"from_y"`
	ToX   int32 `json:"to_x"`
	ToY   int32 `json:"to_y"`
	VerifyArgs
}

type FocusWindowArgs struct {
	Handle uintptr `json:"handle"`
}

type ListWindowsResult struct {
	Windows []actions.WindowInfo `json:"windows"`
}

type SetVolumeArgs struct {
	Percent uint32 `json:"percent"`
}

type MuteArgs struct {
	Mute bool `json:"mute"`
}

type SetClipboardArgs struct {
	Text string `json:"text"`
}

type OpenURLArgs struct {
	URL string `json:"url"`
	VerifyArgs
}

type WaitArgs struct {
	Ms int32 `json:"ms"`
}

type PixelColorArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type LaunchAppArgs struct {
	Path string `json:"path"`
	VerifyArgs
}

type KillProcessArgs struct {
	PID uint32 `json:"pid"`
}

type MoveWindowArgs struct {
	Handle uintptr `json:"handle"`
	X      int32   `json:"x"`
	Y      int32   `json:"y"`
	Width  int32   `json:"width"`
	Height int32   `json:"height"`
}

type WindowHandleArgs struct {
	Handle uintptr `json:"handle"`
}

type FindWindowArgs struct {
	Title string `json:"title"`
}

type WaitForWindowArgs struct {
	Title     string `json:"title"`
	TimeoutMs int32  `json:"timeout_ms,omitempty"`
}

type NotificationArgs struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type OCRArgs struct {
	X        *int32 `json:"x,omitempty"`
	Y        *int32 `json:"y,omitempty"`
	W        *int32 `json:"w,omitempty"`
	H        *int32 `json:"h,omitempty"`
	Language string `json:"language,omitempty"`
}

type BrightnessArgs struct {
	Percent int `json:"percent"`
}

type PingArgs struct {
	Host string `json:"host"`
}

type FindTextAndClickArgs struct {
	Text      string  `json:"text"`
	Language  string  `json:"language,omitempty"`
	X         *int32  `json:"x,omitempty"`
	Y         *int32  `json:"y,omitempty"`
	W         *int32  `json:"w,omitempty"`
	H         *int32  `json:"h,omitempty"`
	MaxScrolls    *int32 `json:"max_scrolls,omitempty"`
	ScrollClicks  *int32 `json:"scroll_clicks,omitempty"`
	ScrollDown    *bool  `json:"scroll_down,omitempty"`
	WindowTitle   string `json:"window_title,omitempty"`
	SkipMemory    *bool  `json:"skip_memory,omitempty"`
	SkipSystemFind *bool `json:"skip_system_find,omitempty"`
	VerifyArgs
}

type TypeAndSubmitArgs struct {
	Text string `json:"text"`
	VerifyArgs
}

type LaunchAndWaitArgs struct {
	Path        string `json:"path"`
	WindowTitle string `json:"window_title"`
	TimeoutMs   int32  `json:"timeout_ms,omitempty"`
}

type ScreenshotElementArgs struct {
	Handle uintptr `json:"handle"`
}

type OCRWindowArgs struct {
	Handle   uintptr `json:"handle"`
	Language string  `json:"language,omitempty"`
}

type OcrActiveWindowArgs struct {
	Language string `json:"language,omitempty"`
}

type HoverArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type WaitForTextArgs struct {
	Text         string `json:"text"`
	TimeoutMs    int32  `json:"timeout_ms,omitempty"`
	Language     string `json:"language,omitempty"`
	MaxScrolls   *int32 `json:"max_scrolls,omitempty"`
	ScrollClicks *int32 `json:"scroll_clicks,omitempty"`
	ScrollDown   *bool  `json:"scroll_down,omitempty"`
}

type SelectAllAndTypeArgs struct {
	Text string `json:"text"`
	VerifyArgs
}

type ClickMenuItemArgs struct {
	WindowTitle  string `json:"window_title,omitempty"`
	Handle       uintptr `json:"handle,omitempty"`
	MenuItemText string `json:"menu_item_text"`
	Language     string `json:"language,omitempty"`
	VerifyArgs
}

type SetKeyboardLayoutArgs struct {
	Language string `json:"language"`
}

type OpenExplorerArgs struct {
	Path string `json:"path,omitempty"`
}

type FindImageArgs struct {
	TemplateB64 string  `json:"template_b64"`
	ScreenB64   string  `json:"screen_b64,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
}

type SetAudioDeviceArgs struct {
	DeviceID string `json:"device_id"`
}

type DisplayModesArgs struct {
	DeviceName string `json:"device_name"`
}

type RecordScreenArgs struct {
	DurationMs int32 `json:"duration_ms,omitempty"`
	IntervalMs int32 `json:"interval_ms,omitempty"`
}

type UIAFindArgs struct {
	Name         string `json:"name,omitempty"`
	AutomationID string `json:"automation_id,omitempty"`
	ControlType  string `json:"control_type,omitempty"`
}

type UIAGetTextArgs struct {
	Name         string `json:"name,omitempty"`
	AutomationID string `json:"automation_id,omitempty"`
}

type UIASetTextArgs struct {
	Name         string `json:"name,omitempty"`
	AutomationID string `json:"automation_id,omitempty"`
	Value        string `json:"value"`
}

type UIAInvokeArgs struct {
	Name         string `json:"name,omitempty"`
	AutomationID string `json:"automation_id,omitempty"`
}

func uiaFindHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAFindArgs) (*mcp.CallToolResult, any, error) {
	opts := actions.UIAFindOpts{
		Name:         args.Name,
		AutomationID: args.AutomationID,
		ControlType:  args.ControlType,
	}
	elements, err := actions.UIAFindElement(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_find: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"elements": elements}, nil
}

func uiaGetTextHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAGetTextArgs) (*mcp.CallToolResult, any, error) {
	text, err := actions.UIAGetText(args.Name, args.AutomationID)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_get_text: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]string{"text": text}, nil
}

func uiaSetTextHandler(ctx context.Context, req *mcp.CallToolRequest, args UIASetTextArgs) (*mcp.CallToolResult, any, error) {
	if args.Value == "" {
		return nil, nil, fmt.Errorf("uia_set_text: value is required")
	}
	if err := actions.UIASetText(args.Name, args.AutomationID, args.Value); err != nil {
		return nil, nil, fmt.Errorf("uia_set_text: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func uiaInvokeHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAInvokeArgs) (*mcp.CallToolResult, any, error) {
	success, err := actions.UIAInvoke(args.Name, args.AutomationID)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_invoke: %w", err)
	}
	if !success {
		return nil, nil, fmt.Errorf("uia_invoke: element not found or not invocable")
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

type UIAElementAtPointArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func uiaElementAtPointHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAElementAtPointArgs) (*mcp.CallToolResult, any, error) {
	el, err := actions.UIAElementFromPoint(args.X, args.Y)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_element_at_point: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"element": el}, nil
}

type UIAGetAllElementsArgs struct {
	Handle     uintptr `json:"handle"`
	MaxResults int     `json:"max_results,omitempty"`
}

func uiaGetAllElementsHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAGetAllElementsArgs) (*mcp.CallToolResult, any, error) {
	if args.Handle == 0 {
		return nil, nil, fmt.Errorf("uia_get_all_elements: handle is required")
	}
	elements, err := actions.UIAGetAllElements(args.Handle, args.MaxResults)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_get_all_elements: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"elements": elements, "count": len(elements)}, nil
}

type WaitForUIElementArgs struct {
	Handle      uintptr `json:"handle"`
	Name        string  `json:"name,omitempty"`
	ControlType string  `json:"control_type,omitempty"`
	TimeoutMs   int     `json:"timeout_ms,omitempty"`
}

func waitForUIElementHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitForUIElementArgs) (*mcp.CallToolResult, any, error) {
	if args.Name == "" && args.ControlType == "" {
		return nil, nil, fmt.Errorf("wait_for_ui_element: name or control_type required")
	}
	if args.TimeoutMs <= 0 {
		args.TimeoutMs = 10000
	}
	el, err := actions.WaitForUIElement(args.Handle, args.Name, args.ControlType, args.TimeoutMs)
	if err != nil {
		return nil, nil, fmt.Errorf("wait_for_ui_element: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"element": el}, nil
}

// chainInputSchema builds a JSON schema for the chain tool's input.
// We provide this manually because ChainStep contains recursive types
// (IfConfig → []ChainStep → ChainStep → *IfConfig) which cause the
// jsonschema-go auto-generator to error with "cycle detected".
// The manual schema also avoids `items: true` and `type: ["null", "array"]`
// that Gemini MCP schema validator rejects.
func chainInputSchema() *jsonschema.Schema {
	subStep := func() *jsonschema.Schema {
		return &jsonschema.Schema{Type: "object"}
	}
	subStepArray := func() *jsonschema.Schema {
		return &jsonschema.Schema{
			Type:  "array",
			Items: subStep(),
		}
	}

	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"steps": {
				Type: "array",
				Items: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"type":         {Type: "string"},
						"capture":      {Type: "string"},
						"tool":         {Type: "string"},
						"args":         {Type: "object", AdditionalProperties: &jsonschema.Schema{}},
						"wait_ms":      {Type: "integer"},
						"focus_window": {Type: "string"},
						"poll": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"every_ms":     {Type: "integer"},
								"timeout_ms":   {Type: "integer"},
								"ocr_contains": {Type: "string"},
							},
						},
						"if": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"ocr_contains": {Type: "string"},
								"then":         subStepArray(),
								"else":         subStepArray(),
							},
						},
						"loop": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"times": {Type: "integer"},
								"steps": subStepArray(),
							},
						},
						"verify": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"expected": {
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"text":     {Type: "string"},
										"not_text": {Type: "string"},
										"change":   {Type: "boolean"},
										"wait_ms":  {Type: "integer"},
									},
								},
								"retries": {Type: "integer"},
								"wait_ms": {Type: "integer"},
							},
						},
						"verify_ui": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"element_name": {Type: "string"},
								"control_type": {Type: "string"},
								"handle":       {Type: "integer"},
								"timeout_ms":   {Type: "integer"},
								"not_exists":   {Type: "boolean"},
							},
						},
						"if_uia": {
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"element_name": {Type: "string"},
								"control_type": {Type: "string"},
								"handle":       {Type: "integer"},
								"then":         subStepArray(),
								"else":         subStepArray(),
							},
						},
					},
				},
			},
			"timeout_ms": {Type: "integer"},
			"on_error":   {Type: "string"},
		},
	}
}

type ChainArgs struct {
	Steps          []actions.ChainStep `json:"steps"`
	TimeoutMs      int                 `json:"timeout_ms,omitempty"`
	OnError        string              `json:"on_error,omitempty"`
	AutoVerifyFocus bool               `json:"auto_verify_focus,omitempty"`
}

func chainHandler(ctx context.Context, req *mcp.CallToolRequest, args ChainArgs) (*mcp.CallToolResult, any, error) {
	chainReq := actions.ChainRequest{
		Steps:           args.Steps,
		TimeoutMs:       args.TimeoutMs,
		OnError:         args.OnError,
		AutoVerifyFocus: args.AutoVerifyFocus,
	}
	result, err := actions.ExecuteChain(chainReq)
	if err != nil {
		return nil, nil, fmt.Errorf("chain: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func screenshotHandler(ctx context.Context, req *mcp.CallToolRequest, args ScreenshotArgs) (*mcp.CallToolResult, any, error) {
	var b64 string
	var err error

	if args.W != nil && args.H != nil {
		x := int32(0)
		y := int32(0)
		if args.X != nil {
			x = *args.X
		}
		if args.Y != nil {
			y = *args.Y
		}
		b64, err = actions.CaptureRegion(x, y, *args.W, *args.H)
	} else {
		b64, err = actions.CaptureScreen()
	}

	if err != nil {
		return nil, nil, fmt.Errorf("screenshot failed: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: b64},
		},
	}, nil, nil
}

func clickHandler(ctx context.Context, req *mcp.CallToolRequest, args ClickArgs) (*mcp.CallToolResult, any, error) {
	rx, ry, rw, rh := actions.SmartRegionAround(args.X, args.Y, 400)
	if err := preVerifyCheck(args.PreExpected, &rx, &ry, &rw, &rh); err != nil {
		return nil, nil, err
	}
	if err := actions.Click(actions.ClickInput{
		X: args.X, Y: args.Y, Button: args.Button, Clicks: args.Clicks,
	}); err != nil {
		return nil, nil, fmt.Errorf("click failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatClick,
		fmt.Sprintf("click at (%d,%d)", args.X, args.Y))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func moveMouseHandler(ctx context.Context, req *mcp.CallToolRequest, args MoveMouseArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.MoveMouse(args.X, args.Y); err != nil {
		return nil, nil, fmt.Errorf("move_mouse failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func scrollHandler(ctx context.Context, req *mcp.CallToolRequest, args ScrollArgs) (*mcp.CallToolResult, any, error) {
	if err := preVerifyCheck(args.PreExpected, nil, nil, nil, nil); err != nil {
		return nil, nil, err
	}
	clicks := args.Clicks
	if args.Direction == "down" {
		clicks = -clicks
	}
	if err := actions.Scroll(clicks, args.Horizontal); err != nil {
		return nil, nil, fmt.Errorf("scroll failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "scroll")
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, nil, nil, nil, nil))
	}
	return verifiedResult(nil, vr)
}

func keyPressHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyPressArgs) (*mcp.CallToolResult, any, error) {
	if err := preVerifyCheck(args.PreExpected, nil, nil, nil, nil); err != nil {
		return nil, nil, err
	}
	if err := actions.KeyPress(args.Keys); err != nil {
		return nil, nil, fmt.Errorf("key_press failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key press")
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, nil, nil, nil, nil))
	}
	return verifiedResult(nil, vr)
}

func keyDownHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyEventArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyDown(args.Key); err != nil {
		return nil, nil, fmt.Errorf("key_down failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key down")
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func keyUpHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyEventArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyUp(args.Key); err != nil {
		return nil, nil, fmt.Errorf("key_up failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key up")
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func keyloggerStartHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyloggerStartArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.StartKeylogger(); err != nil {
		return nil, nil, fmt.Errorf("keylogger_start failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func keyloggerStopHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyloggerStartArgs) (*mcp.CallToolResult, any, error) {
	steps, meta, err := actions.StopKeylogger()
	if err != nil {
		return nil, nil, fmt.Errorf("keylogger_stop failed: %w", err)
	}
	out := map[string]any{
		"meta":  meta,
		"steps": steps,
	}
	jsonBytes, _ := json.MarshalIndent(out, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
	}, nil, nil
}

func keyloggerStatusHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyloggerStatusArgs) (*mcp.CallToolResult, any, error) {
	active, count, dur := actions.KeyloggerStatus()
	status := "inactive"
	if active {
		status = fmt.Sprintf("active - %d events in %s", count, dur)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: status}},
	}, nil, nil
}

func recordReplicateHandler(ctx context.Context, req *mcp.CallToolRequest, args RecordReplicateArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.RecordAndReplicate(args.DurationSecs, args.DelayMs, args.Slowdown, args.Loop)
	if err != nil {
		return nil, nil, fmt.Errorf("record_and_replicate: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func typeHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeArgs) (*mcp.CallToolResult, any, error) {
	cx, cy, cerr := actions.GetCursorPosition()
	if cerr != nil {
		cx, cy = 0, 0
	}
	rx, ry, rw, rh := actions.SmartRegionAround(cx, cy, 400)
	if err := preVerifyCheck(args.PreExpected, &rx, &ry, &rw, &rh); err != nil {
		return nil, nil, err
	}
	if err := actions.TypeText(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "type text")
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func screenSizeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	w, h := actions.ScreenSize()
	return &mcp.CallToolResult{}, ScreenSizeResult{Width: w, Height: h}, nil
}

func cursorPosHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	x, y, err := actions.GetCursorPosition()
	if err != nil {
		return nil, nil, fmt.Errorf("get_cursor_position failed: %w", err)
	}
	return &mcp.CallToolResult{}, CursorPosResult{X: x, Y: y}, nil
}

func dragHandler(ctx context.Context, req *mcp.CallToolRequest, args DragArgs) (*mcp.CallToolResult, any, error) {
	rx, ry, rw, rh := actions.SmartRegionAround(args.ToX, args.ToY, 400)
	if err := preVerifyCheck(args.PreExpected, &rx, &ry, &rw, &rh); err != nil {
		return nil, nil, err
	}
	if err := actions.Drag(args.FromX, args.FromY, args.ToX, args.ToY); err != nil {
		return nil, nil, fmt.Errorf("drag failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral,
		fmt.Sprintf("drag from (%d,%d) to (%d,%d)", args.FromX, args.FromY, args.ToX, args.ToY))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func listWindowsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	windows, err := actions.ListWindows()
	if err != nil {
		return nil, nil, fmt.Errorf("list_windows failed: %w", err)
	}
	return &mcp.CallToolResult{}, ListWindowsResult{Windows: windows}, nil
}

func focusWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args FocusWindowArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FocusWindow(args.Handle); err != nil {
		return nil, nil, fmt.Errorf("focus_window failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	vol, err := actions.GetVolume()
	if err != nil {
		return nil, nil, fmt.Errorf("get_volume failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]uint32{"volume": vol}, nil
}

func setVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, args SetVolumeArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetVolume(args.Percent); err != nil {
		return nil, nil, fmt.Errorf("set_volume failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func setMuteHandler(ctx context.Context, req *mcp.CallToolRequest, args MuteArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetMute(args.Mute); err != nil {
		return nil, nil, fmt.Errorf("set_mute failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getSystemInfoHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetSystemInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_system_info failed: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func getActiveWindowHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetActiveWindowInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_active_window failed: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func getClipboardHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	text, err := actions.GetClipboardText()
	if err != nil {
		return nil, nil, fmt.Errorf("get_clipboard failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]string{"text": text}, nil
}

func setClipboardHandler(ctx context.Context, req *mcp.CallToolRequest, args SetClipboardArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetClipboardText(args.Text); err != nil {
		return nil, nil, fmt.Errorf("set_clipboard failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func openURLHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenURLArgs) (*mcp.CallToolResult, any, error) {
	if err := preVerifyCheck(args.PreExpected, nil, nil, nil, nil); err != nil {
		return nil, nil, err
	}
	if err := actions.OpenURL(args.URL); err != nil {
		return nil, nil, fmt.Errorf("open_url failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("open url %s", args.URL))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, nil, nil, nil, nil))
	}
	return verifiedResult(nil, vr)
}

func waitHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitArgs) (*mcp.CallToolResult, any, error) {
	actions.Wait(args.Ms)
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getPixelColorHandler(ctx context.Context, req *mcp.CallToolRequest, args PixelColorArgs) (*mcp.CallToolResult, any, error) {
	color, err := actions.GetPixelColor(args.X, args.Y)
	if err != nil {
		return nil, nil, fmt.Errorf("get_pixel_color failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]string{"color": color}, nil
}

func listProcessesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	processes, err := actions.ListProcesses()
	if err != nil {
		return nil, nil, fmt.Errorf("list_processes failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"processes": processes}, nil
}

func launchAppHandler(ctx context.Context, req *mcp.CallToolRequest, args LaunchAppArgs) (*mcp.CallToolResult, any, error) {
	if err := preVerifyCheck(args.PreExpected, nil, nil, nil, nil); err != nil {
		return nil, nil, err
	}
	if err := actions.LaunchApp(args.Path); err != nil {
		return nil, nil, fmt.Errorf("launch_app failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatLaunch,
		fmt.Sprintf("launch app %s", args.Path))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, nil, nil, nil, nil))
	}
	return verifiedResult(nil, vr)
}

func killProcessHandler(ctx context.Context, req *mcp.CallToolRequest, args KillProcessArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KillProcess(args.PID); err != nil {
		return nil, nil, fmt.Errorf("kill_process failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func listDisplaysHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	displays, err := actions.ListDisplays()
	if err != nil {
		return nil, nil, fmt.Errorf("list_displays failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"displays": displays}, nil
}

func displayModesHandler(ctx context.Context, req *mcp.CallToolRequest, args DisplayModesArgs) (*mcp.CallToolResult, any, error) {
	modes, err := actions.GetDisplayModes(args.DeviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("get_display_modes failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"modes": modes}, nil
}

func getBatteryHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	status, err := actions.GetBattery()
	if err != nil {
		return nil, nil, fmt.Errorf("get_battery failed: %w", err)
	}
	return &mcp.CallToolResult{}, status, nil
}

func moveWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args MoveWindowArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.MoveWindowByHandle(args.Handle, args.X, args.Y, args.Width, args.Height); err != nil {
		return nil, nil, fmt.Errorf("move_window failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func minimizeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.MinimizeWindow(args.Handle)
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func maximizeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.MaximizeWindow(args.Handle)
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func restoreWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.RestoreWindow(args.Handle)
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func closeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.CloseWindow(args.Handle); err != nil {
		return nil, nil, fmt.Errorf("close_window failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getWindowStateHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	state, err := actions.GetWindowState(args.Handle)
	if err != nil {
		return nil, nil, fmt.Errorf("get_window_state failed: %w", err)
	}
	return &mcp.CallToolResult{}, state, nil
}

func findWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args FindWindowArgs) (*mcp.CallToolResult, any, error) {
	hwnd := actions.FindWindowByTitle(args.Title)
	return &mcp.CallToolResult{}, map[string]any{"handle": hwnd, "found": hwnd != 0}, nil
}

func waitForWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitForWindowArgs) (*mcp.CallToolResult, any, error) {
	timeout := args.TimeoutMs
	if timeout == 0 {
		timeout = 5000
	}
	hwnd, err := actions.WaitForWindow(args.Title, timeout)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "timeout"}},
		}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{}, map[string]any{"handle": hwnd, "found": true}, nil
}

func showNotificationHandler(ctx context.Context, req *mcp.CallToolRequest, args NotificationArgs) (*mcp.CallToolResult, any, error) {
	actions.ShowNotification(args.Title, args.Message)
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func lockWorkstationHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.LockWorkstation()
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func ocrHandler(ctx context.Context, req *mcp.CallToolRequest, args OCRArgs) (*mcp.CallToolResult, any, error) {
	var result *actions.OCRResult
	var err error

	if args.W != nil && args.H != nil {
		x := int32(0)
		y := int32(0)
		if args.X != nil { x = *args.X }
		if args.Y != nil { y = *args.Y }
		result, err = actions.OCRRegion(x, y, *args.W, *args.H, args.Language)
	} else {
		result, err = actions.OCRScreen(args.Language)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("ocr failed: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

type OcrLanguagesArgs struct{}

func ocrLanguagesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	languages, err := actions.OcrLanguages()
	if err != nil {
		return nil, nil, fmt.Errorf("ocr_languages: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"languages": languages}, nil
}

func ocrWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args OCRWindowArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.OCRWindow(args.Handle, args.Language)
	if err != nil {
		return nil, nil, fmt.Errorf("ocr_window: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func ocrActiveWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args OcrActiveWindowArgs) (*mcp.CallToolResult, any, error) {
	hwnd := actions.ForegroundWindowHandle()
	if hwnd == 0 {
		return nil, nil, fmt.Errorf("ocr_active_window: no foreground window")
	}
	result, err := actions.OCRWindow(hwnd, args.Language)
	if err != nil {
		return nil, nil, fmt.Errorf("ocr_active_window: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func getBrightnessHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	b, err := actions.GetBrightness()
	if err != nil {
		return nil, nil, fmt.Errorf("get_brightness failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]int{"brightness": b}, nil
}

func setBrightnessHandler(ctx context.Context, req *mcp.CallToolRequest, args BrightnessArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetBrightness(args.Percent); err != nil {
		return nil, nil, fmt.Errorf("set_brightness failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getIdleTimeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	d, err := actions.GetIdleTime()
	if err != nil {
		return nil, nil, fmt.Errorf("get_idle_time failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"idle_ms": d.Milliseconds()}, nil
}

func getNetworkInfoHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetNetworkInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_network_info failed: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func pingHandler(ctx context.Context, req *mcp.CallToolRequest, args PingArgs) (*mcp.CallToolResult, any, error) {
	reachable, err := actions.PingHost(args.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("ping failed: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]bool{"reachable": reachable}, nil
}

func findTextAndClickHandler(ctx context.Context, req *mcp.CallToolRequest, args FindTextAndClickArgs) (*mcp.CallToolResult, any, error) {
	if err := preVerifyCheck(args.PreExpected, args.X, args.Y, args.W, args.H); err != nil {
		return nil, nil, err
	}
	opts := actions.FindTextOpts{
		Text: args.Text, Language: args.Language,
		RegionX: args.X, RegionY: args.Y, RegionW: args.W, RegionH: args.H,
		WindowTitle: args.WindowTitle,
	}
	if args.MaxScrolls != nil {
		opts.MaxScrolls = *args.MaxScrolls
	}
	if args.ScrollClicks != nil {
		opts.ScrollClicks = *args.ScrollClicks
	}
	if args.ScrollDown != nil {
		opts.ScrollDown = *args.ScrollDown
	}
	if args.SkipMemory != nil {
		opts.SkipMemory = *args.SkipMemory
	}
	if args.SkipSystemFind != nil {
		opts.SkipSystemFind = *args.SkipSystemFind
	}
	cx, cy, err := actions.FindTextAndClick(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("find_text_and_click: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatClick,
		fmt.Sprintf("find text and click: %s", args.Text))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		rx, ry, rw, rh := actions.SmartRegionAround(cx, cy, 400)
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func typeAndSubmitHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeAndSubmitArgs) (*mcp.CallToolResult, any, error) {
	cx, cy, cerr := actions.GetCursorPosition()
	if cerr != nil {
		cx, cy = 0, 0
	}
	rx, ry, rw, rh := actions.SmartRegionAround(cx, cy, 400)
	if err := preVerifyCheck(args.PreExpected, &rx, &ry, &rw, &rh); err != nil {
		return nil, nil, err
	}
	if err := actions.TypeAndSubmit(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type_and_submit: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "type and submit")
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func launchAndWaitHandler(ctx context.Context, req *mcp.CallToolRequest, args LaunchAndWaitArgs) (*mcp.CallToolResult, any, error) {
	timeout := args.TimeoutMs
	if timeout == 0 { timeout = 10000 }
	hwnd, err := actions.LaunchAndWait(args.Path, args.WindowTitle, timeout)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "timeout"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{}, map[string]any{"handle": hwnd, "found": true}, nil
}

func screenshotElementHandler(ctx context.Context, req *mcp.CallToolRequest, args ScreenshotElementArgs) (*mcp.CallToolResult, any, error) {
	b64, err := actions.ScreenshotElement(args.Handle)
	if err != nil {
		return nil, nil, fmt.Errorf("screenshot_element: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: b64}}}, nil, nil
}

func hoverHandler(ctx context.Context, req *mcp.CallToolRequest, args HoverArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.Hover(args.X, args.Y); err != nil {
		return nil, nil, fmt.Errorf("hover: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral,
		fmt.Sprintf("hover at (%d,%d)", args.X, args.Y))
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func waitForTextHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitForTextArgs) (*mcp.CallToolResult, any, error) {
	timeout := args.TimeoutMs
	if timeout == 0 { timeout = 10000 }
	var maxScrolls, scrollClicks int32
	var scrollDown bool
	if args.MaxScrolls != nil { maxScrolls = *args.MaxScrolls }
	if args.ScrollClicks != nil { scrollClicks = *args.ScrollClicks }
	if args.ScrollDown != nil { scrollDown = *args.ScrollDown }
	result, err := actions.WaitForTextScroll(args.Text, timeout, args.Language, maxScrolls, scrollClicks, scrollDown)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "not_found"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{}, result, nil
}

func selectAllAndTypeHandler(ctx context.Context, req *mcp.CallToolRequest, args SelectAllAndTypeArgs) (*mcp.CallToolResult, any, error) {
	cx, cy, cerr := actions.GetCursorPosition()
	if cerr != nil {
		cx, cy = 0, 0
	}
	rx, ry, rw, rh := actions.SmartRegionAround(cx, cy, 400)
	if err := preVerifyCheck(args.PreExpected, &rx, &ry, &rw, &rh); err != nil {
		return nil, nil, err
	}
	if err := actions.SelectAllAndType(args.Text); err != nil {
		return nil, nil, fmt.Errorf("select_all_and_type: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "select all and type")
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, &rx, &ry, &rw, &rh))
	}
	return verifiedResult(nil, vr)
}

func clickMenuItemHandler(ctx context.Context, req *mcp.CallToolRequest, args ClickMenuItemArgs) (*mcp.CallToolResult, any, error) {
	var rx, ry, rw, rh *int32
	var hwnd uintptr
	if args.Handle != 0 {
		hwnd = args.Handle
	} else {
		hwnd = actions.FindWindowByTitle(args.WindowTitle)
	}
	if hwnd != 0 {
		if state, err := actions.GetWindowState(hwnd); err == nil && state.Rect != nil {
			rx, ry, rw, rh = &state.Rect.Left, &state.Rect.Top, &state.Rect.Width, &state.Rect.Height
		}
	}
	if err := preVerifyCheck(args.PreExpected, rx, ry, rw, rh); err != nil {
		return nil, nil, err
	}
	var err error
	if args.Handle != 0 {
		err = actions.ClickMenuItem(args.Handle, args.MenuItemText, args.Language)
	} else {
		err = actions.ClickMenuItemByTitle(args.WindowTitle, args.MenuItemText, args.Language)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("click_menu_item: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatClick,
		fmt.Sprintf("click menu item: %s", args.MenuItemText))
	var vr *actions.VerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.VerifyAction(verifyCfg(args.Expected, rx, ry, rw, rh))
	}
	return verifiedResult(nil, vr)
}

func getUptimeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	d, err := actions.GetUptime()
	if err != nil {
		return nil, nil, fmt.Errorf("get_uptime: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"uptime_ms": d.Milliseconds()}, nil
}

func shutdownHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Shutdown(); err != nil {
		return nil, nil, fmt.Errorf("shutdown: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func restartHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Restart(); err != nil {
		return nil, nil, fmt.Errorf("restart: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func sleepHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Sleep(); err != nil {
		return nil, nil, fmt.Errorf("sleep: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func hibernateHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Hibernate(); err != nil {
		return nil, nil, fmt.Errorf("hibernate: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getKeyboardLayoutHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetKeyboardLayout()
	if err != nil {
		return nil, nil, fmt.Errorf("get_keyboard_layout: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func setKeyboardLayoutHandler(ctx context.Context, req *mcp.CallToolRequest, args SetKeyboardLayoutArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetKeyboardLayout(args.Language); err != nil {
		return nil, nil, fmt.Errorf("set_keyboard_layout: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getDiskUsageHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	disks, err := actions.GetDiskUsage()
	if err != nil {
		return nil, nil, fmt.Errorf("get_disk_usage: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"disks": disks}, nil
}

func openFileExplorerHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenExplorerArgs) (*mcp.CallToolResult, any, error) {
	path := args.Path
	if path == "" { path = "C:\\" }
	if err := actions.OpenFileExplorer(path); err != nil {
		return nil, nil, fmt.Errorf("open_file_explorer: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func openFileLocationHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenExplorerArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.OpenFileLocation(args.Path); err != nil {
		return nil, nil, fmt.Errorf("open_file_location: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func findImageHandler(ctx context.Context, req *mcp.CallToolRequest, args FindImageArgs) (*mcp.CallToolResult, any, error) {
	var result *actions.MatchResult
	var err error
	if args.ScreenB64 != "" {
		result, err = actions.FindImageInRegion(args.ScreenB64, args.TemplateB64, args.Threshold)
	} else {
		result, err = actions.FindImageOnScreen(args.TemplateB64, args.Threshold)
	}
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "no_match"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{}, result, nil
}

func findAllImagesHandler(ctx context.Context, req *mcp.CallToolRequest, args FindImageArgs) (*mcp.CallToolResult, any, error) {
	results, err := actions.FindAllImages(args.ScreenB64, args.TemplateB64, args.Threshold)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, map[string]any{"found": false, "matches": []actions.MatchResult{}}, nil
	}
	return &mcp.CallToolResult{}, map[string]any{"found": len(results) > 0, "matches": results}, nil
}

func listAudioDevicesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	devices, err := actions.ListAudioDevices()
	if err != nil {
		return nil, nil, fmt.Errorf("list_audio_devices: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"devices": devices}, nil
}

func setDefaultAudioDeviceHandler(ctx context.Context, req *mcp.CallToolRequest, args SetAudioDeviceArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetDefaultAudioDevice(args.DeviceID); err != nil {
		return nil, nil, fmt.Errorf("set_default_audio_device: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func recordScreenHandler(ctx context.Context, req *mcp.CallToolRequest, args RecordScreenArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.RecordScreen(args.DurationMs, args.IntervalMs)
	if err != nil {
		return nil, nil, fmt.Errorf("record_screen: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

type LayoutValidateArgs struct {
	Elements       []actions.LayoutElement `json:"elements"`
	WindowTitle    string                  `json:"window_title,omitempty"`
	WindowHandle   uintptr                 `json:"window_handle,omitempty"`
	DriftTolerance int32                   `json:"drift_tolerance,omitempty"`
	Language       string                  `json:"language,omitempty"`
}

func layoutValidateHandler(ctx context.Context, req *mcp.CallToolRequest, args LayoutValidateArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.ValidateLayout(actions.LayoutValidateInput{
		Elements:       args.Elements,
		WindowTitle:    args.WindowTitle,
		WindowHandle:   args.WindowHandle,
		DriftTolerance: args.DriftTolerance,
		Language:       args.Language,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("layout_validate: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func getScreenDPIHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	dpi, err := actions.GetScreenDPI()
	if err != nil {
		return nil, nil, fmt.Errorf("get_screen_dpi: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"monitors": dpi}, nil
}

type DPIPointArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func getDPIPointHandler(ctx context.Context, req *mcp.CallToolRequest, args DPIPointArgs) (*mcp.CallToolResult, any, error) {
	dpi, err := actions.GetDPIScaleForPoint(args.X, args.Y)
	if err != nil {
		return nil, nil, fmt.Errorf("get_dpi_for_point: %w", err)
	}
	scale := (dpi * 100) / 96
	return &mcp.CallToolResult{}, map[string]any{
		"dpi":           dpi,
		"scale_percent": scale,
		"x":             args.X,
		"y":             args.Y,
	}, nil
}

type FocusWindowByTitleArgs struct {
	Title string `json:"title"`
}

func focusWindowByTitleHandler(ctx context.Context, req *mcp.CallToolRequest, args FocusWindowByTitleArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FocusWindowByTitle(args.Title); err != nil {
		return nil, nil, fmt.Errorf("focus_window_by_title: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

type BrowserArgs struct {
	Browser string `json:"browser"`
}

type BrowserNavigateArgs struct {
	Browser string `json:"browser"`
	URL     string `json:"url"`
}

type BrowserSearchArgs struct {
	Browser string `json:"browser"`
	Query   string `json:"query"`
}

type ExplorerPathArgs struct {
	Path string `json:"path"`
}

func browserFocusURLBarHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserFocusURLBar(args.Browser); err != nil {
		return nil, nil, fmt.Errorf("browser_focus_url_bar: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func browserNewTabHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserNewTab(args.Browser); err != nil {
		return nil, nil, fmt.Errorf("browser_new_tab: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func browserNavigateHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserNavigateArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserNavigate(args.Browser, args.URL); err != nil {
		return nil, nil, fmt.Errorf("browser_navigate: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("navigate to %s", args.URL))
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func browserSearchHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserSearchArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserSearch(args.Browser, args.Query); err != nil {
		return nil, nil, fmt.Errorf("browser_search: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("search %s", args.Query))
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func explorerFocusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.ExplorerFocus(); err != nil {
		return nil, nil, fmt.Errorf("explorer_focus: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func explorerOpenPathHandler(ctx context.Context, req *mcp.CallToolRequest, args ExplorerPathArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.ExplorerOpenPath(args.Path); err != nil {
		return nil, nil, fmt.Errorf("explorer_open_path: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

type ONNXDetectArgs struct {
	ImageB64     string  `json:"image_b64,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	IOUThreshold float64 `json:"iou_threshold,omitempty"`
}

func onnxStatusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	status := actions.ONNXStatus()
	return &mcp.CallToolResult{}, status, nil
}

func onnxDetectHandler(ctx context.Context, req *mcp.CallToolRequest, args ONNXDetectArgs) (*mcp.CallToolResult, any, error) {
	// If no image provided, capture full screen
	imgB64 := args.ImageB64
	if imgB64 == "" {
		var err error
		imgB64, err = actions.CaptureScreen()
		if err != nil {
			return nil, nil, fmt.Errorf("onnx_detect screenshot: %w", err)
		}
	}

	result, err := actions.ONNXDetect(actions.DetectionInput{
		ImageB64:     imgB64,
		Threshold:    args.Threshold,
		IOUThreshold: args.IOUThreshold,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("onnx_detect: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func onnxDownloadHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	result, err := actions.ONNXDownload()
	if err != nil {
		return nil, nil, fmt.Errorf("onnx_download: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

type ONNXWatchStartArgs struct {
	IntervalSeconds int `json:"interval_seconds"`
}

func onnxWatchStartHandler(ctx context.Context, req *mcp.CallToolRequest, args ONNXWatchStartArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.StartWatcher(args.IntervalSeconds); err != nil {
		return nil, nil, fmt.Errorf("onnx_watch_start: %w", err)
	}
	return &mcp.CallToolResult{}, actions.GetWatcherStatus(), nil
}

func onnxWatchStopHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.StopWatcher()
	return &mcp.CallToolResult{}, actions.GetWatcherStatus(), nil
}

func onnxWatchStatusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{}, actions.GetWatcherStatus(), nil
}

func onnxWatchCacheHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{}, map[string]any{"detections": actions.GetCachedDetections()}, nil
}

type TemplateStoreArgs struct {
	ElementKey        string   `json:"element_key"`
	Scope             string   `json:"scope,omitempty"`
	CenterX           int32    `json:"center_x"`
	CenterY           int32    `json:"center_y"`
	CropSize          int      `json:"crop_size,omitempty"`
	WindowTitle       string   `json:"window_title,omitempty"`
	SignatureKeywords []string `json:"signature_keywords,omitempty"`
}

type TemplateFindArgs struct {
	ElementKey string  `json:"element_key"`
	Scope      string  `json:"scope,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
}

type TemplateListArgs struct {
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type TemplateForgetArgs struct {
	ElementKey string `json:"element_key"`
	Scope      string `json:"scope,omitempty"`
}

func templateStoreHandler(ctx context.Context, req *mcp.CallToolRequest, args TemplateStoreArgs) (*mcp.CallToolResult, any, error) {
	info, err := actions.TemplateStore(actions.TemplateStoreInput{
		ElementKey:        args.ElementKey,
		Scope:             args.Scope,
		CenterX:           args.CenterX,
		CenterY:           args.CenterY,
		CropSize:          args.CropSize,
		WindowTitle:       args.WindowTitle,
		SignatureKeywords: args.SignatureKeywords,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("template_store: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func templateFindHandler(ctx context.Context, req *mcp.CallToolRequest, args TemplateFindArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.TemplateFind(actions.TemplateFindInput{
		ElementKey: args.ElementKey,
		Scope:      args.Scope,
		Threshold:  args.Threshold,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("template_find: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

func templateListHandler(ctx context.Context, req *mcp.CallToolRequest, args TemplateListArgs) (*mcp.CallToolResult, any, error) {
	results, err := actions.TemplateList(actions.TemplateListInput{
		Scope: args.Scope,
		Limit: args.Limit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("template_list: %w", err)
	}
	if results == nil {
		results = []actions.TemplateInfo{}
	}
	return &mcp.CallToolResult{}, map[string]any{"templates": results}, nil
}

func templateForgetHandler(ctx context.Context, req *mcp.CallToolRequest, args TemplateForgetArgs) (*mcp.CallToolResult, any, error) {
	deleted, err := actions.TemplateForget(args.ElementKey, args.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("template_forget: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"deleted": deleted}, nil
}

type MemorySetArgs struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
	TTL   int    `json:"ttl,omitempty"`
}

type MemoryGetArgs struct {
	Key   string `json:"key"`
	Scope string `json:"scope,omitempty"`
}

type MemorySearchArgs struct {
	Query string `json:"query"`
	Scope string `json:"scope,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type MemoryListArgs struct {
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type MemoryForgetArgs struct {
	Key   string `json:"key,omitempty"`
	Scope string `json:"scope,omitempty"`
	Tags  string `json:"tags,omitempty"`
}

type TrainingSaveSampleArgs struct {
	Category    string `json:"category"`
	TaskPrompt  string `json:"task_prompt"`
	WindowTitle string `json:"window_title,omitempty"`
}

type TrainingListSamplesArgs struct {
	Category   string `json:"category,omitempty"`
	UnusedOnly bool   `json:"unused_only,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type TrainingMarkUsedArgs struct {
	ID int64 `json:"id"`
}

type FindUIElementArgs struct {
	Label       string `json:"label"`
	WindowTitle string `json:"window_title,omitempty"`
	UseOCR      bool   `json:"use_ocr,omitempty"`
}

func memorySetHandler(ctx context.Context, req *mcp.CallToolRequest, args MemorySetArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.MemorySet(actions.MemorySetInput{
		Key:   args.Key,
		Value: args.Value,
		Scope: args.Scope,
		Tags:  args.Tags,
		TTL:   args.TTL,
	}); err != nil {
		return nil, nil, fmt.Errorf("memory_set: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func memoryGetHandler(ctx context.Context, req *mcp.CallToolRequest, args MemoryGetArgs) (*mcp.CallToolResult, any, error) {
	fact, err := actions.MemoryGet(args.Key, args.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("memory_get: %w", err)
	}
	if fact == nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "not_found"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{}, fact, nil
}

func memorySearchHandler(ctx context.Context, req *mcp.CallToolRequest, args MemorySearchArgs) (*mcp.CallToolResult, any, error) {
	results, err := actions.MemorySearch(actions.MemorySearchInput{
		Query: args.Query,
		Scope: args.Scope,
		Limit: args.Limit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("memory_search: %w", err)
	}
	if results == nil {
		results = []actions.MemorySearchResult{}
	}
	return &mcp.CallToolResult{}, map[string]any{"results": results}, nil
}

func memoryListHandler(ctx context.Context, req *mcp.CallToolRequest, args MemoryListArgs) (*mcp.CallToolResult, any, error) {
	results, err := actions.MemoryList(actions.MemoryListInput{
		Scope: args.Scope,
		Tags:  args.Tags,
		Limit: args.Limit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("memory_list: %w", err)
	}
	if results == nil {
		results = []actions.MemorySearchResult{}
	}
	return &mcp.CallToolResult{}, map[string]any{"results": results}, nil
}

func memoryForgetHandler(ctx context.Context, req *mcp.CallToolRequest, args MemoryForgetArgs) (*mcp.CallToolResult, any, error) {
	deleted, err := actions.MemoryForget(actions.MemoryForgetInput{
		Key:   args.Key,
		Scope: args.Scope,
		Tags:  args.Tags,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("memory_forget: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"deleted": deleted}, nil
}

func trainingSaveSampleHandler(ctx context.Context, req *mcp.CallToolRequest, args TrainingSaveSampleArgs) (*mcp.CallToolResult, any, error) {
	b64, err := actions.CaptureScreen()
	if err != nil {
		return nil, nil, fmt.Errorf("training_save_sample screenshot: %w", err)
	}
	winTitle := args.WindowTitle
	if winTitle == "" {
		if info, err := actions.GetActiveWindowInfo(); err == nil {
			winTitle = info.Title
		}
	}
	sample, err := actions.SaveTrainingSample(actions.SaveTrainingSampleInput{
		Source:      actions.TrainingSourceRaw,
		Category:    args.Category,
		TaskPrompt:  args.TaskPrompt,
		ImageB64:    b64,
		WindowTitle: winTitle,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("training_save_sample: %w", err)
	}
	return &mcp.CallToolResult{}, sample, nil
}

func trainingListSamplesHandler(ctx context.Context, req *mcp.CallToolRequest, args TrainingListSamplesArgs) (*mcp.CallToolResult, any, error) {
	samples, err := actions.TrainingSampleList(actions.TrainingListInput{
		Category:   args.Category,
		UnusedOnly: args.UnusedOnly,
		Limit:      args.Limit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("training_list_samples: %w", err)
	}
	if samples == nil {
		samples = []actions.TrainingSampleMeta{}
	}
	return &mcp.CallToolResult{}, map[string]any{"samples": samples}, nil
}

func trainingStatsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	stats, err := actions.TrainingStatsReport()
	if err != nil {
		return nil, nil, fmt.Errorf("training_stats: %w", err)
	}
	return &mcp.CallToolResult{}, stats, nil
}

func trainingMarkUsedHandler(ctx context.Context, req *mcp.CallToolRequest, args TrainingMarkUsedArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TrainingMarkUsed(args.ID); err != nil {
		return nil, nil, fmt.Errorf("training_mark_used: %w", err)
	}
	return &mcp.CallToolResult{}, map[string]any{"marked": args.ID}, nil
}

func findUIElementHandler(ctx context.Context, req *mcp.CallToolRequest, args FindUIElementArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.FindUIElement(actions.FindUIElementInput{
		Label:       args.Label,
		WindowTitle: args.WindowTitle,
		UseOCR:      args.UseOCR,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("find_ui_element: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

type PriorStatsArgs struct {
	MinCount int `json:"min_count,omitempty"`
}

func priorStatsHandler(ctx context.Context, req *mcp.CallToolRequest, args PriorStatsArgs) (*mcp.CallToolResult, any, error) {
	stats, err := actions.GetPriorStats(args.MinCount)
	if err != nil {
		return nil, nil, fmt.Errorf("priors_stats: %w", err)
	}
	return &mcp.CallToolResult{}, stats, nil
}

type ExportYoloDatasetArgs struct {
	OutputDir string `json:"output_dir"`
	MinSignal int    `json:"min_signal,omitempty"`
}

func exportYoloDatasetHandler(ctx context.Context, req *mcp.CallToolRequest, args ExportYoloDatasetArgs) (*mcp.CallToolResult, any, error) {
	outDir := args.OutputDir
	if outDir == "" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return nil, nil, fmt.Errorf("output_dir is required and APPDATA is not set")
		}
		outDir = filepath.Join(appData, "go-mcp-computer-use", "yolo_dataset")
	}
	stats, err := actions.ExportYoloDataset(outDir, args.MinSignal)
	if err != nil {
		return nil, nil, fmt.Errorf("export_yolo_dataset: %w", err)
	}
	return &mcp.CallToolResult{}, stats, nil
}

type TrainingCleanupNoiseArgs struct {
	MaxAgeHours int  `json:"max_age_hours,omitempty"`
	DryRun      bool `json:"dry_run,omitempty"`
}

func trainingCleanupNoiseHandler(ctx context.Context, req *mcp.CallToolRequest, args TrainingCleanupNoiseArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.TrainingCleanupNoise(args.MaxAgeHours, args.DryRun)
	if err != nil {
		return nil, nil, fmt.Errorf("training_cleanup_noise: %w", err)
	}
	return &mcp.CallToolResult{}, result, nil
}

type SetConfigArgs struct {
	TrainingEnabled      *bool    `json:"training_enabled,omitempty"`
	PriorAdjustment      *bool    `json:"prior_adjustment,omitempty"`
	VerifyBounds         *bool    `json:"verify_bounds,omitempty"`
	LogLevel             string   `json:"log_level,omitempty"`
	WatcherEnabled       *bool    `json:"watcher_enabled,omitempty"`
	WatcherIntervalSecs  *int     `json:"watcher_interval_seconds,omitempty"`
	ToolDenylist         []string `json:"tool_denylist,omitempty"`
	RetentionDays        *int     `json:"retention_days,omitempty"`
	ChainAbortEnabled    *bool    `json:"chain_abort_enabled,omitempty"`
	ChainAbortKeys       string   `json:"chain_abort_keys,omitempty"`
	ChainAbortPollMs     *int     `json:"chain_abort_poll_ms,omitempty"`
	WindowLockEnabled    *bool    `json:"window_lock_enabled,omitempty"`
	WindowLockAutoFocus  *bool    `json:"window_lock_auto_focus,omitempty"`
	LogFileEnabled       *bool    `json:"log_file_enabled,omitempty"`
	LogFileMaxSizeMB     *int     `json:"log_file_max_size_mb,omitempty"`
	LogFileRetention     *int     `json:"log_file_retention,omitempty"`
	DashboardEnabled     *bool    `json:"dashboard_enabled,omitempty"`
}

type GetLogsArgs struct {
	Level        string `json:"level,omitempty"`
	Lines        int    `json:"lines,omitempty"`
	Search       string `json:"search,omitempty"`
	SinceMinutes int    `json:"since_minutes,omitempty"`
}

type ReportIssueArgs struct {
	Title    string   `json:"title"`
	Body     string   `json:"body,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	AutoLogs *bool    `json:"auto_logs,omitempty"`
}

type ImageDiffArgs struct {
	Before        string `json:"before"`
	After         string `json:"after"`
	Threshold     *int   `json:"threshold,omitempty"`
	GenerateImage *bool  `json:"generate_image,omitempty"`
}

type ListDirectoryArgs struct {
	Path string `json:"path"`
}

type ReadFileArgs struct {
	Path     string `json:"path"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

type WriteFileArgs struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Overwrite *bool  `json:"overwrite,omitempty"`
	VerifyArgs
}

type FindFilesArgs struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
}

type CopyFileArgs struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	VerifyArgs
}

type MoveFileArgs struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	VerifyArgs
}

type DeleteFileArgs struct {
	Path string `json:"path"`
	VerifyArgs
}

type CreateDirectoryArgs struct {
	Path string `json:"path"`
	VerifyArgs
}

type GetFileInfoArgs struct {
	Path string `json:"path"`
}

type SetWorkingDirectoryArgs struct {
	Path string `json:"path"`
}

type DataLogQueryArgs struct {
	Table   string `json:"table"`
	Source  string `json:"source,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Success *bool  `json:"success,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type DataLogExportArgs struct {
	SessionID string `json:"session_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func datalogQueryHandler(ctx context.Context, req *mcp.CallToolRequest, args DataLogQueryArgs) (*mcp.CallToolResult, any, error) {
	rows, err := actions.QueryDataLog(actions.DataLogQuery{
		Table: args.Table, Source: args.Source, Tool: args.Tool,
		Success: args.Success, Limit: args.Limit, Offset: args.Offset,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("datalog_query: %w", err)
	}
	rowsJSON, _ := json.Marshal(rows)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(rowsJSON)}},
	}, map[string]any{"count": len(rows), "rows": rows}, nil
}

func datalogExportHandler(ctx context.Context, req *mcp.CallToolRequest, args DataLogExportArgs) (*mcp.CallToolResult, any, error) {
	out, err := actions.ExportTrainingData(args.SessionID, args.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("datalog_export: %w", err)
	}
	pairsJSON, _ := json.Marshal(out.Pairs)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(pairsJSON)}},
	}, out, nil
}

func datalogStatusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	stats, err := actions.DataLogStatsReport()
	if err != nil {
		return nil, nil, fmt.Errorf("datalog_status: %w", err)
	}
	return &mcp.CallToolResult{}, stats, nil
}

func datalogStatsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	stats, err := actions.DataLogStatsReport()
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(stats, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "datalog://stats",
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

func datalogCommandsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	rows, err := actions.QueryDataLog(actions.DataLogQuery{Table: "commands", Limit: 20})
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(rows, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "datalog://commands",
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

func datalogOCRResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	rows, err := actions.QueryDataLog(actions.DataLogQuery{Table: "ocr", Limit: 10})
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(rows, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "datalog://ocr",
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

func datalogPairsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	rows, err := actions.QueryDataLog(actions.DataLogQuery{Table: "pairs", Limit: 20})
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(rows, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "datalog://pairs",
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

func adaptiveAnalysisResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	analysis := actions.Adaptive.Analyze()
	b, _ := json.MarshalIndent(analysis, "", "  ")
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      "adaptive://analysis",
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

type AgentAnalyzeArgs struct{}
type AgentSuggestArgs struct {
	OCRText string `json:"ocr_text"`
	Limit   int    `json:"limit,omitempty"`
}
type AgentTrainArgs struct{}

type ChainPredictArgs struct {
	OCRText     string `json:"ocr_text"`
	WindowTitle string `json:"window_title,omitempty"`
}

func chainPredictHandler(ctx context.Context, req *mcp.CallToolRequest, args ChainPredictArgs) (*mcp.CallToolResult, any, error) {
	if args.OCRText == "" {
		return nil, nil, fmt.Errorf("ocr_text is required")
	}
	var result *actions.SequencePredictionResult
	if args.WindowTitle != "" {
		result = actions.Adaptive.PredictSequenceActionsWithWindow(args.OCRText, args.WindowTitle)
	} else {
		result = actions.Adaptive.PredictSequenceActions(args.OCRText)
	}
	if result == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "no predictions available — ML model not trained yet. Run agent_train first."}},
		}, map[string]any{"primary": nil, "next": []any{}}, nil
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, result, nil
}

func agentAnalyzeHandler(ctx context.Context, req *mcp.CallToolRequest, _ AgentAnalyzeArgs) (*mcp.CallToolResult, any, error) {
	analysis := actions.Adaptive.Analyze()
	b, _ := json.MarshalIndent(analysis, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, analysis, nil
}

func agentSuggestHandler(ctx context.Context, req *mcp.CallToolRequest, args AgentSuggestArgs) (*mcp.CallToolResult, any, error) {
	if args.OCRText == "" {
		return nil, nil, fmt.Errorf("ocr_text is required")
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 5
	}
	predictions := actions.Adaptive.PredictActions(args.OCRText, limit)
	if predictions == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "no predictions available - try training the model first with agent_train"}},
		}, map[string]any{"predictions": []any{}}, nil
	}
	b, _ := json.MarshalIndent(predictions, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, map[string]any{"predictions": predictions}, nil
}

func agentTrainHandler(ctx context.Context, req *mcp.CallToolRequest, _ AgentTrainArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.Adaptive.TrainFromDatalog(); err != nil {
		return nil, nil, fmt.Errorf("agent_train: %w", err)
	}
	analysis := actions.Adaptive.Analyze()
	b, _ := json.MarshalIndent(analysis, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, analysis, nil
}

type TaskBeginArgs struct {
	Description string `json:"description"`
}

type TaskEndArgs struct {
	Summary string `json:"summary,omitempty"`
	Success bool   `json:"success,omitempty"`
}

func taskBeginHandler(_ context.Context, _ *mcp.CallToolRequest, args TaskBeginArgs) (*mcp.CallToolResult, any, error) {
	info, err := actions.TaskBegin(actions.TaskInput{Description: args.Description})
	if err != nil {
		return nil, nil, fmt.Errorf("task_begin: %w", err)
	}
	return &mcp.CallToolResult{}, info, nil
}

func taskEndHandler(_ context.Context, _ *mcp.CallToolRequest, args TaskEndArgs) (*mcp.CallToolResult, any, error) {
	info, err := actions.TaskEnd(actions.TaskEndInput{Summary: args.Summary, Success: args.Success})
	if err != nil {
		return nil, nil, fmt.Errorf("task_end: %w", err)
	}
	b, _ := json.MarshalIndent(info.Insights, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, info, nil
}

func introspectionAnalyzeHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	tasks, err := actions.IntrospectionAnalyze()
	if err != nil {
		return nil, nil, fmt.Errorf("introspection_analyze: %w", err)
	}
	b, _ := json.MarshalIndent(tasks, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, map[string]any{"tasks": tasks, "count": len(tasks)}, nil
}

func systemFindStatsHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	lastUsed, count := actions.SystemFindStats()
	return &mcp.CallToolResult{}, map[string]any{
		"last_used": lastUsed,
		"count":     count,
	}, nil
}

func taskIsActiveHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	active := actions.TaskIsActive()
	return &mcp.CallToolResult{}, map[string]any{"active": active}, nil
}

func bridgeDebugHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info := actions.BridgeDebugInfo()
	return &mcp.CallToolResult{}, info, nil
}

func setConfigHandler(ctx context.Context, req *mcp.CallToolRequest, args SetConfigArgs) (*mcp.CallToolResult, any, error) {
	cfg := actions.ActiveConfig
	if cfg == nil {
		cfg = config.Default()
		actions.ActiveConfig = cfg
	}

	changed := false

	if args.TrainingEnabled != nil {
		val := *args.TrainingEnabled
		if cfg.TrainingEnabled != val {
			cfg.TrainingEnabled = val
			changed = true
		}
	}
	if args.PriorAdjustment != nil {
		val := *args.PriorAdjustment
		if cfg.PriorAdjustment != val {
			cfg.PriorAdjustment = val
			changed = true
		}
	}
	if args.VerifyBounds != nil {
		val := *args.VerifyBounds
		if cfg.VerifyBounds != val {
			cfg.VerifyBounds = val
			changed = true
		}
	}
	if args.LogLevel != "" {
		if cfg.LogLevel != args.LogLevel {
			cfg.LogLevel = args.LogLevel
			changed = true
		}
	}

	if args.WatcherEnabled != nil {
		val := *args.WatcherEnabled
		wantRunning := val
		status := actions.GetWatcherStatus()
		if wantRunning && !status.Running {
			interval := cfg.WatcherIntervalSecs
			if interval < 1 {
				interval = 5
			}
			if err := actions.StartWatcher(interval); err != nil {
				return nil, nil, fmt.Errorf("start watcher: %w", err)
			}
			changed = true
			cfg.WatcherAutoStart = true
		} else if !wantRunning && status.Running {
			actions.StopWatcher()
			changed = true
			cfg.WatcherAutoStart = false
		}
	}

	if args.WatcherIntervalSecs != nil {
		val := *args.WatcherIntervalSecs
		if val < 1 {
			val = 5
		}
		if cfg.WatcherIntervalSecs != val {
			cfg.WatcherIntervalSecs = val
			changed = true
			status := actions.GetWatcherStatus()
			if status.Running {
				actions.StopWatcher()
				if err := actions.StartWatcher(val); err != nil {
					slog.Warn("restart watcher with new interval failed", "error", err)
				}
			}
		}
	}

	if args.ToolDenylist != nil {
		cfg.ToolDenylist = args.ToolDenylist
		changed = true
	}

	if args.RetentionDays != nil {
		val := *args.RetentionDays
		if val < 0 {
			val = 0
		}
		if cfg.RetentionDays != val {
			cfg.RetentionDays = val
			changed = true
		}
	}

	if args.ChainAbortEnabled != nil {
		val := *args.ChainAbortEnabled
		if cfg.ChainAbortEnabled != val {
			cfg.ChainAbortEnabled = val
			changed = true
			if val {
				actions.InitAbortFromConfig(cfg.ChainAbortEnabled, cfg.ChainAbortKeys, cfg.ChainAbortPollMs)
			} else {
				actions.StopAbortPoller()
			}
		}
	}
	if args.ChainAbortKeys != "" {
		if cfg.ChainAbortKeys != args.ChainAbortKeys {
			cfg.ChainAbortKeys = args.ChainAbortKeys
			changed = true
			if cfg.ChainAbortEnabled {
				actions.InitAbortFromConfig(cfg.ChainAbortEnabled, cfg.ChainAbortKeys, cfg.ChainAbortPollMs)
			}
		}
	}
	if args.ChainAbortPollMs != nil {
		val := *args.ChainAbortPollMs
		if val < 10 {
			val = 10
		}
		if cfg.ChainAbortPollMs != val {
			cfg.ChainAbortPollMs = val
			changed = true
			if cfg.ChainAbortEnabled {
				actions.InitAbortFromConfig(cfg.ChainAbortEnabled, cfg.ChainAbortKeys, cfg.ChainAbortPollMs)
			}
		}
	}
	if args.WindowLockEnabled != nil {
		val := *args.WindowLockEnabled
		if cfg.WindowLockEnabled != val {
			cfg.WindowLockEnabled = val
			changed = true
		}
	}
	if args.WindowLockAutoFocus != nil {
		val := *args.WindowLockAutoFocus
		if cfg.WindowLockAutoFocus != val {
			cfg.WindowLockAutoFocus = val
			changed = true
		}
	}
	if args.LogFileEnabled != nil {
		val := *args.LogFileEnabled
		if cfg.LogFileEnabled != val {
			cfg.LogFileEnabled = val
			changed = true
		}
	}
	if args.LogFileMaxSizeMB != nil {
		val := *args.LogFileMaxSizeMB
		if val < 1 {
			val = 10
		}
		if cfg.LogFileMaxSizeMB != val {
			cfg.LogFileMaxSizeMB = val
			changed = true
		}
	}
	if args.LogFileRetention != nil {
		val := *args.LogFileRetention
		if val < 1 {
			val = 7
		}
		if cfg.LogFileRetention != val {
			cfg.LogFileRetention = val
			changed = true
		}
	}
	if args.DashboardEnabled != nil {
		val := *args.DashboardEnabled
		if cfg.DashboardEnabled != val {
			cfg.DashboardEnabled = val
			changed = true
		}
	}

	if changed {
		slog.Info("config updated", "training_enabled", cfg.TrainingEnabled,
			"prior_adjustment", cfg.PriorAdjustment, "verify_bounds", cfg.VerifyBounds,
			"log_level", cfg.LogLevel, "tool_denylist", cfg.ToolDenylist,
			"retention_days", cfg.RetentionDays, "log_file_enabled", cfg.LogFileEnabled)
		if err := cfg.Save(); err != nil {
			slog.Warn("failed to save config", "error", err)
		}
	}

	watcherStatus := actions.GetWatcherStatus()

	return &mcp.CallToolResult{}, map[string]any{
		"training_enabled":        cfg.TrainingEnabled,
		"prior_adjustment":        cfg.PriorAdjustment,
		"verify_bounds":           cfg.VerifyBounds,
		"log_level":               cfg.LogLevel,
		"watcher_running":         watcherStatus.Running,
		"watcher_interval_secs":   cfg.WatcherIntervalSecs,
		"tool_denylist":           cfg.ToolDenylist,
		"retention_days":          cfg.RetentionDays,
		"chain_abort_enabled":     cfg.ChainAbortEnabled,
		"chain_abort_keys":        cfg.ChainAbortKeys,
		"chain_abort_poll_ms":     cfg.ChainAbortPollMs,
		"window_lock_enabled":     cfg.WindowLockEnabled,
		"window_lock_auto_focus":  cfg.WindowLockAutoFocus,
		"log_file_enabled":        cfg.LogFileEnabled,
		"log_file_max_size_mb":    cfg.LogFileMaxSizeMB,
		"log_file_retention":      cfg.LogFileRetention,
		"dashboard_enabled":       cfg.DashboardEnabled,
		"saved":                   changed,
	}, nil
}

func getLogsHandler(ctx context.Context, req *mcp.CallToolRequest, args GetLogsArgs) (*mcp.CallToolResult, any, error) {
	logPath := logging.LogPath()
	if logPath == "" {
		return nil, nil, fmt.Errorf("log path not available (APPDATA not set)")
	}

	lines := args.Lines
	if lines < 1 {
		lines = 50
	}

	entries, totalLines, truncated := logging.ReadLogs(logPath, lines, args.Level, args.Search, args.SinceMinutes)

	return &mcp.CallToolResult{}, map[string]any{
		"entries":    entries,
		"count":      len(entries),
		"total_lines": totalLines,
		"truncated":  truncated,
		"log_path":   logPath,
	}, nil
}

func reportIssueHandler(ctx context.Context, req *mcp.CallToolRequest, args ReportIssueArgs) (*mcp.CallToolResult, any, error) {
	logPath := logging.LogPath()

	autoLogs := true
	if args.AutoLogs != nil {
		autoLogs = *args.AutoLogs
	}

	issue, err := logging.GenerateIssue(args.Title, args.Body, logPath, autoLogs)
	if err != nil {
		return nil, nil, fmt.Errorf("generate issue: %w", err)
	}

	return &mcp.CallToolResult{}, map[string]any{
		"title":          issue.Title,
		"body":           issue.Body,
		"issue_url":      issue.IssueURL,
		"log_lines_included": issue.LogLines,
		"submitted":      issue.IssueURL != "",
	}, nil
}

func imageDiffHandler(ctx context.Context, req *mcp.CallToolRequest, args ImageDiffArgs) (*mcp.CallToolResult, any, error) {
	if args.Before == "" || args.After == "" {
		return nil, nil, fmt.Errorf("image_diff: both 'before' and 'after' base64 PNG images are required")
	}

	opts := actions.ImageDiffOpts{
		GenerateImage: args.GenerateImage != nil && *args.GenerateImage,
	}
	if args.Threshold != nil {
		opts.Threshold = *args.Threshold
	}

	result, err := actions.ImageDiff(args.Before, args.After, opts)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{}, result, nil
}

func listDirectoryHandler(ctx context.Context, req *mcp.CallToolRequest, args ListDirectoryArgs) (*mcp.CallToolResult, any, error) {
	entries, err := actions.ListDirectory(args.Path)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{}, map[string]any{"entries": entries}, nil
}

func readFileHandler(ctx context.Context, req *mcp.CallToolRequest, args ReadFileArgs) (*mcp.CallToolResult, any, error) {
	page := args.Page
	if page <= 0 {
		page = 1
	}
	pageSize := args.PageSize
	if pageSize <= 0 {
		pageSize = actions.DefaultPageSize
	}
	result, err := actions.ReadFile(args.Path, page, pageSize)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result.Content}},
	}, result, nil
}

func writeFileHandler(ctx context.Context, req *mcp.CallToolRequest, args WriteFileArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FilePreCheck(args.PreExpected, args.Path); err != nil {
		return nil, nil, err
	}
	overwrite := false
	if args.Overwrite != nil {
		overwrite = *args.Overwrite
	}
	result, err := actions.WriteFile(args.Path, args.Content, overwrite)
	if err != nil {
		return nil, nil, err
	}
	var vr *actions.FileVerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.FilePostVerify(args.Expected, result.Path)
	}
	if vr != nil && !vr.Passed {
		return nil, nil, fmt.Errorf("verify: %s", vr.Reason)
	}
	return &mcp.CallToolResult{}, result, nil
}

func findFilesHandler(ctx context.Context, req *mcp.CallToolRequest, args FindFilesArgs) (*mcp.CallToolResult, any, error) {
	matches, err := actions.FindFiles(args.Path, args.Pattern)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{}, map[string]any{"matches": matches}, nil
}

func copyFileHandler(ctx context.Context, req *mcp.CallToolRequest, args CopyFileArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FilePreCheck(args.PreExpected, args.Source); err != nil {
		return nil, nil, err
	}
	if err := actions.CopyFile(args.Source, args.Destination); err != nil {
		return nil, nil, err
	}
	var vr *actions.FileVerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.FilePostVerify(args.Expected, args.Destination)
	}
	if vr != nil && !vr.Passed {
		return nil, nil, fmt.Errorf("verify: %s", vr.Reason)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func moveFileHandler(ctx context.Context, req *mcp.CallToolRequest, args MoveFileArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FilePreCheck(args.PreExpected, args.Source); err != nil {
		return nil, nil, err
	}
	if err := actions.MoveFile(args.Source, args.Destination); err != nil {
		return nil, nil, err
	}
	var vr *actions.FileVerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.FilePostVerify(args.Expected, args.Destination)
	}
	if vr != nil && !vr.Passed {
		return nil, nil, fmt.Errorf("verify: %s", vr.Reason)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func deleteFileHandler(ctx context.Context, req *mcp.CallToolRequest, args DeleteFileArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FilePreCheck(args.PreExpected, args.Path); err != nil {
		return nil, nil, err
	}
	result, err := actions.DeleteFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	var vr *actions.FileVerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.FilePostVerify(args.Expected, args.Path)
	}
	if vr != nil && !vr.Passed {
		return nil, nil, fmt.Errorf("verify: %s", vr.Reason)
	}
	return &mcp.CallToolResult{}, result, nil
}

func createDirectoryHandler(ctx context.Context, req *mcp.CallToolRequest, args CreateDirectoryArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FilePreCheck(args.PreExpected, args.Path); err != nil {
		return nil, nil, err
	}
	if err := actions.CreateDirectory(args.Path); err != nil {
		return nil, nil, err
	}
	var vr *actions.FileVerifyResult
	if shouldVerify(args.AutoVerify, args.Expected) {
		vr = actions.FilePostVerify(args.Expected, args.Path)
	}
	if vr != nil && !vr.Passed {
		return nil, nil, fmt.Errorf("verify: %s", vr.Reason)
	}
	return &mcp.CallToolResult{}, map[string]any{"ok": true}, nil
}

func getFileInfoHandler(ctx context.Context, req *mcp.CallToolRequest, args GetFileInfoArgs) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetFileInfo(args.Path)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{}, info, nil
}

func setWorkingDirectoryHandler(ctx context.Context, req *mcp.CallToolRequest, args SetWorkingDirectoryArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetWorkingDirectory(args.Path); err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{}, map[string]string{"working_directory": actions.GetWorkingDirectory()}, nil
}

func getWorkingDirectoryHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{}, map[string]string{"working_directory": actions.GetWorkingDirectory()}, nil
}

func resetStateHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.Adaptive.Reset()
	actions.ResetBridgeState()
	slog.Info("server state reset — adaptive engine + bridge buffer cleared")
	return &mcp.CallToolResult{}, map[string]string{"status": "reset complete", "cleared": "adaptive engine, bridge buffer"}, nil
}

func dismissAllMenusHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	// OCR before to capture what menus are open
	before, err := actions.OCRScreen("")
	if err != nil {
		return nil, nil, fmt.Errorf("dismiss_all_menus ocr before: %w", err)
	}
	menuWords := []string{"delete", "copy", "paste", "cut", "undo", "redo", "select all", "inspect", "back", "forward", "refresh", "save as", "print", "more tools", "open link", "copy link", "bookmark"}
	var detectedMenus []string
	lower := strings.ToLower(before.Text)
	for _, mw := range menuWords {
		if strings.Contains(lower, mw) {
			detectedMenus = append(detectedMenus, mw)
		}
	}
	// Press Escape to dismiss
	if err := actions.KeyPress([]string{"ESC"}); err != nil {
		return nil, nil, fmt.Errorf("dismiss_all_menus escape: %w", err)
	}
	actions.Wait(300)
	// OCR after to verify
	after, err := actions.OCRScreen("")
	if err != nil {
		return nil, nil, fmt.Errorf("dismiss_all_menus ocr after: %w", err)
	}
	afterLower := strings.ToLower(after.Text)
	menusStillOpen := false
	for _, mw := range menuWords {
		if strings.Contains(afterLower, mw) {
			menusStillOpen = true
			break
		}
	}
	result := map[string]any{
		"esc_pressed":    true,
		"menus_before":   detectedMenus,
		"menus_still_open": menusStillOpen,
	}
	if menusStillOpen {
		result["warning"] = "menus may still be open after Escape"
	}
	return &mcp.CallToolResult{}, result, nil
}

func chainAbortHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	aborted := actions.IsChainAborted()
	return &mcp.CallToolResult{}, map[string]any{
		"aborted": aborted,
	}, nil
}

type SetWindowLockArgs struct {
	Handle uintptr `json:"handle"`
}

func setWindowLockHandler(_ context.Context, _ *mcp.CallToolRequest, args SetWindowLockArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetWindowLock(args.Handle); err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{}, map[string]any{
		"locked": true,
		"handle": args.Handle,
		"title":  actions.GetWindowLockTitle(),
		"pid":    actions.GetWindowLockPID(),
	}, nil
}

func clearWindowLockHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.ClearWindowLock()
	return &mcp.CallToolResult{}, map[string]any{
		"cleared": true,
	}, nil
}

func New(version string) *mcp.Server {
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("config load failed, using defaults", "error", err)
		cfg = config.Default()
	}
	actions.ActiveConfig = cfg

	if cfg.LogFileEnabled {
		logDir := logging.LogDir()
		if logDir != "" {
			maxMB := cfg.LogFileMaxSizeMB
			if maxMB < 1 {
				maxMB = 10
			}
			keep := cfg.LogFileRetention
			if keep < 1 {
				keep = 7
			}
			if fh, err := logging.Init(logDir, maxMB, keep, cfg.LogLevelSlog()); err != nil {
				slog.Warn("file logging init failed", "error", err)
			} else {
				slog.Info("file logging enabled", "path", logging.LogPath(), "max_mb", maxMB, "retention", keep)
				defer fh.Close()
			}
		}
	}

	level := new(slog.LevelVar)
	level.Set(slog.Level(cfg.LogLevelSlog()))
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("starting go-mcp-computer-use", "version", version, "tools", 145, "tools_doc", "docs/tools.md")

	if cfg.UIAWarmup {
		go func() {
			slog.Debug("uia warmup starting (async)")
			if err := actions.WarmupUIA(); err != nil {
				slog.Warn("uia warmup failed (UIA tools may be slow on first call)", "error", err)
			} else {
				slog.Info("uia warmup complete")
			}
		}()
	} else {
		slog.Info("uia warmup disabled by config")
	}

	if cfg.WatcherAutoStart && cfg.TrainingEnabled {
		go func() {
			secs := cfg.WatcherIntervalSecs
			if secs < 1 {
				secs = 5
			}
			slog.Info("auto-starting background watcher", "interval_seconds", secs)
			if err := actions.StartWatcher(secs); err != nil {
				slog.Warn("watcher auto-start failed", "error", err)
			}
		}()
	} else if cfg.WatcherAutoStart && !cfg.TrainingEnabled {
		slog.Info("watcher auto-start skipped: training disabled")
	}

	if cfg.RetentionDays > 0 && cfg.TrainingEnabled {
		actions.StartRetentionPruner(cfg.RetentionDays)
		slog.Info("retention pruner started", "retention_days", cfg.RetentionDays)
	}

	if cfg.ChainAbortEnabled {
		pollMs := cfg.ChainAbortPollMs
		if pollMs < 10 {
			pollMs = 50
		}
		actions.InitAbortFromConfig(cfg.ChainAbortEnabled, cfg.ChainAbortKeys, pollMs)
		slog.Info("chain abort hotkey enabled", "keys", cfg.ChainAbortKeys, "poll_ms", pollMs)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "go-mcp-computer-use",
		Version: version,
	}, nil)

	actions.EnsureAdaptive()

	server.AddResource(&mcp.Resource{
		URI:         "datalog://stats",
		Name:        "datalog-stats",
		Description: "Current datalog row counts",
		MIMEType:    "application/json",
	}, datalogStatsResource)
	server.AddResource(&mcp.Resource{
		URI:         "datalog://commands",
		Name:        "datalog-commands",
		Description: "Recent command log entries",
		MIMEType:    "application/json",
	}, datalogCommandsResource)
	server.AddResource(&mcp.Resource{
		URI:         "datalog://ocr",
		Name:        "datalog-ocr",
		Description: "Recent OCR snapshot entries",
		MIMEType:    "application/json",
	}, datalogOCRResource)
	server.AddResource(&mcp.Resource{
		URI:         "datalog://pairs",
		Name:        "datalog-pairs",
		Description: "Recent training pair entries",
		MIMEType:    "application/json",
	}, datalogPairsResource)
	server.AddResource(&mcp.Resource{
		URI:         "adaptive://analysis",
		Name:        "adaptive-analysis",
		Description: "Adaptive engine analysis with timing stats, success rates, and learned sequences",
		MIMEType:    "application/json",
	}, adaptiveAnalysisResource)

	addToolClean(server, &mcp.Tool{
		Name:        "screenshot",
		Description: "Capture the screen or a region. If w/h omitted, captures full screen.",
	}, screenshotHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "click",
		Description: "Click at screen coordinates x,y. Button: left/right/middle. Clicks: 1 or 2.",
	}, clickHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "move_mouse",
		Description: "Move mouse cursor to x,y.",
	}, moveMouseHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "scroll",
		Description: "Scroll the mouse wheel. Positive clicks = up, negative = down. Set horizontal=true for horizontal scroll.",
	}, scrollHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "key_press",
		Description: "Press key combination. Example: [\"Ctrl\", \"C\"] for copy.",
	}, keyPressHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "key_down",
		Description: "Hold a key down (does not release it). Use key_up to release. Example: \"W\"",
	}, keyDownHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "key_up",
		Description: "Release a key that was held down with key_down. Example: \"W\"",
	}, keyUpHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "keylogger_start",
		Description: "Start recording keyboard and mouse input for replay",
	}, keyloggerStartHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "keylogger_stop",
		Description: "Stop recording and return recorded sequence as chain steps",
	}, keyloggerStopHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "keylogger_status",
		Description: "Check if keylogger is active and event count",
	}, keyloggerStatusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "record_and_replicate",
		Description: "Record mouse and keyboard events for N seconds, then automatically replay them as a chain. Supports slowdown factor and loop count for repeated execution.",
	}, recordReplicateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "type",
		Description: "Type text at the currently focused element.",
	}, typeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_screen_size",
		Description: "Get the screen dimensions.",
	}, screenSizeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_cursor_position",
		Description: "Get the current mouse cursor position.",
	}, cursorPosHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "drag",
		Description: "Drag mouse from (from_x, from_y) to (to_x, to_y).",
	}, dragHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "list_windows",
		Description: "List all visible windows with their handles, titles, PIDs, and bounding rect (x, y, width, height). This includes background windows (minimized, behind other windows). Cross-reference with list_displays monitor positions to determine which screen each window occupies. Returns every top-level window — use get_window_state on a handle to check if it is actually foreground, minimized, or behind other windows.",
	}, listWindowsHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "focus_window",
		Description: "Bring a window to the foreground by handle. Also restores the window if it is minimized. ALWAYS call this before typing, clicking, or OCR-ing a window that is not the current foreground window. Use get_window_state to check if a window is already foreground before calling.",
	}, focusWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "focus_window_by_title",
		Description: "Find a window by title and focus it, clicking its title bar to ensure activation. Useful before keyboard input in chain steps.",
	}, focusWindowByTitleHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "browser_focus_url_bar",
		Description: "Focus a browser window's URL bar. Supports Firefox (Ctrl+T), Chrome/Edge (Ctrl+L), and other browsers. Provide browser name (firefox, chrome, edge, brave, opera) or window title substring.",
	}, browserFocusURLBarHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "browser_new_tab",
		Description: "Open a new tab in a browser window. Uses Ctrl+T for all browsers.",
	}, browserNewTabHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "browser_navigate",
		Description: "Open a new tab in a browser and navigate to a URL.",
	}, browserNavigateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "browser_search",
		Description: "Open a new tab in a browser and perform a search query.",
	}, browserSearchHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "explorer_focus",
		Description: "Focus an existing File Explorer window.",
	}, explorerFocusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "explorer_open_path",
		Description: "Open a File Explorer window at the specified path. Reuses existing window when possible.",
	}, explorerOpenPathHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_volume",
		Description: "Get the current system volume level (0-100).",
	}, getVolumeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the system volume level (0-100).",
	}, setVolumeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_mute",
		Description: "Mute or unmute the system audio.",
	}, setMuteHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_system_info",
		Description: "Get system information (hostname, OS, RAM).",
	}, getSystemInfoHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_active_window",
		Description: "Get the current foreground window info (handle, title, PID, and bounding rect).",
	}, getActiveWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_clipboard",
		Description: "Read text from the clipboard.",
	}, getClipboardHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_clipboard",
		Description: "Write text to the clipboard.",
	}, setClipboardHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "open_url",
		Description: "Open a URL in the default browser.",
	}, openURLHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "wait",
		Description: "Wait for N milliseconds before the next action.",
	}, waitHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_pixel_color",
		Description: "Get the hex color at screen coordinates x,y.",
	}, getPixelColorHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "list_processes",
		Description: "List all running processes with PID, name, and thread count.",
	}, listProcessesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "launch_app",
		Description: "Launch an application by path or shell command.",
	}, launchAppHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "kill_process",
		Description: "Terminate a process by PID.",
	}, killProcessHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "list_displays",
		Description: "List all monitors with resolution and position.",
	}, listDisplaysHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_display_modes",
		Description: "Get all available display modes (resolution, refresh rate, color depth) for a monitor by device name.",
	}, displayModesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_battery",
		Description: "Get battery status (percentage, charging, on battery).",
	}, getBatteryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "move_window",
		Description: "Move and resize a window by handle.",
	}, moveWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "minimize_window",
		Description: "Minimize a window by handle.",
	}, minimizeWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "maximize_window",
		Description: "Maximize a window by handle.",
	}, maximizeWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "restore_window",
		Description: "Restore a minimized or maximized window by handle.",
	}, restoreWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "close_window",
		Description: "Close a window by handle.",
	}, closeWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_window_state",
		Description: "Get window state: visible (WS_VISIBLE flag — NOT obscured-by-other-windows), minimized, maximized, fullscreen, foreground (is this the active/focused window?), z_order (0=topmost, higher=deeper behind other windows), and bounding rect. A window can be visible with high z_order but completely hidden behind other windows. Use this before interacting: if NOT foreground, call focus_window first (z_order will become 0); if visible vs other windows check, compare z_order values between windows.",
	}, getWindowStateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_window",
		Description: "Find a window handle by title.",
	}, findWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "wait_for_window",
		Description: "Wait for a window with the given title to appear. Returns handle or timeout.",
	}, waitForWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "show_notification",
		Description: "Show a Windows notification message box.",
	}, showNotificationHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "lock_workstation",
		Description: "Lock the workstation.",
	}, lockWorkstationHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "ocr",
		Description: "Extract text from screen using Windows OCR. Supports full screen, specific monitor (screen=N where N is the display index from list_displays, 0-based), or region (x,y,w,h).",
	}, ocrHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_brightness",
		Description: "Get the current display brightness level (0-100).",
	}, getBrightnessHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_brightness",
		Description: "Set the display brightness level (0-100).",
	}, setBrightnessHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_idle_time",
		Description: "Get the system idle time (time since last user input) in milliseconds.",
	}, getIdleTimeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_network_info",
		Description: "Get network information: hostname, IP addresses, DNS servers, default gateway.",
	}, getNetworkInfoHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "ping",
		Description: "Ping a host to check network reachability.",
	}, pingHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_text_and_click",
		Description: "Find text on screen using OCR and click at its location. Uses a smart cascade: checks spatial memory (where text was seen before), then system find-text (Ctrl+F in browsers/apps), then OCR with optional scrolling. Use max_scrolls=5 for scrollable pages. Returns error with visible text if not found.",
	}, findTextAndClickHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "type_and_submit",
		Description: "Type text and press Enter (e.g. for form submission or search).",
	}, typeAndSubmitHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "launch_and_wait",
		Description: "Launch an application and wait for its window to appear.",
	}, launchAndWaitHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "screenshot_element",
		Description: "Take a screenshot of a specific window by handle.",
	}, screenshotElementHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "hover",
		Description: "Move the mouse to coordinates and wait briefly (for tooltips/hover menus).",
	}, hoverHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "wait_for_text",
		Description: "Wait for text to appear on screen. Polls OCR until found or timeout. Supports scrolling with max_scrolls to find text on scrollable pages.",
	}, waitForTextHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "select_all_and_type",
		Description: "Select all text (Ctrl+A) and type replacement text.",
	}, selectAllAndTypeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "click_menu_item",
		Description: "Find a window by title, then click a menu item or button using OCR within that window.",
	}, clickMenuItemHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_uptime",
		Description: "Get the system uptime (time since last boot).",
	}, getUptimeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "shutdown",
		Description: "Shut down the computer.",
	}, shutdownHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "restart",
		Description: "Restart the computer.",
	}, restartHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "sleep",
		Description: "Put the computer to sleep.",
	}, sleepHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "hibernate",
		Description: "Hibernate the computer.",
	}, hibernateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_keyboard_layout",
		Description: "Get the current keyboard layout / input language.",
	}, getKeyboardLayoutHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_keyboard_layout",
		Description: "Set the keyboard layout / input language (e.g. 'en-US', 'ja-JP').",
	}, setKeyboardLayoutHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_disk_usage",
		Description: "Get disk usage information for all drives.",
	}, getDiskUsageHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "open_file_explorer",
		Description: "Open File Explorer to a specified path (default: C:\\).",
	}, openFileExplorerHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "open_file_location",
		Description: "Open File Explorer with a specific file selected.",
	}, openFileLocationHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_screen_dpi",
		Description: "Get per-monitor screen DPI and scale percentage.",
	}, getScreenDPIHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_dpi_for_point",
		Description: "Get DPI and scale percentage at a specific screen coordinate. Useful for determining which monitor a coordinate is on and its scaling factor, especially in mixed-DPI multi-monitor setups.",
	}, getDPIPointHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_image",
		Description: "Find a template image on screen using NCC template matching. Provide template as base64 PNG. Returns coordinates of best match.",
	}, findImageHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_all_images",
		Description: "Find ALL occurrences of a template image on screen using NCC template matching. Provide template as base64 PNG. Returns array of matches with coordinates and scores.",
	}, findAllImagesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "ocr_languages",
		Description: "List all available Windows OCR languages. Returns array of language objects with tag, display_name, and native_name.",
	}, ocrLanguagesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "ocr_window",
		Description: "Extract text from a specific window by handle using Windows OCR. Captures what is currently visible in the window's region. If the window is minimized, behind other windows, or off-screen, the captured region will show whatever is on top at those screen coordinates. Use get_window_state to check state, then focus_window or restore_window first if needed.",
	}, ocrWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "ocr_active_window",
		Description: "Extract text from the currently active/foreground window using Windows OCR.",
	}, ocrActiveWindowHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "list_audio_devices",
		Description: "List all audio playback and recording devices.",
	}, listAudioDevicesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_default_audio_device",
		Description: "Set the default audio playback device by device ID.",
	}, setDefaultAudioDeviceHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "record_screen",
		Description: "Record screen frames at fixed intervals. Returns base64 images. Duration in ms, interval in ms.",
	}, recordScreenHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "chain",
		Description: "Execute a sequence of steps sequentially server-side. Steps can call any tool, wait, capture output, and use {{variable}} substitution. Mouse-based tools (click, move_mouse, hover, drag) auto-capture the UIA element at their target coordinates and include it in step output as 'element_at_point'. New step types: verify_ui (UIA element presence/absence check), if_uia (branch on element existence). New chain-callable tools: uia_find, uia_get_element_at_point, uia_get_all_elements, uia_set_text, wait_for_ui_element.",
		InputSchema: chainInputSchema(),
	}, chainHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_find",
		Description: "Find UI elements within windows by name, automation_id, or control_type using UI Automation. Returns bounding rectangles and properties (type, enabled state, etc.). Use this to locate text boxes, address bars, search menus, title bars, buttons, and other controls by their automation identity. The target window should be foreground (use focus_window first) for reliable results — some UIA providers only respond when the window is active.",
	}, uiaFindHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_get_text",
		Description: "Get text from a UI element by name or automation_id using UI Automation.",
	}, uiaGetTextHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_invoke",
		Description: "Click or invoke a UI element by name or automation_id using UI Automation.",
	}, uiaInvokeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_set_text",
		Description: "Set text in a UI element by name or automation_id using UI Automation.",
	}, uiaSetTextHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_get_element_at_point",
		Description: "Identify a UI element at screen coordinates (x, y) using UI Automation. Returns the element's name, control_type, automation_id, bounding rect, and whether it is enabled. Use this after clicking or hovering to validate what was under the cursor, or to determine what element exists at a given point before interacting.",
	}, uiaElementAtPointHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "uia_get_all_elements",
		Description: "Get all immediate child UI elements in a window by handle (title bar, menu bar, content panes, toolbars, status bar — one level deep, not recursive DOM tree). Returns name, control_type, automation_id, bounding rect, and enabled state for each. Use this to understand a window's full control surface — text boxes, buttons, search fields, address bars, menus, etc. The window should be foreground (use focus_window first) for reliable results. Use max_results to cap output.",
	}, uiaGetAllElementsHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "wait_for_ui_element",
		Description: "Wait for a UI element to appear in a window, identified by name or control_type. Polls UIA FindFirst on the window's descendants until found or timeout. Use this for content verification after an action (e.g., wait for a dialog to appear after clicking a button). Default timeout is 10 seconds.",
	}, waitForUIElementHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_status",
		Description: "Check ONNX runtime and model availability. Returns presence of YOLO model, MobileNet model, and onnxruntime.dll.",
	}, onnxStatusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_detect",
		Description: "Run YOLO-based UI element detection on a screenshot (or full screen if no image provided). Returns detected elements with class labels, confidence scores, and bounding boxes. Requires onnxruntime.dll and YOLO model file.",
	}, onnxDetectHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_download",
		Description: "Check and prepare ONNX model files. Lists which models are present and which need manual download.",
	}, onnxDownloadHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_watch_start",
		Description: "Start a background watcher that periodically screenshots the screen, runs ONNX detection, and caches results. Takes interval_seconds (default 5).",
	}, onnxWatchStartHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_watch_stop",
		Description: "Stop the background ONNX watcher.",
	}, onnxWatchStopHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_watch_status",
		Description: "Get the current ONNX watcher state: running, interval, last run time, cache size.",
	}, onnxWatchStatusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "onnx_watch_cache",
		Description: "Retrieve cached detections from the background watcher. Returns the most recent detection results with timestamps and saved reference paths.",
	}, onnxWatchCacheHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "layout_validate",
		Description: "Validate stored UI element layout against the current screen. Checks window existence, position drift, and OCR keyword verification. Returns adjusted coordinates and confidence levels (ok/drifted/stale).",
	}, layoutValidateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "template_store",
		Description: "Capture a UI element template from the current screen by cropping around a coordinate. Stores as base64 PNG in the element_templates table for visual re-identification.",
	}, templateStoreHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "template_find",
		Description: "Find a stored UI element template on the current screen using NCC template matching. Returns coordinates, score, and drift from stored position.",
	}, templateFindHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "template_list",
		Description: "List stored UI element templates with metadata (element key, scope, window title, hit count, etc.).",
	}, templateListHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "template_forget",
		Description: "Delete a stored UI element template by element_key and optional scope.",
	}, templateForgetHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "memory_set",
		Description: "Store a fact into the memory store. Fields: key (required), value (required, any JSON value), scope, tags (comma-separated), ttl (optional expiry in seconds).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"key": {"type": "string"},
				"value": {"description": "any JSON value"},
				"scope": {"type": "string"},
				"tags": {"type": "string"},
				"ttl": {"type": "integer"}
			},
			"required": ["key", "value"],
			"additionalProperties": false
		}`),
	}, memorySetHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "memory_get",
		Description: "Retrieve a fact from the memory store by key and optional scope.",
	}, memoryGetHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "memory_search",
		Description: "Full-text search across keys, values, scope, and tags using FTS5. Supports SQLite FTS5 query syntax.",
	}, memorySearchHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "memory_list",
		Description: "List stored facts under a scope with optional tag filter.",
	}, memoryListHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "memory_forget",
		Description: "Delete facts by key, scope, or tags. At least one filter is required to prevent accidental mass deletion.",
	}, memoryForgetHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "training_save_sample",
		Description: "Capture screenshot and save as a training sample with a task prompt (e.g. 'click the submit button'). The ONNX model learns from these during idle retraining.",
	}, trainingSaveSampleHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "training_list_samples",
		Description: "List saved training samples, optionally filtered by category or unused-only status.",
	}, trainingListSamplesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "training_stats",
		Description: "Get training data statistics: total samples, unused samples, breakdown by category, disk usage.",
	}, trainingStatsHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "training_mark_used",
		Description: "Mark a training sample as used (after the model has been trained on it).",
	}, trainingMarkUsedHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_ui_element",
		Description: "Find a UI element on screen by label. Checks memory first (from past ONNX detections), then runs ONNX detection, then falls back to OCR. Stores findings in memory for future reuse. Use this when the AI needs to locate an element it has seen before or needs to find programmatically.",
	}, findUIElementHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "priors_stats",
		Description: "Show learned element frequency and position statistics per window. Returns priors with sample count, frequency, and position distributions. Use min_count to filter out low-sample entries.",
	}, priorStatsHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "export_yolo_dataset",
		Description: "Export unused training samples as a YOLO-format dataset (images + labels + dataset.yaml) for external training with Ultralytics or other YOLO frameworks. Outputs to a directory of your choice.",
	}, exportYoloDatasetHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "training_cleanup_noise",
		Description: "Delete low-signal (signal_level=0) training samples older than max_age_hours. Use dry_run=true to see what would be deleted without actually removing anything. Returns deleted count and freed bytes.",
	}, trainingCleanupNoiseHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "datalog_query",
		Description: "Query the action/OCR data log. Table: commands, chains, ocr, or pairs. Filter by source, tool, success. Returns recent rows with all columns.",
	}, datalogQueryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "datalog_export",
		Description: "Export OCR+command training pairs as JSON for ML training. Optionally filter by session_id. Returns pairs with before/after OCR text and command JSON.",
	}, datalogExportHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "datalog_status",
		Description: "Get data logging statistics: count of commands, chains, OCR snapshots, and training pairs logged to the datalog database.",
	}, datalogStatusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "agent_analyze",
		Description: "Analyze the adaptive engine state — timing stats, success rates per tool, and learned OCR→command sequences. Returns a full report for AI decision-making.",
	}, agentAnalyzeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "agent_suggest",
		Description: "Given OCR screen text, predict the best next command based on past successful sequences. Returns ranked predictions with confidence scores and optional coord (x, y, confidence, samples) for click/hover/move_mouse.",
	}, agentSuggestHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "agent_train",
		Description: "Train the adaptive engine from datalog training_pairs. Rebuilds the OCR→command word index and sequence cache. Call after the datalog has accumulated new pairs.",
	}, agentTrainHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "task_begin",
		Description: "Mark the start of a task for post-task introspection. Call before the first tool call in a task.",
	}, taskBeginHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "task_end",
		Description: "Mark the end of a task. Returns mined insights: slow/failed tools, OCR stats, repeat patterns, and improvement suggestions.",
	}, taskEndHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "introspection_analyze",
		Description: "View task history with mined insights from past task_begin/task_end sessions.",
	}, introspectionAnalyzeHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "system_find_stats",
		Description: "Get system-find usage statistics: last used timestamp and total call count.",
	}, systemFindStatsHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "task_is_active",
		Description: "Check if a task session is currently active (between task_begin and task_end).",
	}, taskIsActiveHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "bridge_debug",
		Description: "Debug the OCR→command bridge state — shows recent OCR buffer, pending command, and timing info.",
	}, bridgeDebugHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "chain_predict",
		Description: "Predict the next action plus future actions from OCR text using the transformer model. Returns the primary prediction (tool, coordinates, args) and optionally a sequence of N future actions. Use the sequence to auto-generate chain steps.",
	}, chainPredictHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_config",
		Description: "Update runtime configuration. Accepts any subset of: training_enabled (stop/start background screenshot saving), prior_adjustment (enable/disable ML prior confidence tuning), verify_bounds (toggle coordinate bounds checking), log_level (debug/info/warn/error), watcher_enabled (start/stop the background screenshot watcher), watcher_interval_seconds (change polling frequency while running), tool_denylist (list of tool names to disable, e.g. [\"shutdown\",\"restart\"]), retention_days (auto-prune training samples older than N days, 0=disabled), chain_abort_enabled (enable/disable global hotkey abort), chain_abort_keys (hotkey combo like \"Ctrl+Shift+Escape\"), chain_abort_poll_ms (polling interval), window_lock_enabled (enable/disable screen tool locking), window_lock_auto_focus (auto re-focus locked window), log_file_enabled (enable/disable file-based logging), log_file_max_size_mb (max MB per log file before rotation), log_file_retention (number of rotated log files to keep), dashboard_enabled (enable/disable web dashboard on random port). Changes persist to disk.",
	}, setConfigHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "List directory contents. Returns entries with name, size, is_dir, mod_time, and mode.",
	}, listDirectoryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "read_file",
		Description: "Read a file with automatic type detection. Supports plaintext (txt, json, csv, yaml, etc.), docx, xlsx, pdf, and images (via OCR). Use page and page_size to paginate long content. Default page_size=8000 chars.",
	}, readFileHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write content to a file. Supports plaintext, docx (creates from text, preserves structure on overwrite), xlsx (TSV content becomes cells), and PDF (text creates PDF, JSON fills existing form fields). Requires overwrite=true to replace existing files.",
	}, writeFileHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "find_files",
		Description: "Recursively search for files matching a glob pattern (e.g. '*.go', '**/*.md').",
	}, findFilesHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "copy_file",
		Description: "Copy a file or directory (recursively) from source to destination.",
	}, copyFileHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename a file or directory.",
	}, moveFileHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file or directory to the Recycle Bin (uses SHFileOperationW with FOF_ALLOWUNDO).",
	}, deleteFileHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "create_directory",
		Description: "Create a directory (recursive, like mkdir -p).",
	}, createDirectoryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Get file or directory metadata: size, mod_time, is_dir, mode.",
	}, getFileInfoHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_working_directory",
		Description: "Set the working directory for relative path resolution in file tools.",
	}, setWorkingDirectoryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_working_directory",
		Description: "Get the current working directory used for relative path resolution.",
	}, getWorkingDirectoryHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "reset_state",
		Description: "Clear accumulated server state (adaptive engine stats, bridge buffer). Use between heavy batch operations to prevent state accumulation and timeouts.",
	}, resetStateHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "dismiss_all_menus",
		Description: "Press Escape to dismiss open context menus/dialogs. OCRs before and after to detect which menus were open and whether they closed.",
	}, dismissAllMenusHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "chain_abort",
		Description: "Check if the global chain abort hotkey has been pressed since last check. Returns {aborted: true} when the configured hotkey combo is detected. The abort is consumed on read (auto-resets). Call before starting long chains or poll periodically.",
	}, chainAbortHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "set_window_lock",
		Description: "Lock the active chain to a specific window by handle. Screen-touching tools (click, type, OCR, etc.) will verify the locked window is foreground before executing. If window_lock_auto_focus is enabled, automatically re-focuses the locked window when it loses foreground. Use GetWindowState or list_windows to find the handle.",
	}, setWindowLockHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "clear_window_lock",
		Description: "Release the window lock. Screen-touching tools will no longer be restricted to a specific window.",
	}, clearWindowLockHandler)

	addToolClean(server, &mcp.Tool{
		Name:        "get_logs",
		Description: "Read server log entries from the file-based log. Returns recent log lines with timestamps, levels, and messages. Useful for diagnosing tool failures, crashes, and errors after they occur.",
	}, safeHandler("get_logs", getLogsHandler))

	addToolClean(server, &mcp.Tool{
		Name:        "report_issue",
		Description: "Generate a GitHub issue report with system info, recent error logs, and context. If gh CLI is available, creates the issue automatically. Otherwise returns the markdown body for manual submission.",
	}, safeHandler("report_issue", reportIssueHandler))

	addToolClean(server, &mcp.Tool{
		Name:        "image_diff",
		Description: "Compare two base64-encoded PNG screenshots pixel by pixel. Returns statistics: changed_pixels, total_pixels, change_ratio (0-1), mean_diff (0-255), max_diff (0-255), same (bool). Optionally generates a diff image with changed pixels highlighted in red. Use threshold (0-255, default 30) to control sensitivity.",
	}, imageDiffHandler)

	if len(cfg.ToolDenylist) > 0 {
		var denied []string
		for _, name := range cfg.ToolDenylist {
			denied = append(denied, strings.ToLower(name))
		}
		server.RemoveTools(denied...)
		slog.Info("tool denylist applied", "denied", denied)
	}

	return server
}
