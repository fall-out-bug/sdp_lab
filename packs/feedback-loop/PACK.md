# Feedback Loop Stage Pack

Status: v1
Date: 2026-03-23
Scope: cross-cutting feedback / resume loop for cards in `needs_input` or `blocked`

## Purpose

The feedback loop stage handles moments where the control tower needs human/admin input before it can continue responsibly.

This stage should:
- generate a normalized feedback packet from the card state
- let external systems deliver that packet to humans/admins
- accept a normalized answer back in
- resume the card lifecycle without giving up orchestrator ownership

This is a cross-cutting pack.
It is used from shaping, execution, and later review/release flows.

---

## Entry / exit

### Entry
The card is in `needs_input` or `blocked` and the missing information/action is explicit.

### Exit
A feedback answer has been applied and the card moves back into a live lifecycle state, typically:
- `clarifying`, or
- `ready` if the ready gate now passes

---

## Canonical commands

### Generate a feedback packet
```bash
sdp card feedback --project <project-id> --id <card-id>
```

### Export a feedback packet for external delivery
```bash
sdp card feedback-export --project <project-id> --id <card-id> --output <feedback.json>
```

### Apply feedback inline
```bash
sdp card resume \
  --project <project-id> \
  --id <card-id> \
  --answers "answer 1;answer 2" \
  --decisions "decision 1" \
  --updates "update 1" \
  --admin-actions "action 1" \
  --unblock "reason 1;reason 2" \
  --target-status clarifying
```

### Import a normalized feedback answer
```bash
sdp card resume-import --project <project-id> --id <card-id> --input <answer.json>
```

### Hygiene check
```bash
sdp doctor control
```

---

## Loop rules

### 1. Make the missing thing explicit
A good feedback loop starts with a card that clearly states what is missing:
- unanswered questions
- decisions
- author updates
- admin actions
- unblock reasons

### 2. Keep transport provider-agnostic
Telegram, Slack, email, or anything else can deliver the packet.
The control tower should still own the normalized structure and resume logic.

### 3. Resume through the control-store command path
Do not manually mutate card state in ad hoc ways after a human reply.
Use `resume` / `resume-import` so the state transition stays explicit and reproducible.

### 4. Feedback is a loop, not a side channel
The point is not just sending a message.
The point is closing the loop back into the lifecycle.

---

## Practical guidance

### From shaping
Use the feedback loop when the card cannot become `ready` without human/admin clarification or approval.

### From execution
Use the feedback loop when execution returns blocked conditions, open decisions, or missing context that cannot be resolved autonomously.

### Resume target
Default resumption is usually `clarifying`.
Only target `ready` when the card now actually passes the ready gate.

---

## Common patterns

### Export to an external messaging layer
```bash
sdp card feedback-export --project myapp --id feature-001 --output /tmp/feedback.json
```

External tooling can then deliver `/tmp/feedback.json` however it wants.

### Apply feedback directly from operator input
```bash
sdp card resume \
  --project myapp \
  --id feature-001 \
  --answers "enterprise users only" \
  --decisions "ship behind feature flag" \
  --target-status clarifying
```

### Import normalized answer from an external system
```bash
sdp card resume-import --project myapp --id feature-001 --input /tmp/answer.json
```

---

## Anti-patterns

### Free-text side-channeling
Bad:
- humans answer in chat
- operators manually reinterpret and patch the card by hand

Good:
- normalize the answer
- import it through the resume flow

### Hidden blockers
Bad:
- card is marked `needs_input` but nobody can tell what input is required

Good:
- make feedback request / decision / unblock payload explicit on the card

### Manual state-jumping
Bad:
- flipping a card back to `ready` without applying the actual answer payload

Good:
- apply the answer first
- let the lifecycle resume from there

---

## Related docs

- `docs/CONTROL_STORE_SKELETON.md`
- `docs/ORCHESTRATOR_ACTIONS_AND_FEEDBACK_CONTRACT.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `../shaping/PACK.md`
- `../execution-bridge/PACK.md`
- `../README.md`
