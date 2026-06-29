param(
    [switch]$NoCGO,
    [switch]$Release,
    [string]$Output = "mcp-server.exe"
)

$ErrorActionPreference = "Stop"
$ver = (Get-Content (Join-Path $PSScriptRoot "..\VERSION") -Raw).Trim()

Write-Host "=== go-mcp-computer-use build ===" -ForegroundColor Cyan
Write-Host "Version: $ver" -ForegroundColor Gray

$go = Get-Command "go" -ErrorAction SilentlyContinue
if (-not $go) { Write-Host "Go not found." -ForegroundColor Red; exit 1 }

if ($NoCGO) {
    $env:CGO_ENABLED = "0"
    Write-Host "CGO disabled — ONNX tools excluded" -ForegroundColor Yellow
} else {
    $zig = Get-Command "zig" -ErrorAction SilentlyContinue
    if (-not $zig) {
        Write-Host "Zig not found. Install Zig (winget install zig) or use -NoCGO for a limited build." -ForegroundColor Red
        exit 1
    }
    $env:CC = "zig cc"
    $env:CGO_ENABLED = "1"
    Write-Host "C compiler: Zig cc ($(zig version))" -ForegroundColor Cyan
}

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
