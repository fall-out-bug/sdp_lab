# Harness Contract Gates Runbook

Status: active
Audience: operators, tech leads, agent platform owners

## 1. Why this exists

This runbook enforces feature execution quality at harness level and prevents silent drift of:

- acceptance criteria
- required metrics
- evidence completeness
- quality gates

The implementation is policy-driven and does not rely on agent self-discipline.

## 1.1 Where it works

This does not magically enforce in repositories where it is not installed.

Enforcement works only in repos that:

- run `sdp-guard --check-contract` in CI,
- store contract/snapshot artifacts in `.sdp/contracts/`,
- make `protocol-compliance` a required PR check.

For another repository, copy this workflow pattern and guard command integration into that repo's CI.

## 2. Artifacts

- Contract schema: `specs/runtime/schemas/feature-task-contract.schema.json`
- Contract example: `specs/examples/feature-task-contract.example.json`
- Snapshot example: `specs/examples/feature-task-snapshot.example.json`
- Clarification example: `specs/examples/clarification-change.example.json`
- Guard command: `cmd/sdp-guard/main.go`

## 3. Core commands

### 3.1 Contract compliance check

```bash
sdp-guard \
  --check-contract \
  --contract .sdp/contracts/feature-F123.json \
  --snapshot .sdp/contracts/feature-F123.snapshot.json
```

Exit codes:

- `0`: all required gates pass
- `1`: blocked by drift or failed gate
- `2`: command or file error

### 3.2 Clarification classification

```bash
sdp-guard \
  --classify-clarification \
  --contract .sdp/contracts/feature-F123.json \
  --clarification .sdp/contracts/feature-F123.change.json
```

Chat text classification (no structured JSON required):

```bash
sdp-guard \
  --classify-clarification \
  --clarification-text "добавь метрику latency и усили quality gate"
```

Classifications:

- `no_impact`
- `additive`
- `reductive`
- `policy_sensitive`

### 3.3 Apply clarification

Additive clarification:

```bash
sdp-guard \
  --apply-clarification \
  --contract .sdp/contracts/feature-F123.json \
  --clarification .sdp/contracts/feature-F123.change.json
```

Reductive or policy-sensitive clarification (approval required):

```bash
sdp-guard \
  --apply-clarification \
  --contract .sdp/contracts/feature-F123.json \
  --clarification .sdp/contracts/feature-F123.change.json \
  --approved-by "tech-lead"
```

## 4. User clarification UX

When user sends clarification in chat, harness flow is:

1. Convert clarification to structured change JSON.
2. Run `--classify-clarification`.
3. If `additive`: apply automatically and bump contract version.
4. If `reductive` or `policy_sensitive`: block until approved.
5. Store applied clarification in `change_requests` audit trail.

## 5. Blocking behavior

Contract check blocks transition when at least one gate returns `block`.

Implemented hard gates:

- `requirement_integrity`
- `evidence`
- `metric_parity`
- `quality`
- `process`

Example blocked output:

```json
{
  "blocked": true,
  "gate_results": [
    {
      "gate_id": "metric_parity",
      "status": "block",
      "violations": [
        {
          "type": "metric_drop",
          "field": "required_metrics",
          "message": "required metric \"gate_pass_rate\" is missing"
        }
      ]
    }
  ]
}
```

## 6. CI integration

Add a required CI job before merge:

```bash
sdp-guard \
  --check-contract \
  --contract .sdp/contracts/feature-${FEATURE_ID}.json \
  --snapshot .sdp/contracts/feature-${FEATURE_ID}.snapshot.json
```

Recommended PR summary fields:

- `AC coverage`
- `Metric parity`
- `Evidence completeness`
- `Gate summary`

## 7. Operational notes

- Keep contract immutable except via clarification flow.
- Treat reductive changes as change requests, not direct edits.
- Store snapshots at each phase transition for drift forensics.
