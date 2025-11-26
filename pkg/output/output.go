package output

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Color codes
var (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorBlue    = "\033[34m"
	ColorCyan    = "\033[36m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorRed     = "\033[31m"
	ColorMagenta = "\033[35m"
	ColorSubLog  = "\033[38;5;244m"
)

var colorsDisabled = false

// DisableColors turns off colored output
func DisableColors() {
	colorsDisabled = true
	ColorReset = ""
	ColorBold = ""
	ColorBlue = ""
	ColorCyan = ""
	ColorGreen = ""
	ColorYellow = ""
	ColorRed = ""
	ColorMagenta = ""
	ColorSubLog = ""
}

func init() {
	// Auto-disable colors if NO_COLOR env var is set
	if os.Getenv("NO_COLOR") != "" {
		DisableColors()
	}
}

func isTerminal() bool {
	if colorsDisabled {
		return false
	}
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func isStdinTerminal() bool {
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// PrintCommand prints a command being executed
func PrintCommand(args ...string) {
	if !isTerminal() {
		fmt.Printf("│ %s\n", strings.Join(args, " "))
		return
	}
	fmt.Printf("%s│ %s%s\n", ColorSubLog, strings.Join(args, " "), ColorReset)
}

// PrintSublog prints a subordinate log message
func PrintSublog(message string) {
	if !isTerminal() {
		fmt.Printf("│ %s\n", message)
		return
	}
	fmt.Printf("%s│ %s%s\n", ColorSubLog, message, ColorReset)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	if !isTerminal() {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", message)
		return
	}
	fmt.Fprintf(os.Stderr, "%s⚠️  %s%s\n", ColorYellow, message, ColorReset)
}

// PrintError prints an error message
func PrintError(message string) {
	if !isTerminal() {
		fmt.Fprintf(os.Stderr, "❌ %s\n", message)
		return
	}
	fmt.Fprintf(os.Stderr, "%s❌ %s%s\n", ColorRed, message, ColorReset)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	if !isTerminal() {
		fmt.Printf("✅ %s\n", message)
		return
	}
	fmt.Printf("%s✅ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	if !isTerminal() {
		fmt.Printf("ℹ️  %s\n", message)
		return
	}
	fmt.Printf("%sℹ️  %s%s\n", ColorCyan, message, ColorReset)
}

// PrintBlocked prints a blocked action message with styling
func PrintBlocked(action, cluster, reason string) {
	if !isTerminal() {
		fmt.Fprintf(os.Stderr, "🚫 BLOCKED: Action '%s' is not allowed on cluster '%s'\n", action, cluster)
		fmt.Fprintf(os.Stderr, "│ Reason: %s\n", reason)
		return
	}
	fmt.Fprintf(os.Stderr, "%s🚫 BLOCKED:%s Action '%s' is not allowed on cluster '%s'%s\n",
		ColorRed, ColorBold, action, cluster, ColorReset)
	fmt.Fprintf(os.Stderr, "%s│ Reason: %s%s\n", ColorSubLog, reason, ColorReset)
}

// PrintConfirmationHeader prints the header for a confirmation prompt
func PrintConfirmationHeader(action, cluster, tier string) {
	if !isTerminal() {
		fmt.Fprintf(os.Stderr, "⚠️  CONFIRMATION REQUIRED\n")
		fmt.Fprintf(os.Stderr, "│ Action:  %s\n", action)
		fmt.Fprintf(os.Stderr, "│ Cluster: %s (%s)\n", cluster, tier)
		return
	}
	fmt.Fprintf(os.Stderr, "%s⚠️  CONFIRMATION REQUIRED%s\n", ColorYellow+ColorBold, ColorReset)
	fmt.Fprintf(os.Stderr, "%s│ Action:  %s%s\n", ColorSubLog, action, ColorReset)
	fmt.Fprintf(os.Stderr, "%s│ Cluster: %s%s (%s)%s\n", ColorSubLog, ColorCyan, cluster, tier, ColorReset)
}

// PromptConfirmation asks the user to confirm an action
// Returns true if confirmed, false otherwise
func PromptConfirmation(prompt string) bool {
	// If stdin is not a terminal (piped input), don't prompt
	if !isStdinTerminal() {
		PrintError("Cannot prompt for confirmation: stdin is not a terminal. Use --yes to skip confirmation.")
		return false
	}

	if isTerminal() {
		fmt.Fprintf(os.Stderr, "%s%s [y/N]: %s", ColorYellow, prompt, ColorReset)
	} else {
		fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// PrintContext prints the current context information
func PrintContext(context, tier string) {
	if !isTerminal() {
		fmt.Printf("│ Context: %s (%s)\n", context, tier)
		return
	}
	fmt.Printf("%s│ Context: %s%s%s (%s)%s\n",
		ColorSubLog, ColorCyan, context, ColorSubLog, tier, ColorReset)
}

