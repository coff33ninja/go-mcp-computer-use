param(
    [switch]$Force
)

<#
.SYNOPSIS
    Generate Windows icon resource (.syso) for embedding into mcp-server.exe.
    Uses rsrc (github.com/akavel/rsrc) to compile app.ico into a COFF object.
.DESCRIPTION
    Run before build. The .syso file is placed alongside Go source in cmd/mcp-server/
    so the Go linker automatically picks it up.
#>

$ErrorActionPreference = "Stop"
$repoRoot    = Resolve-Path (Join-Path $PSScriptRoot "..")
$icoPath     = Join-Path (Join-Path $repoRoot "icons") "app.ico"
$sysoOut     = Join-Path (Join-Path (Join-Path $repoRoot "cmd") "mcp-server") "rsrc_windows.syso"

if (-not (Test-Path $icoPath)) {
    Write-Error "Icon not found at $icoPath — run icons/gen-icons.py first"
    exit 1
}

# Install rsrc if not present
$rsrc = Get-Command "rsrc" -ErrorAction SilentlyContinue
if (-not $rsrc) {
    Write-Host "Installing rsrc (github.com/akavel/rsrc)..." -ForegroundColor Gray
    go install github.com/akavel/rsrc@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
    $rsrc = Get-Command "rsrc" -ErrorAction SilentlyContinue
    if (-not $rsrc) {
        Write-Error "rsrc not found after install"
        exit 1
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
