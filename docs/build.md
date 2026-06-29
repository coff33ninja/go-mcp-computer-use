# Build & Usage

## Requirements

- Windows 10 or 11
- Go 1.26+ (to build from source)
- **Zig** 0.16+ (for CGO — `winget install zig`)

The project uses Zig `cc` as the C cross-compiler for CGO (needed by the `onnxruntime_go` dependency for ONNX ML inference). Install Zig once, then any `go build` with `CC="zig cc" CGO_ENABLED=1` works.

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
.\scripts\build.ps1              # with Zig cc + CGO (default)
.\scripts\build.ps1 -NoCGO       # limited build, no ONNX tools
```

Cross-compile from Linux/macOS:

```bash
CC="zig cc" CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o mcp-server.exe ./cmd/mcp-server/
```

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
