# Control Store Skeleton

Status: initial implementation
Date: 2026-03-22

## What exists now

A first file-backed control-store skeleton has been added:

- `internal/control/`
  - file-backed `FeatureCard` write model
  - automatic intake artifact creation
  - project board snapshot derivation
  - portfolio snapshot derivation
  - feedback packet generation
  - feedback application and resume
- `cmd/sdp-control/`
  - `card-create`
  - `card-clarify`
  - `card-needs-input`
  - `card-ready`
  - `card-park`
  - `card-execute`
  - `card-feedback`
  - `card-resume`
  - `board-build`
  - `board-show`

## Current scope

This is intentionally small.
It proves the storage/projection model before deeper orchestration and before any dashboard implementation.

## Current behavior

### `sdp-control card-create`
Creates:
- YAML card in `.sdp/control/projects/<project>/cards/`
- Markdown intake artifact in `.sdp/control/projects/<project>/intake/`

### `sdp-control card-execute`
Executes a ready card:
- Creates feature-level Beads issue with card details
- Persists Beads ID in `linked_beads_ids`
- Changes card status to `executing`
- Auto-builds project and portfolio snapshots

### `sdp-control card-feedback`
Generates a concise feedback packet from a card in `needs_input` or `blocked` state:
- Extracts all feedback-related fields: `needs_feedback_from`, `feedback_request`, `decision_required`, `author_update`, `admin_action_required`, `blocking_reasons`
- Returns structured JSON packet for human/admin review
- Fails for cards not in feedback-waiting state

### `sdp-control card-resume`
Applies feedback answers back onto the card and resumes lifecycle:
- Accepts answers, decisions, updates, and admin actions
- Resolves blocking reasons
- Clears feedback request fields
- Moves card to `clarifying` (default) or `ready` (if card meets ready gate)
- Validates ready gate before moving to `ready`

### `sdp-control board-build`
Builds:
- project snapshot if `--project` is set
- portfolio snapshot otherwise

### `sdp-control board-show`
Currently rebuilds and prints the relevant snapshot.

## Next implementation steps

1. attach Beads bridge operations
2. wire orchestrator actions onto store more directly
3. add richer status views / UI later
4. add explicit feedback-message emission helpers
5. add resume/reconciliation flows for long-running orchestration
   / UI later
