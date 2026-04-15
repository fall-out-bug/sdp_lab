package metrics

import ()

var knownTypes = map[string]bool{
	"feat": true, "fix": true, "chore": true, "refactor": true,
	"docs": true, "test": true, "ci": true, "build": true,
	"perf": true, "style": true,
}

// AnalyzeHygiene computes commit hygiene metrics from raw git data.
func AnalyzeHygiene(data *GitData) *Hygiene {
	if data == nil || len(data.Commits) == 0 {
		return nil
	}
	h := &Hygiene{
		CommitTypeBreakdown: make(map[string]int),
	}

	ticketPatterns := []struct {
		re string
		ex string
	}{
		{`#[0-9]+`, "#"},
		{`[A-Z]+-[0-9]+`, "JIRA-style"},
		{`fixes #[0-9]+|closes #[0-9]+|resolves #[0-9]+`, "fixes/closes"},
	}

	var ticketLinked, conventional int
	var totalMsgLen int
	var totalFiles int
	var monoStyleCount int
	var fixCount, featCount int

	for _, c := range data.Commits {
		// Message length
		totalMsgLen += len(c.Subject)
		totalFiles += len(c.Files)
		if len(c.Files) > 10 {
			monoStyleCount++
		}

		// Ticket linkage
		combined := c.Subject + " " + c.Body
		linked := false
		for _, tp := range ticketPatterns {
			if containsPattern(combined, tp.re) {
				linked = true
				found := false
				for _, p := range h.TicketPatternsFound {
					if p == tp.ex {
						found = true
						break
					}
				}
				if !found {
					h.TicketPatternsFound = append(h.TicketPatternsFound, tp.ex)
				}
			}
		}
		if linked {
			ticketLinked++
		}

		// Conventional commits: type(scope): description
		subj := c.Subject
		ccType := parseConventionalType(subj)
		if ccType != "" {
			conventional++
			h.CommitTypeBreakdown[ccType]++
			if ccType == "fix" {
				fixCount++
			}
			if ccType == "feat" {
				featCount++
			}
		} else {
			h.CommitTypeBreakdown["other"]++
		}
	}

	n := len(data.Commits)
	h.TicketLinkedRatio = safeRatio(ticketLinked, n)
	h.ConventionalCommitsRatio = safeRatio(conventional, n)
	h.AvgMessageLength = float64(totalMsgLen) / float64(n)
	h.AvgFilesPerCommit = float64(totalFiles) / float64(n)
	h.MonorepoStyleRatio = safeRatio(monoStyleCount, n)

	total := fixCount + featCount
	if total > 0 {
		h.FixToFeatureRatio = float64(fixCount) / float64(total)
	}

	return h
}

func parseConventionalType(subject string) string {
	// Format: type(scope): description or type: description
	idx := indexOfChar(subject, ':')
	if idx < 1 || idx > 20 {
		return ""
	}
	candidate := subject[:idx]
	// Remove optional scope
	if parenIdx := indexOfChar(candidate, '('); parenIdx > 0 {
		candidate = candidate[:parenIdx]
	}
	if knownTypes[candidate] {
		return candidate
	}
	// Check for known types case-insensitively
	lower := toLower(candidate)
	if knownTypes[lower] {
		return lower
	}
	return ""
}

func indexOfChar(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func containsPattern(s, pattern string) bool {
	// Simple pattern matching for our specific patterns
	if pattern == `#[0-9]+` {
		return containsHashNumber(s)
	}
	if pattern == `[A-Z]+-[0-9]+` {
		return containsJiraStyle(s)
	}
	if contains(pattern, "fixes") || contains(pattern, "closes") || contains(pattern, "resolves") {
		lower := toLower(s)
		return contains(lower, "fixes #") || contains(lower, "closes #") || contains(lower, "resolves #")
	}
	return false
}

func containsHashNumber(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return true
		}
	}
	return false
}

func containsJiraStyle(s string) bool {
	// Pattern: [A-Z]+-[0-9]+
	i := 0
	for i < len(s) {
		// Find start of uppercase sequence
		if s[i] >= 'A' && s[i] <= 'Z' {
			start := i
			for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
				i++
			}
			if i > start && i < len(s) && s[i] == '-' {
				i++
				if i < len(s) && s[i] >= '0' && s[i] <= '9' {
					return true
				}
			}
		} else {
			i++
		}
	}
	return false
}

func safeRatio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
