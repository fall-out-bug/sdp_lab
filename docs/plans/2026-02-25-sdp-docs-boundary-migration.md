# sdp ↔ sdp_dev Documentation Boundary Migration

**Date:** 2026-02-25  
**Goal:** sdp = protocol only; sdp_dev = implementation, roadmap, workstreams  
**Status:** DONE

## Boundary (per AGENTS.md)

| sdp (protocol, public) | sdp_dev (lab, private) |
|------------------------|------------------------|
| prompts, JSON schemas, hooks | Go code, K8s, roadmap, research |
| PROTOCOL.md, CLI_REFERENCE | workstreams, plans, reviews |
| README, CLAUDE.md, PRODUCT_VISION | AGENTS.md, beads mapping |

## Migration Executed

### Moved sdp → sdp_dev

- docs/vision, runbooks, design, integrations, beads-integration
- docs/AGENT_TEAMS.md, INCIDENT_RESPONSE.md, SLOS.md
- docs/reference (full set), docs/decisions (ADRs)
- docs/attestation (kept coding-workflow-v1 in sdp)
- specs/ → docs/specs/

### Removed from sdp

- docs/workstreams/, roadmap/, plans/, reviews/
- AGENTS.md, AGENT_HANDOFF.md
- docs/decisions/, docs/reference (replaced with minimal)
- specs/, docs/github-integration, releases, workflow-decision

### Kept in sdp (protocol)

- README, CLAUDE.md, PRODUCT_VISION, CONTRIBUTING, SECURITY, CHANGELOG
- docs/PROTOCOL.md, CLI_REFERENCE.md, MANIFESTO.md
- docs/reference/ (minimal: README, PRINCIPLES, GLOSSARY, MODELS, *-spec)
- docs/concepts/, compliance/, intent/, attestation/
- prompts/, schema/, templates/
