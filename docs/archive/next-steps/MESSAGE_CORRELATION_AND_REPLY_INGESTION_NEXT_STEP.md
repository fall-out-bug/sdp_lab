# Message Correlation & Reply Ingestion — Next Step

Status: working next step
Date: 2026-03-22
Scope: close the loop between outbound feedback packets and inbound replies for control cards

## Goal

Build the next thin layer above feedback I/O so the control tower can:
- correlate outbound feedback requests with cards
- ingest normalized replies back into the right card
- resume orchestration with less manual glue

## Current baseline

Already implemented:
- internal feedback packet generation
- feedback export/import CLI helpers
- feedback apply/resume flow
- ready gate
- Beads bridge for ready cards

What is still missing is the practical reply-correlation seam.

## Next implementation target

Implement a provider-agnostic correlation and reply-ingestion layer:

1. define a normalized outbound message envelope with correlation fields
2. define a normalized inbound reply envelope with correlation fields
3. add CLI/helpers to create correlation-ready outbound payloads from cards
4. add CLI/helpers to ingest a reply envelope and route it to the correct card
5. wire reply ingestion into existing feedback resume flow
6. add tests

## Constraints

- do not couple to a specific messaging platform
- do not redesign control-store architecture
- keep orchestrator-centric ownership intact
- board remains visualization only
- prefer thin practical implementation over abstraction

## Desired outcome

The system should support a realistic loop:
- orchestrator emits a feedback packet for a card
- external delivery layer sends it
- reply comes back in normalized form with correlation data
- system maps reply to the correct card
- card resumes automatically

That closes the next major loop toward a real externalized orchestration system.
