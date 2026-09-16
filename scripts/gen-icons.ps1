param(
    [switch]$Force
)

# Generates the Windows resource object (.syso) embedding the app icon plus a
# VS_VERSION_INFO resource (CompanyName, LegalCopyright under Apache-2.0, File/
# Product version, etc.) that Windows Explorer shows in Properties -> Details.
# Uses go-winres (github.com/tc-hib/go-winres), which supersedes akavel/rsrc
# (rsrc only embeds an icon, no version-info block).

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")

$goWinres = Get-Command "go-winres" -ErrorAction SilentlyContinue
if (-not $goWinres) {
    Write-Host "Installing go-winres..." -ForegroundColor Gray
    go install github.com/tc-hib/go-winres@latest
    $env:PATH = "$env:USERPROFILE\go\bin;$env:PATH"
    $goWinres = Get-Command "go-winres" -ErrorAction SilentlyContinue
    if (-not $goWinres) {
        Write-Host "go-winres not found after install - skipping resource generation" -ForegroundColor Yellow
        exit 0
    }
}

$ver = (Get-Content (Join-Path $repoRoot "VERSION") -Raw).Trim()

# Render winres/winres.json from the template with the version substituted.
$templatePath = Join-Path $repoRoot "winres\winres.json.template"
$jsonOut = Join-Path $repoRoot "winres\winres.json"
if (-not (Test-Path $templatePath)) {
    Write-Host "winres template not found at $templatePath - skipping resource generation" -ForegroundColor Yellow
    exit 0
}
$rendered = (Get-Content $templatePath -Raw).Replace("{{VERSION}}", $ver)
# Write WITHOUT a UTF-8 BOM: PowerShell 5.1's Set-Content -Encoding UTF8 (and 7's)
# can prepend a BOM, which go-winres rejects ("invalid character 'ï' looking for
# beginning of value"). [IO.File]::WriteAllText with a BOM-less UTF8Encoding works
# identically on both hosts.
[System.IO.File]::WriteAllText($jsonOut, $rendered, [System.Text.UTF8Encoding]::new($false))

$sysoOut = Join-Path $repoRoot "cmd\mcp-server\rsrc_windows_amd64.syso"
# Remove the legacy akavel/rsrc .syso (icon only, no version-info) so the two
# resource objects don't both define RT_GROUP_ICON / RT_VERSION.
$legacySyso = Join-Path $repoRoot "cmd\mcp-server\rsrc_windows.syso"
if (Test-Path $legacySyso) {
    Remove-Item $legacySyso -Force
}

Write-Host "Generating version-info + icon resource (go-winres) v$ver ..." -ForegroundColor Cyan
Push-Location $repoRoot
& "$($goWinres.Source)" make --in winres/winres.json --arch amd64 --out "cmd/mcp-server/rsrc" 2>&1
$ok = $?
Pop-Location
if (-not $ok) { exit 1 }

if (-not (Test-Path $sysoOut)) {
    Write-Host "FAILED: $sysoOut not produced" -ForegroundColor Red
    exit 1
}
Write-Host "OK: $sysoOut" -ForegroundColor Green
