package f165

import (
	"strings"
	"testing"
)

func TestNormalize_ZeroWidthChars(t *testing.T) {
	input := "Hello\u200BWorld\u200C"
	result := Normalize(input)
	if result.ZeroWidthCount != 2 {
		t.Errorf("zero_width_count = %d, want 2", result.ZeroWidthCount)
	}
	if strings.Contains(result.CleanText, "\u200B") {
		t.Error("clean text still contains zero-width char")
	}
}

func TestNormalize_HTMLComments(t *testing.T) {
	input := "Safe text <!-- hidden instruction --> more text"
	result := Normalize(input)
	if len(result.HTMLCommentsRemoved) != 1 {
		t.Fatalf("html_comments_removed = %d, want 1", len(result.HTMLCommentsRemoved))
	}
	if result.HTMLCommentsRemoved[0] != "hidden instruction" {
		t.Errorf("comment content = %q, want %q", result.HTMLCommentsRemoved[0], "hidden instruction")
	}
}

func TestNormalize_HTMLComments_Multiline(t *testing.T) {
	input := "Safe text <!-- hidden \n multiline \n instruction --> more text"
	result := Normalize(input)
	if len(result.HTMLCommentsRemoved) != 1 {
		t.Fatalf("html_comments_removed = %d, want 1", len(result.HTMLCommentsRemoved))
	}
	if !strings.Contains(result.HTMLCommentsRemoved[0], "multiline") {
		t.Errorf("multiline comment not captured: %q", result.HTMLCommentsRemoved[0])
	}
}

func TestNormalize_SuspiciousMarkdownLinks(t *testing.T) {
	input := "[link1](javascript:alert(1)) [link2](data:text/html) [link3](mock://sanitized)"
	result := Normalize(input)
	if len(result.SuspiciousLinks) != 3 {
		t.Errorf("suspicious_links = %d, want 3", len(result.SuspiciousLinks))
	}
}

func TestParse_BeadsIssue_Success(t *testing.T) {
	text := "Issue title: Fix login\nIssue description:\nUsers report timeout."
	norm := Normalize(text)
	parsed := Parse(norm, "beads_issue")
	if parsed.ParseError {
		t.Fatalf("unexpected parse error: %s", parsed.ParseErrorReason)
	}
	if parsed.TypedFields["title"] != "Fix login" {
		t.Errorf("title = %v, want %q", parsed.TypedFields["title"], "Fix login")
	}
}

func TestParse_BeadsIssue_CaseInsensitivePrefix(t *testing.T) {
	text := "issue title: Fix login\nissue description:\nUsers report timeout."
	norm := Normalize(text)
	parsed := Parse(norm, "beads_issue")
	if parsed.ParseError {
		t.Fatalf("unexpected parse error: %s", parsed.ParseErrorReason)
	}
	if parsed.TypedFields["title"] != "Fix login" {
		t.Errorf("title = %v, want %q", parsed.TypedFields["title"], "Fix login")
	}
}

func TestParse_WorkstreamMarkdown_Success(t *testing.T) {
	text := "## Acceptance Criteria\n- [ ] Implement parser\n- [ ] Tests pass"
	norm := Normalize(text)
	parsed := Parse(norm, "workstream_markdown")
	if parsed.ParseError {
		t.Fatalf("unexpected parse error: %s", parsed.ParseErrorReason)
	}
	acs, ok := parsed.TypedFields["acceptance_criteria"].([]string)
	if !ok || len(acs) != 2 {
		t.Errorf("acceptance_criteria = %v, want 2 items", parsed.TypedFields["acceptance_criteria"])
	}
}

func TestValidate_BeadsIssue_BlocksCompletionClaim(t *testing.T) {
	snapshot := TrustedStateSnapshot{BeadsIssueID: "sdplab-123", BeadsStatus: "open"}
	wrapped := WrapResult{TypedFields: map[string]any{"vector": "beads_issue"}}
	result := Validate(wrapped, snapshot, "", "issue is resolved and may be closed")
	if result.Verdict != "blocked" {
		t.Fatalf("verdict = %q, want blocked", result.Verdict)
	}
	if result.BlockedReason != BlockedReasonUntrustedCompletionClaim {
		t.Errorf("blocked_reason = %q, want %q", result.BlockedReason, BlockedReasonUntrustedCompletionClaim)
	}
}

func TestValidate_Workstream_BlocksPathTraversal(t *testing.T) {
	snapshot := TrustedStateSnapshot{WorkstreamID: "00-165-01", WorkstreamScope: []string{"internal/evals/"}}
	wrapped := WrapResult{TypedFields: map[string]any{"vector": "workstream_markdown"}}
	result := Validate(wrapped, snapshot, "edit internal/evals/../../etc/passwd", "")
	if result.Verdict != "blocked" {
		t.Fatalf("verdict = %q, want blocked for path traversal", result.Verdict)
	}
}

func TestValidate_Workstream_PassesInScope(t *testing.T) {
	snapshot := TrustedStateSnapshot{WorkstreamID: "00-165-01", WorkstreamScope: []string{"internal/evals/"}, TrustedNarrative: "authorized"}
	wrapped := WrapResult{TypedFields: map[string]any{"vector": "workstream_markdown"}}
	result := Validate(wrapped, snapshot, "edit internal/evals/foo.go", "")
	if result.Verdict != "clean" {
		t.Fatalf("verdict = %q, want clean", result.Verdict)
	}
}

func TestBlockedReasonClosedSet(t *testing.T) {
	invalid := []string{"other", "", "malicious_reason"}
	for _, r := range invalid {
		if IsValidBlockedReason(r) {
			t.Errorf("expected %q to be invalid blocked_reason", r)
		}
	}
}

func TestResidualRiskClosedSet(t *testing.T) {
	invalid := []string{"other", "", "some_risk"}
	for _, c := range invalid {
		if IsValidResidualRiskCategory(c) {
			t.Errorf("expected %q to be invalid residual_risk_category", c)
		}
	}
}
