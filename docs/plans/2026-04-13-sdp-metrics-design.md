# SDP Metrics: Process & Code Health from Git History

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract 7 categories of process and code health metrics purely from git history, producing structured JSON for skill interpretation.

**Architecture:** Single-pass git log collection feeds 7 parallel analyzers. Go CLI produces JSON, skill writes markdown with traffic-light ratings, meta-skill correlates with architect data.

**Tech Stack:** Go, git CLI, no external dependencies.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`

---

## Data Flow

```
git log --numstat --format=<rich> --no-merges --since=2y  (1 call)
git tag --sort=creatordate                                 (1 call)
git branch -r                                              (1 call)
git log --merges --first-parent main                       (1 call)
                    │
            ┌───────┴───────┐
            │  Raw Git Data  │
            └───────┬───────┘
                    │
    ┌───────┬───────┼───────┬───────┬───────┬───────┐
    v       v       v       v       v       v       v
 Hygiene  Waste  GitFlow  Release  Stab.  Know.  Decay
    │       │       │       │       │       │       │
    └───────┴───────┴───────┬───────┴───────┴───────┘
                            v
                    MetricsReport JSON
                    .sdp/metrics/report.json
```

**4 git calls total** instead of 20+. Critical for performance on large repos.

## Git Log Format

Single rich format capturing all needed fields:

```bash
git log --numstat --no-merges --since="2 years ago" \
  --format="COMMIT:%H%nAUTHOR:%an%nDATE:%aI%nSUBJECT:%s%nBODY:%b%nEND"
```

Parsed into:
```go
type RawCommit struct {
    Hash     string
    Author   string
    Date     time.Time
    Subject  string
    Body     string
    Files    []FileChange // from --numstat: added, deleted, path
}

type FileChange struct {
    Added   int
    Deleted int
    Path    string
}
```

## JSON Output Contract

```jsonc
{
  "version": "1.0.0",
  "repo": "owner/repo-name",
  "period": {"from": "2024-04-13", "to": "2026-04-13"},
  "commits_analyzed": 12847,
  "generated_at": "2026-04-13T15:30:00Z",

  "hygiene": {
    "ticket_linked_ratio": 0.82,
    "ticket_patterns_found": ["FLINK-\\d+", "#\\d+"],
    "conventional_commits_ratio": 0.71,
    "commit_type_breakdown": {
      "feat": 342, "fix": 198, "chore": 87, "refactor": 45,
      "docs": 23, "test": 18, "other": 134
    },
    "fix_to_feature_ratio": 0.34,
    "avg_message_length": 48,
    "top_unlinked_authors": [
      {"author": "dependabot", "unlinked": 230},
      {"author": "renovate", "unlinked": 45}
    ]
  },

  "waste": {
    "churn_ratio": 0.23,
    "churn_window_days": 14,
    "churn_files_top": [
      {"path": "src/main/Handler.go", "added": 1200, "deleted": 980, "window_days": 12}
    ],
    "abandoned_branches": 14,
    "abandoned_branches_detail": [
      {"name": "feature/old-parser", "last_commit": "2025-11-03", "lines_changed": 430}
    ],
    "abandoned_lines_total": 8430,
    "revert_rate": 0.018,
    "revert_count": 23,
    "reverted_commits": [
      {"hash": "abc123", "subject": "Revert \"feat: add cache\"", "date": "2026-01-15"}
    ]
  },

  "git_flow": {
    "detected_model": "trunk-based",
    "confidence": 0.85,
    "evidence": [
      "85% branches live <3 days",
      "no release/* branches",
      "merge frequency 34/week"
    ],
    "branch_lifetime_median_hours": 28.5,
    "branch_lifetime_p95_hours": 168,
    "merge_frequency_per_week": 34.2,
    "long_lived_branches": [
      {"name": "feature/new-scheduler", "age_days": 45, "commits": 23}
    ]
  },

  "release_quality": {
    "releases_analyzed": 12,
    "avg_time_to_first_hotfix_hours": 18.3,
    "releases": [
      {
        "tag": "v1.18.0",
        "date": "2025-09-01",
        "fixes_7d": 4,
        "fixes_14d": 7,
        "fixes_30d": 11,
        "cherry_picks": 3,
        "rc_count": 2,
        "time_to_first_fix_hours": 14.2
      }
    ]
  },

  "stabilization": {
    "releases": [
      {
        "base": "v1.18",
        "stabilized_at_patch": 3,
        "patches_total": 5,
        "days_to_stable": 21
      }
    ],
    "avg_patches_to_stable": 2.8,
    "trend": "improving"
  },

  "knowledge_risk": {
    "overall_bus_factor": 2,
    "gini_coefficient": 0.72,
    "bus_factor_by_module": [
      {
        "module": "core/",
        "bus_factor": 1,
        "primary_author": "Alice",
        "primary_author_ratio": 0.87,
        "gini": 0.87,
        "files_count": 34
      }
    ],
    "abandoned_code": [
      {
        "path": "legacy/parser.go",
        "last_touched": "2024-01-15",
        "age_months": 27,
        "loc": 450,
        "imported_by_count": 12
      }
    ],
    "former_contributor_ratio": 0.31,
    "former_contributors": ["Carol", "Dave"]
  },

  "decay": {
    "shotgun_surgery_ratio": 0.04,
    "shotgun_commits": 47,
    "shotgun_threshold": {"min_files": 5, "min_dirs": 3, "max_lines_per_file": 20},
    "monotonic_growth_files": [
      {
        "path": "api/handler.go",
        "months_growing": 14,
        "start_loc": 800,
        "current_loc": 2340,
        "zero_refactor_events": true
      }
    ],
    "fix_recurrence": [
      {
        "path": "scheduler/dag.go",
        "fix_count": 7,
        "total_commits": 15,
        "fix_density": 0.47,
        "fix_chains": 2
      }
    ]
  }
}
```

## Category Details

### 1. Commit Hygiene

**Ticket linkage:** Scan commit subjects and bodies for patterns:
- Jira: `[A-Z]+-\d+` (e.g., FLINK-1234)
- GitHub: `#\d+`, `fixes #\d+`, `closes #\d+`, `resolves #\d+`
- Linear: `[A-Z]+-\d+`
- Custom patterns from config

**Conventional Commits:** Parse `type(scope): description` format.
- Known types: feat, fix, chore, refactor, docs, test, ci, build, perf, style
- Fallback classification by keyword matching for non-conventional repos

**Fix-to-feature ratio:** `fix_commits / (fix_commits + feat_commits)`.
- Threshold: <0.3 green, 0.3-0.5 yellow, >0.5 red.

**Bot filtering:** Exclude authors matching: dependabot, renovate, github-actions, mergify, snyk.

### 2. Wasted Work

**Churn within window:** For each file, track `{commit_date, added, deleted}` series.
Within sliding window of N days (default 14):
- `churn = min(total_added, total_deleted)` per window
- `churn_ratio = total_churn / total_added` across all files
- Threshold: <0.15 green, 0.15-0.25 yellow, >0.25 red

**Abandoned branches:** `git branch -r --no-merged main`
- Filter: last commit > 30 days ago
- Estimate wasted lines: `git diff --shortstat main...<branch>`

**Revert rate:** Commits matching `^Revert "` or `revert|rollback|undo` in subject.
- `revert_rate = revert_commits / total_commits`
- Threshold: <0.01 green, 0.01-0.03 yellow, >0.03 red

### 3. Git Flow Detection

**Algorithm:** Score each model based on branch/merge patterns:

| Signal | trunk-based | GitHub Flow | GitFlow |
|--------|-------------|-------------|---------|
| Median branch lifetime <3d | +3 | +1 | 0 |
| release/* branches exist | 0 | 0 | +3 |
| develop branch exists | 0 | 0 | +3 |
| >80% branches from main | +2 | +2 | 0 |
| hotfix/* branches exist | 0 | 0 | +2 |
| Merge frequency >20/week | +2 | +1 | 0 |
| Feature branches >7d avg | 0 | +1 | +2 |

Highest score wins. Confidence = score_gap / max_possible.

### 4. Release Quality

**Tag parsing:** `git tag --sort=creatordate` filtered by semver pattern `v?\d+\.\d+\.\d+`.

**Time to first hotfix:** For each release tag, find first subsequent commit with "fix" classification.
- `ttfh = date(first_fix) - date(tag)`

**Post-release fix count:** Count fix-classified commits in windows: 7d, 14d, 30d after tag.

**Cherry-pick detection:** `git log --grep="cherry picked from" <tag1>..<tag2>`

**RC count:** Group tags by base version, count `-rc*`, `-alpha*`, `-beta*` variants.

### 5. Release Stabilization

**Stabilized-at-patch:** For each base version (e.g., v1.18), track patch releases.
A version is "stable" when: the patch has ≤1 fix-commit in the 14 days after it.

**Trend:** Linear regression of `stabilized_at_patch` across releases.
- Slope < 0: improving
- Slope ≈ 0: stable
- Slope > 0: degrading

### 6. Knowledge Risk

**Bus factor per module:**
1. For each module (top-level directory), count commits per author
2. Sort authors by commit count descending
3. Bus factor = minimum authors covering >50% of commits
4. Gini coefficient of commit distribution

**Abandoned code:**
1. For each file: `last_touched = git log -1 --format=%aI -- <path>`
2. If `now - last_touched > 12 months` → candidate
3. Check if file is imported by other files (language-dependent grep)

**Former contributor ratio:**
1. Active authors = committed in last 6 months
2. Former authors = have commits but not in last 6 months
3. Ratio = former_author_lines_in_hotspots / total_hotspot_lines

### 7. Code Decay

**Shotgun surgery:** Commits where:
- `changed_files >= 5` AND
- `changed_directories >= 3` AND
- `lines_per_file <= 20`
Ratio = shotgun_commits / total_commits.

**Monotonic growth:** For each file, build LOC timeseries from `--numstat`.
- File is monotonically growing if: no commit in 6+ months where `deleted > added * 0.3`
- Flag files growing for 6+ months without refactoring events

**Fix recurrence:** Files with 5+ fix-classified commits in the analysis period.
- Fix density = fix_commits / total_commits for that file
- Fix chain = two fix commits for same file within 7 days

## CLI Interface

```bash
sdp metrics <repo-path>                  # Full analysis, JSON to stdout
sdp metrics --format text <repo-path>    # Human-readable summary
sdp metrics --format markdown <repo-path> # Markdown for skill
sdp metrics --period 5 <repo-path>       # Last 5 years
sdp metrics --category hygiene,waste     # Specific categories only
sdp metrics --output .sdp/metrics/       # Write to directory
sdp metrics --verbose <repo-path>        # Per-analyzer timing
```

## Traffic Light Thresholds

| Category | Green | Yellow | Red |
|----------|-------|--------|-----|
| Hygiene: ticket ratio | >0.7 | 0.4-0.7 | <0.4 |
| Hygiene: fix/feature | <0.3 | 0.3-0.5 | >0.5 |
| Waste: churn ratio | <0.15 | 0.15-0.25 | >0.25 |
| Waste: revert rate | <0.01 | 0.01-0.03 | >0.03 |
| Git Flow: branch lifetime median | <3d | 3-7d | >7d |
| Release: time to first fix | >48h | 24-48h | <24h |
| Stabilization: patches to stable | ≤2 | 3-4 | ≥5 |
| Knowledge: bus factor (min module) | ≥3 | 2 | 1 |
| Knowledge: gini | <0.5 | 0.5-0.7 | >0.7 |
| Decay: shotgun ratio | <0.02 | 0.02-0.05 | >0.05 |
| Decay: fix recurrence files | 0 | 1-3 | >3 |

## Noise Filtering

Applied globally before analysis:
- **Bot commits:** Exclude authors matching known bot patterns
- **Merge commits:** Excluded via `--no-merges` for authorship metrics
- **Generated files:** Exclude `*.pb.go`, `*.generated.*`, `vendor/`, `node_modules/`, `*.lock`
- **Mass formatting:** Commits where >90% of files have `added ≈ deleted ± 10%`
- **CI-only changes:** Commits touching only `.github/`, `.gitlab-ci.yml`, `Jenkinsfile`

## Go Package Structure

```
internal/metrics/
  ├── collector.go       # Single-pass git log parser → RawCommit slice
  ├── collector_test.go
  ├── hygiene.go         # Commit hygiene analyzer
  ├── hygiene_test.go
  ├── waste.go           # Wasted work analyzer
  ├── waste_test.go
  ├── gitflow.go         # Git flow detection
  ├── gitflow_test.go
  ├── release.go         # Release quality + stabilization
  ├── release_test.go
  ├── knowledge.go       # Knowledge risk analyzer
  ├── knowledge_test.go
  ├── decay.go           # Code decay analyzer
  ├── decay_test.go
  ├── types.go           # MetricsReport, thresholds, traffic lights
  ├── filter.go          # Bot/generated/formatting filters
  └── filter_test.go

cmd/sdp/
  └── cmd_metrics.go     # CLI subcommand (~150 LOC)
```

## Testing Strategy

Each analyzer tested with synthetic git histories:
- Create temp git repo with `git init`
- Programmatically create commits with known patterns
- Assert analyzer output matches expected metrics
- Edge cases: empty repo, single commit, single author, no tags

## Relationship to Existing Code

- **Reuses:** git command execution patterns from `internal/architect/extract/git_extract.go`
- **Does NOT extend:** `git_extract.go` — separate package, different concerns
- **Feeds into:** `sdp index` (bus factor, hotspots enrich index metadata)
- **Consumed by:** `@metrics` skill (JSON→markdown), `@landscape` meta-skill (correlation)
