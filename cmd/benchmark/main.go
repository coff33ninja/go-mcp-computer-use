package main

import (
	"fmt"
	"math"
	"time"

	"github.com/user/go-mcp-computer-use/internal/actions"
	"github.com/user/go-mcp-computer-use/internal/config"
)

func main() {
	actions.ActiveConfig = config.Default()
	actions.SetDPIAware()
	if err := actions.CheckScreenshotPermission(); err != nil {
		fmt.Printf("FAIL: %v\n", err)
		return
	}

	w, h := actions.ScreenSize()
	fmt.Printf("Screen: %dx%d\n\n", w, h)

	screenshot := measure("full screen capture 5x", func() {
		for i := 0; i < 5; i++ {
			actions.CaptureScreen()
		}
	}) / 5

	regionSize := int32(400)
	region := measure("region capture 400x400 5x", func() {
		for i := 0; i < 5; i++ {
			actions.CaptureRegion(0, 0, regionSize, regionSize)
		}
	}) / 5

	b64, _ := actions.CaptureScreen()
	b64region, _ := actions.CaptureRegion(0, 0, 400, 400)

	ocrFull := measure("screen OCR 3x", func() {
		for i := 0; i < 3; i++ {
			actions.OCRScreen("en-US")
		}
	}) / 3

	ocrRegion := measure("region OCR 400x400 3x", func() {
		for i := 0; i < 3; i++ {
			actions.OCRRegion(0, 0, 400, 400, "en-US")
		}
	}) / 3

	nullTmpl := "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAFklEQVQ4y2P4z8BQz0BKYBg1YNQAkgAAX5AD/8OTiJcAAAAASUVORK5CYII="
	tmplMatch := measure("template match 16x16 full screen 5x", func() {
		for i := 0; i < 5; i++ {
			actions.FindImage(b64, nullTmpl, 0.7)
		}
	}) / 5

	tmplRegion := measure("template match 16x16 in region 3x", func() {
		for i := 0; i < 3; i++ {
			actions.FindImage(b64region, nullTmpl, 0.7)
		}
	}) / 3

	findText := measure("find_text_and_click (unlikely text) 3x", func() {
		for i := 0; i < 3; i++ {
			actions.FindTextAndClick(actions.FindTextOpts{
				Text: fmt.Sprintf("__benchmark_nonexistent_%d__", i),
			})
		}
	}) / 3

	pixel := measure("get_pixel_color 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetPixelColor(100, 100)
		}
	}) / 5

	idle := measure("get_idle_time 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetIdleTime()
		}
	}) / 5

	uptime := measure("get_uptime 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetUptime()
		}
	}) / 5

	disk := measure("get_disk_usage 3x", func() {
		for i := 0; i < 3; i++ {
			actions.GetDiskUsage()
		}
	}) / 3

	proc := measure("list_processes 5x", func() {
		for i := 0; i < 5; i++ {
			actions.ListProcesses()
		}
	}) / 5

	disp := measure("list_displays 5x", func() {
		for i := 0; i < 5; i++ {
			actions.ListDisplays()
		}
	}) / 5

	vol := measure("get_volume 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetVolume()
		}
	}) / 5

	batt := measure("get_battery 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetBattery()
		}
	}) / 5

	windows := measure("list_windows 5x", func() {
		for i := 0; i < 5; i++ {
			actions.ListWindows()
		}
	}) / 5

	dpi := measure("get_screen_dpi 5x", func() {
		for i := 0; i < 5; i++ {
			actions.GetScreenDPI()
		}
	}) / 5

	kblayout := measure("get_keyboard_layout 3x", func() {
		for i := 0; i < 3; i++ {
			actions.GetKeyboardLayout()
		}
	}) / 3

	network := measure("get_network_info 3x", func() {
		for i := 0; i < 3; i++ {
			actions.GetNetworkInfo()
		}
	}) / 3

	fmt.Println("=== BENCHMARK RESULTS (average time per call) ===")
	Result("Screenshot (full)", "ms", float64(screenshot.Nanoseconds())/1e6)
	Result("Screenshot (region 400x400)", "ms", float64(region.Nanoseconds())/1e6)
	Result("OCR (full screen)", "ms", float64(ocrFull.Nanoseconds())/1e6)
	Result("OCR (region 400x400)", "ms", float64(ocrRegion.Nanoseconds())/1e6)
	Result("Template match (full screen)", "ms", float64(tmplMatch.Nanoseconds())/1e6)
	Result("Template match (in region)", "ms", float64(tmplRegion.Nanoseconds())/1e6)
	Result("find_text_and_click", "ms", float64(findText.Nanoseconds())/1e6)
	Result("get_pixel_color", "µs", float64(pixel.Nanoseconds())/1e3)
	Result("get_idle_time", "µs", float64(idle.Nanoseconds())/1e3)
	Result("get_uptime", "µs", float64(uptime.Nanoseconds())/1e3)
	Result("get_disk_usage", "ms", float64(disk.Nanoseconds())/1e6)
	Result("list_processes", "ms", float64(proc.Nanoseconds())/1e6)
	Result("list_windows", "ms", float64(windows.Nanoseconds())/1e6)
	Result("list_displays", "ms", float64(disp.Nanoseconds())/1e6)
	Result("get_screen_dpi", "ms", float64(dpi.Nanoseconds())/1e6)
	Result("get_volume", "µs", float64(vol.Nanoseconds())/1e3)
	Result("get_battery", "µs", float64(batt.Nanoseconds())/1e3)
	Result("get_keyboard_layout", "ms", float64(kblayout.Nanoseconds())/1e6)
	Result("get_network_info", "ms", float64(network.Nanoseconds())/1e6)
	fmt.Println()
}

func measure(label string, fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

func Result(name, unit string, val float64) {
	if val >= 100 {
		fmt.Printf("%-35s %8.0f %s\n", name, val, unit)
	} else if val >= 1 {
		fmt.Printf("%-35s %8.1f %s\n", name, val, unit)
	} else {
		fmt.Printf("%-35s %8.0f %s\n", name, math.Round(val), unit)
	}
}
