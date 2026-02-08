package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GitHub config - REPLACE WITH YOUR VALUES
const (
	githubOwner = "Aimable2002"
	githubRepo  = "keke_aia"
	apiURL      = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func handleUpgrade() {
	logInfo("Checking for updates...")

	// Get latest release from GitHub
	resp, err := http.Get(apiURL)
	if err != nil {
		logError(fmt.Sprintf("Failed to check for updates: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logError(fmt.Sprintf("GitHub API returned status %d", resp.StatusCode))
		return
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logError(fmt.Sprintf("Failed to parse release info: %v", err))
		return
	}

	latestVersion := release.TagName
	currentVersion := version

	if latestVersion == currentVersion {
		logSuccess(fmt.Sprintf("Already up to date (%s)", currentVersion))
		return
	}

	logInfo(fmt.Sprintf("Upgrading %s → %s", currentVersion, latestVersion))

	// Find correct binary for this OS/arch
	assetName := getAssetName()
	checksumName := "keke_checksums.txt"

	var downloadURL string
	var checksumURL string

	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
		}
		if asset.Name == checksumName {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		logError(fmt.Sprintf("No binary found for %s/%s", runtime.GOOS, runtime.GOARCH))
		return
	}

	// Download checksum
	var expectedChecksum string
	if checksumURL != "" {
		logInfo("Downloading checksum...")
		checksumData, err := downloadFile(checksumURL)
		if err != nil {
			logWarning("Failed to download checksum, skipping verification")
		} else {
			expectedChecksum = parseChecksum(string(checksumData), assetName)
		}
	}

	// Download binary archive
	logInfo("Downloading binary...")
	archiveData, err := downloadFile(downloadURL)
	if err != nil {
		logError(fmt.Sprintf("Failed to download: %v", err))
		return
	}

	// Verify checksum
	if expectedChecksum != "" {
		logInfo("Verifying checksum...")
		hash := sha256.Sum256(archiveData)
		actualChecksum := hex.EncodeToString(hash[:])
		if actualChecksum != expectedChecksum {
			logError("Checksum mismatch! Download may be corrupted. Aborting.")
			return
		}
		logSuccess("Checksum verified")
	}

	// Extract binary
	logInfo("Extracting...")
	binaryData, err := extractBinary(archiveData, assetName)
	if err != nil {
		logError(fmt.Sprintf("Failed to extract: %v", err))
		return
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		logError(fmt.Sprintf("Cannot determine binary path: %v", err))
		return
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	// Replace binary
	if err := os.WriteFile(execPath, binaryData, 0755); err != nil {
		logError(fmt.Sprintf("Failed to replace binary: %v", err))
		logWarning("You may need to run with sudo/admin privileges")
		return
	}

	logSuccess(fmt.Sprintf("Upgraded to %s", latestVersion))
	logInfo("Run 'keke version' to confirm")
}

func getAssetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	ext := ".tar.gz"
	if osName == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("keke_%s_%s%s", osName, arch, ext)
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func parseChecksum(checksumFile, filename string) string {
	lines := strings.Split(checksumFile, "\n")
	for _, line := range lines {
		if strings.Contains(line, filename) {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return ""
}

func extractBinary(archiveData []byte, archiveName string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".tar.gz") {
		return extractTarGz(archiveData)
	}
	if strings.HasSuffix(archiveName, ".zip") {
		return extractZip(archiveData)
	}
	return nil, fmt.Errorf("unknown archive format: %s", archiveName)
}

func extractTarGz(data []byte) ([]byte, error) {
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "keke-*.tar.gz")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	tmpFile.Close()

	f, err := os.Open(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		name := filepath.Base(header.Name)
		if name == "keke" || name == "keke.exe" {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("keke binary not found in archive")
}

func extractZip(data []byte) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "keke-*.zip")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	tmpFile.Close()

	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "keke.exe" || name == "keke" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("keke binary not found in zip")
}