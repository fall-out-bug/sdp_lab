# Beads SDP Requirements (Private)

Status: baseline v1
Scope: SDP integration with Beads for autonomous task tracking and sync

## 1. Sync branch

**Requirement:** All clones and agents must use a predictable branch for Beads commits.

- **Config:** `.beads/config.yaml` sets `sync-branch: <branch>` (e.g. `beads-sync` or `main`)
- **Env override:** `BEADS_SYNC_BRANCH` or `BD_SYNC_BRANCH` for local override
- **Behavior:** `bd sync` commits `.beads/issues.jsonl` and `.beads/metadata.json` to this branch
- **SDP usage:** `SDP_REPO_BRANCH` in worker manifests should align with sync-branch when agents run `bd sync`

**Upstream PR candidate:** Document `sync-branch` as first-class config; ensure `bd sync --import-only` respects remote branch for pull-before-merge semantics.

---

## 2. Batch API for scheduler

**Requirement:** Scheduler needs to fetch ready tasks without daemon dependency and with minimal latency.

- **Current:** `bd ready --label autonomy --json` returns JSON array of ready issues
- **Gap:** No explicit batch mode; daemon may add latency; `--no-daemon` forces direct DB access
- **Desired:** `bd ready --json --no-daemon` or equivalent for programmatic use in scheduler loops
- **Filter:** `--label autonomy`, `--label strict-evidence`, optional `--label workstream:X`

**Upstream PR candidate:** Add `bd ready --filter workstream:X` or extend `--label` to support workstream prefix. Document `--no-daemon` for headless/scheduler use.

---

## 3. Workstream as first-class label

**Requirement:** Filter tasks by workstream for orchestrator dispatch.

- **Current:** Labels like `workstream:handoff-validation`, `workstream:generic` are custom
- **Gap:** `bd ready --label workstream:generic` works but workstream is not a reserved concept in Beads
- **Desired:** Either keep as label convention, or add `--workstream X` filter for clarity

**Upstream PR candidate:** Optional `--workstream` flag as sugar over `--label workstream:X`. Low priority; label-based filter is sufficient.

---

## 4. Spec ID and acceptance in create

**Requirement:** `bd create` must support `--spec-id` and `--description` / `--acceptance` for SDP task creation.

- **Current:** `bd create "title" -t task -p 1 --spec-id "path" --description "..." --labels "autonomy,strict-evidence,..." --json`
- **Status:** Supported; document in BEADS_AUTONOMY_SPEC

---

## 5. SDP adapter (Go wrapper)

**Implementation:** `internal/beads/adapter.go` provides typed Go API over `bd` CLI:

- `Ready(labels []string, limit int) ([]Issue, error)` — `bd ready --label X --json`
- `Show(id string) (*Issue, error)` — `bd show <id> --json`
- `Claim(id string) error` — `bd update <id> --claim`
- `Close(id string, reason string) error` — `bd close <id> --reason "..."`
- `Sync(importOnly bool) error` — `bd sync` or `bd sync --import-only`
- `Create(opts CreateOpts) (string, error)` — `bd create` with typed options

**Rationale:** Decouples SDP from bd CLI output format; enables testing with mock; prepares for future Beads API if upstream adds one.

---

## 6. Typed findings contract

**Requirement:** Findings from review, CI, `drift`, and `QA/UAT` must re-enter SDP as typed `beads issue` entries.

- **Contract:** `docs/protocol/BEADS_FINDINGS_CONTRACT.md`
- **Required fields:** `source`, linked `feature`, linked `workstream`, `blocking`, finding title/description, priority, `PR` or artifact reference
- **Canonical mapping:** issue fields carry summary and priority; labels carry finding source, linked `feature`, linked `workstream`, and blocking semantics; notes are supplemental only
- **Behavior:** blocking findings must return to the ready queue and block merge-ready state until resolved
- **Scope:** applies to `@review`, `@qa`, PR gates, and any automated findings import path

**Upstream PR candidate:** Support stable metadata fields or labels for typed findings so SDP does not rely on free-form notes for `source`, linkage, and blocking semantics.

---

## 7. Preflight sequence in orchestrate

**Requirement:** Before dispatching a task, the pod must have the latest Beads state.

**Sequence:**
1. `git pull origin $SDP_REPO_BRANCH` (or `git fetch` + `git rebase FETCH_HEAD` to avoid multi-branch rebase issues)
2. `bd sync --import-only` — import remote JSONL if newer

**Implementation:** `scripts/orchestrate_k8s_issue.sh` preflight and `cmd/opencode-agent` syncWorkspace already perform git sync; ensure `bd sync --import-only` runs after git pull in orchestrate preflight.

---

## 8. Upstream PR candidates summary

| Candidate | Description | Priority |
|-----------|-------------|----------|
| sync-branch | First-class config; document import-only semantics | High |
| bd ready --filter | Extend filters (workstream, spec_id) for scheduler | Medium |
| --no-daemon batch | Document headless/scheduler usage; ensure no-daemon path is stable | Medium |
| Hooks for claim/close | Callbacks for SDP trace, evidence injection | Low (future) |

---

## 9. References

- [BEADS_AUTONOMY_SPEC.md](BEADS_AUTONOMY_SPEC.md) — fields, labels, transitions, evidence
- [protocol/BEADS_FINDINGS_CONTRACT.md](protocol/BEADS_FINDINGS_CONTRACT.md) — typed findings contract
- [.beads/config.yaml](../.beads/config.yaml) — repo config
- [internal/beads/client.go](../internal/beads/client.go) — Go wrapper client
