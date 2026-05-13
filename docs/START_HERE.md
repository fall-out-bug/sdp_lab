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
- [reference/product-surface.md](reference/product-surface.md) is the maturity
  boundary: stable Toolkit, Operator Mode, lab-only, and research surfaces.
