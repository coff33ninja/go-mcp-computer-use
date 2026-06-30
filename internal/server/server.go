package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/go-mcp-computer-use/internal/actions"
	"github.com/user/go-mcp-computer-use/internal/config"
)

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
}

type MoveMouseArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type ScrollArgs struct {
	Clicks    int32  `json:"clicks"`
	Direction string `json:"direction,omitempty"`
}

type KeyPressArgs struct {
	Keys []string `json:"keys"`
}

type KeyEventArgs struct {
	Key string `json:"key"`
}

type KeyloggerStartArgs struct{}

type KeyloggerStatusArgs struct{}

type TypeArgs struct {
	Text string `json:"text"`
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
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	X        *int32 `json:"x,omitempty"`
	Y        *int32 `json:"y,omitempty"`
	W        *int32 `json:"w,omitempty"`
	H        *int32 `json:"h,omitempty"`
}

type TypeAndSubmitArgs struct {
	Text string `json:"text"`
}

type LaunchAndWaitArgs struct {
	Path        string `json:"path"`
	WindowTitle string `json:"window_title"`
	TimeoutMs   int32  `json:"timeout_ms,omitempty"`
}

type ScreenshotElementArgs struct {
	Handle uintptr `json:"handle"`
}

type HoverArgs struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type WaitForTextArgs struct {
	Text      string `json:"text"`
	TimeoutMs int32  `json:"timeout_ms,omitempty"`
	Language  string `json:"language,omitempty"`
}

type SelectAllAndTypeArgs struct {
	Text string `json:"text"`
}

type ClickMenuItemArgs struct {
	WindowTitle  string `json:"window_title"`
	MenuItemText string `json:"menu_item_text"`
	Language     string `json:"language,omitempty"`
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"elements": elements}, nil
}

func uiaGetTextHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAGetTextArgs) (*mcp.CallToolResult, any, error) {
	text, err := actions.UIAGetText(args.Name, args.AutomationID)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_get_text: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]string{"text": text}, nil
}

func uiaInvokeHandler(ctx context.Context, req *mcp.CallToolRequest, args UIAInvokeArgs) (*mcp.CallToolResult, any, error) {
	success, err := actions.UIAInvoke(args.Name, args.AutomationID)
	if err != nil {
		return nil, nil, fmt.Errorf("uia_invoke: %w", err)
	}
	if !success {
		return nil, nil, fmt.Errorf("uia_invoke: element not found or not invocable")
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

type ChainArgs struct {
	Steps     []actions.ChainStep `json:"steps"`
	TimeoutMs int                 `json:"timeout_ms,omitempty"`
	OnError   string              `json:"on_error,omitempty"`
}

func chainHandler(ctx context.Context, req *mcp.CallToolRequest, args ChainArgs) (*mcp.CallToolResult, any, error) {
	chainReq := actions.ChainRequest{
		Steps:     args.Steps,
		TimeoutMs: args.TimeoutMs,
		OnError:   args.OnError,
	}
	result, err := actions.ExecuteChain(chainReq)
	if err != nil {
		return nil, nil, fmt.Errorf("chain: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
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
	if err := actions.Click(actions.ClickInput{
		X:      args.X,
		Y:      args.Y,
		Button: args.Button,
		Clicks: args.Clicks,
	}); err != nil {
		return nil, nil, fmt.Errorf("click failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatClick,
		fmt.Sprintf("click at (%d,%d)", args.X, args.Y))
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func moveMouseHandler(ctx context.Context, req *mcp.CallToolRequest, args MoveMouseArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.MoveMouse(args.X, args.Y); err != nil {
		return nil, nil, fmt.Errorf("move_mouse failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func scrollHandler(ctx context.Context, req *mcp.CallToolRequest, args ScrollArgs) (*mcp.CallToolResult, any, error) {
	clicks := args.Clicks
	if args.Direction == "down" {
		clicks = -clicks
	}
	if err := actions.Scroll(clicks); err != nil {
		return nil, nil, fmt.Errorf("scroll failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "scroll")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func keyPressHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyPressArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyPress(args.Keys); err != nil {
		return nil, nil, fmt.Errorf("key_press failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key press")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func keyDownHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyEventArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyDown(args.Key); err != nil {
		return nil, nil, fmt.Errorf("key_down failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key down")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func keyUpHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyEventArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyUp(args.Key); err != nil {
		return nil, nil, fmt.Errorf("key_up failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral, "key up")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func keyloggerStartHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyloggerStartArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.StartKeylogger(); err != nil {
		return nil, nil, fmt.Errorf("keylogger_start failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
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

func typeHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TypeText(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "type text")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func screenSizeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	w, h := actions.ScreenSize()
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, ScreenSizeResult{Width: w, Height: h}, nil
}

func cursorPosHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	x, y, err := actions.GetCursorPosition()
	if err != nil {
		return nil, nil, fmt.Errorf("get_cursor_position failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, CursorPosResult{X: x, Y: y}, nil
}

func dragHandler(ctx context.Context, req *mcp.CallToolRequest, args DragArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.Drag(args.FromX, args.FromY, args.ToX, args.ToY); err != nil {
		return nil, nil, fmt.Errorf("drag failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatGeneral,
		fmt.Sprintf("drag from (%d,%d) to (%d,%d)", args.FromX, args.FromY, args.ToX, args.ToY))
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func listWindowsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	windows, err := actions.ListWindows()
	if err != nil {
		return nil, nil, fmt.Errorf("list_windows failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, ListWindowsResult{Windows: windows}, nil
}

func focusWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args FocusWindowArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FocusWindow(args.Handle); err != nil {
		return nil, nil, fmt.Errorf("focus_window failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func getVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	vol, err := actions.GetVolume()
	if err != nil {
		return nil, nil, fmt.Errorf("get_volume failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]uint32{"volume": vol}, nil
}

func setVolumeHandler(ctx context.Context, req *mcp.CallToolRequest, args SetVolumeArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetVolume(args.Percent); err != nil {
		return nil, nil, fmt.Errorf("set_volume failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func setMuteHandler(ctx context.Context, req *mcp.CallToolRequest, args MuteArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetMute(args.Mute); err != nil {
		return nil, nil, fmt.Errorf("set_mute failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func getSystemInfoHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetSystemInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_system_info failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, info, nil
}

func getActiveWindowHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetActiveWindowInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_active_window failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, info, nil
}

func getClipboardHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	text, err := actions.GetClipboardText()
	if err != nil {
		return nil, nil, fmt.Errorf("get_clipboard failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]string{"text": text}, nil
}

func setClipboardHandler(ctx context.Context, req *mcp.CallToolRequest, args SetClipboardArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetClipboardText(args.Text); err != nil {
		return nil, nil, fmt.Errorf("set_clipboard failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func openURLHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenURLArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.OpenURL(args.URL); err != nil {
		return nil, nil, fmt.Errorf("open_url failed: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("open url %s", args.URL))
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func waitHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitArgs) (*mcp.CallToolResult, any, error) {
	actions.Wait(args.Ms)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func getPixelColorHandler(ctx context.Context, req *mcp.CallToolRequest, args PixelColorArgs) (*mcp.CallToolResult, any, error) {
	color, err := actions.GetPixelColor(args.X, args.Y)
	if err != nil {
		return nil, nil, fmt.Errorf("get_pixel_color failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]string{"color": color}, nil
}

func listProcessesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	processes, err := actions.ListProcesses()
	if err != nil {
		return nil, nil, fmt.Errorf("list_processes failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"processes": processes}, nil
}

func launchAppHandler(ctx context.Context, req *mcp.CallToolRequest, args LaunchAppArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.LaunchApp(args.Path); err != nil {
		return nil, nil, fmt.Errorf("launch_app failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func killProcessHandler(ctx context.Context, req *mcp.CallToolRequest, args KillProcessArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KillProcess(args.PID); err != nil {
		return nil, nil, fmt.Errorf("kill_process failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func listDisplaysHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	displays, err := actions.ListDisplays()
	if err != nil {
		return nil, nil, fmt.Errorf("list_displays failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"displays": displays}, nil
}

func displayModesHandler(ctx context.Context, req *mcp.CallToolRequest, args DisplayModesArgs) (*mcp.CallToolResult, any, error) {
	modes, err := actions.GetDisplayModes(args.DeviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("get_display_modes failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"modes": modes}, nil
}

func getBatteryHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	status, err := actions.GetBattery()
	if err != nil {
		return nil, nil, fmt.Errorf("get_battery failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, status, nil
}

func moveWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args MoveWindowArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.MoveWindowByHandle(args.Handle, args.X, args.Y, args.Width, args.Height); err != nil {
		return nil, nil, fmt.Errorf("move_window failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func minimizeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.MinimizeWindow(args.Handle)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func maximizeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.MaximizeWindow(args.Handle)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func restoreWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	actions.RestoreWindow(args.Handle)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func closeWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.CloseWindow(args.Handle); err != nil {
		return nil, nil, fmt.Errorf("close_window failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func getWindowStateHandler(ctx context.Context, req *mcp.CallToolRequest, args WindowHandleArgs) (*mcp.CallToolResult, any, error) {
	state, err := actions.GetWindowState(args.Handle)
	if err != nil {
		return nil, nil, fmt.Errorf("get_window_state failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, state, nil
}

func findWindowHandler(ctx context.Context, req *mcp.CallToolRequest, args FindWindowArgs) (*mcp.CallToolResult, any, error) {
	hwnd := actions.FindWindowByTitle(args.Title)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"handle": hwnd, "found": hwnd != 0}, nil
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
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"handle": hwnd, "found": true}, nil
}

func showNotificationHandler(ctx context.Context, req *mcp.CallToolRequest, args NotificationArgs) (*mcp.CallToolResult, any, error) {
	actions.ShowNotification(args.Title, args.Message)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func lockWorkstationHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.LockWorkstation()
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
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
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, result, nil
}

func getBrightnessHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	b, err := actions.GetBrightness()
	if err != nil {
		return nil, nil, fmt.Errorf("get_brightness failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]int{"brightness": b}, nil
}

func setBrightnessHandler(ctx context.Context, req *mcp.CallToolRequest, args BrightnessArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetBrightness(args.Percent); err != nil {
		return nil, nil, fmt.Errorf("set_brightness failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func getIdleTimeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	d, err := actions.GetIdleTime()
	if err != nil {
		return nil, nil, fmt.Errorf("get_idle_time failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]any{"idle_ms": d.Milliseconds()}, nil
}

func getNetworkInfoHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetNetworkInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get_network_info failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, info, nil
}

func pingHandler(ctx context.Context, req *mcp.CallToolRequest, args PingArgs) (*mcp.CallToolResult, any, error) {
	reachable, err := actions.PingHost(args.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("ping failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, map[string]bool{"reachable": reachable}, nil
}

func findTextAndClickHandler(ctx context.Context, req *mcp.CallToolRequest, args FindTextAndClickArgs) (*mcp.CallToolResult, any, error) {
	opts := actions.FindTextOpts{
		Text:     args.Text,
		Language: args.Language,
		RegionX:  args.X,
		RegionY:  args.Y,
		RegionW:  args.W,
		RegionH:  args.H,
	}
	if err := actions.FindTextAndClick(opts); err != nil {
		return nil, nil, fmt.Errorf("find_text_and_click: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatClick,
		fmt.Sprintf("find text and click: %s", args.Text))
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func typeAndSubmitHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeAndSubmitArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TypeAndSubmit(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type_and_submit: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "type and submit")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func launchAndWaitHandler(ctx context.Context, req *mcp.CallToolRequest, args LaunchAndWaitArgs) (*mcp.CallToolResult, any, error) {
	timeout := args.TimeoutMs
	if timeout == 0 { timeout = 10000 }
	hwnd, err := actions.LaunchAndWait(args.Path, args.WindowTitle, timeout)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "timeout"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"handle": hwnd, "found": true}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func waitForTextHandler(ctx context.Context, req *mcp.CallToolRequest, args WaitForTextArgs) (*mcp.CallToolResult, any, error) {
	timeout := args.TimeoutMs
	if timeout == 0 { timeout = 10000 }
	result, err := actions.WaitForText(args.Text, timeout, args.Language)
	if err != nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "not_found"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

func selectAllAndTypeHandler(ctx context.Context, req *mcp.CallToolRequest, args SelectAllAndTypeArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SelectAllAndType(args.Text); err != nil {
		return nil, nil, fmt.Errorf("select_all_and_type: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatType, "select all and type")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func clickMenuItemHandler(ctx context.Context, req *mcp.CallToolRequest, args ClickMenuItemArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.ClickMenuItem(args.WindowTitle, args.MenuItemText, args.Language); err != nil {
		return nil, nil, fmt.Errorf("click_menu_item: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func getUptimeHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	d, err := actions.GetUptime()
	if err != nil {
		return nil, nil, fmt.Errorf("get_uptime: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"uptime_ms": d.Milliseconds()}, nil
}

func shutdownHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Shutdown(); err != nil {
		return nil, nil, fmt.Errorf("shutdown: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func restartHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Restart(); err != nil {
		return nil, nil, fmt.Errorf("restart: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func sleepHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Sleep(); err != nil {
		return nil, nil, fmt.Errorf("sleep: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func hibernateHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.Hibernate(); err != nil {
		return nil, nil, fmt.Errorf("hibernate: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func getKeyboardLayoutHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info, err := actions.GetKeyboardLayout()
	if err != nil {
		return nil, nil, fmt.Errorf("get_keyboard_layout: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, info, nil
}

func setKeyboardLayoutHandler(ctx context.Context, req *mcp.CallToolRequest, args SetKeyboardLayoutArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetKeyboardLayout(args.Language); err != nil {
		return nil, nil, fmt.Errorf("set_keyboard_layout: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func getDiskUsageHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	disks, err := actions.GetDiskUsage()
	if err != nil {
		return nil, nil, fmt.Errorf("get_disk_usage: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"disks": disks}, nil
}

func openFileExplorerHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenExplorerArgs) (*mcp.CallToolResult, any, error) {
	path := args.Path
	if path == "" { path = "C:\\" }
	if err := actions.OpenFileExplorer(path); err != nil {
		return nil, nil, fmt.Errorf("open_file_explorer: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func openFileLocationHandler(ctx context.Context, req *mcp.CallToolRequest, args OpenExplorerArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.OpenFileLocation(args.Path); err != nil {
		return nil, nil, fmt.Errorf("open_file_location: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

func listAudioDevicesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	devices, err := actions.ListAudioDevices()
	if err != nil {
		return nil, nil, fmt.Errorf("list_audio_devices: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"devices": devices}, nil
}

func setDefaultAudioDeviceHandler(ctx context.Context, req *mcp.CallToolRequest, args SetAudioDeviceArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.SetDefaultAudioDevice(args.DeviceID); err != nil {
		return nil, nil, fmt.Errorf("set_default_audio_device: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func recordScreenHandler(ctx context.Context, req *mcp.CallToolRequest, args RecordScreenArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.RecordScreen(args.DurationMs, args.IntervalMs)
	if err != nil {
		return nil, nil, fmt.Errorf("record_screen: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

type LayoutValidateArgs struct {
	Elements       []actions.LayoutElement `json:"elements"`
	WindowTitle    string                  `json:"window_title,omitempty"`
	DriftTolerance int32                   `json:"drift_tolerance,omitempty"`
	Language       string                  `json:"language,omitempty"`
}

func layoutValidateHandler(ctx context.Context, req *mcp.CallToolRequest, args LayoutValidateArgs) (*mcp.CallToolResult, any, error) {
	result, err := actions.ValidateLayout(actions.LayoutValidateInput{
		Elements:       args.Elements,
		WindowTitle:    args.WindowTitle,
		DriftTolerance: args.DriftTolerance,
		Language:       args.Language,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("layout_validate: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

func getScreenDPIHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	dpi, err := actions.GetScreenDPI()
	if err != nil {
		return nil, nil, fmt.Errorf("get_screen_dpi: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"monitors": dpi}, nil
}

type FocusWindowByTitleArgs struct {
	Title string `json:"title"`
}

func focusWindowByTitleHandler(ctx context.Context, req *mcp.CallToolRequest, args FocusWindowByTitleArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.FocusWindowByTitle(args.Title); err != nil {
		return nil, nil, fmt.Errorf("focus_window_by_title: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func browserNewTabHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserNewTab(args.Browser); err != nil {
		return nil, nil, fmt.Errorf("browser_new_tab: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func browserNavigateHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserNavigateArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserNavigate(args.Browser, args.URL); err != nil {
		return nil, nil, fmt.Errorf("browser_navigate: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("navigate to %s", args.URL))
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func browserSearchHandler(ctx context.Context, req *mcp.CallToolRequest, args BrowserSearchArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.BrowserSearch(args.Browser, args.Query); err != nil {
		return nil, nil, fmt.Errorf("browser_search: %w", err)
	}
	actions.SaveSnapshotAfterAction(actions.TrainingSourceRaw, actions.TrainingCatNavigate,
		fmt.Sprintf("search %s", args.Query))
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func explorerFocusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	if err := actions.ExplorerFocus(); err != nil {
		return nil, nil, fmt.Errorf("explorer_focus: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func explorerOpenPathHandler(ctx context.Context, req *mcp.CallToolRequest, args ExplorerPathArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.ExplorerOpenPath(args.Path); err != nil {
		return nil, nil, fmt.Errorf("explorer_open_path: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

type ONNXDetectArgs struct {
	ImageB64     string  `json:"image_b64,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	IOUThreshold float64 `json:"iou_threshold,omitempty"`
}

func onnxStatusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	status := actions.ONNXStatus()
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, status, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

func onnxDownloadHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	result, err := actions.ONNXDownload()
	if err != nil {
		return nil, nil, fmt.Errorf("onnx_download: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

type ONNXWatchStartArgs struct {
	IntervalSeconds int `json:"interval_seconds"`
}

func onnxWatchStartHandler(ctx context.Context, req *mcp.CallToolRequest, args ONNXWatchStartArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.StartWatcher(args.IntervalSeconds); err != nil {
		return nil, nil, fmt.Errorf("onnx_watch_start: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, actions.GetWatcherStatus(), nil
}

func onnxWatchStopHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	actions.StopWatcher()
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, actions.GetWatcherStatus(), nil
}

func onnxWatchStatusHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, actions.GetWatcherStatus(), nil
}

func onnxWatchCacheHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"detections": actions.GetCachedDetections()}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, info, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"templates": results}, nil
}

func templateForgetHandler(ctx context.Context, req *mcp.CallToolRequest, args TemplateForgetArgs) (*mcp.CallToolResult, any, error) {
	deleted, err := actions.TemplateForget(args.ElementKey, args.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("template_forget: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"deleted": deleted}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func memoryGetHandler(ctx context.Context, req *mcp.CallToolRequest, args MemoryGetArgs) (*mcp.CallToolResult, any, error) {
	fact, err := actions.MemoryGet(args.Key, args.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("memory_get: %w", err)
	}
	if fact == nil {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "not_found"}}}, map[string]any{"found": false}, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, fact, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"results": results}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"results": results}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"deleted": deleted}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, sample, nil
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
		samples = []actions.TrainingSample{}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"samples": samples}, nil
}

func trainingStatsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	stats, err := actions.TrainingStatsReport()
	if err != nil {
		return nil, nil, fmt.Errorf("training_stats: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, stats, nil
}

func trainingMarkUsedHandler(ctx context.Context, req *mcp.CallToolRequest, args TrainingMarkUsedArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TrainingMarkUsed(args.ID); err != nil {
		return nil, nil, fmt.Errorf("training_mark_used: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"marked": args.ID}, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

type PriorStatsArgs struct {
	MinCount int `json:"min_count,omitempty"`
}

func priorStatsHandler(ctx context.Context, req *mcp.CallToolRequest, args PriorStatsArgs) (*mcp.CallToolResult, any, error) {
	stats, err := actions.GetPriorStats(args.MinCount)
	if err != nil {
		return nil, nil, fmt.Errorf("priors_stats: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, stats, nil
}

type ExportYoloDatasetArgs struct {
	OutputDir string `json:"output_dir"`
	MinSignal int    `json:"min_signal,omitempty"`
}

func exportYoloDatasetHandler(ctx context.Context, req *mcp.CallToolRequest, args ExportYoloDatasetArgs) (*mcp.CallToolResult, any, error) {
	if args.OutputDir == "" {
		return nil, nil, fmt.Errorf("output_dir is required")
	}
	stats, err := actions.ExportYoloDataset(args.OutputDir, args.MinSignal)
	if err != nil {
		return nil, nil, fmt.Errorf("export_yolo_dataset: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, stats, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, result, nil
}

type SetConfigArgs struct {
	TrainingEnabled      *bool   `json:"training_enabled,omitempty"`
	PriorAdjustment      *bool   `json:"prior_adjustment,omitempty"`
	VerifyBounds         *bool   `json:"verify_bounds,omitempty"`
	LogLevel             string  `json:"log_level,omitempty"`
	WatcherEnabled       *bool   `json:"watcher_enabled,omitempty"`
	WatcherIntervalSecs  *int    `json:"watcher_interval_seconds,omitempty"`
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, stats, nil
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, info, nil
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

func bridgeDebugHandler(_ context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	info := actions.BridgeDebugInfo()
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, info, nil
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

	if changed {
		slog.Info("config updated", "training_enabled", cfg.TrainingEnabled,
			"prior_adjustment", cfg.PriorAdjustment, "verify_bounds", cfg.VerifyBounds,
			"log_level", cfg.LogLevel)
		if err := cfg.Save(); err != nil {
			slog.Warn("failed to save config", "error", err)
		}
	}

	watcherStatus := actions.GetWatcherStatus()

	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{
		"training_enabled":       cfg.TrainingEnabled,
		"prior_adjustment":       cfg.PriorAdjustment,
		"verify_bounds":          cfg.VerifyBounds,
		"log_level":              cfg.LogLevel,
		"watcher_running":        watcherStatus.Running,
		"watcher_interval_secs":  cfg.WatcherIntervalSecs,
		"saved":                  changed,
	}, nil
}

func New(version string) *mcp.Server {
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("config load failed, using defaults", "error", err)
		cfg = config.Default()
	}
	actions.ActiveConfig = cfg

	level := new(slog.LevelVar)
	level.Set(slog.Level(cfg.LogLevelSlog()))
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("starting go-mcp-computer-use", "version", version, "tools", 111, "tools_doc", "docs/tools.md")

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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "screenshot",
		Description: "Capture the screen or a region. If w/h omitted, captures full screen.",
	}, screenshotHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "click",
		Description: "Click at screen coordinates x,y. Button: left/right. Clicks: 1 or 2.",
	}, clickHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_mouse",
		Description: "Move mouse cursor to x,y.",
	}, moveMouseHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "scroll",
		Description: "Scroll the mouse wheel. Positive clicks = up, negative = down.",
	}, scrollHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "key_press",
		Description: "Press key combination. Example: [\"Ctrl\", \"C\"] for copy.",
	}, keyPressHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "key_down",
		Description: "Hold a key down (does not release it). Use key_up to release. Example: \"W\"",
	}, keyDownHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "key_up",
		Description: "Release a key that was held down with key_down. Example: \"W\"",
	}, keyUpHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keylogger_start",
		Description: "Start recording keyboard and mouse input for replay",
	}, keyloggerStartHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keylogger_stop",
		Description: "Stop recording and return recorded sequence as chain steps",
	}, keyloggerStopHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keylogger_status",
		Description: "Check if keylogger is active and event count",
	}, keyloggerStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "type",
		Description: "Type text at the currently focused element.",
	}, typeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_screen_size",
		Description: "Get the screen dimensions.",
	}, screenSizeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_cursor_position",
		Description: "Get the current mouse cursor position.",
	}, cursorPosHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "drag",
		Description: "Drag mouse from (from_x, from_y) to (to_x, to_y).",
	}, dragHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_windows",
		Description: "List all visible windows with their handles, titles, and PIDs.",
	}, listWindowsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "focus_window",
		Description: "Bring a window to the foreground by handle.",
	}, focusWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "focus_window_by_title",
		Description: "Find a window by title and focus it, clicking its title bar to ensure activation. Useful before keyboard input in chain steps.",
	}, focusWindowByTitleHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "browser_focus_url_bar",
		Description: "Focus a browser window's URL bar. Supports Firefox (Ctrl+T), Chrome/Edge (Ctrl+L), and other browsers. Provide browser name (firefox, chrome, edge, brave, opera) or window title substring.",
	}, browserFocusURLBarHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "browser_new_tab",
		Description: "Open a new tab in a browser window. Uses Ctrl+T for all browsers.",
	}, browserNewTabHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "browser_navigate",
		Description: "Open a new tab in a browser and navigate to a URL.",
	}, browserNavigateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "browser_search",
		Description: "Open a new tab in a browser and perform a search query.",
	}, browserSearchHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "explorer_focus",
		Description: "Focus an existing File Explorer window.",
	}, explorerFocusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "explorer_open_path",
		Description: "Open a File Explorer window at the specified path. Reuses existing window when possible.",
	}, explorerOpenPathHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_volume",
		Description: "Get the current system volume level (0-100).",
	}, getVolumeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the system volume level (0-100).",
	}, setVolumeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_mute",
		Description: "Mute or unmute the system audio.",
	}, setMuteHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_system_info",
		Description: "Get system information (hostname, OS, RAM).",
	}, getSystemInfoHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_active_window",
		Description: "Get the current foreground window info.",
	}, getActiveWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_clipboard",
		Description: "Read text from the clipboard.",
	}, getClipboardHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_clipboard",
		Description: "Write text to the clipboard.",
	}, setClipboardHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_url",
		Description: "Open a URL in the default browser.",
	}, openURLHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "wait",
		Description: "Wait for N milliseconds before the next action.",
	}, waitHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_pixel_color",
		Description: "Get the hex color at screen coordinates x,y.",
	}, getPixelColorHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_processes",
		Description: "List all running processes with PID, name, and thread count.",
	}, listProcessesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "launch_app",
		Description: "Launch an application by path or shell command.",
	}, launchAppHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "kill_process",
		Description: "Terminate a process by PID.",
	}, killProcessHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_displays",
		Description: "List all monitors with resolution and position.",
	}, listDisplaysHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_display_modes",
		Description: "Get all available display modes (resolution, refresh rate, color depth) for a monitor by device name.",
	}, displayModesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_battery",
		Description: "Get battery status (percentage, charging, on battery).",
	}, getBatteryHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_window",
		Description: "Move and resize a window by handle.",
	}, moveWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "minimize_window",
		Description: "Minimize a window by handle.",
	}, minimizeWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "maximize_window",
		Description: "Maximize a window by handle.",
	}, maximizeWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "restore_window",
		Description: "Restore a minimized or maximized window by handle.",
	}, restoreWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "close_window",
		Description: "Close a window by handle.",
	}, closeWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_window_state",
		Description: "Get window state (visible, minimized, maximized, position, size).",
	}, getWindowStateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_window",
		Description: "Find a window handle by title.",
	}, findWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "wait_for_window",
		Description: "Wait for a window with the given title to appear. Returns handle or timeout.",
	}, waitForWindowHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "show_notification",
		Description: "Show a Windows notification message box.",
	}, showNotificationHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "lock_workstation",
		Description: "Lock the workstation.",
	}, lockWorkstationHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ocr",
		Description: "Extract text from screen using Windows OCR. Supports full screen or region (x,y,w,h).",
	}, ocrHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_brightness",
		Description: "Get the current display brightness level (0-100).",
	}, getBrightnessHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_brightness",
		Description: "Set the display brightness level (0-100).",
	}, setBrightnessHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_idle_time",
		Description: "Get the system idle time (time since last user input) in milliseconds.",
	}, getIdleTimeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_network_info",
		Description: "Get network information: hostname, IP addresses, DNS servers, default gateway.",
	}, getNetworkInfoHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Ping a host to check network reachability.",
	}, pingHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_text_and_click",
		Description: "Find text on screen using OCR and click at its location. Optional region x,y,w,h to search within.",
	}, findTextAndClickHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "type_and_submit",
		Description: "Type text and press Enter (e.g. for form submission or search).",
	}, typeAndSubmitHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "launch_and_wait",
		Description: "Launch an application and wait for its window to appear.",
	}, launchAndWaitHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "screenshot_element",
		Description: "Take a screenshot of a specific window by handle.",
	}, screenshotElementHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hover",
		Description: "Move the mouse to coordinates and wait briefly (for tooltips/hover menus).",
	}, hoverHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "wait_for_text",
		Description: "Wait for text to appear on screen. Polls OCR until found or timeout.",
	}, waitForTextHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "select_all_and_type",
		Description: "Select all text (Ctrl+A) and type replacement text.",
	}, selectAllAndTypeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "click_menu_item",
		Description: "Find a window by title, then click a menu item or button using OCR within that window.",
	}, clickMenuItemHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_uptime",
		Description: "Get the system uptime (time since last boot).",
	}, getUptimeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "shutdown",
		Description: "Shut down the computer.",
	}, shutdownHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "restart",
		Description: "Restart the computer.",
	}, restartHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sleep",
		Description: "Put the computer to sleep.",
	}, sleepHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hibernate",
		Description: "Hibernate the computer.",
	}, hibernateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_keyboard_layout",
		Description: "Get the current keyboard layout / input language.",
	}, getKeyboardLayoutHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_keyboard_layout",
		Description: "Set the keyboard layout / input language (e.g. 'en-US', 'ja-JP').",
	}, setKeyboardLayoutHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_disk_usage",
		Description: "Get disk usage information for all drives.",
	}, getDiskUsageHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_file_explorer",
		Description: "Open File Explorer to a specified path (default: C:\\).",
	}, openFileExplorerHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_file_location",
		Description: "Open File Explorer with a specific file selected.",
	}, openFileLocationHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_screen_dpi",
		Description: "Get per-monitor screen DPI and scale percentage.",
	}, getScreenDPIHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_image",
		Description: "Find a template image on screen using NCC template matching. Provide template as base64 PNG. Returns coordinates of best match.",
	}, findImageHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_audio_devices",
		Description: "List all audio playback and recording devices.",
	}, listAudioDevicesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_default_audio_device",
		Description: "Set the default audio playback device by device ID.",
	}, setDefaultAudioDeviceHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "record_screen",
		Description: "Record screen frames at fixed intervals. Returns base64 images. Duration in ms, interval in ms.",
	}, recordScreenHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "chain",
		Description: "Execute a sequence of steps sequentially server-side. Steps can call any tool, wait, capture output, and use {{variable}} substitution.",
	}, chainHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uia_find",
		Description: "Find UI elements by name, automation_id, or control_type using UI Automation. Returns bounding rectangles and properties.",
	}, uiaFindHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uia_get_text",
		Description: "Get text from a UI element by name or automation_id using UI Automation.",
	}, uiaGetTextHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uia_invoke",
		Description: "Click or invoke a UI element by name or automation_id using UI Automation.",
	}, uiaInvokeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_status",
		Description: "Check ONNX runtime and model availability. Returns presence of YOLO model, MobileNet model, and onnxruntime.dll.",
	}, onnxStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_detect",
		Description: "Run YOLO-based UI element detection on a screenshot (or full screen if no image provided). Returns detected elements with class labels, confidence scores, and bounding boxes. Requires onnxruntime.dll and YOLO model file.",
	}, onnxDetectHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_download",
		Description: "Check and prepare ONNX model files. Lists which models are present and which need manual download.",
	}, onnxDownloadHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_watch_start",
		Description: "Start a background watcher that periodically screenshots the screen, runs ONNX detection, and caches results. Takes interval_seconds (default 5).",
	}, onnxWatchStartHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_watch_stop",
		Description: "Stop the background ONNX watcher.",
	}, onnxWatchStopHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_watch_status",
		Description: "Get the current ONNX watcher state: running, interval, last run time, cache size.",
	}, onnxWatchStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "onnx_watch_cache",
		Description: "Retrieve cached detections from the background watcher. Returns the most recent detection results with timestamps and saved reference paths.",
	}, onnxWatchCacheHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "layout_validate",
		Description: "Validate stored UI element layout against the current screen. Checks window existence, position drift, and OCR keyword verification. Returns adjusted coordinates and confidence levels (ok/drifted/stale).",
	}, layoutValidateHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "template_store",
		Description: "Capture a UI element template from the current screen by cropping around a coordinate. Stores as base64 PNG in the element_templates table for visual re-identification.",
	}, templateStoreHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "template_find",
		Description: "Find a stored UI element template on the current screen using NCC template matching. Returns coordinates, score, and drift from stored position.",
	}, templateFindHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "template_list",
		Description: "List stored UI element templates with metadata (element key, scope, window title, hit count, etc.).",
	}, templateListHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "template_forget",
		Description: "Delete a stored UI element template by element_key and optional scope.",
	}, templateForgetHandler)

	mcp.AddTool(server, &mcp.Tool{
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_get",
		Description: "Retrieve a fact from the memory store by key and optional scope.",
	}, memoryGetHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_search",
		Description: "Full-text search across keys, values, scope, and tags using FTS5. Supports SQLite FTS5 query syntax.",
	}, memorySearchHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_list",
		Description: "List stored facts under a scope with optional tag filter.",
	}, memoryListHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_forget",
		Description: "Delete facts by key, scope, or tags. At least one filter is required to prevent accidental mass deletion.",
	}, memoryForgetHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "training_save_sample",
		Description: "Capture screenshot and save as a training sample with a task prompt (e.g. 'click the submit button'). The ONNX model learns from these during idle retraining.",
	}, trainingSaveSampleHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "training_list_samples",
		Description: "List saved training samples, optionally filtered by category or unused-only status.",
	}, trainingListSamplesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "training_stats",
		Description: "Get training data statistics: total samples, unused samples, breakdown by category, disk usage.",
	}, trainingStatsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "training_mark_used",
		Description: "Mark a training sample as used (after the model has been trained on it).",
	}, trainingMarkUsedHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_ui_element",
		Description: "Find a UI element on screen by label. Checks memory first (from past ONNX detections), then runs ONNX detection, then falls back to OCR. Stores findings in memory for future reuse. Use this when the AI needs to locate an element it has seen before or needs to find programmatically.",
	}, findUIElementHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "priors_stats",
		Description: "Show learned element frequency and position statistics per window. Returns priors with sample count, frequency, and position distributions. Use min_count to filter out low-sample entries.",
	}, priorStatsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "export_yolo_dataset",
		Description: "Export unused training samples as a YOLO-format dataset (images + labels + dataset.yaml) for external training with Ultralytics or other YOLO frameworks. Outputs to a directory of your choice.",
	}, exportYoloDatasetHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "training_cleanup_noise",
		Description: "Delete low-signal (signal_level=0) training samples older than max_age_hours. Use dry_run=true to see what would be deleted without actually removing anything. Returns deleted count and freed bytes.",
	}, trainingCleanupNoiseHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "datalog_query",
		Description: "Query the action/OCR data log. Table: commands, chains, ocr, or pairs. Filter by source, tool, success. Returns recent rows with all columns.",
	}, datalogQueryHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "datalog_export",
		Description: "Export OCR+command training pairs as JSON for ML training. Optionally filter by session_id. Returns pairs with before/after OCR text and command JSON.",
	}, datalogExportHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "datalog_status",
		Description: "Get data logging statistics: count of commands, chains, OCR snapshots, and training pairs logged to the datalog database.",
	}, datalogStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_analyze",
		Description: "Analyze the adaptive engine state — timing stats, success rates per tool, and learned OCR→command sequences. Returns a full report for AI decision-making.",
	}, agentAnalyzeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_suggest",
		Description: "Given OCR screen text, predict the best next command based on past successful sequences. Returns ranked predictions with confidence scores.",
	}, agentSuggestHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_train",
		Description: "Train the adaptive engine from datalog training_pairs. Rebuilds the OCR→command word index and sequence cache. Call after the datalog has accumulated new pairs.",
	}, agentTrainHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "task_begin",
		Description: "Mark the start of a task for post-task introspection. Call before the first tool call in a task.",
	}, taskBeginHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "task_end",
		Description: "Mark the end of a task. Returns mined insights: slow/failed tools, OCR stats, repeat patterns, and improvement suggestions.",
	}, taskEndHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "introspection_analyze",
		Description: "View task history with mined insights from past task_begin/task_end sessions.",
	}, introspectionAnalyzeHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "bridge_debug",
		Description: "Debug the OCR→command bridge state — shows recent OCR buffer, pending command, and timing info.",
	}, bridgeDebugHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_config",
		Description: "Update runtime configuration. Accepts any subset of: training_enabled (stop/start background screenshot saving), prior_adjustment (enable/disable ML prior confidence tuning), verify_bounds (toggle coordinate bounds checking), log_level (debug/info/warn/error), watcher_enabled (start/stop the background screenshot watcher), watcher_interval_seconds (change polling frequency while running). Changes persist to disk. Use this to disable data collection or control the tool at runtime.",
	}, setConfigHandler)

	return server
}
