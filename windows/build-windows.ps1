[CmdletBinding()]
param(
    [ValidateSet("amd64", "386", "arm64")]
    [string]$Architecture = "amd64",
    [switch]$NoCGO,
    [switch]$Help
)
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$frontendDir = Join-Path $repoRoot "frontend"
$frontendDist = Join-Path $frontendDir "dist"
$webDir = Join-Path $repoRoot "web"
$webHtml = Join-Path $webDir "html"
$outputPath = Join-Path $repoRoot "sui.exe"
$stageDir = $null
$backupDir = $null

function Restore-PreviousWebAssets {
    if ($script:backupDir -and (Test-Path -LiteralPath (Join-Path $script:backupDir "html")) -and -not (Test-Path -LiteralPath $script:webHtml)) {
        Move-Item -LiteralPath (Join-Path $script:backupDir "html") -Destination $script:webHtml
    }
}

if ($Help) {
    Write-Host "Usage: .\build-windows.ps1 [-Architecture <arch>] [-NoCGO] [-Help]"
    Write-Host "Architectures: amd64, 386, arm64"
    Write-Host "Examples:"
    Write-Host "  .\build-windows.ps1                    # Build amd64 with CGO"
    Write-Host "  .\build-windows.ps1 -Architecture 386 # Build 32-bit Windows with CGO"
    Write-Host "  .\build-windows.ps1 -Architecture arm64 -NoCGO"
    exit 0
}

$Architecture = $Architecture.ToLowerInvariant()

Write-Host "Building S-UI for Windows ($Architecture)..." -ForegroundColor Green

# Check if Go is installed
try {
    $goVersion = go version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Go not found"
    }
    Write-Host "Go version: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Check if Node.js is installed
try {
    $nodeVersion = node --version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Node.js not found"
    }
    Write-Host "Node.js version: $nodeVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Node.js is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Node.js from https://nodejs.org/" -ForegroundColor Yellow
    exit 1
}

# Build frontend without touching the last known-good embedded assets.
Write-Host "Building frontend..." -ForegroundColor Yellow
Push-Location -LiteralPath $frontendDir

try {
    Write-Host "Installing dependencies..." -ForegroundColor Cyan
    npm ci
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to install frontend dependencies"
    }

    npm run lint -- --max-warnings=0
    if ($LASTEXITCODE -ne 0) {
        throw "Frontend lint failed"
    }

    npm run test
    if ($LASTEXITCODE -ne 0) {
        throw "Frontend unit tests failed"
    }

    npm run build
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to build frontend"
    }

    npm run verify:dist
    if ($LASTEXITCODE -ne 0) {
        throw "Frontend production dist verification failed"
    }
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    Pop-Location
    exit 1
}

Pop-Location

try {
    # Stage beside web/html so a failed copy cannot partially overwrite the
    # embedded assets. Keep the previous directory available for rollback.
    New-Item -ItemType Directory -Path $webDir -Force | Out-Null
    $stageDir = Join-Path $webDir (".html-stage." + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $stageDir | Out-Null
    Copy-Item -Path (Join-Path $frontendDist "*") -Destination $stageDir -Recurse -Force

    if (Test-Path -LiteralPath $webHtml) {
        $backupDir = Join-Path $webDir (".html-backup." + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $backupDir | Out-Null
        Move-Item -LiteralPath $webHtml -Destination (Join-Path $backupDir "html")
    }

    try {
        Move-Item -LiteralPath $stageDir -Destination $webHtml
        $stageDir = $null
    }
    catch {
        Restore-PreviousWebAssets
        throw "Failed to replace embedded web assets: $($_.Exception.Message)"
    }

    if ($backupDir) {
        Remove-Item -LiteralPath $backupDir -Recurse -Force
        $backupDir = $null
    }
}
catch {
    if ($stageDir -and (Test-Path -LiteralPath $stageDir)) {
        Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    try {
        Restore-PreviousWebAssets
    }
    catch {
        [Console]::Error.WriteLine("Error: Failed to restore prior web assets from ${backupDir}: $($_.Exception.Message)")
    }
    if ($backupDir -and (Test-Path -LiteralPath $backupDir)) {
        $savedHtml = Join-Path $backupDir "html"
        if (-not (Test-Path -LiteralPath $savedHtml)) {
            Remove-Item -LiteralPath $backupDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        else {
            Write-Warning "Prior web assets remain preserved at $savedHtml"
        }
    }
    Write-Host "Error: $_" -ForegroundColor Red
    exit 1
}

# Build backend without fallback: a partial-tag or CGO fallback build can
# silently omit capabilities promised by the release artifact.
Write-Host "Building backend..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = $Architecture
$env:CGO_ENABLED = if ($NoCGO) { "0" } else { "1" }

$buildTags = "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0,with_tailscale,with_wireguard"
$ldFlags = "-w -s -X github.com/deposist/s-ui-x/config.ArtifactPlatform=$Architecture -X internal/godebug.defaultGODEBUG=multipathtcp=0 -checklinkname=0"

try {
    & go -C $repoRoot build -ldflags $ldFlags -tags $buildTags -o $outputPath main.go
    if ($LASTEXITCODE -ne 0) {
        throw "Backend build failed for Windows $Architecture with CGO_ENABLED=$env:CGO_ENABLED; no fallback was attempted"
    }
    Write-Host "Built successfully with CGO_ENABLED=$env:CGO_ENABLED" -ForegroundColor Green
} catch {
    Write-Host "Error: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Build completed successfully!" -ForegroundColor Green
Write-Host "Output: $outputPath" -ForegroundColor Green

# Show file info
if (Test-Path -LiteralPath $outputPath) {
    $fileInfo = Get-Item -LiteralPath $outputPath
    Write-Host "File size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Cyan
    Write-Host "Created: $($fileInfo.CreationTime)" -ForegroundColor Cyan
}
