# SDP Surface Contract Normalization — Delta Design

**Date:** 2026-04-20
**Status:** Proposed
**Reframes:** previous monolithic baseline at the same path into delta features `F137`–`F139`
**Related:** [2026-04-20-mini-harness-orchestrator-design.md](2026-04-20-mini-harness-orchestrator-design.md), [2026-04-20-sweep-v2-design.md](2026-04-20-sweep-v2-design.md), [2026-04-13-sdp-toolkit-vision-design.md](2026-04-13-sdp-toolkit-vision-design.md), [2026-04-13-sdp-skill-architecture-design.md](2026-04-13-sdp-skill-architecture-design.md), [2026-04-13-sdp-mcp-design.md](2026-04-13-sdp-mcp-design.md)

---

## 1. Problem Statement

The repo already shipped the first useful versions of the toolkit surface:

- `F125` shipped the intent-model transition
- `F126` shipped the initial MCP server

That is not the same as having one coherent, machine-readable surface contract.

Current gaps:

1. **CLI truth is fragmented.** `cmd/sdp/` and `cmd/sdp-*` still act like parallel entrypoints. A human can guess the right binary. A downstream orchestrator cannot.
2. **Skill catalog is noisy.** The repo still carries a large mixed catalog of active, partial, duplicate, and legacy skill files. This is bad UX for operators and bad DX for automation.
3. **MCP parity is not contract-driven.** `F126` shipped a working server, but tool/resource/prompt discovery is still too hand-wired. That invites drift from the CLI and skill truth.
4. **Docs drift is still cheap.** There is no single parity rule that forces `CLI registry -> reference docs`, `active skill set -> catalog`, and `catalog/registry -> MCP exposure`.
5. **Downstream design work is blocked on surface stability.** Mini-harness orchestration and sweep need a stable contract. They should not depend on an informal reading of `cmd/` and `.agents/skills/`.

This is why the old one-epic framing is wrong. The repo does not need a new root epic that reopens shipped work. It needs delta features that normalize the surface above already shipped `F125` and `F126`.

## 2. Why Existing F125 and F126 Are Not Enough

### `F125` remains shipped

`F125` owns the intent model. It already established the product decision that the user-facing workflow is intent-routed rather than a flat skill zoo.

What `F125` does **not** own:

- historical cleanup of every legacy skill file
- machine-readable catalog packaging
- deprecation headers and alias policy
- harness-facing docs parity across the remaining skill surface

Those are normalization tasks, not a rewrite of the intent model.

### `F126` remains shipped

`F126` owns the initial MCP server. It already proved that SDP can expose tools, resources, and prompts over MCP.

What `F126` does **not** own:

- registry-driven CLI discovery as the canonical upstream contract
- automatic parity between CLI registry and MCP tools
- automatic parity between skill catalog and MCP prompts/resources
- version/hash surfaces that let downstream automation reason about compatibility

Those are extension tasks, not evidence that `F126` was incomplete or not shipped.

### `F134` remains the phase CLI owner

`F134` owns the phase command surface and runtime phase semantics. This design must not absorb `sdp plan`, `sdp review`, or `sdp eval` semantics into a new generic CLI epic.

## 3. Feature Split

This design creates three delta features, not one umbrella epic.

| Feature | ID | Owns | Does not own |
|---|---|---|---|
| CLI Surface Normalization | `F137` | unified `sdp` entrypoint, registry/discovery, help/json/version contract, shim/deprecation policy | skill merge, MCP parity logic, phase semantics |
| Skill Catalog Normalization | `F138` | skill inventory cleanup, canonical catalog artifact, deprecation map, docs/harness sync | new intent model, bootstrap generation, MCP server internals |
| MCP Contract Parity | `F139` | registry/catalog-driven MCP parity, schema/version/hash surface, handshake validation | the original shipped `F126` server scope, skill rationalization itself |

### Why this split is the least wrong

- `F137` gives downstream automation a stable CLI contract.
- `F138` isolates UX/DX cleanup from registry refactor risk.
- `F139` extends shipped MCP instead of rewriting roadmap history.

### No umbrella tracker

Do **not** create a separate "framework normalization master epic".

That would create a second status layer with no operational value and blur ownership across three distinct execution lanes.

## 4. Ownership Boundaries

These boundaries are explicit and must stay explicit in roadmap, workstreams, and beads.

- `F125` remains the shipped owner of the intent model.
- `F126` remains the shipped owner of the initial MCP server.
- `F134` remains the owner of phase commands and runtime phase semantics.
- `F130` remains a downstream consumer of normalized outputs for harness config generation.
- `F135` and mini-harness orchestration depend on the normalized CLI contract but are not part of this design.
- `F136` peer memory may extend MCP later, but it is not a driver for this normalization lane.

## 5. Per-Feature Workstreams

### F137 — CLI Surface Normalization

**Priority:** `P1`
**Primary downstream:** mini-harness, sweep, any machine caller that needs stable discovery

| WS | Title | Outcome |
|---|---|---|
| `00-137-01` | Command inventory + contract freeze | authoritative inventory of `cmd/sdp` and `cmd/sdp-*`, with keep/shim/retire decisions |
| `00-137-02` | Registry core + discovery contract | canonical registry, `sdp help --json`, and version/hash metadata |
| `00-137-03` | Command migration onto registry | high-value commands moved under the registry without changing product behavior |
| `00-137-04` | Shim wrappers + deprecation warnings | thin wrappers for legacy binaries, with explicit stderr warnings and grace policy |
| `00-137-05` | CLI reference + parity gate | reference docs and lint rules that keep registry and docs aligned |

**DAG:** `00-137-01 -> 00-137-02 -> {00-137-03, 00-137-04} -> 00-137-05`

### F138 — Skill Catalog Normalization

**Priority:** `P1`
**Primary downstream:** operators, harness docs, skill consumers, future bootstrap generation

| WS | Title | Outcome |
|---|---|---|
| `00-138-01` | Skill inventory + canonical merge map | authoritative keep/merge/deprecate/remove map for current skill files |
| `00-138-02` | Catalog artifact generation | machine-readable `skills/index.json` source of truth |
| `00-138-03` | Canonical skill consolidation | active skill surface reduced to the intended catalog with explicit legacy treatment |
| `00-138-04` | Harness/docs sync + catalog lint | docs and harness-facing command surfaces follow the same catalog truth |

**DAG:** `00-138-01 -> {00-138-02, 00-138-03} -> 00-138-04`

### F139 — MCP Contract Parity

**Priority:** `P1`
**Primary downstream:** MCP clients, discovery tooling, cross-harness verification

| WS | Title | Outcome |
|---|---|---|
| `00-139-01` | CLI-to-MCP mapping contract | explicit rules for mapping registry and catalog truth into MCP surfaces |
| `00-139-02` | Auto-generated MCP tool exposure | tool registration derived from CLI registry rather than manual duplication |
| `00-139-03` | Prompt/resource parity | prompt/resource exposure follows the same truth as `F125` and `F138` |
| `00-139-04` | Handshake validation + reference docs | end-to-end verification plus durable docs for the parity surface |

**DAG:** `00-139-01 -> {00-139-02, 00-139-03} -> 00-139-04`

## 6. Beads Structure

### Required issue shape

- `3` feature-level beads of type `epic`
- `13` workstream-level beads of type `task`

### Parent/child model

- each `F137` / `F138` / `F139` epic owns only its own leaf tasks
- later findings are created as `bug` or `task` with `discovered-from:<leaf-id>`

### Cross-feature dependency model

- `00-139-02` depends on `00-139-01` and `00-137-03`
- `00-139-03` depends on `00-139-01` and `00-138-03`
- `00-139-04` depends on `00-139-02` and `00-139-03`

`F138` should generally follow `F137`, but it is not a blanket hard block. The dependency should live at the leaf level where it is real.

## 7. Roadmap and Index Impact

The roadmap and workstream index must add a new lane:

`Phase Surface Contract Normalization`

That lane contains only:

- `F137` CLI Surface Normalization
- `F138` Skill Catalog Normalization
- `F139` MCP Contract Parity

What must stay unchanged:

- `F125` remains shipped with follow-up tail only
- `F126` remains shipped
- historical toolkit docs remain historical

The new lane must explicitly state that these are delta features on top of shipped `F125` and `F126`.

## 8. Dependencies and Sequencing

### Primary sequence

`F137 -> F138 -> F139`

This is the recommended sequence because it minimizes churn:

1. stabilize CLI truth
2. normalize catalog and docs against that truth
3. expose parity into MCP against stable CLI and catalog inputs

### Downstream dependencies

- mini-harness has a **hard dependency** on `F137`, especially `00-137-02` and `00-137-03`
- sweep has a **hard dependency** on `F137` and a **soft/downstream dependency** on `F139`
- `F130` consumes normalized catalog outputs later but is not blocked on the whole lane

## 9. Acceptance Criteria Per Feature

### F137

- [ ] a single CLI registry is the source of truth for documented commands in scope
- [ ] `sdp help --json` returns a machine-readable discovery surface
- [ ] `sdp version` includes compatibility metadata needed by downstream automation
- [ ] legacy binaries in scope are either wrapped by shims or explicitly declared out of scope
- [ ] CLI reference docs and doc lint can detect registry/doc drift

### F138

- [ ] active skill files have an authoritative keep/merge/deprecate map
- [ ] `skills/index.json` exists as machine-readable truth
- [ ] deprecated skill names are handled explicitly instead of lingering silently
- [ ] `AGENTS.md`, harness-facing commands, and reference docs stop disagreeing about the active catalog

### F139

- [ ] MCP tool exposure is derived from CLI truth rather than handwritten duplication
- [ ] MCP prompts/resources match the post-normalization skill catalog and intent surface
- [ ] the parity surface is versioned and testable
- [ ] end-to-end handshake validation and reference docs exist for the normalized MCP contract

## 10. Non-Goals

- reopening or rewriting shipped `F125` as if intent routing had not shipped
- reopening or rewriting shipped `F126` as if MCP had not shipped
- absorbing `F134` phase command ownership into a generic CLI normalization track
- rewriting command business logic for feature behavior changes unrelated to surface normalization
- introducing an umbrella epic that duplicates the three-feature execution model
