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

**quick (30s):** `sdp scout` only (cached if <1hr old). Output: project card with language, LOC, deps, health. For "quick look", "what is this?".
**standard (5-15 min):** scout + architect + metrics. Output: architecture diagram, health report, risks, tech debt. For "analyze this", before feature work.
**deep (15-30 min):** standard + spec + index build. Output: complete knowledge base in `.sdp/manifest.md` + `.sdp/index.db`. For "full analysis", docs generation.

## Artifact Freshness

Cached artifacts skip re-computation: scout <1hr, architect/metrics <24hr, spec/index <7 days. Prevents redundant work while ensuring freshness.

## Partial Mode Selection

Question focus triggers targeted tools: "how healthy"→scout+metrics, "architecture"→scout+architect, "API contracts"→scout+spec, "where is X"→scout+index, "full picture"→all tools.

Time budget overrides: "quick look"→scout only regardless of focus.

## Architect Output

Required: new codebase, >10K LOC, architecture questions. Optional: quick mode, <5K LOC, health-only focus, recent scout.json.

## Output Format

**Quick:** Plain text project card (language, LOC, health, risk). **Standard:** Markdown report (architecture, components, health, risks). **Deep:** Complete knowledge base (`.sdp/manifest.md` context primer, `.sdp/index.db` for queries, `.sdp/spec.json` specs).

## Input Expectations

- **Path:** Defaults to `.`
- **Questions:** Optional focus (architecture, health, dependencies, docs)
- **Mode:** Optional — auto-detected from query depth, or explicit `--depth quick|standard|deep`

## Legacy Aliases

@scout → quick mode, @architect/@metrics → standard mode, @spec/@index → deep mode, @landscape → standard/deep mode

## Embedded Practices

**@guard:** Automatic quality gate via hooks. NOT a user-facing skill.

## Artifacts Created

`.sdp/scout.json`, `.sdp/architect.json`, `.sdp/metrics.json`, `.sdp/spec.json`, `.sdp/manifest.md`, `.sdp/index.db` (deep)

## Acceptance Boundaries

NOT for: code generation (@build), bug fixing (@fix), review (@review), deployment (@operate)
