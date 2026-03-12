# Reality-Pro External Evidence Ingestion

Updated: 2026-03-12

## Purpose

Close the biggest current gap between the private `reality-pro` baseline and the consulting-grade spec: it still reasons mostly from repos, not from normalized external evidence.

## Scope

- add `--with-docs` and explicit doc-root support to `sdp-reality-pro-ingest`
- ingest optional docs, ADRs, and runbooks into `repo-memory.json`
- surface evidence coverage in `docs/reality/multi-repo-map.md`
- wire normalized evidence sources into `reality-pro-review` and `reality-pro-report`

## Execution Slice

- `F093`
- `00-093-01`
- Bead: `sdplab-wdu`

## Done Signal

`reality-pro` is more than repo topology when one run can:

1. ingest external evidence paths on demand
2. persist normalized evidence sources into repo memory
3. cite those sources in review and report artifacts
4. show operators whether the reposet has thin or rich consulting evidence

## Result

Completed in `00-093-01`: `reality-pro` now supports `--with-docs`, persists normalized evidence sources in `repo-memory.json`, carries those sources through review/report artifacts, and renders evidence coverage plus sample sources in `docs/reality/multi-repo-map.md`.
