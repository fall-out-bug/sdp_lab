# Redaction Checklist (Private)

Use this checklist before publishing any artifact to OSS.

## Checklist

1. Prepare sanitized draft using `docs/OSS_EXPORT_TEMPLATE.md`.
2. Confirm no private-only sections are included.
3. Run redaction gate:

```bash
go run ./cmd/redaction-check --file <path-to-export-draft>
```

4. Ensure output `ok: true`.
5. Attach gate output to Beads task notes.

## Gate fail behavior

- If `ok: false`, do not export.
- Remove violating lines and rerun gate.
