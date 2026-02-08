package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ─── ASK (General AI Chat Assistant) ─────────────────────────────────────────
// For general questions, conversations, explanations
// For CODING tasks, use 'keke code' instead
// NOTE: This DOES use credits (it's an AI chat model)

func handleAsk(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if len(args) == 0 {
		logError("Usage: keke ask \"your question\"")
		logInfo("Examples:")
		logInfo("  keke ask \"what is quantum computing?\"")
		logInfo("  keke ask \"explain how neural networks work\"")
		logInfo("  keke ask \"help me plan a research project\"")
		logInfo("")
		logInfo("For coding tasks, use: keke code \"your task\"")
		logInfo("")
		logWarning("Note: This command uses credits")
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

	logInfo("AI thinking...")

	// Call AI for general conversation (no file tools)
	response, err := callGeneralAI(prompt, model, auth)
	if err != nil {
		logError(fmt.Sprintf("AI error: %v", err))
		return
	}

	// Display response
	fmt.Println(response.Message)
	printDivider()
	logInfo(fmt.Sprintf("Total credits used: %d", response.CreditsUsed))
}

// ─── CALL GENERAL AI ─────────────────────────────────────────────────────────

func callGeneralAI(prompt, model string, auth *AuthData) (*AIResponse, error) {
	conversation := []map[string]string{
		{
			"role":    "user",
			"content": prompt,
		},
	}

	payload := map[string]interface{}{
		"conversation": conversation,
		"model":        model,
		"mode":         "general", // General conversation mode (no file tools)
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

// ─── TYPES (kept from original ask.go) ──────────────────────────────────────

type AIResponse struct {
	Message     string   `json:"message"`
	Actions     []Action `json:"actions"`
	CreditsUsed int      `json:"credits_used"`
	Done        bool     `json:"done"`
}

type Action struct {
	Type    string `json:"type"`    // read_file, write_file, execute_command, etc.
	Path    string `json:"path"`    // for file operations
	Content string `json:"content"` // for write_file
	Command string `json:"command"` // for execute_command
	
	// Research-specific fields
	Format       string                 `json:"format"`        // for load_dataset
	AnalysisType string                 `json:"analysis_type"` // for analyze_data
	ModelType    string                 `json:"model_type"`    // for train_model
	VizType      string                 `json:"viz_type"`      // for visualize
	Parameters   map[string]interface{} `json:"parameters"`    // for various actions
}
