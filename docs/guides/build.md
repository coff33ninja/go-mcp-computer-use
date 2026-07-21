# Build & Usage

## Requirements

- Windows 10 or 11
- Go 1.26+ (to build from source)
- **Zig** 0.16+ (for CGO — `winget install zig`)

CGO is mandatory — ONNX runtime requires it, and Zig `cc` serves as the C cross-compiler. Install Zig once, then any `go build` with `CC="zig cc" CGO_ENABLED=1` works.

> **Note:** The Go-native transformer engine (Gorgonia) does NOT require CGO — it's pure Go. CGO is only needed for ONNX runtime (YOLO/MobileNet inference). If you build without CGO, you lose ONNX tools but the transformer and all other tools still work.

> **Gorgonia env var (go1.26+):** Go 1.26+ requires `ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH=go1.26` for Gorgonia. Add it to the `env` field in your MCP client config. A `launch.ps1` wrapper is also available for manual use.

## Quick Start

```powershell
git clone https://github.com/coff33ninja/go-mcp-computer-use.git
cd go-mcp-computer-use
.\scripts\build.ps1
.\mcp-server.exe
```

Or use the install script:

```powershell
.\scripts\install.ps1
```

## Build

```powershell
.\scripts\build.ps1              # requires Zig cc + CGO (ONNX-enabled)
```

Cross-compile from Linux/macOS:

```bash
CC="zig cc" CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o mcp-server.exe ./cmd/mcp-server/
```

CGO is mandatory — ONNX runtime requires it, and Zig cc handles the cross-compilation. Install Zig 0.16+ via `winget install zig`.

The Go-native transformer engine (action prediction) is pure Go and builds without CGO — it only needs `go build`. ONNX tools (YOLO/MobileNet) require CGO.

## Running

```powershell
.\launch.ps1              # recommended — sets Gorgonia env var automatically
.\mcp-server.exe          # also works if ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH is set
```

For MCP client configs, point to `launch.ps1` instead of `mcp-server.exe`.

## Performance

Benchmark results (1600x900 display, averaged):

| Operation | Time | vs Previous |
|---|---|---|
| Screenshot (full) | 104 ms | |
| Screenshot (400x400 region) | 17 ms | |
| OCR (full screen) | **292 ms** | 2.2x faster (native COM WinRT) |
| OCR (400x400 region) | **68 ms** | 8x faster (native COM WinRT) |
| Template match (full screen) | 16 ms | |
| Template match (in region) | 2 ms | |
| find_text_and_click | **275 ms** | 2.9x faster |
| get_pixel_color | 18 ms | |
| get_keyboard_layout | 667 ms | |
| get_network_info | 10 ms | |
| list_processes | 14 ms | |
| get_volume | 10 ms | |

Run `go run .\cmd\benchmark\` locally to produce current numbers.
