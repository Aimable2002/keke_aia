# Keke CLI Installer for Windows
# Usage: irm https://raw.githubusercontent.com/Aimable2002/keke_aia/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$GitHubOwner = "Aimable2002"
$GitHubRepo  = "keke_aia"
$InstallDir  = Join-Path $env:LOCALAPPDATA "Keke"
$BinaryName  = "keke.exe"

function Info    { Write-Host "  ► $args" -ForegroundColor Cyan }
function Success { Write-Host "  ✓ $args" -ForegroundColor Green }
function Warn    { Write-Host "  ⚠ $args" -ForegroundColor Yellow }
function Err     { Write-Host "  ✗ $args" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  ╔══════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "  ║     Keke CLI Installer v2.0      ║" -ForegroundColor Cyan
Write-Host "  ║  AI Developer in Your Terminal   ║" -ForegroundColor Cyan
Write-Host "  ╚══════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Enable TLS 1.2 for GitHub downloads
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

Info "Detecting system..."

# Architecture detection
$Arch = $null
$envArch = $env:PROCESSOR_ARCHITECTURE

if ($envArch -eq "AMD64") {
    $Arch = "amd64"
} elseif ($envArch -eq "ARM64") {
    $Arch = "arm64"
} elseif ($envArch -eq "x86") {
    if ($env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
        $Arch = "amd64"
    } else {
        Err "32-bit Windows is not supported"
    }
} else {
    Err "Unsupported architecture: $envArch"
}

Info "System: Windows / $Arch"

Info "Checking latest version..."

# Get latest version using GitHub API
$LatestVersion = $null
$hasCurl = Get-Command curl.exe -ErrorAction SilentlyContinue

if ($hasCurl) {
    try {
        $apiResponse = & curl.exe -s -L "https://api.github.com/repos/$GitHubOwner/$GitHubRepo/releases"
        $releases = $apiResponse | ConvertFrom-Json
        if ($releases -and $releases.Count -gt 0) {
            $LatestVersion = $releases[0].tag_name
        }
    } catch {
        # Continue to fallback
    }
}

# Fallback to PowerShell method
if ([string]::IsNullOrEmpty($LatestVersion)) {
    try {
        $apiUrl = "https://api.github.com/repos/$GitHubOwner/$GitHubRepo/releases"
        $releases = Invoke-RestMethod -Uri $apiUrl -ErrorAction Stop
        
        if ($releases -and $releases.Count -gt 0) {
            $LatestVersion = $releases[0].tag_name
        }
    } catch {
        Err "Could not determine latest version. Please check your internet connection."
    }
}

if ([string]::IsNullOrEmpty($LatestVersion)) {
    Err "Could not determine latest version"
}

Info "Latest version: $LatestVersion"

$ArchiveName  = "keke_windows_${Arch}.zip"
$DownloadURL  = "https://github.com/$GitHubOwner/$GitHubRepo/releases/download/$LatestVersion/$ArchiveName"

$TmpDir = Join-Path $env:TEMP "keke_install_$(Get-Random)"
New-Item -Type Directory -Path $TmpDir | Out-Null

Info "Downloading $ArchiveName..."
$ArchivePath = Join-Path $TmpDir "keke.zip"

# Download using curl if available (most reliable)
$downloadSuccess = $false

if ($hasCurl) {
    try {
        & curl.exe -L -o $ArchivePath $DownloadURL --silent --show-error --fail
        if ($LASTEXITCODE -eq 0 -and (Test-Path $ArchivePath)) {
            $downloadSuccess = $true
        }
    } catch {
        # Continue to fallback
    }
}

# Fallback to .NET WebClient
if (-not $downloadSuccess) {
    try {
        $ProgressPreference = 'SilentlyContinue'
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($DownloadURL, $ArchivePath)
        
        if (Test-Path $ArchivePath) {
            $downloadSuccess = $true
        }
    } catch {
        # Continue to final fallback
    }
}

# Final fallback: Invoke-WebRequest
if (-not $downloadSuccess) {
    try {
        Invoke-WebRequest -Uri $DownloadURL -OutFile $ArchivePath -UseBasicParsing -ErrorAction Stop
        if (Test-Path $ArchivePath) {
            $downloadSuccess = $true
        }
    } catch {
        # All methods failed
    }
}

if (-not $downloadSuccess) {
    Write-Host ""
    Write-Host "  Download failed. Please try:" -ForegroundColor Yellow
    Write-Host "  1. Check your internet connection" -ForegroundColor Gray
    Write-Host "  2. Download manually from: $DownloadURL" -ForegroundColor Gray
    Write-Host ""
    Err "Download failed"
}

Success "Download complete"

Info "Extracting..."
try {
    # Try using tar.exe (Windows 10+)
    if (Get-Command tar.exe -ErrorAction SilentlyContinue) {
        & tar.exe -xf $ArchivePath -C $TmpDir 2>$null
        if ($LASTEXITCODE -ne 0) {
            # Fallback to PowerShell
            Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force
        }
    } else {
        Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force
    }
} catch {
    Err "Failed to extract archive: $($_.Exception.Message)"
}

$BinaryPath = Join-Path $TmpDir $BinaryName
if (-not (Test-Path $BinaryPath)) {
    $extractedFiles = Get-ChildItem -Path $TmpDir -Recurse | Select-Object -ExpandProperty Name
    Err "Binary '$BinaryName' not found. Found: $($extractedFiles -join ', ')"
}

Info "Installing to $InstallDir..."

if (-not (Test-Path $InstallDir)) {
    New-Item -Type Directory -Path $InstallDir | Out-Null
}

Copy-Item -Path $BinaryPath -Destination (Join-Path $InstallDir $BinaryName) -Force

# Add to PATH
$CurrentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")

if ($CurrentPath -notmatch [regex]::Escape($InstallDir)) {
    Info "Adding to PATH..."
    [System.Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Success "Added to PATH"
}

# Cleanup
Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue

Write-Host ""
Success "Keke CLI installed successfully!"
Write-Host ""

# Try to get version
try {
    $version = & (Join-Path $InstallDir $BinaryName) version 2>&1
    Info "Version: $version"
} catch {
    # Version command not available
}

Write-Host ""
Write-Host "  ╔════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "  ║          Quick Start Guide             ║" -ForegroundColor Green
Write-Host "  ╚════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "  1. Open a NEW terminal window" -ForegroundColor Yellow
Write-Host "  2. Navigate to your project:" -ForegroundColor Gray
Write-Host "     cd your-project" -ForegroundColor Cyan
Write-Host ""
Write-Host "  3. Initialize Keke:" -ForegroundColor Gray
Write-Host "     keke init" -ForegroundColor Cyan
Write-Host ""
Write-Host "  4. Login to your account:" -ForegroundColor Gray
Write-Host "     keke login" -ForegroundColor Cyan
Write-Host ""
Write-Host "  5. Start using Keke:" -ForegroundColor Gray
Write-Host "     keke ask `"your question here`"" -ForegroundColor Cyan
Write-Host "     keke research `"research topic`"" -ForegroundColor Cyan
Write-Host "     keke credits" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Need help? Run: keke --help" -ForegroundColor Gray
Write-Host ""
Info "Documentation: https://github.com/$GitHubOwner/$GitHubRepo"
Write-Host ""