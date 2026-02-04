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

$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq [System.Runtime.InteropServices.Architecture]::X64) {
    "amd64"
} elseif ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
    "arm64"
} else {
    Err "Unsupported architecture"
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
Invoke-WebRequest -Uri $ChecksumURL -OutFile $ChecksumFile

Info "Downloading $ArchiveName..."
$ArchivePath = Join-Path $TmpDir "keke.zip"
Invoke-WebRequest -Uri $DownloadURL -OutFile $ArchivePath

Info "Verifying checksum..."

$ChecksumContent = Get-Content $ChecksumFile
$ExpectedHash = $null

foreach ($line in $ChecksumContent) {
    if ($line -match $ArchiveName) {
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

$InstalledBinary = Join-Path $InstallDir $BinaryName
if (Test-Path $InstalledBinary) {
    Success "Keke installed successfully"
    Write-Host ""
    $version = & $InstalledBinary version
    Info "Version: $version"
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