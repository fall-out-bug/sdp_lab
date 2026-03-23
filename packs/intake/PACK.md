# Intake Stage Pack

Status: v1
Date: 2026-03-23
Scope: raw request → `FeatureCard` in the control store

## Purpose

The intake stage exists to capture a request cheaply without losing truth.

This stage should:
- preserve the original request as `raw_request`
- create the card in the control store
- create the intake artifact automatically
- make the card visible on project/portfolio snapshots

This stage should **not** try to fully shape or execute the work.

---

## Entry / exit

### Entry
A request exists but is not yet represented as a control-store card.

### Exit
A card exists in `inbox` with:
- immutable intake truth in `raw_request`
- an intake artifact under `.sdp/control/projects/<project>/intake/`
- board snapshots rebuilt

Next stage: `../shaping/PACK.md`

---

## Canonical commands

### Create a card
```bash
sdp card create --project <project-id> --title "<title>" --raw "<original request>"
```

What this stage relies on today:
- card YAML written to `.sdp/control/projects/<project>/cards/`
- intake artifact created automatically
- project and portfolio snapshots updated automatically

### Check hygiene
```bash
sdp doctor control
```

### View the current board state
```bash
sdp board show --project <project-id>
```

---

## Stage rules

### 1. Intake must stay cheap
Do not require full scope, target repo, or execution plan just to capture an idea.

### 2. `raw_request` is intake truth
Do not overwrite the original ask to make it cleaner.
Normalization belongs in shaping, not intake.

### 3. The card starts as a board/control object
The board is the visualization surface for the card lifecycle.
It is not the execution system.

### 4. Trace starts here
If work matters enough to enter SDP control flow, it should enter through a card and intake artifact rather than appearing later as an orphan execution task.

---

## What “good intake” looks like

Good intake usually means:
- title is short and operator-readable
- `raw_request` preserves the actual ask
- project routing is correct
- intake artifact exists
- snapshots reflect the new card

That is enough.
More detail can wait for shaping.

---

## Common patterns

### Quick idea capture
```bash
sdp card create --project myapp --title "Add dark mode" --raw "Users keep asking for a dark mode option in settings."
```

### Capture from an external message or note
Use the original text as `--raw`, even if it is messy.
The goal is preservation first, normalization later.

### Intake verification pass
```bash
sdp doctor control
sdp board show --project myapp
```

---

## Anti-patterns

### Over-shaping at intake
Bad:
- demanding full implementation detail before creating the card
- stuffing execution decisions into the intake step

Good:
- create the card quickly
- shape it later if it survives attention triage

### Editing away the original ask
Bad:
- replacing `raw_request` with a polished paraphrase

Good:
- keep the original ask intact
- add normalized meaning later via `sdp card clarify`

### Skipping the control object
Bad:
- creating downstream execution work without a card

Good:
- create the card first
- bridge to execution only after shaping

---

## Related docs

- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/CONTROL_STORE_SKELETON.md`
- `docs/ARTIFACT_PROVENANCE_INTAKE.md`
- `../README.md`
