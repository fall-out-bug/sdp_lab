# CLI Naming Policy

Status: canonical policy
Date: 2026-03-22
Scope: command naming for SDP tooling

## Goal

Keep SDP command surfaces coherent as the system grows.

## Canonical rule

**`sdp` is the root command.**

New functionality should branch from `sdp`, not from a growing family of separate binaries like `sdp-control`, `sdp-ready`, etc.

## Implications

### Preferred shape
- `sdp card ...`
- `sdp board ...`
- `sdp doctor ...`
- `sdp dispatch ...`
- `sdp result ...`

### Avoid as long-term canon
- `sdp-control ...`
- ad-hoc sibling binaries for each subsystem

## Transitional rule

If a temporary implementation binary already exists (for example `sdp-control`), it may remain as a **compatibility shim** during migration.

But:
- docs should point to `sdp ...` as canonical
- new surface area should not keep expanding under the old temporary binary name

## Migration status

**COMPLETE**: The control-tower command surface has been migrated to `sdp` canonical root.

Current command mappings:
- `sdp-control card-create` → `sdp card create`
- `sdp-control card-clarify` → `sdp card clarify`
- `sdp-control card-needs-input` → `sdp card needs-input`
- `sdp-control card-ready` → `sdp card ready`
- `sdp-control card-park` → `sdp card park`
- `sdp-control card-execute` → `sdp card execute`
- `sdp-control card-feedback` → `sdp card feedback`
- `sdp-control card-feedback-export` → `sdp card feedback-export`
- `sdp-control card-message-export` → `sdp card message-export`
- `sdp-control card-resume` → `sdp card resume`
- `sdp-control card-resume-import` → `sdp card resume-import`
- `sdp-control card-reply-ingest` → `sdp card reply-ingest`
- `sdp-control board-build` → `sdp board build`
- `sdp-control board-show` → `sdp board show`
- `sdp-control doctor control` → `sdp doctor control`
- `sdp-control dispatch-card` → `sdp dispatch card`
- `sdp-control result-ingest` → `sdp result ingest`
- `sdp-control attention` → `sdp attention`

The `sdp-control` binary remains as a temporary compatibility shim and shows deprecation notice on every invocation.

## Short formula

- `sdp` = canonical root
- `sdp-control` = temporary compatibility shim (deprecated)
