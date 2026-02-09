package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	// "regexp"
	"strings"
	"time"
)

func handleCode(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if !isProjectInitialized() {
		logError("Project not initialized. Run 'keke init'")
		return
	}

	if len(args) == 0 {
		showCodeHelp()
		return
	}

	model := "smart"
	provider := "groq"
	var promptParts []string

	for i, arg := range args {
		switch arg {
		case "--fast":
			model = "fast"
		case "--smart":
			model = "smart"
		case "--deep":
			model = "deep"
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
			}
		default:
			if i > 0 && args[i-1] == "--provider" {
				continue
			}
			promptParts = append(promptParts, arg)
		}
	}

	promptText := strings.Join(promptParts, " ")
	if promptText == "" {
		logError("No prompt provided")
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	logInfo(fmt.Sprintf("Model: %s | Provider: %s", model, provider))
	
	conversationLoop(promptText, model, provider, auth)
}

func showCodeHelp() {
	logInfo("Usage: keke code \"your task\"")
	fmt.Println()
	logInfo("Examples:")
	logInfo("  keke code \"add a README\"")
	logInfo("  keke code \"create a REST API with auth\"")
	logInfo("  keke code \"fix the bug in server.go\"")
	fmt.Println()
	logInfo("Flags:")
	logInfo("  --fast       Fast model (fewer credits)")
	logInfo("  --smart      Smart model (default)")
	logInfo("  --deep       Deep model (best quality)")
	logInfo("  --provider   Choose AI provider (groq|anthropic)")
}

func conversationLoop(initialPrompt, model, provider string, auth *AuthData) {
	var sessionID string
	totalCredits := 0

	// First message
	response, err := callDatabaseAI(initialPrompt, model, provider, "", auth)
	if err != nil {
		logError(fmt.Sprintf("AI error: %v", err))
		return
	}

	sessionID = response.SessionID
	totalCredits += response.CreditsUsed

	// Handle response (recursively handles tool calls)
	handleAIResponseWithTools(response, model, provider, sessionID, auth, &totalCredits)
	
	printDivider()
	logInfo(fmt.Sprintf("Credits used: %d", totalCredits))
}

// ✅ CRITICAL FIX: Stop when AI responds conversationally (no tool calls)
func handleAIResponseWithTools(
	response *AIResponse, 
	model, provider, sessionID string, 
	auth *AuthData, 
	totalCredits *int,
) {
	// ✅ If no tool calls, AI responded conversationally - show message and STOP
	if len(response.ToolCalls) == 0 {
		if response.Message != "" {
			printDivider()
			fmt.Println(response.Message)
		}
		return // ✅ STOP - don't continue loop
	}

	// Execute tool calls
	results := executeToolCalls(response.ToolCalls)
	
	// Send results back
	newResponse, err := sendToolResultsToDatabaseAI(results, model, provider, sessionID, auth)
	if err != nil {
		logError(fmt.Sprintf("Failed to send tool results: %v", err))
		return
	}
	
	*totalCredits += newResponse.CreditsUsed
	
	// Recursively handle the new response
	handleAIResponseWithTools(newResponse, model, provider, sessionID, auth, totalCredits)
}

func callDatabaseAI(promptText, model, provider, sessionID string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"model":    model,
		"provider": provider,
		"mode":     "code",
		"user_id":  auth.UserID,
	}

	if sessionID != "" {
		payload["session_id"] = sessionID
	}

	if promptText != "" {
		payload["message"] = promptText
	}

	jsonData, _ := json.Marshal(payload)
	
	resp, err := makeAuthenticatedRequestWithTimeout(
		"POST",
		EndpointAI,
		bytes.NewBuffer(jsonData),
		auth,
		120*time.Second,
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

func sendToolResultsToDatabaseAI(results []ToolResult, model, provider, sessionID string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"tool_results": results,
		"model":        model,
		"provider":     provider,
		"mode":         "code",
		"user_id":      auth.UserID,
		"session_id":   sessionID,
	}

	jsonData, _ := json.Marshal(payload)
	
	resp, err := makeAuthenticatedRequestWithTimeout(
		"POST",
		EndpointAI,
		bytes.NewBuffer(jsonData),
		auth,
		120*time.Second,
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

// Permission helpers
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
	}
	return false
}

func requestPermission(permType, message string) bool {
	fmt.Println()
	logWarning("PERMISSION REQUEST")
	fmt.Println(message)

	response := prompt("Allow? (y/n)")
	allowed := strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"

	if allowed {
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
		logSuccess("Permission granted")
	}

	return allowed
}

func readPermissions() (*Permissions, error) {
	data, err := os.ReadFile(projectPermissionsFile())
	if err != nil {
		return &Permissions{}, nil
	}
	var p Permissions
	json.Unmarshal(data, &p)
	return &p, nil
}

func writePermissions(perms *Permissions) error {
	data, _ := json.MarshalIndent(perms, "", "  ")
	return os.WriteFile(projectPermissionsFile(), data, 0644)
}

func createSnapshot(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	snapshotName := fmt.Sprintf("%s.%s.snap", filepath.Base(filePath), timestamp)
	snapshotPath := filepath.Join(projectSnapshotsDir(), snapshotName)

	return os.WriteFile(snapshotPath, content, 0644)
}