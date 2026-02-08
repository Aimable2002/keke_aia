package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ─── CODE - Dedicated Coding Assistant ──────────────────────────────────────
// Specifically for: creating files, modifying code, debugging, refactoring
// Separate from 'ask' which is for general questions/conversations

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
		logError("Usage: keke code \"your coding task\"")
		logInfo("Examples:")
		logInfo("  keke code \"create a calculator app\"")
		logInfo("  keke code \"add a login page\"")
		logInfo("  keke code \"fix the bug in auth.go\"")
		logInfo("  keke code \"refactor this function\" --smart")
		logInfo("  keke code \"run tests and fix failures\" --deep")
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

	// Start conversation loop with AI in CODE mode
	codeLoop(prompt, model, auth)
}

// ─── CODE CONVERSATION LOOP ──────────────────────────────────────────────────

func codeLoop(initialPrompt, model string, auth *AuthData) {
	var conversationHistory []map[string]string

	// Add initial user prompt
	conversationHistory = append(conversationHistory, map[string]string{
		"role":    "user",
		"content": initialPrompt,
	})

	maxIterations := 20
	iteration := 0

	for iteration < maxIterations {
		iteration++

		// Call AI in CODE mode
		response, err := callCodeAI(conversationHistory, model, auth)
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
			// AI is done - display final message
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
	}

	logWarning("Max iterations reached. AI may need more steps.")
}

// ─── CALL CODE AI ────────────────────────────────────────────────────────────

func callCodeAI(conversation []map[string]string, model string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"conversation": conversation,
		"model":        model,
		"mode":         "code", // CODE mode
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
