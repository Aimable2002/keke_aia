package main

import (
	"fmt"
	"io/ioutil"
	// "os"
	"path/filepath"
	"sort"
	"strings"
)

// ─── ROLLBACK ────────────────────────────────────────────────────────────────
// Restore files from snapshots (CLI-only, no AI involved)

func handleRollback(args []string) {
	if !isProjectInitialized() {
		logError("Project not initialized. Run 'keke init'")
		return
	}

	snapDir := projectSnapshotsDir()

	// List all snapshots
	files, err := ioutil.ReadDir(snapDir)
	if err != nil {
		logError("No snapshots found")
		return
	}

	if len(files) == 0 {
		logInfo("No snapshots available")
		return
	}

	// Group snapshots by original file
	snapshots := make(map[string][]SnapshotInfo)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".snap") {
			continue
		}

		// Parse: filename.timestamp.snap
		parts := strings.Split(file.Name(), ".")
		if len(parts) < 3 {
			continue
		}

		originalFile := strings.Join(parts[:len(parts)-2], ".")
		timestamp := parts[len(parts)-2]

		snapshots[originalFile] = append(snapshots[originalFile], SnapshotInfo{
			OriginalFile: originalFile,
			Timestamp:    timestamp,
			SnapshotFile: file.Name(),
			Path:         filepath.Join(snapDir, file.Name()),
		})
	}

	// If specific file given, filter to that
	if len(args) > 0 {
		targetFile := args[0]
		if snaps, ok := snapshots[targetFile]; ok {
			snapshots = map[string][]SnapshotInfo{targetFile: snaps}
		} else {
			logError(fmt.Sprintf("No snapshots found for: %s", targetFile))
			return
		}
	}

	// Display available snapshots
	printDivider()
	logInfo("Available snapshots:")
	fmt.Println()

	var allSnapshots []SnapshotInfo
	for _, snaps := range snapshots {
		// Sort by timestamp (newest first)
		sort.Slice(snaps, func(i, j int) bool {
			return snaps[i].Timestamp > snaps[j].Timestamp
		})
		allSnapshots = append(allSnapshots, snaps...)
	}

	for i, snap := range allSnapshots {
		fmt.Printf("  %d. %s (from %s)\n", i+1, snap.OriginalFile, snap.Timestamp)
	}

	printDivider()

	// Prompt for selection
	response := prompt("Enter number to restore (or 'c' to cancel)")
	if response == "c" || response == "" {
		logInfo("Cancelled")
		return
	}

	var index int
	fmt.Sscanf(response, "%d", &index)
	if index < 1 || index > len(allSnapshots) {
		logError("Invalid selection")
		return
	}

	snapshot := allSnapshots[index-1]

	// Confirm
	confirm := prompt(fmt.Sprintf("Restore %s? This will OVERWRITE current version. (y/n)", snapshot.OriginalFile))
	if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
		logInfo("Cancelled")
		return
	}

	// Read snapshot
	content, err := ioutil.ReadFile(snapshot.Path)
	if err != nil {
		logError(fmt.Sprintf("Failed to read snapshot: %v", err))
		return
	}

	// Write to original location
	if err := ioutil.WriteFile(snapshot.OriginalFile, content, 0644); err != nil {
		logError(fmt.Sprintf("Failed to restore: %v", err))
		return
	}

	logSuccess(fmt.Sprintf("Restored: %s", snapshot.OriginalFile))
	logInfo(fmt.Sprintf("From snapshot: %s", snapshot.Timestamp))
}

// ─── TYPES ───────────────────────────────────────────────────────────────────

type SnapshotInfo struct {
	OriginalFile string
	Timestamp    string
	SnapshotFile string
	Path         string
}