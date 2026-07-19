param(
    [switch]$Integration,
    [switch]$Short,
    [int]$TimeoutSec = 120
)

$ErrorActionPreference = "Stop"

Write-Host "=== go-mcp-computer-use test ===" -ForegroundColor Cyan

# Detect Zig and configure CGO for C interop tests
$zig = Get-Command "zig" -ErrorAction SilentlyContinue
if ($zig) {
    $env:CC = "zig cc"
    $env:CGO_ENABLED = "1"
    $env:CGO_CFLAGS = "-mcpu=x86_64_v2 -fno-sanitize=all -Wno-error=unused-command-line-argument"
    $env:CGO_LDFLAGS = "-mcpu=x86_64_v2 -fno-sanitize=all -Wno-error=unused-command-line-argument"
    Write-Host "C compiler: Zig cc ($(zig version))" -ForegroundColor Cyan
} else {
    Write-Host "Zig not found — using default C compiler" -ForegroundColor Yellow
    $env:CGO_ENABLED = "1"
}

# Run lint first
Write-Host "`n[*] Running go vet..." -ForegroundColor Gray
go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "go vet failed" -ForegroundColor Red
    exit 1
}

# Build test
Write-Host "`n[*] Building..." -ForegroundColor Gray
go build ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}
Write-Host "Build OK" -ForegroundColor Green

# Run unit tests
$testArgs = @("test", "./...", "-count=1", "-v")
if ($Short) {
    $testArgs += "-short"
}
$testArgs += "-timeout", "${TimeoutSec}s"

if ($Integration) {
    $testArgs += "-tags=integration"
    Write-Host "`n[*] Running ALL tests (including integration)..." -ForegroundColor Cyan
} else {
    Write-Host "`n[*] Running unit tests (skip integration)..." -ForegroundColor Cyan
    $testArgs += "-run", "^(?!TestChain_)"
}

Write-Host "  go $($testArgs -join ' ')" -ForegroundColor Gray
& go @testArgs

if ($LASTEXITCODE -ne 0) {
    Write-Host "`nSome tests failed" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host "`n=== All tests passed ===" -ForegroundColor Green
