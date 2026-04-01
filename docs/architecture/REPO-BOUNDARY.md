# sdp vs sdp_dev: Repository Boundary

> **Purpose:** Clarify which binaries and artifacts live where. Prevents confusion for contributors.

## Overview

| Repo | Location | Visibility | Role |
|------|----------|------------|------|
| **sdp_dev** | This repo (root) | Private | Lab: orchestration, CI loop, evidence CLI, research |
| **sdp** | Submodule at `sdp/` | Public | Protocol: schemas, prompts, sdp-plugin CLI |

**Rule:** All development happens in sdp_dev. Touch `sdp/` only when publishing protocol artifacts.
**Canonical remote:** `sdp/` must come from `https://github.com/fall-out-bug/sdp.git`, not from a local relative clone path.

---

## Component → Repo → Publish?

| Component | Repo | Binary/Artifact | Published to sdp? |
|-----------|------|-----------------|-------------------|
| **sdp-orchestrate** | sdp_dev | `bin/sdp-orchestrate` | No — lab only |
| **sdp-ci-loop** | sdp_dev | `bin/sdp-ci-loop` | No — lab only |
| **sdp-evidence** | sdp_dev | `bin/sdp-evidence` | No — lab only (may release separately later) |
| **sdp-guard** | sdp_dev | `bin/sdp-guard` | No — lab only |
| **sdp-eval** | sdp_dev | `bin/sdp-eval` | No — lab only |
| **sdp** (quality, apply, build, verify) | sdp | `sdp` CLI | Yes — via sdp-plugin in protocol repo |
| **Schemas** | sdp | `sdp/schema/*.json` | Yes |
| **Prompts/Skills** | sdp | `sdp/prompts/skills/*` | Yes |
| **Hooks** | sdp | `sdp/hooks/` | Yes |

---

## Build Commands

**sdp_dev (lab tooling):**
```bash
make build-sdp-orchestrate   # → bin/sdp-orchestrate
make build-sdp-guard         # → bin/sdp-guard
make build-sdp-eval          # → bin/sdp-eval
make build-sdp-ci-loop       # → bin/sdp-ci-loop
make build-sdp-evidence      # → bin/sdp-evidence
```

**sdp (protocol CLI):**
```bash
cd sdp/sdp-plugin && go build -o sdp ./cmd/sdp   # → sdp CLI (quality, apply, build, verify)
```

---

## When to Publish to sdp

- Schema changes (evidence, intent, ws-verdict)
- New or updated prompts/skills
- Hook changes
- sdp-plugin bug fixes or features that affect protocol consumers

See AGENTS.md Step 5 for the publish workflow.
