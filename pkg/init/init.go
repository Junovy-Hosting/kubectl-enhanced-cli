package init

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/config"
	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/kubectl"
	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/output"
)

// Options for config initialization
type Options struct {
	// Non-interactive mode options
	NonInteractive  bool
	Force           bool     // Overwrite existing config
	ProdPatterns    []string // Production cluster patterns
	StagingPatterns []string // Staging cluster patterns
	DevPatterns     []string // Development cluster patterns
	ProdActions     []string // Actions requiring confirmation on prod
	StagingActions  []string // Actions requiring confirmation on staging
	BlockedActions  []string // Globally blocked actions
	OutputPath      string   // Custom output path
}

// DefaultOptions returns default initialization options
func DefaultOptions() *Options {
	return &Options{
		NonInteractive:  false,
		Force:           false,
		ProdPatterns:    []string{"*-prod", "*-production", "prod-*", "production-*"},
		StagingPatterns: []string{"*-staging", "*-stg", "staging-*", "stg-*"},
		DevPatterns:     []string{"*-dev", "*-development", "dev-*", "development-*", "local*", "minikube", "docker-desktop", "kind-*"},
		ProdActions:     []string{"delete", "drain"},
		StagingActions:  []string{"delete"},
		BlockedActions:  []string{},
		OutputPath:      "",
	}
}

// Run executes the config initialization
func Run(opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = config.ConfigPath()
	}

	// Check if config already exists
	if _, err := os.Stat(outputPath); err == nil && !opts.Force {
		if opts.NonInteractive {
			return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
		}
		output.PrintWarning(fmt.Sprintf("Config file already exists at %s", outputPath))
		if !promptYesNo("Do you want to overwrite it?", false) {
			output.PrintSublog("Initialization cancelled")
			return nil
		}
	}

	var cfg *config.Config
	var err error

	if opts.NonInteractive {
		cfg = buildConfigFromOptions(opts)
	} else {
		cfg, err = runInteractiveInit(opts)
		if err != nil {
			return err
		}
	}

	// Write config to file
	if err := writeConfig(cfg, outputPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	output.PrintSuccess(fmt.Sprintf("Config file created at %s", outputPath))
	return nil
}

// runInteractiveInit runs the interactive configuration wizard
func runInteractiveInit(opts *Options) (*config.Config, error) {
	fmt.Println()
	output.PrintInfo("kubectl-enhanced-cli Configuration Wizard")
	fmt.Println()
	output.PrintSublog("This wizard will help you create a configuration file for RBAC controls.")
	output.PrintSublog("You can always edit the config file manually later.")
	fmt.Println()

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: make(map[string]config.ClusterRules),
		Tiers:    make(map[string]config.TierConfig),
	}

	// Step 1: Detect and categorize clusters
	contexts, err := kubectl.GetAllContexts()
	if err != nil {
		output.PrintWarning("Could not fetch kubectl contexts. Using default patterns.")
		contexts = []string{}
	}

	if len(contexts) > 0 {
		fmt.Println()
		output.PrintInfo("Detected kubectl contexts:")
		for i, ctx := range contexts {
			fmt.Printf("  %d. %s\n", i+1, ctx)
		}
		fmt.Println()

		// Ask if user wants to configure specific clusters
		if promptYesNo("Would you like to configure rules for specific clusters?", true) {
			cfg.Clusters = configureSpecificClusters(contexts)
		}
	}

	// Step 2: Configure tier patterns
	fmt.Println()
	output.PrintInfo("Configuring tier-based patterns")
	output.PrintSublog("Tiers let you apply rules based on cluster naming patterns.")
	fmt.Println()

	// Production tier
	if promptYesNo("Configure production tier patterns?", true) {
		cfg.Tiers["production"] = configureTier("production", opts.ProdPatterns, opts.ProdActions)
	}

	// Staging tier
	if promptYesNo("Configure staging tier patterns?", true) {
		cfg.Tiers["staging"] = configureTier("staging", opts.StagingPatterns, opts.StagingActions)
	}

	// Development tier
	if promptYesNo("Configure development tier patterns?", true) {
		cfg.Tiers["development"] = configureTier("development", opts.DevPatterns, []string{})
	}

	// Step 3: Configure global defaults
	fmt.Println()
	output.PrintInfo("Configuring global defaults")
	fmt.Println()

	if promptYesNo("Require confirmation for all destructive actions on unknown clusters?", false) {
		cfg.Defaults.RequireConfirmation = true
	}

	// Ask about blocked actions
	fmt.Println()
	output.PrintSublog("You can block certain actions entirely (they will always be denied).")
	if promptYesNo("Would you like to block any actions globally?", false) {
		cfg.Defaults.BlockedActions = selectActions("Select actions to block globally", []string{})
	}

	return cfg, nil
}

// configureSpecificClusters lets user configure rules for specific clusters
func configureSpecificClusters(contexts []string) map[string]config.ClusterRules {
	clusters := make(map[string]config.ClusterRules)
	
	fmt.Println()
	output.PrintSublog("For each cluster, you can set its tier and specific rules.")
	output.PrintSublog("Press Enter to skip a cluster.")
	fmt.Println()

	for _, ctx := range contexts {
		fmt.Printf("Configure cluster '%s'? ", ctx)
		tier := promptWithDefault("tier (production/staging/development/skip)", "skip")
		
		if tier == "skip" || tier == "" {
			continue
		}

		var actions []string
		switch tier {
		case "production":
			actions = []string{"delete", "drain"}
		case "staging":
			actions = []string{"delete"}
		default:
			actions = []string{}
		}

		if promptYesNo(fmt.Sprintf("  Customize actions for %s?", ctx), false) {
			actions = selectActions("  Select actions requiring confirmation", actions)
		}

		clusters[ctx] = config.ClusterRules{
			Tier:                tier,
			RequireConfirmation: actions,
			BlockedActions:      []string{},
		}
		fmt.Println()
	}

	return clusters
}

// configureTier helps configure a tier interactively
func configureTier(tierName string, defaultPatterns, defaultActions []string) config.TierConfig {
	fmt.Println()
	output.PrintSublog(fmt.Sprintf("Configuring %s tier:", tierName))

	// Patterns
	fmt.Printf("  Current patterns: %s\n", strings.Join(defaultPatterns, ", "))
	patterns := defaultPatterns
	if promptYesNo("  Modify patterns?", false) {
		patternsStr := promptWithDefault("  Enter patterns (comma-separated)", strings.Join(defaultPatterns, ","))
		patterns = parseCommaSeparated(patternsStr)
	}

	// Actions
	actions := defaultActions
	fmt.Printf("  Actions requiring confirmation: %s\n", formatActions(defaultActions))
	if promptYesNo("  Modify actions?", false) {
		actions = selectActions("  Select actions requiring confirmation", defaultActions)
	}

	return config.TierConfig{
		Patterns:            patterns,
		RequireConfirmation: actions,
		BlockedActions:      []string{},
	}
}

// selectActions presents a multi-select for actions
func selectActions(prompt string, defaults []string) []string {
	allActions := []string{"delete", "drain", "scale", "edit", "apply", "exec", "rollout"}
	
	fmt.Println(prompt + ":")
	for i, action := range allActions {
		marker := "[ ]"
		for _, d := range defaults {
			if d == action {
				marker = "[x]"
				break
			}
		}
		fmt.Printf("  %d. %s %s\n", i+1, marker, action)
	}

	input := promptWithDefault("Enter numbers (comma-separated) or 'none'", formatActionNumbers(defaults, allActions))
	
	if input == "none" || input == "" {
		return []string{}
	}

	// Parse selected numbers
	selected := []string{}
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			if num >= 1 && num <= len(allActions) {
				selected = append(selected, allActions[num-1])
			}
		}
	}

	return selected
}

// formatActionNumbers returns the default action numbers as a string
func formatActionNumbers(actions []string, allActions []string) string {
	if len(actions) == 0 {
		return "none"
	}
	nums := []string{}
	for _, action := range actions {
		for i, a := range allActions {
			if a == action {
				nums = append(nums, fmt.Sprintf("%d", i+1))
				break
			}
		}
	}
	return strings.Join(nums, ",")
}

// formatActions formats actions for display
func formatActions(actions []string) string {
	if len(actions) == 0 {
		return "(none)"
	}
	return strings.Join(actions, ", ")
}

// buildConfigFromOptions creates a config from non-interactive options
func buildConfigFromOptions(opts *Options) *config.Config {
	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      opts.BlockedActions,
		},
		Clusters: make(map[string]config.ClusterRules),
		Tiers:    make(map[string]config.TierConfig),
	}

	if len(opts.ProdPatterns) > 0 {
		cfg.Tiers["production"] = config.TierConfig{
			Patterns:            opts.ProdPatterns,
			RequireConfirmation: opts.ProdActions,
			BlockedActions:      []string{},
		}
	}

	if len(opts.StagingPatterns) > 0 {
		cfg.Tiers["staging"] = config.TierConfig{
			Patterns:            opts.StagingPatterns,
			RequireConfirmation: opts.StagingActions,
			BlockedActions:      []string{},
		}
	}

	if len(opts.DevPatterns) > 0 {
		cfg.Tiers["development"] = config.TierConfig{
			Patterns:            opts.DevPatterns,
			RequireConfirmation: []string{},
			BlockedActions:      []string{},
		}
	}

	return cfg
}

// writeConfig writes the config to a YAML file
func writeConfig(cfg *config.Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Generate YAML with comments
	content := generateConfigYAML(cfg)

	return os.WriteFile(path, []byte(content), 0644)
}

// generateConfigYAML generates a well-commented YAML config
func generateConfigYAML(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString("# kubectl-enhanced-cli Configuration\n")
	sb.WriteString("# Generated by 'kctl init'\n")
	sb.WriteString("#\n")
	sb.WriteString("# This file controls RBAC-like protections for kubectl commands.\n")
	sb.WriteString("# Edit this file to customize behavior per cluster or tier.\n\n")

	sb.WriteString("# Global defaults applied to all clusters unless overridden\n")
	sb.WriteString("defaults:\n")
	sb.WriteString(fmt.Sprintf("  require_confirmation: %v\n", cfg.Defaults.RequireConfirmation))
	if len(cfg.Defaults.BlockedActions) > 0 {
		sb.WriteString("  blocked_actions:\n")
		for _, action := range cfg.Defaults.BlockedActions {
			sb.WriteString(fmt.Sprintf("    - %s\n", action))
		}
	} else {
		sb.WriteString("  blocked_actions: []\n")
	}

	sb.WriteString("\n# Explicit cluster rules (highest priority)\n")
	sb.WriteString("# Use exact context names or glob patterns\n")
	sb.WriteString("clusters:\n")
	if len(cfg.Clusters) == 0 {
		sb.WriteString("  # Example:\n")
		sb.WriteString("  # my-prod-cluster:\n")
		sb.WriteString("  #   tier: production\n")
		sb.WriteString("  #   require_confirmation: [delete, drain]\n")
		sb.WriteString("  #   blocked_actions: []\n")
	} else {
		// Sort cluster names for consistent output
		clusterNames := make([]string, 0, len(cfg.Clusters))
		for name := range cfg.Clusters {
			clusterNames = append(clusterNames, name)
		}
		sort.Strings(clusterNames)

		for _, name := range clusterNames {
			rules := cfg.Clusters[name]
			sb.WriteString(fmt.Sprintf("  %s:\n", name))
			sb.WriteString(fmt.Sprintf("    tier: %s\n", rules.Tier))
			writeYAMLStringArray(&sb, "    require_confirmation", rules.RequireConfirmation)
			writeYAMLStringArray(&sb, "    blocked_actions", rules.BlockedActions)
		}
	}

	sb.WriteString("\n# Tier-based rules (fallback when no explicit cluster match)\n")
	sb.WriteString("# Clusters are matched against tier patterns\n")
	sb.WriteString("tiers:\n")

	// Write tiers in a consistent order
	tierOrder := []string{"production", "staging", "development"}
	for _, tierName := range tierOrder {
		if tier, ok := cfg.Tiers[tierName]; ok {
			sb.WriteString(fmt.Sprintf("  %s:\n", tierName))
			sb.WriteString("    patterns:\n")
			for _, pattern := range tier.Patterns {
				sb.WriteString(fmt.Sprintf("      - \"%s\"\n", pattern))
			}
			writeYAMLStringArray(&sb, "    require_confirmation", tier.RequireConfirmation)
			writeYAMLStringArray(&sb, "    blocked_actions", tier.BlockedActions)
			sb.WriteString("\n")
		}
	}

	// Write any additional tiers not in the standard order
	for tierName, tier := range cfg.Tiers {
		found := false
		for _, t := range tierOrder {
			if t == tierName {
				found = true
				break
			}
		}
		if !found {
			sb.WriteString(fmt.Sprintf("  %s:\n", tierName))
			sb.WriteString("    patterns:\n")
			for _, pattern := range tier.Patterns {
				sb.WriteString(fmt.Sprintf("      - \"%s\"\n", pattern))
			}
			writeYAMLStringArray(&sb, "    require_confirmation", tier.RequireConfirmation)
			writeYAMLStringArray(&sb, "    blocked_actions", tier.BlockedActions)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func writeYAMLStringArray(sb *strings.Builder, key string, values []string) {
	if len(values) == 0 {
		sb.WriteString(fmt.Sprintf("%s: []\n", key))
	} else {
		sb.WriteString(fmt.Sprintf("%s:\n", key))
		for _, v := range values {
			sb.WriteString(fmt.Sprintf("      - %s\n", v))
		}
	}
}

// Prompt helpers
func promptYesNo(question string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes
	}
	return response == "y" || response == "yes"
}

func promptWithDefault(prompt, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)
	
	if response == "" {
		return defaultVal
	}
	return response
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

