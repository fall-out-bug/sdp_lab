# Discovery Hypothesis

**Raw idea:** SDP как нативный harness: порт pi-mono agent-loop на Go, чтобы SDP сам был harness'ом с железной SDLC-дисциплиной — гейты, роли, мульти-LLM оркестрация, без зависимости от cc/codex/opencode

## Test Card (Strategyzer)

**We believe** Infrastructure engineers at enterprise software companies need to port the pi-mono agent-loop to Go so that the SDP can natively act as its own harness because external dependencies create integration overhead and misalignment with native infrastructure goals

**To verify this**, we will 5-question interview with 3 infrastructure engineers who have attempted to integrate pi-mono with external harnesses

**We'll measure** Percentage of interviewees who rank dependency elimination as their top priority

**We are right if** 8 out of 10 interviewees rank eliminating external harness dependencies as their #1 friction point within 14 days

## Assumptions (RAT-Ranked)

| Rank | Assumption | Risk | Uncertainty | RAT Score |
|------|-----------|------|-------------|----------|
| 1 | A Go-native SDP harness would provide sufficient SDLC discipline without external tools | high | high | 9 |
| 2 | Multi-LLM orchestration within the SDP would provide tangible efficiency gains | high | high | 9 |
| 3 | Platform architects can design effective built-in SDLC gates and role-based controls | medium | medium | 4 |
| 4 | The appetite for this solution justifies a large development investment | medium | medium | 4 |
| 5 | Infrastructure engineers perceive external harness dependencies as a significant operational burden | medium | low | 2 |
| 6 | Development teams would adopt a native SDP harness if it streamlines their workflows | low | medium | 2 |

**Riskiest assumption (rank 1):** A Go-native SDP harness would provide sufficient SDLC discipline without external tools

## Requirements

- User can enforce SDLC discipline through built-in gating mechanisms without external tools
- User can configure role-based access controls for different team members within the system
- User can orchestrate multiple LLMs through a unified interface within the SDP
- User can port existing agent-loop implementations to the native Go harness
