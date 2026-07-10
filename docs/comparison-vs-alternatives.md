# Comparison: go-mcp-computer-use vs Alternatives

A survey of the computer-use agent landscape as of mid-2026, with detailed deep-dive comparisons against the most relevant projects.

---

## 1. Landscape Overview

| Project | Lang | MCP | Desktop | ONNX/ML | Training | Memory | Tools | Stars | Open Source |
|---------|------|-----|---------|---------|---------|--------|-------|-------|-------------|
| **[go-mcp-computer-use](https://github.com/coff33ninja/go-mcp-computer-use)** | Go | ✅ | ✅ Win | ✅ YOLO11+MobileNet | ✅ auto-collect | ✅ SQLite FTS | 134 | — | ✅ MIT |
| **[Cua](https://github.com/trycua/cua)** | Py/HTML | ✅ | ✅ Win/Mac/Linux | ❌ | ❌ | ❌ | SDK | 19.4k | ✅ |
| **[Agent-S](https://github.com/simular-ai/Agent-S)** | Python | ❌ | ✅ Win/Mac/Linux | ❌ | ❌ | ✅ in-context RL | SDK | 12k | ✅ |
| **[Bytebot](https://github.com/bytebot-ai/bytebot)** | TypeScript | ❌ | ❌ Linux container | ❌ | ❌ | ❌ | SDK | 11.1k | ✅ |
| **[MS Magentic-UI](https://github.com/microsoft/magentic-ui)** | C# | ❌ | ✅ Win | ❌ | ❌ | ❌ | SDK | 9.9k | ✅ |
| **[Windows-MCP](https://github.com/CursorTouch/Windows-MCP)** | Python | ✅ | ✅ Win | ❌ | ❌ | ❌ | ~30 | 6k | ✅ MIT |
| **[DesktopCtl](https://github.com/yaroshevych/desktopctl)** | Rust | ❌ | ✅ Win/Mac/Linux | ✅ GPU OCR | ❌ | ❌ | CLI | — | ✅ MIT |
| **[Windows MCP Server](https://github.com/sbroenne/mcp-windows)** | C# | ✅ | ✅ Win | ❌ | ❌ | ❌ | ~15 | 57 | ✅ MIT |
| **[Computer Control MCP](https://github.com/AB498/computer-control-mcp)** | Python | ✅ | ✅ Win/Mac/Linux | ✅ RapidOCR+ONNX | ❌ | ❌ | ~10 | 150 | ✅ MIT |
| **[Anthropic Computer Use](https://docs.anthropic.com/en/docs/agents-and-tools/computer-use)** | — | ❌ | ❌ cloud VM | ✅ Claude vision | ❌ | ❌ | API | — | ❌ |
| **[OpenAI CUA](https://openai.com/index/computer-using-agent/)** | — | ❌ | ❌ cloud VM | ✅ GPT-4o vision | ❌ | ❌ | API | — | ❌ |
| **[Browser Use](https://github.com/browser-use/browser-use)** | Python | ❌ | ❌ browser only | ❌ | ❌ | ❌ | SDK | — | ✅ |

---

## 2. vs Other MCP Servers for Windows

### go-mcp-computer-use vs [Windows-MCP](https://github.com/CursorTouch/Windows-MCP) (CursorTouch)

Windows-MCP is the closest direct competitor — both are MCP servers for Windows automation. Key differences:

| Dimension | go-mcp-computer-use | Windows-MCP |
|-----------|-------------------|-------------|
| **Language** | Go (static binary, ~16 MB) | Python (requires Python + deps) |
| **OCR** | Native WinRT COM (2-8x faster than PowerShell) + PowerShell fallback | PowerShell OCR via pyautogui |
| **ML detection** | YOLO11n + MobileNetV3 ONNX inference | None |
| **Memory cache** | SQLite FTS5 with TTL | None |
| **Training pipeline** | Auto-screenshots per action, categorized, YOLO export | None |
| **UI Automation** | Native COM UIAutomation (IUIAutomation) | UIA via Python |
| **Tool count** | 132 | ~30 |
| **Chained automation** | if/else, loops, polling, verify, error handling | None |
| **Browser automation** | No (basic find_text_and_click) | Snapshot + DOM mode for Chrome/Edge/Firefox |
| **Security** | `tool_denylist` config-based enable/disable (stdio/local only) | Bearer token, IP allowlist, TLS, OAuth 2.0 + PKCE, CORS, per-tool enable/disable |
| **Network transport** | stdio | stdio, SSE, streamable HTTP |
| **State management** | Reflection engine, adaptive engine, statistical priors | None |

**Verdict**: Windows-MCP has better security and network support. go-mcp-computer-use wins on ML/memory/training depth. They're complementary — Windows-MCP for secure remote access, go-mcp-computer-use for agent learning and adaptation.

### go-mcp-computer-use vs [Windows MCP Server](https://github.com/sbroenne/mcp-windows) (sbroenne)

Windows MCP Server uses a fundamentally different approach: **UI Automation tree access** instead of pixel-based interaction. It finds buttons by automation name/ID, not by screenshot/OCR. This makes it faster and more reliable for standard Windows apps, but useless for games, custom controls, or any app without UIA metadata.

| Approach | go-mcp-computer-use | Windows MCP Server |
|----------|-------------------|-------------------|
| **Element finding** | Pixels (screenshot → ONNX/OCR) | UIA tree (name/ID/type) |
| **Screenshot needed?** | Yes (every action) | No (for UIA elements) |
| **Works with games** | ✅ Yes (pixel-based) | ❌ No |
| **DPI independence** | ❌ Requires DPI normalization | ✅ Automatic |
| **Theme independence** | ❌ Retrains on theme change | ✅ Automatic |
| **Fallback** | Memory → ONNX → OCR | UIA → mouse/keyboard (coordinates) |

**Verdict**: go-mcp-computer-use covers the full spectrum (pixel + UIA + OCR). Windows MCP Server is more reliable for standard apps but has no fallback for apps without UIA metadata.

### go-mcp-computer-use vs [DesktopCtl](https://github.com/yaroshevych/desktopctl)

DesktopCtl is a Rust CLI tool for desktop control, not an MCP server. It uses GPU-accelerated text recognition and exposes a deterministic CLI interface. Like go-mcp-computer-use, it supports "bring your own AI."

| Feature | go-mcp-computer-use | DesktopCtl |
|---------|-------------------|------------|
| **Interface** | MCP protocol (132 tools) | CLI commands |
| **Vision** | ONNX YOLO11 + MobileNetV3 | GPU-accelerated OCR (no object detection) |
| **Privacy** | Screenshots shared with AI agent | Screenshots stay local by default |
| **Transport** | stdio (TCP optional) | CLI pipes |
| **Language** | Go | Rust |
| **Unique** | Training pipeline, memory, priors, 132 tools | `screen tokenize` for structured UI tokens |

**Verdict**: DesktopCtl prioritizes privacy (screenshots not shared by default) and structured token output. go-mcp-computer-use prioritizes depth (training, memory, 132 tools).

---

## 3. vs Agent Frameworks

### go-mcp-computer-use vs [Cua](https://github.com/trycua/cua) (trycua, 19.4k★)

Cua is the largest open-source computer-use project. It provides sandboxes (VM/container images) for any OS plus drivers for background desktop control. It includes Cua Bench for standardized benchmarks and Lume for macOS VM management.

| Dimension | go-mcp-computer-use | Cua |
|-----------|-------------------|-----|
| **Scope** | Single-server MCP toolset | Full infrastructure: sandboxes + SDK + benchmarks + drivers |
| **OS support** | Windows only | Linux, macOS, Windows, Android (via VM/container) |
| **Background control** | Foreground only (cursor capture) | ✅ Background (no cursor steal) via Cua Drivers |
| **Sandboxing** | None (runs on host OS) | QEMU VMs + Docker containers, local or cloud |
| **Benchmarks** | None published | Cua Bench (standardized eval suite) |
| **RL training** | None | RL environments for model training |
| **MCP support** | Native MCP server | Cua Drivers expose MCP server |
| **Memory** | SQLite FTS5 with TTL | None |
| **Training pipeline** | Auto-collected screenshots, YOLO export | None (RL-focused instead) |
| **Install size** | ~16 MB binary + ~11 MB model | Multi-GB VMs + Python SDK |

**Verdict**: Cua is an infrastructure platform for building and evaluating computer-use agents. go-mcp-computer-use is a focused MCP tool server for Windows. Cua's background control and VM sandboxing are features go-mcp-computer-use lacks entirely. go-mcp-computer-use's memory cache and training pipeline have no Cua equivalent.

### go-mcp-computer-use vs [Agent-S](https://github.com/simular-ai/Agent-S) (simular-ai, 12k★)

Agent-S is an open agentic framework focused on in-context reinforcement learning — it learns from past mistakes within a session. It supports all major OS platforms.

| Feature | go-mcp-computer-use | Agent-S |
|---------|-------------------|---------|
| **Learning approach** | Statistical priors (cross-session) + memory cache | In-context RL (within session) |
| **Memory** | SQLite persistent (FTS5, TTL) | Episodic (session-scoped) |
| **Platform** | Windows only | Multi-platform |
| **MCP** | Native MCP server | Not MCP-native |
| **GUI agent** | MCP tool interface | Agent framework with planning/grounding |

**Verdict**: Agent-S's in-context RL is orthogonal to go-mcp-computer-use's persistent priors. An agent using go-mcp-computer-use could theoretically implement Agent-S-style RL at the LLM layer while benefiting from go-mcp-computer-use's persistent memory.

---

## 4. vs Cloud APIs

### go-mcp-computer-use vs [Anthropic Computer Use](https://docs.anthropic.com/en/docs/agents-and-tools/computer-use) / [OpenAI CUA](https://openai.com/index/computer-using-agent/)

Both Anthropic and OpenAI offer computer-use as cloud APIs — the AI model sees screenshots and returns mouse/keyboard commands. No local ML, no training pipeline, no memory.

| Feature | go-mcp-computer-use | Anthropic / OpenAI CUA |
|---------|-------------------|----------------------|
| **Where AI runs** | Client-side (your LLM via MCP) | Cloud (Anthropic/OpenAI servers) |
| **Screenshots** | Stay local (sent to your LLM) | Sent to cloud API |
| **ML detection** | Local ONNX (YOLO11) | Cloud vision model |
| **Latency** | OCR: 500-2000ms, ONNX: 100-200ms | API round-trip: 2-10s |
| **Training** | Auto-collect + export for fine-tuning | None exposed |
| **Memory** | Persistent SQLite | None |
| **Offline capable** | Yes (all inference local) | No (requires API) |
| **Cost** | Free (own compute) | Per-token API pricing |
| **Model choice** | Any LLM (bring your own) | Vendor-locked |

**Verdict**: Cloud APIs give you a built-in vision model but lock you into their ecosystem. go-mcp-computer-use gives you local infrastructure and lets you choose any LLM — but you need to provide the model.

### go-mcp-computer-use vs [Browser Use](https://github.com/browser-use/browser-use)

Browser Use is the leading open-source browser automation framework. It achieves 89.1% on WebVoyager but is browser-only — no desktop app control.

| Feature | go-mcp-computer-use | Browser Use |
|---------|-------------------|-------------|
| **Scope** | Full desktop (any app) | Browser only |
| **Integration** | MCP protocol | Python SDK + Playwright |
| **Element finding** | ONNX + OCR + UIA + template match | DOM selectors + accessibility tree |
| **Desktop apps** | ✅ All Win32/UWP apps | ❌ Browser only |
| **Benchmark** | None published | 89.1% on WebVoyager (586 tasks) |
| **CAPTCHA** | Not handled | Manual configuration |

**Verdict**: Not direct competitors. Browser Use is better at web tasks (DOM access is faster and more reliable than pixel-based). go-mcp-computer-use is the only option for non-browser desktop apps.

---

## 5. vs Microsoft Windows Recall

| Dimension | Windows Recall | go-mcp-computer-use |
|-----------|---------------|---------------------|
| **Why it exists** | User's "photographic memory" — find anything you've seen on your PC via semantic search | MCP server — gives AI agents mouse, keyboard, screen, and system control |
| **Who uses it** | End users (opt-in on Copilot+ PCs) | AI agents/developers via MCP protocol |
| **Primary interaction** | GUI app with timeline + natural-language search box | Function calls (134 MCP tools) from an AI agent |
| **Snapshot consumer** | The user themselves, later | An AI model, in real-time during task execution |
| **Captures because** | Time passes (periodic, every ~3-5 seconds) | The AI does something (click, type, navigate) or watcher polls |

This is the most important distinction. Recall is a **read-only archival tool for humans**. go-mcp-computer-use is a **read-write remote control for AI**. One logs what happened; the other makes things happen.

See [`guides/accessibility.md`](guides/accessibility.md) for the assistive technology context and dual-use considerations.

---

## 7. Snapshot Collection

| Aspect | Windows Recall | go-mcp-computer-use |
|--------|---------------|---------------------|
| **Trigger** | Periodic (every ~3-5 seconds) + on screen content change | On AI action (click, type, navigate) + optional watcher (every N seconds) |
| **What is captured** | Active window (full screen possible) | Full screen |
| **Storage format** | AVIF (≈170-220 KB per frame) encrypted with AES-256-GCM | PNG (unencrypted) in categorized folders |
| **Storage location** | `%LOCALAPPDATA%\CoreAIPlatform\UKP\Recall\V1\` | `%APPDATA%\go-mcp-computer-use\training\raw\{category}\` |
| **Metadata DB** | SQLite SEE (`ukg.db`) with AES-256-GCM encryption (VBS Enclave) | SQLite (`samples.db`) with WAL mode, no encryption |
| **Retention** | Configurable (30/60/90/180 days) | Configurable via `retention_days` (0=disabled, auto-prunes samples older than N days) |
| **Collection rate** | Continuous — every ~3-5s, 24/7 while user is active | On-demand — only when AI acts or watcher is enabled |
| **Per-action context** | Window title, PID, monitor ID, dwell time | Task prompt, window title, ONNX detections, OCR text, signal level |

### Collection frequency comparison

Recall captures ~720-1200 screenshots per hour of user activity. go-mcp-computer-use captures 1 screenshot per AI tool call plus optional watcher frames. For a typical AI session with 50 tool calls over 10 minutes, that's ~50 action frames + watcher frames (if enabled).

---

## 8. AI / ML Pipeline

| Component | Windows Recall | go-mcp-computer-use |
|-----------|---------------|---------------------|
| **On-device AI** | Windows Copilot Runtime (NPU-accelerated) | ONNX Runtime (CPU via `onnxruntime_go`) |
| **Model** | Multiple proprietary models (Screen Region Detector, OCR, Image Encoder, Audio Encoder, Natural Language Parser) | YOLO11n (80-class COCO object detection) |
| **OCR** | Local Tesseract-based engine via Copilot Runtime | Windows.Media.Ocr (native WinRT COM) |
| **Semantic search** | DiskANN vector indexes — 4 image embedding types + 4 text embedding types, 512-D float32 vectors | None (no semantic indexing) |
| **Training** | Pre-trained Microsoft models, no user-data training | Self-learning statistical priors (element frequency + position per window), optional YOLO dataset export for external training |
| **Sensitive content** | Microsoft Purview Exact Data Match filter + sensitivity label detection | None (raw screenshots contain whatever is on screen) |
| **What the AI understands** | Everything visible: text, images, layout, window titles | COCO objects: person, laptop, cell_phone, book, tv, etc. |

Recall's AI pipeline is dramatically more sophisticated — it builds a full semantic index of everything on screen, enabling natural-language queries. go-mcp-computer-use only detects 80 generic COCO object classes.

---

## 9. Security Architecture

| Layer | Windows Recall | go-mcp-computer-use |
|-------|---------------|---------------------|
| **Encryption at rest** | AES-256-GCM per snapshot + encrypted vector DB | None — plain PNG files + plaintext SQLite |
| **Key management** | TPM 2.0 + Windows Hello ESS biometric → ECDH P-384 → VBS Enclave sealed key | None |
| **Memory isolation** | VBS Enclave (VTL1) — hypervisor-level isolation from kernel and admin | None — runs in user mode process |
| **Process protection** | `aihost.exe` as Protected Process Light (PPL) | Standard user process |
| **Access control** | Windows Hello biometric every session + periodic re-auth | No access control (open to any MCP client) |
| **Auth bypass research** | TotalRecall / TotalRecall Reloaded — same-user malware can extract decrypted data from `aihost.exe` memory via COM | No auth needed — any MCP client that reaches the server has full control |
| **In-transit** | Local only (no network path) | MCP stdio (local) or TCP (if configured, no encryption) |

### Critical difference

Recall's security architecture is **defense-in-depth** designed to resist malware on the same machine. It uses enterprise-grade isolation (VBS Enclaves, TPM, PPL) because it stores a complete recording of the user's digital life.

go-mcp-computer-use has **no encryption, no isolation, no access control**. It's designed as a local tool for AI agents, not as a long-term archival store. The security model is: "don't expose the MCP server to untrusted clients." It's a tool, not a vault.

---

## 10. Privacy Controls

| Control | Windows Recall | go-mcp-computer-use |
|---------|---------------|---------------------|
| **Opt-in required** | Yes — off by default, user must enable | No — starts collecting when AI agent connects |
| **Disable collection** | Settings toggle or GPO (`DisableSnapshots = 1`) | `set_config training_enabled: false` or config file |
| **Disable watcher** | N/A (always-on when enabled) | `set_config watcher_enabled: false` or `onnx_watch_stop` |
| **App/website filtering** | Per-app and per-website exclusion list | None |
| **Sensitive content filtering** | Purview-based: credit cards, passwords, PII detected and excluded | None |
| **Private browsing** | Excluded by default (supported browser detection) | None |
| **Delete data** | Settings → delete snapshots, configurable retention | `training_cleanup_noise` (manual) or `retention_days` auto-prune |
| **Export data** | No built-in export | `export_yolo_dataset` to dump all images + labels |
| **Audit** | Windows Event Log + diagnostic data | `training_stats` for counts and disk usage |

Recall's privacy controls are **mature and granular** — app filtering, sensitive content redaction, private browsing detection, configurable retention. Microsoft learned from the June 2024 privacy backlash.

go-mcp-computer-use's controls are **basic but functional** — `set_config` to stop collection, `tool_denylist` to restrict tools, `retention_days` for auto-pruning, cleanup to delete noise, export to inspect data. No content filtering, no app exclusions.

---

## 11. Storage Comparison (Real-World Estimates)

### Recall (1 hour of active use)

| Storage item | Size |
|-------------|------|
| 720 AVIF snapshots (3-second intervals) | ~120-150 MB |
| SQLite db (ukg.db) with OCR + metadata | ~50-100 MB |
| DiskANN vector indexes (8 embedding types) | ~200-400 MB |
| **Total per hour** | **~370-650 MB** |
| **Total per 8-hour day** | **~3-5 GB** |

### go-mcp-computer-use (1 hour of AI interaction)

| Storage item | Size |
|-------------|------|
| ~50 action snapshots (PNG, ~200-500 KB each) | ~10-25 MB |
| Watcher frames if enabled (720 at 5s, ~200 KB each as PNG) | ~140 MB |
| SQLite db (samples.db) | ~1-5 MB |
| **Total per hour (watcher off)** | **~11-30 MB** |
| **Total per hour (watcher on)** | **~150-170 MB** |

> Note: Recall uses AVIF (~170-220 KB/frame) while go-mcp-computer-use uses PNG (~200-500 KB/frame). AVIF is ~2x more efficient for screen content. Recall also stores 8 vector embeddings per frame which dominates storage.

---

## 12. Threat Model Comparison

### Windows Recall threat model

From Microsoft's [security architecture blog](https://blogs.windows.com/windowsexperience/2024/09/27/update-on-recall-security-and-privacy-architecture/) and third-party analysis:

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Another user on same PC reads snapshots | VBS Enclave + TPM-sealed keys + Windows Hello auth | Mitigated |
| Admin/malware reads disk directly | Disk encryption (BitLocker) + per-file AES-256-GCM | Mitigated at rest; unmitigated after auth (TotalRecall Reloaded) |
| Remote attacker exfiltrates DB | Local-only storage, no network path | Mitigated |
| Kernel-level malware reads enclave memory | Hypervisor isolation (VBS) | Mitigated at VTL1 boundary |
| Malware in user context reads during auth session | PPL process + COM auth timeout | Partially mitigated (TotalRecall Reloaded extracts from `aihost.exe`) |
| Physical theft of device | BitLocker + TPM + Windows Hello | Mitigated |

### go-mcp-computer-use threat model

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Another user reads training data | No mitigation — plain PNG files on disk | **Unmitigated** |
| Malware reads screenshots | No encryption — plain files + plaintext DB | **Unmitigated** |
| Remote attacker connects to MCP server | Should not expose over network | Operational control |
| AI agent misuses tools (shutdown, kill, launch) | All capabilities are intentionally exposed to AI | By design |
| Malware writes to training store | No integrity checks | **Unmitigated** |
| Physical theft of device | File-level access — no encryption at rest | **Unmitigated** |

### Key takeaway

Recall is designed to **survive a compromised OS** (VBS Enclave isolates from kernel). go-mcp-computer-use is designed for **convenience in a trusted environment**. If an attacker has user-level access to the machine, go-mcp-computer-use's training data is trivially readable. Recall's data would require additional exploitation steps.

---

## 13. Summary

| | Windows Recall | go-mcp-computer-use |
|---|---------------|---------------------|
| **Launched** | April 2025 (GA), re-architected after June 2024 privacy backlash | June 2026 (active development, v0.2.9) |
| **Hardware required** | Copilot+ PC (Qualcomm Snapdragon X / AMD Ryzen AI / Intel Core Ultra with NPU) | Any Windows 10/11 x64 |
| **AI chip required** | NPU + CPU + GPU | CPU only |
| **Software dependencies** | Windows 11 24H2+, Copilot Runtime | Go binary (CGO for ONNX via Zig cc) |
| **Binary size** | Part of Windows, ~hundreds of MB of AI models | ~16 MB (Go binary + Zig C runtime) + ~11 MB (YOLO model) |
| **Purpose** | Memory augmentation for humans | Remote control for AI agents |
| **Data sensitivity** | FULL RECORD of everything you do | Screenshots of AI actions + detected objects |
| **Encryption** | Enterprise-grade (TPM + VBS + AES-256-GCM) | None |
| **Granularity of control** | Per-app, per-website, content filtering, retention | Global on/off, watcher on/off/interval, tool denylist, retention pruning |
| **Semantic search** | Yes — natural language across all history | No — only 80-class YOLO labels |
| **Open source** | No (proprietary Microsoft) | Yes (MIT) |
| **Portability** | Windows 11 Copilot+ only | Any Windows 10/11, Go cross-compilable |
| **Use case overlap** | Finding something you saw yesterday | AI clicking buttons and reading screens |

---

## 14. What Each Can Learn From the Other

### go-mcp-computer-use could adopt from Recall

| Feature | Why |
|---------|-----|
| **Sensitive content filtering** | Prevent AI from capturing passwords, financial data, PII in training data |
| **Encryption at rest** | Basic DPAPI- or AES-based file encryption for stored screenshots |
| **App/URL filtering** | Exclude known sensitive apps (browsers in private mode, password managers) from auto-save |
| ~~**Retention policies**~~ | ~~Auto-prune samples older than N days to bound disk usage~~ — **done v0.2.37** (`retention_days`) |
| **Semantic indexing** | Enable the AI to search past screenshots for context ("what did I see in that window earlier?") |
| **Lock-on-auth** | Require Windows Hello or a session token to access training data |

### Windows Recall could adopt from go-mcp-computer-use

| Feature | Why |
|---------|-----|
| **Open source** | Let the community audit the VBS enclave code and privacy controls |
| **Per-action context** | Store the action that triggered a snapshot (click, type, etc.) for richer recall |
| **Tool-based control** | Expose settings programmatically (not just GUI/GPO) for automation |
| **Multi-platform** | Work on non-Copilot+ PCs and Windows 10 |
| **MCP integration** | Let AI agents search and interact with recall history programmatically |

---

## 15. Verdict

**Windows Recall** is a polished, enterprise-grade memory prosthesis for humans. It trades hardware requirements (NPU, TPM 2.0, Windows 11) for deep integration, semantic search, and defense-in-depth security. The re-architecture after the June 2024 disclosure was thorough — the cryptography and VBS isolation are genuinely well-designed. The remaining attack surface (TotalRecall Reloaded extracting data from `aihost.exe` post-auth) is a fundamental tension between usability and security that no system has fully solved.

**go-mcp-computer-use** is an open-source, lightweight, portable tool for AI agents to control Windows. It makes no security guarantees about stored data because data persistence is incidental to its purpose — the primary goal is real-time computer control, not archival. The training pipeline and statistical priors exist to make the AI agent more effective, not to preserve history.

**They could coexist**: An AI agent using go-mcp-computer-use could, in theory, query Windows Recall's semantic index via the Recall WinRT API to understand what the user was doing before the AI took over. But their threat models and design philosophies are fundamentally different — one is a vault, the other is a toolbox.

> **Bottom line:** If you want to search your past activity, use Recall. If you want an AI to control your computer, use go-mcp-computer-use. If you're worried about the screenshots, Recall's encryption is stronger but go-mcp-computer-use gives you simpler controls to just stop collecting.
