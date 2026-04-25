# SDP Quick Start

Get SDP running in your project in 30 minutes or less.

## Choose Your Path

| Path | Time | What You Get | When to Use |
|------|------|-------------|-------------|
| **CI Gates Only** | 30 min | Evidence validation, protocol compliance, coverage enforcement at PR time | First pilot, low-risk evaluation |
| **Contracted Runtime** | 60 min | CI gates + schema validation at ingest, runtime decisions, handoff events | Production adoption, multi-agent coordination |
| **Full Orchestration** | 2-4 hours | Event-driven agent loop, findings, runtime decisions, staged rollout | Full SDP lifecycle management |

## CI Gates Only (30 min)

The fastest adoption path. No changes to your agent stack or runtime.

```bash
# 1. Install
go install ./cmd/sdp@latest

# 2. Initialize
cd /path/to/your/repo
sdp init

# 3. Add CI gates to your workflow
# See docs/reference/enterprise-pilot-ci-gate-only.md for full setup

# 4. Record evidence on your next PR
sdp skill record --skill build --type plan \
  --ws-id 00-001-01 \
  --data '{"scope_files":["cmd/app/main.go"],"action":"add feature","feature_id":"F001"}'

# 5. Push and verify gates pass
gh pr create --title "SDP pilot" --body "First PR with SDP gates"
```

**Full guide**: [enterprise-pilot-ci-gate-only.md](reference/enterprise-pilot-ci-gate-only.md)

## Contracted Runtime (60 min)

Extends CI gates with runtime schema validation and event-driven coordination.

```bash
# Prerequisite: CI gates only pilot complete

# 1. Enable contracted runtime mode
sdp config set runtime.mode contracted

# 2. Validate schemas at ingest
sdp contract validate --schemas schema/contracts/

# 3. Run pilot with runtime events
sdp run --mode pilot
```

**Full guide**: [enterprise-pilot-contracted-runtime.md](reference/enterprise-pilot-contracted-runtime.md)

## Rollback and Disable

If something goes wrong, SDP provides safe rollback paths:

```bash
# Immediate disable (stops all gates)
sdp config set gates.enabled false

# Safe rollback (preserves audit trail)
sdp deploy rollback <previous-tag>

# Full removal
rm -rf .sdp/
# Remove SDP jobs from .github/workflows/ci.yml
```

**Full guide**: [enterprise-pilot-rollback.md](reference/enterprise-pilot-rollback.md)

## Key Commands

| Command | Purpose |
|---------|---------|
| `sdp init` | Initialize SDP in a repo |
| `sdp version` | Check installed version |
| `sdp verify` | Run protocol consistency checks |
| `sdp skill record` | Record an evidence event |
| `sdp scope check` | Validate workstream scope |
| `sdp deploy staging` | Deploy to staging |
| `sdp deploy prod` | Promote to production |
| `sdp deploy rollback` | Rollback deployment |
| `sdp coverage check` | Enforce coverage minimum |
| `sdp policy check` | Run policy gate |

## Getting Help

- **Troubleshooting**: See the troubleshooting section in each pilot guide
- **Evidence schema**: [schema/evidence.schema.json](../schema/evidence.schema.json)
- **Contract workflow**: [reference/CONTRACT-WORKFLOW.md](reference/CONTRACT-WORKFLOW.md)
- **Evidence coverage**: [reference/EVIDENCE-COVERAGE.md](reference/EVIDENCE-COVERAGE.md)
