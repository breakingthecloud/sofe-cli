# Contributing to SOFE CLI

Thanks for your interest in contributing to the SOFE CLI!

## What is sofe-cli?

A Go binary with 19 commands for FinOps evaluation, interactive TUI, Terraform scanning, and cloud API access. It talks to either a local sofe-server or the cloud API (api.sofe.dev).

## Setting Up Development

```bash
git clone https://github.com/breakingthecloud/sofe-cli.git
cd sofe-cli
go mod download
go build -o sofe .
./sofe --help
```

### For local mode (needs sofe-server):
```bash
pip install sofe sofe-server
sofe-server &  # starts on :8080
./sofe evaluate --profile your-aws-profile
```

### For cloud mode:
```bash
./sofe login  # device flow, opens browser
./sofe evaluate --cloud
```

## Project Structure

```
sofe-cli/
├── main.go                     # Entry point
├── cmd/
│   ├── root.go                 # Cobra root command + banner
│   ├── evaluate.go             # sofe evaluate
│   ├── interactive.go          # sofe interactive (bubbletea TUI)
│   ├── terraform.go            # sofe terraform (pre-deploy scan)
│   ├── explain.go              # sofe explain (AI)
│   ├── serve.go                # sofe serve / serve stop
│   ├── login.go                # sofe login (device flow)
│   ├── history.go / diff.go    # Cloud evaluation history
│   ├── watch.go / top.go       # Monitoring commands
│   └── helpers.go              # Shared client constructors
├── internal/
│   ├── client/client.go        # HTTP client for API calls
│   ├── config/config.go        # ~/.sofe/config.yaml management
│   ├── output/output.go        # Terminal formatting (colors, tables)
│   └── terraform/              # tfplan parser + 6 pre-deploy policies
│       ├── parser.go
│       └── policies.go
└── tests/                      # Integration test scripts
```

## Adding a New Command

1. Create `cmd/your_command.go`:
```go
package cmd

import "github.com/spf13/cobra"

var yourCmd = &cobra.Command{
    Use:   "your-command",
    Short: "What it does",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(yourCmd)
}
```

2. If it calls the cloud API, use helpers:
```go
client := getCloudClient()  // api.sofe.dev
aiClient := getAIClient()   // ai-api.sofe.dev
```

3. Build and test:
```bash
go build -o sofe .
./sofe your-command
```

## Adding Terraform Policies

Add to `internal/terraform/policies.go`:
```go
func checkYourPolicy(r Resource) *Finding {
    // Check resource fields
    // Return nil if ok, or &Finding{...} if violation
}
```

Register in `DefaultPolicies()` slice.

## Code Standards

- Go 1.21+
- Use cobra for commands
- Use bubbletea/lipgloss for TUI components
- `internal/` packages for non-exported logic
- No hardcoded credentials (everything from config/env)
- `getCloudClient()` / `getAIClient()` for API access

## Release Process

Releases are automatic via goreleaser:
1. Tag: `git tag v0.7.0`
2. Push: `git push origin v0.7.0`
3. GitHub Actions builds binaries for all platforms
4. Release published with changenotes

## Pull Requests

- Create feature branch: `feat/add-costs-command`
- Ensure `go build` passes
- Test the command works (local or cloud mode)
- Submit PR with description

## What We Accept

- ✅ New commands that wrap existing API endpoints
- ✅ TUI improvements (bubbletea components)
- ✅ New Terraform policies
- ✅ Output format improvements
- ✅ Bug fixes
- ✅ Shell completions (bash, zsh, fish)

## What We Don't Accept

- ❌ Direct AWS calls (CLI should talk to server/API, not AWS directly)
- ❌ Embedded secrets or tokens
- ❌ Heavy dependencies (keep binary small)
- ❌ Breaking changes to existing command flags

## Questions?

Open an issue or see [sofe.dev/docs/cli](https://sofe.dev/docs/cli).
