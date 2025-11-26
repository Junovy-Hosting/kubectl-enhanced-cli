package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetClusterRules_ExactMatch(t *testing.T) {
	cfg := &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: map[string]ClusterRules{
			"prod-cluster": {
				Tier:                "production",
				RequireConfirmation: []string{"delete", "drain"},
				BlockedActions:      []string{},
			},
			"staging-cluster": {
				Tier:                "staging",
				RequireConfirmation: []string{"delete"},
				BlockedActions:      []string{},
			},
		},
		Tiers: map[string]TierConfig{},
	}

	tests := []struct {
		name            string
		context         string
		expectedTier    string
		expectedConfirm []string
	}{
		{
			name:            "exact match production",
			context:         "prod-cluster",
			expectedTier:    "production",
			expectedConfirm: []string{"delete", "drain"},
		},
		{
			name:            "exact match staging",
			context:         "staging-cluster",
			expectedTier:    "staging",
			expectedConfirm: []string{"delete"},
		},
		{
			name:            "no match falls to default",
			context:         "unknown-cluster",
			expectedTier:    "default",
			expectedConfirm: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := cfg.GetClusterRules(tt.context)
			if rules.Tier != tt.expectedTier {
				t.Errorf("GetClusterRules(%q).Tier = %q, want %q", tt.context, rules.Tier, tt.expectedTier)
			}
			if len(rules.RequireConfirmation) != len(tt.expectedConfirm) {
				t.Errorf("GetClusterRules(%q).RequireConfirmation = %v, want %v",
					tt.context, rules.RequireConfirmation, tt.expectedConfirm)
			}
		})
	}
}

func TestGetClusterRules_PatternMatch(t *testing.T) {
	cfg := &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: map[string]ClusterRules{
			"prod-*": {
				Tier:                "production",
				RequireConfirmation: []string{"delete", "drain"},
				BlockedActions:      []string{},
			},
			"*-staging": {
				Tier:                "staging",
				RequireConfirmation: []string{"delete"},
				BlockedActions:      []string{},
			},
		},
		Tiers: map[string]TierConfig{},
	}

	tests := []struct {
		name         string
		context      string
		expectedTier string
	}{
		{
			name:         "prefix pattern match",
			context:      "prod-us-east-1",
			expectedTier: "production",
		},
		{
			name:         "suffix pattern match",
			context:      "us-west-staging",
			expectedTier: "staging",
		},
		{
			name:         "no pattern match",
			context:      "dev-cluster",
			expectedTier: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := cfg.GetClusterRules(tt.context)
			if rules.Tier != tt.expectedTier {
				t.Errorf("GetClusterRules(%q).Tier = %q, want %q", tt.context, rules.Tier, tt.expectedTier)
			}
		})
	}
}

func TestGetClusterRules_TierPatterns(t *testing.T) {
	cfg := &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: map[string]ClusterRules{},
		Tiers: map[string]TierConfig{
			"production": {
				Patterns:            []string{"*-prod", "*-production", "prod-*"},
				RequireConfirmation: []string{"delete", "drain"},
				BlockedActions:      []string{},
			},
			"staging": {
				Patterns:            []string{"*-staging", "*-stg", "staging-*"},
				RequireConfirmation: []string{"delete"},
				BlockedActions:      []string{},
			},
			"development": {
				Patterns:            []string{"*-dev", "dev-*", "minikube", "docker-desktop", "kind-*"},
				RequireConfirmation: []string{},
				BlockedActions:      []string{},
			},
		},
	}

	tests := []struct {
		name            string
		context         string
		expectedTier    string
		expectedConfirm []string
	}{
		{
			name:            "matches production suffix pattern",
			context:         "cluster-prod",
			expectedTier:    "production",
			expectedConfirm: []string{"delete", "drain"},
		},
		{
			name:            "matches production prefix pattern",
			context:         "prod-us-east-1",
			expectedTier:    "production",
			expectedConfirm: []string{"delete", "drain"},
		},
		{
			name:            "matches staging pattern",
			context:         "app-staging",
			expectedTier:    "staging",
			expectedConfirm: []string{"delete"},
		},
		{
			name:            "matches development minikube",
			context:         "minikube",
			expectedTier:    "development",
			expectedConfirm: []string{},
		},
		{
			name:            "matches development docker-desktop",
			context:         "docker-desktop",
			expectedTier:    "development",
			expectedConfirm: []string{},
		},
		{
			name:            "matches development kind pattern",
			context:         "kind-test-cluster",
			expectedTier:    "development",
			expectedConfirm: []string{},
		},
		{
			name:            "no tier match falls to default",
			context:         "unknown-cluster-name",
			expectedTier:    "default",
			expectedConfirm: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := cfg.GetClusterRules(tt.context)
			if rules.Tier != tt.expectedTier {
				t.Errorf("GetClusterRules(%q).Tier = %q, want %q", tt.context, rules.Tier, tt.expectedTier)
			}
			if len(rules.RequireConfirmation) != len(tt.expectedConfirm) {
				t.Errorf("GetClusterRules(%q).RequireConfirmation = %v, want %v",
					tt.context, rules.RequireConfirmation, tt.expectedConfirm)
			}
		})
	}
}

func TestGetClusterRules_Priority(t *testing.T) {
	// Test that explicit cluster rules take priority over tier patterns
	cfg := &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: false,
			BlockedActions:      []string{},
		},
		Clusters: map[string]ClusterRules{
			// Exact match should take priority
			"special-prod": {
				Tier:                "special",
				RequireConfirmation: []string{"scale"},
				BlockedActions:      []string{"delete"},
			},
		},
		Tiers: map[string]TierConfig{
			"production": {
				Patterns:            []string{"*-prod"},
				RequireConfirmation: []string{"delete", "drain"},
				BlockedActions:      []string{},
			},
		},
	}

	// "special-prod" should match the explicit cluster rule, not the tier pattern
	rules := cfg.GetClusterRules("special-prod")
	if rules.Tier != "special" {
		t.Errorf("Expected tier 'special' for explicit cluster match, got %q", rules.Tier)
	}
	if len(rules.BlockedActions) != 1 || rules.BlockedActions[0] != "delete" {
		t.Errorf("Expected blocked_actions [delete] for explicit match, got %v", rules.BlockedActions)
	}

	// "other-prod" should match the tier pattern
	rules2 := cfg.GetClusterRules("other-prod")
	if rules2.Tier != "production" {
		t.Errorf("Expected tier 'production' for tier pattern match, got %q", rules2.Tier)
	}
}

func TestGetClusterRules_DefaultRequireConfirmation(t *testing.T) {
	cfg := &Config{
		Defaults: DefaultsConfig{
			RequireConfirmation: true, // Global confirmation enabled
			BlockedActions:      []string{},
		},
		Clusters: map[string]ClusterRules{},
		Tiers:    map[string]TierConfig{},
	}

	rules := cfg.GetClusterRules("unknown-cluster")

	if rules.Tier != "default" {
		t.Errorf("Expected tier 'default', got %q", rules.Tier)
	}

	// When global RequireConfirmation is true, it should return delete and drain
	expectedActions := []string{"delete", "drain"}
	if len(rules.RequireConfirmation) != len(expectedActions) {
		t.Errorf("Expected RequireConfirmation %v, got %v", expectedActions, rules.RequireConfirmation)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Check defaults
	if cfg.Defaults.RequireConfirmation != false {
		t.Error("Default RequireConfirmation should be false")
	}

	// Check that production tier exists
	if _, ok := cfg.Tiers["production"]; !ok {
		t.Error("Default config should have production tier")
	}

	// Check that staging tier exists
	if _, ok := cfg.Tiers["staging"]; !ok {
		t.Error("Default config should have staging tier")
	}

	// Check that development tier exists
	if _, ok := cfg.Tiers["development"]; !ok {
		t.Error("Default config should have development tier")
	}

	// Verify production tier has expected patterns
	prodTier := cfg.Tiers["production"]
	if len(prodTier.Patterns) == 0 {
		t.Error("Production tier should have patterns")
	}

	// Verify production tier has delete and drain in RequireConfirmation
	hasDelete := false
	hasDrain := false
	for _, action := range prodTier.RequireConfirmation {
		if action == "delete" {
			hasDelete = true
		}
		if action == "drain" {
			hasDrain = true
		}
	}
	if !hasDelete || !hasDrain {
		t.Errorf("Production tier should require confirmation for delete and drain, got %v",
			prodTier.RequireConfirmation)
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		str      string
		expected bool
	}{
		// Simple patterns
		{"*-prod", "cluster-prod", true},
		{"*-prod", "my-app-prod", true},
		{"*-prod", "prod-cluster", false},
		{"prod-*", "prod-cluster", true},
		{"prod-*", "cluster-prod", false},

		// Multiple wildcards
		{"*-*-prod", "us-east-prod", true},
		{"*-*-prod", "prod", false},

		// Exact match (no wildcards)
		{"minikube", "minikube", true},
		{"minikube", "minikube-2", false},
		{"docker-desktop", "docker-desktop", true},

		// Complex patterns
		{"kind-*", "kind-test", true},
		{"kind-*", "kind-", true},
		{"kind-*", "kind", false},

		// Case sensitivity
		{"*-PROD", "cluster-prod", false}, // glob is case-sensitive
		{"*-prod", "cluster-PROD", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.str, func(t *testing.T) {
			result := matchGlob(tt.pattern, tt.str)
			if result != tt.expected {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.str, result, tt.expected)
			}
		})
	}
}

func TestLoadFromPath(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
defaults:
  require_confirmation: true
  blocked_actions:
    - exec

clusters:
  my-prod-cluster:
    tier: production
    require_confirmation:
      - delete
      - drain
    blocked_actions: []

tiers:
  production:
    patterns:
      - "*-prod"
    require_confirmation:
      - delete
    blocked_actions: []
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	// Verify loaded values
	if !cfg.Defaults.RequireConfirmation {
		t.Error("Expected Defaults.RequireConfirmation to be true")
	}

	if len(cfg.Defaults.BlockedActions) != 1 || cfg.Defaults.BlockedActions[0] != "exec" {
		t.Errorf("Expected Defaults.BlockedActions = [exec], got %v", cfg.Defaults.BlockedActions)
	}

	if cluster, ok := cfg.Clusters["my-prod-cluster"]; !ok {
		t.Error("Expected cluster 'my-prod-cluster' to be loaded")
	} else if cluster.Tier != "production" {
		t.Errorf("Expected cluster tier 'production', got %q", cluster.Tier)
	}

	if tier, ok := cfg.Tiers["production"]; !ok {
		t.Error("Expected tier 'production' to be loaded")
	} else if len(tier.Patterns) != 1 || tier.Patterns[0] != "*-prod" {
		t.Errorf("Expected tier patterns [*-prod], got %v", tier.Patterns)
	}
}

func TestLoadFromPath_FileNotFound(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadFromPath_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = LoadFromPath(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestConfigPath(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	path := ConfigPath()
	expected := "/custom/config/kubectl-enhanced/config.yaml"
	if path != expected {
		t.Errorf("ConfigPath() with XDG_CONFIG_HOME = %q, want %q", path, expected)
	}

	// Test without XDG_CONFIG_HOME
	os.Unsetenv("XDG_CONFIG_HOME")
	path = ConfigPath()
	home, _ := os.UserHomeDir()
	expected = filepath.Join(home, ".config", "kubectl-enhanced", "config.yaml")
	if path != expected {
		t.Errorf("ConfigPath() without XDG_CONFIG_HOME = %q, want %q", path, expected)
	}
}

// Test real-world cluster naming conventions
func TestGetClusterRules_RealWorldContextNames(t *testing.T) {
	cfg := Default()

	tests := []struct {
		name         string
		context      string
		expectedTier string
	}{
		// Common production naming patterns
		// Note: glob patterns like *-prod will match anything ending in -prod
		{"AWS EKS prod", "arn:aws:eks:us-east-1:123456:cluster/my-app-prod", "production"}, // ARN with -prod suffix matches
		{"GKE production", "gke_my-project_us-central1_production", "default"},             // GKE format - no matching pattern
		{"simple prod suffix", "my-cluster-prod", "production"},
		{"production suffix", "app-production", "production"},
		{"prod prefix", "prod-us-east-1", "production"},

		// Common staging patterns
		{"staging suffix", "app-staging", "staging"},
		{"stg suffix", "my-app-stg", "staging"},
		{"staging prefix", "staging-cluster", "staging"},

		// Common development patterns
		{"dev suffix", "my-app-dev", "development"},
		{"development suffix", "app-development", "development"},
		{"local dev", "local-cluster", "development"},
		{"minikube", "minikube", "development"},
		{"docker desktop", "docker-desktop", "development"},
		{"kind cluster", "kind-my-test", "development"},

		// Admin context patterns (common in multi-user clusters)
		// Note: kubernetes-admin@dds-prod ends in -prod, so it matches *-prod
		{"kubernetes-admin prefix", "kubernetes-admin@dds-prod", "production"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := cfg.GetClusterRules(tt.context)
			if rules.Tier != tt.expectedTier {
				t.Errorf("GetClusterRules(%q).Tier = %q, want %q", tt.context, rules.Tier, tt.expectedTier)
			}
		})
	}
}

