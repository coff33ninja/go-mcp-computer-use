param(
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
Write-Host "=== Bumped version to $tag  ==="
Write-Host "    Read the changelog: docs/meta/CHANGELOG.md"

# ---- Step 2: Read changelog section for commit body ----
$changelog = Get-Content docs/meta/CHANGELOG.md -Raw
$pattern = "(?ms)## \[$version\].*?(?=\n## \[|\z)"
$commitBody = ""
if ($changelog -match $pattern) {
    $commitBody = $Matches[0].Trim()
}
# Write commit message to temp file to avoid multiline quoting issues
$commitMsg = "Bumped version to $tag. Read the changelog for details."
if ($commitBody) {
    $commitMsg += "`n`n$commitBody"
}
$msgFile = "$env:TEMP\mcp-commit-msg.txt"
$commitMsg | Set-Content -Path $msgFile -Encoding UTF8

# ---- Step 3: Commit and tag ----
Write-Host "Regenerating tools.md..."
go run ./scripts/gen-tools-doc.go
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
$since = (Get-Date).ToUniversalTime().AddSeconds(-30)
while ($elapsed -lt $maxWait) {
    $runsJson = gh run list --workflow=Release --limit 5 --json databaseId,status,headBranch,conclusion,createdAt 2>$null
    $run = ($runsJson | ConvertFrom-Json) | Where-Object { $_.headBranch -eq $tag -and [DateTime]$_.createdAt -ge $since } | Sort-Object createdAt -Descending | Select-Object -First 1
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

# ---- Step 6: Pull latest (CI may have pushed doc fixes, tool count patches, etc.) ----
Write-Host "Pulling latest from origin..."
git pull --ff-only
if ($LASTEXITCODE) {
    Write-Host "  fast-forward failed, trying rebase..." -ForegroundColor Yellow
    git pull --rebase
    if ($LASTEXITCODE) { throw "git pull failed — resolve conflicts manually" }
}

# ---- Step 7: Download release asset ----
Write-Host "Downloading mcp-server.exe from release $tag..."
$dlDir = "$env:TEMP\mcp-release"
Remove-Item $dlDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $dlDir -Force | Out-Null

$dlAttempt = 0
do {
    gh release download $tag --pattern "mcp-server.exe" --dir $dlDir 2>$null
    if ($LASTEXITCODE -and $dlAttempt -lt 6) {
        Write-Host "  release not ready yet, retrying in 10s... (attempt $($dlAttempt+1))"
        Start-Sleep -Seconds 10
        $dlAttempt++
    }
} while ($LASTEXITCODE -and $dlAttempt -lt 6)
if ($LASTEXITCODE) { throw "Failed to download mcp-server.exe after 6 attempts" }

# ---- Step 8-10: Spawn background cleanup + replace + relaunch ----
# Spawn as detached process so killing OpenCode doesn't kill us mid-flight
Write-Host "Scheduling cleanup, replacement, and relaunch (background)..."
$src = "$dlDir\mcp-server.exe"
if (-not (Test-Path $src)) { throw "Downloaded file not found at $src" }

$postScript = @"
Start-Sleep -Seconds 3
Get-Process -Name 'mcp-server' -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id `$_.Id -Force }
Get-Process -Name 'OpenCode' -ErrorAction SilentlyContinue | ForEach-Object { Stop-Process -Id `$_.Id -Force }
Start-Sleep -Seconds 3
Copy-Item '$src' '$PWD\mcp-server.exe' -Force
Remove-Item '$dlDir' -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item '$msgFile' -Force -ErrorAction SilentlyContinue
Start-Process -FilePath '$OpenCodeDesktop' -Verb RunAs
"@
$postScriptPath = "$env:TEMP\mcp-post-cleanup-$(Get-Random).ps1"
$postScript | Set-Content -Path $postScriptPath -Encoding UTF8
Start-Process -FilePath "powershell" -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$postScriptPath`"" -WindowStyle Hidden

Write-Host "=== Done ==="
Write-Host "OpenCode Desktop will restart as admin in a few seconds."
Write-Host "MCP server v$version is updated and queued for replacement."
