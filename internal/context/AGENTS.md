# SDP Runtime Assumptions — sdp-context-core v1

**Package**: `internal/context/` (package `sdpcontext`)  
**Contract**: v1.0.0  
**Module**: `sdp_dev`

## Context Objects

**Primary output**: `RepoMap` — complete repository structure for SDP runtime.  
**Secondary**: `DiffResult` — change-aware retrieval for incremental processing.

## Environment Requirements

- **Git**: required for `DiffRetriever` operations; binary must be in PATH
- **File system**: POSIX-compatible access for `RepoMapper` traversal
- **No network**: all operations are local-only

## Configuration

- **PromptBudget**: configurable per model (gpt-4, claude-opus, etc.)
- **Cache hashing**: deterministic SHA-256; inputs sorted before hashing
- **Layer injection**: order preserved; truncation removes oldest layers first

## Dependencies

**No internal SDP dependencies** — this is a leaf substrate package.  
Imports only: Go stdlib (`context`, `time`).

## Version Control

- **Current support**: git-only
- **Git required**: `DiffRetriever` assumes git repository
- **Future extensibility**: interface contract allows VCS-agnostic implementations

## File System Assumptions

- **POSIX paths**: `/` as separator; relative paths from repo root
- **Stat access**: requires read access for file metadata (mtime, size)
- **Symbolic links**: followed during repo mapping
- **Binary files**: detected and skipped during line counting

## Token Budget Heuristics

- **Chars per token**: 4 (ASCII average)
- **Context reservation**: 15-25% of total budget for runtime injection
- **Layer ordering**: array order preserved during prompt construction
