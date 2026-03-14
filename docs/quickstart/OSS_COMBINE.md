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

### Guard Denial

If `sdp-guard` blocks a command:

```
Error: Command blocked by policy:
  - git push --force is not allowed in build phase
```

Use the recovery guidance:

```bash
sdp-ready --instructions --action resolve_blockers
```

## Failure Recovery Guidance

SDP provides actionable next steps for all failure modes.

### Get Recovery Instructions

```bash
sdp-ready --instructions
```

Output includes:
- What action to take
- Why it's recommended
- Command to execute
- Expected outcome

### Common Failure Scenarios

| Failure | Recovery Command |
|---------|------------------|
| Missing tools | `sdp up --profile oss-combine --dry-run` |
| Blocked work | `bd blocked` then resolve each blocker |
| Guard denial | Use safe alternative (e.g., `git push` instead of `--force`) |
| Invalid config | Edit `.sdp/config.yaml` and re-run `sdp up` |
| CI findings | `sdp-gh-findings-sync` to create fix tasks |

### Blocked Work Recovery

When all work is blocked:

```bash
$ sdp-ready
📋 Ready work (0 ready, 3 blocked):
  ○ [P2] sdplab-abc: Feature X (blocked_by: sdplab-0)
  ○ [P2] sdplab-def: Feature Y (blocked_by: sdplab-0)
  ○ [P2] sdplab-ghi: Feature Z (blocked_by: sdplab-1)

Next action:
- Recommendation: Resolve blockers
- Reason: No ready work: 3 issue(s) are blocked
- Command: bd blocked
```

Resolve the blocking issues, then work becomes ready automatically.

### JSON Output for Agents

All guidance is machine-readable:

```bash
sdp-ready --format status-view --instructions
```

See [NEXT_STEP_GUIDE.md](./NEXT_STEP_GUIDE.md) for contract schemas.

## Next Steps

1. **Configure Beads prefix** - Edit `.sdp/config.yaml` to match your project
2. **Run CI validation** - Push a PR to trigger protocol checks
3. **Sync findings** - Use `sdp-gh-findings-sync` to create improvement tasks
4. **Review contracts** - Check `schema/contracts/` for integration schemas

## Reference

- [CONTRACTS.md](../protocol/CONTRACTS.md) - Contract schema documentation
- [FINDINGS_SCHEMA.md](../protocol/FINDINGS_SCHEMA.md) - Findings schema documentation
- [ROADMAP.md](../roadmap/ROADMAP.md) - Feature roadmap
