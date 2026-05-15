# Slice 2 doc-sync resolution

Date: 2026-05-15

Owner: `sdplab-t5k3`

## Baseline

Command:

```bash
go run ./cmd/sdp-doc-sync --mode check --strict
```

Baseline result before Slice 2 cleanup:

- exit status: 2
- findings: 56 errors, 0 warnings

Buckets:

- broken active-doc local links
- missing ROADMAP feature records
- missing INDEX feature records
- scaffold workstreams without `## Beads` or `## Acceptance Criteria`
- workstreams with placeholder Beads text and no concrete `sdplab-*` reference

## Resolutions

### Broken links

- Removed the broken `.beads/index.json` markdown link from the F141 plan. The
  F128 reference remains textual because the old Beads index file no longer
  exists.
- Repointed the F145 plan's old `internal/dispatch/local.go` link to the current
  `internal/dispatch/ollama_client.go` context and marked the old file as
  historical.
- Removed the broken markdown link in `00-145-06` while preserving the historical
  deletion intent for `internal/dispatch/local.go`.

### ROADMAP and INDEX feature records

- Added active ROADMAP records for `F133`, `F135`, and `F163`.
- Added INDEX rows for `F131`, `F133`, and the post-F150 features `F151` through
  `F160`.
- Updated F145/F146/F147 INDEX rows from stale backlog/no-ws text to the actual
  shipped workstream ranges.

### Scaffold section normalization

- Converted inline Beads text in `00-083-01`, `00-084-01`, `00-085-01`, and
  `00-101-02` into real `## Beads` sections.
- Added design-gate acceptance criteria to `00-083-01`, `00-084-01`, and
  `00-085-01`; these files remain `status: design-pending` and explicitly route
  to `/design` before `/build`.
- Renamed legacy `## Acceptance` headings in `00-101-02` and `00-133-01` to
  `## Acceptance Criteria` and converted existing bullets into checkboxes.

### Beads placeholders

- Replaced F145 and F146 placeholder Beads text with their existing mapped leaf
  `sdplab-*` IDs from `.beads-sdp-mapping.jsonl`.
- Created `sdplab-itzf` as a single historical anchor for F147 because the F147
  workstreams were merged in `d08e716c` with placeholder mapping text and no leaf
  Beads issues are discoverable in the current DB. INDEX now labels F147 as
  historical with leaf Beads not preserved; `sdplab-itzf` is not a 1:1
  implementation map and does not make F147 work newly executable.

## Final Evidence

```bash
go run ./cmd/sdp-doc-sync --mode check --strict
OK: documentation consistency passed
```

```bash
go run ./cmd/sdp-protocol-check --format json --strict
{
  "issues": []
}
```
