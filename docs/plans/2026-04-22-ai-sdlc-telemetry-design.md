# AI SDLC Telemetry — Market Scan + Proposed Tracing for SDP

**Date:** 2026-04-22
**Workstream:** sdplab-6x39
**Status:** design — pre-implementation
**Authored by:** @think (2 parallel research agents: market landscape + OTel GenAI tracing) → synthesis.

The operator asked: *"нам нужна глобальная трассировка использования, чтобы собирать метрики использования, успешности, хитрейт по доставке ценности сквозь ревью"* — i.e. a cross-cutting trace that joins skill invocations, tool uses, review findings, merge outcomes, and delivery success. This doc lays out what the market actually measures, what the real pitfalls are, and a concrete offline-first OTel-native proposal for SDP.

---

## 1. Market scan — what AI SDLC teams actually measure (2024–2026)

### 1.1 Frameworks

| Framework | What it measures (AI era) | Useful for SDP? |
|---|---|---|
| **DORA 2025** | Throughput (DF, LT) + Stability (CFR, Recovery Time) + **new Rework Rate** (5th metric, F128-like "change touched again within 2 weeks"). Retired Low/Med/High tiers for 7 team archetypes. | ✓ Rework rate + CFR are lighthouse for AI-era quality |
| **SPACE** | Satisfaction, Performance, Activity, Communication, Efficiency. Mostly survey + PR count + review latency. | ⚠ Mostly survey-driven — not auto-telemetry |
| **DX Core 4** (Abi Noda et al., Dec 2024) | Speed (diffs/eng) × Effectiveness (DXI 14-factor) × Quality (CFR) × Impact (% time on new capabilities). DX's AI Measurement Framework = Utilization × Impact × Cost. | ✓ Quadrant framing avoids "speed at any cost" trap; Impact axis is exactly "value delivery" |

**Critical finding from DORA 2025:** AI amplifies existing culture. Throughput ↑ (+21% tasks, +98% merged PRs) but stability ↓ (**+9% bugs**, **+54% bugs/dev YoY** in heavy-AI teams). Review time **+91%**. Time-to-merge went from **1.2 → 3.5 days**. This is the **"Acceleration Whiplash"** — and it's exactly why SDP's review-centric model (with `@review`, `/codex:rescue`, P3 strictness) is valuable — provided we can **prove** it beats the median through data.

### 1.2 What vendors actually ship (telemetry APIs, April 2026)

| Vendor | Metrics exposed | Notable gotcha |
|---|---|---|
| **GitHub Copilot Metrics API** (GA Oct 2025) | `total_active_users`, `total_engaged_users`, `total_code_suggestions`, `total_code_acceptances`, `total_code_lines_suggested`, `total_code_lines_accepted`, `agent_mode_dau/wau`, editor/language/model breakdown | 5-seat minimum, 1-day lag, **IDE-only** (excludes GitHub.com, CLI, mobile) |
| **Cursor Team Analytics** | AI share of committed code (on-device git-diff signature match), `total_accepts`, `total_cents`, Tab vs Composer, CSV export API | Post-commit signature match — misses non-committed suggestions |
| **Windsurf** | AI-written-code %, subteam analytics, FedRAMP High audit logs | Enterprise tier only |
| **Sourcegraph Cody Analytics** | Cross-repo indexing + admin token controls | Separate cloud property |
| **Amazon Q Developer** | Hourly dashboard + CloudTrail → Athena for deep queries; S3 CSV export | Conflates IDE and CLI events |
| **JetBrains AI** | AI Credits consumption, acceptance rate for **agent/Junie/Codex** (excludes inline completions since they don't bill) | Acceptance rate masks the high-volume completion path |
| **Claude Code Analytics API** (`/v1/organizations/usage_report/claude_code`) | sessions, LOC, commits, PRs, tool usage, token/cost by model, `terminal_type` | Native telemetry is there — just off by default in self-hosted/CLI |

### 1.3 AI-specific metrics (new in last 18 months)

| Metric | Definition | Honest critique |
|---|---|---|
| **Acceptance rate** | `accepted_suggestions / shown_suggestions` | **Gameable.** Stack Overflow 2025: only 33 % of devs trust AI output; 66 % cite "almost right" as #1 pain — high acceptance ≠ high quality |
| **AI-generated LOC / % PR** | signature-match against commits, or acceptance×suggestion size | **Meaningless alone.** Faros: +21 % tasks, +98 % merged PRs, +154 % PR size, but +91 % review time — net org-level DORA flat |
| **Rework rate** | DORA 5th metric: deployments unplanned from prior incident | **Best AI-era signal** — captures the "looks ok, broke later" failure mode |
| **First-time-right / review-pass-rate** | PRs merging without change requests | CodeRabbit: AI PRs have ~1.7× more issues; 17 % of AI PRs contain sev-9/10 |
| **Time-to-merge (AI vs non-AI)** | Faros median 1.2 → 3.5 days post-AI | Rises because bottleneck **shifts to review** |
| **Regression rate** | bugs/dev in heavy-AI teams | Faros: +54 % YoY |
| **Cost per merged PR** | `$tokens / merged_PRs` | Grafana default alerts at $1 000/day or 3× anomaly |
| **RAG hit rate** | retrieval-precision@k / grounded-answer rate | Tracked via OpenInference `retrieval.*` spans |

### 1.4 Known flaws of popular metrics — what actually happens at scale

1. **Acceptance ≠ value.** ~46 % offer rate, ~30 % true acceptance, only 33 % trust.
2. **AI LOC / % is a vanity metric** unless tied to downstream outcomes (merge, rework, incident).
3. **METR RCT (16 devs, 246 tasks, mature repos, July 2025):** devs *expected* −24 %, *felt* −20 %, **actually +19 % completion time**. The feb-2026 follow-up shows likely speedup *now* but with heavy self-selection bias.
4. **Bottleneck migrates; dashboards don't.** Coding speeds up, review becomes the constraint. Cursor's acquisition of Graphite is the honest admission. PRs merging with zero review rose +31 %.
5. **Seniority dilutes the lift.** MIT/Princeton/Wharton n=4,867: above-median-tenure devs show no significant gain. Org-wide KPIs mis-price this.
6. **Goodhart at scale.** DX explicitly warns against measuring diffs/engineer at individual level.
7. **Telemetry coverage holes.** Copilot excludes GitHub.com/mobile/CLI. JetBrains excludes in-editor completions. No vendor links to merged-PR outcomes natively. Dashboards measure **the editor, not delivery**.

### 1.5 Standards for AI tracing

- **OpenTelemetry GenAI semantic conventions** — *Development/experimental* as of v1.36 (April 2026). Opt-in: `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`. Canonical spans: `gen_ai.client`, `invoke_agent`, `execute_tool`. Core attributes: `gen_ai.provider.name`, `gen_ai.operation.name`, `gen_ai.request.model`, `gen_ai.usage.input_tokens` / `output_tokens`, `gen_ai.conversation.id`, `gen_ai.agent.*`, `gen_ai.tool.name` / `.call.id`.
- **OpenInference (Arize)** — more mature; first-class cache-token accounting, `openinference.span.kind` ∈ {LLM, CHAIN, TOOL, RETRIEVER, EMBEDDING, AGENT}; native in Phoenix/LangSmith/Opik.
- **Platform pricing gotchas**: LangSmith per-trace at agent scale explodes (users sample to 0.1 % to survive); Langfuse "units" ≈ 12 per 10-span trace; Arize charges per-GB on payload → RAG hits hard; Helicone proxy sees only request/response boundary.

Emerging pattern: **Helicone gateway (cost) + Langfuse/Phoenix (deep traces)** for teams that go cloud; **file exporter + reducer** for offline-first.

---

## 2. SDP-specific framing — what "value delivered through review" looks like

Our differentiator is that SDP already has the pieces that vendor dashboards lack:

| SDP signal | Corresponds to |
|---|---|
| `@build` / `@review` / `@fix` subagent invocations | Coding speed + rework cycles |
| `@review` **findings by severity** + `consecutive_clean_cycles` | Review strictness + first-time-right |
| `/codex:rescue` cycle count + `tests_passed` | Independent quality check |
| `bd close` on epic + child WS | Merged outcome — ground truth |
| `delivery-loop` phase durations | Time-to-merge decomposition |
| P3 auto-spin-out bead | Deferred debt ("Goodhart tax") |
| `@llm-council` veto counts | Design-quality audit trail |

We are in a rare position: we can **stitch skill invocation → subagent chain → review verdict → merge outcome → post-merge rework** into one trace. Vendor dashboards can't — they stop at the IDE boundary.

### 2.1 SDP-native metrics (proposed)

Seven metrics, each tied to a traceable span chain:

| # | Metric | Definition | Trace primitive |
|---|---|---|---|
| **M1** | **Review pass rate** | % of `@review` calls that return APPROVED on first try | count `gen_ai.operation.name=invoke_agent`, `gen_ai.agent.name=review`, `sdp.review.verdict=APPROVED` on cycle=1 |
| **M2** | **Build rework rate** | avg cycles in Phase 1 before APPROVED | max(`sdp.phase.cycle_number`) per epic |
| **M3** | **Codex stability index** | avg consecutive clean codex cycles needed before exit (target ≥ 2) | max(`sdp.codex.consecutive_clean`) per PR |
| **M4** | **P3 deferral rate** | % features that spin out P3 at cycle-5 cap | count spin-out beads / count merged features |
| **M5** | **Delivery velocity (time-to-merge decomposition)** | phase_0 / phase_1_build / phase_3_codex / phase_4_closeout durations | span.end - span.start on phase spans |
| **M6** | **Cost per merged feature** | sum(`gen_ai.usage.*_tokens × model_price`) for all spans under trace_id | aggregate on `epic_bead_id` + `pr_merged=true` |
| **M7** | **Post-merge rework rate** (DORA 5th) | % merged PRs whose files are touched again within 14 days | join traces on `git log --since` for merged commits |

M7 is the lighthouse metric: directly measures **value actually delivered** (vs Faros's +98 % merged PRs that are frequently reverted).

### 2.2 Derived dashboards

Three views an operator actually wants:

1. **"Am I shipping junk?"** — weekly M7 × M3 × M1 × CFR. One number: `trust_index`.
2. **"Where does time go?"** — weekly M5 stacked by phase + top-5 longest subagent invocations.
3. **"What's it costing?"** — weekly M6 by feature + model breakdown + cost-per-merged-feature trend.

---

## 3. Proposed tracing architecture

### 3.1 Principles

1. **Offline-first.** Default path writes to `.sdp/traces/YYYY-MM-DD/*.jsonl.zst`. No cloud dependency for collection. Export is opt-in (reuse F128 consent model).
2. **OTel-native wire format.** OTLP-JSON (not a custom extension of `schema/ux-metrics.schema.json`). Rationale: forward compat with GenAI semconv; zero vendor lock-in; `ux-metrics.schema.json` becomes the **aggregator output**, not the input.
3. **Multi-harness.** Each harness emits OTel; a local Collector is the single sink. Shim via hooks where native support is missing.
4. **Privacy-first.** No prompt bodies, tool I/O bodies, or secrets by default. `SDP_TRACE_CONSENT=content` elevates (reuses F128 consent gate).
5. **Correlation across layers.** Trace-ID (W3C Trace Context) propagates harness → skill → subagent → subprocess → MCP boundary. Plus `gen_ai.conversation.id` as stitching fallback for post-compaction recovery.

### 3.2 Reference stack

```
┌─────────────────────────────────────────────────────────────────────┐
│  Claude Code / Codex CLI / OpenCode / Cursor                        │
│    • native OTel emit (Claude Code, Codex)                          │
│    • hook shim via otel-cli (OpenCode, Cursor)                      │
│    • PreToolUse/PostToolUse emit `execute_tool` spans universally   │
│    • skill entry/exit wrappers emit `invoke_agent` spans            │
└────────────────────────────┬────────────────────────────────────────┘
                             │ OTLP gRPC :4317 / HTTP :4318
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Local OpenTelemetry Collector (sdp-telemetry.service)              │
│    receivers: otlp, otlpjsonfile (retroactive replay)               │
│    processors: batch, resource (inject sdp.consent.level),          │
│                transform (redact unexpected fields)                 │
│    extensions: file_storage (crash-safe WAL)                        │
│    exporters:                                                       │
│      file (default): .sdp/traces/YYYY-MM-DD/spans.jsonl.zst         │
│      otlphttp (opt-in export): → Langfuse / Tempo / ...             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│  sdp-telemetry-reducer (Go binary, runs on cron or pre-commit)      │
│    reads .sdp/traces/**/*.jsonl.zst                                 │
│    groups by trace_id / epic_bead_id                                │
│    emits .sdp/metrics/YYYY-MM-DD.ux.json (ux-metrics.schema.json)  │
│    emits .sdp/metrics/YYYY-MM-DD.sdp.json (SDP-specific M1–M7)     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                ┌────────────┴────────────────────────┐
                ▼                                     ▼
      `sdp usage report`                  Optional: Grafana (local SQLite)
      CLI table output                    or Langfuse self-hosted pointed
                                          at the same Collector
```

### 3.3 What each hook emits

**PreToolUse hook** (any harness) — starts an `execute_tool` span:

```bash
TRACEPARENT="${TRACEPARENT:-$(sdp trace init --tool "$TOOL_NAME")}"
otel-cli span start --name "execute_tool $TOOL_NAME" --tp "$TRACEPARENT" \
  --attrs "\
gen_ai.operation.name=execute_tool,\
gen_ai.tool.name=$TOOL_NAME,\
sdp.session.id=$SDP_SESSION_ID,\
sdp.epic.bead_id=${SDP_EPIC_BEAD_ID:-},\
sdp.skill.name=${SDP_SKILL_NAME:-},\
sdp.harness=${SDP_HARNESS:-claude-code}\
"
echo "$TRACEPARENT" > .sdp/traces/current.env
```

**PostToolUse hook** — ends it with exit status + emits workflow events (existing behavior preserved).

**Skill entry/exit wrapper** — wraps `@<skill>` invocations in `invoke_agent` spans:

```bash
# .agents/skills/_wrap.sh
sdp trace span-start "invoke_agent $1" \
  --attr "gen_ai.agent.name=$1" \
  --attr "sdp.skill.name=$1"
```

**Delivery-loop specifics** — append per-phase spans:

| Span name | Attributes |
|---|---|
| `delivery_loop.phase_0_bootstrap` | `sdp.feature_id`, `sdp.workstream_count` |
| `delivery_loop.phase_1_build.cycle_N` | `sdp.review.verdict`, `sdp.findings.count`, `sdp.findings.by_severity` (json) |
| `delivery_loop.phase_2_pr` | `sdp.pr.number`, `sdp.quality_gate.result` |
| `delivery_loop.phase_3_codex.cycle_N` | `sdp.codex.consecutive_clean`, `sdp.codex.tests_passed`, `sdp.codex.findings.count` |
| `delivery_loop.phase_4_closeout` | `sdp.closeout.children_closed`, `sdp.closeout.duration_ms` |

### 3.4 Trace-ID propagation — the hard part

W3C Trace Context via `TRACEPARENT` env var. Flow:

1. `/deliver` generates root span → writes `TRACEPARENT` to subprocess env **and** to `.sdp/traces/current.env` (belt-and-suspenders for when env isn't inherited across subagent fork).
2. `scripts/sdp-dispatch.sh subagent <skill>` (already shipped in deliver redesign) explicitly reads `.sdp/traces/current.env` and forwards to the subagent.
3. MCP server boundary: inject `traceparent` into request metadata; extract server-side.
4. On compaction: `gen_ai.conversation.id` survives — used by reducer to stitch orphan spans to parent trace.
5. `sdp_epic_bead_id` carried as resource attribute on every span — lets the reducer aggregate across compactions **even if** trace-id is lost.

### 3.5 Storage, retention, export

- **Location:** `.sdp/traces/YYYY-MM-DD/spans.jsonl.zst` (one file per day, zstd-compressed).
- **Retention:** 30 days local (Collector's file exporter `max_days: 30`). Aggregated metrics (`*.ux.json`, `*.sdp.json`) kept indefinitely — small, auditable.
- **Opt-in export:** Collector's second `otlphttp` exporter is enabled only if `.sdp/telemetry-consent.json` has `export.enabled=true`. Points at self-hosted Langfuse or a team-controlled OTel Collector.
- **Privacy gate:** `transform` processor drops any span attribute matching `gen_ai.prompt`, `gen_ai.completion`, `gen_ai.input.messages`, `gen_ai.output.messages`, `tool.input.*`, `tool.output.*` unless `sdp.consent.level=content`.

---

## 4. Proposed WS breakdown (8 workstreams)

| WS | Scope | Deps |
|---|---|---|
| **00-TEL-01** Schema + consent | Extend `schema/ux-metrics.schema.json` with M1–M7 output fields; new `schema/sdp-trace-events.schema.json`. Reuse F128 consent model. | — |
| **00-TEL-02** `sdp trace` CLI | `sdp trace init/span-start/span-end/export` — wraps `otel-cli` for cross-harness use; handles `.sdp/traces/current.env`. | TEL-01 |
| **00-TEL-03** Collector service | `scripts/install-sdp-telemetry.sh` (launchd/systemd); default config writing to `.sdp/traces/`. | TEL-02 |
| **00-TEL-04** Hook shims | Update `.claude/hooks/{Pre,Post}ToolUse.sh` + `.codex/hooks/` + `.opencode/hooks/` to emit `execute_tool` spans. Existing behavior preserved. | TEL-02 |
| **00-TEL-05** Delivery-loop instrumentation | Emit phase spans from `delivery-loop` skill; pass `sdp.feature_id`, cycle metadata. | TEL-02, delivery-loop v2 |
| **00-TEL-06** Reducer | Go binary `cmd/sdp-telemetry-reducer/main.go` — reads JSONL, computes M1–M7, writes `ux-metrics.schema.json`-compliant output. | TEL-01 |
| **00-TEL-07** `sdp usage report` CLI | Prints M1–M7 tables `--since 7d` / `--feature FXXX`. | TEL-06 |
| **00-TEL-08** Dashboard option | Optional: Langfuse self-hosted docker-compose + pre-wired queries, OR Grafana panel pack pointing at SQLite reducer output. | TEL-06 |

**Recommended sequence:** TEL-01 → TEL-02 → TEL-03/04 (parallel) → TEL-05 → TEL-06 → TEL-07 (TEL-08 optional, skip at v1).

---

## 5. Anti-goals (what we explicitly won't measure at v1)

| Deliberately skipped | Why |
|---|---|
| **Acceptance rate per Bash/Edit** | Goodhart's law + vendor data shows it doesn't correlate with value. Track only at `@review` / `/codex:rescue` boundaries, not inline. |
| **AI-generated LOC %** | Meaningless without downstream linkage; we have the linkage (bead close + M7), so we measure outcomes directly instead. |
| **Per-user leaderboards** | DX explicitly warns against measuring diffs/engineer individually. Aggregate only. |
| **Prompt/completion content capture** | Opt-in only (`SDP_TRACE_CONSENT=content`); default is metadata-only. |
| **Cloud analytics at v1** | Offline-first. Langfuse self-host is v2 option, not v1 dependency. |
| **Real-time streaming dashboards** | Cron reducer + Grafana-at-rest is enough; real-time is a v2 affordance. |

## 6. Minority / open questions (for next cycle)

1. **Does capturing `@review` finding bodies cross a privacy line?** Findings reference code file:line — possibly OK but depends on repo sensitivity. Propose: capture severity + rule + hash of snippet by default; full body only under `SDP_TRACE_CONSENT=content`.
2. **Should MCP servers emit spans?** MCP has no native OTel today. We'd need a propagator shim — probably v2.
3. **How do we handle `@llm-council` traces?** 6-role multi-round sessions can blow up span counts (6 × 5 rounds = 30 spans per council). Propose: emit one `invoke_agent` per role-round, flatten votes into attributes, not child spans.
4. **Cost attribution for subagent models.** Haiku/Sonnet mix in delivery-loop — need per-model pricing table (pull from Anthropic billing API or hardcode).
5. **Beads-telemetry joint store.** Should bead events (open/close/claim) live in the same trace stream or a parallel one? Trace is transient (30 d retention); beads are ground truth. Propose: emit a correlation span `sdp.bead.event` with bead_id and trace_id but keep beads as SoT.

## 7. Closeout

This doc is the **design** for sdplab-6x39. Next: socratic dialogue + council critique on §3 (architecture) and §4 (WS breakdown), then spin up the TEL-01–08 WS beads.

Synthesis is complete. Ready for operator decision: which WS to spin up first, and whether to invoke council before implementation.
