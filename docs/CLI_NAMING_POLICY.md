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

## Current migration target

Move the existing control-tower command surface from:
- `sdp-control <...>`

to:
- `sdp card <...>`
- `sdp board <...>`
- `sdp doctor control`
- `sdp dispatch card`
- `sdp result ingest`
- `sdp attention`

## Short formula

- `sdp` = canonical root
- old one-off binaries = temporary compatibility only
