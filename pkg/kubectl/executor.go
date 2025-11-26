package kubectl

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// GetCurrentContext returns the current kubectl context name
func GetCurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Return stderr if available for better error messages
		if stderr.Len() > 0 {
			return "", &ContextError{Message: strings.TrimSpace(stderr.String())}
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ContextError represents an error getting the kubectl context
type ContextError struct {
	Message string
}

func (e *ContextError) Error() string {
	return e.Message
}

// Execute runs kubectl with the given arguments and returns the exit code
func Execute(args []string) int {
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		// Non-exit error (e.g., kubectl not found)
		return 1
	}

	return 0
}

// ExecuteWithOutput runs kubectl and captures the output
func ExecuteWithOutput(args []string) (string, string, int) {
	cmd := exec.Command("kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// GetClusterInfo returns information about the current cluster
func GetClusterInfo() (server string, err error) {
	stdout, _, exitCode := ExecuteWithOutput([]string{
		"config", "view", "--minify", "-o", "jsonpath={.clusters[0].cluster.server}",
	})

	if exitCode != 0 {
		return "", &ContextError{Message: "failed to get cluster info"}
	}

	return strings.TrimSpace(stdout), nil
}

// GetNamespace returns the namespace from args or the default namespace
func GetNamespace(args []string) string {
	// Check if namespace is specified in args
	for i, arg := range args {
		if arg == "-n" || arg == "--namespace" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if strings.HasPrefix(arg, "-n=") {
			return strings.TrimPrefix(arg, "-n=")
		}
		if strings.HasPrefix(arg, "--namespace=") {
			return strings.TrimPrefix(arg, "--namespace=")
		}
	}

	// Get default namespace from context
	stdout, _, exitCode := ExecuteWithOutput([]string{
		"config", "view", "--minify", "-o", "jsonpath={.contexts[0].context.namespace}",
	})

	if exitCode == 0 && strings.TrimSpace(stdout) != "" {
		return strings.TrimSpace(stdout)
	}

	return "default"
}

// CheckKubectlAvailable checks if kubectl is available in PATH
func CheckKubectlAvailable() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

// GetAllContexts returns all available kubectl contexts
func GetAllContexts() ([]string, error) {
	stdout, _, exitCode := ExecuteWithOutput([]string{
		"config", "get-contexts", "-o", "name",
	})

	if exitCode != 0 {
		return nil, &ContextError{Message: "failed to get contexts"}
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	contexts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			contexts = append(contexts, line)
		}
	}

	return contexts, nil
}

