package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func handleReadFile(action Action) string {
	path := action.Path

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

func handleWriteFile(action Action) string {
	path := action.Path
	content := action.Content

	if !checkPermission("write") {
		if !requestPermission("write", fmt.Sprintf("AI wants to write: %s", path)) {
			return "Permission denied by user"
		}
	}

	if err := createSnapshot(path); err != nil {
		logWarning(fmt.Sprintf("Failed to create snapshot: %v", err))
	}

	if err := writeFileToWorkspace(path, content); err != nil {
		return fmt.Sprintf("Error writing file: %v", err)
	}

	logSuccess(fmt.Sprintf("Wrote: %s", path))
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
}

func writeFileToWorkspace(filename, content string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Don't use filepath.Base() - we want to preserve the directory structure
	// Just clean the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filename)
	
	// Prevent going outside current directory
	if strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("invalid path: cannot write outside project directory")
	}
	
	fullPath := filepath.Join(cwd, cleanPath)

	// Create all parent directories
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

func handleExecuteCommand(action Action) string {
	command := action.Command

	if !checkPermission("execute") {
		if !requestPermission("execute", fmt.Sprintf("AI wants to run: %s", command)) {
			return "Permission denied by user"
		}
	}

	logInfo(fmt.Sprintf("Running: %s", command))

	var cmd *exec.Cmd
	
	// Detect OS and use appropriate shell
	if os.PathSeparator == '\\' {
		// Windows - use cmd.exe
		cmd = exec.Command("cmd", "/C", command)
	} else {
		// Unix/Linux/Mac - use sh
		cmd = exec.Command("sh", "-c", command)
	}
	
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("Command failed: %v\nOutput: %s", err, string(output))
	}

	logSuccess("Command completed")
	return string(output)
}

func handleListFiles(action Action) string {
	dir := action.Path
	if dir == "" {
		dir = "."
	}

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

// truncate - helper to truncate long strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}