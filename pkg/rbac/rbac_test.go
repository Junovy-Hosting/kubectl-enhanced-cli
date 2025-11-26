package rbac

import (
	"testing"

	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/config"
)

func TestDetectAction(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		// Basic commands
		{
			name:     "simple delete",
			args:     []string{"delete", "pod", "foo"},
			expected: ActionDelete,
		},
		{
			name:     "simple get",
			args:     []string{"get", "pods"},
			expected: "get",
		},
		{
			name:     "simple drain",
			args:     []string{"drain", "node-1"},
			expected: ActionDrain,
		},
		{
			name:     "simple scale",
			args:     []string{"scale", "deployment/app", "--replicas=3"},
			expected: ActionScale,
		},

		// Namespace flag before action
		{
			name:     "namespace short flag before delete",
			args:     []string{"-n", "default", "delete", "configmap", "foo"},
			expected: ActionDelete,
		},
		{
			name:     "namespace long flag before delete",
			args:     []string{"--namespace", "kube-system", "delete", "pod", "bar"},
			expected: ActionDelete,
		},
		{
			name:     "namespace equals syntax before delete",
			args:     []string{"--namespace=production", "delete", "deployment", "app"},
			expected: ActionDelete,
		},

		// Multiple flags before action
		{
			name:     "multiple flags before delete",
			args:     []string{"-n", "default", "-l", "app=test", "delete", "pods"},
			expected: ActionDelete,
		},
		{
			name:     "output flag before get",
			args:     []string{"-o", "wide", "get", "pods"},
			expected: "get",
		},
		{
			name:     "context and namespace before delete",
			args:     []string{"--context", "prod-cluster", "-n", "default", "delete", "svc", "my-svc"},
			expected: ActionDelete,
		},

		// Flags after action (should still detect action)
		{
			name:     "delete with flags after",
			args:     []string{"delete", "pod", "foo", "--grace-period=0", "--force"},
			expected: ActionDelete,
		},
		{
			name:     "drain with flags",
			args:     []string{"drain", "node-1", "--ignore-daemonsets", "--delete-emptydir-data"},
			expected: ActionDrain,
		},

		// Edge cases
		{
			name:     "empty args",
			args:     []string{},
			expected: ActionUnknown,
		},
		{
			name:     "only flags",
			args:     []string{"-n", "default"},
			expected: ActionUnknown,
		},
		{
			name:     "cordon action",
			args:     []string{"cordon", "node-1"},
			expected: ActionCordon,
		},
		{
			name:     "uncordon action",
			args:     []string{"uncordon", "node-1"},
			expected: ActionCordon,
		},
		{
			name:     "exec action",
			args:     []string{"exec", "-it", "pod-name", "--", "bash"},
			expected: ActionExec,
		},
		{
			name:     "apply action",
			args:     []string{"apply", "-f", "deployment.yaml"},
			expected: ActionApply,
		},
		{
			name:     "create action",
			args:     []string{"create", "namespace", "test"},
			expected: ActionCreate,
		},
		{
			name:     "rollout action",
			args:     []string{"rollout", "restart", "deployment/app"},
			expected: ActionRollout,
		},
		{
			name:     "patch action",
			args:     []string{"patch", "deployment", "app", "-p", `{"spec":{"replicas":3}}`},
			expected: ActionPatch,
		},
		{
			name:     "edit action",
			args:     []string{"edit", "deployment", "app"},
			expected: ActionEdit,
		},

		// Complex real-world scenarios
		{
			name:     "delete with selector",
			args:     []string{"-n", "production", "--selector", "app=legacy", "delete", "pods"},
			expected: ActionDelete,
		},
		{
			name:     "kubeconfig flag",
			args:     []string{"--kubeconfig", "/path/to/config", "delete", "pod", "test"},
			expected: ActionDelete,
		},
		{
			name:     "mixed equals and space flags",
			args:     []string{"--namespace=prod", "--context", "my-cluster", "delete", "cm", "config"},
			expected: ActionDelete,
		},

		// Safe operations (not in destructive list)
		{
			name:     "describe",
			args:     []string{"describe", "pod", "foo"},
			expected: "describe",
		},
		{
			name:     "logs",
			args:     []string{"logs", "-f", "pod-name"},
			expected: "logs",
		},
		{
			name:     "port-forward",
			args:     []string{"port-forward", "svc/my-svc", "8080:80"},
			expected: "port-forward",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAction(tt.args)
			if result != tt.expected {
				t.Errorf("DetectAction(%v) = %q, want %q", tt.args, result, tt.expected)
			}
		})
	}
}

func TestIsBlocked(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		rules    config.ResolvedRules
		expected bool
	}{
		{
			name:   "action is blocked",
			action: ActionDelete,
			rules: config.ResolvedRules{
				BlockedActions: []string{"delete"},
			},
			expected: true,
		},
		{
			name:   "action is not blocked",
			action: ActionDelete,
			rules: config.ResolvedRules{
				BlockedActions: []string{"drain"},
			},
			expected: false,
		},
		{
			name:   "empty blocked list",
			action: ActionDelete,
			rules: config.ResolvedRules{
				BlockedActions: []string{},
			},
			expected: false,
		},
		{
			name:   "multiple blocked actions",
			action: ActionDrain,
			rules: config.ResolvedRules{
				BlockedActions: []string{"delete", "drain", "scale"},
			},
			expected: true,
		},
		{
			name:   "case insensitive matching",
			action: "DELETE",
			rules: config.ResolvedRules{
				BlockedActions: []string{"delete"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlocked(tt.action, tt.rules)
			if result != tt.expected {
				t.Errorf("IsBlocked(%q, %+v) = %v, want %v", tt.action, tt.rules, result, tt.expected)
			}
		})
	}
}

func TestRequiresConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		rules    config.ResolvedRules
		expected bool
	}{
		{
			name:   "action requires confirmation",
			action: ActionDelete,
			rules: config.ResolvedRules{
				RequireConfirmation: []string{"delete"},
			},
			expected: true,
		},
		{
			name:   "action does not require confirmation",
			action: "get",
			rules: config.ResolvedRules{
				RequireConfirmation: []string{"delete", "drain"},
			},
			expected: false,
		},
		{
			name:   "empty confirmation list",
			action: ActionDelete,
			rules: config.ResolvedRules{
				RequireConfirmation: []string{},
			},
			expected: false,
		},
		{
			name:   "drain rule covers cordon",
			action: ActionCordon,
			rules: config.ResolvedRules{
				RequireConfirmation: []string{"drain"},
			},
			expected: true,
		},
		{
			name:   "edit rule covers patch",
			action: ActionPatch,
			rules: config.ResolvedRules{
				RequireConfirmation: []string{"edit"},
			},
			expected: true,
		},
		{
			name:   "apply rule covers create",
			action: ActionCreate,
			rules: config.ResolvedRules{
				RequireConfirmation: []string{"apply"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RequiresConfirmation(tt.action, tt.rules)
			if result != tt.expected {
				t.Errorf("RequiresConfirmation(%q, %+v) = %v, want %v", tt.action, tt.rules, result, tt.expected)
			}
		})
	}
}

func TestMatchAction(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		action   string
		expected bool
	}{
		// Exact matches
		{"exact delete", "delete", "delete", true},
		{"exact drain", "drain", "drain", true},
		{"no match", "delete", "get", false},

		// Case insensitive
		{"case insensitive rule", "DELETE", "delete", true},
		{"case insensitive action", "delete", "DELETE", true},

		// Aliases
		{"drain covers drain", "drain", "drain", true},
		{"drain covers cordon", "drain", "cordon", true},
		{"edit covers edit", "edit", "edit", true},
		{"edit covers patch", "edit", "patch", true},
		{"apply covers apply", "apply", "apply", true},
		{"apply covers create", "apply", "create", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchAction(tt.rule, tt.action)
			if result != tt.expected {
				t.Errorf("matchAction(%q, %q) = %v, want %v", tt.rule, tt.action, result, tt.expected)
			}
		})
	}
}

func TestDescribeAction(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{ActionDelete, "Delete resources"},
		{ActionDrain, "Drain node (evict all pods)"},
		{ActionCordon, "Cordon/uncordon node"},
		{ActionScale, "Scale deployment replicas"},
		{ActionEdit, "Edit resource configuration"},
		{ActionPatch, "Patch resource"},
		{ActionApply, "Apply configuration"},
		{ActionCreate, "Create resource"},
		{ActionExec, "Execute command in pod"},
		{ActionRollout, "Manage rollout"},
		{"unknown-action", "unknown-action"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := DescribeAction(tt.action)
			if result != tt.expected {
				t.Errorf("DescribeAction(%q) = %q, want %q", tt.action, result, tt.expected)
			}
		})
	}
}

func TestGetActionSeverity(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{ActionDelete, "high"},
		{ActionDrain, "high"},
		{ActionScale, "medium"},
		{ActionCordon, "medium"},
		{ActionEdit, "medium"},
		{ActionPatch, "medium"},
		{ActionRollout, "medium"},
		{ActionApply, "low"},
		{ActionCreate, "low"},
		{"get", "none"},
		{"describe", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := GetActionSeverity(tt.action)
			if result != tt.expected {
				t.Errorf("GetActionSeverity(%q) = %q, want %q", tt.action, result, tt.expected)
			}
		})
	}
}

