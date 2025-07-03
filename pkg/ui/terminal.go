package ui

import (
	"fmt"
	"os"
)

// ASCII logo for the application
const ASCIILogo = `
    ╔══════════════════════════════════════════════════════════════╗
    ║ ██╗███╗   ██╗███████╗████████╗ █████╗  ██████╗ ██████╗  █████╗  ║
    ║ ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██╔════╝ ██╔══██╗██╔══██╗ ║
    ║ ██║██╔██╗ ██║███████╗   ██║   ███████║██║  ███╗██████╔╝███████║ ║
    ║ ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║   ██║██╔══██╗██╔══██║ ║
    ║ ██║██║ ╚████║███████║   ██║   ██║  ██║╚██████╔╝██║  ██║██║  ██║ ║
    ║ ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝ ║
    ║        NETRUNNER EDITION - PHOTO EXTRACTION UTILITY v2.0        ║
    ╚══════════════════════════════════════════════════════════════╝
`

// Color functions for terminal output
var (
	Cyan    = colorize("\033[36m%s\033[0m")
	Yellow  = colorize("\033[33m%s\033[0m")
	Red     = colorize("\033[31m%s\033[0m")
	Green   = colorize("\033[32m%s\033[0m")
	Magenta = colorize("\033[35m%s\033[0m")
	Dim     = colorize("\033[2m%s\033[0m")
)

// colorize returns a function that wraps text with ANSI color codes
func colorize(colorString string) func(string) string {
	return func(text string) string {
		return fmt.Sprintf(colorString, text)
	}
}

// quietMode determines if UI output should be suppressed
var quietMode bool

// progressOnlyMode determines if only progress should be shown
var progressOnlyMode bool

// SetQuietMode enables or disables quiet mode
func SetQuietMode(quiet bool) {
	quietMode = quiet
}

// IsQuietMode returns true if quiet mode is enabled
func IsQuietMode() bool {
	// Check environment variable for quiet mode
	if os.Getenv("IGSCRAPER_QUIET") == "true" {
		return true
	}
	return quietMode
}

// SetProgressOnlyMode enables or disables progress-only mode
func SetProgressOnlyMode(progressOnly bool) {
	progressOnlyMode = progressOnly
}

// IsProgressOnlyMode returns true if progress-only mode is enabled
func IsProgressOnlyMode() bool {
	return progressOnlyMode
}

// PrintLogo prints the ASCII logo with color
func PrintLogo() {
	if IsQuietMode() || IsProgressOnlyMode() {
		return
	}
	fmt.Print(Cyan(ASCIILogo))
}

// PrintError prints an error message in red
func PrintError(msg string, args ...interface{}) {
	// Always print errors, even in quiet mode
	if len(args) > 0 {
		fmt.Println(Red(msg + ": " + fmt.Sprintf("%v", args[0])))
	} else {
		fmt.Println(Red(msg))
	}
}

// PrintSuccess prints a success message in green
func PrintSuccess(msg string) {
	if IsQuietMode() || IsProgressOnlyMode() {
		return
	}
	fmt.Println(Green(msg))
}

// PrintInfo prints an info message in cyan
func PrintInfo(label string, value string) {
	if IsQuietMode() || IsProgressOnlyMode() {
		return
	}
	fmt.Printf("%s: %s\n", Cyan(label), Yellow(value))
}

// PrintWarning prints a warning message in yellow
func PrintWarning(msg string, args ...interface{}) {
	if IsQuietMode() || IsProgressOnlyMode() {
		return
	}
	if len(args) > 0 {
		fmt.Println(Yellow(msg + ": " + fmt.Sprintf("%v", args[0])))
	} else {
		fmt.Println(Yellow(msg))
	}
}

// PrintHighlight prints a highlighted message in magenta
func PrintHighlight(msg string) {
	if IsQuietMode() || IsProgressOnlyMode() {
		return
	}
	fmt.Println(Magenta(msg))
}