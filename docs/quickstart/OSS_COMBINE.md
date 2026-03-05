# OSS Combine Quickstart

This guide walks you through setting up the OSS Combine integration environment for SDP.

## Prerequisites

Before running `sdp up`, ensure you have the following tools installed:

| Tool | Purpose | Install |
|------|---------|---------|
| `git` | Version control | https://git-scm.com/downloads |
| `gh` | GitHub CLI | https://cli.github.com/ |
| `bd` | Beads issue tracker | `go install github.com/your-org/beads@latest` |
| `go` | Go runtime (1.21+) | https://go.dev/doc/install |
| `jq` | JSON processor | https://stedolan.github.io/jq/download/ |
| `opa` | Open Policy Agent | https://www.openpolicyagent.org/docs/latest/#running-opa |

## Quick Start

### 1. Provision the Environment

```bash
# From your project root
sdp up --profile oss-combine
```

This command:
- Validates all required tools
- Creates `.sdp/` directory structure
- Generates configuration files
- Sets up evidence collection paths

### 2. Verify the Setup

```bash
# Check created directories
ls -la .sdp/

# View configuration
cat .sdp/config.yaml
```

Expected structure:
```
.sdp/
├── config.yaml           # SDP configuration
├── evidence/             # Evidence envelopes
├── checkpoints/          # Session checkpoints
├── findings/             # CI findings cache
├── sessions/             # Session state
└── traces/               # Execution traces
```

### 3. Dry-Run Mode

Preview what `sdp up` would do without making changes:

```bash
sdp up --profile oss-combine --dry-run
```

### 4. Rollback

Remove all provisioned state:

```bash
sdp up --profile oss-combine --rollback
```

## Configuration

### `.sdp/config.yaml`

The main configuration file controls:

```yaml
version: "1.0"
profile: oss-combine

beads:
  prefix: sdplab-        # Issue prefix for this project
  db: .beads/beads.db    # Beads database location

github:
  findings_artifact: sdp-findings  # Artifact name for CI findings
  workflows:
    - CI
    - Protocol Validation

contracts:
  orchestration_event: schema/contracts/orchestration-event.schema.json
  runtime_decision: schema/contracts/runtime-decision.schema.json

findings:
  protocol: schema/findings/protocol-findings.schema.json
  docs: schema/findings/docs-findings.schema.json
```

## Integration with Beads

### Create Issues

```bash
# Create a new issue linked to a workstream
bd create "Implement feature X" \
  --labels F069,00-069-01 \
  --prefix sdplab-
```

### Sync from CI Findings

```bash
# Pull findings from GitHub CI and create Beads tasks
sdp-gh-findings-sync --repo owner/repo --dir .sdp/findings
```

## Evidence Collection

Evidence is automatically collected during agent sessions:

```
.sdp/evidence/
├── run-123.json         # Evidence envelope for run 123
├── run-124.json         # Evidence envelope for run 124
└── ...
```

Validate evidence:

```bash
sdp-evidence validate --evidence .sdp/evidence/run-123.json
```

## Troubleshooting

### Missing Tools

If `sdp up` reports missing tools:

```
Error: Missing required tools:
  - bd: Install beads: go install github.com/your-org/beads@latest
```

Install the missing tool and run `sdp up` again.

### Permission Denied

If you get permission errors:

```bash
# Check directory permissions
ls -la .sdp/

# Fix permissions if needed
chmod -R 755 .sdp/
```

### Git Repository Not Found

Ensure you're in a git repository:

```bash
git status
```

If not, initialize one:

```bash
git init
```

## Next Steps

1. **Configure Beads prefix** - Edit `.sdp/config.yaml` to match your project
2. **Run CI validation** - Push a PR to trigger protocol checks
3. **Sync findings** - Use `sdp-gh-findings-sync` to create improvement tasks
4. **Review contracts** - Check `schema/contracts/` for integration schemas

## Reference

- [CONTRACTS.md](../protocol/CONTRACTS.md) - Contract schema documentation
- [FINDINGS_SCHEMA.md](../protocol/FINDINGS_SCHEMA.md) - Findings schema documentation
- [ROADMAP.md](../roadmap/ROADMAP.md) - Feature roadmap
