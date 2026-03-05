# OSS Combine Integration Wiring

> **Version:** 1.0.0
> **Last Updated:** 2026-03-01
> **Profile:** oss-combine

## Overview

The OSS Combine profile wires together three open-source tools with SDP contracts:

| Tool | Role | Contract |
|------|------|----------|
| OhMyOpenCode (OMO) | Session orchestration, agent runtime | `orchestration-event.schema.json` |
| Beads | Issue tracking, task queue | `runtime-decision.schema.json` |
| Gas Town | Build monitoring, witness escalation | `orchestration-event.schema.json` |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        SDP Core                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Guard     │  │  Evidence   │  │   Contract SDK          │  │
│  │  (scopes)   │  │  (in-toto)  │  │   (interfaces)          │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
└─────────┼────────────────┼──────────────────────┼───────────────┘
          │                │                      │
          ▼                ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Adapter Layer                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  OMO        │  │   Beads     │  │   Gas Town              │  │
│  │  Adapter    │  │   Adapter   │  │   Adapter               │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
└─────────┼────────────────┼──────────────────────┼───────────────┘
          │                │                      │
          ▼                ▼                      ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐
│ OpenCode    │  │   Beads     │  │   Gas Town                  │
│ Session     │  │   DB        │  │   Build System              │
└─────────────┘  └─────────────┘  └─────────────────────────────┘
```

## Adapter Endpoints

### OMO (OhMyOpenCode)

```yaml
endpoint: opencode://session
contract: schema/contracts/orchestration-event.schema.json
```

The OMO adapter bridges SDP orchestration with OpenCode sessions:

- **Event Types**: `session_start`, `task_claim`, `phase_transition`, `session_end`
- **Decision Points**: Continue, pause, escalate, rollback
- **Integration**: Uses OpenCode's agent model for role-based execution

### Beads

```yaml
endpoint: beads://local
contract: schema/contracts/runtime-decision.schema.json
```

The Beads adapter provides task queue management:

- **Actions**: `claim`, `update`, `close`, `sync`
- **State Machine**: `OPEN` → `IN_PROGRESS` → `CLOSED` / `BLOCKED`
- **Integration**: Uses `bd` CLI for issue operations

### Gas Town

```yaml
endpoint: gastown://build
contract: schema/contracts/orchestration-event.schema.json
```

The Gas Town adapter provides build monitoring:

- **Events**: `build.started`, `build.completed`, `witness.triggered`
- **Witness Integration**: Escalation to human review on policy violation
- **Integration**: GUPP pattern for guarded upstream package propagation

## Guard Scope Configuration

### Default Scope

```yaml
name: default
path: docs/workstreams/backlog
evidence_path: .sdp/evidence
```

The default scope covers workstream execution:
- Tasks claimed from Beads queue
- Evidence written to `.sdp/evidence/`
- Checkpoints stored in `.sdp/checkpoints/`

### Contracts Scope

```yaml
name: contracts
path: internal/adapters
evidence_path: .sdp/evidence
```

The contracts scope covers adapter development:
- SDK interface changes require evidence
- Contract compatibility tests validate changes
- Breaking changes require migration plan

### Bridge Scope

```yaml
name: bridge
path: internal/bridge
evidence_path: .sdp/evidence
```

The bridge scope covers GitHub-Beads sync:
- Findings from CI are synced to Beads
- Idempotency prevents duplicate issues
- Deduplication by finding fingerprint

## Evidence Path Initialization

The `sdp up --profile oss-combine` command creates:

```
.sdp/
├── evidence/          # in-toto attestations
├── checkpoints/       # FSM state snapshots
├── findings/          # CI findings from GitHub
├── sessions/          # Session logs
└── traces/            # OpenTelemetry traces
```

### Evidence Structure

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "predicateType": "sdp.dev/coding-workflow/v1",
  "predicate": {
    "workstream_id": "00-069-02",
    "beads_id": "sdplab-vq7",
    "phases": [...],
    "artifacts": [...],
    "verdict": "passed"
  }
}
```

## Config Lint Command

Verify configuration validity:

```bash
sdp config lint --profile oss-combine
```

Checks:
- [x] All adapter endpoints defined
- [x] All contract paths exist
- [x] Guard scopes reference valid paths
- [x] Evidence paths are writable
- [x] Required fields present

## Profile Upgrade Path

### From v1.0.0 to v1.1.0

1. Backup existing config:
   ```bash
   cp -r configs/profiles/oss-combine configs/profiles/oss-combine.bak
   ```

2. Generate new config:
   ```bash
   sdp up --profile oss-combine --upgrade
   ```

3. Validate:
   ```bash
   sdp config lint --profile oss-combine
   ```

### Migration from Manual Setup

If you previously configured SDP manually:

1. Export existing settings:
   ```bash
   cat .sdp/config.yaml > /tmp/old-config.yaml
   ```

2. Run profile provisioning:
   ```bash
   sdp up --profile oss-combine
   ```

3. Merge custom settings:
   ```bash
   # Compare and merge differences
   diff /tmp/old-config.yaml configs/profiles/oss-combine/config.yaml
   ```

## Troubleshooting

### Missing Adapter Endpoint

**Error**: `adapter endpoint not configured: omo`

**Solution**: Check `configs/profiles/oss-combine/adapters.yaml` for missing endpoint.

### Guard Scope Not Found

**Error**: `guard scope not found: default`

**Solution**: Ensure `configs/profiles/oss-combine/guard.yaml` contains the scope definition.

### Evidence Path Not Writable

**Error**: `evidence path not writable: .sdp/evidence`

**Solution**: Check directory permissions or create manually:
```bash
mkdir -p .sdp/evidence && chmod 755 .sdp/evidence
```

## References

- [OSS Combine Quickstart](../quickstart/OSS_COMBINE.md)
- [Adapter SDK](../protocol/CONTRACTS.md)
- [F069-01: OSS Bootstrap](../workstreams/backlog/00-069-01.md)
- [F068-02: Adapter SDK](../workstreams/backlog/00-068-02.md)
