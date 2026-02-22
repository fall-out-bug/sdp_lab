# FR-012: OpenRouter Integration + Agent Model Assignment

Priority: P1
Effort: 3d
Dependencies: FR-011 (registry has projects with model config)

## Problem

Models are hardcoded (`glm-5`, `glm-4.7`). OpenRouter is already supported (`internal/llm/openrouter.go`), but:
- No dynamic model assignment by roles
- No support for multiple providers simultaneously
- Model fallback chain is static
- No cost tracking / budget enforcement

## Existing Code

| File | Status |
|------|--------|
| `internal/llm/openrouter.go` | ✅ OpenRouterClient with Chat/Complete |
| `internal/policy/model_chain.go` | ⚠️ Hardcoded: glm-5 → glm-4.7 → escalated |
| `internal/policy/provider.go` | ⚠️ Minimal routing logic |
| `internal/agent/model_selector.go` | ⚠️ Static per-role defaults |
| `internal/adapter/policy_gate.go` | ✅ PreDispatchModelAllowlist |
| `deploy/k8s/control/policy-service.yaml` | ⚠️ ConfigMap: allowlist = "glm-5,glm-4.7" |
| `docs/MODEL_POLICY.md` | ✅ Documented |

## OpenRouter Model Catalog (February 2026)

Top models for coding based on real usage on OpenRouter:

| # | Model | Provider | Price (in/out per 1M) | Context | SWE-Bench | Best For |
|---|-------|----------|-----------------------|---------|-----------|----------|
| 1 | **MiniMax M2.5** | minimax | $0.30 / $1.10 | 197K | 80.2% | Coding #1, planning, token-efficient |
| 2 | **Kimi K2.5** | moonshotai | $0.23 / $3.00 | 262K | strong | Visual coding, agentic swarm |
| 3 | **GLM-5** | z-ai | $0.30 / $2.55 | 205K | strong | System design, agent workflows |
| 4 | **Claude Opus 4.6** | anthropic | $5.00 / $25.00 | 1M | frontier | Complex refactors, long-horizon agents |
| 5 | **Gemini 3 Flash** | google | $0.50 / $3.00 | 1M | good | Fast agentic, multi-turn |
| 6 | **GPT-5.2** | openai | $1.75 / $14.00 | 400K | frontier | General reasoning, tool calling |
| 7 | **Trinity Large** | arcee-ai | FREE | 131K | decent | Free fallback, open-weight |
| 8 | **Claude Sonnet 4.6** | anthropic | $3.00 / $15.00 | 1M | strong | Iterative dev, project management |
| 9 | **Claude Sonnet 4.5** | anthropic | $3.00 / $15.00 | 1M | strong | Agentic workflows, tool orchestration |
| 10 | **DeepSeek V3.2** | deepseek | $0.26 / $0.38 | large | ~90% GPT-5.1 | Ultra-cheap coding, 100x cheaper than Opus |
| 11 | **Devstral 2** | mistral | $0.05 / $0.22 | 256K | 73%+ | Cheapest agentic coding |
| 12 | **Grok 4.1 Fast** | x-ai | mid | large | strong | Fast reasoning |
| 13 | **Qwen 3.5 Plus** | qwen | $0.40 / $2.40 | large | strong | Hybrid architecture, strong codegen |

### Model Selection Strategy

**Three tiers:**
- **Tier 1 (Frontier)**: Claude Opus 4.6, GPT-5.2 — for critical decisions (orchestrator, evaluator, telemetry-analyzer). Expensive, but maximum quality.
- **Tier 2 (Workhorse)**: GLM-5, MiniMax M2.5, Kimi K2.5, Claude Sonnet 4.6 — main workhorse models. Price/quality balance.
- **Tier 3 (Economy)**: DeepSeek V3.2, Devstral 2, Trinity Large (free) — for bulk operations, fallback. Minimum cost.

## Design

### Dynamic Model Config via ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: model-policy
  namespace: sdp-control
data:
  policy.yaml: |
    providers:
      openrouter:
        api_key_secret: openrouter-api-key
        base_url: https://openrouter.ai/api/v1
      glm:
        api_key_secret: glm-api-key
        base_url: https://api.z.ai/api/coding/paas/v4
      openai:
        api_key_secret: openai-api-key
        base_url: https://api.openai.com/v1
      anthropic:
        api_key_secret: anthropic-api-key
        base_url: https://api.anthropic.com/v1

    roles:
      # ── Core agent roles (Tier 2: quality coding) ──
      analyst:
        primary: glm-5                                    # $0.30/$2.55 — system design expert
        fallback: openrouter/anthropic/claude-sonnet-4.6  # $3/$15 — strong analysis
        economy: openrouter/deepseek/deepseek-v3.2        # $0.26/$0.38 — 90% quality, 1/100 cost
      coder:
        primary: glm-4.7                                   # GLM coding plan
        fallback: openrouter/moonshotai/kimi-k2.5          # $0.23/$3 — visual coding, agentic
        economy: openrouter/deepseek/deepseek-v3.2         # ultra-cheap coding
      reviewer:
        primary: glm-5                                     # thorough review
        fallback: openrouter/anthropic/claude-sonnet-4.6   # strong reasoning
        economy: openrouter/minimax/minimax-m2.5           # $0.30/$1.10 — #1 programming
      reviewer-security:
        primary: openrouter/anthropic/claude-sonnet-4.5    # $3/$15 — security-focused
        fallback: glm-5
        economy: openrouter/deepseek/deepseek-v3.2
      reviewer-dx:
        primary: glm-5
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/minimax/minimax-m2.5
      builder:
        primary: openrouter/minimax/minimax-m2.5           # $0.30/$1.10 — SWE-bench 80.2%
        fallback: openrouter/moonshotai/kimi-k2.5
        economy: openrouter/mistralai/devstral-2512        # $0.05/$0.22 — cheapest
      verifier:
        primary: glm-4.7
        fallback: openrouter/deepseek/deepseek-v3.2        # cheap but accurate
        economy: openrouter/mistralai/devstral-2512

      # ── Orchestration & autonomy (Tier 1-2: critical decisions) ──
      orchestrator:
        primary: glm-5
        fallback: openrouter/anthropic/claude-opus-4.6     # $5/$25 — long-horizon agent
        economy: openrouter/anthropic/claude-sonnet-4.6
      autonomy-worker:
        primary: glm-5
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/google/gemini-3-flash-preview  # $0.50/$3 — fast agentic
      brain-gateway:
        primary: glm-5
        fallback: openrouter/anthropic/claude-opus-4.6
        economy: openrouter/openai/gpt-5.2                 # $1.75/$14

      # ── Analysis & intake (Tier 2: analytical) ──
      retro-agent:
        primary: glm-5
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/google/gemini-3-flash-preview
      self-improve-agent:
        primary: glm-5
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/deepseek/deepseek-v3.2
      evaluator:
        primary: openrouter/anthropic/claude-opus-4.6      # needs frontier reasoning
        fallback: glm-5
        economy: openrouter/anthropic/claude-sonnet-4.5
      intake-analyzer:
        primary: openrouter/google/gemini-3-flash-preview  # fast, 1M context
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/deepseek/deepseek-v3.2

      # ── New agents FR-014..016 (Tier 1-2) ──
      telemetry-analyzer:
        primary: openrouter/anthropic/claude-opus-4.6      # deep pattern analysis
        fallback: glm-5
        economy: openrouter/anthropic/claude-sonnet-4.5
      feature-orchestrator:
        primary: glm-5
        fallback: openrouter/anthropic/claude-sonnet-4.6
        economy: openrouter/google/gemini-3-flash-preview
      cicd-agent:
        primary: glm-4.7                                    # simple tasks, cheap
        fallback: openrouter/deepseek/deepseek-v3.2
        economy: openrouter/mistralai/devstral-2512

      # ── Alternative runtime ──
      openclaw-agent:
        primary: openrouter/minimax/minimax-m2.5            # #1 programming
        fallback: openrouter/moonshotai/kimi-k2.5
        economy: openrouter/arcee-ai/trinity-large-preview:free  # FREE

    allowlist:
      # Tier 1 — Frontier
      - openrouter/anthropic/claude-opus-4.6      # $5/$25
      - openrouter/openai/gpt-5.2                 # $1.75/$14
      # Tier 2 — Workhorse
      - glm-5                                      # $0.30/$2.55
      - glm-4.7                                    # GLM coding plan
      - openrouter/anthropic/claude-sonnet-4.6     # $3/$15
      - openrouter/anthropic/claude-sonnet-4.5     # $3/$15
      - openrouter/minimax/minimax-m2.5            # $0.30/$1.10
      - openrouter/moonshotai/kimi-k2.5            # $0.23/$3
      - openrouter/google/gemini-3-flash-preview   # $0.50/$3
      - openrouter/qwen/qwen-3.5-plus             # $0.40/$2.40
      # Tier 3 — Economy
      - openrouter/deepseek/deepseek-v3.2          # $0.26/$0.38
      - openrouter/mistralai/devstral-2512         # $0.05/$0.22
      - openrouter/arcee-ai/trinity-large-preview:free  # FREE

    budget:
      daily_limit_usd: 50.0
      per_run_limit_usd: 5.0
      alert_threshold_pct: 80

    # Cost optimization: when budget > 80%, auto-switch to economy tier
    cost_optimization:
      auto_downgrade_at_pct: 80
      never_downgrade_roles:
        - evaluator
        - telemetry-analyzer
```

### Model Selector Refactor

Replace `internal/agent/model_selector.go` and `internal/policy/model_chain.go`:

```go
type ModelSelector interface {
    SelectForRole(role string, riskClass string) (Model, error)
    NextFallback(current Model) (Model, bool)
    TrackUsage(model Model, tokens int, latencyMs int64)
}
```

### OpenRouter Enhanced Client

Extend `internal/llm/openrouter.go`:
- Streaming support
- Usage tracking (tokens, cost)
- Rate limiting per model
- Timeout per provider

## Acceptance Criteria

- [ ] ConfigMap-based model policy (hot-reload)
- [ ] Per-role model assignment (20 roles: analyst, coder, reviewer, reviewer-security, reviewer-dx, builder, verifier, orchestrator, autonomy-worker, brain-gateway, retro-agent, self-improve-agent, evaluator, intake-analyzer, telemetry-analyzer, feature-orchestrator, cicd-agent, openclaw-agent)
- [ ] OpenRouter routing with provider prefix
- [ ] Model fallback chain from ConfigMap
- [ ] Usage tracking per run (tokens, cost estimate)
- [ ] PolicyGate validates against dynamic allowlist
- [ ] Budget enforcement (per-run, daily)
- [ ] Observability: model selection events in telemetry
