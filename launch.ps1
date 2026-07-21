# launch.ps1 — wrapper that sets required env vars and starts mcp-server.exe
# Use this instead of running mcp-server.exe directly.
# Sets ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH for Gorgonia (Go 1.26+).

$env:ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH = "go1.26"

$exe = Join-Path $PSScriptRoot "mcp-server.exe"
if (-not (Test-Path $exe)) {
    Write-Error "mcp-server.exe not found in $PSScriptRoot"
    exit 1
}

& $exe @args
