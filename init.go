package main

import (
	"fmt"
	"os"
)

func handleInit() {
	if isProjectInitialized() {
		logWarning("This project is already initialized (.keke/ exists)")
		logInfo("Run 'keke login' if you haven't logged in yet")
		return
	}

	logInfo("Initializing Keke in this project...")

	// Create .keke/
	if err := os.MkdirAll(projectDir(), 0755); err != nil {
		logError(fmt.Sprintf("Failed to create .keke/: %v", err))
		return
	}

	// Create snapshots/
	if err := os.MkdirAll(projectSnapshotsDir(), 0755); err != nil {
		logError(fmt.Sprintf("Failed to create snapshots/: %v", err))
		return
	}

	// Create permissions.json (empty for now, server validates)
	perms := &Permissions{}
	if err := writePermissions(perms); err != nil {
		logError(fmt.Sprintf("Failed to create permissions.json: %v", err))
		return
	}

	// Create changelog.md
	changelog := `# Keke Changelog

All changes made by Keke will be logged here.
This file helps you understand what changed and why.

---
`
	if err := os.WriteFile(projectChangelogFile(), []byte(changelog), 0644); err != nil {
		logError(fmt.Sprintf("Failed to create changelog.md: %v", err))
		return
	}

	// Create context.json (AI memory - managed by server)
	if err := os.WriteFile(projectContextFile(), []byte("{}\n"), 0644); err != nil {
		logError(fmt.Sprintf("Failed to create context.json: %v", err))
		return
	}

	// Add .keke/ to .gitignore if git repo exists
	if _, err := os.Stat(".git"); err == nil {
		addToGitignore()
	}

	logSuccess("Project initialized")
	printDivider()
	logInfo("Created .keke/")
	logInfo("  permissions.json  — permission grants (validated on server)")
	logInfo("  snapshots/        — file backups for rollback")
	logInfo("  changelog.md      — auto-generated change log")
	logInfo("  context.json      — AI working memory")
	printDivider()

	if !isLoggedIn() {
		logWarning("Not logged in. Run 'keke login' to continue")
	} else {
		logSuccess("Ready! (Phase 2 will add 'keke ask' command)")
	}
}

func addToGitignore() {
	gitignorePath := ".gitignore"
	
	// Read existing gitignore
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		// Create new gitignore
		content = []byte("")
	}

	contentStr := string(content)
	
	// Check if .keke/ already in gitignore
	if !contains(contentStr, ".keke/") {
		// Add .keke/ to gitignore
		newContent := contentStr
		if len(contentStr) > 0 && contentStr[len(contentStr)-1] != '\n' {
			newContent += "\n"
		}
		newContent += "\n# Keke AI Terminal\n.keke/\n"
		
		os.WriteFile(gitignorePath, []byte(newContent), 0644)
		logInfo("Added .keke/ to .gitignore")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		containsHelper(s[1:], substr))))
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}