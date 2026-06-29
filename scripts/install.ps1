param(
    [string]$InstallDir = "$env:LOCALAPPDATA\go-mcp-computer-use",
    [switch]$Update,
    [switch]$UseZig
)

$ErrorActionPreference = "Stop"

$repo = "https://github.com/coff33ninja/go-mcp-computer-use"

Write-Host "go-mcp-computer-use installer" -ForegroundColor Cyan
Write-Host ""

# Check Go
$go = Get-Command "go" -ErrorAction SilentlyContinue
if (-not $go) {
    Write-Host "Go is required to build from source." -ForegroundColor Yellow
    Write-Host "Install from: https://go.dev/dl/" -ForegroundColor Yellow
    exit 1
}

# Check / install Zig
$zig = Get-Command "zig" -ErrorAction SilentlyContinue
if ($UseZig -and -not $zig) {
    Write-Host "Zig not found. Installing Zig..." -ForegroundColor Yellow
    $zigUrl = "https://ziglang.org/download/0.16.0/zig-x86_64-windows-0.16.0.zip"
    $zigZip = "$env:TEMP\zig.zip"
    $zigDir = "$env:LOCALAPPDATA\zig"
    try {
        Invoke-WebRequest -Uri $zigUrl -OutFile $zigZip -ErrorAction Stop
        Expand-Archive -Path $zigZip -DestinationPath $zigDir -Force
        $zigPath = "$zigDir\zig-x86_64-windows-0.16.0\zig.exe"
        if (Test-Path -LiteralPath $zigPath) {
            $env:Path += ";$(Split-Path $zigPath)"
            [Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path","User") + ";$(Split-Path $zigPath)", "User")
            $zig = $zigPath
            Write-Host "Zig installed to $zigPath" -ForegroundColor Green
        }
    } catch {
        Write-Host "Zig install failed: $_" -ForegroundColor Red
        Write-Host "Continuing without Zig..." -ForegroundColor Yellow
        $UseZig = $false
    }
    Remove-Item $zigZip -ErrorAction SilentlyContinue
}

# Create install dir
if (-not (Test-Path -LiteralPath $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$exePath = "$InstallDir\mcp-server.exe"

# Check if already installed
if ((Test-Path -LiteralPath $exePath) -and -not $Update) {
    Write-Host "Already installed at: $exePath" -ForegroundColor Green
    Write-Host "Run with -Update to rebuild." -ForegroundColor Yellow
    exit 0
}

# Clone or pull
$srcDir = "$env:TEMP\go-mcp-computer-use"
if (Test-Path -LiteralPath $srcDir) {
    Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue
}

Write-Host "Cloning repository..." -ForegroundColor Gray
git clone --depth 1 $repo $srcDir 2>$null
if (-not $?) {
    Write-Host "Failed to clone repository. Check your network and git installation." -ForegroundColor Red
    exit 1
}

# Build
Write-Host "Building mcp-server.exe..." -ForegroundColor Gray
Push-Location $srcDir
try {
    if ($UseZig -and $zig) {
        $env:CC = "zig cc"
        $env:CGO_ENABLED = "1"
        Write-Host "Using Zig cc as C compiler" -ForegroundColor Cyan
    } else {
        $env:CGO_ENABLED = "0"
    }
    go build -o $exePath -ldflags="-s -w" .\cmd\mcp-server\
    if (-not $?) {
        Write-Host "Build failed." -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}

# Clean up source
Remove-Item -Recurse -Force $srcDir -ErrorAction SilentlyContinue

# Create default config
$configDir = "$env:USERPROFILE\.config\go-mcp-computer-use"
$configPath = "$configDir\config.json"
if (-not (Test-Path -LiteralPath $configPath)) {
    if (-not (Test-Path -LiteralPath $configDir)) {
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }
    @{
        log_level         = "info"
        mouse_speed       = 500
        click_delay_ms    = 100
        verify_bounds     = $true
        action_timeout_ms = 30000
    } | ConvertTo-Json | Set-Content -Path $configPath
}

Write-Host ""
Write-Host "Installed: $exePath" -ForegroundColor Green
Write-Host "Config:    $configPath" -ForegroundColor Green
Write-Host ""
Write-Host "Add to opencode.json:" -ForegroundColor Cyan
Write-Host "  `"command`": `"$exePath`"" -ForegroundColor Gray
if ($UseZig) {
    Write-Host ""
    Write-Host "Cross-compile for ARM64 Windows:" -ForegroundColor Cyan
    Write-Host "  CC=`"zig cc`" GOOS=windows GOARCH=arm64 go build -o $InstallDir\mcp-server-arm64.exe .\cmd\mcp-server\" -ForegroundColor Gray
}
