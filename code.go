package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// SIMPLIFIED: No local session storage
// Backend manages everything via database
// ═══════════════════════════════════════════════════════════════════════════

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
	
	// Simple conversation loop - backend handles session persistence
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
	fmt.Println()
	logInfo("Conversation history is automatically saved on the server")
}

// ═══════════════════════════════════════════════════════════════════════════
// CONVERSATION LOOP
// ═══════════════════════════════════════════════════════════════════════════

func conversationLoop(initialPrompt, model, provider string, auth *AuthData) {
	var sessionID string
	maxIterations := 20
	iteration := 0
	totalCredits := 0

	// First message - backend will create/reuse session based on user_id
	response, err := callDatabaseAI(initialPrompt, model, provider, auth)
	if err != nil {
		logError(fmt.Sprintf("AI error: %v", err))
		return
	}

	sessionID = response.SessionID
	totalCredits += response.CreditsUsed

	// Handle first response
	continueLoop := handleAIResponseWithTools(response, model, provider, sessionID, auth, &totalCredits)
	
	if response.Done || !continueLoop {
		printDivider()
		logInfo(fmt.Sprintf("Credits used: %d", totalCredits))
		return
	}

	// Continue conversation if needed
	for iteration < maxIterations && continueLoop {
		iteration++

		response, err = callDatabaseAI("continue", model, provider, auth)
		if err != nil {
			logError(fmt.Sprintf("AI error: %v", err))
			return
		}

		totalCredits += response.CreditsUsed

		continueLoop = handleAIResponseWithTools(response, model, provider, sessionID, auth, &totalCredits)

		if response.Done || !continueLoop {
			break
		}
	}

	printDivider()
	logInfo(fmt.Sprintf("Total credits: %d", totalCredits))

	if iteration >= maxIterations {
		logWarning("Max iterations reached. Continue with another 'keke code' command.")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// RESPONSE HANDLER WITH TOOL SUPPORT
// ═══════════════════════════════════════════════════════════════════════════

func handleAIResponseWithTools(response *AIResponse, model, provider, sessionID string, auth *AuthData, totalCredits *int) bool {
	// Handle tool calls
	if len(response.ToolCalls) > 0 {
		results := executeToolCalls(response.ToolCalls)
		
		// Send results back to AI (backend will append to session)
		newResponse, err := sendToolResultsToDatabaseAI(results, model, provider, auth)
		if err != nil {
			logError(fmt.Sprintf("Failed to send tool results: %v", err))
			return false
		}
		
		*totalCredits += newResponse.CreditsUsed
		
		// Recursively handle the AI's response after receiving tool results
		return handleAIResponseWithTools(newResponse, model, provider, sessionID, auth, totalCredits)
	}

	// Handle regular response
	return handleAIResponse(response)
}

// ═══════════════════════════════════════════════════════════════════════════
// API CALLS - Database-backed sessions
// ═══════════════════════════════════════════════════════════════════════════

func callDatabaseAI(promptText, model, provider string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"message":  promptText,
		"model":    model,
		"provider": provider,
		"mode":     "code",
		"user_id":  auth.UserID, // Backend uses this to get/create session
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

func sendToolResultsToDatabaseAI(results []ToolResult, model, provider string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"tool_results": results,
		"model":        model,
		"provider":     provider,
		"mode":         "code",
		"user_id":      auth.UserID, // Backend uses this to find session
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

// ═══════════════════════════════════════════════════════════════════════════
// RESPONSE HANDLER
// ═══════════════════════════════════════════════════════════════════════════

func handleAIResponse(response *AIResponse) bool {
	message := response.Message

	// AI created a plan
	if containsPlan(message) {
		plan := extractPlan(message)
		if plan != nil {
			printDivider()
			logInfo("AI CREATED IMPLEMENTATION PLAN:")
			fmt.Println()
			displayPlanCompact(plan)
			printDivider()

			approval := prompt("Approve plan? (y/n)")
			approval = strings.ToLower(strings.TrimSpace(approval))

			if approval == "n" || approval == "no" {
				fmt.Println()
				logInfo("Plan rejected. Tell AI what to change in your next command.")
				return false
			}

			fmt.Println()
			logSuccess("Plan approved! AI will start implementation...")
			fmt.Println()
			return true
		}
	}

	// AI created code files
	filesCreated := extractAndWriteCodeBlocks(message)
	if len(filesCreated) > 0 {
		logSuccess("Created/updated:")
		for _, file := range filesCreated {
			fmt.Printf("  ✓ %s\n", file)
		}
		fmt.Println()
		return true
	}

	// AI is asking questions
	if containsQuestion(message) {
		printCleanMessage(message)
		fmt.Println()
		logInfo("Respond with another 'keke code \"your answer\"' to continue")
		return false
	}

	// AI says it's done
	if isDoneResponse(message) {
		printCleanMessage(message)
		logSuccess("Task completed!")
		return false
	}

	// Default: Show message
	printCleanMessage(message)
	return true
}

// ═══════════════════════════════════════════════════════════════════════════
// DETECTION HELPERS
// ═══════════════════════════════════════════════════════════════════════════

func containsPlan(message string) bool {
	lower := strings.ToLower(message)
	return (strings.Contains(lower, `"steps"`) && strings.Contains(lower, `"project_structure"`)) ||
		   (strings.Contains(lower, "step 1:") && strings.Contains(lower, "step 2:"))
}

func containsQuestion(message string) bool {
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "?") {
			if !strings.HasPrefix(strings.ToLower(trimmed), "what if") &&
			   !strings.HasPrefix(strings.ToLower(trimmed), "why not") {
				return true
			}
		}
	}
	return false
}

func isDoneResponse(message string) bool {
	lower := strings.ToLower(message)
	doneIndicators := []string{
		"task completed",
		"implementation complete",
		"all done",
		"finished implementing",
		"successfully created",
		"project is ready",
	}
	
	for _, indicator := range doneIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════
// PLAN STRUCTURES
// ═══════════════════════════════════════════════════════════════════════════

type ExecutionPlan struct {
	ProjectStructure []FolderStructure `json:"project_structure"`
	Technologies     []string          `json:"technologies"`
	Steps            []PlanStep        `json:"steps"`
	EstimatedTime    string            `json:"estimated_time"`
	Overview         string            `json:"overview"`
}

type FolderStructure struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type PlanStep struct {
	StepNumber  int      `json:"step_number"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Actions     []string `json:"actions"`
}

func extractPlan(message string) *ExecutionPlan {
	jsonStr := extractJSON(message)
	if jsonStr == "" {
		return nil
	}

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil
	}

	return &plan
}

func extractJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(text); i++ {
		if text[i] == '{' {
			depth++
		} else if text[i] == '}' {
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}

	return ""
}

func displayPlanCompact(plan *ExecutionPlan) {
	if plan.Overview != "" {
		fmt.Println(plan.Overview)
		fmt.Println()
	}

	if len(plan.Technologies) > 0 {
		logInfo("Technologies: " + strings.Join(plan.Technologies, ", "))
		fmt.Println()
	}

	if len(plan.Steps) > 0 {
		logInfo(fmt.Sprintf("IMPLEMENTATION PLAN (%d steps):", len(plan.Steps)))
		for _, step := range plan.Steps {
			fmt.Printf("\n  Step %d: %s\n", step.StepNumber, step.Title)
			if step.Description != "" {
				fmt.Printf("  %s\n", step.Description)
			}
			if len(step.Files) > 0 {
				fmt.Printf("  Files: %s\n", strings.Join(step.Files, ", "))
			}
		}
		fmt.Println()
	}

	if plan.EstimatedTime != "" {
		logInfo("Estimated time: " + plan.EstimatedTime)
		fmt.Println()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// CODE EXTRACTION
// ═══════════════════════════════════════════════════════════════════════════

func extractAndWriteCodeBlocks(message string) []string {
	var filesCreated []string

	pattern := regexp.MustCompile("```([a-z]*) ([^\\n]+)\\n([\\s\\S]*?)```")
	matches := pattern.FindAllStringSubmatch(message, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		filepath := strings.TrimSpace(match[2])
		content := match[3]

		if filepath != "" && content != "" {
			if writeFile(filepath, content) {
				filesCreated = append(filesCreated, filepath)
			}
		}
	}

	return filesCreated
}

func writeFile(filename, content string) bool {
	if filename == "" {
		return false
	}

	if !checkPermission("write") {
		if !requestPermission("write", fmt.Sprintf("Create/update: %s", filename)) {
			return false
		}
	}

	createSnapshot(filename)

	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logError(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
			return false
		}
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		logError(fmt.Sprintf("Failed to write %s: %v", filename, err))
		return false
	}

	return true
}

// ═══════════════════════════════════════════════════════════════════════════
// MESSAGE DISPLAY
// ═══════════════════════════════════════════════════════════════════════════

func printCleanMessage(message string) {
	codeBlockPattern := regexp.MustCompile("(?s)```[^`]*```")
	cleaned := codeBlockPattern.ReplaceAllString(message, "[code]")

	cleaned = strings.ReplaceAll(cleaned, `{"`, "[plan]")

	lines := strings.Split(cleaned, "\n")
	var nonEmpty []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "[code]" && trimmed != "[plan]" {
			nonEmpty = append(nonEmpty, line)
		}
	}

	if len(nonEmpty) > 0 {
		fmt.Println(strings.Join(nonEmpty, "\n"))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PERMISSION HELPERS
// ═══════════════════════════════════════════════════════════════════════════

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
	// Check if file exists before trying to snapshot
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist yet, skip snapshot (not an error for new files)
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