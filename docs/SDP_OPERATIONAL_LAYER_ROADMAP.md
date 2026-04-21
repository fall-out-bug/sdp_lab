# SDP Operational Layer Roadmap

Status: working roadmap
Date: 2026-03-22
Scope: consolidate what already exists in SDP, define next operational-layer moves, and separate OSS core from market moat

## Goal

Avoid two failure modes:
- pretending SDP has nothing and rebuilding from scratch
- dumping everything into one undifferentiated pile

This roadmap starts from current reality in `sdp_lab`, then defines:
1. what already exists
2. what should be unified next
3. what belongs in OSS core
4. what should remain part of the market moat

---

## 1. What already exists in SDP

SDP is not starting from zero. It already contains a substantial amount of operational structure.

### A. Contracts and schemas already exist
Examples already present:
- `schema/evidence-envelope.schema.json`
- `schema/handoff-analyst.schema.json`
- `schema/handoff-coder.schema.json`
- `schema/handoff-reviewer.schema.json`
- `schema/contracts/runtime-decision.schema.json`
- `schema/contracts/orchestration-event.schema.json`
- `schema/contracts/beads-queue-view.schema.json`
- `schema/contracts/feature-card.schema.json`
- `schema/contracts/project-board-snapshot.schema.json`
- `schema/contracts/portfolio-board-snapshot.schema.json`

### B. Templates already exist
Examples already present:
- `docs/templates/task-brief.template.md`
- `docs/templates/implementation-plan.template.md`
- `docs/templates/verification-note.template.md`
- `docs/templates/review-note.template.md`
- `docs/templates/handoff-note.template.md`
- `internal/workstream/template.go`
- `internal/profile/config_templates.go`
- `templates/`

### C. Doctor / validation lineage already exists
Signals already present:
- `docs/plans/2026-03-08-sdp-cli-control-tower-pack.md`
- `docs/reviews/sdp-cli-ux-exploration-report.md`
- `docs/reference/runbooks.md` references to `sdp doctor`
- `internal/evidence/schema_test.go`
- `hooks/validate-artifacts.sh`
- validation scripts under `scripts/`

### D. Evidence / provenance / artifact discipline already exists
Examples already present:
- `docs/AGENT_ARTIFACT_COMMUNICATION_PROTOCOL.md`
- `docs/ARTIFACT_PROVENANCE_INTAKE.md`
- `docs/ARTIFACT_PROVENANCE_HASH_CHAIN_CONTRACT.md`
- `docs/OBSERVABILITY_METRICS_TRACE_SCHEMA_INTAKE.md`
- `docs/attestation/coding-workflow-v1.md`
- `specs/strict-evidence-template.json`

### E. Control tower foundation now exists
Recent additions already present:
- `FeatureCard` contract + schema
- project/portfolio board snapshots
- control-store skeleton in `internal/control`
- card lifecycle actions
- Beads bridge (`card-execute`)
- feedback/resume flow
- feedback export/import
- message-correlation bridge
- shared canon (`CONTROL_TOWER_CANON.md`)
- session start canon (`docs/SESSION_START_CANON.md`)

### Key reality check
The right move now is not to invent the whole operational layer from zero.
The right move is to **unify and sharpen** what already exists.

---

## 2. What should be unified next

These are the operational-layer pieces that exist partially or implicitly, but are not yet crisp enough.

### A. `sdp doctor control`
Need a doctor pass specifically for control-tower hygiene.

Examples of checks:
- cards without intake artifacts
- ready cards missing ready-gate fields
- executing cards without `linked_beads_ids`
- needs_input cards without feedback/decision payload
- snapshot drift from card state

### B. Stage packs
Need to package process guidance by stage rather than leaving it scattered across docs.

Current v1 packs now present under `packs/`:
- `packs/intake/PACK.md`
- `packs/shaping/PACK.md`
- `packs/execution-bridge/PACK.md`
- `packs/feedback-loop/PACK.md`

Those packs should be treated as the official thin stage-oriented guide for the currently implemented control-store slice.
They make the lifecycle discoverable without redefining the architecture.

Recommended pack set over time:
- intake pack
- shaping pack
- execution bridge pack
- feedback loop pack
- verification/review pack
- release pack
- retro pack

The current slice should stay thin: practical operator/agent guidance first, richer templates and automation later.

### C. Template/generator discipline
Need stronger separation between:
- canonical source templates
- generated operational outputs
- user-facing/operator-facing renderings

### D. Launch scaffolding
Need standard launch briefs/handoff scaffolding so implementation sessions start from the same canon by default.

### E. Process hygiene telemetry
Need lightweight health views for:
- feedback debt
- blocked debt
- release debt
- review debt
- stale cards/tasks
- trace completeness

---

## 3. OSS core vs market moat

This split should be explicit.

---

## 3.1 What should be in SDP OSS core

These are the things that make SDP genuinely useful to others and strengthen adoption.

### A. Core contracts and schemas
Good OSS candidates:
- FeatureCard schema
- ProjectBoardSnapshot schema
- PortfolioBoardSnapshot schema
- evidence envelope schemas
- handoff schemas
- runtime/orchestration contracts where they are generic

### B. Artifact taxonomy and basic templates
Good OSS candidates:
- task-brief
- implementation-plan
- verification-note
- review-note
- handoff-note
- release-note / migration-note templates

### C. Basic control-store engine
Good OSS candidates:
- file-backed FeatureCard store
- snapshot derivation
- lifecycle basics
- ready gate
- feedback/resume basics
- basic Beads bridge

### D. Doctor / validation core
Good OSS candidates:
- schema validation
- artifact linkage checks
- control-store consistency checks
- snapshot derivation checks
- trace hygiene basics

### E. Session-start and launch canon
Good OSS candidates:
- session-start canon
- launch brief scaffolding
- anti-confusion rules
- practical implementation-slice discipline

### F. Public runbooks and integration patterns
Good OSS candidates:
- Beads integration pattern at the generic level
- Gastown-inspired dashboard/read-model pattern at the generic level
- provider-agnostic feedback packet patterns

---

## 3.2 What should remain market moat / private advantage

These are the things that create differentiated operator leverage and should not be dumped wholesale into OSS.

### A. Cross-layer orchestration doctrine
Keep private:
- the most effective orchestration policies
- default automation heuristics
- escalation thresholds
- human/admin interruption strategy
- priority arbitration rules

### B. Portfolio/operator intelligence
Keep private:
- cross-project prioritization logic
- operator control heuristics
- enterprise/portfolio attention routing
- organization-specific exception handling logic

### C. Advanced stage packs
Keep private or partially export only:
- high-performance multi-agent stage packs
- proprietary execution/review/release orchestration patterns
- premium launch packs and role overlays

### D. Rich process telemetry and scoring
Keep private:
- maturity scoring
- debt scoring
- orchestration quality scoring
- agent/operator performance heuristics
- premium review/release risk analytics

### E. Private commercial/enterprise overlays
Keep private:
- internal boundary maps
- private export/redaction rules
- enterprise roadmap specifics
- premium operator workflows

---

## 4. Recommended next implementation sequence

### Phase 1 — unify the control doctor layer
Build first:
- `sdp doctor control`
- minimal control-store hygiene checks
- snapshot consistency checks

### Phase 2 — formalize stage packs
Status: first lightweight slice implemented.

Start with lightweight packs for:
- intake
- shaping
- feedback loop
- execution bridge

Current outputs:
- `packs/README.md`
- `packs/intake/PACK.md`
- `packs/shaping/PACK.md`
- `packs/execution-bridge/PACK.md`
- `packs/feedback-loop/PACK.md`

Next polish for this phase should focus on doc wiring, template discipline, keeping the packs discoverable from session-start docs, and keeping template references honest against the actual repo.

### Phase 3 — strengthen generator discipline
Move toward:
- canonical templates
- generated operator outputs
- generated launch briefs

### Phase 4 — add process hygiene telemetry
Build thin operator health surfaces over:
- feedback debt
- blocked debt
- stale cards
- missing trace/evidence links

---

## 5. Immediate recommendation

The best immediate implementation slice is:

### `sdp doctor control`

Why:
- it builds on things already present
- it improves confidence in the current control tower immediately
- it is valuable both for OSS core and internal leverage
- it gives SDP a more gstack-like operational sharpness without cargo culting gstack

---

## 6. Short formula

### Already in SDP
- contracts
- templates
- evidence discipline
- validation lineage
- control-tower skeleton

### Next to unify
- doctor control
- stage packs
- generators
- launch scaffolding
- hygiene telemetry

### OSS core
- the engine, contracts, templates, validation basics

### Market moat
- advanced orchestration doctrine, operator intelligence, premium process telemetry, and high-leverage execution/review/release strategy
