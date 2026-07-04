# SOFE CLI

> Command-line interface for the SOFE Open FinOps Engine. Evaluate cloud cost policies locally or via the cloud API.

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

### Local Mode (self-hosted, no account needed)

```bash
# Start the local evaluation server
sofe serve

# Evaluate policies against your AWS account
sofe evaluate --profile default --policies ./policies/

# With auto-start server
sofe evaluate --auto-serve --profile default
```

### Cloud Mode (uses api.sofe.dev)

```bash
# Set your API key (get one at platform.sofe.dev)
sofe config set api-key sk_sofe_your_key_here

# Evaluate via cloud
sofe evaluate --cloud

# Or pass key directly
sofe evaluate --cloud --api-key sk_sofe_xxx
```

## Configuration

```bash
# Show current config
sofe config show

# Set values
sofe config set mode cloud          # default to cloud mode
sofe config set api-key sk_sofe_xxx # persist API key
sofe config set profile my-profile  # AWS profile for local mode
sofe config set format json         # output format (table|json|markdown)
```

Config stored at `~/.sofe/config.yaml` (permissions 0600).

Environment variables (override config):
- `SOFE_API_KEY` — API key for cloud mode
- `SOFE_CLOUD_URL` — custom cloud URL (default: https://api.sofe.dev)

## Commands

| Command | Description |
|---------|-------------|
| `sofe evaluate` | Run policies against AWS resources |
| `sofe serve` | Start local evaluation server (port 8080) |
| `sofe policies` | List available policies |
| `sofe config set` | Set configuration value |
| `sofe config show` | Show current configuration |

## Evaluate Flags

| Flag | Description |
|------|-------------|
| `--cloud` | Use cloud API instead of local server |
| `--api-key` | API key for cloud mode |
| `--profile` | AWS profile (local mode) |
| `--policies` | Policies directory |
| `--fail-on` | Exit 1 if findings at/above severity (critical\|high\|medium\|low) |
| `--resource-types` | Filter resource types (comma-separated) |
| `--auto-serve` | Auto-start local server if not running |
| `--format` | Output format (table\|json\|markdown) |

## CI/CD

```bash
# Block deploys with critical findings
sofe evaluate --cloud --api-key $SOFE_API_KEY --fail-on high
```

## Local vs Cloud

| | Local | Cloud |
|--|-------|-------|
| Needs | sofe-server running locally | API key from platform.sofe.dev |
| Auth | None (localhost) | X-API-Key header |
| AWS access | Your local credentials | Connected account (IAM role) |
| Rate limit | None | 10/day (free), 1000/day (pro) |
| History | None | Stored in platform |

## Links

- **Engine:** [github.com/breakingthecloud/sofe](https://github.com/breakingthecloud/sofe) (Python, Apache 2.0)
- **Platform:** [platform.sofe.dev](https://platform.sofe.dev) (sign up, API keys)
- **Docs:** [sofe.dev/docs](https://sofe.dev/docs)
- **PyPI:** [pypi.org/project/sofe](https://pypi.org/project/sofe)

## License

Apache 2.0
