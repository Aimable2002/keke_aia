# Keke CLI Installer for Windows
# Usage: irm https://install.keke.dev/win | iex

$ErrorActionPreference = "Stop"

$GitHubOwner = "Aimable2002"
$GitHubRepo  = "keke_aia"
$InstallDir  = Join-Path $env:APPDATA "keke"
$BinaryName  = "keke.exe"

function Info    { Write-Host "  ► $args" -ForegroundColor Cyan }
function Success { Write-Host "  ✓ $args" -ForegroundColor Green }
function Err     { Write-Host "  ✗ $args" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  Keke CLI Installer" -ForegroundColor Cyan
Write-Host "  AI developer in your terminal" -ForegroundColor DarkGray
Write-Host ""

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

try {
    $response = Invoke-WebRequest -Uri "https://github.com/$GitHubOwner/$GitHubRepo/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue
} catch {
    $response = $_.Exception.Response
}

$LatestVersion = $response.Headers["Location"] -split "/" | Select-Object -Last 1

if ([string]::IsNullOrEmpty($LatestVersion)) {
    Err "Could not determine latest version"
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

Info "Downloading checksums..."
$ChecksumFile = Join-Path $TmpDir "checksums.txt"
try {
    Invoke-WebRequest -Uri $ChecksumURL -OutFile $ChecksumFile
} catch {
    Err "Failed to download checksums: $_"
}

Info "Downloading $ArchiveName..."
$ArchivePath = Join-Path $TmpDir "keke.zip"
try {
    Invoke-WebRequest -Uri $DownloadURL -OutFile $ArchivePath
} catch {
    Err "Failed to download binary: $_"
}

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
    Err "Checksum not found for $ArchiveName"
}

$ActualHash = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash.ToLower()

if ($ActualHash -ne $ExpectedHash) {
    Err "Checksum mismatch! Expected: $ExpectedHash, Got: $ActualHash"
}

Success "Checksum verified"

Info "Extracting..."
Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

$BinaryPath = Join-Path $TmpDir $BinaryName
if (-not (Test-Path $BinaryPath)) {
    Err "Binary not found in archive"
}

Info "Installing to $InstallDir..."

if (-not (Test-Path $InstallDir)) {
    New-Item -Type Directory -Path $InstallDir | Out-Null
}

Copy-Item -Path $BinaryPath -Destination (Join-Path $InstallDir $BinaryName) -Force

$CurrentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")

if ($CurrentPath -notmatch [regex]::Escape($InstallDir)) {
    Info "Adding to PATH..."
    [System.Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Cleanup temp directory
if (Test-Path $TmpDir) {
    Remove-Item -Recurse -Force $TmpDir
}

$InstalledBinary = Join-Path $InstallDir $BinaryName
if (Test-Path $InstalledBinary) {
    Success "Keke installed successfully"
    Write-Host ""
    try {
        $version = & $InstalledBinary version 2>&1
        Info "Version: $version"
    } catch {
        # Version command failed, but installation succeeded
    }
    Info "Next steps:"
    Write-Host "  1. cd your-project"
    Write-Host "  2. keke init"
    Write-Host "  3. keke login"
    Write-Host "  4. keke credits"
    Write-Host ""
    Info "Restart your terminal for PATH changes to take effect"
} else {
    Err "Installation failed"
}