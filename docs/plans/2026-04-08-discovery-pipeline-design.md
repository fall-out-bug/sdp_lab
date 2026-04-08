# Discovery Pipeline — Design Document

**Date:** 2026-04-08  
**Status:** Validated — ready for prototyping  
**Stage:** Stage 0 in canonical SDP pipeline (precedes spec)

---

## 1. Problem Statement

Product/feature discovery is fragmented across incompatible tools. A developer with a raw idea must manually stitch together: problem framing, hypothesis formulation, market scanning, desk research validation, and experiment design — across 5–10 different tools with no shared state.

No existing tool covers the full pipeline end-to-end. The gap is confirmed by three independent research scans (web/product, OSS, academic) and validated by 8 LLM models in benchmark testing (all returned PIVOT reasoning that points to Claim 5 — hallucination in scan — as the critical risk, not the gap itself).

---

## 2. Solution: Automated Discovery Pipeline

A 4-phase agent-driven pipeline that takes a raw idea and produces a validated hypothesis with go/no-go verdict. Integrated into SDP as **Stage 0** — feeds directly into spec phase on GO verdict.

### Two entry points, one pipeline

```
Entry A: bd discover "raw idea"     → creates beads issue type:discovery
Entry B: beads issue type:discovery → agent auto-picks up
```

---

## 3. Pipeline Architecture

```
IDEA (free text)
    │
┌───▼──────────────────────────────────────────────────────┐
│  PHASE 1 · FRAME                                          │
│  Agent: JTBD canvas, problem reframe, appetite/scope      │
│  Out: problem_frame.md · beads issue created              │
└───┬──────────────────────────────────────────────────────┘
    │ CHECKPOINT A — typed clarification
    │ (missing_info / ambiguous_requirement / approach_choice)
┌───▼──────────────────────────────────────────────────────┐
│  PHASE 2 · HYPOTHESIZE                                    │
│  Agent: Strategyzer Test Card, RAT ranking, requirements  │
│  Out: hypothesis.md · beads issue updated                 │
└───┬──────────────────────────────────────────────────────┘
    │ CHECKPOINT B
┌───▼──────────────────────────────────────────────────────┐
│  PHASE 3 · SCAN                                           │
│  Agent: parallel MCP scan across 7 source types           │
│  GPT Researcher: get_research_context() only              │
│  ┌────────────────────────────────────────────────────┐   │
│  │  DEPTH SIGNAL MECHANISM                             │   │
│  │  coverage_score per result · 7 heuristics           │   │
│  │  Section A (settled) / Section B (flagged)          │   │
│  └────────────────────────────────────────────────────┘   │
│  Out: scan.md · landscape JSON · depth flags              │
└───┬──────────────────────────────────────────────────────┘
    │ CHECKPOINT C — two-section: settled + flagged items
    │ human: [D]eep-dive / [P]rovisional / [I]gnore per flag
┌───▼──────────────────────────────────────────────────────┐
│  PHASE 4a · DESK RESEARCH VALIDATION                      │
│  Agent: 3–5 claims → evidence for/against per claim       │
│  Verdict: supported / contradicted / insufficient_data    │
│  Out: validation.md · validation_brief JSON               │
└───┬──────────────────────────────────────────────────────┘
    │ if insufficient_data → Phase 4b
    │ CHECKPOINT D
┌───▼──────────────────────────────────────────────────────┐
│  PHASE 4b · EXPERIMENT (conditional)                      │
│  Agent: selects format from experiment matrix             │
│  Generates: experiment brief · dochild beads issue        │
│  Collects: Fireflies auto + manual bd update              │
│  Out: experiment.md → updates verdict                     │
└───┬──────────────────────────────────────────────────────┘
    │ FINAL VERDICT
    ├── GO    → feature card + beads issue type:feature
    ├── PIVOT → back to Phase 2 with new inputs
    └── KILL  → close issue, artifacts preserved as ADR
```

### Checkpoint model (async-aware)

```
human online?
  yes → chat dialogue, wait for response
  no  → update beads issue (notes/design) + git push + notify
        (Slack webhook / Gmail MCP)
        pipeline waits, resumes on approve
```

---

## 4. Depth Signal Mechanism

**Problem formalized from dogfooding:** DeerFlow (50K★) appeared in OSS scan with 3 sentences of description and received ADOPT/EXTRACT verdict. The rule that catches this:

> Any verdict ≠ IGNORE where `primary_source_read = false` → automatic depth flag.

### Coverage envelope (per scan result)

```json
{
  "disposition": "ADOPT|EXTRACT|INSPIRE|MONITOR|IGNORE",
  "disposition_confidence": 0.62,
  "coverage": {
    "primary_source_read": false,
    "architecture_reviewed": false,
    "description_sentences": 3,
    "multi_source_corroboration": false,
    "coverage_score": 0.18
  },
  "depth_flag": {
    "flagged": true,
    "reason": "high_importance_low_coverage",
    "recommended_action": "deep_dive",
    "blocking": false
  }
}
```

### 7 heuristics for auto-flagging

| # | Rule | Notes |
|---|---|---|
| H1 | Stars > 5K + < 5 sentences from primary source | Volume signal without substance |
| H2 | Mentioned in 3+ sources, none read directly | Cross-referencing ≠ primary reading |
| H3 | Verdict ≠ IGNORE + `primary_source_read = false` | Universal stop |
| H4 | ADOPT or EXTRACT + no architecture review | Hard blocker for high-stakes verdict |
| H5 | Released < 6 months ago + < 10 sentences | Sparse secondary coverage on new tools |
| H6 | Description contains "orchestration", "pipeline", "governance" + shallow coverage | Domain relevance trigger |
| H7 | `disposition_confidence < 0.5` on ADOPT/EXTRACT | Confidence-verdict mismatch |

### Checkpoint UX

```
SCAN CHECKPOINT — 2 items require depth decision

Section A — Settled (3 items, coverage ≥ 0.6)
  ...presented as normal output...

Section B — Flagged
  1. DeerFlow (50K★) — coverage 0.18 — provisional: ADOPT/EXTRACT
     Risk: Architecture classification on surface data.
     [D] Deep dive now (~15 min)
     [P] Proceed provisional (tagged sdp:scan:unverified)
     [I] Downgrade to MONITOR

  2. GPT Researcher (18K★) — coverage 0.31 — provisional: INSPIRE
     [D] / [P] / [I]
```

`sdp:scan:unverified` items cannot be cited as settled evidence in downstream feature cards without human override or resolved deep-dive.

---

## 5. Experiment Design Matrix

| Hypothesis type | Cheap (agent does it) | Medium | Expensive |
|---|---|---|---|
| "Demand exists for X" | Post in community (HN/Reddit/Indie Hackers) | Landing + waitlist | Custdev interviews (3–5) |
| "People will pay for X" | Price-anchoring survey | Landing with pricing | Pre-sales / LOI |
| "Technical solution viable" | OSS spike / PoC script | CLI prototype | Full PoC with integration |
| "Better than alternatives" | Feature comparison matrix | Prototype + user test | A/B vs analogue |

### Experiment brief format

```yaml
type: community_post | survey | landing | custdev | prototype | spike
objective: "Validate claim N: ..."
method: "..."
success_metric: ">50 signups in 14 days"
failure_metric: "<10 signups in 14 days"
effort:
  agent: "generate landing page copy + structure"
  human: "publish, share link, monitor for 14 days"
deadline: YYYY-MM-DD
result_collection:
  auto: [fireflies_transcripts, landing_analytics]
  manual: "bd update <id> --notes 'N signups, key feedback'"
```

---

## 6. Artifact Map

```
Phase 1  → docs/discovery/YYYY-MM-DD-<slug>-frame.md
Phase 2  → docs/discovery/YYYY-MM-DD-<slug>-hypothesis.md
Phase 3  → docs/discovery/YYYY-MM-DD-<slug>-scan.md
Phase 4a → docs/discovery/YYYY-MM-DD-<slug>-validation.md
Phase 4b → docs/discovery/YYYY-MM-DD-<slug>-experiment.md
Final    → docs/discovery/YYYY-MM-DD-<slug>-verdict.md  (sводный)

beads:
  - type:discovery (parent, tracks full pipeline)
  - type:task (child, per experiment human action)
  - type:feature (created on GO verdict, links to discovery)
```

---

## 7. Building Blocks

### Research engine: GPT Researcher (EXTRACT pattern)

```
GPT Researcher via gptr-mcp
  └── get_research_context()  ← use ONLY this
      └── our synthesis layer (structured prompts + JSON schema)
          ├── 4-phase methodology injection
          ├── citation verification pass
          └── coverage scoring per result

DO NOT use GPT Researcher's report generator — hallucinated citations (issue #1572)
```

**Configuration via env (routes through OpenRouter):**
```bash
SMART_LLM_MODEL=deepseek/deepseek-v3.2
FAST_LLM_MODEL=deepseek/deepseek-v3.2
STRATEGIC_LLM_MODEL=openai/gpt-5.4-mini
OPENAI_BASE_URL=https://openrouter.ai/api/v1
```

### MCP stack for Phase 3

| Layer | MCP | Cost signal |
|---|---|---|
| Web semantic | `exa-labs/exa-mcp-server` (Deep mode) | ~$0.006/query |
| Web breadth | Brave Search MCP | $5/1k queries |
| Academic | `openags/paper-search-mcp` | Free (20+ sources) |
| HN | Apify HN MCP | Free |
| Reddit | Apify Reddit MCP | Compute cents |
| GitHub | `github/github-mcp-server` + `mcp-github-trending` | Free |
| Reviews | Apify multi-platform (G2+Capterra+Trustpilot) | ~$0.006/page |

**Gaps to build:**
- Crunchbase thin MCP (Basic API free, 200 calls/day) — ~1 day effort

### DeerFlow patterns to borrow (INSPIRE)

- **Loop detection** — sliding-window hash, warn×3, hard stop×5 → reimplement in Go
- **Middleware chain** — `before_model()` / `after_model()` hooks for cross-cutting concerns
- **Tiered memory** — `workContext / recentMonths / longTermBackground / facts[]`
- **4-phase research methodology** — Broad survey → Targeted deep dive → Diversity validation (6 types) → Synthesis check
- **Typed clarification** — `missing_info | ambiguous_requirement | approach_choice | risk_confirmation`

---

## 8. Model Stack (empirically validated)

Benchmarked 2026-04-08 via OpenRouter against real reasoning task (hypothesis validation).

```yaml
planner:
  model: deepseek/deepseek-v3.2      # $0.00017/call · 163K ctx · fastest JSON
  fallback: qwen/qwen3-32b           # $0.00018/call · 0.9s

synthesizer:
  model: deepseek/deepseek-v3.2      # $0.00028/call · correct EXTRACT/INSPIRE classification
  fallback: google/gemini-3-flash-preview  # $0.00193 · faster, same alignment

reasoner:
  default:    openai/gpt-5.4-mini    # $0.00228/call · 1M ctx · new default
  balanced:   deepseek/r1-0528       # $0.00444/call · explicit reasoning chain
  critical:   anthropic/claude-sonnet-4.6  # $0.01118 · best evidence calibration
  kill_check: google/gemini-2.5-pro  # $0.03304 · most skeptical · use for final veto

embeddings:
  model: openai/text-embedding-3-small  # $0.02/1M · on OpenRouter
```

### Cost per session

| Scenario | Cost |
|---|---|
| Quick scan (Phase 1–3, no deep-dives) | $1–3 |
| Standard (full pipeline, 2–3 deep-dives) | $4–8 |
| Heavy (5+ deep-dives + experiment design) | $10–18 |

**Cost controls:**
- Cap GPT Researcher depth at 2 (not 3) — saves ~60% LLM cost
- Limit Exa Deep queries to 5 per scan (ADOPT/EXTRACT candidates only)
- Max 3 auto-triggered deep-dives per session
- Human override always available at checkpoint

---

## 9. Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| DeerFlow | INSPIRE only | LangGraph conflict with beads; 16GB RAM; all pipeline stages still need building |
| GPT Researcher | EXTRACT retrieval only | Report generator unreliable; `get_research_context()` is solid |
| Synthesis layer | Build own | JSON schema enforcement + citation verification + coverage scoring |
| Deep-dive trigger | Human-gated | Cost and scope; agent flags, human approves |
| Twitter/X | Skip | $5K/month for useful API tier; HN + Reddit sufficient |
| LinkedIn | Exa proxy | Active scraping gets accounts blocked |
| Orchestration | Go + beads | Preserve existing stack; no LangGraph dependency |
| Checkpoint async | Hybrid | Chat if online; beads + notify if offline |

---

## 10. Critical Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Hallucinated market data | 🔴 High | Require source URL for every quantitative claim; separate verified/estimated |
| Confirmation bias in scan | 🔴 High | Adversarial prompting; require disconfirming evidence section |
| Circular validation | 🟡 Medium | Require ≥1 real-world signal before GO verdict |
| Coverage score gaming | 🟡 Medium | H3 is a hard gate — no verdict without primary source read |
| Cost overrun | 🟡 Medium | Session budget cap; human approves deep-dives |

Claim 5 ("agents produce reliable scan") was marked `contradicted/high` by 4 of 8 benchmark models. **Citation verification layer is mandatory, not optional.**

---

## 11. What's NOT in Scope (v1)

- Automatic Twitter/X monitoring
- LinkedIn company signal scraping
- Automatic landing page deployment (agent generates, human deploys)
- A/B testing execution (PostHog/GrowthBook integration deferred)
- Multi-user / team collaborative discovery
- Discovery for non-developer audiences

---

## 12. Next Step

Prototype: minimal Phase 3 SCAN with depth signal mechanism.

Entry: `bd discover "idea"` → Phase 1 FRAME → Phase 3 SCAN with coverage envelope → checkpoint output.

Validate the depth signal heuristics against real ideas before building Phases 2 and 4.
