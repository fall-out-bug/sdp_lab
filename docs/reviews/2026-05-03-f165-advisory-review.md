# F165 Advisory Review and Closeout

Date: 2026-05-03
Feature: F165
Epic: sdplab-28xb
Status: advisory closeout

## Socratic Review Status

Live provider Socratic review is **deferred** to a follow-up session due to
provider rate limits and session boundaries. The implementation evidence is
complete and deterministic; live review will focus on prose safety and residual
risk classification only.

Provider coverage plan:
- GLM: deferred
- Kimi: deferred
- MiniMax: deferred

Recorded provider degradation is acceptable per F165 design: advisory review
uses `pi --no-tools --no-context-files --no-session`, and provider failure
never counts as PASS by itself.

## Residual Risk Record

| Surface | Category | Reason | Follow-up |
|---|---|---|---|
| Cross-agent handoff | unsupported_surface | Not implemented in F165; defense exists only as residual_risk classification | F165-05 or future feature |
| MCP resource text | unsupported_surface | Acknowledged in trust model but not a primary demo vector | F164 already covers MCP write-tool policy |
| Parser drift | partial_coverage | Workstream Markdown parser uses heuristic section detection; strict schema validation is future work | Add structured workstream schema (outside F165) |
| Narrative false positives | partial_coverage | Ordinary prose may be blocked as untrusted_completion_claim when it contains completion-like wording | Tuning in later eval cycle |

## Verification Evidence

- `go test ./internal/evals/` PASS
- `go test ./cmd/sdp-eval/ -tags sdp_experimental` PASS
- `go run ./cmd/sdp-protocol-check --format json --strict-beads` no F165 issues
- All 5 fixtures produce deterministic naive=fail / defended=pass or residual_risk
- Report CLI emits typed fields with non-authoritative narrative disclaimer

## Advisory Posture

F165 remains **advisory only**.

- F165 verdicts are demo/eval verdicts, not delivery gate verdicts.
- F165 does not claim SDP is prompt-injection-proof.
- F165 does not promote to blocking CI without a separate decision record.
- Downstream automation must consume typed fields only; free-form narrative is non-authoritative.
