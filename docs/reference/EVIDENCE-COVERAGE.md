# Evidence Coverage Matrix (F056)

Skill → event types emitted. Use for pipeline verification and `sdp log show --type=X`.

---

## Evidence Tracking Policy (WS-067-07: AC4)

### Tracked in Git (Source of Truth)

| Path | Purpose | Why Tracked |
|------|---------|-------------|
| `.sdp/log/events.jsonl` | Evidence event log | Audit trail for AI decisions |
| `.sdp/config.yml` | Project configuration | Required for reproducibility |
| `.sdp/guard-rules.yml` | Quality gate rules | Required for CI/local consistency |
| `.beads/issues.jsonl` | Task tracking | Session persistence |

### Not Tracked (Generated/Runtime)

| Pattern | Why Not Tracked |
|---------|-----------------|
| `*.out` | Build/test coverage artifacts |
| `.sdp/memory.db` | SQLite index (rebuildable) |
| `.sdp/checkpoints/` | Runtime state (resumable) |
| `coverage.html` | Generated report |
| `bin/`, `dist/` | Compiled binaries |

### Merge Strategy (.gitattributes)

```gitattributes
# Union merge for concurrent appends
.sdp/log/events.jsonl merge=union

# Beads custom merge
.beads/issues.jsonl merge=beads
```

---

## Pipeline chain

### Modern Intent Pipeline
`@build --mode idea → @build --mode feature → @review → @operate --mode deploy`

### Legacy Skill Pipeline (historical context)
`idea → deploy`

| Phase   | Skill     | Event type(s)     | CLI / trigger                          | Modern Intent Equivalent        |
|---------|-----------|-------------------|----------------------------------------|---------------------------------|
| Idea    | @idea     | plan              | `sdp idea record`, `sdp parse` (per WS) | `@build --mode idea`            |
| Design  | @design   | plan              | `sdp design record`, `sdp parse`       | `@build --mode idea`            |
| Build   | @build    | generation        | TDD runner (F054)                       | `@build --mode feature`         |
| Review  | @review   | verification      | `sdp verify` (per gate)                 | `@review` (unchanged)           |
| Deploy  | @deploy   | approval          | `sdp deploy --target main`              | `@operate --mode deploy`        |

## Skill × event types

| Skill      | plan | generation | verification | approval | Notes                          | Intent Equivalent              |
|------------|------|------------|--------------|----------|--------------------------------|--------------------------------|
| @vision    | ✓    |            |              |          | `sdp skill record --skill vision --type plan` (legacy, use @understand) | N/A (understand skill)          |
| @reality   |      |            | ✓            |          | `sdp skill record --skill reality --type verification` (legacy, use @review) | N/A (review skill)              |
| @idea      | ✓    |            |              |          | `sdp idea record` (legacy, use @build --mode idea) | `@build --mode idea`            |
| @design    | ✓    |            |              |          | `sdp design record`, `sdp parse` (legacy, use @build --mode idea) | `@build --mode idea`            |
| @build     |      | ✓          |              |          | Evidence layer (F054)          | `@build --mode feature`         |
| @review    |      |            | ✓            |          | `sdp verify` (per gate)       | `@review` (unchanged)           |
| @deploy    |      |            |              | ✓        | `sdp deploy` (legacy, use @operate) | `@operate --mode deploy`        |
| @oneshot   | ✓    |            |              | ✓        | `sdp orchestrate` (legacy, use @build --mode prototype) | `@build --mode prototype` or `@operate --mode plan` |
| @prototype |      | ✓          |              |          | `sdp prototype` (after WS gen) | `@build --mode prototype`        |
| @hotfix    |      | ✓          |              | ✓        | `sdp skill record` (2 calls) (legacy, use @fix) | `@fix --mode quick`             |
| @bugfix    |      | ✓          | ✓            |          | `sdp skill record` (gen + verification) (legacy, use @fix) | `@fix --mode systematic`        |
| @issue     | ✓    |            |              |          | `sdp skill record --skill issue --type plan` (legacy, use @fix) | `@fix --mode systematic`        |
| @debug     |      |            | ✓            |          | `sdp skill record --skill debug --type verification` (legacy, use @fix) | `@fix --mode systematic`        |

## Commands

- **Show recent events:** `sdp log show` (last 20)
- **Filter by type:** `sdp log show --type=plan` (or `generation`, `verification`, `approval`, `decision`, `lesson`)
- **Trace by commit/WS:** `sdp log trace [commit-sha]` or `sdp log trace --ws 00-056-01`
- **Chain integrity:** `sdp log trace --verify`

## Schema

Event types: `plan`, `generation`, `verification`, `approval`, `decision`, `lesson`.  
Schema: [schema/evidence.schema.json](../../schema/evidence.schema.json).
