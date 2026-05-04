# F166 Codex/Pi Gateway Compatibility Research

Date: 2026-05-04
Feature: F166
Workstream: 00-166-07
Beads: sdplab-n7jj

## Verdict

Codex and Pi can be routed through `sdp-llm-gateway`, but not through the current
simplified demo schema. The gateway needs two OpenAI-compatible surfaces:

- Codex: `/v1/responses` with server-sent events and terminal
  `response.completed`.
- Pi: `/v1/chat/completions` with `stream: true` Chat Completions streaming.

This is compatible with the F166 thin-shim decision. It does not justify adopting
LiteLLM as the gateway dependency.

Socratic `pi-review` result: approved on 2026-05-04 after fixing blocking
findings around dependency status, portable evidence, F167 boundary, and
test-plan specificity. Final review had 0 P0, 0 P1, 9 advisory findings, with
2/3 reviewer quorum.

## Evidence

Versions tested:

| Harness | Version | Smoke model id |
|---|---|---|
| Codex | `codex-cli 0.128.0` | `fake-codex-model` |
| Pi | `0.72.1` | `fake-pi-model` |

Codex:

- Official Codex config documents custom `model_providers.<id>.base_url`,
  `openai_base_url`, and `wire_api = "responses"`.
- Local `codex exec` smoke with a fake provider confirmed that Codex posts to
  `/v1/responses`, asks for `text/event-stream`, and fails/retries when a fake
  non-stream JSON response closes before `response.completed`.

Pi:

- Local Pi docs document custom providers in `~/.pi/agent/models.json`; the
  required portable shape is:

```json
{
  "providers": {
    "sdp-gateway": {
      "baseUrl": "http://127.0.0.1:8788/v1",
      "api": "openai-completions",
      "apiKey": "dummy",
      "models": [{"id": "fake-pi-model"}],
      "compat": {
        "supportsDeveloperRole": false,
        "supportsReasoningEffort": false
      }
    }
  }
}
```

- A custom provider with `api: "openai-completions"`, fake `baseUrl`, dummy
  `apiKey`, and a fake model was accepted by Pi.
- Local smoke confirmed Pi posts to `/v1/chat/completions` with `stream: true`
  and `stream_options.include_usage: true`.

## Design Impact

The implementation must treat harness compatibility as a guard-facing adapter
problem:

- normalize Responses and Chat Completions requests into the guard input model;
- run deterministic scanner and optional local chunked classifier before egress;
- return protocol-shaped safe responses for blocked input;
- avoid raw streaming output before output guard policy has scanned or buffered
  assistant text;
- record compact audit evidence with harness, endpoint surface, model, verdict,
  stream mode, and upstream-called state.

## Sources

- Codex config reference:
  https://developers.openai.com/codex/config-reference#configtoml
- Pi provider/model docs were inspected from the installed `pi` package. The
  portable provider shape used by this workstream is included above because the
  installed package docs are local filesystem artifacts.

## Design Contract

- `docs/plans/2026-05-03-f166-runtime-llm-guard-gateway-design.md`
  section `Codex And Pi Harness Compatibility`
