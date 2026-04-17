---
name: build
description: Create new features, components, or systems — from idea to implementation with TDD.
version: 1.0.0
tags:
  - feature
  - implementation
  - tdd
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# build

## Purpose

Create something new: features, components, systems, prototypes. TDD by default.
Absorbs: @feature, @idea, @design, @ux, @vision, @oneshot, @prototype.

## When to Use

Implementing features, creating designs, prototyping, user-facing work with acceptance criteria.

## Modes

**idea:** Problem → design doc (no implementation). Output: design with requirements, approach, tradeoffs. For: "design a...", "how should we...", planning phase.
**feature:** Idea → design → implement → test → PR (full cycle). Output: complete feature with tests. For: "implement...", "build...", "add...".
**prototype:** Skip design, build quickly, mark experimental. Output: working prototype + TODO list. For: "prototype...", "quick mock...", "proof of concept".

## Routing Rules

Scope based on: (1) Request type: "Design..."→idea, "Implement..."→feature, "Prototype..."→prototype.
(2) Available context: design doc exists? skip to implementation. (3) User preference.
(4) Complexity: single button→prototype, new auth system→feature.

## Input Expectations

- **Requirement:** Clear description of what to build
- **Context:** Design docs (optional), codebase context (auto-detected)
- **Mode:** Optional — auto-detected from request language
- **Acceptance criteria:** Optional — generated if not provided

## Legacy Aliases

@feature→feature mode, @idea/@design/@ux/@vision→idea mode, @oneshot→feature mode, @prototype→prototype mode

## Embedded Practices

**@tdd:** Test-first DEFAULT workflow. Write failing test → verify fails → implement → verify passes → refactor. NEVER skipped for production code. NOT a separate skill — embedded in @build and @fix.

**@guard:** Pre-commit quality gate runs automatically via hooks. NOT invoked manually.

**@go-modern:** Language style convention (documented in CLAUDE.md). Applied automatically during implementation. NOT a skill entry point.

## Artifacts Created

**idea:** Design document (`docs/design/*.md`)
**feature:** Implementation code with TDD tests, docs, PR
**prototype:** Working code, [PROTOTYPE] label, TODO list for productionization

## Acceptance Boundaries

NOT for: understanding code (@understand), fixing bugs (@fix), review (@review), deployment (@operate)

Quality gates: all tests pass, follows conventions, docs updated, PR ready
