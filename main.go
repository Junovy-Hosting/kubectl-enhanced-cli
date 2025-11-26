package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/config"
	initpkg "github.com/bobbydrake/kubectl-enhanced-cli/pkg/init"
	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/kubectl"
	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/output"
	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/rbac"
)

// Version information (set at build time with -ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	args := os.Args[1:]

	// Detect if running as kubectl plugin (kubectl enhanced ...)
	// In plugin mode, kubectl strips "enhanced" from args
	execName := filepath.Base(os.Args[0])
	isPlugin := execName == "kubectl-enhanced"

	// Handle version flag
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Printf("kubectl-enhanced-cli %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Handle help flag
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printUsage(isPlugin)
		os.Exit(0)
	}

	// Handle config-path flag
	if len(args) > 0 && args[0] == "--config-path" {
		fmt.Println(config.ConfigPath())
		os.Exit(0)
	}

	// Handle init command
	if len(args) > 0 && args[0] == "init" {
		handleInit(args[1:])
		return
	}

	// Check if kubectl is available
	if !kubectl.CheckKubectlAvailable() {
		output.PrintError("kubectl not found in PATH")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		if !os.IsNotExist(err) {
			output.PrintWarning(fmt.Sprintf("Could not load config: %v (using defaults)", err))
		}
		cfg = config.Default()
	}

	// Get current kubectl context
	context, err := kubectl.GetCurrentContext()
	if err != nil {
		output.PrintError(fmt.Sprintf("Failed to get current context: %v", err))
		output.PrintSublog("Make sure kubectl is configured with a valid context")
		os.Exit(1)
	}

	// Extract --yes/-y flags before processing
	hasYesFlag := false
	filteredArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--yes" || arg == "-y" {
			hasYesFlag = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	// Detect the action from kubectl args
	action := rbac.DetectAction(args)

	// Get rules for the current cluster
	rules := cfg.GetClusterRules(context)

	// Check if action is blocked
	if rbac.IsBlocked(action, rules) {
		output.PrintBlocked(action, context, fmt.Sprintf("Action '%s' is configured as blocked for tier '%s'", action, rules.Tier))
		os.Exit(1)
	}

	// Check if confirmation is required
	if rbac.RequiresConfirmation(action, rules) && !hasYesFlag {
		namespace := kubectl.GetNamespace(args)
		
		output.PrintConfirmationHeader(
			rbac.DescribeAction(action),
			context,
			rules.Tier,
		)
		output.PrintSublog(fmt.Sprintf("Namespace: %s", namespace))
		output.PrintSublog(fmt.Sprintf("Command: kubectl %s", formatArgs(args)))
		fmt.Fprintln(os.Stderr) // Empty line for spacing

		confirmed := output.PromptConfirmation("Do you want to proceed?")
		if !confirmed {
			output.PrintSublog("Operation cancelled by user")
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr) // Empty line before output
	}

	// Execute kubectl command
	exitCode := kubectl.Execute(args)
	os.Exit(exitCode)
}

func printUsage(isPlugin bool) {
	var cmdExample string
	if isPlugin {
		cmdExample = "kubectl enhanced"
	} else {
		cmdExample = "kctl"
	}

	fmt.Printf(`kubectl-enhanced-cli - kubectl wrapper with RBAC controls

Usage:
  %s <kubectl-args>
  %s init [flags]            # Create/configure config file

Description:
  A kubectl wrapper that adds safety controls for production clusters.
  It can block or require confirmation for destructive operations
  based on per-cluster configuration.

Invocation Modes:
  kctl <args>              # Wrapper mode (direct invocation)
  kubectl enhanced <args>  # Plugin mode (via kubectl)

Commands:
  init          Create a configuration file (interactive or scripted)
                Run '%s init --help' for more information

Flags:
  --yes, -y       Skip confirmation prompts
  --version, -v   Print version information
  --help, -h      Print this help message
  --config-path   Print the config file path

Configuration:
  Config file: %s

  Create a config file interactively:
    %s init

  Or non-interactively with defaults:
    %s init --non-interactive

Examples:
  %s get pods                    # Safe operation, passes through
  %s delete pod my-pod           # May require confirmation on prod clusters
  %s delete pod my-pod --yes     # Skip confirmation
  %s drain node-1                # Requires confirmation on prod clusters

Protected Actions (configurable):
  - delete    Delete resources
  - drain     Drain/cordon nodes

For more information, see the README.md
`, cmdExample, cmdExample, cmdExample, config.ConfigPath(), cmdExample, cmdExample, cmdExample, cmdExample, cmdExample, cmdExample)
}

func formatArgs(args []string) string {
	return strings.Join(args, " ")
}

// handleInit processes the init command for config creation
func handleInit(args []string) {
	opts := initpkg.DefaultOptions()

	// Parse init-specific flags
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--help", "-h":
			printInitUsage()
			os.Exit(0)
		case "--non-interactive", "-n":
			opts.NonInteractive = true
		case "--force", "-f":
			opts.Force = true
		case "--output", "-o":
			if i+1 < len(args) {
				opts.OutputPath = args[i+1]
				i++
			}
		case "--prod-patterns":
			if i+1 < len(args) {
				opts.ProdPatterns = parseCommaSeparated(args[i+1])
				i++
			}
		case "--staging-patterns":
			if i+1 < len(args) {
				opts.StagingPatterns = parseCommaSeparated(args[i+1])
				i++
			}
		case "--dev-patterns":
			if i+1 < len(args) {
				opts.DevPatterns = parseCommaSeparated(args[i+1])
				i++
			}
		case "--prod-actions":
			if i+1 < len(args) {
				opts.ProdActions = parseCommaSeparated(args[i+1])
				i++
			}
		case "--staging-actions":
			if i+1 < len(args) {
				opts.StagingActions = parseCommaSeparated(args[i+1])
				i++
			}
		case "--blocked-actions":
			if i+1 < len(args) {
				opts.BlockedActions = parseCommaSeparated(args[i+1])
				i++
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				output.PrintError(fmt.Sprintf("Unknown flag: %s", args[i]))
				printInitUsage()
				os.Exit(1)
			}
		}
		i++
	}

	if err := initpkg.Run(opts); err != nil {
		output.PrintError(err.Error())
		os.Exit(1)
	}
}

func printInitUsage() {
	fmt.Printf(`kctl init - Create a configuration file

Usage:
  kctl init [flags]

Description:
  Creates a configuration file for kubectl-enhanced-cli. By default, runs in
  interactive mode, guiding you through the setup process. Use --non-interactive
  for scripted/automated setup.

Flags:
  -h, --help              Show this help message
  -n, --non-interactive   Run in non-interactive mode (uses defaults or provided flags)
  -f, --force             Overwrite existing config file without prompting
  -o, --output PATH       Write config to a custom path (default: %s)

Non-interactive mode options:
  --prod-patterns PATTERNS      Comma-separated production cluster patterns
                                (default: *-prod,*-production,prod-*,production-*)
  --staging-patterns PATTERNS   Comma-separated staging cluster patterns
                                (default: *-staging,*-stg,staging-*,stg-*)
  --dev-patterns PATTERNS       Comma-separated development cluster patterns
                                (default: *-dev,*-development,dev-*,local*,minikube,docker-desktop,kind-*)
  --prod-actions ACTIONS        Actions requiring confirmation on production
                                (default: delete,drain)
  --staging-actions ACTIONS     Actions requiring confirmation on staging
                                (default: delete)
  --blocked-actions ACTIONS     Globally blocked actions (default: none)

Examples:
  # Interactive mode (recommended for first-time setup)
  kctl init

  # Non-interactive with defaults
  kctl init --non-interactive

  # Non-interactive with custom patterns
  kctl init -n --prod-patterns "prod-*,*-prd" --prod-actions "delete,drain,scale"

  # Overwrite existing config
  kctl init --force

  # Write to custom location
  kctl init -o /path/to/config.yaml
`, config.ConfigPath())
}

func parseCommaSeparated(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
