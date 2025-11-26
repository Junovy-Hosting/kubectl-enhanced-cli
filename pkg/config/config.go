package config

import (
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"
)

// Config represents the kubectl-enhanced-cli configuration
type Config struct {
	Defaults DefaultsConfig          `yaml:"defaults"`
	Clusters map[string]ClusterRules `yaml:"clusters"`
	Tiers    map[string]TierConfig   `yaml:"tiers"`
}

// DefaultsConfig represents global default settings
type DefaultsConfig struct {
	RequireConfirmation bool     `yaml:"require_confirmation"`
	BlockedActions      []string `yaml:"blocked_actions"`
}

// ClusterRules represents rules for a specific cluster
type ClusterRules struct {
	Tier                string   `yaml:"tier"`
	RequireConfirmation []string `yaml:"require_confirmation"`
	BlockedActions      []string `yaml:"blocked_actions"`
}

// TierConfig represents rules for a tier of clusters
type TierConfig struct {
	Patterns            []string `yaml:"patterns"`
	RequireConfirmation []string `yaml:"require_confirmation"`
	BlockedActions      []string `yaml:"blocked_actions"`
}

// ResolvedRules represents the final resolved rules for a cluster
type ResolvedRules struct {
	Tier                string
	RequireConfirmation []string
	BlockedActions      []string
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "kubectl-enhanced", "config.yaml")
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "kubectl-enhanced", "config.yaml")
}

// Load loads the configuration from the default config path
func Load() (*Config, error) {
	return LoadFromPath(ConfigPath())
}

// LoadFromPath loads configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: make(map[string]ClusterRules),
		Tiers: map[string]TierConfig{
			"production": {
				Patterns:            []string{"*-prod", "*-production", "prod-*", "production-*"},
				RequireConfirmation: []string{"delete", "drain"},
				BlockedActions:      []string{},
			},
			"staging": {
				Patterns:            []string{"*-staging", "*-stg", "staging-*", "stg-*"},
				RequireConfirmation: []string{"delete"},
				BlockedActions:      []string{},
			},
			"development": {
				Patterns:            []string{"*-dev", "*-development", "dev-*", "development-*", "local*", "minikube", "docker-desktop", "kind-*"},
				RequireConfirmation: []string{},
				BlockedActions:      []string{},
			},
		},
	}
}

// GetClusterRules returns the resolved rules for a given cluster context
func (c *Config) GetClusterRules(context string) ResolvedRules {
	// 1. Check for exact cluster match
	if rules, ok := c.Clusters[context]; ok {
		return ResolvedRules{
			Tier:                rules.Tier,
			RequireConfirmation: rules.RequireConfirmation,
			BlockedActions:      rules.BlockedActions,
		}
	}

	// 2. Check for glob pattern match in clusters
	for pattern, rules := range c.Clusters {
		if matchGlob(pattern, context) {
			return ResolvedRules{
				Tier:                rules.Tier,
				RequireConfirmation: rules.RequireConfirmation,
				BlockedActions:      rules.BlockedActions,
			}
		}
	}

	// 3. Check tier patterns
	for tierName, tier := range c.Tiers {
		for _, pattern := range tier.Patterns {
			if matchGlob(pattern, context) {
				return ResolvedRules{
					Tier:                tierName,
					RequireConfirmation: tier.RequireConfirmation,
					BlockedActions:      tier.BlockedActions,
				}
			}
		}
	}

	// 4. Return defaults
	confirmActions := []string{}
	if c.Defaults.RequireConfirmation {
		// If global require_confirmation is true, default to common destructive actions
		confirmActions = []string{"delete", "drain"}
	}

	return ResolvedRules{
		Tier:                "default",
		RequireConfirmation: confirmActions,
		BlockedActions:      c.Defaults.BlockedActions,
	}
}

// matchGlob checks if a string matches a glob pattern
func matchGlob(pattern, str string) bool {
	// Try to compile and match with gobwas/glob for advanced patterns
	g, err := glob.Compile(pattern)
	if err != nil {
		// Fall back to filepath.Match for simple patterns
		matched, _ := filepath.Match(pattern, str)
		return matched
	}
	return g.Match(str)
}

