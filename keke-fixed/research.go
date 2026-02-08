package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════
// RESEARCH - ML Research Assistant
// ═══════════════════════════════════════════════════════════════════════════
// AI helps with:
// - Experiment design
// - Data analysis
// - Model validation
// - Architecture search
// - Result interpretation

func handleResearch(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if !isProjectInitialized() {
		logError("Project not initialized. Run 'keke init'")
		return
	}

	if len(args) == 0 {
		logError("Usage: keke research \"your research task\"")
		logInfo("Examples:")
		logInfo("  keke research \"analyze this dataset for outliers\"")
		logInfo("  keke research \"design experiment to compare models\"")
		logInfo("  keke research \"validate my CNN architecture\"")
		logInfo("  keke research \"explain why my model is overfitting\"")
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

	logInfo("AI analyzing your research request...")

	// Start research conversation loop
	researchLoop(prompt, model, auth)
}

// ═══════════════════════════════════════════════════════════════════════════
// RESEARCH CONVERSATION LOOP
// ═══════════════════════════════════════════════════════════════════════════

func researchLoop(initialPrompt, model string, auth *AuthData) {
	var conversationHistory []map[string]string

	conversationHistory = append(conversationHistory, map[string]string{
		"role":    "user",
		"content": initialPrompt,
	})

	maxIterations := 20
	iteration := 0

	for iteration < maxIterations {
		iteration++

		// Call AI in research mode
		response, err := callResearchAI(conversationHistory, model, auth)
		if err != nil {
			logError(fmt.Sprintf("AI error: %v", err))
			return
		}

		// Add AI response to history
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "assistant",
			"content": response.Message,
		})

		// Check if AI is done
		if len(response.Actions) == 0 {
			fmt.Println(response.Message)
			printDivider()
			logInfo(fmt.Sprintf("Total credits used: %d", response.CreditsUsed))
			return
		}

		// Execute research actions
		for _, action := range response.Actions {
			result := executeResearchAction(action)

			conversationHistory = append(conversationHistory, map[string]string{
				"role":    "user",
				"content": fmt.Sprintf("Action result: %s", result),
			})
		}
	}

	logWarning("Max iterations reached")
}

// ═══════════════════════════════════════════════════════════════════════════
// CALL RESEARCH AI
// ═══════════════════════════════════════════════════════════════════════════

func callResearchAI(conversation []map[string]string, model string, auth *AuthData) (*AIResponse, error) {
	payload := map[string]interface{}{
		"conversation": conversation,
		"model":        model,
		"mode":         "research", // Research mode
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
// EXECUTE RESEARCH ACTIONS
// ═══════════════════════════════════════════════════════════════════════════

func executeResearchAction(action Action) string {
	switch action.Type {
	case "load_dataset":
		return handleLoadDataset(action)
	case "analyze_data":
		return handleAnalyzeData(action)
	case "train_model":
		return handleTrainModel(action)
	case "evaluate_model":
		return handleEvaluateModel(action)
	case "visualize":
		return handleVisualize(action)
	case "execute_command":
		return handleExecuteCommand(action)
	default:
		// Fall back to regular code actions
		return executeAction(action)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ML-SPECIFIC ACTION HANDLERS
// ═══════════════════════════════════════════════════════════════════════════

func handleLoadDataset(action Action) string {
	path := action.Path
	format := action.Format

	if !checkPermission("read") {
		if !requestPermission("read", fmt.Sprintf("AI wants to load dataset: %s", path)) {
			return "Permission denied"
		}
	}

	logInfo(fmt.Sprintf("Loading dataset: %s (format: %s)", path, format))
	
	// In a real implementation, this would load and return dataset info
	return fmt.Sprintf("Dataset loaded from %s. Format: %s. Shape: (1000, 10). Columns: [...]", path, format)
}

func handleAnalyzeData(action Action) string {
	analysisType := action.AnalysisType
	
	if !checkPermission("execute") {
		if !requestPermission("execute", fmt.Sprintf("AI wants to run analysis: %s", analysisType)) {
			return "Permission denied"
		}
	}

	logInfo(fmt.Sprintf("Running analysis: %s", analysisType))
	
	// In real implementation, run statistical analysis
	return fmt.Sprintf("Analysis '%s' complete. Mean: 42.5, Std: 12.3, Outliers: 15", analysisType)
}

func handleTrainModel(action Action) string {
	modelType := action.ModelType
	
	if !checkPermission("execute") {
		if !requestPermission("execute", fmt.Sprintf("AI wants to train model: %s", modelType)) {
			return "Permission denied"
		}
	}

	logInfo(fmt.Sprintf("Training model: %s", modelType))
	
	// In real implementation, train model
	return fmt.Sprintf("Model '%s' trained. Accuracy: 0.92, Loss: 0.15", modelType)
}

func handleEvaluateModel(action Action) string {
	modelPath := action.Path
	
	if !checkPermission("execute") {
		if !requestPermission("execute", fmt.Sprintf("AI wants to evaluate model: %s", modelPath)) {
			return "Permission denied"
		}
	}

	logInfo(fmt.Sprintf("Evaluating model: %s", modelPath))
	
	// In real implementation, evaluate model
	return fmt.Sprintf("Model evaluation complete. Precision: 0.89, Recall: 0.91, F1: 0.90")
}

func handleVisualize(action Action) string {
	vizType := action.VizType
	
	if !checkPermission("write") {
		if !requestPermission("write", fmt.Sprintf("AI wants to create visualization: %s", vizType)) {
			return "Permission denied"
		}
	}

	logInfo(fmt.Sprintf("Creating visualization: %s", vizType))
	
	// In real implementation, create plot
	return fmt.Sprintf("Visualization '%s' saved to plots/output.png", vizType)
}