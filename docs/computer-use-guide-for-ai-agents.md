# Computer Use: A Guide for AI Agents & Prompt Engineers

> How humans interact with computers through peripherals — and how AI agents bridge that gap via MCP

---

## Part 1: The Human-Peripheral Interface

Humans interact with computers through **peripherals** — external devices that bridge intent to action.

### The Three Core Channels

| Channel | Peripherals | What It Does |
|---------|-------------|--------------|
| **Input** | Keyboard, mouse, touch, microphone, camera, joystick, stylus, eye tracker, BCI | Translates human physical action into digital signals |
| **Output** | Monitor, speakers, printer, haptics, braille display | Translates digital state into human-senseable information |
| **Storage** | SSD, HDD, USB drive, floppy disk | Persists digital state across sessions |

### Evolution Timeline

```
1940s ── Punch cards (batch input, no feedback loop)
1960s ── Keyboard + command line (text in → text out, expert-only)
1968 ── The Mother of All Demos (mouse, GUI, video conferencing)
1980s ── GUI revolution (Mac, Windows — click instead of type)
1990s ── USB plug-and-play, multimedia peripherals
2000s ── Touchscreens, smartphones, gesture recognition
2010s ── Voice assistants (Siri, Alexa), eye tracking
2020s ── Brain-computer interfaces (BCI), computer-use agents (CUAs)
2025+ ── AI agents operate the same peripherals humans do
```

### Why Peripherals Matter

The critical insight: **peripherals are the bottleneck**. A human can think at ~400 words/min but type at ~40 wpm. The mouse converts spatial thinking into pointer movement. Voice converts speech into text. Each peripheral trades off **bandwidth vs. precision vs. learnability**.

This is exactly why **computer-use agents (CUAs)** matter — they operate the same peripheral layer, which means they can use ANY software ever built, not just software with an API.

---

## Part 2: The MCP Bridge — How AI Agents Use Computers

### What is MCP?

The **Model Context Protocol (MCP)** is an open standard (Anthropic, 2024) that standardizes how AI models connect to external tools and data sources. Instead of every AI having bespoke tool integrations, MCP provides a universal protocol:

```
┌─────────────┐     MCP Protocol     ┌──────────────┐
│  AI Model   │ ◄──────────────────► │  MCP Server  │
│  (Client)   │   tool discover/     │  (computer)  │
│             │   call/result        │              │
└─────────────┘                      └──────────────┘
                                            │
                                    ┌───────┴───────┐
                                    │  Peripherals   │
                                    │  (mouse, kbd,  │
                                    │   screen, etc) │
                                    └───────────────┘
```

### The go-mcp-computer-use Server

The `mcp-server.exe` from [coff33ninja/go-mcp-computer-use](https://github.com/coff33ninja/go-mcp-computer-use) exposes **108 tools** that mirror every human peripheral capability on Windows:

#### Vision (Human: eyes → screen)
- `screenshot` — capture what's on screen (like human looking)
- `ocr` — read text from screen (like human reading)
- `find_image` — locate visual elements (like human searching)
- `get_pixel_color` — check exact color at a point
- `onnx_detect` — ML-based UI element detection (like human recognizing buttons)

#### Pointing (Human: hand → mouse)
- `click` — left/right click at coordinates
- `move_mouse` — reposition cursor
- `scroll` — scroll up/down
- `drag` — click-hold-and-move (for selection, drag-and-drop)
- `hover` — position cursor for tooltips

#### Text Input (Human: fingers → keyboard)
- `type` — type text character by character
- `key_press` — key combos (Ctrl+C, Alt+Tab, Win+D)
- `type_and_submit` — type then press Enter (for search bars, forms)
- `select_all_and_type` — select all existing text, replace it

#### Window Management (Human: spatial awareness)
- `list_windows` — see all open windows with handles, titles, PIDs
- `focus_window` — bring a window to foreground
- `get_window_state` — position, size, visibility, min/max state
- `move_window` — reposition/resize windows
- `close_window` — terminate a window

#### System Awareness (Human: senses)
- `get_system_info` — hostname, OS, RAM
- `get_disk_usage` — free space per drive
- `get_network_info` — IPs, DNS, gateway
- `get_battery` — battery percentage, charging status
- `list_processes` — all running processes with PIDs

### The Perceive-Reason-Act Loop

An AI agent using computer use follows this loop — exactly mirroring human interaction:

```
┌─────────────────────────────────────────────────────────┐
│                    THE CUA LOOP                          │
│                                                         │
│  1. PERCEIVE ── screenshot + OCR → understand state     │
│  2. REASON   ── decide what action to take next        │
│  3. ACT      ── click / type / scroll / key_press      │
│  4. OBSERVE  ── screenshot + OCR → verify result        │
│  5. REPEAT   ── loop until goal is achieved             │
└─────────────────────────────────────────────────────────┘
```

---

## Part 3: Prompt Engineering for Computer-Use Agents

### Core Prompting Principles

#### 1. Describe What You See, Then Decide What to Do

Bad: `"Search Google for cat memes"`
Good:
```
I can see the desktop. There's a browser window on the right monitor 
showing a GitHub repo, and a terminal on the left.
Action: Click the Edge address bar, type the search, press Enter.
```

#### 2. Use Coordinates + Context Together

Screen coordinates are fragile (window positions change). Always pair them with visual verification:

```
Try clicking at approximately (2550, 175) where the search box should be.
Verify with OCR afterward.
```

#### 3. Chain Operations Explicitly

Human: type → press Enter → wait → read result → decide next

```
Step 1: focus_window(handle)
Step 2: click(x, y) on the search box
Step 3: type("best cat memes 2026")
Step 4: key_press(["Enter"])
Step 5: wait(3000ms)
Step 6: ocr() to read results
Step 7: decide next action based on what's on screen
```

#### 4. Error Recovery Is Part of the Prompt

Things go wrong: windows don't load, elements shift, CAPTCHAs appear.

```
If OCR doesn't show the expected search results:
  1. Screenshot again to check state
  2. If a CAPTCHA appears, pause and ask human
  3. If page didn't load, wait and retry
  4. If wrong window is focused, re-focus and retry
```

### Prompt Patterns That Work

#### The "Ground-Truth-First" Pattern

Start with a screenshot/OCR before any action. Never assume state.

```
[screenshot] → What's on screen? → [action] → [screenshot] → verify
```

#### The "Small-Step" Pattern

Break complex tasks into atomic steps with verification gates.

```
Step 1: Open browser → [screenshot] → verify browser loaded
Step 2: Navigate to URL → [screenshot] → verify page loaded
Step 3: Find search box on page → [click] → [type] → [Enter]
Step 4: [screenshot] → verify results appeared
```

#### The "Fail-Fast" Pattern

Check preconditions before acting. If a window isn't found, don't blindly click.

```
if window "Edge" not in list_windows:
  launch_app("msedge.exe")
  wait(2000ms)
  verify window appeared
```

### Common Failure Modes

| Failure | Cause | Mitigation |
|---------|-------|------------|
| Click missed | Window moved/z-order changed | OCR-verify before clicking |
| Typed text wrong | Focus was on wrong element | Click to focus first |
| UIPI block | Targeting admin window | Run server elevated |
| OCR garbled | Small font, low contrast | OCR region, increase scale |
| Infinite loop | No stop condition | Always include a max-retry count |
| Layout change | Site updated UI | Use `find_ui_element` (memory→ONNX→OCR cascade) |

---

## Part 4: How I (Big Pickle) Started Using MCP — A Case Study

### Session 0: Discovery

The user had an MCP server already configured at `D:\SCRIPTS\mcp_experiment\mcp-server.exe` (v0.2.9, 108 tools, Go + Win32/COM). The opencode config at `C:\Users\WORKSHOP\.config\opencode\opencode.jsonc` was simple:

```json
{
  "mcp": {
    "computer_use": {
      "type": "local",
      "command": ["D:\\SCRIPTS\\mcp_experiment\\mcp-server.exe"]
    }
  }
}
```

### Session 1: Verification

Running `opencode mcp list` confirmed the server was connected:
```
● ✓ computer_use connected
     D:\SCRIPTS\mcp_experiment\mcp-server.exe
```

The MCP server's tools surfaced as native function calls — no special syntax needed.

### Session 2: Basic Diagnostics

First real use: system inspection. The tools returned live, real-time data:

```
Tool                   Response
─────────────────────────────────────────────────────────
get_system_info        DESKTOP-912AKHP, 12GB RAM, Windows
get_screen_size        3840×1080 (dual monitor)
get_disk_usage         C: 161GB free (32%), D: 113GB free (84%)
get_network_info       3 IPs, 2 DNS servers
get_battery            Desktop — no battery
list_processes         143 processes, mcp-server.exe PID 5808
```

### Session 3: Interactive Control

Full browser navigation via the peripheral abstraction:

1. **`list_windows`** → found Edge (handle 66738, PID 11040)
2. **`focus_window(66738)`** → brought Edge to foreground
3. **`click(2550, 175)`** → focused the Google search box
4. **`type("best cat memes 2026")`** → typed the query
5. **`key_press(["Enter"])`** → submitted the search
6. **`ocr()`** → read back the search suggestions and results

Every step mirrored what a human would do — look, point, click, type, read.

### What Made It Work

| Factor | Why It Mattered |
|--------|----------------|
| **MCP protocol** | Standardized tool discovery — no custom integration code needed |
| **OS-native APIs** | The server uses Win32 `SendInput`, GDI `BitBlt`, COM `UIAutomation` — same APIs Windows uses internally |
| **Chained tools** | `wait+ocr+click` sequences let me build reliable multi-step operations |
| **OCR feedback loop** | Every action was verified by reading the screen state afterward |
| **Window handles** | Direct window targeting (not fragile pixel coordinates) |

---

## Part 5: Practical Prompt Recipes

### Recipe 1: "What's on my computer right now?"

```
Steps:
1. list_windows() → see all open applications
2. get_disk_usage() → check storage
3. get_system_info() → check RAM
4. list_processes() → see what's running (top 10 by memory)
```

### Recipe 2: "Search the web for something"

```
Steps:
1. list_windows() → find browser handle
2. focus_window(handle) → bring browser forward
3. If no browser open: launch_app("msedge.exe"), wait(2000ms)
4. browser_focus_url_bar("edge") → focus the address bar
5. type("search query") → type what to search
6. key_press(["Enter"]) → submit
7. wait(3000ms) → let results load
8. ocr() → read the results page
```

### Recipe 3: "Read what's in that window"

```
Steps:
1. list_windows() → find target window by title
2. get_window_state(handle) → get its position rect
3. ocr(region=window_rect) → read all text in that window
4. Present summary of what's there
```

### Recipe 4: "Automate a repetitive task"

```
Pattern: screenshot → analyze → act → verify → loop

loop(max=5):
  1. screenshot() → capture current state
  2. ocr() → read text elements
  3. Decide next action based on state
  4. click(x,y) or type(text) or key_press(keys)
  5. wait(1000ms)
  6. if goal_reached: break
```

---

## Part 6: Beyond the Basic Loop — Agent Architecture

The Perceive-Reason-Act loop (Part 2) is the foundation. But a production agent system splits each phase into dedicated layers with distinct responsibilities.

### The Full Agent Stack

```
LLM ────────── Cognitive Layer (reasoning + planning)
  ↓
MCP ────────── Orchestration Layer (skill decomposition + execution logic)
  ↓
Controller ─── Physical Layer (mouse, keyboard, window system)
  ↓
Perception ─── Vision Layer (OCR + ML element detection + screen capture)
  ↓
World ───────── Desktop / Browser / Applications
  ↑
Memory ─────── State Layer (UI element positions, action history, confidence)
```

Each layer does one job well — no overlap in responsibility.

### How a Task Flows Through the Stack

```
1. Perception (ML Vision + OCR)
   Screenshot → UI elements detected → coordinates + labels produced

2. State Building (Memory Layer)
   "Search bar was at (800, 120) last time"
   "Confidence: 0.92"

3. Reasoning (LLM)
   Interprets goal: "find article about MCP"
   Chooses strategy: open browser → search → click first result

4. Planning (MCP)
   Breaks strategy into atomic skills:
     browser_focus_url_bar()
     type("MCP protocol guide")
     key_press(["Enter"])
     wait_for_load()
     ocr()

5. Execution (Controller)
   Mouse + keyboard actions are performed

6. Feedback Loop
   Vision confirms the result
   If mismatch → retry or escalate to LLM for re-planning
```

### The Upgrade Path: From Stateless to Stateful

Basic agents work like a state machine with no memory:

```
do action → hope OCR confirms → continue
```

The next level adds prediction + verification:

```
predict outcome → act → verify → repair if wrong
```

| Capability | Basic Agent | Advanced Agent |
|------------|-------------|----------------|
| Element location | Searches every time | Remembers from past sessions |
| Error recovery | Blind retry | Detect failure → re-plan |
| Timing | Fixed waits | Wait-for-state-change |
| UI change tolerance | Breaks on any shift | Adapts via confidence decay |
| Cross-machine | Tied to dev setup | Generalizes via ML |

### ML Vision + Spatial Memory

**OCR tells you what text is on screen. ML vision tells you what's on screen.**

The addition of a computer vision model for UI element detection changes everything:

```
OCR:                  "Search  Images  Gmail"
ML Vision:            [button: "Search"] [link: "Images"] [button: "Gmail"]
                      at (x=800,y=120)    at (x=900,y=120)  at (x=1000,y=120)
```

With persistent element memory:

```
Element: "Search button"
  ─ last_seen: (x=812, y=430)
  ─ confidence: 0.92
  ─ context: Google homepage
  ─ visual_hash: <feature vector>
```

The agent doesn't "hunt" every time — it remembers where reality usually is, removing 60–80% of the instability in UI agents.

### Division of Responsibilities

| Layer | Role | Danger If Overstepped |
|-------|------|----------------------|
| **LLM** | Intent + strategy | Micromanaging clicks instead of planning outcomes |
| **MCP** | Structured decomposition | Making decisions instead of executing structure |
| **ML Vision** | Ground-truth perception | Overconfident labels corrupt the whole chain |
| **Memory** | Probabilistic UI map | Stale positions cause incorrect decisions |
| **Controller** | Dumb but precise execution | Adding logic where only movement is needed |

### The Convergence: LLM + MCP + ML

You are not building "an AI that uses a computer." You are building a **closed-loop embodied agent operating inside a GUI environment** — with perception, memory, and reasoning as separate subsystems.

```
Vision Model ──→ "What exists on screen"
Memory Layer ──→ "Where it usually is"
MCP Skills ────→ "What can be done"
LLM ───────────→ "What should be done"
Controller ────→ "Do it"
Feedback ──────→ "Did it work?"
```

This is the difference between:
- Prompt chaining systems that collapse at scale
- Automation scripts that break on any change
- Simple RPA bots with no adaptability

...and a real **digital operator** that understands the interface it controls.

### Design Constraint Summary

```
✅ LLM = intent + strategy
✅ MCP = structured decomposition
✅ ML = ground truth perception
✅ Memory = probabilistic UI map
✅ Controller = dumb but precise execution
❌ No overlap in responsibility
```

When any layer tries to do everything, the system becomes fragile. When they stay layered, the system becomes a platform — not just a project tied to one developer's machine.

---

## Appendix: The Tool Landscape (2026)

### Computer-Use Agent Ecosystem

| Layer | Examples |
|-------|----------|
| **MCP Servers** | go-mcp-computer-use (Windows), Playwright MCP (web), Browserbase (cloud) |
| **CUA Frameworks** | OpenAI CUA, Anthropic Computer Use, Browser Use (open source), Stagehand |
| **Protocols** | MCP (tool-to-AI), A2A (agent-to-agent), ACP (agent communication) |
| **IDE Integration** | Claude Code, Cursor, Windsurf, Cline, Continue, OpenCode, Roo Code |

### Key Trends (2026 Research)

- **MCP + A2A hybrid** architectures achieve 40-60% faster workflow development
- **Visual grounding** replaces fragile CSS selectors — AI agents find elements by what they look like
- **Training pipelines** extend: every CUA action saves screenshots for model fine-tuning
- **Security** is the #1 concern: prompt injection, UIPI bypass, clipboard theft, screenshot exposure
- **Accessibility** is the killer app: AI-as-hands-free-interface for users with physical limitations

---

> *Written by Big Pickle, an AI agent who took its first steps with MCP by listing windows, clicking buttons, and googling cat memes — all through the same peripherals humans have used for 40 years.*
