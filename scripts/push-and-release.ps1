param(
    [switch]$NoCGO,
    [string]$OpenCodeDesktop = "$env:LOCALAPPDATA\Programs\@opencode-aidesktop\OpenCode.exe"
)

# ---- Step 0: Validate ----
$ErrorActionPreference = "Stop"
if (-not (Test-Path $OpenCodeDesktop)) {
    Write-Error "OpenCode Desktop not found at $OpenCodeDesktop"
    exit 1
}

# ---- Step 1: Read version ----
$version = (Get-Content VERSION -Raw).Trim()
if (-not $version) {
    Write-Error "VERSION file is empty"
    exit 1
}
$tag = "v$version"
Write-Host "=== Release: $tag ==="

# ---- Step 2: Read changelog section for commit body ----
$changelog = Get-Content docs/CHANGELOG.md -Raw
$pattern = "(?ms)## \[$version\].*?(?=\n## \[|\z)"
$commitBody = ""
if ($changelog -match $pattern) {
    $commitBody = $Matches[0].Trim()
}
# Write commit message to temp file to avoid multiline quoting issues
$commitMsg = "release: $tag"
if ($commitBody) {
    $commitMsg += "`n`n$commitBody"
}
$msgFile = "$env:TEMP\mcp-commit-msg.txt"
$commitMsg | Set-Content -Path $msgFile -Encoding UTF8

# ---- Step 3: Commit and tag ----
Write-Host "Committing..."
git add -A
git commit -F $msgFile
if ($LASTEXITCODE) { throw "git commit failed" }

Write-Host "Tagging $tag..."
git tag $tag

# ---- Step 4: Push ----
Write-Host "Pushing commits..."
git push
if ($LASTEXITCODE) { throw "git push failed" }

Write-Host "Pushing tag $tag..."
git push origin $tag
if ($LASTEXITCODE) { throw "git push tag failed" }

# ---- Step 5: Wait for release workflow ----
Write-Host "Waiting for release workflow to finish..."
$runId = $null
$maxWait = 900
$elapsed = 0
while ($elapsed -lt $maxWait) {
    $runsJson = gh run list --workflow=Release --limit 5 --json databaseId,status,headBranch,conclusion 2>$null
    $run = ($runsJson | ConvertFrom-Json) | Where-Object { $_.headBranch -eq $tag } | Select-Object -First 1
    if ($run) {
        if ($run.status -eq "completed") {
            $runId = $run.databaseId
            if ($run.conclusion -ne "success") {
                throw "Release workflow failed: $($run.conclusion)"
            }
            break
        }
        Write-Host "  workflow running... ($($elapsed)s)"
    } else {
        Write-Host "  waiting for trigger... ($($elapsed)s)"
    }
    Start-Sleep -Seconds 15
    $elapsed += 15
}
if (-not $runId) {
    throw "Release workflow did not complete within ${maxWait}s"
}
Write-Host "Release workflow completed successfully."

# ---- Step 6: Download release asset ----
$exeName = if ($NoCGO) { "mcp-server.exe" } else { "mcp-server-cgo.exe" }
$localExe = "mcp-server.exe"

Write-Host "Downloading $exeName from release $tag..."
$dlDir = "$env:TEMP\mcp-release"
Remove-Item $dlDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $dlDir -Force | Out-Null

$dlAttempt = 0
do {
    gh release download $tag --pattern $exeName --dir $dlDir 2>$null
    if ($LASTEXITCODE -and $dlAttempt -lt 6) {
        Write-Host "  release not ready yet, retrying in 10s... (attempt $($dlAttempt+1))"
        Start-Sleep -Seconds 10
        $dlAttempt++
    }
} while ($LASTEXITCODE -and $dlAttempt -lt 6)
if ($LASTEXITCODE) { throw "Failed to download $exeName after 6 attempts" }

# ---- Step 7: Replace exe in project root ----
Write-Host "Replacing $localExe..."
$src = "$dlDir\$exeName"
if (-not (Test-Path $src)) { throw "Downloaded file not found at $src" }
Copy-Item -Path $src -Destination "$PWD\$localExe" -Force

# Cleanup
Remove-Item $dlDir -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item $msgFile -Force -ErrorAction SilentlyContinue
Write-Host "$localExe updated from release $tag"

# ---- Step 8: Kill OpenCode Desktop processes ----
Write-Host "Closing OpenCode Desktop..."
Get-Process -Name "OpenCode" -ErrorAction SilentlyContinue | ForEach-Object {
    Write-Host "  killing PID $($_.Id)"
    Stop-Process -Id $_.Id -Force
}
Start-Sleep -Seconds 3

# ---- Step 9: Relaunch as Admin ----
Write-Host "Launching OpenCode Desktop as Administrator..."
Start-Process -FilePath $OpenCodeDesktop -Verb RunAs

Write-Host "=== Done ==="
Write-Host "OpenCode Desktop is restarting as admin."
Write-Host "MCP server v$version is updated and ready."
