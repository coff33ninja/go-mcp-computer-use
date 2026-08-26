param(
    [switch]$Force
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$icoPath = Join-Path (Join-Path $repoRoot "icons") "app.ico"
$sysoOut = Join-Path (Join-Path (Join-Path $repoRoot "cmd") "mcp-server") "rsrc_windows.syso"

if (-not (Test-Path $icoPath)) {
    Write-Host "Icon not found at $icoPath - skipping icon generation" -ForegroundColor Yellow
    exit 0
}

$rsrc = Get-Command "rsrc" -ErrorAction SilentlyContinue
if (-not $rsrc) {
    Write-Host "Installing rsrc..." -ForegroundColor Gray
    go install github.com/akavel/rsrc@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
    $rsrc = Get-Command "rsrc" -ErrorAction SilentlyContinue
    if (-not $rsrc) {
        Write-Host "rsrc not found after install - skipping icon generation" -ForegroundColor Yellow
        exit 0
    }
}

$sysoDir = Split-Path $sysoOut -Parent
if (-not (Test-Path $sysoDir)) {
    New-Item -ItemType Directory -Path $sysoDir -Force | Out-Null
}

Write-Host "Generating $sysoOut from $icoPath" -ForegroundColor Cyan
& rsrc -ico $icoPath -o $sysoOut
if (-not $?) { exit 1 }

Write-Host "OK: $sysoOut" -ForegroundColor Green
