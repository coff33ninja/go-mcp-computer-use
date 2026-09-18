param(
    [switch]$Train
)
$ErrorActionPreference = "Stop"
$env:ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH = "go1.26"
Set-Location (Join-Path $PSScriptRoot "..")
if ($Train) {
    go run ./cmd/ml-eval -train
} else {
    go run ./cmd/ml-eval
}
exit $LASTEXITCODE
