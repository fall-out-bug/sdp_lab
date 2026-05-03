# Pi Harness Configuration for sdp_lab

This directory configures [Pi](https://pi.dev) as a first-class harness for the SDP platform.

## Structure

```
.pi/
  settings.json      # Pi settings: model, skills, prompts, extensions
  extensions/
    sdp.ts           # SDP integration: tools, commands, status, footer
  prompts/           # Generated command templates from sdp.manifest.yaml
  skills/            # Pi-specific agent skills (Agent Skills standard, <name>/SKILL.md)
    architect/       # Design & build
    deployer/        # Operations
    devops/          # CI/CD & infrastructure
    implementer/     # Execution & TDD
    orchestrator/    # Coordination
    planner/         # Scope & decomposition
    qa/              # Quality & test strategy
    reviewer/        # Code review
    security/        # Threat modeling
    spec-reviewer/   # Spec compliance
    sre/             # Reliability & observability
    tech-lead/       # Technical direction
```

## What It Does

### 1. Skills Discovery
Skills are loaded from `.pi/skills/` (canonical Agent Skills standard):

**Workflow skills** (from `.agents/skills/<name>/SKILL.md`, auto-discovered by Pi):
- `beads` — Task tracker integration
- `feature`, `build`, `ship`, `review` — Delivery loop
- `discovery`, `design` — Discovery phase
- `debug`, `bugfix`, `hotfix` — Incident response
- `protocol-consistency`, `go-modern` — Quality gates
- …and 20+ more

**Agent skills** (Pi-specific, `.pi/skills/<name>/SKILL.md`):
- `architect`, `planner`, `implementer` — Design & build
- `reviewer`, `qa`, `security` — Quality & review
- `devops`, `sre`, `deployer` — Operations
- `orchestrator`, `tech-lead`, `spec-reviewer` — Coordination

### 2. Prompt Templates
`.pi/prompts/` contains generated command templates:

- `/beads`, `/feature`, `/build`, `/review`, `/ship`
- `/debug`, `/deploy`, `/test`, `/vision`
- …mapped from `sdp.manifest.yaml`

### 3. Extension: `sdp.ts`
Registers custom tools and commands:

**Tools (LLM-callable):**
| Tool | Purpose |
|------|---------|
| `sdp` | Run SDP CLI (`scout`, `metrics`, `manifest`, `doctor`, …) |
| `bd` | Beads task tracker (`ready`, `show`, `update`, `create`) |
| `sdp_review` | Run `sdp-pi-review` code quality gates |
| `workgraph` | Compile or inspect `.sdp/workgraph.lock.json` |

**Commands (user-triggered):**
| Command | Action |
|---------|--------|
| `/ws` | Show ready workstreams |
| `/bd <args>` | Run beads command |
| `/review [scope]` | Run Pi review (`auto` default) |
| `/sdp <args>` | Run SDP CLI command |

**UI Enhancements:**
- Session start: shows ready workstream count + git branch
- Footer: git branch + `sdp_lab` label
- Status line: beads ready count + current branch
- Safety gate: blocks destructive `sdp reset`/`clean`/`destroy`

## Usage

From the repo root:

```bash
# Interactive mode with SDP context
pi

# Run with a specific skill
/skill:feature

# Check ready workstreams
/ws

# Run code review
/review

# Run SDP doctor
/sdp doctor

# Check beads status
/bd ready
```

## First-Time Setup

1. Ensure `sdp` CLI is built:
   ```bash
   go build -o bin/sdp ./cmd/sdp
   ```
2. Ensure `bd` (beads) is installed and authenticated.
3. Ensure `go` is available if you plan to use `sdp-pi-review` or `sdp-harness` (they auto-build via `go run`).

## Maintenance

When `sdp.manifest.yaml` changes, regenerate adapters:

```bash
sdp generate-adapters
```

Then verify Pi prompts in `.pi/prompts/` are updated.
