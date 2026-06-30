# Build & Usage

## Requirements

- Windows 10 or 11
- Go 1.26+ (to build from source)
- **Zig** 0.16+ (for CGO — `winget install zig`)

CGO is mandatory — ONNX runtime requires it, and Zig `cc` serves as the C cross-compiler. Install Zig once, then any `go build` with `CC="zig cc" CGO_ENABLED=1` works.

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

---

<sub><sup>
CGO is mandatory. Zig cc is mandatory. if you don't have Zig installed, you get to learn about it today. "just run `winget install zig`" we say, as if that doesn't download an entire programming language just to compile a Go binary. the cross-compile command is 127 characters long and needs `CGO_ENABLED=1`, `GOOS=windows`, `GOARCH=amd64`, `CC="zig cc"`, and your firstborn child's name. and after all that, the binary still crashes on Pentium Gold G5400s because we forgot to pin `-mcpu=x86_64_v2`. good times.
</sup></sub>
