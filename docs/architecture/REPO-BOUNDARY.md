# sdp_lab vs sdp: Repository Boundary

> **Purpose:** Clarify which binaries and artifacts live where. Prevents confusion for contributors.

## Overview

| Repo | Location | Visibility | Role |
|------|----------|------------|------|
| **sdp_lab** | This repo (root) | Private | Lab: orchestration, CI loop, evidence CLI, research |
| **sdp** | Submodule at `sdp/` | Public | Protocol: schemas, prompts, sdp-plugin CLI |

**Rule:** All development happens in sdp_lab. Touch `sdp/` only when publishing protocol artifacts.
**Canonical remote:** `sdp/` must come from `https://github.com/fall-out-bug/sdp.git`, not from a local relative clone path.
**Historical labels:** old plans, workstreams, and bead IDs may still say `sdp_dev`. In current docs, that means this same root repo.

---

## Component → Repo → Publish?

| Component | Repo | Binary/Artifact | Published to sdp? |
|-----------|------|-----------------|-------------------|
| **sdp-orchestrate** | sdp_lab | `bin/sdp-orchestrate` | No — lab only |
| **sdp-ci-loop** | sdp_lab | `bin/sdp-ci-loop` | No — lab only |
| **sdp-evidence** | sdp_lab | `bin/sdp-evidence` | No — lab only (may release separately later) |
| **sdp-guard** | sdp_lab | `bin/sdp-guard` | No — lab only |
| **sdp-eval** | sdp_lab | `bin/sdp-eval` | No — lab only |
| **sdp** (quality, apply, build, verify) | sdp | `sdp` CLI | Yes — via sdp-plugin in protocol repo |
| **Schemas** | sdp | `sdp/schema/*.json` | Yes |
| **Prompts/Skills** | sdp | `sdp/prompts/skills/*` | Yes |
| **Hooks** | sdp | `sdp/hooks/` | Yes |

---

## Build Commands

**sdp_lab (lab tooling):**
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

See AGENTS.md Step 8 for the publish workflow.
