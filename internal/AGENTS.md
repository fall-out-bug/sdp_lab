# internal — Agent Contract

## Scope

This subtree owns first-party Go packages for SDP runtime, CLI backends,
orchestration, evidence, model routing, policy, dispatch, and evaluation.

## Contract

Package APIs must be readable, bounded, and testable. Stable package contracts
belong in the nearest package-local `AGENTS.md` when a package has runtime,
security, provider, evidence, or extraction assumptions.

## Dependencies

Prefer stdlib and existing internal substrate packages. New cross-package
dependencies must improve UX or DX and must not create hidden coupling between
lab-only, experimental, and stable release surfaces.

## Runtime Assumptions

Most packages run locally inside a git worktree. Network, provider credentials,
filesystem writes, subprocess execution, or long-running background behavior must
be explicit in the package-local contract or code comments near the boundary.

## Local Rules

- Read `docs/reference/go-patterns.md` before Go edits.
- Preserve package boundaries from `docs/reference/product-surface.md` and
  `docs/reference/maturity-matrix.md`.
- Add or update package-local `AGENTS.md` for packages with non-obvious runtime
  assumptions.
- Do not use `context.Background()` in long-running or request-scoped paths when
  a caller context is available.
- Do not treat denylist-based subprocess filtering as a security boundary.
