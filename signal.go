package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func handleSignal(args []string) {
	if !isLoggedIn() {
		logError("Not logged in. Run 'keke login'")
		return
	}

	if len(args) == 0 {
		logError("Usage: keke signal <SYMBOL> [--timeframe 1H|4H|1D] [--provider anthropic|openai|groq|openrouter]")
		logInfo("Examples:")
		logInfo("  keke signal SPY")
		logInfo("  keke signal AAPL --timeframe 4H")
		logInfo("  keke signal TSLA --timeframe 1D --provider anthropic")
		logInfo("")
		logInfo("Popular symbols:")
		logInfo("  Stocks: SPY, QQQ, AAPL, TSLA, NVDA, MSFT")
		logInfo("  Crypto: BTCUSD, ETHUSD, SOLUSD")
		return
	}

	symbol := strings.ToUpper(args[0])
	timeframe := "4H"
	provider := "anthropic"

	for i := 1; i < len(args); i++ {
		if args[i] == "--timeframe" && i+1 < len(args) {
			timeframe = strings.ToUpper(args[i+1])
			i++
		} else if args[i] == "--provider" && i+1 < len(args) {
			provider = strings.ToLower(args[i+1])
			i++
		}
	}

	if len(symbol) < 2 {
		logError("Invalid symbol format. Examples: SPY, AAPL, TSLA, BTCUSD")
		return
	}

	validProviders := []string{"anthropic", "openai", "groq", "openrouter"}
	isValidProvider := false
	for _, vp := range validProviders {
		if provider == vp {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		logError(fmt.Sprintf("Invalid provider: %s. Valid options: anthropic, openai, groq, openrouter", provider))
		return
	}

	auth, err := readAuth()
	if err != nil {
		logError(fmt.Sprintf("Failed to read auth: %v", err))
		return
	}

	providerName := getProviderDisplayName(provider)
	symbolType := "stock"
	if strings.HasSuffix(symbol, "USD") {
		symbolType = "crypto"
	}
	
	logInfo(fmt.Sprintf("🔍 Analyzing %s (%s) on %s timeframe...", symbol, symbolType, timeframe))
	logInfo(fmt.Sprintf("🤖 Using %s AI", providerName))
	printDivider()

	signal, err := getTradeSignal(symbol, timeframe, provider, auth)
	if err != nil {
		logError(fmt.Sprintf("Signal error: %v", err))
		return
	}

	displaySignal(signal)

	printDivider()
	logInfo(fmt.Sprintf("AI Provider:  %s", getProviderDisplayName(signal.AIProvider)))
	logInfo(fmt.Sprintf("Credits used: %d", signal.CreditsUsed))
	logWarning("⚠ This is AI analysis, NOT financial advice. Trade at your own risk.")
}

func getTradeSignal(symbol, timeframe, provider string, auth *AuthData) (*TradeSignal, error) {
	payload := map[string]interface{}{
		"symbol":      symbol,
		"timeframe":   timeframe,
		"ai_provider": provider,
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

	var signal TradeSignal
	if err := json.NewDecoder(resp.Body).Decode(&signal); err != nil {
		return nil, err
	}

	return &signal, nil
}

func displaySignal(signal *TradeSignal) {
	fmt.Println()
	
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
	
	fmt.Printf("%s%s%s %s %s%s\n", bold, directionColor, directionSymbol, directionText, signal.Symbol, reset)
	fmt.Println()

	logInfo(fmt.Sprintf("Entry Price:  $%.2f", signal.EntryPrice))
	fmt.Printf("%s%sTP (Target):   $%.2f%s (+%.2f points)\n", bold, green, signal.TakeProfit, reset, signal.TPPips)
	fmt.Printf("%s%sSL (Stop):     $%.2f%s (-%.2f points)\n", bold, red, signal.StopLoss, reset, signal.SLPips)
	fmt.Println()

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

	fmt.Printf("%s━━━ Market Analysis ━━━%s\n", dim, reset)
	fmt.Println(signal.Analysis)
	fmt.Println()

	if len(signal.KeyFactors) > 0 {
		fmt.Printf("%s━━━ Key Factors ━━━%s\n", dim, reset)
		for _, factor := range signal.KeyFactors {
			fmt.Printf("  %s•%s %s\n", cyan, reset, factor)
		}
		fmt.Println()
	}

	if len(signal.Warnings) > 0 {
		fmt.Printf("%s%s━━━ Risk Warnings ━━━%s\n", bold, yellow, reset)
		for _, warning := range signal.Warnings {
			fmt.Printf("  %s⚠%s %s\n", yellow, reset, warning)
		}
		fmt.Println()
	}

	if signal.TradePlan != "" {
		fmt.Printf("%s━━━ Trade Plan ━━━%s\n", dim, reset)
		fmt.Println(signal.TradePlan)
		fmt.Println()
	}
}

func getProviderDisplayName(provider string) string {
	switch provider {
	case "anthropic":
		return "Anthropic Claude"
	case "openai":
		return "OpenAI GPT-4"
	case "groq":
		return "Groq Llama"
	case "openrouter":
		return "OpenRouter"
	default:
		return provider
	}
}

type TradeSignal struct {
	Symbol      string   `json:"symbol"`
	Direction   string   `json:"direction"`
	EntryPrice  float64  `json:"entry_price"`
	TakeProfit  float64  `json:"take_profit"`
	StopLoss    float64  `json:"stop_loss"`
	TPPips      float64  `json:"tp_pips"`
	SLPips      float64  `json:"sl_pips"`
	RiskReward  float64  `json:"risk_reward"`
	Timeframe   string   `json:"timeframe"`
	Confidence  int      `json:"confidence"`
	Analysis    string   `json:"analysis"`
	KeyFactors  []string `json:"key_factors"`
	Warnings    []string `json:"warnings"`
	TradePlan   string   `json:"trade_plan"`
	CreditsUsed int      `json:"credits_used"`
	RoundsUsed  int      `json:"rounds_used"`
	AIProvider  string   `json:"ai_provider"`
}