# SDP Dogfooding Observations

This document tracks issues and improvement opportunities discovered while using SDP tools.

## 2026-03-01 Session

### sdp-ready

**Issue 1**: Priority display inconsistency
- JSON output shows priority as integer (1, 2, 3)
- Text output shows priority as "P1", "P2", etc.
- **Recommendation**: Standardize to consistent format

**Issue 2**: Phase awareness
- Tool surfaces F060 (Phase 8-9 K8s) work alongside F061 (Phase 5 current)
- No filtering by phase/applicability
- **Recommendation**: Add `--phase` flag or priority boost for current phase work

**Issue 3**: bd ready --format json doesn't exist
- Had to use `--json` instead
- Fixed in internal/beads/client.go

### Beads CLI

**Issue 4**: Unknown flag --comment for bd close
- Expected `--comment` but actual flag is `--reason` or `-r`
- **Recommendation**: Document Beads CLI flags in SDP AGENTS.md

**Issue 5**: Legacy database detected
- Required `bd migrate --update-repo-id` after clone
- **Recommendation**: Add setup step to AGENTS.md

### Formula Apply

**Issue 6**: Flag order sensitivity
- `sdp-beads-bridge formula apply formula.yaml --feature F062` doesn't work
- Must use `sdp-beads-bridge formula apply --feature F062 formula.yaml`
- **Root cause**: Go flag package parses until first positional argument
- **Recommendation**: Update usage docs or switch to cobra/urfavecli

### Variable Substitution

**Issue 7**: Inconsistent template syntax
- Beads formulas use `{{var}}` (no dot)
- Go templates use `{{.var}}` (with dot)
- **Fix**: Support both patterns in substituteVars()

### Workstream Status Tracking

**Issue 8**: Manual sync required
- Workstream files status must match Beads status
- Currently manual process
- **Recommendation**: `sdp sync` command to bidirectional sync

---

## Improvement Backlog

1. `sdp-ready --phase <N>` - Filter by roadmap phase
2. `sdp sync` - Sync workstream status with Beads
3. Better flag parsing (use cobra)
4. Auto-generate Beads issues from workstream files
5. Formula hash verification (detect drift)
