# Beads SDP Requirements (Private)

Status: baseline v2
Scope: SDP integration with Beads for autonomous task tracking and repo snapshot sync

## 1. Branch + snapshot contract

**Requirement:** All clones and agents must use a predictable branch and a predictable repo snapshot flow.

- **sdp_lab:** `.beads/config.yaml` sets `sync-branch: "dev"` and worker manifests default `SDP_REPO_BRANCH=dev`
- **sdp:** `.beads/config.yaml` sets `sync-branch: "main"` and protocol PRs still target `main`
- **Active workflow (bd >= 0.59):** run `./scripts/beads_import_only.sh` after git sync and `./scripts/beads_export.sh` before commit/push
- **Compatibility:** the helper scripts may call `bd sync` only when an older Beads CLI still provides it
- **Tracked state:** `.beads/issues.jsonl` remains the shared repo snapshot; the local Dolt database is hydrated from and exported back to that snapshot

**Operator usage:** `SDP_REPO_BRANCH` in manifests should match the repo branch you expect agents to rebase onto before running `./scripts/beads_import_only.sh`.

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
- `Claim(id string) error` — `bd update <id> --status in_progress`
- `Close(id string, reason string) error` — `bd close <id> --reason "..."`
- `Sync(importOnly bool) error` — repo snapshot import/export (`./scripts/beads_import_only.sh` / `./scripts/beads_export.sh` in active workflows, legacy `bd sync` fallback where still needed)
- `Create(opts CreateOpts) (string, error)` — `bd create` with typed options

**Rationale:** Decouples SDP from bd CLI output format; enables testing with mock; prepares for future Beads API if upstream adds one.

---

## 6. Preflight sequence in orchestrate

**Requirement:** Before dispatching a task, the pod must have the latest Beads state.

**Sequence:**
1. `git fetch origin $SDP_REPO_BRANCH` + `git rebase FETCH_HEAD`
2. `./scripts/beads_import_only.sh` — hydrate the local Dolt-backed Beads DB from the tracked repo snapshot

**Implementation:** `scripts/orchestrate_k8s_issue.sh` already rebases onto `$SDP_REPO_BRANCH` and then runs `./scripts/beads_import_only.sh` in the worker pod.

---

## 7. Upstream PR candidates summary

| Candidate | Description | Priority |
|-----------|-------------|----------|
| repo snapshot helpers | Keep `beads_import_only` / `beads_export` flow documented across repo workflows | High |
| bd ready --filter | Extend filters (workstream, spec_id) for scheduler | Medium |
| --no-daemon batch | Document headless/scheduler usage; ensure no-daemon path is stable | Medium |
| Hooks for claim/close | Callbacks for SDP trace, evidence injection | Low (future) |

---

## 8. References

- [BEADS_AUTONOMY_SPEC.md](BEADS_AUTONOMY_SPEC.md) — fields, labels, transitions, evidence
- [.beads/config.yaml](../.beads/config.yaml) — repo config
- [scripts/beads_import_only.sh](../scripts/beads_import_only.sh) — snapshot import helper
- [scripts/beads_export.sh](../scripts/beads_export.sh) — snapshot export helper
- [internal/beads/client.go](../internal/beads/client.go) — Go wrapper client
