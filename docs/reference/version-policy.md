# Version Policy

> F078-01: Version source of truth and drift check

## Canonical Source

The **canonical version** for SDP is declared in a single place:

**`sdp.manifest.yaml` → `version` field**

All other version declarations in the repository MUST match this value.

## Version Surfaces

The following files declare the SDP release version and must remain synchronized:

| File | Field/Pattern | Type |
|------|--------------|------|
| `sdp.manifest.yaml` | `version` | Canonical |
| `sdp.manifest.yaml` | `sdp_version` | Release |
| `cmd/sdp/templates/sdp.manifest.template.yaml` | `version` | Template |
| `internal/bootstrap/bootstrap.go` | `const version` | Go constant |
| `cmd/sdp/cmd_init.go` | `sdpVersion = "..."` | Go default |
| `configs/profiles/oss-combine/config.yaml` | `profile.version` | Profile |

### Protocol Spec Versions (Independent)

Protocol versions use `v`-prefix and evolve independently of the release version:

| File | Constant | Current |
|------|----------|---------|
| `internal/cli/status_view.go` | `statusSpecVersion` | v1.0 |
| `internal/cli/instructions_view.go` | `instructionSpecVersion` | v1.0 |
| `internal/control/control.go` | `specVersion` | v1.0 |

Protocol versions must be internally consistent with each other but do NOT need to match the release version.

## Bumping the Version

When bumping the SDP version (e.g., `1.0.0` → `1.1.0`):

1. Update `sdp.manifest.yaml` → `version` field
2. Run `scripts/check-version-drift.sh` to identify all surfaces needing update
3. Update each flagged file to match
4. Re-run the drift check to confirm zero drifts
5. Commit all version changes in a single commit

## Compatibility Windows

SDP follows [semver](https://semver.org/):

- **Major (X.0.0):** Breaking changes to the manifest schema, protocol versions, or CLI interface
- **Minor (0.X.0):** New features, new skills/commands/agents, backward-compatible
- **Patch (0.0.X):** Bug fixes, documentation, no new surfaces

### Exception Process

If a version surface cannot be updated immediately (e.g., published artifact in a downstream repo):

1. File a bead with `--type=bug --priority=P2`
2. Tag it `version-exception`
3. The drift check will report it as DRIFT until resolved
4. Maximum exception window: 1 release cycle

## Local Reproduce Command

```bash
# Check for version drift (human-readable)
scripts/check-version-drift.sh

# Check for version drift (machine-readable JSON)
scripts/check-version-drift.sh --json
```

## CI Integration

The drift check runs as part of the `consistency-gate` job in `.github/workflows/ci.yml`. If any version surface drifts from the canonical source, CI will fail.

## Reference

- `sdp.manifest.yaml` — canonical manifest
- `scripts/check-version-drift.sh` — drift enforcement script
- [ROADMAP.md](../../roadmap/ROADMAP.md)
