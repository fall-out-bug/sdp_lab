# SDP Scout: Quick Codebase Reconnaissance

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 30-second assessment of an unknown codebase — language, scale, build system, activity, maturity — producing a structured project card for humans and agents.

**Architecture:** Three-phase POSIX-first scan (identity → scale → health), no external dependencies, no LLM calls. Pure filesystem + git analysis.

**Tech Stack:** Go, os/filepath, exec (git commands only), no CGO.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`

---

## Problem Statement

When you arrive at a new codebase, the first question is "what is this?" Today's answer takes 5-15 minutes of manual exploration or an expensive `sdp architect analyze`. Scout answers in 30 seconds with zero configuration.

**Before:** `sdp architect analyze .` (5-15 min, LLM calls, full report)
**After:** `sdp scout .` (30 sec, no LLM, project card)

Scout is the FRAME phase of existing `sdp discover`, extracted and hardened as a standalone tool.

## Output: Project Card

```jsonc
{
  "version": "1.0.0",
  "scanned_at": "2026-04-13T15:00:00Z",
  "duration_ms": 2700,

  "identity": {
    "name": "sdp",
    "description": null,          // from README first paragraph, or null
    "repo_url": "github.com/user/sdp",
    "primary_language": "go",
    "languages": {
      "go": { "files": 312, "ratio": 0.72 },
      "typescript": { "files": 48, "ratio": 0.11 },
      "shell": { "files": 34, "ratio": 0.08 },
      "yaml": { "files": 24, "ratio": 0.06 },
      "other": { "files": 15, "ratio": 0.03 }
    },
    "build_system": "go-modules",    // "go-modules", "maven", "gradle", "npm", "cargo", "mix", "sbt", "cmake", "make", "bazel"
    "build_files": ["go.mod", "Makefile"],
    "monorepo": false
  },

  "scale": {
    "total_files": 433,
    "total_loc": 48200,
    "source_files": 312,
    "test_files": 67,
    "test_ratio": 0.18,
    "generated_files": 12,
    "vendor_files": 0,
    "max_file_loc": 1240,
    "median_file_loc": 89,
    "directories": 47,
    "depth_max": 6
  },

  "activity": {
    "first_commit": "2024-06-15",
    "last_commit": "2026-04-13",
    "age_months": 22,
    "total_commits": 1247,
    "contributors": 8,
    "active_contributors_90d": 4,
    "commits_30d": 87,
    "commits_90d": 234,
    "active_branches": 5
  },

  "maturity": {
    "has_readme": true,
    "has_license": true,
    "has_ci": true,
    "ci_system": "github-actions",   // "github-actions", "gitlab-ci", "jenkins", "circleci", "travis", null
    "has_tests": true,
    "has_linter": true,
    "has_docker": true,
    "has_releases": true,
    "latest_release": "v7.2.0",
    "release_count": 14,
    "has_codeowners": false,
    "has_contributing": true,
    "has_changelog": false
  },

  "build": {
    "entry_points": ["cmd/sdp/main.go", "cmd/sdp-dispatch/main.go"],
    "config_files": [".goreleaser.yml", ".github/workflows/ci.yml"],
    "package_manager": "go-modules",
    "dependency_count": 42,
    "dependency_file": "go.mod"
  },

  "health_signals": {
    "bus_factor_estimate": 3,        // quick heuristic: authors of 50%+ commits
    "commit_frequency": "high",      // "high" >20/week, "medium" 5-20, "low" <5
    "staleness": "active",           // "active" <7d, "recent" 7-30d, "stale" 30-90d, "dormant" >90d
    "test_coverage_hint": "partial", // "good" >0.3 ratio, "partial" 0.1-0.3, "low" <0.1, "none" 0
    "complexity_hint": "medium"      // from max_file_loc + directory depth + file count
  }
}
```

## Three-Phase Execution

### Phase 1: Identity (target: 500ms)

Fast filesystem scan — no git commands.

1. **Build system detection:** Check root for known build files:
   | File | System |
   |------|--------|
   | `go.mod` | go-modules |
   | `pom.xml` | maven |
   | `build.gradle`, `build.gradle.kts` | gradle |
   | `build.sbt` | sbt |
   | `package.json` | npm |
   | `Cargo.toml` | cargo |
   | `mix.exs` | mix |
   | `CMakeLists.txt` | cmake |
   | `Makefile` (alone) | make |
   | `BUILD`, `WORKSPACE` | bazel |
   | `pyproject.toml`, `setup.py` | python |
   | `Gemfile` | bundler |

2. **Language distribution:** Walk filesystem, count files by extension.
   - Exclude: `vendor/`, `node_modules/`, `.git/`, `__pycache__/`, `target/`, `build/`, `dist/`
   - Map extensions to languages (`.go`→go, `.ts`/`.tsx`→typescript, `.py`→python, etc.)
   - Primary language = highest file count

3. **README extraction:** Read first paragraph of `README.md` (up to 200 chars) for description.

4. **Monorepo detection:** Check for multiple build files in subdirectories, `lerna.json`, `pnpm-workspace.yaml`, or `packages/*/package.json`.

### Phase 2: Scale (target: 500ms)

Filesystem walk with LOC counting.

1. **File counting:** Categorize each file:
   - Source: matches language extensions
   - Test: matches `*_test.go`, `*.test.ts`, `test_*.py`, `*Test.java`, `*Spec.scala`, etc.
   - Generated: matches patterns from `internal/architect/extract/generated.go`
   - Vendor: in `vendor/` or `node_modules/`

2. **LOC counting:** `wc -l` equivalent on source files only.
   - Skip binary files (check first 512 bytes for null bytes)
   - Skip files >100KB (likely generated/data)

3. **Entry points:** Find `main()` or equivalent:
   - Go: files with `func main()` in `cmd/` or root
   - Python: files with `if __name__ == "__main__"`
   - Node: `"main"` or `"bin"` in package.json
   - Java: files with `public static void main`

### Phase 3: Activity & Health (target: 2s)

Git commands — the slowest phase.

```bash
# 1. Commit history summary (single call)
git log --format="%H|%an|%aI" --no-merges --since="2 years ago" | head -5000

# 2. First commit date
git log --reverse --format="%aI" | head -1

# 3. Branch count
git branch -r --no-merged main 2>/dev/null | wc -l

# 4. Tags/releases
git tag --sort=-creatordate | head -20

# 5. CI detection (filesystem, no git)
# Check: .github/workflows/, .gitlab-ci.yml, Jenkinsfile, .circleci/
```

**Derived metrics:**
- `bus_factor_estimate`: Sort authors by commit count, find minimum covering >50%
- `commit_frequency`: commits_30d / 4.3 → per-week rate
- `staleness`: `now - last_commit_date`
- `test_coverage_hint`: `test_files / source_files`
- `complexity_hint`: heuristic from `max_file_loc`, `depth_max`, `total_files`

## Exclusion Patterns

Shared with other SDP tools, defined in `internal/common/exclude.go`:

```go
var DefaultExcludes = []string{
    ".git", "vendor", "node_modules", "__pycache__",
    "target", "build", "dist", ".next", ".nuxt",
    ".terraform", ".gradle", ".mvn",
    "*.pb.go", "*.generated.*", "*.min.js", "*.min.css",
    "*.lock", "*.sum",
}
```

## CLI Interface

```bash
sdp scout <repo-path>                # JSON to stdout
sdp scout --format text <repo-path>  # Human-readable summary
sdp scout --format card <repo-path>  # Compact one-screen card
sdp scout --output .sdp/ <repo-path> # Write to .sdp/scout.json
```

### Human-Readable Output (--format text)

```
 sdp — Go project (48.2K LOC, 433 files)
 ─────────────────────────────────────────
 Languages:  Go 72% | TypeScript 11% | Shell 8% | YAML 6%
 Build:      go-modules (go.mod, Makefile)
 Tests:      67 test files (18% ratio)
 Activity:   1247 commits, 8 contributors, 87 in last 30d
 Age:        22 months (Jun 2024 – Apr 2026)
 Maturity:   README CI Tests Docker Releases(v7.2.0)
 Health:     Bus factor ~3 | Active | Medium complexity
 Entry:      cmd/sdp/main.go, cmd/sdp-dispatch/main.go
```

## Performance Targets

| Phase | Target | What |
|-------|--------|------|
| Identity | <500ms | Filesystem walk, build detection |
| Scale | <500ms | LOC counting, file categorization |
| Activity | <2s | Git history, branches, tags |
| **Total** | **<3s** | **Full project card** |

For very large repos (>100K files): skip LOC counting for non-primary languages, sample git history (last 5000 commits only).

## Relationship to Other Components

```
sdp scout .          →  .sdp/scout.json (fast project card)
    │
    │  feeds into
    ▼
sdp architect .      →  deep analysis (uses scout for quick context)
sdp bootstrap .      →  uses scout.json to detect build system, CI, etc.
sdp index build .    →  uses scout.json for language detection, exclusions
```

Scout is intentionally fast and shallow. It answers "what is this?" not "how is it built?" — that's architect's job.

## Go Package Structure

```
internal/scout/
  ├── scout.go           # Orchestrates three phases, produces ProjectCard
  ├── scout_test.go
  ├── identity.go        # Phase 1: language, build system, README
  ├── identity_test.go
  ├── scale.go           # Phase 2: file counts, LOC, entry points
  ├── scale_test.go
  ├── activity.go        # Phase 3: git history, branches, tags
  ├── activity_test.go
  ├── types.go           # ProjectCard, Identity, Scale, Activity, etc.
  ├── health.go          # Health signal derivation heuristics
  ├── health_test.go
  └── languages.go       # Extension→language mapping, test file patterns

internal/common/
  └── exclude.go         # Shared exclusion patterns (used by scout, index, metrics)

cmd/sdp/
  └── cmd_scout.go       # CLI subcommand (~100 LOC)
```

## Testing Strategy

1. **Identity tests:** Create temp directories with known build files, verify detection.
   - Go, Python, Node, Java, Rust, mixed-language repos
   - Monorepo detection edge cases

2. **Scale tests:** Create temp files with known LOC, verify counting.
   - Binary file exclusion
   - Generated file detection
   - Large file skipping

3. **Activity tests:** Create temp git repos with scripted commit history.
   - Bus factor calculation
   - Staleness detection
   - Edge cases: single commit, single author, no tags

4. **Integration test:** Run against sdp_lab repo itself, verify reasonable output.

## Design Decisions

1. **No LLM calls.** Scout must be deterministic and fast. LLM-based description generation belongs to architect.

2. **No CGO dependencies.** Scout should build easily everywhere. Pure Go + exec for git.

3. **POSIX-first.** Git is the only external dependency. Everything else is filesystem operations.

4. **Sampling for large repos.** Cap git history at 5000 commits, skip LOC for minor languages. Accuracy degrades gracefully.

5. **Shared exclusion patterns.** `internal/common/exclude.go` used by scout, index, and metrics to avoid duplicating filter logic.
