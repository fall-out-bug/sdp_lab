# SDP Maturity Matrix

> **Canonical source** for component maturity labels. Referenced by README, roadmap, and compliance docs.
> **Last updated:** 2026-04-26

## Maturity Levels

| Level | Label | Meaning | Graduation Criteria | Rollback Trigger |
|-------|-------|---------|---------------------|------------------|
| 3 | **GA** | Production-ready. Used by default in all workflows. Backward-compatible within major version. | 90-day burn-in in production; zero P0/P1 regressions; test coverage >= 80%; docs complete (reference + runbook). | 2+ P1 regressions in 30 days; documented design flaw requiring breaking change. |
| 2 | **Beta** | Functional for intended use case. API may change. Not recommended as sole workflow path. | Feature-complete for stated scope; test coverage >= 60%; reference doc exists; at least 1 non-author user has used successfully. | P0 regression in GA dependency; scope expansion requiring API redesign. |
| 1 | **Experimental** | Proof-of-concept or partial implementation. May have zero callers. Not guaranteed to compile in all configurations. | Compiles; basic happy-path test passes; documented intent in reference or plan doc. | No commits in 90 days; superseded by alternative implementation. |

## Component Matrix

### CLI Binaries (cmd/)

| Component | Maturity | Owner | Notes |
|-----------|----------|-------|-------|
| `sdp` | GA | platform | Main CLI entry point |
| `sdp-harness` | Experimental | platform | Needs LiveGateway (F106) |
| `sdp-orchestrate` | GA | platform | Feature-level orchestration |
| `sdp-orchestrate-daemon` | Beta | platform | Daemon variant |
| `sdp-guard` | GA | platform | Scope enforcement |
| `sdp-ci-loop` | GA | platform | CI feedback loop |
| `sdp-strataudit` | GA | platform | Strategic LLM audit |
| `sdp-eval` | Beta | platform | Eval runner |
| `sdp-evidence` | GA | platform | in-toto attestations |
| `sdp-doc-sync` | GA | platform | Doc link checker + sync |
| `sdp-beads-bridge` | GA | platform | Beads <-> SDP bridge |
| `sdp-control` | GA | platform | Control plane operations |
| `sdp-dispatch` | Beta | platform | Dispatch layer |
| `sdp-a2a` | Beta | platform | Agent-to-agent protocol |
| `sdp-gh-findings-sync` | GA | platform | GitHub findings -> Beads |
| `sdp-ready` | GA | platform | Ready check (pre-flight) |
| `sdp-up` | GA | platform | Project bootstrap |
| `sdp-ws-verdict-validate` | GA | platform | Workstream verdict validation |
| `sdp-omc-guard` | Beta | platform | OMC scope guard |
| `sdp-protocol-check` | GA | platform | Protocol validation |
| `sdp-healthcheck` | GA | platform | Health check endpoint |
| `sdp-mcp` | Beta | platform | MCP server |
| `sdp-session-audit` | Beta | platform | Session audit trail |

### Internal Packages (internal/)

#### Discovery

| Package | Maturity | Owner | Notes |
|---------|----------|-------|-------|
| `internal/discovery` | GA | discovery | 4-phase LLM pipeline |
| `internal/architect` | GA | discovery | C4 analysis + runtime coupling |
| `internal/strataudit` | GA | discovery | Strategic audit, provider-neutral runtime |

#### Delivery

| Package | Maturity | Owner | Notes |
|---------|----------|-------|-------|
| `internal/agentloop` | Experimental | platform | Needs LiveGateway (F106-WS01) |
| `internal/executor` | GA | platform | ServeBridge -> DispatchAndRun |
| `internal/orchestrate` | GA | platform | Feature-level phase orchestration |
| `internal/gate` | GA | platform | Gate filesystem |
| `internal/evidence` | GA | platform | in-toto attestations, EvidenceStore |
| `internal/deploy` | Beta | platform | Docker Compose wrapper |
| `internal/guard` | GA | platform | Scope enforcement logic |
| `internal/ciloop` | GA | platform | CI feedback loop logic |
| `internal/eval` | Beta | platform | Evaluation framework |
| `internal/harness` | GA | platform | Harness lifecycle |
| `internal/kernel` | GA | platform | Execution kernel primitives |

#### Infrastructure

| Package | Maturity | Owner | Notes |
|---------|----------|-------|-------|
| `internal/modelgateway` | Experimental | platform | Library ready, 0 production callers |
| `internal/control` | GA | platform | FeatureCard store |
| `internal/beads` | GA | platform | Beads/Dolt integration |
| `internal/session` | GA | platform | Session store (SQLite WAL) |
| `internal/a2a` | Beta | platform | Agent-to-agent protocol |
| `internal/dispatch` | Beta | platform | Dispatch routing |
| `internal/monitor` | Beta | platform | Metrics/monitoring |
| `internal/docsync` | GA | platform | Doc sync + link checker |
| `internal/workstream` | GA | platform | Workstream parsing + validation |
| `internal/router` | GA | platform | Request routing |
| `internal/runtime` | GA | platform | Runtime utilities |
| `internal/profile` | Beta | platform | Profile management |
| `internal/policy` | Beta | platform | Policy engine |
| `internal/prompt` | GA | platform | Prompt construction |
| `internal/augmentation` | Beta | platform | Context augmentation |
| `internal/bridge` | GA | platform | Bridge abstractions |
| `internal/cli` | GA | platform | CLI helpers |
| `internal/sdputil` | GA | platform | SDP utilities |
| `internal/gitutil` | GA | platform | Git utilities |
| `internal/executil` | GA | platform | Exec utilities |
| `internal/verify` | GA | platform | Verification tools |
| `internal/tower` | Beta | platform | Tower orchestration layer |

#### Not Yet Active (0 callers)

| Package | Maturity | Owner | Notes |
|---------|----------|-------|-------|
| `internal/authz` | Experimental | platform | Written, 0 callers |
| `internal/planner` | Experimental | platform | Written, 0 callers |

### Skills

| Skill | Maturity | Owner | Notes |
|-------|----------|-------|-------|
| `skills/llm-council` | GA | discovery | Multi-model deliberation |
| `skills/strataudit` | GA | discovery | Portable strategy audit |
| `skills/agent-dispatching` | GA | platform | Agent dispatch protocol |
| `AGENTS.md` | GA | platform | Agent instructions |

### CI Gates

| Gate | Maturity | Owner | Notes |
|------|----------|-------|-------|
| build-test | GA | platform | go build + test + lint |
| snapshot-test | GA | platform | Snapshot diff detection |
| push-protection | GA | platform | Prevent direct push to main |
| architect-tests | GA | platform | Architecture regression tests |
| contract-compat | GA | platform | Contract compatibility tests |
| evidence-gate | GA | platform | Evidence schema validation |
| scope-gate | GA | platform | WS scope enforcement |
| protocol-compliance | GA | platform | Contract + snapshot validation |
| consistency-gate | GA | platform | Repo consistency + protocol check |
| coverage-gate | GA | platform | Baseline delta enforcement (blocking) |
| policy-gate | GA | platform | OPA policy evaluation (advisory by default) |
| auto-attestation | GA | platform | Sigstore-signed attestation generation |
| required-checks | GA | platform | Final merge gate (validates all 12) |

## Summary

| Maturity | Count | Percentage |
|----------|-------|------------|
| GA | 57 | 73% |
| Beta | 15 | 19% |
| Experimental | 7 | 8% |
| **Total** | **79** | **100%** |

## Change Log

| Date | Change |
|------|--------|
| 2026-04-26 | Initial matrix created from components.md audit |
| 2026-04-26 | F079-01: Added missing CLI binaries (sdp-healthcheck, sdp-mcp, sdp-session-audit), updated counts |

---

*Source: [components.md](./components.md) for component catalog, [ROADMAP.md](../roadmap/ROADMAP.md) for product direction, [ci-gates-map.md](./ci-gates-map.md) for CI gate details.*
