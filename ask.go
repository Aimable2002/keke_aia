package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

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
		logInfo("")
		logInfo("Flags:")
		logInfo("  --fast    Fast model (fewer credits)")
		logInfo("  --smart   Smart model (default)")
		logInfo("  --deep    Deep model (highest quality)")
		return
	}

	model := "smart"
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

	response, err := callGeneralAI(prompt, model, auth)
	if err != nil {
		logError(fmt.Sprintf("AI error: %v", err))
		return
	}

	fmt.Println()
	printDivider()
	fmt.Println(response.Message)
	printDivider()
	logInfo(fmt.Sprintf("Credits used: %d", response.CreditsUsed))
}

func callGeneralAI(prompt, model string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"message": prompt,
		"model":   model,
		"mode":    "general",
	}

	jsonData, _ := json.Marshal(payload)
	
	// Use longer timeout for AI requests (2 minutes)
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

type AIResponse struct {
	Message     string     `json:"message"`
	Actions     []Action   `json:"actions"`
	SessionID   string     `json:"session_id"`
	CreditsUsed int        `json:"credits_used"`
	TokensUsed  int        `json:"tokens_used"`
	Done        bool       `json:"done"`
	ToolCalls   []ToolCall `json:"tool_calls"` // For Groq/Anthropic tool use
}

type Action struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Command string `json:"command"`
	
	Format       string                 `json:"format"`
	AnalysisType string                 `json:"analysis_type"`
	ModelType    string                 `json:"model_type"`
	VizType      string                 `json:"viz_type"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// ToolCall - Groq/Anthropic style tool call
type ToolCall struct {
	ID       string          `json:"id"`        // Tool call ID for tracking
	Type     string          `json:"type"`      // "function"
	Function FunctionCall    `json:"function"`  // Function details
}

// FunctionCall - function to execute
type FunctionCall struct {
	Name      string          `json:"name"`       // e.g., "execute_command", "write_file"
	Arguments json.RawMessage `json:"arguments"`  // JSON arguments
}

// ToolResult - result of executing a tool
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}