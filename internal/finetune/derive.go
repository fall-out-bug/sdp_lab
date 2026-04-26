package finetune

import (
	"strings"
)

// SystemPrompt is the fixed system message used for every sample. The base
// model and fine-tuned model both see this exact text at inference time.
const SystemPrompt = `You are a task complexity classifier for the SDP dispatch router.
Given a software task brief, classify it on three axes.

Output ONLY a single-line JSON object with these keys:
- complexity: "low" | "medium" | "high"
- task_type:  "feature" | "bugfix" | "refactor" | "test" | "docs"
- risk:       "low" | "high"

No markdown, no commentary, no trailing newline.`

// DeriveComplexity maps a workstream `size` field to the label vocabulary.
func DeriveComplexity(size string) string {
	switch strings.ToUpper(strings.TrimSpace(size)) {
	case "S", "XS":
		return "low"
	case "M":
		return "medium"
	case "L", "XL":
		return "high"
	default:
		return ""
	}
}

// DeriveRiskFromPriority maps a workstream `priority` (P0..P4) or beads
// numeric priority to risk level.
func DeriveRiskFromPriority(priority string) string {
	p := strings.ToUpper(strings.TrimSpace(priority))
	switch p {
	case "P0", "P1", "0", "1":
		return "high"
	case "P2", "P3", "P4", "2", "3", "4":
		return "low"
	default:
		return ""
	}
}

// DeriveTaskType inspects title + body for whole-word keyword signals.
// Returns "" when no signal is strong enough — caller should drop those samples.
//
// Keywords are matched as whole words (surrounded by non-letter characters)
// to avoid false positives like "test" matching "latest" or "bug" matching
// "debug".
func DeriveTaskType(title, body string) string {
	t := strings.ToLower(title + " " + body)

	// Order matters: more specific signals win.
	switch {
	case hasWord(t, "fix", "bug", "bugfix", "regression", "incident", "hotfix", "broken", "crash"):
		return "bugfix"
	case hasWord(t, "test", "tests", "tdd", "coverage", "fixture", "golden"):
		return "test"
	case hasWord(t, "refactor", "rename", "extract", "consolidate", "cleanup", "deduplicate"):
		return "refactor"
	case hasWord(t, "docs", "doc", "documentation", "readme", "guide", "tutorial"):
		return "docs"
	case hasWord(t, "implement", "add", "introduce", "support", "enable", "feature", "build"):
		return "feature"
	}
	return ""
}

// hasWord returns true when any of needles appears in haystack as a whole
// word — bounded by ASCII non-letter characters or string edges.
// haystack must already be lowercased.
func hasWord(haystack string, needles ...string) bool {
	for _, n := range needles {
		idx := 0
		for {
			j := strings.Index(haystack[idx:], n)
			if j < 0 {
				break
			}
			pos := idx + j
			endPos := pos + len(n)
			if isWordEdge(haystack, pos-1) && isWordEdge(haystack, endPos) {
				return true
			}
			idx = pos + 1
		}
	}
	return false
}

// isWordEdge reports whether the byte at index i is a word boundary.
// Out-of-range indices count as boundaries (string edge). ASCII letters and
// digits are word-internal; everything else (space, punctuation) is a boundary.
func isWordEdge(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	c := s[i]
	if c >= 'a' && c <= 'z' {
		return false
	}
	if c >= '0' && c <= '9' {
		return false
	}
	return true
}
