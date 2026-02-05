# Keke CLI Installer for Windows
# Usage: irm https://raw.githubusercontent.com/Aimable2002/keke_aia/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$GitHubOwner = "Aimable2002"
$GitHubRepo  = "keke_aia"
$InstallDir  = Join-Path $env:APPDATA "keke"
$BinaryName  = "keke.exe"

function Info    { Write-Host "  ► $args" -ForegroundColor Cyan }
function Success { Write-Host "  ✓ $args" -ForegroundColor Green }
function Warn    { Write-Host "  ⚠ $args" -ForegroundColor Yellow }
function Err     { Write-Host "  ✗ $args" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  Keke CLI Installer" -ForegroundColor Cyan
Write-Host "  AI developer in your terminal" -ForegroundColor DarkGray
Write-Host ""

# Force TLS 1.2 - CRITICAL for GitHub downloads
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls11 -bor [Net.SecurityProtocolType]::Tls

Info "Detecting system..."

# Robust architecture detection with multiple fallback methods
$Arch = $null

# Method 1: Environment variable (most reliable)
$envArch = $env:PROCESSOR_ARCHITECTURE
if ($envArch -eq "AMD64") {
    $Arch = "amd64"
} elseif ($envArch -eq "ARM64") {
    $Arch = "arm64"
} elseif ($envArch -eq "x86") {
    # Check if running 32-bit PowerShell on 64-bit Windows
    if ($env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
        $Arch = "amd64"
    } elseif ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
        $Arch = "arm64"
    }
}

# Method 2: Try RuntimeInformation (modern systems)
if ([string]::IsNullOrEmpty($Arch)) {
    try {
        $procArch = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture
        if ($procArch -eq 4) {  # X64
            $Arch = "amd64"
        } elseif ($procArch -eq 8) {  # Arm64
            $Arch = "arm64"
        }
    } catch {
        # Not available on older systems, continue to next method
    }
}

# Method 3: WMI fallback (older systems)
if ([string]::IsNullOrEmpty($Arch)) {
    try {
        $wmiArch = (Get-WmiObject -Class Win32_Processor).Architecture
        if ($wmiArch -eq 9) {  # x64
            $Arch = "amd64"
        } elseif ($wmiArch -eq 12) {  # ARM64
            $Arch = "arm64"
        }
    } catch {
        # WMI failed, continue
    }
}

# Final check
if ([string]::IsNullOrEmpty($Arch)) {
    Write-Host ""
    Write-Host "  Unable to detect system architecture automatically." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Detected values:" -ForegroundColor Gray
    Write-Host "    PROCESSOR_ARCHITECTURE: $env:PROCESSOR_ARCHITECTURE" -ForegroundColor Gray
    Write-Host "    PROCESSOR_ARCHITEW6432: $env:PROCESSOR_ARCHITEW6432" -ForegroundColor Gray
    Write-Host ""
    Err "Unsupported or undetectable architecture. Please report this issue at https://github.com/$GitHubOwner/$GitHubRepo/issues"
}

Info "System: windows / $Arch"

Info "Checking latest version..."

# Get latest version - handles pre-releases
$LatestVersion = $null
$LatestRelease = $null

try {
    # Use GitHub API to get all releases (including pre-releases)
    $apiUrl = "https://api.github.com/repos/$GitHubOwner/$GitHubRepo/releases"
    $releases = Invoke-RestMethod -Uri $apiUrl -ErrorAction Stop
    
    if ($releases -and $releases.Count -gt 0) {
        # Get the most recent release (first in the list)
        $LatestRelease = $releases[0]
        $LatestVersion = $LatestRelease.tag_name
    }
} catch {
    # Silently continue to fallback method
}

# Fallback: try to get tags if releases API failed
if ([string]::IsNullOrEmpty($LatestVersion)) {
    try {
        $tagsUrl = "https://api.github.com/repos/$GitHubOwner/$GitHubRepo/tags"
        $tags = Invoke-RestMethod -Uri $tagsUrl -ErrorAction Stop
        
        if ($tags -and $tags.Count -gt 0) {
            $LatestVersion = $tags[0].name
        }
    } catch {
        # Continue to final fallback
    }
}

if ([string]::IsNullOrEmpty($LatestVersion)) {
    Err "Could not determine latest version. Please check your internet connection and verify that releases exist at https://github.com/$GitHubOwner/$GitHubRepo/releases"
}

Info "Latest version: $LatestVersion"

$ArchiveName  = "keke_windows_${Arch}.zip"
$DownloadURL  = "https://github.com/$GitHubOwner/$GitHubRepo/releases/download/$LatestVersion/$ArchiveName"
$ChecksumURL  = "https://github.com/$GitHubOwner/$GitHubRepo/releases/download/$LatestVersion/keke_checksums.txt"

$TmpDir = [System.IO.Path]::GetTempFileName()
Remove-Item $TmpDir
New-Item -Type Directory -Path $TmpDir | Out-Null

$cleanupBlock = { if (Test-Path $TmpDir) { Remove-Item -Recurse -Force $TmpDir } }
Register-EngineEvent PowerShell.Exiting -Action $cleanupBlock | Out-Null

# Try to download and verify checksums (optional but recommended)
$ChecksumVerified = $false
Info "Downloading checksums..."
$ChecksumFile = Join-Path $TmpDir "checksums.txt"
try {
    # Disable progress bar for compatibility
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $ChecksumURL -OutFile $ChecksumFile -UseBasicParsing -ErrorAction Stop
    $ChecksumVerified = $true
} catch {
    Warn "Could not download checksums (this is okay for now)"
}

Info "Downloading $ArchiveName..."
$ArchivePath = Join-Path $TmpDir "keke.zip"

# Try multiple download methods for maximum compatibility
$downloadSuccess = $false

# Method 1: Invoke-WebRequest with UseBasicParsing (most compatible)
try {
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $DownloadURL -OutFile $ArchivePath -UseBasicParsing -TimeoutSec 300 -ErrorAction Stop
    $downloadSuccess = $true
    $ProgressPreference = 'Continue'
} catch {
    $method1Error = $_.Exception.Message
}

# Method 2: .NET WebClient (fallback for older systems)
if (-not $downloadSuccess) {
    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($DownloadURL, $ArchivePath)
        $downloadSuccess = $true
    } catch {
        $method2Error = $_.Exception.Message
    }
}

# Method 3: Start-BitsTransfer (last resort, requires BITS service)
if (-not $downloadSuccess) {
    try {
        Import-Module BitsTransfer -ErrorAction SilentlyContinue
        Start-BitsTransfer -Source $DownloadURL -Destination $ArchivePath -ErrorAction Stop
        $downloadSuccess = $true
    } catch {
        $method3Error = $_.Exception.Message
    }
}

if (-not $downloadSuccess) {
    Write-Host ""
    Write-Host "  Failed to download using all methods:" -ForegroundColor Red
    if ($method1Error) { Write-Host "  - Invoke-WebRequest: $method1Error" -ForegroundColor Gray }
    if ($method2Error) { Write-Host "  - WebClient: $method2Error" -ForegroundColor Gray }
    if ($method3Error) { Write-Host "  - BITS: $method3Error" -ForegroundColor Gray }
    Write-Host ""
    Write-Host "  Please try:" -ForegroundColor Yellow
    Write-Host "  1. Download manually from: $DownloadURL" -ForegroundColor Gray
    Write-Host "  2. Extract to: $InstallDir" -ForegroundColor Gray
    Write-Host "  3. Add to PATH: $InstallDir" -ForegroundColor Gray
    Write-Host ""
    Err "Download failed"
}

Success "Download complete"

if ($ChecksumVerified) {
    Info "Verifying checksum..."

    $ChecksumContent = Get-Content $ChecksumFile
    $ExpectedHash = $null

    foreach ($line in $ChecksumContent) {
        if ($line -match [regex]::Escape($ArchiveName)) {
            $ExpectedHash = ($line -split "\s+")[0]
            break
        }
    }

    if ([string]::IsNullOrEmpty($ExpectedHash)) {
        Warn "Checksum not found for $ArchiveName, continuing anyway"
    } else {
        $ActualHash = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash.ToLower()

        if ($ActualHash -ne $ExpectedHash) {
            Err "Checksum mismatch! Expected: $ExpectedHash, Got: $ActualHash`nThis may indicate a corrupted download or security issue."
        }
        Success "Checksum verified"
    }
}

Info "Extracting..."
try {
    Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force
} catch {
    Err "Failed to extract archive: $($_.Exception.Message)"
}

$BinaryPath = Join-Path $TmpDir $BinaryName
if (-not (Test-Path $BinaryPath)) {
    # List what's actually in the archive for debugging
    $extractedFiles = Get-ChildItem -Path $TmpDir -Recurse | Select-Object -ExpandProperty Name
    Err "Binary '$BinaryName' not found in archive. Found files: $($extractedFiles -join ', ')"
}

Info "Installing to $InstallDir..."

if (-not (Test-Path $InstallDir)) {
    New-Item -Type Directory -Path $InstallDir | Out-Null
}

try {
    Copy-Item -Path $BinaryPath -Destination (Join-Path $InstallDir $BinaryName) -Force
} catch {
    Err "Failed to copy binary to installation directory: $($_.Exception.Message)"
}

# Add to PATH if not already there
$CurrentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")

if ($CurrentPath -notmatch [regex]::Escape($InstallDir)) {
    Info "Adding to PATH..."
    try {
        [System.Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
    } catch {
        Warn "Could not add to PATH automatically. Please add manually: $InstallDir"
    }
}

# Cleanup temp directory
try {
    if (Test-Path $TmpDir) {
        Remove-Item -Recurse -Force $TmpDir
    }
} catch {
    # Cleanup failure is not critical
}

$InstalledBinary = Join-Path $InstallDir $BinaryName
if (Test-Path $InstalledBinary) {
    Success "Keke installed successfully"
    Write-Host ""
    
    # Try to get version info
    try {
        $version = & $InstalledBinary version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Info "Version: $version"
        }
    } catch {
        # Version command not available, that's okay
    }
    
    Write-Host ""
    Info "Next steps:"
    Write-Host "  1. Restart your terminal (or run: `$env:Path = [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'))" -ForegroundColor Gray
    Write-Host "  2. cd your-project" -ForegroundColor Gray
    Write-Host "  3. keke init" -ForegroundColor Gray
    Write-Host "  4. keke login" -ForegroundColor Gray
    Write-Host "  5. keke credits" -ForegroundColor Gray
    Write-Host ""
    Info "Documentation: https://github.com/$GitHubOwner/$GitHubRepo"
    Write-Host ""
} else {
    Err "Installation failed - binary not found at expected location"
}