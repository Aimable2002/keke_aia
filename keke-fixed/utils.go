package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ─── EXECUTE ACTION ──────────────────────────────────────────────────────────
// CLI executes actions requested by AI (with permission checks)

func executeAction(action Action) string {
	switch action.Type {
	case "read_file":
		return handleReadFile(action)
	case "write_file":
		return handleWriteFile(action)
	case "execute_command":
		return handleExecuteCommand(action)
	case "list_files":
		return handleListFiles(action)
	default:
		return fmt.Sprintf("Unknown action type: %s", action.Type)
	}
}

// ─── READ FILE ───────────────────────────────────────────────────────────────

func handleReadFile(action Action) string {
	path := action.Path

	// Check permission
	if !checkPermission("read") {
		if !requestPermission("read", fmt.Sprintf("AI wants to read: %s", path)) {
			return "Permission denied by user"
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	logInfo(fmt.Sprintf("Read: %s (%d bytes)", path, len(content)))
	return string(content)
}

// ─── WRITE FILE ──────────────────────────────────────────────────────────────

func handleWriteFile(action Action) string {
	path := action.Path
	content := action.Content

	// Check permission
	if !checkPermission("write") {
		if !requestPermission("write", fmt.Sprintf("AI wants to write: %s", path)) {
			return "Permission denied by user"
		}
	}

	// Create snapshot BEFORE writing (CLI-side, no AI involved)
	if err := createSnapshot(path); err != nil {
		logWarning(fmt.Sprintf("Failed to create snapshot: %v", err))
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	logSuccess(fmt.Sprintf("Wrote: %s", path))
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
}

// ─── EXECUTE COMMAND ─────────────────────────────────────────────────────────

func handleExecuteCommand(action Action) string {
	command := action.Command

	// Check permission
	if !checkPermission("execute") {
		if !requestPermission("execute", fmt.Sprintf("AI wants to run: %s", command)) {
			return "Permission denied by user"
		}
	}

	logInfo(fmt.Sprintf("Running: %s", command))

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("Command failed: %v\nOutput: %s", err, string(output))
	}

	logSuccess("Command completed")
	return string(output)
}

// ─── LIST FILES ──────────────────────────────────────────────────────────────

func handleListFiles(action Action) string {
	dir := action.Path
	if dir == "" {
		dir = "."
	}

	// Check permission
	if !checkPermission("read") {
		if !requestPermission("read", "AI wants to list files in workspace") {
			return "Permission denied by user"
		}
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .keke, .git, node_modules
		if strings.Contains(path, ".keke") || strings.Contains(path, ".git") || strings.Contains(path, "node_modules") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error listing files: %v", err)
	}

	logInfo(fmt.Sprintf("Listed %d files", len(files)))
	return strings.Join(files, "\n")
}

// ─── PERMISSION CHECKING ─────────────────────────────────────────────────────

func checkPermission(permType string) bool {
	perms, err := readPermissions()
	if err != nil {
		return false
	}

	switch permType {
	case "read":
		return perms.Read
	case "write":
		return perms.Write
	case "execute":
		return perms.Execute
	default:
		return false
	}
}

func requestPermission(permType, message string) bool {
	fmt.Println()
	logWarning("PERMISSION REQUEST")
	fmt.Println(message)
	fmt.Println()

	response := prompt("Allow? (y/n)")
	allowed := strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"

	if allowed {
		// Save permission
		perms, _ := readPermissions()
		switch permType {
		case "read":
			perms.Read = true
		case "write":
			perms.Write = true
		case "execute":
			perms.Execute = true
		}
		writePermissions(perms)
		logSuccess("Permission granted and saved")
	} else {
		logError("Permission denied")
	}

	return allowed
}

// ─── SNAPSHOT (CLI-SIDE, NO AI) ──────────────────────────────────────────────

func createSnapshot(filePath string) error {
	// Check if file exists
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err // File doesn't exist yet, no snapshot needed
	}

	// Create snapshot filename
	timestamp := time.Now().Format("20060102_150405")
	snapshotName := fmt.Sprintf("%s.%s.snap", filepath.Base(filePath), timestamp)
	snapshotPath := filepath.Join(projectSnapshotsDir(), snapshotName)

	// Write snapshot
	if err := os.WriteFile(snapshotPath, content, 0644); err != nil {
		return err
	}

	logInfo(fmt.Sprintf("Snapshot: %s", snapshotName))
	return nil
}

func readPermissions() (*Permissions, error) {
	data, err := os.ReadFile(projectPermissionsFile())
	if err != nil {
		return &Permissions{}, nil // Return empty permissions if file doesn't exist
	}
	var perms Permissions
	json.Unmarshal(data, &perms)
	return &perms, nil
}
