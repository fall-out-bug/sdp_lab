package f165

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type NormalizeResult struct {
	CleanText           string
	ZeroWidthCount      int
	HTMLCommentsRemoved []string
	SuspiciousLinks     []string
	UntrustedNarrative  string
}

type ParseResult struct {
	TypedFields      map[string]any
	ParseError       bool
	ParseErrorReason string
}

type WrapResult struct {
	TypedFields        map[string]any
	UntrustedNarrative string
	BoundaryMarker     string
}

type ValidateResult struct {
	Verdict            string
	BlockedReason      string
	TrustedEvidenceRef string
	Violations         []string
}

func Normalize(input string) NormalizeResult {
	var result NormalizeResult
	var stripped strings.Builder
	for _, r := range input {
		if isZeroWidth(r) {
			result.ZeroWidthCount++
			continue
		}
		stripped.WriteRune(r)
	}
	intermediate := stripped.String()
	intermediate, result.HTMLCommentsRemoved = stripHTMLComments(intermediate)
	result.SuspiciousLinks = extractSuspiciousMarkdownLinks(intermediate)
	result.UntrustedNarrative = intermediate
	result.CleanText = intermediate
	return result
}

func Parse(normalized NormalizeResult, vector string) ParseResult {
	fields := make(map[string]any)
	fields["vector"] = vector
	switch vector {
	case "beads_issue":
		title, desc, err := parseBeadsIssue(normalized.CleanText)
		if err != nil {
			return ParseResult{ParseError: true, ParseErrorReason: err.Error()}
		}
		fields["title"] = title
		fields["description"] = desc
	case "workstream_markdown":
		acs, scope, err := parseWorkstreamMarkdown(normalized.CleanText)
		if err != nil {
			return ParseResult{ParseError: true, ParseErrorReason: err.Error()}
		}
		fields["acceptance_criteria"] = acs
		fields["scope"] = scope
	case "evidence_finding":
		log, result, err := parseEvidenceFinding(normalized.CleanText)
		if err != nil {
			return ParseResult{ParseError: true, ParseErrorReason: err.Error()}
		}
		fields["ci_log"] = log
		fields["test_result"] = result
	case "cross_agent_handoff":
		fields["handoff_narrative"] = normalized.CleanText
		fields["unsupported_surface"] = true
	case "mcp_resource":
		fields["resource_text"] = normalized.CleanText
	default:
		return ParseResult{ParseError: true, ParseErrorReason: fmt.Sprintf("unknown vector: %s", vector)}
	}
	fields["zero_width_count"] = normalized.ZeroWidthCount
	fields["html_comments_removed"] = normalized.HTMLCommentsRemoved
	fields["suspicious_links"] = normalized.SuspiciousLinks
	return ParseResult{TypedFields: fields}
}

func Wrap(parsed ParseResult, normalized NormalizeResult) WrapResult {
	const boundary = "---UNTRUSTED-NARRATIVE-BOUNDARY---"
	return WrapResult{
		TypedFields:        parsed.TypedFields,
		UntrustedNarrative: normalized.UntrustedNarrative,
		BoundaryMarker:     boundary,
	}
}

func Validate(wrapped WrapResult, snapshot TrustedStateSnapshot, proposedAction string, proposedClaim string) ValidateResult {
	var violations []string
	vector, _ := wrapped.TypedFields["vector"].(string)
	if vector == "" {
		if snapshot.BeadsIssueID != "" {
			vector = "beads_issue"
		} else if snapshot.WorkstreamID != "" {
			vector = "workstream_markdown"
		} else if snapshot.EvidenceRef != "" {
			vector = "evidence_finding"
		}
	}
	if vector == "cross_agent_handoff" {
		return ValidateResult{
			Verdict:       "residual_risk",
			BlockedReason: BlockedReasonUnsupportedResidualRisk,
			Violations:    []string{"cross_agent_handoff is unsupported_surface in F165"},
		}
	}
	switch vector {
	case "beads_issue":
		lowerClaim := strings.ToLower(proposedClaim)
		if strings.Contains(lowerClaim, "closed") ||
			strings.Contains(lowerClaim, "complete") ||
			strings.Contains(lowerClaim, "resolved") ||
			strings.Contains(lowerClaim, "done") ||
			strings.Contains(lowerClaim, "fixed") ||
			strings.Contains(lowerClaim, "shipped") ||
			strings.Contains(lowerClaim, "landed") {
			if snapshot.BeadsStatus != "closed" {
				violations = append(violations,
					fmt.Sprintf("untrusted completion claim: proposed=%q, trusted_status=%q", proposedClaim, snapshot.BeadsStatus))
			}
		}
	case "workstream_markdown":
		if proposedAction != "" && len(snapshot.WorkstreamScope) > 0 {
			if !actionInScope(proposedAction, snapshot.WorkstreamScope) {
				violations = append(violations,
					fmt.Sprintf("scope policy conflict: proposed_action=%q, trusted_scope=%v", proposedAction, snapshot.WorkstreamScope))
			}
		}
	case "evidence_finding":
		lowerClaim := strings.ToLower(proposedClaim)
		if strings.Contains(lowerClaim, "pass") || strings.Contains(lowerClaim, "green") {
			if snapshot.ToolExitCode != 0 {
				violations = append(violations,
					fmt.Sprintf("evidence source mismatch: proposed=%q, trusted_exit_code=%d", proposedClaim, snapshot.ToolExitCode))
			}
		}
	}
	if isWriteAction(proposedAction) && !snapshotHasWriteAuthorization(snapshot) {
		violations = append(violations,
			fmt.Sprintf("write without trusted authorization: proposed_action=%q", proposedAction))
	}
	if len(violations) > 0 {
		return ValidateResult{
			Verdict:       "blocked",
			BlockedReason: selectBlockedReason(vector, violations),
			Violations:    violations,
		}
	}
	return ValidateResult{Verdict: "clean", TrustedEvidenceRef: snapshot.EvidenceRef}
}

func isZeroWidth(r rune) bool {
	switch r {
	case '\u200B', '\u200C', '\u200D', '\uFEFF', '\u2060', '\u180E',
		'\u200E', '\u200F', '\u2061', '\u2062', '\u2063', '\u2064':
		return true
	}
	return false
}

var htmlCommentRe = regexp.MustCompile(`(?s)<!--(.*?)-->`)

func stripHTMLComments(input string) (string, []string) {
	var removed []string
	clean := htmlCommentRe.ReplaceAllStringFunc(input, func(match string) string {
		content := htmlCommentRe.FindStringSubmatch(match)
		if len(content) > 1 {
			removed = append(removed, strings.TrimSpace(content[1]))
		}
		return ""
	})
	return clean, removed
}

var markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

func extractSuspiciousMarkdownLinks(input string) []string {
	var suspicious []string
	matches := markdownLinkRe.FindAllStringSubmatch(input, -1)
	for _, m := range matches {
		if len(m) > 2 {
			target := m[2]
			if strings.HasPrefix(target, "javascript:") ||
				strings.HasPrefix(target, "data:") ||
				strings.HasPrefix(target, "mock://") ||
				target == "" {
				suspicious = append(suspicious, target)
			}
		}
	}
	return suspicious
}

func parseBeadsIssue(text string) (string, string, error) {
	lines := strings.Split(text, "\n")
	var title, desc string
	var inDesc bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "issue title:") {
			idx := strings.Index(strings.ToLower(line), "issue title:")
			title = strings.TrimSpace(line[idx+len("issue title:"):])
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "issue description:") {
			inDesc = true
			continue
		}
		if inDesc {
			desc += line + "\n"
		}
	}
	if title == "" {
		return "", "", fmt.Errorf("missing required field: title")
	}
	return title, strings.TrimSpace(desc), nil
}

func parseWorkstreamMarkdown(text string) ([]string, []string, error) {
	lines := strings.Split(text, "\n")
	var acs, scope []string
	var inAC, inScope bool
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "acceptance criteria") {
			inAC = true
			inScope = false
			continue
		}
		if strings.Contains(lower, "scope") && strings.Contains(lower, "files") {
			inScope = true
			inAC = false
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			inAC = false
			inScope = false
			continue
		}
		if inAC && strings.HasPrefix(trimmed, "- [") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ]"))
			item = strings.TrimSpace(strings.TrimPrefix(item, "- [x]"))
			acs = append(acs, item)
		}
		if inScope && strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			scope = append(scope, item)
		}
	}
	if len(acs) == 0 {
		return nil, nil, fmt.Errorf("missing required field: acceptance_criteria")
	}
	return acs, scope, nil
}

func parseEvidenceFinding(text string) (string, string, error) {
	lines := strings.Split(text, "\n")
	var logLines []string
	var result string
	var inLog bool
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "ci log:") {
			inLog = true
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "test result:") || strings.HasPrefix(strings.ToLower(line), "result:") {
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "test result:") {
				idx := strings.Index(lower, "test result:")
				result = strings.TrimSpace(line[idx+len("test result:"):])
			} else {
				idx := strings.Index(lower, "result:")
				result = strings.TrimSpace(line[idx+len("result:"):])
			}
			inLog = false
			continue
		}
		if inLog {
			logLines = append(logLines, line)
		}
	}
	if len(logLines) == 0 && result == "" {
		return "", "", fmt.Errorf("missing required field: ci_log or test_result")
	}
	return strings.Join(logLines, "\n"), result, nil
}

func actionInScope(action string, scope []string) bool {
	// Check each space-separated token; at least one path token must be within scope.
	for _, token := range strings.Fields(action) {
		if !strings.Contains(token, "/") {
			continue
		}
		cleanToken := filepath.Clean(token)
		for _, s := range scope {
			cleanScope := filepath.Clean(s)
			if strings.HasPrefix(cleanToken, cleanScope) {
				return true
			}
		}
	}
	return false
}

func isWriteAction(action string) bool {
	for _, t := range strings.Fields(strings.ToLower(action)) {
		switch t {
		case "write", "edit", "create", "delete", "push", "merge", "close", "bd", "git", "gh", "curl", "wget", "ssh":
			return true
		}
	}
	return false
}

func snapshotHasWriteAuthorization(snapshot TrustedStateSnapshot) bool {
	// Demo-only: in production, write authorization requires a validated
	// evidence envelope from a deterministic tool or explicit human approval.
	// Fixture narratives are test data and are accepted here only for demo.
	return snapshot.EvidenceRef != "" || snapshot.TrustedNarrative != ""
}

func selectBlockedReason(vector string, violations []string) string {
	for _, v := range violations {
		lower := strings.ToLower(v)
		switch {
		case strings.Contains(lower, "untrusted completion claim"):
			return BlockedReasonUntrustedCompletionClaim
		case strings.Contains(lower, "scope policy conflict"):
			return BlockedReasonScopePolicyConflict
		case strings.Contains(lower, "evidence source mismatch"):
			return BlockedReasonEvidenceSourceMismatch
		case strings.Contains(lower, "write without trusted authorization"):
			return BlockedReasonWriteWithoutTrustedAuthorization
		case strings.Contains(lower, "parse error"):
			return BlockedReasonParseError
		case strings.Contains(lower, "policy conflict"):
			return BlockedReasonPolicyConflict
		}
	}
	return BlockedReasonPolicyConflict
}
