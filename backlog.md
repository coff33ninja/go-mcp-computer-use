# Backlog — Every Desktop Ability a Human Has

> Goal: `go-mcp-computer-use` must be able to do *literally everything* a human can do on a Windows PC.
> Key: prompt engineering to chain tools and manage which model gets which subset.

## How to Read

- **HAVE** = implemented (108 tools)
- **NEXT** = high-impact, feasible additions
- **FAR** = possible but lower priority or complex
- Items within a section ordered roughly by priority

---

## 1. VISION — See What's on Screen

### HAVE
- `screenshot` — full screen or region → base64 PNG
- `get_pixel_color` — hex color at x,y
- `get_screen_size` — virtual screen dimensions
- `get_screen_dpi` — per-monitor DPI + scale %
- `ocr` — extract text via Windows.Media.Ocr (any language)
- `find_image` — NCC template matching, returns (x,y,score)
- `record_screen` — frame polling at interval → base64 frame array

### NEXT
| Tool | Why |
|------|-----|
| `ocr_languages` | list installed OCR languages (so agent knows what's available) |
| `find_all_image` | return ALL matches, not just the best one |
| `image_diff` | pixel-level diff between two screenshots (detect changes) |
| `image_histogram` | color histogram analysis (detect dark/bright screens) |
| `match_template_multi` | multi-pass NCC at different scales (handle DPI variation) |

### FAR
| Tool | Why |
|------|-----|
| `detect_ui_elements` | use OCR bounding boxes to classify buttons/text fields/lists |
| `barcode_reader` | decode QR/DataMatrix/Code128 from screen region |
| `face_detection` | WinRT face detection via Camera |
| `screen_recording_video` | encode frames to actual video file (mp4) |
| `real_time_stream` | WebSocket stream of frames for live agent view |
| `color_detection` | find all pixels of a given color on screen |
| `snap_diff` | compare screenshot to a reference "golden" image |

---

## 2. MOUSE — Point and Click

### HAVE
- `click` — left/right, single/double at x,y
- `move_mouse` — move cursor to x,y
- `scroll` — wheel up/down (clicks)
- `drag` — click-hold from→to
- `hover` — move + wait 300ms (for tooltips)
- `get_cursor_position` — current x,y

### NEXT
| Tool | Why |
|------|-----|
| `middle_click` — middle button click | open links in new tab, close tabs |
| `click_hold` / `release` — separate hold/release | complex drag-and-drop, slider manipulation |
| `scroll_smooth` — pixel-based scrolling | precise scroll in lists/canvases |
| `scroll_horizontal` — horizontal wheel/tilt | horizontal scrolling in wide content |
| `drag_relative` — drag by (dx, dy) from current pos | relative drag gestures |

### FAR
| Tool | Why |
|------|-----|
| `mouse_gesture` — recognize shape (L, V, Z) | gesture-based commands |
| `click_all_matches` — click every occurrence on screen | dismiss all notifications, close all tabs |
| `right_click_menu` — right-click + get menu items | context menu interaction |
| `multi_touch` — simulate touch gestures (pinch, swipe) | tablet/touch scenarios |
| `pen_stylus` — WinRT pen simulation | drawing, handwriting |

---

## 3. KEYBOARD — Type and Command

### HAVE
- `type` — send text string
- `key_press` — key combos (Ctrl+C, Alt+Tab, etc.)
- `type_and_submit` — type + Enter
- `select_all_and_type` — Ctrl+A + type

### NEXT
| Tool | Why |
|------|-----|
| `type_character_by_character` — type with per-char delay | simulate human typing for apps that buffer input |
| `type_with_modifiers` — e.g. type "hello" while holding Shift | advanced text entry |
| `key_hold` / `key_release` — separate hold/release | games, modifier management |
| `get_keyboard_state` — which modifier keys are pressed | check CapsLock/NumLock/ScrollLock state |
| `set_keyboard_state` — toggle CapsLock/NumLock | fix common input issues |

### FAR
| Tool | Why |
|------|-----|
| `text_from_clipboard_paste` — paste instead of type | faster, avoids IME/input issues |
| `text_file_input` — type contents of a text file | paste large text without clipboard |
| `ime_text_input` — send text through IME composition | Japanese/Chinese input methods |
| `macro_record` / `macro_playback` — record key sequence | repeatable automation |
| `send_keys_advanced` — SendKeys-style with pauses | legacy app compatibility |

---

## 4. WINDOWS — Manage Screen Real Estate

### HAVE
- `list_windows` — all visible windows (handle, title, PID)
- `focus_window` — bring to foreground
- `find_window` — by title substring
- `wait_for_window` — poll until window appears
- `move_window` — set x,y,w,h
- `minimize_window` / `maximize_window` / `restore_window`
- `close_window`
- `get_window_state` — visible, minimized, maximized, position
- `screenshot_element` — screenshot a specific window

### NEXT
| Tool | Why |
|------|-----|
| `snap_window_left` / `snap_window_right` — snap to half screen | common multi-window workflow |
| `snap_window_top` / `snap_window_bottom` / `snap_window_corner` | quarter-screen layouts |
| `get_window_z_order` — window stacking order | understand overlap |
| `set_window_z_order` — bring to top/bottom/above/below | reorder windows |
| `set_window_transparency` — alpha blend (SetLayeredWindowAttributes) | peek through windows |
| `set_window_title` — change window title | identification |
| `cascade_windows` / `tile_windows` — arrange all windows | classic Windows arrange |
| `find_window_by_pid` — get window owned by a process | process→window mapping |

### FAR
| Tool | Why |
|------|-----|
| `get_window_children` — enumerate child windows | UI Automation for controls |
| `window_screenshot_multi` — screenshot all windows at once | full desktops |
| `scroll_window` — scroll content inside a window | scroll without focus |
| `click_window_at` — click relative to window origin | element positioning |
| `set_window_always_on_top` | floating windows |
| `get_window_class_name` — Win32 class name | identify control types |

---

## 5. VIRTUAL DESKTOPS — Multiple Desktops

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `list_virtual_desktops` | enumerate all desktops |
| `switch_virtual_desktop` (by index or GUID) | move between workspaces |
| `create_virtual_desktop` | new workspace |
| `remove_virtual_desktop` | delete workspace |
| `move_window_to_desktop` | move window to different desktop |
| `get_current_desktop` | which desktop is active |

### FAR
| Tool | Why |
|------|-----|
| `pin_window_to_all_desktops` | show on every desktop |
| `get_desktop_wallpaper` | background image |

---

## 6. PROCESSES — Run and Manage Apps

### HAVE
- `list_processes` — name, PID, threads, parent PID
- `launch_app` — ShellExecute (open verb)
- `kill_process` — TerminateProcess by PID

### NEXT
| Tool | Why |
|------|-----|
| `launch_app_with_args` — specify arguments | run commands with params |
| `launch_app_hidden` — no window | run background processes |
| `launch_app_as_admin` — UAC elevation | admin tasks |
| `launch_app_with_env` — set env vars for child proc | custom environment |
| `wait_for_process_exit` — wait until PID exits | sequential task chaining |
| `launch_and_wait_exit` — launch + wait for exit | run-and-collect |
| `get_process_info` — memory, CPU, command line | process details |

### FAR
| Tool | Why |
|------|-----|
| `set_process_priority` — idle/below_normal/normal/above_normal/high/realtime | CPU management |
| `suspend_process` / `resume_process` | freeze/thaw |
| `get_process_threads` — list threads | diagnostics |
| `run_as_user` — impersonation | different user context |
| `create_process_group` / `kill_process_tree` | manage process families |

---

## 7. FILE SYSTEM — Navigate and Manipulate Files

### HAVE
- `get_disk_usage` — all drives (total, free, used %)
- `open_file_explorer` — open Explorer to path
- `open_file_location` — open Explorer with file selected

### NEXT
| Tool | Why |
|------|-----|
| `list_directory` — list files/subdirs | browse files |
| `read_file` — read file contents (text) | inspect configs, logs |
| `write_file` — write text to file | create/edit files |
| `delete_file` — move to recycle bin or permanent | clean up |
| `copy_file` / `move_file` | file operations |
| `create_directory` | new folders |
| `get_file_info` — size, date, attributes | metadata |
| `find_files` — search by name/pattern | locate files |

### FAR
| Tool | Why |
|------|-----|
| `download_file` — HTTP download to local path | fetch remote files |
| `archive_create` / `archive_extract` — zip/tar,7z | compression |
| `watch_directory` — file system watcher | monitor for changes |
| `read_file_binary` — read as base64 | binary file handling |
| `check_disk_health` — SMART, error counts | diagnostics |
| `get_recycle_bin` — list recycle bin contents | recovery |

---

## 8. CLIPBOARD — Copy and Paste

### HAVE
- `get_clipboard` — text content
- `set_clipboard` — write text

### NEXT
| Tool | Why |
|------|-----|
| `get_clipboard_image` — read image from clipboard | image copy operations |
| `get_clipboard_files` — read copied file paths | file cut/copy |
| `get_clipboard_formats` — list available formats | clipboard inspection |
| `clear_clipboard` — empty clipboard | security cleanup |

### FAR
| Tool | Why |
|------|-----|
| `clipboard_history` — access clipboard history | multi-paste |
| `set_clipboard_image` — copy image to clipboard | image editing context |
| `set_clipboard_files` — copy files to clipboard | file operations |

---

## 9. AUDIO — Hear and Speak

### HAVE
- `get_volume` / `set_volume` — system volume 0-100
- `set_mute` — toggle mute (sets to 0 or 50)
- `list_audio_devices` — render + capture devices
- `set_default_audio_device` — set default render device

### NEXT
| Tool | Why |
|------|-----|
| `get_microphone_mute` / `set_microphone_mute` | mic control for voice |
| `play_sound` — play a .wav file or system sound | audio feedback |
| `play_beep` — beep at frequency/duration | alerts |
| `get_audio_levels` — current VU meter level | volume visualization |
| `record_audio` — record from microphone to file | voice capture |

### FAR
| Tool | Why |
|------|-----|
| `text_to_speech` — SAPI/WinRT speech synthesis | speak to user |
| `speech_to_text` — WinRT speech recognition | transcribe microphone |
| `get_system_sounds` — list available system sounds | customization |
| `set_audio_device_volume` — per-device volume | granular control |
| `audio_mixer` — per-app volume control | app-specific audio |

---

## 10. TEXT-TO-SPEECH & SPEECH-TO-TEXT — Voice Interaction

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `speak` — TTS via WinRT SpeechSynthesis | voice feedback to user |
| `speak_to_file` — generate speech .wav | save speech output |
| `list_tts_voices` — available voices | selection |
| `set_tts_voice` — choose voice | personalization |

### FAR
| Tool | Why |
|------|-----|
| `transcribe_microphone` — live speech→text | voice commands |
| `transcribe_file` — transcribe audio file | offline transcription |
| `list_speech_languages` — supported recognition languages | multi-lingual |
| `continuous_speech_recognition` — real-time streaming | always-listening mode |

---

## 11. POWER & SYSTEM — Control the Machine

### HAVE
- `shutdown` / `restart` / `sleep` / `hibernate`
- `get_uptime` — time since boot
- `get_idle_time` — time since last user input
- `get_battery` — percentage, charging, AC status
- `get_system_info` — hostname, OS, total/free RAM
- `get_disk_usage` — all drives
- `lock_workstation`

### NEXT
| Tool | Why |
|------|-----|
| `set_screen_power` — turn monitor on/off | power saving |
| `get_power_scheme` — current power plan GUID | power management |
| `set_power_scheme` — high performance / balanced / power saver | performance control |
| `get_computer_name` / `set_computer_name` | system identity |
| `get_timezone` / `set_timezone` | time management |
| `get_system_locale` — current locale | localization awareness |

### FAR
| Tool | Why |
|------|-----|
| `get_cpu_usage` / `get_memory_usage` | performance monitoring |
| `get_installed_updates` — Windows Update status | patch management |
| `get_event_log` — tail system/application logs | diagnostics |
| `get_os_version` — detailed version/build | compatibility |
| `screen_off` / `screen_on` — display power state | energy saving |
| `get_system_uptime_seconds` (already have via duration) | — |

---

## 12. NETWORK — Connect and Communicate

### HAVE
- `get_network_info` — hostname, IPs, DNS, gateway
- `ping` — ICMP reachability

### NEXT
| Tool | Why |
|------|-----|
| `get_wifi_ssid` — connected WiFi network name | connectivity awareness |
| `get_wifi_signal_strength` — RSSI | signal quality |
| `list_network_adapters` — adapters + status | diagnostics |
| `get_active_network_connections` — active TCP/UDP | app network awareness |
| `get_firewall_status` — Windows Defender Firewall state | security check |
| `get_proxy_settings` — current proxy config | internet access |
| `speed_test` — network speed (download/upload) | bandwidth check |

### FAR
| Tool | Why |
|------|-----|
| `set_static_ip` / `set_dhcp` | network configuration |
| `enable_network_adapter` / `disable_network_adapter` | connectivity control |
| `get_public_ip` — external IP | internet presence |
| `connect_to_wifi` / `disconnect_wifi` | wireless management |
| `traceroute` / `nslookup` — network diagnostics | troubleshooting |

---

## 13. REGISTRY — System Configuration

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `registry_read` — read a registry value | system settings |
| `registry_list_keys` — enumerate subkeys | registry browsing |

### FAR
| Tool | Why |
|------|-----|
| `registry_write` / `registry_delete` | system configuration |

---

## 14. ENVIRONMENT — User and System Environment

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_env` — get a single env var | config awareness |
| `list_env` — all env vars | full environment |
| `set_env` — set env var for current process | runtime config |

### FAR
| Tool | Why |
|------|-----|
| `set_env_permanent` — persistent env var | system configuration |
| `get_path` / `set_path` — PATH management | executable lookup |

---

## 15. UI AUTOMATION — Interact with Controls

### HAVE
- `uia_find` — find elements by name, automation_id, control_type → bounding rect + properties
- `uia_get_text` — read text from a UI element
- `uia_invoke` — click/invoke a button or control via Invoke/Toggle/Click pattern

### NEXT
| Tool | Why |
|------|-----|
| `uia_get_focused_control` — AutomationElement for focused element | know what's focused |
| `uia_get_text` — get text from a text control | read text fields |
| `uia_set_text` — set text in a text control | input without keyboard simulation |
| `uia_invoke` — click a button by AutomationId/Name | reliable click on controls |
| `uia_get_children` — list child elements of a window | explore UI structure |

### FAR
| Tool | Why |
|------|-----|
| `uia_select` — select from combobox/listbox | dropdown selection |
| `uia_get_table` — read table/grid content | data extraction |
| `uia_scroll` / `uia_select_tab` | control interaction |
| `uia_wait_for_element` — wait for control by AutomationId | robust automation |
| `uia_get_bounding_rect` — get element screen rect | positioning |

---

## 16. TASKBAR & START MENU — Shell Interaction

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `taskbar_search` — type in taskbar search box | quick access |
| `start_menu_open` / `start_menu_close` | start menu access |
| `get_pinned_taskbar_items` — list pinned apps | quick launch awareness |

### FAR
| Tool | Why |
|------|-----|
| `pin_to_taskbar` / `unpin_from_taskbar` | taskbar customization |
| `get_jump_list` — right-click menu for taskbar item | recent files, tasks |
| `action_center_open` / `action_center_close` | notification center |
| `clock_flyout` — open calendar/clock | date/time access |
| `system_tray_open` — show hidden icons | tray access |

---

## 17. NOTIFICATIONS — Alert the User

### HAVE
- `show_notification` — blocking MessageBox

### NEXT
| Tool | Why |
|------|-----|
| `toast_notify` — Windows Toast notification (non-blocking) | modern notifications |
| `toast_dismiss` — dismiss a notification | clean up |

### FAR
| Tool | Why |
|------|-----|
| `get_notification_history` — read notification center | missed alerts |

---

## 18. USB & DEVICES — Peripheral Management

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `list_usb_devices` — connected USB devices | peripheral awareness |
| `eject_usb` — safely remove USB drive | safe removal |

### FAR
| Tool | Why |
|------|-----|
| `list_printers` / `get_default_printer` | printing awareness |
| `get_connected_displays` — external monitors | display config |
| `list_bluetooth_devices` — paired BT devices | wireless peripherals |

---

## 19. TIME & DATE — System Clock

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_system_time` — current date/time | time awareness |
| `get_timezone` | timezone awareness |

### FAR
| Tool | Why |
|------|-----|
| `set_system_time` | time synchronization |
| `get_uptime_display` — human-readable uptime | user communication |

---

## 20. ACCESSIBILITY — Ease of Access

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_high_contrast` — is high contrast mode on | accessibility awareness |
| `get_magnifier_state` — Magnifier on/off + zoom level | support magnification |
| `get_narrator_state` — is Narrator running | screen reader awareness |

### FAR
| Tool | Why |
|------|-----|
| `set_high_contrast` | toggle accessibility
| `get_cursor_size` / `get_text_size` | scaling awareness |

---

## 21. REMOTE SESSION — Terminal Services

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_session_type` — console vs RDP | know if remote |
| `get_active_sessions` — list terminal sessions | multi-user awareness |

### FAR
| Tool | Why |
|------|-----|
| `disconnect_session` / `logoff_session` | session management |

---

## 22. SECURITY & IDENTITY

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `is_admin` — check if running as admin | permission awareness |
| `get_username` — current user | identity |
| `get_user_sid` — user security identifier | identity |
| `get_user_groups` — group membership | permission awareness |

### FAR
| Tool | Why |
|------|-----|
| `uac_prompt` — trigger elevation dialog | admin escalation |
| `get_logged_in_users` — active users | multi-user |
| `get_bitlocker_status` — drive encryption | security status |
| `get_defender_status` — antivirus status | security awareness |

---

## 23. SCREEN — Display Hardware

### HAVE
- `list_displays` — name, res, position, primary
- `get_screen_size` — virtual width/height
- `get_screen_dpi` — per-monitor DPI + scale %
- `get_display_modes` — all available resolutions, refresh rates, color depths per monitor

### NEXT
| Tool | Why |
|------|-----|
| `get_display_orientation` — landscape/portrait | rotation |
| `get_virtual_screen_bounds` — total spanning rect | multi-monitor layout |

### FAR
| Tool | Why |
|------|-----|
| `set_display_resolution` — change resolution | display config |
| `set_display_orientation` | rotation |
| `identify_displays` — flash display number | user communication |

---

## 24. WINDOWS SHELL — OS Integration

### HAVE
- `open_file_explorer`
- `open_file_location`
- `open_url`

### NEXT
| Tool | Why |
|------|-----|
| `open_run_dialog` — Win+R | quick command |
| `open_command_prompt` / `open_powershell` | terminal access |
| `open_control_panel` — specific applet | system settings |
| `open_settings_app` — Windows Settings page | modern settings |
| `get_default_browser` | URL handling awareness |

### FAR
| Tool | Why |
|------|-----|
| `open_recycle_bin` | file recovery |
| `empty_recycle_bin` | cleanup |
| `open_task_manager` | process management |
| `get_association` — file extension→app | default program |

---

## 25. CHAINED / COMPOSITE — Higher-Level Abstractions

### HAVE
- `find_text_and_click` — OCR + click at text
- `wait_for_text` — poll OCR until text appears
- `click_menu_item` — within a window by title
- `launch_and_wait` — launch app + wait for window
- `screenshot_element` — screenshot a window
- `hover` — move + wait
- `type_and_submit` — type + Enter
- `select_all_and_type` — Ctrl+A + type

### NEXT
| Tool | Why |
|------|-----|
| `find_text_and_right_click` | right-click on OCR text |
| `find_text_and_double_click` | double-click on OCR text |
| `find_all_text` — return all matches | multi-target |
| `fill_form` — map field labels→values | form filling |
| `drag_and_drop_file` — drag file from Explorer | file operation |
| `download_and_open` — download URL then open | remote file workflow |
| `scroll_until_text_visible` — scroll + OCR poll | infinite scroll |
| `click_all_matching_text` — click every occurrence | bulk dismissals |
| `dismiss_all_notifications` — find+close | cleanup |
| `type_password` — type from env var | secure input |

### FAR
| Tool | Why |
|------|-----|
| `login_to_website` — URL + credentials → logged in page | web automation |
| `navigate_file_dialog` — Open/Save dialog → select file | file dialog handling |
| `complete_wizard` — button-by-button wizard completion | installer automation |
| `install_app` — download + run installer + accept EULA | app installation |
| `ocr_and_chain` — general OCR → tool call pipeline | agent-in-a-box |

---

## 26. DEBUGGING & DIAGNOSTICS

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `log_memory` — current heap usage | memory monitoring |
| `benchmark_screenshot` — capture timing | performance baseline |
| `debug_ocr` — raw OCR output with confidence | OCR quality assessment |

### FAR
| Tool | Why |
|------|-----|
| `save_screenshot_to_file` — save debug screenshot | examination |
| `save_all_state` — windows + processes + disks | snapshot |

---

## Summary

| Domain | HAVE | NEXT | FAR | Total Possible |
|--------|------|------|-----|---------------|
| Vision | 8 | 5 | 7 | 20 |
| Mouse | 6 | 6 | 5 | 17 |
| Keyboard | 10 | 5 | 6 | 21 |
| Windows | 13 | 9 | 6 | 28 |
| Virtual Desktops | 0 | 6 | 2 | 8 |
| Processes | 4 | 7 | 5 | 16 |
| File System | 3 | 9 | 6 | 18 |
| Clipboard | 2 | 4 | 3 | 9 |
| Audio | 4 | 5 | 6 | 15 |
| TTS / STT | 0 | 4 | 4 | 8 |
| Power & System | 11 | 8 | 6 | 25 |
| Network | 2 | 8 | 5 | 15 |
| Registry | 0 | 2 | 2 | 4 |
| Environment | 0 | 3 | 2 | 5 |
| UI Automation | 3 | 4 | 5 | 12 |
| Taskbar / Start | 0 | 3 | 5 | 8 |
| Notifications | 1 | 2 | 1 | 4 |
| USB & Devices | 0 | 2 | 3 | 5 |
| Time & Date | 0 | 2 | 2 | 4 |
| Accessibility | 0 | 3 | 2 | 5 |
| Remote Session | 0 | 2 | 2 | 4 |
| Security & Identity | 0 | 4 | 4 | 8 |
| Screen (HW) | 6 | 2 | 3 | 11 |
| Windows Shell | 3 | 5 | 3 | 11 |
| Chained | 10 | 11 | 5 | 26 |
| Debugging | 0 | 3 | 2 | 5 |
| Memory & ML | 10 | 5 | 5 | 20 |
| **TOTAL** | **108** | **117** | **101** | **326** |

## Strategy

1. **Build out NEXT items** — these are straightforward and high value (another ~117 tools)
2. **Error wrapping audit** — remaining Slice 4 item for consistent error feedback across all tools
3. **Cross-platform interface** — Linux/macOS stubs for portable task definitions
4. **User-configurable tool subsets** — allow users to load only the tool groups they need per agent
5. **ML model improvement** — fix YOLO11n opset 22 incompatibility, explore UI-specific fine-tuning
