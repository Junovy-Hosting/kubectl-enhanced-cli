# kubectl-enhanced-cli

A kubectl wrapper that adds CLI-level RBAC controls for production safety. It intercepts kubectl commands and can block or require confirmation for destructive operations based on per-cluster configuration.

```bash
kctl -n default delete configmaps game-demo
⚠️  CONFIRMATION REQUIRED
│ Action:  Delete resources
│ Cluster: admin@my-production-cluster (production)
│ Namespace: default
│ Command: kubectl -n default delete configmaps game-demo

Do you want to proceed? [y/N]:
```

## Features

- **Per-cluster access rules**: Configure which actions require confirmation or are blocked based on cluster context
- **Tier-based defaults**: Automatically apply rules based on cluster naming patterns (e.g., `*-prod`, `*-staging`)
- **Confirmation prompts**: Interactive confirmation for dangerous operations on protected clusters
- **Dual invocation modes**: Use as a standalone wrapper (`kctl`) or as a kubectl plugin (`kubectl enhanced`)
- **Passthrough design**: All kubectl commands work exactly as expected, with safety checks added

## Installation

```bash
# Build from source
make build

# Install to PATH (creates symlinks for both modes)
make install

# Initialize config file (interactive wizard)
kctl init

# Or non-interactively with defaults
kctl init --non-interactive
```

After installation, you'll have:

- `kubectl-enhanced-cli` - Main binary
- `kctl` - Symlink for wrapper mode
- `kubectl-enhanced` - Symlink for plugin mode (use as `kubectl enhanced`)

## Usage

### Wrapper Mode

```bash
kctl get pods                    # Safe operations pass through
kctl delete pod my-pod           # May require confirmation on prod clusters
kctl delete pod my-pod --yes     # Skip confirmation prompt
kctl drain node-1                # Requires confirmation on prod clusters
```

### Plugin Mode

```bash
kubectl enhanced get pods
kubectl enhanced delete pod my-pod
kubectl enhanced delete pod my-pod --yes
```

### Special Flags

```bash
kctl --version        # Print version information
kctl --help           # Print help
kctl --config-path    # Print config file location
```

### Config Initialization

The `init` command helps you create a configuration file:

```bash
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
```

**Interactive mode** will:

1. Detect your existing kubectl contexts
2. Let you categorize specific clusters (production, staging, dev)
3. Configure tier patterns for automatic categorization
4. Set up which actions require confirmation

**Non-interactive options:**

- `--prod-patterns` - Comma-separated production cluster patterns
- `--staging-patterns` - Comma-separated staging cluster patterns
- `--dev-patterns` - Comma-separated development cluster patterns
- `--prod-actions` - Actions requiring confirmation on production (default: delete,drain)
- `--staging-actions` - Actions requiring confirmation on staging (default: delete)
- `--blocked-actions` - Globally blocked actions

## Configuration

Configuration file location: `~/.config/kubectl-enhanced/config.yaml`

### Example Configuration

```yaml
# Global defaults applied to all clusters unless overridden
defaults:
  require_confirmation: false
  blocked_actions: []

# Explicit cluster rules (takes priority over tier patterns)
clusters:
  # Exact match for a production cluster
  production-us-east-1:
    tier: production
    require_confirmation: [delete, drain]
    blocked_actions: []

  # Pattern match for all staging clusters
  staging-*:
    tier: staging
    require_confirmation: [delete]
    blocked_actions: []

# Tier-based rules (fallback when no explicit cluster match)
tiers:
  production:
    patterns:
      - "*-prod"
      - "*-production"
      - "prod-*"
      - "production-*"
    require_confirmation:
      - delete
      - drain
    blocked_actions: []

  staging:
    patterns:
      - "*-staging"
      - "*-stg"
      - "staging-*"
    require_confirmation:
      - delete
    blocked_actions: []

  development:
    patterns:
      - "*-dev"
      - "dev-*"
      - "local*"
      - "minikube"
      - "docker-desktop"
      - "kind-*"
    require_confirmation: []
    blocked_actions: []
```

### Configuration Hierarchy

Rules are resolved in the following order:

1. **Exact cluster match** - If the current context matches a key in `clusters` exactly
2. **Pattern cluster match** - If the current context matches a glob pattern in `clusters`
3. **Tier pattern match** - If the current context matches a pattern in any `tiers` entry
4. **Defaults** - Global defaults are used as fallback

### Supported Actions

Actions that can be configured for confirmation or blocking:

| Action    | kubectl Commands                                      |
| --------- | ----------------------------------------------------- |
| `delete`  | `kubectl delete`                                      |
| `drain`   | `kubectl drain`, `kubectl cordon`, `kubectl uncordon` |
| `scale`   | `kubectl scale`                                       |
| `edit`    | `kubectl edit`, `kubectl patch`                       |
| `apply`   | `kubectl apply`, `kubectl create`                     |
| `exec`    | `kubectl exec`                                        |
| `rollout` | `kubectl rollout`                                     |

## How It Works

```
┌─────────────────┐
│  kctl delete    │
│    pod foo      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Get current     │
│ kubectl context │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Load config &   │
│ resolve rules   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Action blocked? │────▶│ Exit with error │
└────────┬────────┘ yes └─────────────────┘
         │ no
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Confirmation    │────▶│ Prompt user     │
│ required?       │ yes │ for confirm     │
└────────┬────────┘     └────────┬────────┘
         │ no                    │
         ▼                       ▼
┌─────────────────────────────────────────┐
│         Execute: kubectl delete pod foo │
└─────────────────────────────────────────┘
```

## Environment Variables

- `NO_COLOR` - Disable colored output when set to any value
- `XDG_CONFIG_HOME` - Override default config directory (default: `~/.config`)
- `KUBECONFIG` - Standard kubectl config file location

## Comparison with kubectl

| Feature              | kubectl | kubectl-enhanced-cli        |
| -------------------- | ------- | --------------------------- |
| Basic operations     | ✅      | ✅ (passthrough)            |
| Production safety    | ❌      | ✅ Configurable per cluster |
| Confirmation prompts | ❌      | ✅ For destructive actions  |
| Action blocking      | ❌      | ✅ Configurable             |
| Tier-based rules     | ❌      | ✅ Pattern matching         |

## Development

```bash
# Build for development (no version info)
make build-dev

# Run tests
make test

# Clean build artifacts
make clean
```

## License

MIT License - See LICENSE file for details.
