// Package backlog audits coverage between beads features and docs/workstreams/backlog/.
// F142-05: drift gate — beads feature without ws scaffold AND without children is a
// "picker bait" and must be flagged. Used by `sdp doctor backlog`.
package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Feature represents a beads issue of type feature or epic.
type Feature struct {
	BeadID    string
	FID       string // e.g. F141, F101-02
	Title     string
	Status    string // "open", "closed", ...
	IssueType string // "feature" or "epic"
	DepCount  int    // resolved + unresolved children (dependency_count from beads)
}

// Finding describes a single audit flag.
type Finding struct {
	BeadID string
	FID    string
	Title  string
	Reason string // human-readable reason for the flag
}

// Result holds the outcome of an Audit call.
type Result struct {
	Findings []Finding // empty = clean
	Checked  int
}

// AuditOpts configures an Audit run.
//
//   RepoRoot      - to resolve docs/workstreams/backlog
//   IncludeStatus - which beads statuses to check (default: ["open"])
//   Strict        - if true, also flag features whose ws status==design-pending (default false)
type AuditOpts struct {
	RepoRoot      string
	IncludeStatus []string
	Strict        bool
}

// fidRe matches F-identifiers like F141, F101-02.
var fidRe = regexp.MustCompile(`^F([0-9]+)(?:-([0-9]+))?$`)

// wsGlobForFID returns the glob (or exact path for sub-features) inside the
// wsDir corresponding to a given FID.  Mirrors the shell logic in deliver-pick.sh:
//
//	F141    -> wsDir/00-141-*.md   (glob)
//	F101-02 -> wsDir/00-101-02.md  (exact)
func wsGlobForFID(wsDir, fid string) string {
	m := fidRe.FindStringSubmatch(fid)
	if m == nil {
		return ""
	}
	epicNum := 0
	_, _ = fmt.Sscanf(m[1], "%d", &epicNum)

	if m[2] == "" {
		// bare epic — glob all sub-workstreams
		return filepath.Join(wsDir, fmt.Sprintf("00-%03d-*.md", epicNum))
	}
	subNum := 0
	_, _ = fmt.Sscanf(m[2], "%d", &subNum)
	return filepath.Join(wsDir, fmt.Sprintf("00-%03d-%02d.md", epicNum, subNum))
}

// countWSFiles returns the number of ws files matching the pattern, and for
// the first match, its frontmatter status (empty string if none / no frontmatter).
func countWSFiles(pattern string) (count int, firstStatus string) {
	// Use filepath.Glob for both exact paths and globs.
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return 0, ""
	}
	count = len(matches)
	firstStatus = readWSStatus(matches[0])
	return count, firstStatus
}

// readWSStatus reads the `status:` field from YAML frontmatter of a ws file.
// Returns empty string if the file cannot be read or has no frontmatter status.
func readWSStatus(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inFM := false
	for _, line := range lines {
		if line == "---" {
			if !inFM {
				inFM = true
				continue
			}
			// Second --- closes frontmatter
			break
		}
		if inFM && strings.HasPrefix(line, "status:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			return val
		}
	}
	return ""
}

// defaultStatuses is the fallback when AuditOpts.IncludeStatus is empty.
var defaultStatuses = []string{"open"}

// statusSet converts a slice to a lookup map.
func statusSet(statuses []string) map[string]bool {
	s := make(map[string]bool, len(statuses))
	for _, v := range statuses {
		s[v] = true
	}
	return s
}

// Audit runs the backlog audit.  The Feature list is provided by the caller
// (from `bd list --type=feature --json` or a test fixture); Audit itself is
// pure-data and performs no subprocess calls.
func Audit(opts AuditOpts, features []Feature) Result {
	wsDir := filepath.Join(opts.RepoRoot, "docs", "workstreams", "backlog")

	statuses := opts.IncludeStatus
	if len(statuses) == 0 {
		statuses = defaultStatuses
	}
	allowed := statusSet(statuses)

	var result Result

	for _, f := range features {
		// Filter by status.
		if !allowed[f.Status] {
			continue
		}
		// Only flag feature / epic types.
		if f.IssueType != "feature" && f.IssueType != "epic" {
			continue
		}
		// Skip if no FID — not a workstream feature.
		if f.FID == "" {
			continue
		}

		result.Checked++

		pattern := wsGlobForFID(wsDir, f.FID)
		if pattern == "" {
			// Malformed FID — flag as suspicious.
			result.Findings = append(result.Findings, Finding{
				BeadID: f.BeadID,
				FID:    f.FID,
				Title:  f.Title,
				Reason: fmt.Sprintf("malformed FID %q — cannot compute ws glob", f.FID),
			})
			continue
		}

		wsCount, firstStatus := countWSFiles(pattern)

		if wsCount == 0 && f.DepCount == 0 {
			result.Findings = append(result.Findings, Finding{
				BeadID: f.BeadID,
				FID:    f.FID,
				Title:  f.Title,
				Reason: "no ws file and no children — picker bait",
			})
			continue
		}

		if opts.Strict && wsCount > 0 && firstStatus == "design-pending" && f.DepCount == 0 {
			result.Findings = append(result.Findings, Finding{
				BeadID: f.BeadID,
				FID:    f.FID,
				Title:  f.Title,
				Reason: fmt.Sprintf("ws marked design-pending (%s) and no children", filepath.Base(pattern)),
			})
		}
	}

	return result
}

// FormatReport returns a human-readable multi-line report suitable for terminal output.
func FormatReport(r Result) string {
	var sb strings.Builder

	if len(r.Findings) == 0 {
		sb.WriteString(fmt.Sprintf("ok: 0 finding(s) — %d feature(s) checked, all have ws scaffold or children\n", r.Checked))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("BACKLOG DRIFT — %d finding(s) from %d feature(s) checked:\n", len(r.Findings), r.Checked))
	for _, f := range r.Findings {
		sb.WriteString(fmt.Sprintf("  [%s] %s (%s): %s\n", f.BeadID, f.FID, f.Title, f.Reason))
	}
	sb.WriteString("\nFix: create a ws scaffold in docs/workstreams/backlog/ or add children to the feature.\n")
	sb.WriteString("Hint: run `sdp doctor backlog` locally to see the full list before pushing.\n")

	return sb.String()
}
