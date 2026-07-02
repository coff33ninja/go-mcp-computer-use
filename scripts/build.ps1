param(
    [switch]$Release,
    [string]$Output = "mcp-server.exe"
)

$ErrorActionPreference = "Stop"
$ver = (Get-Content (Join-Path $PSScriptRoot "..\VERSION") -Raw).Trim()

Write-Host "=== go-mcp-computer-use build ===" -ForegroundColor Cyan
Write-Host "Version: $ver" -ForegroundColor Gray

$go = Get-Command "go" -ErrorAction SilentlyContinue
if (-not $go) { Write-Host "Go not found." -ForegroundColor Red; exit 1 }

$zig = Get-Command "zig" -ErrorAction SilentlyContinue
if (-not $zig) {
    Write-Host "Zig not found. Install Zig (winget install zig)." -ForegroundColor Red
    exit 1
}
$env:CC = "zig cc"
$env:CGO_ENABLED = "1"
# Pin CPU baseline to x86_64_v2 so binary runs on older CPUs
$env:CGO_CFLAGS = "-mcpu=x86_64_v2 -fno-sanitize=all -Wno-error=unused-command-line-argument"
$env:CGO_LDFLAGS = "-mcpu=x86_64_v2 -fno-sanitize=all -Wno-error=unused-command-line-argument"
Write-Host "C compiler: Zig cc ($(zig version))" -ForegroundColor Cyan
Write-Host "CGO_CFLAGS: $env:CGO_CFLAGS" -ForegroundColor Gray
Write-Host "CGO_LDFLAGS: $env:CGO_LDFLAGS" -ForegroundColor Gray

# Generate icon resource (.syso) for embedding
Write-Host "Generating icon resource..." -ForegroundColor Gray
& "$PSScriptRoot\gen-icons.ps1"
if (-not $?) { exit 1 }

$ldflags = "-s -w -X main.Version=$ver"
if (-not $Release) {
    $ldflags = "-X main.Version=$ver"
}

Write-Host "Building..." -ForegroundColor Gray
go build -ldflags="$ldflags" -o $Output .\cmd\mcp-server\
if (-not $?) { exit 1 }

$sizeBytes = (Get-Item $Output).Length
$mib = [math]::Round($sizeBytes / 1048576, 1)
Write-Host "OK: $Output ($mib` MB)" -ForegroundColor Green
