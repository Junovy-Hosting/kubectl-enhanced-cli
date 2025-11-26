package rbac

import (
	"strings"

	"github.com/bobbydrake/kubectl-enhanced-cli/pkg/config"
)

// Action types that can be detected from kubectl commands
const (
	ActionDelete  = "delete"
	ActionDrain   = "drain"
	ActionCordon  = "cordon"
	ActionScale   = "scale"
	ActionEdit    = "edit"
	ActionPatch   = "patch"
	ActionApply   = "apply"
	ActionCreate  = "create"
	ActionExec    = "exec"
	ActionRollout = "rollout"
	ActionUnknown = "unknown"
)

// DestructiveActions maps kubectl commands to their action type
var DestructiveActions = map[string]string{
	"delete":   ActionDelete,
	"drain":    ActionDrain,
	"cordon":   ActionCordon,
	"uncordon": ActionCordon,
	"scale":    ActionScale,
	"edit":     ActionEdit,
	"patch":    ActionPatch,
	"apply":    ActionApply,
	"create":   ActionCreate,
	"exec":     ActionExec,
	"rollout":  ActionRollout,
}

// Flags that take a value argument (the next arg is the value, not a command)
var flagsWithValues = map[string]bool{
	"-n":              true,
	"--namespace":     true,
	"-l":              true,
	"--selector":      true,
	"-o":              true,
	"--output":        true,
	"-f":              true,
	"--filename":      true,
	"--context":       true,
	"--kubeconfig":    true,
	"--cluster":       true,
	"--user":          true,
	"-c":              true,
	"--container":     true,
	"--field-selector": true,
	"--sort-by":       true,
	"--template":      true,
	"-p":              true,
	"--patch":         true,
	"--type":          true,
	"--replicas":      true,
	"--timeout":       true,
	"--grace-period":  true,
}

// DetectAction analyzes kubectl arguments and returns the action type
func DetectAction(args []string) string {
	if len(args) == 0 {
		return ActionUnknown
	}

	// Skip flags and their values to find the actual command
	skipNext := false
	for _, arg := range args {
		// Skip the value of a flag that takes an argument
		if skipNext {
			skipNext = false
			continue
		}

		// Handle --flag=value format
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			continue
		}

		// Check if this is a flag
		if strings.HasPrefix(arg, "-") {
			// Check if this flag takes a value
			if flagsWithValues[arg] {
				skipNext = true
			}
			continue
		}

		// This is a non-flag argument - check if it's a known action
		if action, ok := DestructiveActions[arg]; ok {
			return action
		}

		// For commands like "kubectl get", the first non-flag is the command
		// If it's not a known destructive action, it's likely safe
		return arg
	}

	return ActionUnknown
}

// IsBlocked checks if an action is blocked by the rules
func IsBlocked(action string, rules config.ResolvedRules) bool {
	for _, blocked := range rules.BlockedActions {
		if matchAction(blocked, action) {
			return true
		}
	}
	return false
}

// RequiresConfirmation checks if an action requires confirmation
func RequiresConfirmation(action string, rules config.ResolvedRules) bool {
	for _, confirm := range rules.RequireConfirmation {
		if matchAction(confirm, action) {
			return true
		}
	}
	return false
}

// matchAction checks if an action matches a rule
// Supports exact match and some aliases
func matchAction(rule, action string) bool {
	rule = strings.ToLower(rule)
	action = strings.ToLower(action)

	// Exact match
	if rule == action {
		return true
	}

	// Handle aliases
	switch rule {
	case ActionDrain:
		// "drain" rule also covers cordon/uncordon
		return action == ActionDrain || action == ActionCordon
	case ActionDelete:
		return action == ActionDelete
	case ActionScale:
		return action == ActionScale
	case ActionEdit:
		return action == ActionEdit || action == ActionPatch
	case ActionApply:
		return action == ActionApply || action == ActionCreate
	case ActionExec:
		return action == ActionExec
	case ActionRollout:
		return action == ActionRollout
	}

	return false
}

// GetActionSeverity returns a severity level for display purposes
func GetActionSeverity(action string) string {
	switch action {
	case ActionDelete, ActionDrain:
		return "high"
	case ActionScale, ActionCordon:
		return "medium"
	case ActionEdit, ActionPatch, ActionRollout:
		return "medium"
	case ActionApply, ActionCreate:
		return "low"
	default:
		return "none"
	}
}

// DescribeAction returns a human-readable description of the action
func DescribeAction(action string) string {
	switch action {
	case ActionDelete:
		return "Delete resources"
	case ActionDrain:
		return "Drain node (evict all pods)"
	case ActionCordon:
		return "Cordon/uncordon node"
	case ActionScale:
		return "Scale deployment replicas"
	case ActionEdit:
		return "Edit resource configuration"
	case ActionPatch:
		return "Patch resource"
	case ActionApply:
		return "Apply configuration"
	case ActionCreate:
		return "Create resource"
	case ActionExec:
		return "Execute command in pod"
	case ActionRollout:
		return "Manage rollout"
	default:
		return action
	}
}

