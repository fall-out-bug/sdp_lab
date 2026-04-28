# SDP Telemetry Schema Reference

**Version:** 1.1.0
**Last Updated:** 2026-04-27
**Status:** Stable (v1 MVP)

## Overview

SDP telemetry uses an allowlist-based schema to ensure privacy and data integrity. All trace events must conform to `schema/telemetry/sdp-trace-events.schema.json`. Unknown attributes are rejected at emit-time.

## Consent Levels

SDP telemetry implements a four-tier consent model via the `SDP_TRACE_CONSENT` environment variable:

| Level | Description | Example Data |
|-------|-------------|--------------|
| **none** | Telemetry completely disabled | No data collected, no local storage, no export |
| **metadata** (default) | Structural metadata only, no content | Tool names, durations, exit codes, SHA-1 hashes (first 8 chars) |
| **findings** | Includes finding messages without code content | Review verdicts, finding messages, rule names |
| **content** | Full content including code and prompts | Code snippets, patch diffs, prompt bodies (opt-in only) |

### Setting Consent Level

```bash
export SDP_TRACE_CONSENT=none      # Disable all telemetry
export SDP_TRACE_CONSENT=metadata  # Default
export SDP_TRACE_CONSENT=findings  # Include finding messages
export SDP_TRACE_CONSENT=content   # Full content (opt-in, debug only)
```

## Enable, Inspect, and Disable

### Enable Telemetry (local-only, default)

By default, SDP collects telemetry locally with metadata-level consent. No outbound data transfer occurs unless explicitly configured.

```bash
# Default behavior: local-only metadata telemetry
# No configuration needed - this is the default state.

# Optionally raise consent level for richer local data:
export SDP_TRACE_CONSENT=findings
```

### Enable OTEL Export

Exporting telemetry to an external OTEL collector requires **both** explicit consent and an explicit endpoint. This is a two-gate design: neither condition alone is sufficient.

```bash
# Step 1: Set OTEL collector endpoint
export SDP_OTEL_ENDPOINT=http://localhost:4318/v1/traces

# Step 2: Ensure consent level allows export (not "none")
export SDP_TRACE_CONSENT=metadata  # or findings, content

# Optional: authentication headers
export SDP_OTEL_HEADERS="api-key=secret123,tenant=acme"

# Optional: export timeout (default: 5 seconds)
export SDP_OTEL_TIMEOUT=10

# Optional: service name (default: sdp-telemetry)
export SDP_OTEL_SERVICE_NAME=my-project

# Optional: disable TLS for local development
export SDP_OTEL_INSECURE=true
```

**Important:** If `SDP_OTEL_ENDPOINT` is not set, no outbound data is sent regardless of consent level. If consent is `none`, no outbound data is sent regardless of endpoint configuration.

### Inspect Current Configuration

Use the `sdp telemetry inspect` command to see the current state:

```bash
sdp telemetry inspect
```

Output shows:
- Current consent level
- OTEL export status (enabled/blocked/not configured)
- Local storage directory
- Export endpoint details (if configured)

Use `sdp telemetry consent` (no arguments) for a quick summary:

```bash
sdp telemetry consent
```

### Disable Telemetry

There are three ways to disable telemetry:

```bash
# Option 1: Set consent to "none" (disables all telemetry including local)
export SDP_TRACE_CONSENT=none

# Option 2: Set the disable flag
export SDP_TRACE_DISABLED=true

# Option 3: Remove OTEL endpoint to disable export only (local storage continues)
unset SDP_OTEL_ENDPOINT
```

To verify telemetry is disabled:

```bash
sdp telemetry inspect
# Should show: "Telemetry completely disabled" or "OTEL export: not configured"
```

## Consent and Export Contract

The export contract enforces these invariants:

1. **No default-on outbound.** Export requires `SDP_OTEL_ENDPOINT` to be explicitly set.
2. **Consent gates content.** The `SDP_TRACE_CONSENT` level determines what data is included in exported spans.
3. **"none" blocks everything.** Setting consent to `none` disables all telemetry, including local storage.
4. **Both conditions required.** Export only happens when both an endpoint is configured AND consent is not `none`.

| SDP_OTEL_ENDPOINT | SDP_TRACE_CONSENT | Local Storage | OTEL Export |
|-------------------|-------------------|---------------|-------------|
| unset             | any               | yes (unless none) | no |
| set               | none              | no            | no |
| set               | metadata          | yes           | yes (metadata attrs) |
| set               | findings          | yes           | yes (metadata + findings attrs) |
| set               | content           | yes           | yes (all attrs) |

## Span Kinds

### 1. execute_tool

Emitted for each tool invocation (Bash, Read, Edit, Grep, Glob, Write, etc.).

**Required Attributes:**
- `gen_ai.operation.name`: Always "execute_tool"
- `gen_ai.tool.name`: Tool name (e.g., "Bash", "Read", "Edit")
- `gen_ai.tool.call.id`: Unique identifier for this tool call instance
- `sdp.session.id`: SDP session identifier
- `sdp.epic.bead_id`: Parent epic bead ID (from `.sdp/state/current-feature`)
- `sdp.harness`: Harness type (claude-code, codex, opencode, cursor)

**Optional Attributes:**
- `gen_ai.tool.type`: Tool type ("host" or "mcp")
- `sdp.phase.name`: Delivery loop phase name
- `sdp.phase.cycle_number`: Cycle number within phase
- `sdp.tool.exit_code`: Tool exit code (0 = success)
- `sdp.tool.duration_ms`: Execution duration in milliseconds
- `sdp.tool.error`: Error message if tool failed (findings level)
- `sdp.tool.input_hash`: SHA-1 hash (first 8 chars) of tool input (metadata level)
- `sdp.tool.output_hash`: SHA-1 hash (first 8 chars) of tool output (metadata level)

**Example:**
```json
{
  "span_kind": "execute_tool",
  "attributes": {
    "gen_ai.operation.name": "execute_tool",
    "gen_ai.tool.name": "Bash",
    "gen_ai.tool.call.id": "call_1234567890",
    "sdp.session.id": "sess_abc123",
    "sdp.epic.bead_id": "sdplab-kh8j",
    "sdp.harness": "claude-code",
    "sdp.phase.name": "phase_1_build",
    "sdp.phase.cycle_number": 2,
    "sdp.tool.exit_code": 0,
    "sdp.tool.duration_ms": 142
  }
}
```

### 2. invoke_agent

Emitted for skill/subagent invocation (e.g., @review, @build, @codex).

**Required Attributes:**
- `gen_ai.operation.name`: Always "invoke_agent"
- `gen_ai.agent.name`: Agent/skill name (e.g., "review", "build", "codex")
- `sdp.session.id`: SDP session identifier
- `sdp.epic.bead_id`: Parent epic bead ID

**Optional Attributes:**
- `gen_ai.conversation.id`: Conversation identifier for correlation
- `sdp.skill.name`: Skill name
- `sdp.phase.name`: Delivery loop phase name
- `sdp.phase.cycle_number`: Cycle number within phase
- `sdp.agent.duration_ms`: Agent execution duration in milliseconds
- `sdp.review.verdict`: Review verdict (findings level)
- `sdp.review.findings_count`: Total number of findings (findings level)
- `sdp.review.findings_by_severity`: JSON string mapping severity to count (findings level)
- `sdp.codex.consecutive_clean`: Number of consecutive clean codex cycles
- `sdp.codex.tests_passed`: Whether all tests passed
- `sdp.council.round`: Council round number
- `sdp.council.roles`: Array of council roles participating

**Example:**
```json
{
  "span_kind": "invoke_agent",
  "attributes": {
    "gen_ai.operation.name": "invoke_agent",
    "gen_ai.agent.name": "review",
    "sdp.session.id": "sess_abc123",
    "sdp.epic.bead_id": "sdplab-kh8j",
    "sdp.phase.name": "phase_1_build",
    "sdp.phase.cycle_number": 2,
    "sdp.agent.duration_ms": 5230,
    "sdp.review.verdict": "APPROVED",
    "sdp.review.findings_count": 0
  }
}
```

### 3. delivery_loop_phase

Emitted for delivery loop phase transitions.

**Required Attributes:**
- `sdp.phase.name`: Phase name (phase_0_bootstrap, phase_1_build, phase_2_pr, phase_3_codex, phase_4_closeout)
- `sdp.session.id`: SDP session identifier
- `sdp.epic.bead_id`: Parent epic bead ID

**Optional Attributes:**
- `sdp.phase.cycle_number`: Cycle number within phase
- `sdp.feature_id`: Feature identifier
- `sdp.workstream_count`: Number of workstreams in this feature
- `sdp.phase.duration_ms`: Phase duration in milliseconds
- `sdp.pr.number`: Pull request number
- `sdp.quality_gate.result`: Quality gate result (PASSED, FAILED, SKIPPED)
- `sdp.closeout.children_closed`: Number of child beads closed
- `sdp.closeout.duration_ms`: Closeout duration in milliseconds

**Example:**
```json
{
  "span_kind": "delivery_loop_phase",
  "attributes": {
    "sdp.phase.name": "phase_1_build",
    "sdp.session.id": "sess_abc123",
    "sdp.epic.bead_id": "sdplab-kh8j",
    "sdp.phase.cycle_number": 1,
    "sdp.feature_id": "F140",
    "sdp.workstream_count": 5
  }
}
```

### 4. sdp_bead_event

Emitted for bead state transitions.

**Required Attributes:**
- `sdp.bead.id`: Bead identifier (e.g., sdplab-xxxx)
- `sdp.bead.event`: Bead event type (claimed, started, completed, blocked, unblocked, closed, reopened)

**Optional Attributes:**
- `sdp.bead.previous_status`: Previous bead status (OPEN, IN_PROGRESS, BLOCKED, CLOSED)
- `sdp.bead.new_status`: New bead status
- `sdp.session.id`: SDP session identifier
- `sdp.epic.bead_id`: Parent epic bead ID
- `trace.id`: Associated trace ID

**Example:**
```json
{
  "span_kind": "sdp_bead_event",
  "attributes": {
    "sdp.bead.id": "sdplab-kh8j",
    "sdp.bead.event": "closed",
    "sdp.bead.previous_status": "IN_PROGRESS",
    "sdp.bead.new_status": "CLOSED",
    "sdp.session.id": "sess_abc123",
    "sdp.epic.bead_id": "sdplab-snn1"
  }
}
```

## Sampling Policy

SDP telemetry uses a three-tier sampling strategy to manage disk footprint:

### Head-Based Sampling
- **Default rate:** 100% (all spans recorded)
- Applied at trace start

### Tail-Based Drop Rules
Fast-successful utility tools are dropped as noise:
- Tool: Read, Glob, Grep
- Max duration: < 10 ms
- Require error: No (only drop if no error)

### Hash-Based Sampling
For sessions exceeding 100,000 spans:
- Sample rate: 1/N on trace_id hash
- Prevents disk exhaustion on extreme usage

## Disk Footprint Envelope

| Load | Spans/day | Bytes/span (avg) | JSONL/day | 30-day uncompressed |
|------|-----------|------------------|-----------|---------------------|
| Light (1 session/day) | 2,000 | 1.2 KB | 2.4 MB | 72 MB |
| Typical (3-5 sessions) | 10,000 | 1.2 KB | 12 MB | 360 MB |
| Heavy (parallel loops) | 50,000 | 1.2 KB | 60 MB | 1.8 GB |

**Compression:** zstd -3 on JSONL achieves 8-12× ratio
- Heavy case (30-day): ~150-220 MB compressed

## Storage Location

```
.sdp/traces/
├── YYYY-MM-DD/
│   └── spans.jsonl          # Live-write file (plain JSONL)
└── YYYY-MM-DD/
    └── spans.jsonl.zst      # Compressed archives (previous days)
```

## Trace-ID Propagation

SDP uses two canonical channels for trace correlation:

1. **TRACEPARENT** (W3C Trace Context): In-process and child subprocess correlation
2. **sdp.epic.bead_id** (span attribute): Cross-session and cross-compaction aggregation

The `gen_ai.conversation.id` is kept as a diagnostic attribute only.

## Privacy Guarantees

- **Allowlist enforcement:** Unknown attributes rejected at emit-time
- **Consent gating:** Content levels strictly enforced
- **No PII:** Names, emails, usernames never collected
- **Local-first:** Data stays on disk by default; export is opt-in
- **Audit trail:** CI contract tests verify allowlist compliance

## Related Files

- Schema: `schema/telemetry/sdp-trace-events.schema.json`
- UX Metrics: `schema/ux-metrics.schema.json` (includes M1-M7)
- Model Pricing: `configs/model_pricing.json`
- Verification: `scripts/verify_trace_attrs.sh`, `scripts/verify_pricing_freshness.sh`

## Metrics M1-M7

See `schema/ux-metrics.schema.json` for the seven AI SDLC metrics computed from trace data:

- **M1:** Review pass rate
- **M2:** Build rework rate
- **M3:** Codex stability index
- **M4:** P3 deferral rate
- **M5:** Delivery velocity (time-to-merge decomposition)
- **M6:** Cost per merged feature
- **M7:** Post-merge rework rate (DORA 5th metric)
