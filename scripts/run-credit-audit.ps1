param()

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$outDir = Join-Path (Join-Path $repoRoot "docs") "meta"
$outFile = Join-Path $outDir "credit-audit-report.json"
$version = (Get-Content (Join-Path $repoRoot "VERSION") -Raw).Trim()

Write-Host "=== credit-audit v$version ===" -ForegroundColor Cyan
Write-Host "Building..." -ForegroundColor Cyan
Push-Location $repoRoot
try {
    $env:CGO_ENABLED = "1"
    $env:CC = "zig cc"
    go build -o credit-audit-tmp.exe ./cmd/credit-audit/
    if ($LASTEXITCODE -ne 0) { throw "build failed" }

    Write-Host "Running probes (66 tools, ~90s)..." -ForegroundColor Cyan
    .\credit-audit-tmp.exe -json > $outFile
    if ($LASTEXITCODE -ne 0) { throw "audit failed" }

    Remove-Item .\credit-audit-tmp.exe -Force
} finally {
    Pop-Location
}

$report = Get-Content $outFile -Raw | ConvertFrom-Json
$totalTools = $report.total_tools
$failed = $report.failed
$totalMB = $report.total_human
$cost = $report.estimated_cost_usd

Write-Host "PASS: credit-audit v$version" -ForegroundColor Green
Write-Host "  $totalTools tools, $failed failed, $totalMB, ~`$$cost at `$3/M"
Write-Host "Report: $outFile" -ForegroundColor Gray
