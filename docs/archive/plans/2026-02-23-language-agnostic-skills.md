# Language-Agnostic Skills

> **Status:** Research complete
> **Date:** 2026-02-23
> **Goal:** Remove Go-specific hardcoding from universal SDP skills

---

## Overview

### Problem

SDP Manifesto claims language-agnostic protocol, but 5 skills + AGENTS.md have ~25 Go-specific references in CRITICAL execution paths. The recent `@oneshot` v8.0.0 and `@build` v7.0.0 rewrites made this worse.

### Root Cause

Design docs were written for THIS project (Go), not for SDP as universal protocol. Go commands leaked from "project config" into "universal protocol".

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Boundary | Skills = universal protocol; AGENTS.md = project-specific config |
| Abstraction | Skills reference "quality gates (AGENTS.md)" instead of `go test` |
| Future | `sdp test`/`sdp build` CLI wrappers when CLI matures |
| Migration | Replace hardcoded Go commands in 5 skills with AGENTS.md references |

---

## 1. Protocol vs Implementation Boundary

> **Experts:** Sam Newman (bounded context), Rob Pike (simplicity), Andrej Karpathy (adaptive workflows)

### Two-Layer Architecture

| Layer | Contains | Language-specific? | Example |
|-------|----------|--------------------|---------|
| `sdp/prompts/skills/` | Universal workflow | NO | "Run quality gates" |
| `AGENTS.md` | Project config | YES | `go test ./...` |

Skills say WHAT to do. AGENTS.md says HOW (with what tools).

### Why This Works

The LLM reads AGENTS.md at session start. It already knows the project's toolchain. When a skill says "run quality gates", the LLM substitutes `go test ./...` for Go projects, `npm test` for Node projects, etc. No detection logic needed.

---

## 2. Migration: Go References to Remove

| Skill | Current (Go-specific) | New (universal) |
|-------|----------------------|-----------------|
| `@build` | `go test -coverprofile` | "Run test suite with coverage (see AGENTS.md for toolchain)" |
| `@build` | `golangci-lint run` | "Run linter (see AGENTS.md)" |
| `@build` | `wc -l *.go` | "Check LOC for source files" |
| `@tdd` | `go test ./...` | "Run test suite" |
| `@tdd` | `go vet ./...` | "Run static analysis" |
| `@bugfix` | `go test/build/vet` | "Run quality gates (AGENTS.md)" |
| `@oneshot` | `go test ./...` | "Run quality gates (AGENTS.md)" |
| `@deploy` | `go test ./... -q` | "Run quality gates (AGENTS.md)" |

### What Stays Go-Specific

- `AGENTS.md` Quality Gates section — this IS project config
- Go examples in inline comments — fine as illustration
- `go.mod`, `internal/` directory references — project structure, not protocol

---

## 3. Future: `sdp` CLI Abstraction

When `sdp` CLI matures (F014+):

```bash
sdp test          # detects go.mod → go test ./...
sdp build         # detects go.mod → go build ./...
sdp lint          # detects go.mod → golangci-lint run
sdp coverage      # detects go.mod → go test -coverprofile + go tool cover
```

Detection order: `go.mod` → `package.json` → `Cargo.toml` → `pyproject.toml` → `Makefile`

Skills would then say `sdp test` instead of "run quality gates". But this is a future feature, not needed now.

---

## Implementation Plan

### Phase 1: Now (this session)

- [ ] Replace Go commands in `@build/SKILL.md` with AGENTS.md references
- [ ] Replace Go commands in `@tdd/SKILL.md` with AGENTS.md references
- [ ] Replace Go commands in `@bugfix/SKILL.md` with AGENTS.md references
- [ ] Replace Go commands in `@oneshot/SKILL.md` with AGENTS.md references
- [ ] Replace Go commands in `@deploy/SKILL.md` with AGENTS.md references

### Phase 2: Future (F014+ CLI)

- [ ] Add `sdp test` / `sdp build` / `sdp lint` with auto-detection
- [ ] Update skills to use `sdp` CLI commands

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Go-specific refs in skills | ~25 | 0 (in CRITICAL paths) |
| Skills usable for non-Go projects | 0 | All |
| AGENTS.md as single toolchain source | partial | complete |
