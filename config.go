package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	// "runtime"
)

// Version - injected at build time
var Version = "v0.1.0"

// API Configuration
const (
	APIBaseURL = "https://ecpyqmpgqzitduidnfey.supabase.co/functions/v1"
	
	EndpointAuth    = APIBaseURL + "/auth-Function"
	EndpointWhoami  = APIBaseURL + "/whoami"
	EndpointCredits = APIBaseURL + "/credit-function"
	EndpointAI      = APIBaseURL + "/swift-handler"      // Coding assistant
	EndpointSignal  = APIBaseURL + "/swift-service"  // ✅ NEW: Forex trading signals
)

// OAuth Configuration
const (
	CallbackPort = "8080"
	CallbackPath = "/callback"
)

// Global paths (~/.keke/)
func globalDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".keke")
}

func globalAuthFile() string {
	return filepath.Join(globalDir(), "auth.json")
}

// Project paths (.keke/)
func projectDir() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".keke")
}

func projectPermissionsFile() string {
	return filepath.Join(projectDir(), "permissions.json")
}

func projectSnapshotsDir() string {
	return filepath.Join(projectDir(), "snapshots")
}

func projectChangelogFile() string {
	return filepath.Join(projectDir(), "changelog.md")
}

func projectContextFile() string {
	return filepath.Join(projectDir(), "context.json")
}

// AuthData - token storage structure
type AuthData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Plan         string `json:"plan"`
	PCHash       string `json:"pc_hash"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Read auth from ~/.keke/auth.json
func readAuth() (*AuthData, error) {
	data, err := os.ReadFile(globalAuthFile())
	if err != nil {
		return nil, err
	}
	var auth AuthData
	err = json.Unmarshal(data, &auth)
	return &auth, err
}

// Write auth to ~/.keke/auth.json
func writeAuth(auth *AuthData) error {
	if err := os.MkdirAll(globalDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(globalAuthFile(), data, 0600)
}

// Check if logged in
func isLoggedIn() bool {
	_, err := readAuth()
	return err == nil
}

// Check if project initialized
func isProjectInitialized() bool {
	_, err := os.Stat(projectDir())
	return err == nil
}

// Get OS and Arch for updates
// func getOS() string {
// 	return runtime.GOOS
// }

// func getArch() string {
// 	return runtime.GOARCH
// }

// Permissions structure
type Permissions struct {
	Read    bool `json:"read"`
	Write   bool `json:"write"`
	Execute bool `json:"execute"`
}

// Write permissions to project
func writePermissions(perms *Permissions) error {
	data, err := json.MarshalIndent(perms, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectPermissionsFile(), data, 0644)
}