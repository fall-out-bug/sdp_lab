// Package adapters — drift detection helpers for `sdp doctor adapters`.
//
// CheckDrift compares a generated adapter map against files already on disk
// (inside outDir, typically .sdp/generated/).  It also scans the live harness
// directories (.claude/, .opencode/, .codex/, .cursor/rules/) for orphan files
// that are not produced by the generator.
package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// DriftResult holds the outcome of a drift check.
type DriftResult struct {
	// Drifts is a list of human-readable drift descriptions (content differs or file missing).
	Drifts []string
	// Orphans is a list of live-tree files not present in the generated map.
	Orphans []string
}

// IsClean returns true when there are no drifts and no orphans.
func (r *DriftResult) IsClean() bool {
	return len(r.Drifts) == 0 && len(r.Orphans) == 0
}

// knownNonManifestFiles is the whitelist of files in the live harness trees
// that are intentionally not produced by the adapter generator.
var knownNonManifestFiles = map[string]bool{
	".claude/commands/sweep.md": true,
	".codex/skills/README.md":   true,
}

// liveHarnessDirs lists the directories to scan for orphan detection.
// Paths are relative to repoRoot.
var liveHarnessDirs = []string{
	".claude/commands",
	".claude/agents",
	".opencode/agent",
	".opencode/skill",
	".codex/skills",
	".cursor/rules",
	".pi/prompts",
	".pi/skills",
}

// CheckDrift compares the generated adapter map against on-disk files in
// outDir (typically .sdp/generated/ relative to repoRoot) and scans live
// harness directories under repoRoot for orphan files.
//
// generated: map of relative path → expected bytes (from adapters.Generate).
// outDir:    absolute path to the directory holding on-disk generated files.
// repoRoot:  absolute path to the repo root (used for orphan scanning).
func CheckDrift(generated map[string][]byte, outDir, repoRoot string) (*DriftResult, error) {
	result := &DriftResult{}

	// ── 1. Drift: compare generated vs on-disk ──────────────────────────────
	for _, rel := range sortedKeys(generated) {
		dest := filepath.Join(outDir, rel)
		existing, err := os.ReadFile(dest)
		if err != nil {
			result.Drifts = append(result.Drifts, fmt.Sprintf("MISSING  %s", rel))
			continue
		}
		if string(existing) != string(generated[rel]) {
			result.Drifts = append(result.Drifts, fmt.Sprintf("MODIFIED %s", rel))
		}
	}

	// ── 2. Orphan scan: live tree files not in generated map ────────────────
	genSet := make(map[string]bool, len(generated))
	for rel := range generated {
		genSet[rel] = true
	}

	for _, dir := range liveHarnessDirs {
		absDir := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(absDir); err != nil {
			// Directory may not exist yet — skip.
			continue
		}
		if err := filepath.WalkDir(absDir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if knownNonManifestFiles[rel] {
				return nil
			}
			if !genSet[rel] {
				result.Orphans = append(result.Orphans, rel)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Strings(result.Orphans)

	return result, nil
}

// FormatDriftReport renders a human-readable report of drift and orphan findings.
func FormatDriftReport(result *DriftResult, outDir string, strict bool) string {
	var sb strings.Builder

	if len(result.Drifts) == 0 {
		sb.WriteString(fmt.Sprintf("ok: 0 drift(s) — generated files in %s are up to date\n", outDir))
	} else {
		sb.WriteString(fmt.Sprintf("DRIFT DETECTED — %d file(s) out of sync with manifest:\n", len(result.Drifts)))
		for _, d := range result.Drifts {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", d))
		}
		sb.WriteString("\nFix: run `sdp generate-adapters --write --out .sdp/generated`\n")
	}

	if len(result.Orphans) == 0 {
		sb.WriteString("ok: 0 orphan(s)\n")
	} else {
		label := "WARNING"
		if strict {
			label = "ERROR"
		}
		sb.WriteString(fmt.Sprintf("\n%s — %d orphan file(s) not in manifest:\n", label, len(result.Orphans)))
		for _, o := range result.Orphans {
			sb.WriteString(fmt.Sprintf("  ? %s\n", o))
		}
		if strict {
			sb.WriteString("\nFix: remove orphan files or add them to sdp.manifest.yaml\n")
		} else {
			sb.WriteString("\nHint: add to sdp.manifest.yaml or use --strict to treat as errors\n")
		}
	}

	return sb.String()
}
