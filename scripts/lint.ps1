param(
    [switch]$Fix
)

$ErrorActionPreference = "Stop"

Write-Host "=== go vet ===" -ForegroundColor Cyan
$result = go vet ./... 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host $result -ForegroundColor Red
    Write-Host "FAIL: go vet found issues. See .govetallow for conventions." -ForegroundColor Red
    exit 1
}
Write-Host "PASS: go vet clean" -ForegroundColor Green

Write-Host "=== go build ===" -ForegroundColor Cyan
$ver = (Get-Content VERSION -Raw).Trim()
go build -ldflags="-X main.Version=$ver" ./cmd/mcp-server/
if ($LASTEXITCODE -ne 0) {
    exit 1
}
Write-Host "PASS: build ok" -ForegroundColor Green

if ($Fix) {
    Write-Host "=== go test (short) ===" -ForegroundColor Cyan
    go test -short -count=1 ./internal/actions/ 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAIL: tests failed" -ForegroundColor Red
        exit 1
    }
    Write-Host "PASS: tests ok" -ForegroundColor Green
}
