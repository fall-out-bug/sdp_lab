# FeatureCard Storage & Layout Proposal

Status: working proposal
Date: 2026-03-22
Scope: first implementation shape for SDP control tower storage
Related:
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/BOARD_SNAPSHOT_CONTRACTS_WORKING_MODEL.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`
- `schema/contracts/feature-card.schema.json`
- `schema/contracts/project-board-snapshot.schema.json`
- `schema/contracts/portfolio-board-snapshot.schema.json`

## Goal

Choose a first storage/layout model for the SDP control tower that is:
- simple
- git-friendly
- transparent to inspect
- compatible with Beads and SDP artifacts
- easy to project into board snapshots

This proposal intentionally avoids premature database engineering.

---

## 1. Recommendation

### Use file-backed `FeatureCard` documents as the write model

Each feature card should live as its own file.

Recommended format:
- **YAML** for the canonical write model

Recommended reason:
- easy for humans to read/edit
- git-friendly
- maps naturally to the current contract/example docs
- easy to diff and review
- easy to derive snapshots from

### Use JSON for derived snapshots

Recommended format:
- **JSON** for `ProjectBoardSnapshot` and `PortfolioBoardSnapshot`

Reason:
- aligns with machine-readable schemas
- UI/CLI/web consumers can read directly
- explicit separation between write model and read models

---

## 2. Proposed directory layout

```text
.sdp/control/
  projects/
    <project_id>/
      cards/
        feature-<project>-YYYY-MM-DD-001.yaml
        feature-<project>-YYYY-MM-DD-002.yaml
      snapshots/
        board.json
      intake/
        feature-<project>-YYYY-MM-DD-001.md
  portfolio/
    snapshot.json
```

### Meaning

#### `cards/`
Canonical `FeatureCard` write model files.

#### `snapshots/board.json`
Derived `ProjectBoardSnapshot` for one project.

#### `intake/`
Early SDP intake artifacts or task briefs linked from cards.

#### `portfolio/snapshot.json`
Derived `PortfolioBoardSnapshot` for the whole system.

---

## 3. Why keep cards and intake artifacts near each other

Because SDP starts at intake.

The relationship should be obvious on disk:
- card exists
- intake artifact exists
- card links to that artifact

This keeps the trace spine inspectable even before execution tasks exist in Beads.

---

## 4. Card file naming

Recommended naming:

```text
feature-<project_id>-YYYY-MM-DD-NNN.yaml
```

Examples:
- `feature-openclaw-2026-03-22-001.yaml`
- `feature-opencode-2026-03-22-002.yaml`
- `feature-gastown-2026-03-22-001.yaml`

### Why this naming works
- stable and sortable
- human-readable
- consistent with schema pattern
- easy to correlate with intake artifacts

---

## 5. Intake artifact naming

Recommended naming:

```text
feature-<project_id>-YYYY-MM-DD-NNN.md
```

Stored under:

```text
.sdp/control/projects/<project_id>/intake/
```

This keeps the first SDP trace artifact aligned with the card ID.

---

## 6. Snapshot derivation model

### Project board snapshot
Derived from:
- all card files under one project
- optional Beads execution summary for linked issues

Output:
- `.sdp/control/projects/<project_id>/snapshots/board.json`

### Portfolio snapshot
Derived from:
- all project board snapshots, or directly from all cards
- optional aggregated execution summaries

Output:
- `.sdp/control/portfolio/snapshot.json`

### Important rule
Snapshots are **derived**, not edited directly.

---

## 7. Relationship to Beads

### What stays in Beads
- execution issues
- child tasks
- dependencies
- findings/rework tasks
- execution lifecycle state

### What stays in FeatureCard storage
- raw request
- shaping state
- clarification state
- intake artifact link
- board-level status
- bridge links to Beads

### Bridge rule
A `FeatureCard` may have empty `linked_beads_ids` until execution objects actually exist.

That is correct and expected.

---

## 8. Relationship to SDP artifacts

### At intake
- card points to an intake artifact in `intake/`

### As the feature matures
Card may also link to:
- implementation plan
- verification note
- review note
- handoff note
- release/migration notes

These can remain in other canonical SDP artifact directories if needed; the card only needs stable references.

---

## 9. Minimal implementation commands needed later

A future CLI/tool layer should minimally support:

### Card write operations
- create card
- update card status
- clarify card
- mark ready
- park card
- link beads issue
- attach artifact reference

### Snapshot operations
- rebuild project snapshot
- rebuild portfolio snapshot
- show project board
- show portfolio view

These commands should operate on files first.

---

## 10. Why not store raw intake directly in Beads first

Because Beads is the execution graph.

If we store every immature feature idea there immediately:
- raw inbox becomes noisy
- execution graph gets polluted with vague requests
- shaping state is harder to express cleanly
- it becomes harder to distinguish feature maturation from execution decomposition

So:
- FeatureCard storage first
- Beads bridge second

---

## 11. Why not start with SQLite first

SQLite may become useful later.
But for the first implementation, it is the wrong optimization target.

### Costs of starting with SQLite
- extra tooling and migration overhead
- less transparent for humans
- harder to inspect in git review
- pushes architecture toward a mini-platform too early

### Better first step
Use files until the pain is real.
Then migrate if needed.

---

## 12. Suggested migration path if files become insufficient later

If file-backed cards later become too limiting:

### Stage 1
Canonical files + generated snapshots

### Stage 2
Optional indexing layer (cache/db) for performance

### Stage 3
Promote DB/index as runtime acceleration layer while files remain source of truth

That path preserves transparency while allowing scale-up later.

---

## 13. Immediate recommendation

Implement this first:

```text
.sdp/control/
  projects/
    openclaw/
      cards/
      intake/
      snapshots/
    opencode/
      cards/
      intake/
      snapshots/
    beads/
      cards/
      intake/
      snapshots/
  portfolio/
```

Then wire:
1. create card
2. create intake artifact
3. rebuild snapshots
4. display via CLI or UI

That is enough to prove the model before building more machinery.

---

## 14. Short formula

- **YAML cards** = write model
- **Markdown intake artifacts** = SDP trace starting point
- **JSON snapshots** = read models
- **Beads** = execution graph

That is the recommended first storage layout.
