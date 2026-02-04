package main

import (
	"fmt"
	"os"
)

var version = "v0.1.0" // Injected by goreleaser

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		showHelp()
		return
	}

	command := args[0]

	switch command {
	case "version", "--version", "-v":
		fmt.Println(version)

	case "init":
		handleInit()

	case "signup":
		handleSignup()

	case "login":
		handleLogin()

	case "logout":
		handleLogout()

	case "whoami":
		handleWhoami()

	case "credits":
		handleCredits()

	case "ask":
		handleAsk(args[1:])

	case "research":
		handleResearch(args[1:])

	case "signal":
		handleSignal(args[1:])

	case "rollback":
		handleRollback(args[1:])

	case "upgrade":
		handleUpgrade()

	case "help", "--help", "-h":
		showHelp()

	default:
		logError(fmt.Sprintf("Unknown command: %s", command))
		logInfo("Run 'keke help' for available commands")
		os.Exit(1)
	}
}

func showHelp() {
	printHeader()
	logInfo("AI agent for software + ML research in your terminal")
	printDivider()
	fmt.Println()

	fmt.Println("  SOFTWARE DEVELOPMENT")
	fmt.Println()
	printCmd("init", "Initialize Keke in this project")
	printCmd("ask", "AI coding assistant (--fast/--smart/--deep)")
	printCmd("rollback", "Restore file from snapshot")
	fmt.Println()

	fmt.Println("  ML RESEARCH")
	fmt.Println()
	printCmd("research", "AI research assistant for experiments & analysis")
	fmt.Println()

	fmt.Println("  TRADING")
	fmt.Println()
	printCmd("signal", "Forex market analysis & predictions")
	fmt.Println()

	fmt.Println("  ACCOUNT")
	fmt.Println()
	printCmd("signup", "Create new account")
	printCmd("login", "Log in (Email or Gmail)")
	printCmd("logout", "Log out")
	printCmd("whoami", "Show account info")
	printCmd("credits", "Check credit balance")
	fmt.Println()

	fmt.Println("  SYSTEM")
	fmt.Println()
	printCmd("upgrade", "Update to latest version")
	printCmd("version", "Show version")
	printCmd("help", "Show this help")
	fmt.Println()

	printDivider()
	logInfo("Software:    keke ask \"add login feature\"")
	logInfo("Research:    keke research \"analyze this dataset\"")
	logInfo("Trading:     keke signal EURUSD --timeframe 4H")
	fmt.Println()
}

func printCmd(name, desc string) {
	padding := 12 - len(name)
	if padding < 1 {
		padding = 1
	}
	spaces := ""
	for i := 0; i < padding; i++ {
		spaces += " "
	}
	fmt.Printf("    \033[36m%s\033[0m%s\033[2m%s\033[0m\n", name, spaces, desc)
}