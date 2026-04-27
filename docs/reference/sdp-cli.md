# SDP CLI Reference

> **Generated from**: `internal/cli/registry.go` + `internal/cli/commands.go`  
> **Feature**: F137-05 (CLI reference + parity gate)  
> **Last updated**: 2026-04-27

This document is the canonical reference for all `sdp` subcommands. It mirrors the registry state in `internal/cli/`. The doc-sync parity rule (`sdp-doc-sync`) checks for drift between this file and the registered commands.

---

## Table of Contents

- [Card commands](#card-commands)
- [Board commands](#board-commands)
- [Doctor commands](#doctor-commands)
- [Dispatch commands](#dispatch-commands)
- [Result commands](#result-commands)
- [Orchestrate commands](#orchestrate-commands)
- [Query commands](#query-commands-require-beadsdual-mode)
- [Deploy commands](#deploy-commands)
- [Discovery commands](#discovery-commands-stage-0)
- [Pipeline commands](#pipeline-commands)
- [Phase commands](#phase-commands)
- [Scout commands](#scout-commands)
- [Spec commands](#spec-commands)
- [Index commands](#index-commands)
- [Bootstrap commands](#bootstrap-commands)
- [Rules commands](#rules-commands)
- [Build commands](#build-commands)
- [Reset commands](#reset-commands)
- [Coverage commands](#coverage-commands)
- [Skills commands](#skills-commands)
- [Legacy shims (deprecated)](#legacy-shims-deprecated)

---

## Card commands

### `sdp card`

Manage feature cards through their lifecycle.

```
sdp card <create|show|clarify|needs-input|ready|park|execute|heartbeat|feedback|feedback-export|message-export|resume|resume-import|reply-ingest|deliver>
```

**Subcommands:** `create`, `show`, `clarify`, `needs-input`, `ready`, `park`, `execute`, `heartbeat`, `feedback`, `feedback-export`, `message-export`, `resume`, `resume-import`, `reply-ingest`, `deliver`

**Examples:**
```bash
sdp card create --project myproject --title 'Add feature' --raw 'description'
sdp card show --project myproject --id card-123
sdp card ready --project myproject --id card-123
```

---

## Board commands

### `sdp board`

Manage kanban boards for projects.

```
sdp board <build|show>
```

**Subcommands:** `build`, `show`

**Examples:**
```bash
sdp board build --project myproject
sdp board show --project myproject
```

---

## Doctor commands

### `sdp doctor`

Diagnose issues with SDP control state, adapters, and backlog.

```
sdp doctor <control|adapters|backlog|all>
```

**Subcommands:** `control`, `adapters`, `backlog`, `all`

**Examples:**
```bash
sdp doctor control
sdp doctor adapters --strict
sdp doctor backlog
```

---

## Dispatch commands

### `sdp dispatch`

Dispatch cards for execution.

```
sdp dispatch <card|next>
```

**Subcommands:** `card`, `next`

**Examples:**
```bash
sdp dispatch card --project myproject --id card-123
sdp dispatch next
```

---

## Result commands

### `sdp result`

Ingest and manage execution results.

```
sdp result ingest
```

**Subcommands:** `ingest`

**Examples:**
```bash
sdp result ingest --file result.json
```

---

## Orchestrate commands

### `sdp orchestrate`

Run orchestration loop for result processing.

```
sdp orchestrate once
```

**Subcommands:** `once`

**Examples:**
```bash
sdp orchestrate once
```

---

## Query commands (require beads/dual mode)

### `sdp why`

Show why a card is blocked.

```
sdp why <card-id>
```

**Examples:**
```bash
sdp why card-123
```

### `sdp next`

Show next actionable items.

```
sdp next [--limit N]
```

**Examples:**
```bash
sdp next
sdp next --limit 20
```

### `sdp missing`

Show items lacking evidence.

```
sdp missing [project-id]
```

**Examples:**
```bash
sdp missing
sdp missing myproject
```

### `sdp approve`

Resolve a human gate.

```
sdp approve <card-id>
```

**Examples:**
```bash
sdp approve card-123
```

### `sdp trace`

Show full feature trace with evidence.

```
sdp trace <card-id>
```

**Examples:**
```bash
sdp trace card-123
```

---

## Deploy commands

### `sdp deploy`

Manage deployments to staging and production.

```
sdp deploy <staging|prod|rollback>
```

**Subcommands:** `staging`, `prod`, `rollback`

**Examples:**
```bash
sdp deploy staging /path/to/project
sdp deploy prod v1.2.3 /path/to/project
sdp deploy rollback v1.2.2 /path/to/project
```

---

## Discovery commands (Stage 0)

### `sdp discover`

Run discovery pipeline (FRAME + SCAN + checkpoint).

```
sdp discover "raw idea"
```

**Examples:**
```bash
sdp discover "Add user authentication"
```

---

## Pipeline commands

### `sdp intent`

Create intake card from raw intent.

```
sdp intent "description"
```

**Examples:**
```bash
sdp intent "I need to add OAuth support"
```

### `sdp status`

Show card status and phase.

```
sdp status <card-id>
```

### `sdp stuck`

Show stuck/long-running cards.

```
sdp stuck
```

### `sdp eval`

Run build evaluation manually.

```
sdp eval <card-id>
```

### `sdp clarify`

Run clarification manually.

```
sdp clarify <card-id>
```

### `sdp plan`

Show plan for a card.

```
sdp plan <card-id>
```

### `sdp approve-plan`

Approve a pending plan.

```
sdp approve-plan <card-id>
```

### `sdp attention`

Show cards requiring attention.

```
sdp attention
```

> **Ownership note (F125):** `intent`, `status`, `stuck`, `eval`, `clarify`, `plan`, `approve-plan`, `discover` — F125 owns the behavioral semantics. F137 registers these for discovery only; functional changes belong in F125.

---

## Phase commands

### `sdp phase`

Run phase-specific operations (F134 owns phase semantics).

```
sdp phase <plan|review|eval>
```

**Subcommands:** `plan`, `review`, `eval`

**Examples:**
```bash
sdp phase plan --feature-id F042
sdp phase review --ws-id 00-042-01
sdp phase eval --run-id run-123
```

> **Ownership note (F134):** F134 owns phase semantics and runtime. F137 registers `sdp phase` for discovery; phase behavior stays in F134.

---

## Scout commands

### `sdp scout`

Scout repository for code patterns.

```
sdp scout [--format json|text|card] [--output DIR] <repo-path>
```

**Examples:**
```bash
sdp scout /path/to/repo
sdp scout --format json --output ./results /path/to/repo
```

---

## Spec commands

### `sdp spec`

Extract specifications from repository.

```
sdp spec [--format json|text] [--category api|rules|invariants|sla] [--output DIR] [--enrich] [--diff] <repo-path>
```

**Examples:**
```bash
sdp spec /path/to/repo
sdp spec --category api --format json /path/to/repo
```

---

## Index commands

### `sdp index`

Build and query repository indexes.

```
sdp index <build|stats|manifest>
```

**Subcommands:** `build`, `stats`, `manifest`

**Examples:**
```bash
sdp index build /path/to/repo
sdp index stats /path/to/repo
sdp index manifest --output ./docs /path/to/repo
```

---

## Bootstrap commands

### `sdp bootstrap`

Initialize repository with SDP workstreams.

```
sdp bootstrap [--dry-run] [--force] [--beads] [--yes] [--auto-curate] [--only TYPES] <repo-path>
```

**Examples:**
```bash
sdp bootstrap /path/to/repo
sdp bootstrap --dry-run --only feature,epic /path/to/repo
```

---

## Rules commands

### `sdp rules`

Update constraint rules from evidence sources.

```
sdp rules update <repo-path> [--source-evidence=<dir>] [--manifest=<file>] [--format json|text]
```

**Examples:**
```bash
sdp rules update /path/to/repo
sdp rules update --source-evidence ./evidence /path/to/repo
```

---

## Build commands

### `sdp build`

Run build pipeline for a feature idea.

```
sdp build "<idea>" [--strict] [--local] [--sandbox=<type>] [--dry-run] [--format json|text] [--output DIR] [--timeout DURATION]
```

**Examples:**
```bash
sdp build "Add user authentication"
sdp build --strict --format json "Add OAuth support"
```

---

## Reset commands

### `sdp reset`

Reset checkpoint for a feature.

```
sdp reset --feature F042 [--dry-run] [--yes]
```

**Examples:**
```bash
sdp reset --feature F042 --dry-run
```

---

## Coverage commands

### `sdp coverage-scan`

Scan code coverage against thresholds.

```
sdp coverage-scan [--path DIR] [--threshold PCT] [--format text|json] [--skip-test] [--package PATTERN] [--coverprofile FILE]
```

**Examples:**
```bash
sdp coverage-scan --threshold 80
sdp coverage-scan --format json --package ./internal/...
```

---

## Skills commands

### `sdp skills`

Manage and augment skills.

```
sdp skills <augment|update>
```

**Subcommands:** `augment`, `update`

**Examples:**
```bash
sdp skills augment --stack config.json
sdp skills update --project-root /path/to/project
```

---

## Legacy shims (deprecated)

The following standalone binaries forward to `sdp <subcommand>` with a deprecation warning. They remain callable during the migration grace period. See [`deprecation-implementation-guide.md`](deprecation-implementation-guide.md) and [`docs/reference/cmd-inventory.md`](cmd-inventory.md) for the full disposition matrix.

| Legacy binary | Replacement | Removal target |
|---|---|---|
| `sdp-beads-bridge` | `sdp beads` | v2.0.0 |
| `sdp-ci-loop` | `sdp ci-loop` | v2.0.0 |
| `sdp-dispatch` | `sdp dispatch profile` | v2.0.0 |
| `sdp-doc-sync` | `sdp doc-sync` | v2.0.0 |
| `sdp-eval` | `sdp skill-eval` | v2.0.0 |
| `sdp-evidence` | `sdp evidence` | v2.0.0 |
| `sdp-gh-findings-sync` | `sdp gh-sync` | v2.0.0 |
| `sdp-guard` | `sdp guard` | v2.0.0 |
| `sdp-harness` | `sdp harness` | v2.0.0 |
| `sdp-healthcheck` | `sdp healthcheck` | v2.0.0 |
| `sdp-omc-guard` | `sdp omc-guard` | v2.0.0 |
| `sdp-orchestrate` | `sdp orchestrate daemon` | v2.0.0 |
| `sdp-orchestrate-daemon` | `sdp orchestrate daemon` | v2.0.0 |
| `sdp-protocol-check` | `sdp protocol-check` | v2.0.0 |
| `sdp-ready` | `sdp ready` | v2.0.0 |
| `sdp-session-audit` | `sdp session-audit` | v2.0.0 |
| `sdp-strataudit` | `sdp strataudit` | v2.0.0 |
| `sdp-up` | `sdp up` | v2.0.0 |
| `sdp-ws-verdict-validate` | `sdp ws-validate` | v2.0.0 |

**`sdp-control`** — marked `retire` (emits deprecation warning, no replacement; legacy control CLI).

**Out of scope (infrastructure services, not CLI commands):**
- `sdp-a2a` — HTTP server (F126 territory)
- `sdp-mcp` — MCP server (F126 territory)
