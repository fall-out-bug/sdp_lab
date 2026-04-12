# StratAudit

## Purpose

Portable strategy traceability audit over a document corpus.

Use StratAudit when you need evidence-backed extraction, trace linking, and a report pack from strategy, roadmap, architecture, design, or execution documents.

Primary artifacts:

- `.strataudit/report.json`
- `.strataudit/report.html`
- `.strataudit/similarity_distribution.json`
- `.strataudit/strataudit.db`

## Use When

- the user wants to audit strategic alignment across a real document corpus
- the user needs traceability between levels such as strategy, architecture, design, and implementation
- the user needs document-backed findings, not free-form synthesis

## Do Not Use When

- the user only wants a prose summary without evidence
- the corpus is too small or too informal to justify traceability output
- there is no accessible document set yet

## Runtime Order

Prefer runtimes in this order:

1. host-native injected runtime from the harness
2. configured OpenAI-compatible network runtime
3. OpenRouter as the default accelerator/fallback path
4. `sdp-strataudit` CLI only when no host-side integration exists

The CLI can resolve only network runtimes from config. It cannot create a host-native runtime by itself.

## Required Inputs

- a corpus root or explicit source directories
- `strataudit.yaml`
- a runtime capable of chat completion and embeddings

## Execution Rules

- audit only against real document text
- do not invent traces, initiatives, or quotes
- preserve source language unless a later report layer explicitly derives display fields
- prefer the package API for harness integration; use the CLI as operational fallback

## Minimal Flow

1. load `strataudit.yaml`
2. resolve runtime: injected first, configured network runtime second
3. run ingest → extract → link → analyze → report
4. return artifact paths and key caveats

## References

- `docs/STRATAUDIT.md`
- `internal/strataudit/`
- `cmd/sdp-strataudit/`
