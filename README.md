<p align="center">
  <img alt="SOFE CLI" src="https://img.shields.io/badge/⌨️-SOFE_CLI-3B82F6?style=for-the-badge" height="50">
</p>

<p align="center">
  <b>Command-line interface for the SOFE Open FinOps Engine</b><br>
  19 commands, interactive TUI, AI-powered remediation.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a>
  ·
  <a href="#commands">Commands</a>
  ·
  <a href="#new-in-v030">v0.3 Features</a>
  ·
  <a href="#ecosystem">Ecosystem</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/license-Apache_2.0-3B82F6?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/modes-Local+Cloud-3B82F6?style=flat-square" alt="Local + Cloud">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square" alt="PRs">
</p>

---

Evaluate FinOps policies, browse findings in an interactive TUI, get AI-powered remediation commands — all from your terminal.

```bash
# Quick install (macOS/Linux)
curl -fsSL https://sofe.dev/install.sh | bash

# Evaluate your AWS account
sofe evaluate --profile default --policies ./policies/
```

## Install

```bash
# Quick install (macOS/Linux)
curl -fsSL https://sofe.dev/install.sh | bash

# Go install
go install github.com/breakingthecloud/sofe-cli@latest

# Or download binary from GitHub Releases
# https://github.com/breakingthecloud/sofe-cli/releases
```

## Quick Start

### Local Mode

```bash
# Start the local evaluation server
sofe serve

# Evaluate policies against your AWS account
sofe evaluate --profile default --policies ./policies/

# With auto-start server
sofe evaluate --auto-serve --profile default
```

### Cloud Mode

```bash
# Set your API key (get one at platform.sofe.dev)
sofe config set api-key sk_sofe_your_key_here

# Evaluate via cloud
sofe evaluate --cloud

# Or pass key directly
sofe evaluate --cloud --api-key sk_sofe_xxx
```

## Commands

### Core

| Command | Description |
|---------|-------------|
| `sofe evaluate` | Run policies against AWS resources (spinner + summary card) |
| `sofe interactive` | Split-panel TUI (findings + detail + AI) |
| `sofe history` | List past evaluations (cloud mode) |
| `sofe watch` | Monitor with live sparkline + change detection |

### AI + Remediation

| Command | Description |
|---------|-------------|
| `sofe explain` | AI explanation for a specific finding |
| `sofe remediate` | Show/execute remediation CLI commands |
| `sofe top` | Findings ranked by frequency |
| `sofe diff` | Compare two evaluations (new/fixed/unchanged) |

### Account

| Command | Description |
|---------|-------------|
| `sofe login` | Authenticate via browser (device flow) |
| `sofe upgrade` | Connect to SOFE Cloud or upgrade to Pro |
| `sofe status` | Show account status (tier, evals, AI usage) |
| `sofe config` | Manage configuration (set/show) |

### Utilities

| Command | Description |
|---------|-------------|
| `sofe serve` | Start local evaluation server (port 8080) |
| `sofe policies` | List available policies |
| `sofe changelog` | Show release notes |
| `sofe version` | Build info (commit, date, Go, OS) |
| `sofe terraform` | Scan Terraform plans |
| `sofe completion` | Shell autocompletion (bash/zsh/fish) |

## New in v0.3.0

### Interactive TUI

```bash
sofe interactive              # Browse findings with keyboard navigation
sofe interactive abc123-eval-id  # Or specify an evaluation
```

Navigate with ↑/↓, press Enter for details, `e` for AI explanation, `q` to quit.

### AI Explain & Remediate

```bash
sofe explain abc123-eval-id --finding 2     # AI explanation
sofe remediate abc123-eval-id --finding 2   # Show remediation commands
sofe remediate abc123-eval-id --finding 2 --execute  # Execute interactively
```

### Upgrade & Status

```bash
sofe upgrade                    # Connect CLI to SOFE Cloud
sofe upgrade pro                # Upgrade to Pro tier
sofe status                     # Check your status
```

## Evaluate Flags

| Flag | Description |
|------|-------------|
| `--cloud` | Use cloud API instead of local server |
| `--api-key` | API key for cloud mode |
| `--profile` | AWS profile (local mode) |
| `--policies` | Policies directory |
| `--fail-on` | Exit 1 if findings at/above severity |
| `--resource-types` | Filter resource types (comma-separated) |
| `--auto-serve` | Auto-start local server if not running |
| `--format` | Output format (table\|json\|markdown) |

## Configuration

```bash
sofe config show                              # Show current config
sofe config set mode cloud                    # Default to cloud mode
sofe config set api-key sk_sofe_xxx           # Persist API key
sofe config set profile my-profile            # AWS profile for local mode
sofe config set format json                   # Output format
```

Config stored at `~/.sofe/config.yaml` (permissions 0600).

Environment variables: `SOFE_API_KEY`, `SOFE_CLOUD_URL`.

## Local vs Cloud

| | Local | Cloud |
|--|-------|-------|
| Needs | sofe-server running locally | API key from platform.sofe.dev |
| Auth | None (localhost) | X-API-Key header |
| AWS access | Your local credentials | Connected account (IAM role) |
| Rate limit | None | 10/day (free), 1000/day (pro) |
| History | None | Stored in platform |

## CI/CD

```bash
# Block deploys with critical findings
sofe evaluate --cloud --api-key $SOFE_API_KEY --fail-on high
```

## Ecosystem

| Project | Description |
|---------|-------------|
| [sofe](https://github.com/breakingthecloud/sofe) | Python engine (collectors + policies) |
| [sofe-server](https://github.com/breakingthecloud/sofe-server) | REST API server |
| [sofe-action](https://github.com/breakingthecloud/sofe-action) | GitHub Action |
| [platform.sofe.dev](https://platform.sofe.dev) | SaaS dashboard (free tier) |
| [sofe.dev/docs](https://sofe.dev/docs) | Documentation |

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

<p align="center">
  <a href="https://sofe.dev">sofe.dev</a> · <a href="https://github.com/breakingthecloud/sofe">Engine</a> · <a href="https://finoptix.dev">finoptix.dev</a>
</p>
<p align="center">
  <sub>19 commands. Zero AWS bill surprises.</sub>
</p>
