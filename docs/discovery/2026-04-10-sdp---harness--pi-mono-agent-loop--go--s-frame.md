# Discovery Frame

**Raw idea:** SDP как нативный harness: порт pi-mono agent-loop на Go, чтобы SDP сам был harness'ом с железной SDLC-дисциплиной — гейты, роли, мульти-LLM оркестрация, без зависимости от cc/codex/opencode

## Problem Statement

Current agent-loop implementations (like pi-mono) often rely on external harnesses or platforms (e.g., cc/codex/opencode) to enforce SDLC discipline, creating dependencies, integration overhead, and potential misalignment with native infrastructure goals. This limits the ability to have a self-contained, disciplined system with built-in gating, role-based controls, and multi-LLM orchestration.

## Jobs to Be Done

- Infrastructure engineers need to port the pi-mono agent-loop to Go so that the SDP (Software Development Platform) can natively act as its own harness, eliminating external dependencies.
- Platform architects need to design and integrate built-in SDLC gates, role-based access controls, and multi-LLM orchestration directly within the SDP to enforce ironclad development discipline.
- Development teams need to utilize the native SDP harness to streamline workflows, ensure compliance, and orchestrate multiple LLMs without relying on third-party tools.

**Appetite:** large

**Scope:** Port the core agent-loop logic from pi-mono to Go; design and implement native SDLC gating mechanisms, role-based permission systems, and multi-LLM orchestration within the SDP; ensure the SDP operates independently of cc/codex/opencode dependencies; include testing and documentation for the new native harness functionality.
