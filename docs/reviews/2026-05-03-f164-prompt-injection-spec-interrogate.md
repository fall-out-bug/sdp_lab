# F164 Prompt Injection Spec Interrogation

Feature: F164
Date: 2026-05-03
Artifact:

- `docs/security/f164-prompt-injection-threat-model.md`
- `docs/security/f164-prompt-injection-test-cases.md`

## Verdict

PASS after revision round 2.

## Critic Round 1

Providers:

- `zai/glm-5.1`
- `kimi-coding/k2p6`
- `minimax/MiniMax-M2.7`

Blocking and major themes:

- F164 needed explicit ownership and decision authority.
- "Continue the original task" was not testable.
- MCP was in scope while its runtime policy substrate remained open.
- Provider failure handling and final approval rules were ambiguous.
- Phase allowlists were asserted but not defined.
- Missing deterministic evidence did not state fail-open or fail-closed behavior.
- Beads findings could launder injected text back into trusted workflow state.
- Observability, containment, and rollout/backward compatibility were absent.
- Prompt bundle supply-chain detection did not specify who or what validates hashes.

## Resolution

The revised specs now define:

- F164 as a standalone security feature.
- Human maintainer decision authority.
- Baseline exposure and first-baseline measurement requirement.
- Trusted instruction, untrusted content, task continuation, suspicious prompt-like content, allowed tools, human confirmation, provider failure, and actor-intent semantics.
- Phase tool policy and allowlist enforcement points.
- Deterministic, judgment, and policy gate classes.
- Minimum evidence provenance fields owned by F164-02.
- Beads typed metadata versus untrusted narrative separation.
- MCP boundary limited to SDP-controlled MCP mapping and tool policy.
- Observability events, metrics, payload storage rules, and containment behavior.
- Rollout and backward compatibility stages.
- Corpus versioning, benign controls, multi-vector cases, MCP tool-description poisoning, and prompt bundle supply-chain checks.

## Judge Round 2

Providers:

- `zai/glm-5.1`: PASS, unresolved `[]`
- `kimi-coding/k2p6`: PASS, unresolved `[]`
- `minimax/MiniMax-M2.7`: PASS, unresolved `[]`

The judges accepted that the final revision addressed:

- phase model and allowlist enforcement;
- forbidden-behavior detection;
- containment after successful injection;
- evidence schema ownership;
- backward compatibility;
- Beads finding poisoning loop;
- provider failure final-approval rule.

## Next Action

Convert F164 into one Beads epic plus executable leaf workstreams. Then run the workstreams through Socratic review before execution.

## CI Integration Decision

F164-07 implements advisory-first CI:

- Static corpus, prompt-surface PI-013, and mock trace regressions can block PRs.
- Live-provider evals are skipped or reported as advisory-degraded when credentials are absent.
- Moving live-provider findings or prompt hardening warnings to blocking requires a separate decision record and baseline trend evidence.
