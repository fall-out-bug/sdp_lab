# sdp_lab vs sdp: Repository Boundary

> **Purpose:** Clarify which binaries and artifacts live where and when to publish. Prevents confusion for contributors.

## Overview

| Repo | Location | Visibility | Role |
|------|----------|------------|------|
| **sdp_lab** | This repo (root) | Private | Lab: orchestration, CI loop, evidence CLI, research, protocol artifacts (native) |
| **sdp** | Public mirror (`fall-out-bug/sdp`) | Public | Downstream mirror of protocol artifacts published from sdp_lab |

**Rule:** All development happens in sdp_lab. Publish to the public `sdp` repo via `scripts/sdp-publish.sh` when protocol artifacts change.
**Historical labels:** old plans, workstreams, and bead IDs may still say `sdp_dev`. In current docs, that means this same root repo.

---

## Component -> Publish?

| Component | Binary/Artifact | Published to sdp? |
|-----------|-----------------|-------------------|
| **sdp-orchestrate** | `bin/sdp-orchestrate` | No -- lab only |
| **sdp-ci-loop** | `bin/sdp-ci-loop` | No -- lab only |
| **sdp-evidence** | `bin/sdp-evidence` | No -- lab only (may release separately later) |
| **sdp-guard** | `bin/sdp-guard` | No -- lab only |
| **sdp-eval** | `bin/sdp-eval` | No -- lab only |
| **sdp** (quality, apply, build, verify) | `sdp` CLI | Yes -- via `scripts/sdp-publish.sh` |
| **Schemas** | `sdp/schema/*.json` | Yes |
| **Prompts/Skills** | `sdp/prompts/skills/*` | Yes |
| **Hooks** | `sdp/hooks/` | Yes -- native directory in sdp_lab |

---

## Build Commands

**sdp_lab (lab tooling):**
```bash
make build-sdp-orchestrate   # -> bin/sdp-orchestrate
make build-sdp-guard         # -> bin/sdp-guard
make build-sdp-eval          # -> bin/sdp-eval
make build-sdp-ci-loop       # -> bin/sdp-ci-loop
make build-sdp-evidence      # -> bin/sdp-evidence
```

**sdp (protocol CLI):**
```bash
cd sdp/sdp-plugin && go build -o sdp ./cmd/sdp   # -> sdp CLI (quality, apply, build, verify)
```

---

## When to Publish to sdp

- Schema changes (evidence, intent, ws-verdict)
- New or updated prompts/skills
- Hook changes
- sdp-plugin bug fixes or features that affect protocol consumers
- Quickstart or CLI reference updates

Run `scripts/sdp-publish.sh` after merge to `main`. See [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md) for the full publish workflow.
