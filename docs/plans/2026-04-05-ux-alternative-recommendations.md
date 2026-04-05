# UX Alternative Recommendations

**Date:** 2026-04-05
**Status:** Additive review, does not replace existing UX documents
**Related:**
- [UX Audit Results](2026-04-05-ux-audit-results.md)
- [UX Improvement Proposals](2026-04-05-ux-improvement-proposals.md)
- [UX Improvement Specifications](2026-04-05-ux-improvement-specs.md)
- [Canonical Happy Path](../reference/canonical-happy-path.md)
- [Public Quick Start](../../sdp/docs/QUICKSTART.md)
- [Public Protocol Overview](../../sdp/docs/PROTOCOL.md)

---

## 1. Why This Document Exists

The current UX package identifies many real friction points.

The gap is not effort.
The gap is framing.

The existing documents mostly optimize prompt-surface friction:

- too much reading
- too many questions in `@feature`
- too many agents in `@review`
- too much harness inconsistency

That is directionally useful, but incomplete.

The deeper problem is product split-brain:

- public onboarding tells a `Local Mode` CLI-first story
- prompt docs tell a `@feature -> @oneshot -> @review -> @deploy` story
- canonical internal docs tell a stage-first model with two modes
- `sdp_lab` adds a stronger operator and orchestration model on top

Until SDP states one honest default product path, UX work will keep improving local surfaces while preserving global confusion.

---

## 2. Core Position

### Main thesis

SDP's biggest UX problem is not "too many docs."

It is that users cannot tell which product they are using:

1. local single-repo assistant
2. prompt bundle for IDEs
3. queue-backed operator system
4. private lab orchestration stack

The next UX iteration should start by fixing product truth, not by polishing each surface in isolation.

### Product decision that should drive the UX package

Public SDP should present one default story:

- **Default public on-ramp:** `Local Mode`
- **Advanced operating model:** `Operator Mode`
- **Skills and CLI:** two control surfaces for the same stage model
- **Harnesses:** adapters with explicit support levels, not equal products

This is already consistent with the current canonical references.
The UX package should adopt that model explicitly instead of continuing to audit and plan around a prompt-first baseline.

---

## 3. What The Current UX Package Gets Right

The current documents correctly identify several real problems:

- progressive disclosure is weak
- brownfield adoption is underdesigned
- harness support is uneven
- recovery paths are underspecified
- naming around `deploy` is misleading
- orchestrator visibility needs improvement

These should stay.

The issue is priority and framing, not the existence of the findings.

---

## 4. What Is Missing

### 4.1 Missing: one explicit product truth

The current documents do not start from the canonical stage model.
They start from command and harness friction.

What is missing:

- one statement of the default public path
- one statement of what is advanced vs default
- one statement of which surfaces are canonical vs convenience
- one statement of what "supported" means for each harness

Without this, every future doc can still tell a different story.

### 4.2 Missing: inventory of what already ships

The package treats several discoverability problems as missing capabilities.
That is risky.

Public SDP already has:

- `sdp demo`
- `sdp plan`
- `sdp apply`
- `sdp status --text`
- `sdp next`
- `sdp init --skip-beads`
- `sdp hooks uninstall`
- `sdp checkpoint resume`
- `sdp session repair`

This means some proposed fixes should be framed as:

- expose better
- align docs
- unify terminology
- improve post-command guidance

Not:

- invent a new adjacent workflow

### 4.3 Missing: stage x mode x surface matrix

The UX package reviews happy paths, harnesses, and orchestrator separately.
That is not enough.

It needs one matrix:

| Stage | Local Mode | Operator Mode | CLI | Skills | Board |
|---|---|---|---|---|---|
| bootstrap | yes | yes | primary | optional | no |
| intake | optional | required | partial | partial | primary |
| shaping | yes | yes | yes | yes | visibility |
| execution | yes | yes | yes | yes | visibility |
| findings | local | queue-backed | yes | yes | primary in operator mode |
| delivery | local approval | PR + queue + QA/UAT | yes | yes | visibility |

Without this matrix, the team keeps fixing one dimension while breaking another.

### 4.4 Missing: honest support policy

The current parity proposal implies that all four harnesses should feel equally first-class soon.
That is expensive and not necessary.

What is missing:

- `recommended`
- `supported`
- `compatible`
- `experimental`

Example:

- Claude Code: recommended
- OpenCode: supported
- Cursor: compatible
- Codex: compatible

That is a stronger UX move than pretending parity already exists.

### 4.5 Missing: risk-based review model

The current proposal scales `@review` by diff size alone.
That is too weak.

What is missing is review selection by change risk:

- auth and identity
- payments
- migrations
- secrets
- CI and deployment
- concurrency and state
- public APIs

A 20-line schema migration can be riskier than a 200-line refactor.

### 4.6 Missing: artifact lifecycle

The package says little about stale planning artifacts.

What is missing:

- when an idea draft is authoritative
- when UX notes are stale
- when a feature plan is superseded
- how a user restarts without accumulating junk
- how checkpoints, evidence, and workstreams relate over time

Without an artifact lifecycle, progressive disclosure turns into progressive burial.

### 4.7 Missing: trust and consent UX

Brownfield adoption is not only about gates.
It is also about trust.

What is missing:

- install preview
- exact file-change preview before merge into existing IDE config
- explicit backup and restore messaging
- clear explanation when a guard blocks an action
- clear "what happened" and "how to undo it"

If SDP changes a repo aggressively, users will stop trusting it before they ever reach value.

### 4.8 Missing: telemetry and rollout

The package proposes large changes without a release measurement plan.

It should tie every UX change to measurable outcomes:

- time to first value
- first successful plan/apply/build
- recovery success rate
- brownfield setup completion rate
- harness-specific completion rate
- support burden per harness

There is already baseline material in the repo.
The UX package should extend it, not ignore it.

### 4.9 Missing: narrower brownfield wedge

The brownfield spec is directionally right but too large for a first delivery slice.

What is missing is an MVP adoption wedge:

1. safe install
2. no-destructive merge behavior
3. no mandatory Beads
4. no blocking quality gates at start
5. clear next step
6. clear rollback

Tracker adapters, multi-language depth, and full graduation logic can follow later.

### 4.10 Missing: contributor UX separated from user UX

The audit mixes:

- public adopter UX
- daily operator UX
- `sdp_lab` contributor DX

These matter, but they are not the same product problem.

The package should distinguish:

- **product UX** for adopters
- **operator UX** for queue-backed delivery
- **maintainer DX** for SDP contributors

If not, internal pain will keep hijacking public onboarding priority.

---

## 5. What Should Be Added To Each Existing Document

### 5.1 Additions to `ux-audit-results`

Add a short section before the detailed findings:

#### A. Current canonical model

State:

- `Local Mode` is the public default
- `Operator Mode` is advanced
- CLI and skills are parallel control surfaces, not separate workflows

#### B. Shipped vs missing

Add a table:

| Capability | Shipped | Discoverable | Consistent docs | Notes |
|---|---|---|---|---|
| `sdp demo` | yes | partial | partial | hidden from some prompt docs |
| `sdp status --text` | yes | yes | inconsistent in audit narrative | avoid claiming JSON-only globally |
| `hooks uninstall` | yes | low | low | recovery surface exists |
| `checkpoint resume` | yes | medium | fragmented | wording inconsistent |

#### C. Support-level view

Add a table for harness promise:

| Harness | Current promise | Real support level | Recommended action |
|---|---|---|---|
| Claude | first-class | recommended | keep |
| OpenCode | partial | supported/experimental | tighten docs |
| Cursor | thin | compatible | be honest |
| Codex | thin | compatible | be honest |

### 5.2 Additions to `ux-improvement-proposals`

Add a new first proposal ahead of the current P0 set:

#### Proposal 0: Canonical Product Contract

Intent:

- define one honest default path
- align CLI, skills, onboarding, and operator docs to one stage model
- explicitly mark advanced surfaces as advanced

Deliverables:

- one source-of-truth table for stages and control surfaces
- one support policy for harnesses
- one "default public path" statement reused everywhere

Also add:

- a separate proposal for trust/consent UX
- a separate proposal for telemetry and rollout gating

### 5.3 Additions to `ux-improvement-specs`

Add three missing specs:

#### SPEC-00: Canonical Surface Contract

Scope:

- align `QUICKSTART`, `PROTOCOL`, harness guides, and prompt docs
- define default public path
- define advanced operator path
- define command naming and stage mapping

#### SPEC-06: Trust, Preview, and Safe Recovery

Scope:

- install preview
- config merge preview
- clear backup/restore flow
- guard block messages with fix path
- restart/reset/archive rules for planning artifacts

#### SPEC-07: UX Telemetry and Rollout Gates

Scope:

- define metrics
- capture baselines
- gate releases on selected UX KPIs
- compare Local Mode vs Operator Mode completion
- compare harness completion and support load

---

## 6. Recommended Reprioritization

The current package overweights parity and broad brownfield architecture too early.

Recommended order:

### P0

1. **Canonical Product Contract**
   - one default public story
   - one stage model
   - one support policy

2. **Brownfield MVP**
   - safe install
   - non-destructive merge
   - no mandatory Beads
   - no blocking gates on day one
   - clear uninstall/rollback

3. **Unified status and recovery**
   - next-step guidance
   - resume guidance
   - repair guidance
   - human-readable state across CLI and prompt surfaces

### P1

4. **Progressive disclosure refinement**
   - shorter top-level docs
   - adaptive planning depth
   - clearer workstream outputs

5. **Risk-based review routing**
   - diff size plus risk profile
   - not diff size alone

### P2

6. **Harness parity expansion**
   - after support policy is explicit
   - after Local Mode default path is strong
   - after brownfield MVP works

This order is more pragmatic because it improves user truth before adapter coverage.

---

## 7. Suggested Practical Delivery Sequence

### Phase 1: Make the product honest

Ship:

- aligned `QUICKSTART`
- aligned `PROTOCOL`
- aligned harness onboarding pages
- one support-level table
- one default-path diagram

Success condition:

- a new user can answer "What is the default way to use SDP?" in one minute

### Phase 2: Make adoption safe

Ship:

- preview install changes
- merge instead of overwrite
- backup and restore path
- `skip Beads` path that stays coherent
- uninstall and rollback docs

Success condition:

- a brownfield repo can try SDP without fear of config loss

### Phase 3: Make recovery obvious

Ship:

- unified resume language
- unified repair language
- next-step hints after every important command
- explicit stale-state messaging

Success condition:

- recovery no longer depends on reading historical docs

### Phase 4: Optimize interaction depth

Ship:

- adaptive `@feature`
- risk-based `@review`
- better artifact freshness checks

Success condition:

- daily operators see less ceremony without losing control

### Phase 5: Expand harness support honestly

Ship:

- capability manifests if still needed
- harness-specific prompts
- fallback paths where they are worth maintaining

Success condition:

- each harness has a clear promise and a reliable minimum experience

---

## 8. Concrete Acceptance Criteria That Should Exist But Do Not Yet

The current specs need outcome-level criteria, not only structure-level criteria.

Recommended additions:

- A first-time public user reaches `sdp demo` success in under 5 minutes.
- A brownfield user can run `sdp init` without losing existing IDE config.
- A user can tell whether Beads is required for their chosen mode before install completes.
- After any failed command, SDP provides one explicit next command to try.
- After any blocking guard action, SDP explains why it blocked and how to proceed safely.
- A user can see whether a harness is recommended, supported, compatible, or experimental before choosing it.
- Planning artifacts older than the current accepted feature state are visibly marked stale or superseded.

---

## 9. Recommended Document Outcome

The current UX package should not be discarded.

It should be extended with:

1. one product-truth layer
2. one support-policy layer
3. one telemetry layer
4. one safer brownfield MVP definition
5. one artifact lifecycle definition

That would turn the current package from a useful surface audit into a coherent product UX strategy.

---

## 10. Final Recommendation

Do not treat the next UX cycle as "make prompts shorter and adapters more equal."

Treat it as:

1. make the product story honest
2. make adoption safe
3. make recovery obvious
4. reduce ceremony where it hurts
5. expand parity only after the default path is strong

That ordering is better for UX, better for trust, and cheaper to ship.
