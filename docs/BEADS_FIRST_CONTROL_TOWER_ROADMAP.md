# Beads-First Control Tower Roadmap

Status: proposed execution roadmap
Date: 2026-03-24
Owner: Андрей + Клавдий
Context: think-tank synthesis, Beads-first architectural pivot
Related:
- `SDP_SPEC_DRIVEN_PIPELINE_CANON.md`
- `CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md`
- `DISPATCH_BRIDGE_SPEC.md`
- `docs/roadmap/ROADMAP.md`

---

## Executive Position

SDP не должен развивать параллельный control-state store рядом с Beads.

**Целевой инвариант:**
- **Beads** = единственный durable execution graph / operational source of truth
- **SDP Control Tower** = orchestration + policy + views + derived artifacts
- **TaskContract / provenance / evidence** = semantic and verification layer поверх Beads
- **opencode** = execution runtime
- **A2A** = transport/API surface

Иными словами:

> **Beads stores work. SDP defines meaning, constraints, and trust.**

---

## The Problem We Are Solving

Сейчас в SDP существует split-brain:
- `FeatureCard` и `.sdp/control/*.yaml|json` живут как самостоятельный state layer
- Beads уже живёт как dependency-aware durable graph
- orchestrate/runtime state живёт ещё в своих артефактах
- board/snapshot — ещё одна проекция

Это допустимо как промежуточная стадия, но не как целевая архитектура.

Если оставить так дальше, будут хронические проблемы:
- `ready` / `blocked` / `in_progress` расходятся между слоями
- orchestration policy дублирует Beads semantics
- сложно объяснять, что является truth
- труднее строить A2A, federation, governance и dogfooding

---

## Architectural End-State

### Truth hierarchy

1. **Constitution / Policy docs** — правила системы
2. **TaskContract artifacts** — semantic contract of execution
3. **Beads graph** — operational lifecycle truth
4. **Artifacts / evidence / provenance records** — verifiable reality
5. **Snapshots / boards / executive views** — derived representations only

### What lives where

#### Beads owns
- work item identity
- status
- priority
- dependencies
- readiness / blockers
- gates
- claim / assignment semantics
- parent-child graph
- labels
- machine metadata

#### SDP owns
- intent normalization
- clarification semantics
- contract generation
- provenance chain
- evidence requirements
- compliance evaluation
- routing policy
- human/admin UX
- views and summaries

#### opencode owns
- actual execution
- code changes
- tool use
- runtime logs
- local produced artifacts

#### A2A owns
- external task API
- transport contract
- authn/authz at API boundary

---

## Artifact Decomposition

Ниже — как раскладывать roadmap **по SDP артефактам**, а не по абстрактным фазам.

---

## Artifact A — Constitution Layer

### Files
- `docs/constitution.yaml`
- `docs/specs/project-registry.yaml`
- ADRs in `docs/decisions/`

### Role
Определяет, какие constraints нельзя нарушать без явного architectural decision.

### Work to do
1. Зафиксировать Beads-first как explicit decision:
   - `ADR: Beads as operational source of truth`
2. Зафиксировать слой границ:
   - Beads vs SDP vs opencode vs A2A vs OpenClaw
3. Зафиксировать migration invariants:
   - no shadow lifecycle state
   - no control-owned ready queue
   - no rewriting checkpoint loop during storage migration

### Deliverable
Новый ADR + обновлённая constitution section.

---

## Artifact B — Feature / Intent Spec Layer

### Current form
- `FeatureCard`
- intake artifacts in `.sdp/control/...`

### Target form
Feature intent остаётся как semantic artifact, но перестаёт быть самостоятельным durable lifecycle store.

### Decision
**FeatureCard stays as schema, not as competing source of truth.**

### Work to do
1. Определить canonical mapping `FeatureCard -> Beads Issue + Metadata + Contract artifact`
2. Разделить fields на:
   - typed Beads fields
   - `Issue.Metadata.sdp.*`
   - external artifact payload (`contract.json`, `intake.md`, etc.)
3. Упростить FeatureCard model до:
   - intake/spec semantics
   - links to bead id / contract id / provenance id
   - no independent lifecycle engine

### Deliverable
- `docs/FEATURECARD_BEADS_MAPPING.md` (new)
- updated `internal/control` type comments / docs

---

## Artifact C — Beads Mapping / Operational Graph

### Role
This is the heart of the pivot.

### Work to do
1. Ввести canonical SDP metadata schema for Beads issues:

```json
{
  "sdp": {
    "card_id": "F069",
    "phase": "clarify|spec|build|verify|review|release",
    "contract": {"id": "CTR-001", "hash": "sha256:..."},
    "executor": {"role": "omo-implementation", "session_id": "...", "state": "running"},
    "review": {"state": "pending", "attempts": 0},
    "delivery": {"target": "staging", "state": "pending", "rollback_count": 0},
    "provenance": {"packet_hash": "...", "prompt_hash": "..."}
  }
}
```

2. Define issue/gate taxonomy for SDP:
- feature
- clarify
- contract
- review
- qa
- release
- gate:human
- gate:ci
- gate:pr
- gate:timer

3. Decide what remains files vs what becomes Beads-native:

**Keep as files/artifacts:**
- snapshots
- dispatch packets
- result packets
- provenance files
- evidence files
- intake markdown

**Move to Beads-backed truth:**
- readiness
- blockers
- current work status
- assignment / claim
- follow-up routing references

### Deliverable
- `docs/BEADS_SDP_SCHEMA.md` (new)
- maybe `docs/BEADS_GATE_MODEL.md` (new)

---

## Artifact D — Repository / Storage Boundary

### Problem
Нельзя просто импортнуть `github.com/steveyegge/beads/internal/storage` из SDP.

### Immediate practical approach
Сначала вычленить persistence boundary внутри SDP, не дожидаясь идеального API в Beads.

### Work to do
1. Add repository abstraction inside `internal/control`:
- `CardRepository`
- `ArtifactStore`
- maybe `ProjectionStore`

2. Implement:
- `FileCardRepository` (current behavior preserved)
- `BeadsCardRepository` (later)

3. Keep `control.Store` public API stable.

### Deliverable
- `internal/control/repository.go`
- `internal/control/repo_file.go`
- `internal/control/artifacts.go`
- optional placeholder `internal/control/repo_beads.go`

### Exit criterion
Current tests pass while persistence is abstracted.

---

## Artifact E — Dispatch / Execution Bridge

### Role
Translate Beads-backed ready work into opencode execution.

### Current state
P0-P5 bridge work already exists. Good.

### Next move
Repoint dispatch semantics from control-owned ready logic to Beads-first ready/claim logic.

### Work to do
1. Canonicalize dispatch cycle:
- query ready work
- policy rank
- claim
- build packet
- execute
- ingest result

2. Separate concerns:
- policy ranking = SDP
- ready semantics = Beads
- execution = opencode
- result ingestion = SDP

3. Add claim/lease semantics if Beads lacks them directly.

### Deliverable
- `docs/BEADS_DISPATCH_MODEL.md` (new)
- code changes in `internal/executor/bridge.go`, `internal/executor/loop.go`

---

## Artifact F — Contract Layer

### Role
Semantic source of truth for what “done” means.

### Work to do
1. Keep `TaskContract` as first-class artifact
2. Make contract generation explicit from feature/spec state
3. Link contract to Beads issue via metadata + hash refs
4. Define revision strategy:
- additive changes
- reductive changes
- policy-sensitive changes

### Deliverable
- `docs/TASK_CONTRACT_LIFECYCLE.md` (new)
- refinement of existing contract docs

---

## Artifact G — Evidence / Provenance Layer

### Role
This is the trust moat.

### Work to do
1. Define canonical evidence package per execution type:
- implementation
- review
- qa
- release
2. Define provenance chain end-to-end:
- intake hash
- contract hash
- dispatch packet hash
- prompt hash
- artifact hash
- result packet hash
3. Define how refs are stored in Beads metadata vs file artifacts

### Deliverable
- `docs/EVIDENCE_PACKAGE_SPEC.md` (new)
- `docs/PROVENANCE_CHAIN_CANON.md` (new or update existing provenance docs)

---

## Artifact H — Views / Board / Executive Surface

### Role
Derived visibility only. Never source of truth.

### Work to do
1. Explicitly redefine board as projection over Beads + artifacts
2. Kill any logic in board/snapshot generation that invents lifecycle truth
3. Add human-useful queries:
- what is next?
- why blocked?
- what lacks evidence?
- what needs my approval?

### Deliverable
- `docs/CONTROL_TOWER_VIEW_MODEL.md` (new)
- later CLI work: `sdp why`, `sdp trace`, `sdp gates`, `sdp evidence`

---

## Implementation Roadmap

### Phase R0 — Decision & Canon Cleanup

**Goal:** freeze target architecture before coding more glue.

#### Tasks
- write ADR for Beads-first pivot
- update canon language: Beads = operational truth, FeatureCard = semantic artifact/projection
- rewrite `CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md` next phases around Beads-first

#### Exit
- architectural direction documented
- no ambiguity about source of truth

---

### Phase R1 — Storage Boundary Extraction

**Goal:** decouple `control.Store` from file persistence without changing behavior.

#### Tasks
- introduce repository interfaces
- move YAML persistence into `FileCardRepository`
- keep snapshots/artifacts outside repository
- keep tests green

#### Exit
- `control.Store` orchestrates, repository persists
- no user-visible change

---

### Phase R2 — Beads Mapping Canon

**Goal:** define exact data model before implementation.

#### Tasks
- write mapping doc for FeatureCard → Beads issue
- define metadata schema
- define gate taxonomy
- define what stays file-based

#### Exit
- model decisions explicit
- no hand-wavy “we’ll figure it out in code”

---

### Phase R3 — Beads Adapter MVP

**Goal:** make Beads-backed repository possible.

#### Options
- inspect Beads deeper and use available extension seam
- or patch Beads with public API / adapter package
- or temporarily use CLI-backed adapter only as transitional shim

#### Warning
CLI-backed full repository is not target architecture. Only bridge if needed.

#### Exit
- can create/load/update/list SDP-backed work via Beads adapter

---

### Phase R4 — Dual-Write / Shadow Read

**Goal:** migrate safely.

#### Tasks
- file repo remains primary
- beads repo shadow-writes
- compare snapshots / card loads / status projections
- surface mismatches

#### Exit
- equivalence confidence high enough to cut over

---

### Phase R5 — Cutover to Beads-First

**Goal:** switch operational truth to Beads.

#### Tasks
- read path flips to Beads-backed repository
- ready / blocked / next-action derived from Beads semantics
- control snapshots become projections only

#### Exit
- no shadow lifecycle decisions in file store

---

### Phase R6 — Contract / Evidence Tightening

**Goal:** make Beads-first stack trustworthy, not just functional.

#### Tasks
- contract refs wired into metadata
- evidence package refs wired into metadata
- provenance chain queryable
- failure / review / qa / release gates consistent

#### Exit
- end-to-end traceability real

---

### Phase R7 — Product Surface Cleanup

**Goal:** make the system usable by humans and agents.

#### Tasks
- board/view model update
- CLI explanations (`why blocked`, `what next`, `what missing`)
- A2A response model aligned with bead-backed task state

#### Exit
- operational clarity for humans and agents

---

## What We Explicitly Do NOT Do In This Roadmap

- do **not** rewrite the existing checkpoint/orchestrate loop during storage migration
- do **not** rebuild board/doctor/attention architecture from scratch
- do **not** invent a second task graph inside SDP
- do **not** push every artifact blob into Beads
- do **not** start federation/Wasteland work in v1
- do **not** collapse all lifecycle semantics into custom statuses when gates/metadata fit better

---

## Immediate Next 5 Concrete Outputs

1. `ADR: Beads-first operational source of truth`
2. `docs/FEATURECARD_BEADS_MAPPING.md`
3. `docs/BEADS_SDP_SCHEMA.md`
4. `internal/control/repository.go` + `repo_file.go`
5. updated `CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md`

---

## Bottom Line

Следующий правильный шаг — **не писать ещё кода в старую схему**, а сначала зафиксировать канон и decomposition по артефактам.

Кодовая формула простая:

> **First: decide what is truth.**  
> **Then: decide how that truth is represented.**  
> **Then: adapt code to that truth.**

Для SDP truth должен быть Beads-backed.
