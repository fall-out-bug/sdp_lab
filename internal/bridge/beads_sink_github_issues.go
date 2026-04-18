package bridge

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// wsidPattern matches the exact workstream ID format: 2 digits, hyphen,
// 3 digits, hyphen, 2 digits (e.g., 00-077-01).  Dates like 2026-04-18
// (4-2-2 digits) are correctly rejected.
var wsidPattern = regexp.MustCompile(`^\d{2}-\d{3}-\d{2}$`)

// SyncGitHubIssues converts GitHub issues into Beads tasks.
// It extracts feature/workstream tags from issue labels and title.
// Issues with "info" or "hint" severity are silently skipped, consistent with
// the CI findings path (see syncProtocolFinding / syncDocsFinding).
func (s *BeadsSink) SyncGitHubIssues(ctx context.Context, repo string, issues []GitHubIssue) error {
	s.mu.Lock()
	s.stats.Processed += len(issues)
	s.mu.Unlock()

	for i := range issues {
		// Handle closed issues: close the matching Beads issue if one exists.
		// Do not create new Beads issues just to close them.
		if strings.EqualFold(issues[i].State, "closed") {
			if err := s.syncClosedGitHubIssue(ctx, repo, &issues[i]); err != nil {
				s.mu.Lock()
				s.stats.Failed++
				s.mu.Unlock()
			}
			continue
		}

		severity := extractSeverityFromLabels(&issues[i])
		if severity == "info" || severity == "hint" {
			s.mu.Lock()
			s.stats.Skipped++
			s.mu.Unlock()
			continue
		}

		if err := s.syncGitHubIssue(ctx, repo, &issues[i]); err != nil {
			s.mu.Lock()
			s.stats.Failed++
			s.mu.Unlock()
			continue
		}
	}

	return nil
}

func (s *BeadsSink) syncGitHubIssue(ctx context.Context, repo string, issue *GitHubIssue) error {
	featureID := extractFeatureID(issue)
	wsID := extractWSID(issue)
	severity := extractSeverityFromLabels(issue)

	ghIssue := TypedFinding{
		Source:      FindingSourceGitHub,
		FeatureID:   featureID,
		WSID:        wsID,
		Blocking:    severity == "error",
		Title:       issue.Title,
		Summary:     truncate(issue.Title, 100),
		Description: buildGitHubIssueDescription(issue, repo),
		Severity:    severity,
		PRURL:       issue.issueURL(),
		DedupKey:    fmt.Sprintf("gh-issue:%s:%d", repo, issue.Number),
	}

	_, err := s.CreateTypedFinding(ctx, ghIssue)
	return err
}

// syncClosedGitHubIssue closes the Beads issue that corresponds to a closed
// GitHub issue.  It looks up the dedup key in the DedupeStore; if no matching
// Beads issue exists it simply skips — we never create an issue just to close it.
func (s *BeadsSink) syncClosedGitHubIssue(ctx context.Context, repo string, issue *GitHubIssue) error {
	dedupKey := fmt.Sprintf("gh-issue:%s:%d", repo, issue.Number)
	findingHash, _ := TypedFindingHashes(TypedFinding{
		Source:   FindingSourceGitHub,
		DedupKey: dedupKey,
		Title:    issue.Title,
	})

	s.dedupe.mu.RLock()
	record, exists := s.dedupe.records[findingHash]
	s.dedupe.mu.RUnlock()

	if !exists || record.IssueID == "" {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	// Already closed — nothing to do.
	if isClosedStatus(record.Status) {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	if s.dryRun {
		fmt.Printf("[DRY-RUN] Would close GitHub issue %s/%d -> Beads %s\n", repo, issue.Number, record.IssueID)
		s.dedupe.RecordClosed(findingHash)
		s.mu.Lock()
		s.stats.Closed++
		s.mu.Unlock()
		return nil
	}

	if err := s.closeBeadsIssue(ctx, record.IssueID, fmt.Sprintf("GitHub issue %s/%d closed", repo, issue.Number)); err != nil {
		return err
	}

	s.dedupe.RecordClosed(findingHash)
	s.mu.Lock()
	s.stats.Closed++
	s.mu.Unlock()
	return nil
}

func buildGitHubIssueDescription(issue *GitHubIssue, repo string) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "**GitHub Issue:** %s\n", issue.issueURL())
	fmt.Fprintf(&buf, "**Repository:** %s\n", repo)
	fmt.Fprintf(&buf, "**State:** %s\n", issue.State)

	if issue.User != nil && issue.User.Login != "" {
		fmt.Fprintf(&buf, "**Author:** @%s\n", issue.User.Login)
	}

	if len(issue.Assignees) > 0 {
		logins := make([]string, 0, len(issue.Assignees))
		for _, a := range issue.Assignees {
			logins = append(logins, "@"+a.Login)
		}
		fmt.Fprintf(&buf, "**Assignees:** %s\n", strings.Join(logins, ", "))
	}

	if issue.Milestone != nil && issue.Milestone.Title != "" {
		fmt.Fprintf(&buf, "**Milestone:** %s\n", issue.Milestone.Title)
	}

	if len(issue.Labels) > 0 {
		names := make([]string, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			names = append(names, l.Name)
		}
		fmt.Fprintf(&buf, "**Labels:** %s\n", strings.Join(names, ", "))
	}

	if issue.Body != "" {
		fmt.Fprintf(&buf, "\n---\n\n%s\n", issue.Body)
	}

	return buf.String()
}

// extractFeatureID tries to extract a feature ID (e.g., F077) from labels or title.
func extractFeatureID(issue *GitHubIssue) string {
	// First check labels for feature tag (e.g., "feature/F077", "F077")
	for _, label := range issue.Labels {
		name := strings.TrimSpace(label.Name)
		if strings.HasPrefix(strings.ToLower(name), "feature/") {
			// Slice the original string to preserve the case of the feature ID.
			return name[len("feature/"):]
		}
		if looksLikeFeatureID(name) {
			return name
		}
	}

	// Then check title for F### pattern
	return extractFeatureIDFromText(issue.Title)
}

// extractWSID tries to extract a workstream ID (e.g., 00-077-02) from labels or title.
func extractWSID(issue *GitHubIssue) string {
	for _, label := range issue.Labels {
		name := strings.TrimSpace(label.Name)
		if strings.HasPrefix(strings.ToLower(name), "workstream/") {
			// TrimPrefix must operate on the lowercased string to match the
			// case-insensitive HasPrefix check above.
			return strings.TrimPrefix(strings.ToLower(name), "workstream/")
		}
		if isWSID(name) {
			return name
		}
	}

	return extractWSIDFromText(issue.Title)
}

// extractSeverityFromLabels determines severity from issue labels.
func extractSeverityFromLabels(issue *GitHubIssue) string {
	for _, label := range issue.Labels {
		name := strings.ToLower(strings.TrimSpace(label.Name))
		switch name {
		case "bug", "critical", "error", "severity:critical", "severity:error":
			return "error"
		case "enhancement", "improvement", "warning", "severity:warning":
			return "warning"
		case "documentation", "info", "severity:info":
			return "info"
		}
	}
	return "warning"
}

// looksLikeFeatureID checks if a string looks like a feature ID (F followed by digits).
func looksLikeFeatureID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != 'F' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// extractFeatureIDFromText extracts F### from text.
func extractFeatureIDFromText(text string) string {
	words := strings.Fields(text)
	for _, w := range words {
		// TrimRight is intentional: it strips individual trailing chars from the
		// cutset ":,;-", not a suffix string. TrimSuffix would require exact match.
		w = strings.TrimRight(w, ":,;-")
		if looksLikeFeatureID(w) {
			return w
		}
	}
	return ""
}

// isWSID checks if a string matches the exact workstream ID format (##-###-##).
// The regex enforces exactly 2-3-2 digits so that dates like 2026-04-18 (4-2-2)
// are correctly rejected.
func isWSID(s string) bool {
	s = strings.TrimSpace(s)
	return wsidPattern.MatchString(s)
}

// extractWSIDFromText extracts ##-###-## pattern from text.
func extractWSIDFromText(text string) string {
	words := strings.Fields(text)
	for _, w := range words {
		// TrimRight is intentional: it strips individual trailing chars from the
		// cutset ":,;-", not a suffix string. TrimSuffix would require exact match.
		w = strings.TrimRight(w, ":,;-")
		if isWSID(w) {
			return w
		}
	}
	return ""
}
