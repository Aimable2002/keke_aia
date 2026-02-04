package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ─── LOGIN ───────────────────────────────────────────────────────────────────

func handleLogin() {
	if isLoggedIn() {
		auth, _ := readAuth()
		logWarning(fmt.Sprintf("Already logged in as %s", auth.Email))
		logInfo("Run 'keke logout' first to switch accounts")
		return
	}

	logInfo("Opening browser for authentication...")

	// Generate PC hash
	pcHash, err := generatePCHash()
	if err != nil {
		logError(fmt.Sprintf("Failed to generate PC identity: %v", err))
		return
	}

	// Start local callback server
	authCodeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errorChan <- fmt.Errorf("no auth code received")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		// Send success page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Keke - Logged In</title>
	<style>
		body {
			font-family: monospace;
			background: #0a0a0a;
			color: #e2e8f0;
			display: flex;
			justify-content: center;
			align-items: center;
			height: 100vh;
			margin: 0;
		}
		.box {
			text-align: center;
			padding: 40px;
			border: 1px solid #333;
			border-radius: 8px;
		}
		.title {
			color: #a78bfa;
			font-size: 24px;
			margin-bottom: 16px;
		}
		.msg {
			color: #64748b;
			font-size: 14px;
		}
	</style>
</head>
<body>
	<div class="box">
		<div class="title">✓ Logged in</div>
		<div class="msg">You can close this window and return to your terminal</div>
	</div>
</body>
</html>`)

		authCodeChan <- code
	})

	server := &http.Server{
		Addr:    ":" + CallbackPort,
		Handler: mux,
	}

	// Check if port available
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		logError(fmt.Sprintf("Port %s is busy. Close whatever is using it and try again", CallbackPort))
		return
	}

	// Build OAuth URL - points to your Supabase function
	callbackURL := fmt.Sprintf("http://localhost:%s%s", CallbackPort, CallbackPath)
	authURL := fmt.Sprintf("%s?redirect=%s", EndpointAuth, callbackURL)

	// Open browser
	openBrowser(authURL)

	// Start server
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	logInfo("Waiting for authentication...")

	// Wait for callback or timeout
	var authCode string
	select {
	case authCode = <-authCodeChan:
		server.Close()
	case err := <-errorChan:
		server.Close()
		logError(err.Error())
		return
	case <-time.After(60 * time.Second):
		server.Close()
		logError("Authentication timed out after 60 seconds")
		return
	}

	// Exchange code for token (calls Supabase function)
	logInfo("Exchanging auth code for token...")

	payload := map[string]string{
		"code":    authCode,
		"pc_hash": pcHash,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(
		EndpointAuth+"/exchange",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		logError(fmt.Sprintf("Failed to exchange token: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		logError(fmt.Sprintf("Authentication failed: %s", string(body)))
		return
	}

	var authData AuthData
	if err := json.NewDecoder(resp.Body).Decode(&authData); err != nil {
		logError(fmt.Sprintf("Invalid response from server: %v", err))
		return
	}

	authData.PCHash = pcHash
	if err := writeAuth(&authData); err != nil {
		logError(fmt.Sprintf("Failed to save auth: %v", err))
		return
	}

	logSuccess("Logged in successfully")
	printDivider()
	logInfo(fmt.Sprintf("Account: %s", authData.Email))
	logInfo(fmt.Sprintf("Plan:    %s", authData.Plan))
	logInfo(fmt.Sprintf("PC ID:   %s", pcHash[:8]+"..."))
	printDivider()
}

// ─── LOGOUT ──────────────────────────────────────────────────────────────────

func handleLogout() {
	if !isLoggedIn() {
		logWarning("Not logged in")
		return
	}

	if err := os.Remove(globalAuthFile()); err != nil {
		logError(fmt.Sprintf("Failed to logout: %v", err))
		return
	}

	logSuccess("Logged out")
}

// ─── WHOAMI ──────────────────────────────────────────────────────────────────

func handleWhoami() {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	// Call server for fresh data
	resp, err := makeAuthenticatedRequest("GET", EndpointWhoami, nil, auth)
	if err != nil {
		logError(fmt.Sprintf("Failed to fetch user info: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		logError(fmt.Sprintf("Server error: %s", string(body)))
		return
	}

	var userData struct {
		Email   string `json:"email"`
		Plan    string `json:"plan"`
		PCHash  string `json:"pc_hash"`
		Credits int    `json:"credits_remaining"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		logError(fmt.Sprintf("Invalid response: %v", err))
		return
	}

	printDivider()
	logInfo(fmt.Sprintf("Account:  %s", userData.Email))
	logInfo(fmt.Sprintf("Plan:     %s", userData.Plan))
	logInfo(fmt.Sprintf("Credits:  %d", userData.Credits))
	logInfo(fmt.Sprintf("PC ID:    %s", auth.PCHash[:8]+"..."))
	printDivider()
}

// ─── CREDITS ─────────────────────────────────────────────────────────────────

func handleCredits() {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	// Call server for credit info (all logic on server)
	resp, err := makeAuthenticatedRequest("GET", EndpointCredits, nil, auth)
	if err != nil {
		logError(fmt.Sprintf("Failed to fetch credits: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		logError(fmt.Sprintf("Server error: %s", string(body)))
		return
	}

	var creditData struct {
		Remaining    int    `json:"remaining"`
		MonthlyLimit int    `json:"monthly_limit"`
		ResetDate    string `json:"reset_date"`
		Plan         string `json:"plan"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&creditData); err != nil {
		logError(fmt.Sprintf("Invalid response: %v", err))
		return
	}

	printDivider()
	logInfo(fmt.Sprintf("Credits:  %d / %d", creditData.Remaining, creditData.MonthlyLimit))
	logInfo(fmt.Sprintf("Plan:     %s", creditData.Plan))
	logInfo(fmt.Sprintf("Resets:   %s", creditData.ResetDate))
	printDivider()

	// Warning if low
	percentage := float64(creditData.Remaining) / float64(creditData.MonthlyLimit) * 100
	if percentage <= 20 && percentage > 0 {
		logWarning("Credit balance is low!")
	} else if creditData.Remaining == 0 {
		logError("No credits remaining. Upgrade your plan to continue.")
	}
}

// ─── PC HASH ─────────────────────────────────────────────────────────────────

func generatePCHash() (string, error) {
	var parts []string

	// Get MAC address
	mac, err := getMACAddress()
	if err == nil && mac != "" {
		parts = append(parts, mac)
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		parts = append(parts, hostname)
	}

	// On macOS: get hardware UUID
	if runtime.GOOS == "darwin" {
		uuid, err := getMacHardwareUUID()
		if err == nil && uuid != "" {
			parts = append(parts, uuid)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("could not determine machine identity")
	}

	// SHA-256 hash
	combined := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

func getMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			if len(iface.HardwareAddr) > 0 {
				return iface.HardwareAddr.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no MAC address found")
}

func getMacHardwareUUID() (string, error) {
	out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Hardware UUID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", fmt.Errorf("UUID not found")
}

// ─── HTTP HELPERS ────────────────────────────────────────────────────────────

func makeAuthenticatedRequest(method, url string, body io.Reader, auth *AuthData) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	req.Header.Set("X-PC-Hash", auth.PCHash)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	}

	exec.Command(cmd, args...).Start()
}