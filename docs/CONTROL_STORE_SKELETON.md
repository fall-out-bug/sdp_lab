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
  - feedback packet generation and export
  - feedback answer import and application
  - feedback-based resume flow
- `cmd/sdp-control/`
  - `card-create`
  - `card-clarify`
  - `card-needs-input`
  - `card-ready`
  - `card-park`
  - `card-execute`
  - `card-feedback` - generates feedback packet to stdout
  - `card-feedback-export` - exports feedback packet to file for external messaging
  - `card-resume` - applies feedback via CLI flags
  - `card-resume-import` - imports feedback answer from file
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

### `sdp-control card-feedback-export`
Exports a feedback packet to a file for external messaging:
- Same packet content as `card-feedback`
- Writes JSON to specified path
- Provider-agnostic format for integration with any messaging system

### `sdp-control card-resume`
Applies feedback answers back onto the card and resumes lifecycle:
- Accepts answers, decisions, updates, and admin actions via CLI flags
- Resolves blocking reasons
- Clears feedback request fields
- Moves card to `clarifying` (default) or `ready` (if card meets ready gate)
- Validates ready gate before moving to `ready`

### `sdp-control card-resume-import`
Imports a feedback answer from a file and applies it:
- Reads normalized answer JSON from file
- Applies same logic as `card-resume`
- Enables external systems to provide answers via file

### Feedback I/O integration

The feedback I/O layer is now complete:
- **Outbound**: `card-feedback-export` emits normalized `FeedbackPacket` to file
- **Inbound**: `card-resume-import` ingests normalized `FeedbackAnswer` from file
- Both commands are provider-agnostic and preserve orchestrator ownership

Typical external integration flow:
1. Card enters `needs_input` or `blocked` state
2. External system calls `card-feedback-export` to get packet
3. External system sends packet via chosen messaging provider
4. Human/admin provides answers
5. External system creates `FeedbackAnswer` JSON file
6. External system calls `card-resume-import` with answer file
7. Card resumes automatically to `clarifying` or `ready`

### `sdp-control board-build`
Builds:
- project snapshot if `--project` is set
- portfolio snapshot otherwise

### `sdp-control board-show`
Currently rebuilds and prints of relevant snapshot.

### `sdp-control doctor control`
Runs hygiene checks across all control-store cards. Validates:
- cards without intake artifacts
- intake artifact files that don't exist
- ready cards missing ready-gate fields
- executing cards without `linked_beads_ids`
- needs_input cards without `feedback_request` or `decision_required`

Returns a concise operator-facing report and exits with non-zero status when checks fail.

## Next implementation steps

1. attach Beads bridge operations
2. wire orchestrator actions onto store more directly
3. add richer status views / UI later
4. integrate with external messaging providers (Slack, email, etc.) on top of file-based I/O
5. add resume/reconciliation flows for long-running orchestration
