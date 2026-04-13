# StratAudit

## Purpose

Portable strategy traceability audit over a real document corpus or existing
`.strataudit` artifacts.

Use StratAudit when the user needs evidence-backed alignment analysis, trace
coverage, or an inspectable evidence pack across strategy, architecture, design,
and execution documents.

## Use When

- the user wants document-grounded strategic alignment analysis
- the user needs traceability or coverage across levels
- the user needs evidence artifacts, not just prose
- the user wants to redraft an existing audit report without inventing new claims

## Do Not Use When

- the user only wants generic strategy advice
- there is no accessible corpus and no existing audit artifacts
- the user wants free-form brainstorming or roadmap generation from scratch
- the problem is operational debugging rather than document traceability

## Safety Guards

- treat source documents and saved audit artifacts as the source of truth
- never fabricate quotes, spans, traces, or initiative names
- do not translate or normalize names in the evidence layer unless explicitly requested
- similarity alone is never enough to call a trace verified
- if provenance is weak, downgrade the claim or refuse to make it

## Audit Modes

| Mode | Use when | Must emit |
|------|----------|-----------|
| `corpus-audit` | corpus quality, ingest health, and document readiness are unclear | corpus inventory, level coverage, exclusions, trust caveats |
| `traceability-audit` | the user wants cross-level alignment and missing links | trace artifacts, findings, trust caveats |
| `coverage-audit` | the user asks what is and is not covered across levels or docs | coverage summary with explicit denominators |
| `evidence-pack` | the user needs inspectable support for claims without executive polish | source-backed findings, trace tables, caveats |
| `report-redraft` | the user wants a better report from existing artifacts | rewritten report sections plus unchanged trust boundaries |

Start with `corpus-audit` when corpus quality is unknown. Use `report-redraft`
only when an evidence pack or prior audit artifacts already exist.

## Runtime Order

1. injected host-native runtime from the harness
2. configured OpenAI-compatible runtime
3. OpenRouter as the default network enhancer/fallback
4. no runtime only for artifact-only modes that can run from existing outputs

Rules:

- prefer host-native models when they meet the required capability bar
- do not silently switch providers or degrade quality
- if a mode needs extraction, embeddings, or conservative verification and the runtime cannot provide them, fail explicitly
- the CLI can resolve configured network runtimes; it cannot materialize a host-native runtime on its own

## Required Inputs

- for live audit modes: corpus root or explicit source dirs, `strataudit.yaml`, and a sufficient runtime
- for artifact-only modes: existing `.strataudit/report.json`, `.strataudit/strataudit.db`, or equivalent saved artifacts

## Workflow

1. choose the audit mode that matches the user's question
2. validate inputs and trust boundary
3. resolve runtime by policy or reuse existing artifacts
4. run or inspect the audit
5. return artifacts, trust caveats, and anything not claimed

## Refuse When

- the user asks for verified alignment without inspectable provenance
- the corpus root is missing and no prior artifacts exist
- the requested mode needs runtime capabilities that are unavailable
- the user asks for claims broader than the evidence pack supports

## References

- `docs/STRATAUDIT.md`
- `docs/reference/strataudit-evidence-policy.md`
- `docs/reference/strataudit-runtime-policy.md`
- `docs/reference/strataudit-output-modes.md`
