# Review and Delivery Trace — Next Step

Status: working next step
Date: 2026-03-23
Scope: first thin review/delivery/rollback visibility slice

## Goal

Extend the control tower past execution result so review and delivery outcomes become visible in the card and board surfaces.

## Do now

1. Add a thin set of review/delivery/rollback fields to the card and schemas.

Suggested first fields:
- `review_state`
- `review_summary`
- `review_ref`
- `delivery_state`
- `delivery_target`
- `delivery_summary`
- `delivery_ref`
- `delivered_at`
- `rollback_ref`
- `rollback_summary`
- `followup_refs`

2. Update real existing paths where useful:
- result ingest for review-oriented outcomes
- card detail / board renderers

3. Add a thin explicit CLI path for recording delivery outcomes if needed.
Suggested shape:
- `sdp card deliver --project <id> --id <card-id> ...`

4. Keep the first slice honest:
- no fake deploy history
- no fake rollback automation
- write review/delivery data only when a real command/result provides it

## Constraints

- thin slice only
- no deploy engine
- no release automation framework
- no event-history rewrite
- no schema explosion

## Desired outcome

After this slice, a colleague should be able to see on the card/board:
- whether the card passed or failed review
- whether it was delivered
- whether it rolled back
- whether a follow-up artifact exists
