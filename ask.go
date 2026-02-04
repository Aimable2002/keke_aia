package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ─── ASK (LAM - Large Action Model) ──────────────────────────────────────────
// AI can READ workspace, WRITE files, and EXECUTE commands

func handleAsk(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if !isProjectInitialized() {
		logError("Project not initialized. Run 'keke init'")
		return
	}

	if len(args) == 0 {
		logError("Usage: keke ask \"your prompt\"")
		logInfo("Examples:")
		logInfo("  keke ask \"add a login page\"")
		logInfo("  keke ask \"fix the bug in auth.go\"")
		logInfo("  keke ask \"run tests and fix any failures\"")
		return
	}

	// Parse flags
	model := "smart" // default
	var promptParts []string

	for _, arg := range args {
		switch arg {
		case "--fast":
			model = "fast"
		case "--smart":
			model = "smart"
		case "--deep":
			model = "deep"
		default:
			promptParts = append(promptParts, arg)
		}
	}

	prompt := strings.Join(promptParts, " ")
	if prompt == "" {
		logError("No prompt provided")
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	logInfo("AI analyzing workspace...")

	// Start conversation loop with AI
	conversationLoop(prompt, model, auth)
}

// ─── CONVERSATION LOOP ───────────────────────────────────────────────────────
// AI can request actions, CLI executes them, sends results back

func conversationLoop(initialPrompt, model string, auth *AuthData) {
	var conversationHistory []map[string]string

	// Add initial user prompt
	conversationHistory = append(conversationHistory, map[string]string{
		"role":    "user",
		"content": initialPrompt,
	})

	maxIterations := 20 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++

		// Send current conversation to AI (via Supabase)
		response, err := callAI(conversationHistory, model, auth)
		if err != nil {
			logError(fmt.Sprintf("AI error: %v", err))
			return
		}

		// Add AI response to history
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "assistant",
			"content": response.Message,
		})

		// Check if AI wants to perform actions
		if len(response.Actions) == 0 {
			// AI is done - just display final message
			fmt.Println(response.Message)
			printDivider()
			logInfo(fmt.Sprintf("Total credits used: %d", response.CreditsUsed))
			return
		}

		// AI requested actions - execute them
		for _, action := range response.Actions {
			result := executeAction(action)

			// Add action result to conversation
			conversationHistory = append(conversationHistory, map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("Action result: %s", result),
			})
		}

		// Continue loop - send results back to AI
	}

	logWarning("Max iterations reached. AI may need more steps.")
}

// ─── CALL AI ─────────────────────────────────────────────────────────────────
// Sends conversation to Supabase, which calls Anthropic/OpenAI

func callAI(conversation []map[string]string, model string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"conversation": conversation,
		"model":        model,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := makeAuthenticatedRequest(
		"POST",
		EndpointAI,
		bytes.NewBuffer(jsonData),
		auth,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 402 {
		return nil, fmt.Errorf("insufficient credits")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s", string(body))
	}

	var response AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

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

// ─── TYPES ───────────────────────────────────────────────────────────────────

type AIResponse struct {
	Message     string   `json:"message"`
	Actions     []Action `json:"actions"`
	CreditsUsed int      `json:"credits_used"`
	Done        bool     `json:"done"`
}

// Add to existing Action type in ask.go

type Action struct {
	Type    string `json:"type"`    // read_file, write_file, execute_command, etc.
	Path    string `json:"path"`    // for file operations
	Content string `json:"content"` // for write_file
	Command string `json:"command"` // for execute_command
	
	// ✅ NEW: Research-specific fields
	Format       string                 `json:"format"`        // for load_dataset
	AnalysisType string                 `json:"analysis_type"` // for analyze_data
	ModelType    string                 `json:"model_type"`    // for train_model
	VizType      string                 `json:"viz_type"`      // for visualize
	Parameters   map[string]interface{} `json:"parameters"`    // for various actions
}