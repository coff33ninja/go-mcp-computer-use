package server

import (
	"context"
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
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func keyPressHandler(ctx context.Context, req *mcp.CallToolRequest, args KeyPressArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.KeyPress(args.Keys); err != nil {
		return nil, nil, fmt.Errorf("key_press failed: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
	}, nil, nil
}

func typeHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TypeText(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type failed: %w", err)
	}
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
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
}

func typeAndSubmitHandler(ctx context.Context, req *mcp.CallToolRequest, args TypeAndSubmitArgs) (*mcp.CallToolResult, any, error) {
	if err := actions.TypeAndSubmit(args.Text); err != nil {
		return nil, nil, fmt.Errorf("type_and_submit: %w", err)
	}
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

func getScreenDPIHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	dpi, err := actions.GetScreenDPI()
	if err != nil {
		return nil, nil, fmt.Errorf("get_screen_dpi: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, map[string]any{"monitors": dpi}, nil
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

	slog.Info("starting go-mcp-computer-use", "version", version, "tools", 70)

	// Warm up UIA to absorb the one-time 16-37s cold-start cost
	if err := actions.WarmupUIA(); err != nil {
		slog.Warn("uia warmup failed (UIA tools may be slow on first call)", "error", err)
	} else {
		slog.Info("uia warmup complete")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "go-mcp-computer-use",
		Version: version,
	}, nil)

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

	return server
}
