# Harness Engineering Landscape, May 2026

Status: research draft
Date: 2026-05-15

This document records current harness-engineering patterns and model/tool
changes relevant to SDP. It separates vendor claims, official docs, academic
findings, and local observations. Numbers from vendor launch pages are useful
for model routing hypotheses, not for merge or release authority.

## Source Classes

- Official / vendor docs: OpenAI Codex and GPT-5.5 docs, Z.AI GLM-5.1 docs,
  Kimi K2.6 page, Qwen3.6 GitHub, OpenCode docs, Pi docs.
- Platform / model infrastructure docs: Hugging Face DeepSeek-V4 analysis,
  AWS Qwen SageMaker availability note.
- Academic / preprint evidence: terminal coding agent architecture, instruction
  adherence in config files, tool-enabled agent security, MCP tool usage.
- Local observations: installed `opencode` is `1.15.0`; installed `codex` is
  `codex-cli 0.130.0`; `pi` is installed at `/opt/homebrew/bin/pi`, but did not
  print a version with `-v`/`--version` in this checkout.

## Executive Summary

The 2026 harness shift is not "models got better at code". The more important
shift is that coding agents are becoming managed runtimes:

- bounded sandboxes and writable roots;
- network policy and approval gates;
- per-agent permissions and model routing;
- context compaction and memory;
- task/flow persistence;
- telemetry and audit trails;
- long-horizon budgets and graceful timeouts;
- explicit degraded-evidence states.

SDP should treat model choice as one routing dimension inside a governed
harness, not as the architecture.

## Current Harness Engineering Patterns

### 1. Agent Loop Is The Product Boundary

OpenAI describes Codex as a suite, but the core reusable concept is the harness:
an agent loop that prepares prompts, calls model inference, invokes tools,
observes results, updates state, and returns an outcome. Codex CLI uses the
Responses API and can be configured against compatible endpoints.

Implication for SDP: keep the canonical model at the loop/runtime level:
instructions, tools, state, evidence, approvals, and model routing. Do not make
one harness-specific prompt stack the product.

### 2. Safety Is Moving To Managed Runtime Controls

OpenAI's May 2026 Codex safety write-up emphasizes sandbox modes, approval
policy, managed network access, managed configuration, and audit logs. OpenCode
documents per-agent permission keys for reads, edits, bash, web, LSP, skills,
task delegation, and external directory access. Pi intentionally keeps the core
minimal and expects teams to add workflows through packages/extensions or
external process controls.

Implication for SDP: prompt-injection safety must be backed by runtime:

- read-only reviewer roles;
- explicit write-enabled implementer roles;
- network allowlists or ask gates;
- `git push`, publish, merge, and external side effects behind explicit gates;
- append-only evidence for what the agent actually did.

### 3. Context Engineering Is A Runtime Concern

The terminal-agent literature now names strict context management as a core
architecture problem. OPENDEV-style patterns include workload-specialized model
routing, planner/executor separation, lazy tool discovery, adaptive compaction,
memory, and event-driven reminders.

A May 2026 instruction-adherence study found no strong effect from several
static config-file structure variables, but did find a within-session compliance
drop as generated work accumulates. This supports SDP's existing bias toward
shorter bounded loops, compaction, and fresh reviewer contexts.

Implication for SDP: improve session lifecycle, not just AGENTS formatting.

### 4. Tool Layer Risk Is Growing

Recent MCP/tool studies show software development dominates agent tooling, and
the share of action tools has grown sharply. Security analysis of privileged
agent environments points to over-privileged tools, capability-intent mismatch,
and ambient authority leakage as practical risk categories.

Implication for SDP: classify tools by side effect, not by convenience:

- perception/read tools;
- analysis tools;
- local writes;
- external writes;
- irreversible or identity-mediated actions.

Each class needs separate permission and evidence policy.

## Model Landscape

### GPT / Codex

OpenAI's current public docs position GPT-5.5 as a strong fit for coding,
tool-heavy agents, long-context retrieval, and product-spec-to-plan workflows.
The migration guidance stresses outcome-first prompts, explicit success
criteria, tool descriptions, structured outputs, prompt caching, and reasoning
effort tuning. It also says coding workflows need stronger orchestration:
reuse, delegation, tests, acceptance criteria, and clear stop/ask rules.

Codex Cloud/Web supports background tasks in cloud environments, parallel work,
GitHub PR workflows, environment setup, internet-access control, IDE delegation,
and GitHub tagging.

SDP routing hypothesis:

- use GPT/Codex-class models for hard synthesis, complex tool orchestration,
  final review, and high-risk refactors;
- use lower reasoning levels or smaller variants only after evals prove they
  preserve evidence quality;
- prefer structured outputs over prompt-described schemas where available.

### GLM

Z.AI documents GLM-5.1 as a flagship long-horizon model with 200K context,
128K maximum output, thinking modes, function calling, context caching,
structured output, MCP integration, and agentic coding use cases. Vendor claims
include up to 8-hour sustained autonomous work and strong SWE-Bench Pro results.

SDP routing hypothesis:

- good candidate for long-horizon implementer/reviewer lanes;
- verify provider stability and context degradation on real SDP tasks before
  trusting it for unattended work;
- use as model-diversity reviewer for prompt/skill/spec work.

### Kimi

Kimi K2.6 is positioned as open-source with coding, long-horizon execution,
agent swarm capabilities, document-to-skill workflows, and Claw Groups for
coordinated multi-agent work. Official access paths include Kimi website, app,
API, and Kimi Code.

SDP routing hypothesis:

- strong fit for UI/front-end generation critique, skill extraction, and
  document-to-workflow experiments;
- useful as a non-OpenAI review lane;
- vendor swarm claims should be treated as design inspiration, not evidence
  until reproduced in SDP.

### Qwen

The Qwen3.6 repository positions Qwen3.6 as focused on stability and real-world
utility, with stronger agentic coding, front-end workflows, repository-level
reasoning, and thinking preservation across iterative development. Official
weights are on Hugging Face and ModelScope; Qwen Code is the terminal agent
optimized for Qwen models; Alibaba Cloud Model Studio provides OpenAI- and
Anthropic-compatible APIs. AWS now lists Qwen3.6-35B-A3B as a 3B-active MoE
optimized for agentic coding workflows.

SDP routing hypothesis:

- good local/open-weight candidate for cheap exploration, codebase mapping, and
  review diversity;
- Qwen3.6-35B-A3B and 27B should be evaluated separately; small active-parameter
  models may be fast enough for many scout/reviewer lanes;
- thinking-preservation is relevant to long-horizon harness design, but should
  not replace external evidence.

### DeepSeek

DeepSeek V4 is current but needs extra caution because official English docs
were harder to source directly. Hugging Face's April 2026 analysis reports V4-Pro
and V4-Flash checkpoints with 1M context, agent-focused long-context
architecture, interleaved thinking across tool calls, a dedicated tool-call
schema, and sandbox infrastructure used for RL rollouts. The same source says
benchmark numbers are competitive but not uniformly SOTA.

SDP routing hypothesis:

- promising for long-context agent workloads and cost-sensitive reviewer lanes;
- tool-call schema differences may require harness adaptation;
- use with explicit provider/endpoint provenance because community reports mix
  API, OpenRouter, chat, and preview variants.

## Tool Landscape

### Codex

Codex is now a family: CLI, Cloud/Web, IDE extension, and app surfaces. The
important 2026 lessons are:

- agent loop and tool mediation are first-class;
- cloud tasks run in task-specific environments;
- background and parallel work are product features;
- safety posture centers on sandbox, approvals, network policy, managed config,
  and telemetry.

SDP implication: Codex should be treated as one high-reliability execution lane
and as a model/harness reference for managed controls.

### OpenCode

OpenCode's current docs support per-agent model overrides, per-agent
permissions, wildcard command permissions, primary/subagent/all modes, hidden
subagents, and task-delegation permissions. Legacy `tools` config is deprecated
in favor of `permission`.

SDP implication: adapter parity should include permission semantics, not just
file presence. Existing SDP guidance to use `--agent implementer` for
non-interactive dispatch remains important, but should be refreshed against the
current OpenCode agent/permission model.

### Pi

Pi's official posture is minimal core plus extensions, skills, prompt templates,
themes, packages, SDK/RPC/event-stream modes, and model registry customization.
It loads `AGENTS.md`/`CLAUDE.md` from global, parent, and current directories.
It intentionally does not include built-in MCP, subagents, permission popups,
plan mode, to-dos, or background bash; those are built through packages or
external tools. The `pi-agents` and `pi-agent-flow` packages add spawn,
sequence, fork, join, loop, budgets, flow persistence, and deadline handling.

SDP implication: Pi is a good experimental harness for explicit workflow
graphs, diverse model review lanes, and local package-based extension. It needs
strict external guardrails for side effects because the core is intentionally
small.

## Consequences For SDP

1. Add runtime permission semantics to the manifest, not just skill/agent file
   parity.
2. Separate static adapter parity from runtime dispatch evidence for every
   harness.
3. Treat long-horizon work as a flow with budgets, checkpoints, compaction, and
   review stops.
4. Route models by role: scout, planner, implementer, reviewer, security,
   synthesis, and judge.
5. Keep model diversity in review lanes; avoid all-reviewer panels from one
   vendor family.
6. Track endpoint provenance and model version in evidence for any external
   model result.
7. Prefer bounded workflows over one giant autonomous run.
8. Encode degraded evidence explicitly: failed provider, timeout, empty output,
   unavailable CLI, unverified benchmark, not-assessed runtime.
9. Treat identity-mediated tools as higher risk than API tools with scoped
   credentials.
10. Measure harness behavior on SDP tasks; do not assume benchmark rankings
    transfer to our workflows.

## References

- OpenAI, "Unrolling the Codex agent loop":
  https://openai.com/index/unrolling-the-codex-agent-loop/
- OpenAI, "Running Codex safely at OpenAI":
  https://openai.com/index/running-codex-safely/
- OpenAI API docs, "Using GPT-5.5":
  https://developers.openai.com/api/docs/guides/latest-model
- OpenAI Developers, "Codex web":
  https://developers.openai.com/codex/cloud
- Z.AI Developer Docs, "GLM-5.1":
  https://docs.z.ai/guides/llm/glm-5.1
- Kimi, "Kimi K2.6":
  https://www.kimi.com/ai-models/kimi-k2-6
- QwenLM/Qwen3.6:
  https://github.com/QwenLM/Qwen3.6
- AWS, "Qwen models on SageMaker JumpStart", 2026-05-04:
  https://aws.amazon.com/about-aws/whats-new/2026/05/qwen-models-on-sagemaker-jumpstart/
- Hugging Face, "DeepSeek-V4: a million-token context that agents can actually use":
  https://huggingface.co/blog/deepseekv4
- OpenCode docs, "Agents":
  https://opencode.ai/docs/agents/
- Pi docs:
  https://pi.dev/docs/latest
- Pi usage docs:
  https://pi.dev/docs/latest/usage
- Pi agents package:
  https://pi.dev/packages/pi-agents
- Pi agent flow package:
  https://pi.dev/packages/pi-agent-flow
- Bui, "Building Effective AI Coding Agents for the Terminal", arXiv:2603.05344:
  https://arxiv.org/abs/2603.05344
- McMillan, "Instruction Adherence in Coding Agent Configuration Files", arXiv:2605.10039:
  https://arxiv.org/abs/2605.10039
- Goel, "Security Risks in Tool-Enabled AI Agents", arXiv:2605.09721:
  https://arxiv.org/abs/2605.09721
- Stein, "How are AI agents used? Evidence from 177,000 MCP tools", arXiv:2603.23802:
  https://arxiv.org/abs/2603.23802

