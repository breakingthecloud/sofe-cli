# ⚡ SOFE CLI

**Fast Go binary for evaluating FinOps policies via the SOFE API.**

The SOFE CLI is a lightweight client that calls [sofe-server](https://github.com/breakingthecloud/sofe-server) to evaluate policies against live AWS infrastructure. Configure it to point at a local server or at `api.sofe.dev` for the hosted tier.

```bash
sofe evaluate --policies ./policies --profile production
```

```
☁️  46 resources scanned | ⚡ 12 findings

  SEVERITY │ POLICY              │ RESOURCE        │ MESSAGE
-----------+---------------------+-----------------+---------------------------
  🟠 high  │ no-idle-ec2         │ i-0abc123def    │ avg_cpu = 2.1% (<5%)
  🟡 medium│ require-cost-tags   │ my-bucket       │ missing: owner
  🔵 low   │ s3-require-env-tag  │ logs-bucket     │ missing: Environment

Summary: 12 findings | Potential savings: $365.00/mo
```

---

## Install

### Binary (download)

```bash
# macOS (Apple Silicon)
curl -L https://github.com/breakingthecloud/sofe-cli/releases/latest/download/sofe-darwin-arm64 -o sofe
chmod +x sofe && sudo mv sofe /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/breakingthecloud/sofe-cli/releases/latest/download/sofe-linux-amd64 -o sofe
chmod +x sofe && sudo mv sofe /usr/local/bin/
```

### Homebrew (coming soon)

```bash
brew install breakingthecloud/tap/sofe
```

### From source

```bash
go install github.com/breakingthecloud/sofe-cli@latest
```

---

## Prerequisites

The CLI requires a running [sofe-server](https://github.com/breakingthecloud/sofe-server):

```bash
# Option 1: pip
pip install sofe-server && sofe-server

# Option 2: Docker
docker run -p 8080:8080 -v ~/.aws:/root/.aws:ro ghcr.io/breakingthecloud/sofe-server
```

---

## Configuration

Create `~/.sofe/config.yaml`:

```yaml
# Local development (no API key)
api_url: http://localhost:8080
api_key: ""
default_format: table
aws_profile: default
policies_dir: ./policies
```

```yaml
# Hosted tier (api.sofe.dev)
api_url: https://api.sofe.dev
api_key: sk-your-api-key-here
default_format: table
aws_profile: production
policies_dir: ./policies
```

If no config file exists, defaults to `http://localhost:8080` with no API key.

---

## Commands

### `sofe evaluate`

Evaluate policies against live AWS resources.

```bash
sofe evaluate --policies ./policies --profile production
sofe evaluate -p ./policies --format json
sofe evaluate -p ./policies --fail-on high          # exit 1 if high+ findings
sofe evaluate -p ./policies --resource-types aws.ec2,aws.s3
```

| Flag | Short | Description |
|------|:-----:|-------------|
| `--policies` | `-p` | Policies directory |
| `--profile` | | AWS profile name |
| `--format` | | Output: table, json, markdown |
| `--fail-on` | | Exit 1 if findings ≥ severity |
| `--resource-types` | | Filter resource types (comma-separated) |

### `sofe health`

Check if sofe-server is running.

```bash
sofe health
# ✅ SOFE Server 0.1.0 (ok)
```

### `sofe policies`

List loaded policies from the server.

```bash
sofe policies
```

```
  NAME                       │ SEVERITY │ RESOURCE TYPES              │ METRIC
-----------------------------+----------+-----------------------------+----------------------
  no-idle-ec2                │ high     │ aws.ec2                     │ avg_cpu_utilization
  require-cost-tags          │ medium   │ aws.ec2, aws.s3, aws.lambda │ has_tag:owner
  no-unattached-ebs          │ medium   │ aws.ebs                     │ attached
  s3-require-environment-tag │ low      │ aws.s3                      │ has_tag:Environment

4 policies loaded
```

---

## Output Formats

### Table (default)

Colored terminal output with severity icons.

### JSON

```bash
sofe evaluate -p ./policies --format json > findings.json
```

Full structured response for automation and CI/CD pipelines.

### Markdown

```bash
sofe evaluate -p ./policies --format markdown >> report.md
```

Markdown table for PR comments and documentation.

---

## CI/CD Integration

### GitHub Actions

```yaml
- name: FinOps Policy Check
  run: |
    curl -L https://github.com/breakingthecloud/sofe-cli/releases/latest/download/sofe-linux-amd64 -o sofe
    chmod +x sofe
    ./sofe evaluate --policies ./policies --fail-on high --format json > findings.json
```

### Exit Codes

| Code | Meaning |
|:----:|---------|
| 0 | No violations (or below `--fail-on` threshold) |
| 1 | Violations found at or above `--fail-on` severity |

---

## Architecture

```
┌──────────────┐         ┌──────────────────┐         ┌─────────────┐
│  sofe (Go)   │──HTTP──▶│  sofe-server     │──boto3──▶│  AWS APIs   │
│              │         │  (FastAPI :8080)  │         │             │
│  ~/.sofe/    │         │                  │         │ EC2, S3,    │
│  config.yaml │         │  imports sofe    │         │ Lambda, RDS │
└──────────────┘         └──────────────────┘         └─────────────┘
```

---

## Cross-Compile Targets

| OS | Arch | Binary |
|----|------|--------|
| macOS | arm64 | `sofe-darwin-arm64` (12MB) |
| Linux | amd64 | `sofe-linux-amd64` (12MB) |
| Linux | arm64 | `sofe-linux-arm64` (11MB) |

Build all:

```bash
GOOS=darwin GOARCH=arm64 go build -o dist/sofe-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -o dist/sofe-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o dist/sofe-linux-arm64 .
```

---

## Related

- [sofe](https://github.com/breakingthecloud/sofe) — Engine library + Python CLI (`pip install sofe`)
- [sofe-server](https://github.com/breakingthecloud/sofe-server) — FastAPI REST API
- [sofe-catalog](https://github.com/breakingthecloud/sofe-catalog) — Browse policies, collectors, coverage
- [PyPI](https://pypi.org/project/sofe/) — Python package

---

## License

Apache 2.0
