# @feature Skill — Full Product Discovery Redesign

> **Status:** Research complete
> **Date:** 2026-02-23
> **Goal:** Transform `@feature` from a thin orchestrator into a full product discovery instrument — with roadmap pre-check, product research loop, UX research, impact analysis, and industry-standard feature brief generation.

---

## Overview

### Goals

1. **Validate before specifying** — answer "should we build this?" before "how should we build this?"
2. **Context-aware depth** — simple features stay lightweight; novel/ambiguous ones get full discovery treatment
3. **UX-first workstreams** — UX findings become structured acceptance criteria, not prose no one reads
4. **Roadmap integrity** — detect duplicates and downstream impacts before committing to a spec

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Research loop structure | Dual-mode with signal-check routing (Obvious / Competitive / Novel) |
| UX research | New `@ux` skill with typed output schema, auto-triggered for user-facing features |
| Skill architecture | New `@discovery` skill; `@idea` and `@design` unchanged |
| Roadmap pre-check | `sdp memory search` after quick interview (Step 1.5) |
| Impact analysis | File-level grep of Scope Files after `@design` (Step 3.5) |

---

## 1. Skill Architecture — What Changes Where

> **Experts:** Sam Newman, Martin Fowler, Andrej Karpathy

### Solution

Create one new skill (`@discovery`) and one new skill (`@ux`). All existing skills remain **unchanged** and fully backward-compatible.

```
@feature (orchestrator — updated)
    │
    ├── @discovery (NEW — pre-requirements gate)
    │     Phase 1: Roadmap pre-check
    │     Phase 2: Signal check → route (Obvious / Competitive / Novel)
    │     Phase 3: Product research loop (web search + @think expert agents)
    │     Phase 4: Industry-standard feature brief
    │     Output: docs/drafts/discovery-{slug}.md
    │
    ├── @idea (UNCHANGED — receives discovery output as --spec)
    │     Skips cycles already answered by @discovery
    │
    ├── @ux (NEW — optional, auto-triggered for user-facing features)
    │     Phase 1: Listening session (6 UX questions)
    │     Phase 2: Autonomous research (codebase scan + pattern check)
    │     Output: docs/ux/{feature}.md (typed structured schema)
    │
    └── @design (UNCHANGED — reads @ux output when present)
          Step 3.5: Impact analysis (NEW — post-design gate)
```

### Backward Compatibility Matrix

| Invocation | Behavior |
|---|---|
| `@feature "auth"` | Full journey: @discovery → @idea → @ux → @design + impact scan |
| `@feature "auth" --quick` | Skips @discovery: @idea → @design (current behavior preserved) |
| `@idea "auth"` | Unchanged — standalone, 12–27 questions, no research loop |
| `@discovery "auth"` | Standalone pre-check; produces discovery brief, stops |
| `@discovery "auth" --skip-research` | Phase 1+2 only: roadmap check + signal check |
| `@ux {feature-id}` | Standalone UX research for any existing feature |
| `@design sdp-xxx` | Unchanged — standalone |

### What Changes in Each File

| File | Change |
|---|---|
| `@feature` (v8.0.0) | Add Step 0 invoking `@discovery`; pass output to `@idea --spec`; add `--quick` flag to skip; add Step 3.5 impact analysis; update CLAUDE.md skill table |
| `@idea` (unchanged) | `--spec path` mode already defined — @discovery output feeds directly in |
| `@design` (unchanged) | Reads `docs/ux/{feature}.md` when present; Step 3.5 lives in @feature, not @design |
| `@discovery` | **New skill** ~250 lines — 4 phases described below |
| `@ux` | **New skill** ~200 lines — 2 phases described below |
| `CLAUDE.md` | Add `@discovery` and `@ux` rows to skill table; update decision tree |

---

## 2. Product Research Loop — Dual-Mode Discovery

> **Experts:** Teresa Torres, Marty Cagan, Sam Newman, Theo Browne

### Solution

`@discovery` Phase 2–3 runs a **signal check** that routes to one of three tracks. The cost of discovery is proportional to the ambiguity of the feature — simple features don't get penalized.

#### Signal Check (~30 seconds, always runs)

1. "What user problem does this solve and for whom?"
2. "Do you know of existing solutions (libraries, tools, competitors)?"
3. `[Web search]` `"{feature_name} existing solutions 2026"`

**Routing:**

| Condition | Track |
|---|---|
| User answers both questions confidently AND search finds ≥1 clear prior art | **OBVIOUS** → `@idea --quiet` |
| User answers but search shows competitive landscape | **COMPETITIVE** → single research pass |
| User uncertain on Q1 OR search shows no clear prior art | **NOVEL** → discovery loop (max 3 iterations) |

Soft override: `"You're on the Obvious track. Type 'research' to switch to Competitive."` 

#### OBVIOUS Track

Skips to `@idea --quiet`. Duration: ~5 minutes total.

#### COMPETITIVE Track (single research pass)

```
Web searches to run:
  1. "{feature_name} best practices {year}"
  2. "{feature_category} open source alternatives"
  3. "how does [top competitor] implement {feature_name}"

Synthesize:
  - Alternatives comparison table (≥3 alternatives)
  - Build-vs-adopt recommendation with rationale
  - Primary differentiator in one sentence

Ask 3 targeted questions:
  1. Differentiation (what makes yours different?)
  2. Constraints (what rules out the adopt option?)
  3. Must-haves vs nice-to-haves

Convergence criteria (all 3 must be met):
  ✓ ≥3 alternatives identified
  ✓ Build-vs-adopt decision stated
  ✓ Primary differentiator articulated in one sentence
```

#### NOVEL Track (iterative loop, max 3 iterations)

Each iteration targets one of Cagan's four risks:

| Iteration | Expert Role | Risk | Web Search Focus |
|---|---|---|---|
| 1 | Product PM | Value risk — is this a real problem worth solving? | user pain points, demand signals |
| 2 | Tech Lead | Feasibility risk — can we build this well? | technical patterns, implementation complexity |
| 3 | DevRel/Strategist | Strategic fit risk — does this belong in the product? | roadmap alignment, user segment fit |

Per iteration: form hypothesis → 1-2 web searches → simulate expert → ask user ONE clarifying question → update risk score.

**JTBD Convergence:** Loop stops when the user can articulate the feature in Jobs-to-be-Done format: `"When [situation], I want to [motivation], so I can [outcome]"` AND all 3 risk scores ≥ 3/5 (total ≥ 9/15).

#### Phase 4: Industry-Standard Feature Brief

`@discovery` generates `docs/drafts/discovery-{slug}.md` with:

```
## Feature Brief: {Name}

### Opportunity Statement
When [situation], [user segment] want to [motivation], so they can [outcome].

### Market Context
- Existing alternatives: [table]
- Build rationale: [why build vs adopt]
- Differentiation: [one sentence]

### Validated Assumptions
- Value risk: [score]/5 — [evidence]
- Feasibility risk: [score]/5 — [evidence]
- Strategic fit: [score]/5 — [evidence]

### Open Questions
- [Q1]: [answer or "unresolved"]

### Research Context (for @idea)
- Alternatives: [list]
- Key constraints: [list]
- Pre-answered cycles: [Vision ✓, Problem ✓, ...]
```

---

## 3. UX Research Component — `@ux` Skill

> **Experts:** Indi Young, Don Norman, Theo Browne

### Solution

New `@ux` skill with two phases. Auto-triggered by `@feature` when @idea output contains user-facing terminology. Produces `docs/ux/{feature}.md` with a typed schema that `@design` explicitly consumes.

**The handoff contract problem:** UX insights in prose markdown get ignored by subsequent agents. The only way UX findings influence workstreams is if they appear as named, structured fields.

#### Phase 1: Listening Session (6 Questions)

These are NOT "what UI do you want?" questions. They are mental model elicitation:

1. **Context of reach:** `"What is the user doing in the 10 minutes before they encounter this feature? What problem are they mid-solving?"`
2. **Mental model gap:** `"What will the user *think* happens when they perform the primary action? Where does that model likely diverge from what the system actually does?"`  *(Don Norman: Gulf of Execution)*
3. **Workaround reality:** `"What do users do today without this feature? The workaround reveals the existing mental model."`
4. **Friction prediction:** `"At which step will most users pause, hesitate, or abandon? What makes that moment hard?"`
5. **Thinking style spectrum:** `"Who is the cautious user who double-checks everything vs. the fast mover who skips instructions? Does the design need to serve both?"` *(Indi Young: thinking styles)*
6. **Accessibility context:** `"Who might be excluded by the obvious implementation? (screen reader, keyboard-only, low bandwidth, cognitive load under stress)"`

#### Phase 2: Autonomous Codebase Research

The AI exploits its codebase access (which no human UX researcher has):

- Scan for existing features with similar user-visible surfaces → find established patterns to follow
- Check for existing accessibility patterns in the codebase
- Cross-reference stated pain points against current error handling → flag "user sees generic error when X happens"
- Flag technical decisions in @idea's output that will create Gulf of Execution/Evaluation problems
- Generate "UX Risk Register": a ranked list of user-visible failure modes

#### Output Schema: `docs/ux/{feature}.md`

```yaml
user_context: "[description of the moment the user reaches for this feature]"
mental_model_gap: "[where user belief ≠ system reality]"
friction_points:
  - step: "[step name]"
    risk: high|medium|low
    description: "[what makes this moment hard]"
    recommendation: "[design mitigation]"
accessibility_notes:
  - "[specific exclusion risk and mitigation]"
thinking_styles:
  cautious_user: "[how design must accommodate them]"
  fast_user: "[how design must accommodate them]"
ux_risks:
  - "[ranked list of user-visible failure modes]"
validated_workaround: "[what users do today]"
```

`@design` reads this file when present and converts `friction_points` and `ux_risks` into acceptance criteria in the relevant workstream files.

**Auto-trigger heuristic** (conservative — avoid noise on infra features):
- Present: `ui`, `user`, `interface`, `dashboard`, `form`, `flow`, `UX`, `screen`, `page`, `button` in @idea output
- Absent: `K8s`, `CRD`, `reconciler`, `stream`, `JetStream`, `CLI-only` (explicit infra signals)
- Always skipped for `@feature "..." --infra`

---

## 4. Roadmap Integration & Impact Analysis

> **Experts:** Marty Cagan, Teresa Torres, Martin Kleppmann, Sam Newman

### Solution

Two-pass, evidence-grounded approach using existing `sdp memory` infrastructure.

#### Step 1.5: Roadmap Pre-Check (inserted after @feature quick interview)

```markdown
After quick interview, extract 3-5 high-signal keywords (nouns + domain verbs).

Run:
  sdp memory stats  # warn if index > 24h old
  sdp memory search "<keyword1> <keyword2> <keyword3>"

Analyze results for:
  - Features in ROADMAP.md covering same domain terms
  - Workstream files with matching Scope Files or goals
  - Existing docs/drafts/idea-*.md that cover similar territory

Present Overlap Report (HIGH and MEDIUM confidence only; log LOW to file):
  "Found N potentially related items:
   [HIGH] F005 Rework Loop — covers [summary]. Similarity reason: [1 sentence]
   [MEDIUM] 00-008-02 — touches [same module]. Overlap type: [data model / API / user flow]"

User resolution (single question):
  A) These are different — proceed with @discovery
  B) This extends F005 — incorporate and modify existing workstream
  C) This supersedes F005 — flag for later review (propose: set F005 status to 'deferred')
  D) Show me more detail before deciding

Gate: Proceed to @discovery only after user resolves.
```

**False positive control:** Search keywords must be specific nouns and domain verbs — NOT generic terms like "add", "update", "implement". If memory search returns > 10 results, reduce keywords to 2 most specific terms.

#### Step 3.5: Impact Analysis (inserted after @design creates workstreams)

```markdown
After @design creates workstream files, read their Scope Files sections.

Run for each new Scope File:
  grep -rl "<scope_file>" docs/workstreams/backlog/*.md
  
Also run:
  sdp memory search "<new feature domain terms>"
  sdp drift detect  # surface already-drifted items

Categorize matches:

  [FILE CONFLICT] Same file scoped by multiple workstreams
    → Recommendation: sequence dependencies; add depends_on field
    
  [DATA BOUNDARY] New feature modifies a type used by another feature
    → Recommendation: extend existing schema workstream OR create schema-first workstream
    
  [DEPENDENCY CHAIN] New feature inserts into existing F00X → F00Y dependency path
    → Recommendation: show updated dependency graph, get confirmation
    
  [PRIORITY SHIFT] New feature is P0 but depends on P2 items blocking other P0 work
    → Recommendation: reprioritize or create prerequisite workstream

Present Impact Report — user must acknowledge before @oneshot:
  "[HIGH] FILE CONFLICT: 00-004-01 and new 00-0XX-01 both scope internal/adapter/agentrun_reconciler.go
   → Proposed: add depends_on: ["00-004-01"] to new workstream
   
   [LOW] No other conflicts found."

For any resolved conflicts: automatically update workstream frontmatter
(add depends_on, related_to, or update status per user decision).
```

---

## Implementation Plan

### Phase 1: Foundation (implement first)

- [ ] Design `docs/drafts/discovery-{slug}.md` output schema
- [ ] Create `@discovery` skill v1.0.0 (Phases 1–4, all three routing tracks)
- [ ] Update `@feature` to invoke `@discovery` as Step 0; pass output to `@idea --spec`; add `--quick` bypass flag
- [ ] Add Step 1.5 Roadmap Pre-Check to `@feature`
- [ ] Add Step 3.5 Impact Analysis to `@feature`
- [ ] Update `CLAUDE.md` skill table and decision tree

### Phase 2: UX Layer

- [ ] Design `docs/ux/{feature}.md` output schema
- [ ] Create `@ux` skill v1.0.0 (6-question listening session + autonomous codebase research)
- [ ] Update `@design` to read `docs/ux/{feature}.md` when present
- [ ] Update `@feature` to auto-trigger `@ux` based on @idea output heuristic
- [ ] Add `--infra` flag to @feature to bypass @ux

### Phase 3: Hardening

- [ ] Test all backward-compatible invocation paths
- [ ] Tune signal-check routing keywords for SDP-specific context
- [ ] Add `@discovery` and `@ux` to `sdp/prompts/skills/` and update `sdp skill list`
- [ ] Write acceptance test scenarios: `@feature "auth"`, `@feature "K8s reconciler"`, `@feature "auth" --quick`

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Duplicate feature rate | Unknown (blind) | 0 duplicates past pre-check |
| Discovery → workstream time | ~20 min | ≤30 min (Competitive), ≤45 min (Novel) |
| UX findings in workstream ACs | 0% (never happens) | ≥80% of user-facing features |
| Impact conflicts caught pre-merge | ~0 (caught at PR) | ≥90% caught at Step 3.5 |
| Feature brief quality | No standard | Passes JTBD format + 4-risk assessment |

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Web search in @discovery returns noise | Provide specific query templates in skill file; require structured synthesis, not raw paste |
| Routing false negatives (developer claims "obvious" when it's novel) | Soft override always available; signal check is non-blocking advisory |
| @ux adds too many friction points for infra teams | `--infra` flag + conservative auto-trigger heuristic |
| memory index staleness breaks pre-check | Warn if index > 24h old; document `sdp memory index` in `@feature` Step 1.5 |
| Research context bloat entering @idea | Structured `research_context` object with named slots; @idea pre-answers covered cycles rather than re-asking |
| @discovery + @feature version skew | Document dependency in skill frontmatter; `@feature` v8 requires `@discovery` v1 |
