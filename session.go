package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// SessionData - persisted conversation state
type SessionData struct {
	SessionID   string    `json:"session_id"`
	Model       string    `json:"model"`
	Provider    string    `json:"provider"`
	LastCommand string    `json:"last_command"`
	UpdatedAt   time.Time `json:"updated_at"`
	MessageCount int      `json:"message_count"`
}

func projectSessionFile() string {
	return filepath.Join(projectDir(), "session.json")
}

// loadSession - load active session from disk
func loadSession() (*SessionData, error) {
	data, err := os.ReadFile(projectSessionFile())
	if err != nil {
		return nil, err
	}
	
	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	
	// Check if session is too old (>1 hour = expired)
	if time.Since(session.UpdatedAt) > time.Hour {
		return nil, os.ErrNotExist // Expired session
	}
	
	return &session, nil
}

// saveSession - persist session to disk
func saveSession(session *SessionData) error {
	session.UpdatedAt = time.Now()
	
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(projectSessionFile(), data, 0644)
}

// clearSession - delete session file
func clearSession() error {
	return os.Remove(projectSessionFile())
}

// hasActiveSession - check if there's a valid session
func hasActiveSession() bool {
	session, err := loadSession()
	return err == nil && session != nil
}