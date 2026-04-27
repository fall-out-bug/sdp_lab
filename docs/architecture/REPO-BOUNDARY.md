# sdp_lab vs sdp: Repository Boundary

> **Purpose:** Clarify which binaries and artifacts live where and when to publish. Prevents confusion for contributors.

## Overview

| Repo | Location | Visibility | Role |
|------|----------|------------|------|
| **sdp_lab** | This repo (root) | Public | Primary workspace: Go code, orchestration, CI loop, evidence CLI, research, protocol artifacts (native) |
| **sdp** | Distilled repo (`fall-out-bug/sdp`) | Public | Distribution/mirror of selected protocol artifacts published from sdp_lab |

**Rule:** all development happens in sdp_lab. Publish to the distilled `sdp` repo via `scripts/sdp-publish.sh` when selected protocol artifacts need distribution.
**Historical labels:** old plans, workstreams, and bead IDs may still say `sdp_dev` (Go module path migrated to `github.com/fall-out-bug/sdp_lab` in F150-03). In current docs, that means this same root repo.

---

## Component -> Publish?

| Component | Binary/Artifact | Published to sdp? |
|-----------|-----------------|-------------------|
| **sdp-orchestrate** | `bin/sdp-orchestrate` | No -- lab only |
| **sdp-ci-loop** | `bin/sdp-ci-loop` | No -- lab only |
| **sdp-evidence** | `bin/sdp-evidence` | No -- lab only (may release separately later) |
| **sdp-guard** | `bin/sdp-guard` | No -- lab only |
| **sdp-eval** | `bin/sdp-eval` | No -- lab only |
| **sdp** (main CLI) | `cmd/sdp` | Primary source lives in sdp_lab; selected distribution docs/artifacts may be mirrored to `sdp` |
| **Schemas** | `schema/*.json` (native) | Yes -- via `sdp-publish.sh` |
| **Agents** | `prompts/agents/*.md` (native) | Yes -- via `sdp-publish.sh` |
| **Prompts/Skills** | `prompts/skills/*/SKILL.md` + `.agents/skills/*.md` (native, dual format pending F138-03) | Yes -- via `sdp-publish.sh` |
| **Hooks** | `.claude/hooks/`, `scripts/hooks/` (native) | Yes -- via `sdp-publish.sh` |
| **Harness entrypoints** | `.cursorrules`, `.cursor/*.md`, `.codex/*.md`, `.opencode/hooks/*`, `.opencode/README.md` | Yes -- via `sdp-publish.sh` |
| **Fallback docs** | `docs/reference/FALLBACK_MODE.md`, `prompts/commands.yml` | Yes -- via `sdp-publish.sh` |

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

**sdp CLI:**

```bash
go build -tags "sqlite_fts5" -o "$(go env GOPATH)/bin/sdp" ./cmd/sdp
sdp init --help
```

The optional `sdp/` directory is a gitignored checkout of the distilled repo for publish checks, not the source of the CLI in this workspace.

---

## When to Publish to sdp

- Schema changes (evidence, intent, ws-verdict)
- New or updated prompts/skills
- Hook changes
- Harness entrypoint changes
- Fallback or command-mapping docs
- Quickstart or CLI reference updates

Run `scripts/sdp-publish.sh` after merge to `main` only when the distilled repo needs the update. See [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md) for the full publish workflow.
