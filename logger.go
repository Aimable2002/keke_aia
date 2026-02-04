package main

import "fmt"

// ANSI color codes
const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
	bold    = "\033[1m"
	dim     = "\033[2m"
)

func logInfo(msg string) {
	fmt.Printf("%s%s►%s %s\n", dim, cyan, reset, msg)
}

func logSuccess(msg string) {
	fmt.Printf("%s%s✓%s %s\n", bold, green, reset, msg)
}

func logWarning(msg string) {
	fmt.Printf("%s%s⚠%s %s\n", bold, yellow, reset, msg)
}

func logError(msg string) {
	fmt.Printf("%s%s✗%s %s\n", bold, red, reset, msg)
}

func printDivider() {
	fmt.Printf("%s────────────────────────────────────────%s\n", dim, reset)
}

func printHeader() {
	fmt.Println()
	fmt.Printf("%s%s  ██╗  ██╗███████╗██╗  ██╗███████╗%s\n", bold, magenta, reset)
	fmt.Printf("%s%s  ██║ ██╔╝██╔════╝██║ ██╔╝██╔════╝%s\n", bold, magenta, reset)
	fmt.Printf("%s%s  █████╔╝ █████╗  █████╔╝ █████╗  %s\n", bold, magenta, reset)
	fmt.Printf("%s%s  ██╔═██╗ ██╔══╝  ██╔═██╗ ██╔══╝  %s\n", bold, magenta, reset)
	fmt.Printf("%s%s  ██║  ██╗███████╗██║  ██╗███████╗%s\n", bold, magenta, reset)
	fmt.Printf("%s%s  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝%s\n", bold, magenta, reset)
	fmt.Println()
}

func prompt(msg string) string {
	fmt.Printf("%s%s►%s %s ", dim, cyan, reset, msg)
	var input string
	fmt.Scanln(&input)
	return input
}