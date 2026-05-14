# Start Here

Status: friendly onboarding entrypoint

This page is the shortest honest route into SDP. Pick the row that matches what
you want to do now.

## Choose Your Path

| You want to... | First useful result | Start here |
|---|---|---|
| Try SDP in your own repo | Install the repo-local `sdp` CLI, verify adapters, and run read-only inspection commands | [QUICKSTART.md](QUICKSTART.md) |
| Understand the available CLI tools | See which commands are stable Toolkit, Operator Mode, or lab/research tooling | [reference/commands.md](reference/commands.md) |
| Understand skills and agents | Map human intents to the real manifest skills and agent prompts | [reference/agent-skill-entry-map.md](reference/agent-skill-entry-map.md) |
| Contribute to `sdp_lab` itself | Load repo rules, find the owning workstream, and use Beads-backed execution | [reference/project-map.md](reference/project-map.md) |
| Operate a full SDP delivery loop | Use Beads, workstreams, early PRs, review findings, QA/UAT, and delivery evidence | [reference/canonical-happy-path.md](reference/canonical-happy-path.md) |

## Common Reader Lenses

| Reader | What to prove in the first session | Stop if this is not true |
|---|---|---|
| Developer new to vibe-coding | SDP installed without global PATH confusion, produced read-only repo facts, and made one safe next command obvious | You cannot explain what files were added or which command is safe to run next |
| Experienced vibecoder | SDP added a workstream/evidence contract around the harness you already use | You only want faster raw edits and do not care about reviewable evidence |
| Multi-harness team | Static adapter coverage and runtime readiness are separated before rollout | The docs imply Claude, OpenCode, Cursor, Codex, and Pi are equivalent workers |
| CTO or architect | You can decide after a short pilot what risk SDP reduces and what is still unproven | The result is only a tool tour with no stop/go criteria |
| Agent entering cold | You can name the repo purpose, stable surfaces, limits, and next allowed action | You cannot cite the source doc for a capability or maturity claim |

## First Run For External Users

Use this path when you want a useful result without adopting the full operator
workflow:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash

./.sdp/bin/sdp manifest validate
./.sdp/bin/sdp doctor adapters
./.sdp/bin/sdp scout --format text .
./.sdp/bin/sdp metrics --format markdown .
./.sdp/bin/sdp index build --format text .
./.sdp/bin/sdp spec --format text .
./.sdp/bin/sdp bootstrap --dry-run --mode brownfield .
```

Use `./.sdp/bin/sdp` until you have verified that `command -v sdp` points to the
repo-local binary. Older global binaries can make real commands look missing.

## First Harness Command After Install

After install and adapter validation, start with a harness-native entrypoint.
These are **not** `sdp` CLI commands. They go through installed adapters:

- `/build` is Claude-style harness command syntax.
- `@build` is the shared harness command syntax used by OpenCode/Cursor.

| Harness | Primary form | Runtime status |
|---|---|---|
| Claude Code | `claude -p "/build 00-XXX-YY"` | Stable primary |
| OpenCode | `opencode run --dir "$PWD" --agent implementer "@build 00-XXX-YY"` | Experimental; requires `--agent implementer` |
| Cursor | `agent -p "@build 00-XXX-YY"` | Secondary validator only; primary dispatch untested |
| Codex | Use repo instructions plus generated `.codex/` prompts | Adapter surface present; autonomous dispatch is not the primary path |
| Pi | Use generated `.pi/` skills/prompts for review or validation | Experimental; autonomous dispatch pending bundle evidence |

`00-XXX-YY` is a **workstream ID** (`00-XXX-YY`) from the operator backlog.
It is not required for external users using local delivery only.

For operator mode inside `sdp_lab`, find real IDs with:

```bash
bd ready
```

```bash
# Claude Code
claude -p "/build 00-XXX-YY"

# OpenCode
opencode run --dir "$PWD" --agent implementer "@build 00-XXX-YY"

# Cursor
agent -p "@build 00-XXX-YY"

# Codex / Pi
# Use generated prompts/skills as validation or manual-assist surfaces until
# runtime dispatch evidence is recorded.
```

Cursor is currently **secondary** and **untested for SDP dispatch**.
Use it for independent validation only, not as your primary automation path.

**OpenCode warning:** non-interactive runs require `--agent implementer`; without it,
`opencode run ...` may return success without making edits.

If you do not yet have a workstream ID, run local delivery without operator
queue first:

```bash
./.sdp/bin/sdp build "what you want to change" --dry-run --format text
```

That keeps you productive before operator mode and Beads-backed workstream
assignment are in place.

If harness dispatch is not available yet in your environment, do not skip onboarding:
use the manual checklists in [reference/FALLBACK_MODE.md](reference/FALLBACK_MODE.md),
then retry with one supported harness.

## What Is Stable Enough To Try First

Stable first-run Toolkit surface:

- `scout`
- `metrics`
- `index build`
- `spec`
- `bootstrap --dry-run`
- `init`
- `manifest`
- `generate-adapters`
- `doctor`

Second-run value after the first cache or setup exists:

- `index query`
- `index find`
- `index deps`
- `index stats`
- `architect`

Operator and lab tooling exists, but it is not the first-run promise. Do not
start with `sdp-harness`, `sdp-up`, Beads, PR gates, deploy, or K8s paths unless
you are explicitly operating or developing SDP.

## CTO Pilot Decision Packet

A useful 30-minute pilot should answer four questions:

- What repository risk did SDP expose that raw agent chat would likely hide?
- Which evidence did it produce: commands, reports, generated adapters, or a dry-run plan?
- Which harness is safe as the primary path today, and which ones are only secondary?
- What remains `not_assessed` or `cannot_verify` after the pilot?

Do not treat a successful install as adoption proof. Adoption proof is a
reviewable change path with scope, evidence, and a clear finding loop.

## Agent Answer Contract

When a developer asks an agent "what is this repo and what can you do with it?",
use this shape:

1. `sdp_lab` is the public workspace for SDP: Go code, orchestration, adapters,
   evals, roadmap, and protocol artifacts.
2. Stable first-run value is repo inspection and safe adapter setup:
   `scout`, `metrics`, `index build`, `spec`, `bootstrap --dry-run`, `init`,
   `manifest`, `generate-adapters`, and `doctor`.
3. Operator Mode adds Beads-backed workstreams, early PRs, gates, findings, and
   QA/UAT for teams that want governed agent delivery.
4. Claude Code is the primary harness today; OpenCode is experimental with
   `--agent implementer`; Cursor, Codex, and Pi are not equivalent primary
   workers yet.
5. Lab/runtime surfaces such as `sdp-harness`, K8s, swarm, and model gateway
   work are not first-run promises.

## For Agents Entering This Repo

This repo has its own execution rules. Use this order:

1. [../AGENTS.md](../AGENTS.md)
2. [reference/project-map.md](reference/project-map.md)
3. [reference/canonical-happy-path.md](reference/canonical-happy-path.md)
4. [workstreams/INDEX.md](workstreams/INDEX.md)
5. `bd ready`

Work in `sdp_lab` is owned by a feature, workstream, and Beads issue. If you
cannot name those, you are still in orientation, not execution.

## Truth Rules

- `sdp.manifest.yaml` is the machine-readable inventory for generated skills,
  commands, and agents.
- `prompts/skills/` and `prompts/agents/` contain canonical prompt bodies.
- `.agents/skills/` contains runtime aliases and harness-specific discovery
  stubs.
- `cmd/sdp/main.go` and `go run ./cmd/sdp --help` are the source of truth for
  the current repo-local CLI.
- `.cursorrules`, `.opencode/`, `.claude/`, `.pi/`, and `.codex/` are generated
  adapters. Edit only canonical sources; regenerate adapters instead of editing
  these directories directly.
- [reference/product-surface.md](reference/product-surface.md) is the maturity
  boundary: stable Toolkit, Operator Mode, lab-only, and research surfaces.
