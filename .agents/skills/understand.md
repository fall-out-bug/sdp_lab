---
name: understand
description: Explore and synthesize codebase knowledge — architecture, health, metrics, and documentation.
version: 1.0.0
tags:
  - discovery
  - architecture
  - analysis
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# understand

## Purpose

Synthesize complete codebase picture: structure, health, architecture, docs.
Absorbs: @scout, @architect, @metrics, @spec, @landscape, @index query.

## When to Use

First time with codebase, need architecture/health check, before feature work, generating docs.

## Modes

**quick (30s):** Read cached `.sdp/scout.json`. Output: project card (language, LOC, deps, health).
**standard (5-15 min):** Scout + architect + metrics. Output: architecture diagram, health report, risks.
**deep (15-30 min):** Standard + spec generation + index build. Output: knowledge base in `.sdp/manifest.md`.

## Routing Rules

Tool composition based on: (1) Cache available? skip scout. (2) User question focus.
(3) Time budget. (4) Project state (Go? Python? → language-specific analysis).

## Input Expectations

- **Path:** Defaults to `.`
- **Questions:** Optional focus (architecture, health, dependencies, docs)
- **Mode:** Optional — auto-detected from query depth

## Legacy Aliases

@scout→quick, @architect/@metrics→standard, @spec/@index→deep, @landscape→standard/deep

## Artifacts Created

`.sdp/scout.json`, `.sdp/architect.json`, `.sdp/metrics.json`, `.sdp/manifest.md`, `.sdp/index.db` (deep)

## Acceptance Boundaries

NOT for: code generation (@build), bug fixing (@fix), review (@review), deployment (@operate)
