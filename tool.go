package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	// "strings"
)

// ═══════════════════════════════════════════════════════════════════════════
// TOOL EXECUTION WITH GROQ COMPATIBILITY
// Groq sometimes sends malformed tool arguments - we handle it gracefully
// ═══════════════════════════════════════════════════════════════════════════

// ExecuteToolCalls - handle tool calls from AI with user permission
func executeToolCalls(toolCalls []ToolCall) []ToolResult {
	var results []ToolResult
	
	if len(toolCalls) == 0 {
		return results
	}

	fmt.Println()
	printDivider()
	logInfo(fmt.Sprintf("AI wants to execute %d action(s)", len(toolCalls)))
	printDivider()
	fmt.Println()

	for i, toolCall := range toolCalls {
		fmt.Printf("[%d/%d] ", i+1, len(toolCalls))
		result := executeToolCall(toolCall)
		results = append(results, result)
		fmt.Println()
	}

	return results
}

// executeToolCall - execute a single tool call with permission
func executeToolCall(toolCall ToolCall) ToolResult {
	funcName := toolCall.Function.Name
	
	// Show what AI wants to do
	displayToolRequest(toolCall)

	// Get user permission (unless already granted for this permission type)
	permType := getPermissionType(funcName)
	if !checkPermission(permType) {
		message := formatToolPermissionMessage(toolCall)
		if !requestPermission(permType, message) {
			return ToolResult{
				ToolCallID: toolCall.ID,
				Error:      "Permission denied by user",
			}
		}
	}

	// Execute the tool
	output, err := dispatchToolCall(toolCall)
	
	result := ToolResult{
		ToolCallID: toolCall.ID,
		Output:     output,
	}
	
	if err != nil {
		result.Error = err.Error()
		logError(fmt.Sprintf("✗ Failed: %v", err))
	} else {
		logSuccess("✓ Completed")
		if output != "" && len(output) < 200 {
			fmt.Printf("  Output: %s\n", output)
		} else if output != "" {
			fmt.Printf("  Output: %s... (%d bytes)\n", truncate(output, 100), len(output))
		}
	}

	return result
}

// displayToolRequest - show what the AI wants to do (with Groq argument parsing)
func displayToolRequest(toolCall ToolCall) {
	funcName := toolCall.Function.Name
	
	switch funcName {
	case "execute_command":
		cmd := parseCommandArgs(toolCall.Function.Arguments)
		logInfo(fmt.Sprintf("🔧 Execute: %s", cmd))
		
	case "write_file":
		path, _ := parseWriteFileArgs(toolCall.Function.Arguments)
		logInfo(fmt.Sprintf("📝 Write: %s", path))
		
	case "read_file":
		path := parseReadFileArgs(toolCall.Function.Arguments)
		logInfo(fmt.Sprintf("📖 Read: %s", path))
		
	case "list_files":
		path := parseListFilesArgs(toolCall.Function.Arguments)
		logInfo(fmt.Sprintf("📁 List: %s", path))
		
	default:
		logInfo(fmt.Sprintf("🔧 Tool: %s", funcName))
	}
}

// formatToolPermissionMessage - create permission message
func formatToolPermissionMessage(toolCall ToolCall) string {
	funcName := toolCall.Function.Name
	
	switch funcName {
	case "execute_command":
		cmd := parseCommandArgs(toolCall.Function.Arguments)
		return fmt.Sprintf("Execute command: %s", cmd)
		
	case "write_file":
		path, _ := parseWriteFileArgs(toolCall.Function.Arguments)
		return fmt.Sprintf("Write to file: %s", path)
		
	case "read_file":
		path := parseReadFileArgs(toolCall.Function.Arguments)
		return fmt.Sprintf("Read file: %s", path)
		
	case "list_files":
		return "List files in workspace"
		
	default:
		return fmt.Sprintf("Use tool: %s", funcName)
	}
}

// getPermissionType - map tool to permission type
func getPermissionType(toolName string) string {
	switch toolName {
	case "execute_command":
		return "execute"
	case "write_file":
		return "write"
	case "read_file", "list_files":
		return "read"
	default:
		return "execute"
	}
}

// dispatchToolCall - execute the actual tool
func dispatchToolCall(toolCall ToolCall) (string, error) {
	funcName := toolCall.Function.Name
	
	switch funcName {
	case "execute_command":
		return executeCommandTool(toolCall.Function.Arguments)
		
	case "write_file":
		return writeFileTool(toolCall.Function.Arguments)
		
	case "read_file":
		return readFileTool(toolCall.Function.Arguments)
		
	case "list_files":
		return listFilesTool(toolCall.Function.Arguments)
		
	default:
		return "", fmt.Errorf("unknown tool: %s", funcName)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GROQ-COMPATIBLE ARGUMENT PARSERS
// Groq sometimes sends strings instead of JSON objects
// ═══════════════════════════════════════════════════════════════════════════

func parseCommandArgs(argsJSON json.RawMessage) string {
	var args struct {
		Command string `json:"command"`
	}
	
	// Try direct unmarshal first
	if err := json.Unmarshal(argsJSON, &args); err == nil && args.Command != "" {
		return args.Command
	}
	
	// Try as JSON string (Groq/OpenAI format)
	var argsString string
	if err := json.Unmarshal(argsJSON, &argsString); err == nil {
		// Try to parse the string as JSON
		if err := json.Unmarshal([]byte(argsString), &args); err == nil && args.Command != "" {
			return args.Command
		}
		// If it's just a plain string, return it
		return argsString
	}
	
	// Last resort: return raw JSON as string
	return string(argsJSON)
}

func parseWriteFileArgs(argsJSON json.RawMessage) (string, string) {
	// Groq/OpenAI return arguments as a JSON string, not an object
	// So we might have: "{"path":"...","content":"..."}" (note the quotes)
	// We need to unmarshal twice: once to get the string, once to parse the JSON
	
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	
	// Try direct unmarshal first (in case it's already an object)
	err := json.Unmarshal(argsJSON, &args)
	if err == nil && args.Path != "" {
		return args.Path, args.Content
	}
	
	// Try unmar shaling as string first (Groq/OpenAI format)
	var argsString string
	if err := json.Unmarshal(argsJSON, &argsString); err == nil {
		// Now unmarshal the string as JSON
		if err := json.Unmarshal([]byte(argsString), &args); err == nil {
			return args.Path, args.Content
		}
	}
	
	return "", ""
}

func parseReadFileArgs(argsJSON json.RawMessage) string {
	var args struct {
		Path string `json:"path"`
	}
	
	// Try direct unmarshal first
	if err := json.Unmarshal(argsJSON, &args); err == nil && args.Path != "" {
		return args.Path
	}
	
	// Try as JSON string (Groq/OpenAI format)
	var argsString string
	if err := json.Unmarshal(argsJSON, &argsString); err == nil {
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(argsString), &args); err == nil && args.Path != "" {
			return args.Path
		}
		// Return the string itself if not empty
		if argsString != "" {
			return argsString
		}
	}
	
	return "."
}

func parseListFilesArgs(argsJSON json.RawMessage) string {
	var args struct {
		Path string `json:"path"`
	}
	
	// Try direct unmarshal first
	if err := json.Unmarshal(argsJSON, &args); err == nil && args.Path != "" {
		return args.Path
	}
	
	// Try as JSON string (Groq/OpenAI format)
	var argsString string
	if err := json.Unmarshal(argsJSON, &argsString); err == nil {
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(argsString), &args); err == nil && args.Path != "" {
			return args.Path
		}
		// Return the string itself if not empty
		if argsString != "" {
			return argsString
		}
	}
	
	return "."
}

// ═══════════════════════════════════════════════════════════════════════════
// TOOL IMPLEMENTATIONS
// ═══════════════════════════════════════════════════════════════════════════

func executeCommandTool(argsJSON json.RawMessage) (string, error) {
	command := parseCommandArgs(argsJSON)
	
	if command == "" {
		return "", fmt.Errorf("no command provided")
	}

	action := Action{
		Type:    "execute_command",
		Command: command,
	}
	
	output := handleExecuteCommand(action)
	return output, nil
}

func writeFileTool(argsJSON json.RawMessage) (string, error) {
	path, content := parseWriteFileArgs(argsJSON)
	
	if path == "" {
		return "", fmt.Errorf("could not extract file path from arguments")
	}

	action := Action{
		Type:    "write_file",
		Path:    path,
		Content: content,
	}
	
	output := handleWriteFile(action)
	return output, nil
}

func readFileTool(argsJSON json.RawMessage) (string, error) {
	path := parseReadFileArgs(argsJSON)
	
	if path == "" {
		return "", fmt.Errorf("no path provided")
	}

	action := Action{
		Type: "read_file",
		Path: path,
	}
	
	output := handleReadFile(action)
	return output, nil
}

func listFilesTool(argsJSON json.RawMessage) (string, error) {
	path := parseListFilesArgs(argsJSON)

	action := Action{
		Type: "list_files",
		Path: path,
	}
	
	output := handleListFiles(action)
	return output, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// SEND TOOL RESULTS BACK TO AI
// ═══════════════════════════════════════════════════════════════════════════

func sendToolResultsToAI(results []ToolResult, model, provider, sessionID string, auth *AuthData) (*AIResponse, error) {
	// Ensure we have a session ID
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required when sending tool results")
	}

	payload := map[string]interface{}{
		"tool_results": results,
		"model":        model,
		"provider":     provider,
		"session_id":   sessionID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

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