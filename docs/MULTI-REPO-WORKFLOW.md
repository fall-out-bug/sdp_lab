# sdp_lab <-> sdp Publish Workflow

If your real goal is "install SDP into my own repo and start using it", leave this doc and go to [../sdp/docs/QUICKSTART.md](../sdp/docs/QUICKSTART.md). This file is for contributors who need to publish protocol artifacts to the public `sdp` repo.

## How It Works

All files live natively in `sdp_lab`. There is no submodule, no separate git checkout, and no pointer updates.

| Path | Repo | Remote | Typical change |
|------|------|--------|----------------|
| Root, `internal/`, `cmd/`, `docs/`, `sdp/` | `sdp_lab` | `https://github.com/fall-out-bug/sdp_lab` | All work happens here |
| `fall-out-bug/sdp` (separate repo) | `sdp` (public mirror) | `https://github.com/fall-out-bug/sdp` | Published artifacts only |

Historical note: many workstreams and beads IDs still use `sdp_dev-*` as a legacy label for the root repo. That is history, not a third repo.

## When to Publish

Publish to the public `sdp` repo when protocol artifacts change and external consumers need the update:

- Schema changes (evidence, intent, ws-verdict)
- New or updated prompts/skills
- Hook changes
- `sdp-plugin` bug fixes or features that affect protocol consumers
- Quickstart or CLI reference updates

## Publish with sdp-publish.sh

```bash
# Publish changed artifacts (copy, commit, push to sdp repo):
scripts/sdp-publish.sh

# Preview what would be published without pushing:
scripts/sdp-publish.sh --dry-run

# Check for drift between sdp_lab source and published sdp repo:
scripts/sdp-publish.sh --check
```

The script copies the relevant files from `sdp_lab` into a checkout of `fall-out-bug/sdp`, commits, and pushes. It uses a manifest to determine which paths to publish.

## Artifacts Published

| Source (sdp_lab) | Destination (sdp repo) |
|---|---|
| `sdp/prompts/` | `prompts/` |
| `sdp/schema/` | `schema/` |
| `sdp/hooks/` | `hooks/` |
| `sdp/sdp-plugin/` | `sdp-plugin/` |
| `sdp/docs/QUICKSTART.md` | `docs/QUICKSTART.md` |

The manifest is maintained in `sdp_lab`. Add or remove paths there when the publish surface changes.

## Branch Defaults

| Repo | Default branch |
|------|----------------|
| `sdp_lab` | `main` |
| `sdp` | `main` |

## Commit Workflow

1. Make changes in `sdp_lab` (including files under `sdp/` — they are native files now).
2. Commit and push in `sdp_lab` as usual.
3. After merge to `main`, run `scripts/sdp-publish.sh` if protocol artifacts changed.

## CI Integration

CI runs `scripts/sdp-publish.sh --check` on every push to `main`. If the check fails, the published `sdp` repo has drifted from the source — someone needs to publish.

## Historical Reference

The `sdp/` directory was previously a git submodule. That model was retired in F128 (ADR: `docs/architecture/adr-retire-sdp-submodule.md`). The old two-phase commit workflow (commit in `sdp/` submodule, then update pointer in `sdp_lab`) no longer applies.
