# Codebase Map

Maps MCP tools → handlers → action functions → documentation.

## Project Structure

```
go-mcp-computer-use/
├── cmd/                          # Entry points
│   ├── mcp-server/               # Main server binary (main.go)
│   ├── benchmark/                # Performance benchmark tool
│   └── ocrhelper/                # OCR helper binary (WinRT OCR via COM)
├── internal/
│   ├── server/                   # MCP server, tool registration, arg types, handlers
│   │   └── server.go             # ~2491 lines — all tool registrations + handlers
│   ├── actions/                  # Core action implementations (46 files)
│   └── config/                   # JSON config loading/saving
├── scripts/
│   ├── gen-tools-doc.go          # Auto-generates docs/reference/tools.md
│   ├── lint.ps1                  # Run vet + build locally
│   ├── build.ps1                 # Zig cc + CGO build
│   └── push-and-release.ps1      # Automated release script
├── docs/
│   ├── architecture.md           # Agent stack architecture
│   ├── ci-cd-pipeline.md         # CI/CD workflow reference
│   ├── security.md               # Data collection, privacy controls
│   ├── comparison-vs-windows-recall.md
│   ├── adr/                      # Architecture Decision Records
│   ├── reference/                # Tools, config, versioning, MCP clients, codebase map
│   ├── guides/                   # Build, accessibility, agent guides, CUA guide
│   └── meta/                     # Plan, backlog, changelog, known issues
├── VERSION                       # Canonical version (semver)
├── config.json                   # Runtime config (per-user)
└── .github/workflows/            # CI, Release, Auto Tag, Module Maintenance
```

## Tool-to-Action Mapping

### Input Simulation (`internal/actions/mouse.go` + `keyboard.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `click` | `clickHandler` | `actions.Click(ClickInput)` | `mouse.go:50` |
| `move_mouse` | `moveMouseHandler` | `actions.MoveMouse(x, y)` | `mouse.go:101` |
| `drag` | `dragHandler` | `actions.Drag(from_x, from_y, to_x, to_y)` | `mouse.go` |
| `scroll` | `scrollHandler` | `actions.Scroll(clicks, horizontal)` | `mouse.go:128` |
| `get_cursor_position` | `cursorPosHandler` | `actions.GetCursorPosition()` | `mouse.go:119` |
| `key_press` | `keyPressHandler` | `actions.KeyPress([]string)` | `keyboard.go:233` |
| `key_down` | `keyDownHandler` | `actions.KeyDown(string)` | `keyboard.go:195` |
| `key_up` | `keyUpHandler` | `actions.KeyUp(string)` | `keyboard.go:214` |
| `type` | `typeHandler` | `actions.TypeText(string)` | `keyboard.go:289` |

### Visual Perception (`internal/actions/ocr.go`, `ocr_com.go`, `template.go`, `onnx.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `screenshot` | `screenshotHandler` | `actions.CaptureScreen(region)` | `screenshot.go` |
| `screenshot_element` | `screenshotElementHandler` | `actions.ScreenshotElement(handle)` | `chained.go:103` |
| `ocr` | `ocrHandler` | `actions.OCRScreen(lang)` / `actions.OCRRegion(x,y,w,h,lang)` | `ocr.go:108/122` |
| `ocr_languages` | `ocrLanguagesHandler` | `actions.OcrLanguages()` | `ocr.go:151` |
| `find_image` | `findImageHandler` | `actions.FindImage(screenB64, templateB64, threshold)` → cascades to ONNX → OCR on failure | `template.go:33` |
| `find_all_images` | `findAllImagesHandler` | `actions.FindAllImages(screenB64, templateB64, threshold)` → cascades to ONNX + OCR on failure | `template.go:158` |
| `get_pixel_color` | `getPixelColorHandler` | `actions.GetPixelColor(x, y)` | `misc.go:208` |
| `get_screen_size` | `screenSizeHandler` | `actions.GetScreenSize()` | `system.go` |
| `record_screen` | `recordScreenHandler` | `actions.RecordScreen(durationMs, intervalMs)` | `recording.go:20` |

### ONNX ML Detection (`internal/actions/onnx.go`, `priors.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `onnx_status` | `onnxStatusHandler` | `actions.ONNXStatus()` | `onnx.go:135` |
| `onnx_detect` | `onnxDetectHandler` | `actions.ONNXDetect(DetectionInput)` | `onnx.go:181` |
| `onnx_download` | `onnxDownloadHandler` | `actions.ONNXDownload()` | `onnx.go:519` |
| `onnx_watch_start` | `onnxWatchStartHandler` | starts background ONNX watcher | `watcher.go` |
| `onnx_watch_stop` | `onnxWatchStopHandler` | stops background watcher | `watcher.go` |
| `onnx_watch_status` | `onnxWatchStatusHandler` | watcher state | `watcher.go` |
| `onnx_watch_cache` | `onnxWatchCacheHandler` | cached detections | `watcher.go` |
| `find_ui_element` | `findUIElementHandler` | multi-stage: memory → ONNX → UIA → OCR | `ui_finder.go` |
| `priors_stats` | `priorStatsHandler` | `actions.GetPriorStats(minCount)` | `priors.go:286` |
| `export_yolo_dataset` | `exportYoloDatasetHandler` | `actions.ExportYoloDataset(dir, minSignal)` | `priors.go:326` |
| `training_cleanup_noise` | `trainingCleanupNoiseHandler` | `actions.TrainingCleanupNoise(hours, dryRun)` | `priors.go:486` |

### Window Management (`internal/actions/window.go`, `window_ext.go`, `dpi.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `list_windows` | `listWindowsHandler` | `actions.ListWindows()` | `window.go` |
| `focus_window` | `focusWindowHandler` | `actions.FocusWindow(handle)` | `window.go` |
| `focus_window_by_title` | `focusWindowByTitleHandler` | `actions.FocusWindowByTitle(title)` | `chained.go:242` |
| `get_active_window` | `getActiveWindowHandler` | `actions.GetActiveWindow()` | `system.go` |
| `get_window_state` | `getWindowStateHandler` | `actions.GetWindowState(handle)` | `window_ext.go` |
| `move_window` | `moveWindowHandler` | `actions.MoveWindow(handle, x, y, w, h)` | `window.go` |
| `minimize_window` | `minimizeWindowHandler` | `actions.MinimizeWindow(handle)` | `window.go` |
| `maximize_window` | `maximizeWindowHandler` | `actions.MaximizeWindow(handle)` | `window.go` |
| `restore_window` | `restoreWindowHandler` | `actions.RestoreWindow(handle)` | `window.go` |
| `close_window` | `closeWindowHandler` | `actions.CloseWindow(handle)` | `window.go` |
| `find_window` | `findWindowHandler` | `actions.FindWindow(title)` | `window.go` |
| `wait_for_window` | `waitForWindowHandler` | `actions.WaitForWindow(title, timeoutMs)` | `window.go` |

### Browser Automation (`internal/actions/browseruse.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `browser_focus_url_bar` | `browserFocusURLBarHandler` | `actions.BrowserFocusURLBar(browser)` | `browseruse.go:104` |
| `browser_new_tab` | `browserNewTabHandler` | `actions.BrowserNewTab(browser)` | `browseruse.go:127` |
| `browser_navigate` | `browserNavigateHandler` | `actions.BrowserNavigate(browser, url)` | `browseruse.go:149` |
| `browser_search` | `browserSearchHandler` | `actions.BrowserSearch(browser, query)` | `browseruse.go:163` |

### File Explorer (`internal/actions/windowexploreruse.go`, `power.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `explorer_focus` | `explorerFocusHandler` | `actions.ExplorerFocus()` | `windowexploreruse.go` |
| `explorer_open_path` | `explorerOpenPathHandler` | `actions.ExplorerOpenPath(path)` | `windowexploreruse.go` |
| `open_file_explorer` | `openFileExplorerHandler` | `actions.OpenFileExplorer(path)` | `power.go:126` |
| `open_file_location` | `openFileLocationHandler` | `actions.OpenFileLocation(path)` | `power.go:137` |

### Process & System (`internal/actions/process.go`, `system.go`, `misc.go`, `dpi.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `list_processes` | `listProcessesHandler` | `actions.ListProcesses()` | `process.go:43` |
| `launch_app` | `launchAppHandler` | `actions.LaunchApp(path)` | `process.go:72` |
| `kill_process` | `killProcessHandler` | `actions.KillProcess(pid)` | `process.go:91` |
| `launch_and_wait` | `launchAndWaitHandler` | `actions.LaunchAndWait(path, title, timeout)` | `chained.go:83` |
| `get_system_info` | `getSystemInfoHandler` | `actions.GetSystemInfo()` | `system.go` |
| `get_screen_dpi` | `getScreenDPIHandler` | `actions.GetScreenDPI()` | `layout.go:54` |
| `get_keyboard_layout` | `getKeyboardLayoutHandler` | `actions.GetKeyboardLayout()` | `layout.go:20` |
| `set_keyboard_layout` | `setKeyboardLayoutHandler` | `actions.SetKeyboardLayout(lang)` | `layout.go:44` |

### Audio (`internal/actions/audio.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `get_volume` | `getVolumeHandler` | `actions.GetVolume()` | `audio.go` (WinRT) |
| `set_volume` | `setVolumeHandler` | `actions.SetVolume(percent)` | `audio.go` (WinRT) |
| `set_mute` | `setMuteHandler` | `actions.SetMute(bool)` | `audio.go` (WinRT) |
| `list_audio_devices` | `listAudioDevicesHandler` | `actions.ListAudioDevices()` | `audio.go:53` |
| `set_default_audio_device` | `setDefaultAudioDeviceHandler` | `actions.SetDefaultAudioDevice(id)` | `audio.go:77` |

### Power & Display (`internal/actions/power.go`, `brightness.go`, `misc.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `shutdown` | `shutdownHandler` | `actions.Shutdown()` | `power.go:34` |
| `restart` | `restartHandler` | `actions.Restart()` | `power.go:42` |
| `sleep` | `sleepHandler` | `actions.Sleep()` | `power.go:50` |
| `hibernate` | `hibernateHandler` | `actions.Hibernate()` | `power.go:58` |
| `get_battery` | `getBatteryHandler` | `actions.GetBattery()` | `misc.go:165` |
| `get_uptime` | `getUptimeHandler` | `actions.GetUptime()` | `power.go:26` |
| `get_idle_time` | `getIdleTimeHandler` | `actions.GetIdleTime()` | `idle.go:19` |
| `get_disk_usage` | `getDiskUsageHandler` | `actions.GetDiskUsage()` | `power.go:90` |
| `get_brightness` | `getBrightnessHandler` | `actions.GetBrightness()` | `brightness.go:23` |
| `set_brightness` | `setBrightnessHandler` | `actions.SetBrightness(percent)` | `brightness.go:10` |
| `list_displays` | `listDisplaysHandler` | `actions.ListDisplays()` | `misc.go:126` |
| `get_display_modes` | `displayModesHandler` | `actions.GetDisplayModes(deviceName)` | `misc.go:140` |

### Network (`internal/actions/network.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `get_network_info` | `getNetworkInfoHandler` | `actions.GetNetworkInfo()` | `network.go:68` |
| `ping` | `pingHandler` | `actions.PingHost(host)` | `network.go:148` |

### Clipboard & URL (`internal/actions/misc.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `get_clipboard` | `getClipboardHandler` | `actions.GetClipboard()` | `misc.go` |
| `set_clipboard` | `setClipboardHandler` | `actions.SetClipboard(text)` | `misc.go` |
| `open_url` | `openURLHandler` | `actions.OpenURL(url)` | `misc.go` |

### UI Automation (`internal/actions/uia.go`, `uia_com.go`, `uipi.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `uia_find` | `uiaFindHandler` | `actions.UIAFind(name, automationId, controlType)` | `uia.go` |
| `uia_get_text` | `uiaGetTextHandler` | `actions.UIAGetText(name, automationId)` | `uia.go` |
| `uia_invoke` | `uiaInvokeHandler` | `actions.UIAInvoke(name, automationId)` | `uia.go` |
| `layout_validate` | `layoutValidateHandler` | `actions.LayoutValidate(elements, windowTitle)` | `validate_layout.go` |

### Compound Actions (`internal/actions/chained.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `hover` | `hoverHandler` | `actions.Hover(x, y)` | `chained.go:133` |
| `find_text_and_click` | `findTextAndClickHandler` | `actions.FindTextAndClick(FindTextOpts)` | `chained.go:19` |
| `type_and_submit` | `typeAndSubmitHandler` | `actions.TypeAndSubmit(text)` | `chained.go:68` |
| `select_all_and_type` | `selectAllAndTypeHandler` | `actions.SelectAllAndType(text)` | `chained.go:181` |
| `click_menu_item` | `clickMenuItemHandler` | `actions.ClickMenuItem(windowTitle, text, lang)` | `chained.go:198` |
| `wait_for_text` | `waitForTextHandler` | `actions.WaitForText(text, timeoutMs, lang)` | `chained.go:152` |
| `wait` | `waitHandler` | `actions.Wait(ms)` | `misc.go:184` |
| `show_notification` | `showNotificationHandler` | `actions.ShowNotification(title, msg)` | `misc.go:192` |
| `lock_workstation` | `lockWorkstationHandler` | `actions.LockWorkstation()` | `misc.go:203` |

### Keylogger (`internal/actions/keylogger.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `keylogger_start` | `keyloggerStartHandler` | `actions.StartKeylogger()` | `keylogger.go:143` |
| `keylogger_stop` | `keyloggerStopHandler` | `actions.StopKeylogger()` | `keylogger.go:303` |
| `keylogger_status` | `keyloggerStatusHandler` | `actions.KeyloggerStatus()` | `keylogger.go:336` |

### Chain Runner (`internal/actions/chain.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `chain` | `chainHandler` | `actions.ExecuteChain(ChainRequest)` | `chain.go:168` |

### Memory Store (`internal/actions/memory.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `memory_set` | `memorySetHandler` | `actions.MemorySet(MemorySetInput)` | `memory.go:125` |
| `memory_get` | `memoryGetHandler` | `actions.MemoryGet(key, scope)` | `memory.go:161` |
| `memory_search` | `memorySearchHandler` | `actions.MemorySearch(MemorySearchInput)` | `memory.go:232` |
| `memory_list` | `memoryListHandler` | `actions.MemoryList(MemoryListInput)` | `memory.go:309` |
| `memory_forget` | `memoryForgetHandler` | `actions.MemoryForget(MemoryForgetInput)` | `memory.go:390` |

### Template Store (`internal/actions/memory.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `template_store` | `templateStoreHandler` | `actions.TemplateStore(TemplateStoreInput)` | `memory.go:492` |
| `template_find` | `templateFindHandler` | `actions.TemplateFind(TemplateFindInput)` | `memory.go:586` |
| `template_list` | `templateListHandler` | `actions.TemplateList(TemplateListInput)` | `memory.go:656` |
| `template_forget` | `templateForgetHandler` | `actions.TemplateForget(key, scope)` | `memory.go:747` |

### Data Logging (`internal/actions/datalog.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `datalog_query` | `datalogQueryHandler` | `actions.QueryDataLog(DataLogQuery)` | `datalog.go:435` |
| `datalog_export` | `datalogExportHandler` | `actions.ExportTrainingData(sessionID, limit)` | `datalog.go:529` |
| `datalog_status` | `datalogStatusHandler` | `actions.DataLogStatsReport()` | `datalog.go:575` |

### Adaptive Engine (`internal/actions/adaptive.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `agent_analyze` | `agentAnalyzeHandler` | `actions.AdaptiveEngine.Analyze()` | `adaptive.go:663` |
| `agent_suggest` | `agentSuggestHandler` | `actions.AdaptiveEngine.PredictActions(ocrText, limit)` | `adaptive.go:486` |
| `agent_train` | `agentTrainHandler` | `actions.AdaptiveEngine.TrainFromDatalog()` | `adaptive.go:342` |

### Introspection (`internal/actions/introspection.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `task_begin` | `taskBeginHandler` | `actions.TaskBegin(TaskInput)` | `introspection.go:82` |
| `task_end` | `taskEndHandler` | `actions.TaskEnd(TaskEndInput)` | `introspection.go:110` |
| `introspection_analyze` | `introspectionAnalyzeHandler` | `actions.IntrospectionAnalyze()` | `introspection.go:351` |
| `bridge_debug` | `bridgeDebugHandler` | `actions.BridgeDebugInfo()` | `datalog.go:305` |

### Training (`internal/actions/training.go`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `training_save_sample` | `trainingSaveSampleHandler` | `actions.SaveTrainingSample(category, prompt)` | `training.go` |
| `training_list_samples` | `trainingListSamplesHandler` | `actions.ListTrainingSamples(category, unusedOnly)` | `training.go` |
| `training_stats` | `trainingStatsHandler` | `actions.TrainingStats()` | `training.go` |
| `training_mark_used` | `trainingMarkUsedHandler` | `actions.MarkTrainingUsed(id)` | `training.go` |

### Runtime Config (`internal/config/`)

| Tool | Handler | Action Function | File |
|------|---------|-----------------|------|
| `set_config` | `setConfigHandler` | runtime config update | config via `set_config` API |

## Action File Index

| File | Lines | Key Types | Key Functions |
|------|-------|-----------|---------------|
| `adaptive.go` | ~700 | `AdaptiveEngine`, `TimingStat`, `WordIndex`, `PredictedAction`, `EngineAnalysis` | `TrainFromDatalog`, `PredictActions`, `Analyze`, `RecordTiming/Success/Result` |
| `audio.go` | ~80 | `AudioDevice` | `ListAudioDevices`, `SetDefaultAudioDevice` (+ WinRT for volume/mute) |
| `brightness.go` | ~30 | — | `SetBrightness`, `GetBrightness` (WinRT) |
| `browseruse.go` | ~180 | — | `BrowserFocusURLBar`, `BrowserNewTab`, `BrowserNavigate`, `BrowserSearch` |
| `chain.go` | ~880 | `ChainRequest`, `ChainStep`, `ChainResult`, `PollConfig`, `IfConfig`, `LoopConfig` | `ExecuteChain`, `execWait`, `execPoll`, `execIf`, `execLoop`, `execTool` + 30 `chain*` wrappers |
| `chained.go` | ~260 | `FindTextOpts` | `FindTextAndClick`, `TypeAndSubmit`, `LaunchAndWait`, `ScreenshotElement`, `Hover`, `WaitForText`, `SelectAllAndType`, `ClickMenuItem`, `FocusWindowByTitle` |
| `datalog.go` | ~600 | `DataLogConfig`, `TrainingPairInput`, `DataLogQuery` | `InitDataLog`, `LogToolCall`, `LogCommand`, `LogChain`, `LogOCRSnapshot`, `BridgeDebugInfo`, `QueryDataLog`, `ExportTrainingData` |
| `dpi.go` | ~180 | `MonitorDPI`, `WindowNormalizer`, `NormalizedElement` | `SetDPIAware`, `ListMonitorDPIs`, `GetDPIScaleForPoint/Window`, `WindowNormalizer.{Normalize,Denormalize}` |
| `idle.go` | ~30 | `LASTINPUTINFO` | `GetIdleTime` |
| `introspection.go` | ~380 | `TaskInput`, `TaskEndInput`, `TaskInfo`, `TaskInsights`, `ToolStat` | `TaskBegin`, `TaskEnd`, `IntrospectionAnalyze`, `mineTaskInsights` |
| `keyboard.go` | ~290 | `keyboardInput`, `charVK` | `KeyDown`, `KeyUp`, `KeyPress`, `TypeText` (SendInput) |
| `keylogger.go` | ~350 | `recordedEvent` | `StartKeylogger`, `StopKeylogger`, `KeyloggerStatus` (WinEvent hook) |
| `layout.go` | ~60 | `KeyboardLayoutInfo` | `GetKeyboardLayout`, `SetKeyboardLayout` |
| `memory.go` | ~760 | `MemoryFact`, `MemorySetInput`, `MemorySearchInput`; `TemplateInfo`, `TemplateStoreInput` | `MemorySet/Get/Search/List/Forget`; `TemplateStore/Find/List/Forget`; `MemoryStoreDetectionElements` |
| `misc.go` | ~210 | `BatteryStatus`, `DisplayInfo`, `DisplayMode` | `ListDisplays`, `GetDisplayModes`, `GetBattery`, `Wait`, `ShowNotification`, `LockWorkstation`, `GetPixelColor` |
| `mouse.go` | ~140 | `ClickInput`, `mouseInput` | `Click` (left/right/**middle**), `MoveMouse`, `GetCursorPosition`, `Scroll(clicks, horizontal)`, `Drag` (SendInput) |
| `network.go` | ~160 | `NetworkInfo` | `GetNetworkInfo`, `PingHost` (iphlpapi) |
| `ocr.go` | ~210 | `OCRResult`, `OCRLine`, `OCRWord`, `LanguageInfo` | `OCRScreen`, `OCRRegion`, `OCRProportionalWindowRegion`, `OcrLanguages` (WinRT COM) |
| `onnx.go` | ~540 | `DetectionInput`, `DetectedElement`, `DetectionOutput`, `ONNXModelStatus` | `InitONNX`, `ONNXDetect`, `ONNXDownload`, `ONNXStatus` |
| `power.go` | ~160 | `DiskUsage` | `GetUptime`, `Shutdown`, `Restart`, `Sleep`, `Hibernate`, `GetDiskUsage`, `OpenFileExplorer`, `OpenFileLocation` |
| `priors.go` | ~500 | `ElementPrior`, `PriorStatsOutput`, `YoloDatasetStats` | `InitPriors`, `UpdatePriorsFromDetections`, `AdjustConfidenceWithPriors`, `GetPriorStats`, `ExportYoloDataset`, `TrainingCleanupNoise` |
| `process.go` | ~100 | `ProcessInfo` | `ListProcesses`, `LaunchApp`, `KillProcess` |
| `recording.go` | ~30 | `RecordedFrame`, `RecordingResult` | `RecordScreen` |
| `screenshot.go` | — | — | `CaptureScreen` (full + region) |
| `system.go` | ~70 | `SystemInfo`, `ActiveWindowInfo` | `GetSystemInfo`, `GetActiveWindow`, `ListWindows` |
| `template.go` | ~510 | `MatchResult` | `FindImage` (NCC → ONNX → OCR cascade), `FindAllImages` (NCC+NMS → ONNX+OCR), `CropRegion`, `ensureScreenB64`, `findImageONNXFallback`, `findAllONNXFallback` |
| `timeout.go` | ~20 | — | `WithTimeout` |
| `training.go` | — | — | `SaveTrainingSample`, `ListTrainingSamples`, `TrainingStats`, `MarkTrainingUsed` |
| `uia.go` | — | — | `UIAFind`, `UIAGetText`, `UIAInvoke` |
| `uia_com.go` | — | COM bindings | UIA COM interface |
| `uipi.go` | — | — | UIPI (User Interface Privilege Isolation) detection, UAC bypass |
| `user32.go` | — | — | User32 DLL proc declarations |
| `validate_layout.go` | — | — | `LayoutValidate` |
| `validate.go` | — | — | Config validation |
| `watcher.go` | — | — | ONNX background watcher (periodic screenshot → detection) |
| `window.go` | — | `WindowInfo` | `FocusWindow`, `FindWindow`, `MoveWindow`, `MinimizeWindow` etc. |
| `window_ext.go` | ~170 | `WindowRect`, `WindowStateInfo` (+`Fullscreen bool`), `MONITORINFO` | `MoveWindowByHandle`, `GetWindowRectByHandle`, `GetWindowState` (now includes fullscreen detection via MonitorFromWindow), `CloseWindow`, `isFullscreen` |
| `windowexploreruse.go` | — | — | `ExplorerFocus`, `ExplorerOpenPath` |
| `winrt.go` | — | WinRT COM bindings | Audio, brightness via WinRT |

## Internal Package Dependencies

```
cmd/mcp-server/ (main)
  └─ internal/server/ (tool registration, handlers)
       ├─ internal/actions/ (all action implementations)
       └─ internal/config/ (JSON config)

internal/actions/ depends on:
  ├─ user32.dll (window management, input via SendInput)         ← docs/reference/windows-dll-ref.md
  ├─ gdi32.dll (screenshot BitBlt, DPI)
  ├─ kernel32.dll (process, idle time, thread attachment)
  ├─ shell32.dll (ShellExecute)
  ├─ shcore.dll (DPI awareness, per-monitor scale)
  ├─ ntdll.dll (NtQuerySystemInformation)
  ├─ powrprof.dll (sleep, hibernate)
  ├─ advapi32.dll (UIPI elevation)
  ├─ iphlpapi.dll (network info)
  ├─ winmm.dll (wave volume)
  ├─ combase.dll (WinRT COM — RoInitialize, HSTRING)
  ├─ ole32.dll / oleaut32.dll (UIA COM)
  ├─ onnxruntime.dll (YOLO/MobileNet inference)
  └─ SQLite (datalog, memory store, priors, introspection)
```

Full DLL proc mapping and COM interface architecture: [`windows-dll-ref.md`](windows-dll-ref.md).<br>
UIPI elevation detection logic (used by keyboard + chained input guards): [`uipi.md`](uipi.md).<br>
COM/WinRT patterns — vtable dispatch, async polling, HSTRING/BSTR, threading: [`com-patterns.md`](com-patterns.md).<br>
Vtable index stability model and CI/CD verification: [`vtable-verification.md`](vtable-verification.md).

## Docs-to-Code Index

| Doc | Covers |
|-----|--------|
| `docs/reference/tools.md` | Auto-generated from `server.go`, lists all 120+ tools with descriptions |
| `docs/reference/codebase-map.md` | (this file) tool→handler→action→file mapping |
| `docs/reference/configuration.md` | `config.json` schema, set_config tool |
| `docs/reference/versioning-strategy.md` | VERSION file, bump rules, release process, changelog convention |
| `docs/reference/mcp-client-configs.md` | Per-agent JSON config examples for 19 MCP clients |
| `docs/reference/models-setup.md` | ONNX model download, format, compatibility |
| `docs/architecture.md` | Agent stack layers (LLM→MCP→Controller/Perception/Memory/Training) |
| `docs/ci-cd-pipeline.md` | CI/release workflows, branching, local build steps |
| `docs/security.md` | Data collection, `set_config` controls, watcher management, `training_cleanup_noise` |
| `docs/comparison-vs-windows-recall.md` | Vs. Microsoft Recall — architecture, privacy, accessibility |
| `docs/guides/build.md` | Build requirements, Zig cc + CGO, cross-compilation |
| `docs/guides/agent-guides.md` | Tool subsets per task type, prompt patterns, agent workflows |
| `docs/guides/accessibility.md` | Assistive technology use cases, hands-free operation |
| `docs/guides/computer-use-guide-for-ai-agents.md` | Perceive-Reason-Act loop, layered agent architecture |
| `docs/adr/adr-001-mcp-sdk-selection.md` | Why `modelcontextprotocol/go-sdk` |
| `docs/adr/adr-002-windows-automation-strategy.md` | Win32 API + native COM/WinRT, CGO only for ONNX |
| `docs/meta/plan.md` | Project plan, progress, prioritized work items |
| `docs/meta/backlog.md` | 326-tool roadmap |
| `docs/meta/known-issues.md` | Known issues and workarounds |
| `docs/meta/CHANGELOG.md` | Release history |
