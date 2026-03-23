# UX-First CLI Surface — Next Step

Status: working next step
Date: 2026-03-23
Scope: first action-oriented UX slice for SDP colleagues

## Goal

Turn the current CLI from mostly raw output into a more useful operator/colleague surface.

## Do now

1. improve `sdp doctor control` output:
   - compact summary first
   - group checks by severity or debt type
   - keep item-level details concise

2. improve `sdp board show` output:
   - default to a readable action-oriented summary
   - show state buckets and top items
   - surface a recommended next action

3. improve `sdp attention` if needed:
   - one-screen digest
   - obvious priority order

4. keep the implementation thin:
   - no web UI
   - no big rendering framework
   - no contract rewrite

## Constraints

- preserve current command surfaces unless a tiny compatibility adjustment is clearly worth it
- prefer additive rendering helpers over broad rewrites
- optimize for operator clarity, not generic prettiness
- one-screen usefulness beats exhaustive output

## Desired outcome

A colleague should be able to run the main SDP CLI surfaces and immediately understand:
- what needs attention
- what is blocked or waiting
- what is ready
- what to do next
