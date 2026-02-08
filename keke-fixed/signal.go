package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════
// SIGNAL - Forex Market Analysis & Trading Predictions
// ═══════════════════════════════════════════════════════════════════════════
// AI analyzes forex pairs and predicts:
// - Trade direction (BUY/SELL/HOLD)
// - Entry price
// - Take Profit (TP)
// - Stop Loss (SL)
// - Risk/Reward ratio
// - Timeframe
// - Professional analysis
//
// IMPORTANT: This is AI prediction, NOT financial advice
// Does NOT execute trades - only predicts and advises

func handleSignal(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if len(args) == 0 {
		logError("Usage: keke signal <PAIR> [--timeframe 1H|4H|1D]")
		logInfo("Examples:")
		logInfo("  keke signal EURUSD")
		logInfo("  keke signal GBPUSD --timeframe 4H")
		logInfo("  keke signal XAUUSD --timeframe 1D")
		logInfo("  keke signal BTCUSD --timeframe 1H")
		return
	}

	// Parse arguments
	pair := strings.ToUpper(args[0])
	timeframe := "4H" // default

	for i := 1; i < len(args); i++ {
		if args[i] == "--timeframe" && i+1 < len(args) {
			timeframe = strings.ToUpper(args[i+1])
			i++
		}
	}

	// Validate pair format
	if len(pair) < 6 {
		logError("Invalid pair format. Examples: EURUSD, GBPUSD, XAUUSD, BTCUSD")
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	logInfo(fmt.Sprintf("🔍 Analyzing %s on %s timeframe...", pair, timeframe))
	logInfo("AI is thinking deeply about market conditions...")
	printDivider()

	// Call AI for market analysis
	signal, err := getForexSignal(pair, timeframe, auth)
	if err != nil {
		logError(fmt.Sprintf("Signal error: %v", err))
		return
	}

	// Display signal
	displaySignal(signal)

	printDivider()
	logInfo(fmt.Sprintf("Credits used: %d", signal.CreditsUsed))
	logWarning("⚠ This is AI analysis, NOT financial advice. Trade at your own risk.")
}

// ═══════════════════════════════════════════════════════════════════════════
// GET FOREX SIGNAL (calls edge function)
// ═══════════════════════════════════════════════════════════════════════════

func getForexSignal(pair, timeframe string, auth *AuthData) (*ForexSignal, error) {
	payload := map[string]interface{}{
		"pair":      pair,
		"timeframe": timeframe,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := makeAuthenticatedRequest(
		"POST",
		EndpointSignal,
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

	var signal ForexSignal
	if err := json.NewDecoder(resp.Body).Decode(&signal); err != nil {
		return nil, err
	}

	return &signal, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// DISPLAY SIGNAL (beautiful terminal output)
// ═══════════════════════════════════════════════════════════════════════════

func displaySignal(signal *ForexSignal) {
	fmt.Println()
	
	// Header with direction
	directionColor := green
	directionSymbol := "▲"
	directionText := "BUY"
	
	if signal.Direction == "SELL" {
		directionColor = red
		directionSymbol = "▼"
		directionText = "SELL"
	} else if signal.Direction == "HOLD" {
		directionColor = yellow
		directionSymbol = "■"
		directionText = "HOLD"
	}
	
	fmt.Printf("%s%s%s %s %s%s\n", bold, directionColor, directionSymbol, directionText, signal.Pair, reset)
	fmt.Println()

	// Price levels
	logInfo(fmt.Sprintf("Entry Price:  %.5f", signal.EntryPrice))
	fmt.Printf("%s%sTP (Target):   %.5f%s (+%.1f pips)\n", bold, green, signal.TakeProfit, reset, signal.TPPips)
	fmt.Printf("%s%sSL (Stop):     %.5f%s (-%.1f pips)\n", bold, red, signal.StopLoss, reset, signal.SLPips)
	fmt.Println()

	// Risk/Reward & Confidence
	logInfo(fmt.Sprintf("Risk/Reward:  1:%.2f", signal.RiskReward))
	logInfo(fmt.Sprintf("Timeframe:    %s", signal.Timeframe))
	
	confidenceColor := green
	if signal.Confidence < 60 {
		confidenceColor = yellow
	}
	if signal.Confidence < 40 {
		confidenceColor = red
	}
	fmt.Printf("%s%sConfidence:   %d%%%s\n", bold, confidenceColor, signal.Confidence, reset)
	fmt.Println()

	// Market Analysis
	fmt.Printf("%s━━━ Market Analysis ━━━%s\n", dim, reset)
	fmt.Println(signal.Analysis)
	fmt.Println()

	// Key Factors
	if len(signal.KeyFactors) > 0 {
		fmt.Printf("%s━━━ Key Factors ━━━%s\n", dim, reset)
		for _, factor := range signal.KeyFactors {
			fmt.Printf("  %s•%s %s\n", cyan, reset, factor)
		}
		fmt.Println()
	}

	// Warnings
	if len(signal.Warnings) > 0 {
		fmt.Printf("%s%s━━━ Risk Warnings ━━━%s\n", bold, yellow, reset)
		for _, warning := range signal.Warnings {
			fmt.Printf("  %s⚠%s %s\n", yellow, reset, warning)
		}
		fmt.Println()
	}

	// Trade Plan
	if signal.TradePlan != "" {
		fmt.Printf("%s━━━ Trade Plan ━━━%s\n", dim, reset)
		fmt.Println(signal.TradePlan)
		fmt.Println()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

type ForexSignal struct {
	Pair        string   `json:"pair"`         // e.g., "EURUSD"
	Direction   string   `json:"direction"`    // "BUY", "SELL", "HOLD"
	EntryPrice  float64  `json:"entry_price"`  // Recommended entry
	TakeProfit  float64  `json:"take_profit"`  // TP level
	StopLoss    float64  `json:"stop_loss"`    // SL level
	TPPips      float64  `json:"tp_pips"`      // TP in pips
	SLPips      float64  `json:"sl_pips"`      // SL in pips
	RiskReward  float64  `json:"risk_reward"`  // R:R ratio
	Timeframe   string   `json:"timeframe"`    // e.g., "4H"
	Confidence  int      `json:"confidence"`   // 0-100%
	Analysis    string   `json:"analysis"`     // Detailed market analysis
	KeyFactors  []string `json:"key_factors"`  // Bullet points of key factors
	Warnings    []string `json:"warnings"`     // Risk warnings
	TradePlan   string   `json:"trade_plan"`   // Step-by-step plan
	CreditsUsed int      `json:"credits_used"` // Credits consumed
}