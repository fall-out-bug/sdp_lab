package bridge

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"strings"
)

func (s *BeadsSink) CreateTypedFinding(ctx context.Context, finding TypedFinding) (string, error) {
	findingHash, payloadHash := TypedFindingHashes(finding)
	decision := s.dedupe.Decide(findingHash, payloadHash)
	if decision.Action == DedupeSkip {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return decision.IssueID, nil
	}
	if (decision.Action == DedupeUpdate || decision.Action == DedupeReopenUpdate) && decision.IssueID == "" {
		decision.Action = DedupeCreate
	}

	title := buildTypedFindingTitle(finding)
	description := buildTypedFindingDescription(finding, findingHash, payloadHash)
	priority := finding.Priority
	if priority == 0 {
		priority = severityToPriority(finding.Severity)
	}
	labels := s.buildLabels(finding.Source, finding.Severity, "", finding.FeatureID, finding.WSID, finding.Blocking, findingHash, payloadHash)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return decision.IssueID, nil
	}

	return s.applyDecision(ctx, decision, title, description, priority, labels)
}

func (s *BeadsSink) CreateReviewFinding(ctx context.Context, input ReviewFindingInput) (string, error) {
	return s.CreateTypedFinding(ctx, TypedFinding{
		Source:       FindingSourceReview,
		FeatureID:    input.FeatureID,
		WSID:         input.WSID,
		Blocking:     input.Blocking,
		Title:        input.Title,
		Summary:      buildReviewSummary(input),
		Description:  buildReviewDescription(input),
		Severity:     input.Severity,
		Priority:     input.Priority,
		PRURL:        input.PRURL,
		ArtifactRef:  input.ArtifactRef,
		EvidenceRef:  input.EvidenceRef,
		TraceRef:     input.TraceRef,
		DriftVerdict: input.DriftVerdict,
		DedupKey:     firstNonEmpty(input.DedupKey, input.Role+":"+input.Title),
	})
}

func (s *BeadsSink) CreateQAFinding(ctx context.Context, input QAFindingInput) (string, error) {
	return s.CreateTypedFinding(ctx, TypedFinding{
		Source:      FindingSourceQA,
		FeatureID:   input.FeatureID,
		WSID:        input.WSID,
		Blocking:    input.Blocking,
		Title:       input.Title,
		Summary:     buildQASummary(input),
		Description: buildQADescription(input),
		Severity:    input.Severity,
		Priority:    input.Priority,
		PRURL:       input.PRURL,
		ArtifactRef: input.ArtifactRef,
		EvidenceRef: input.EvidenceRef,
		TraceRef:    input.TraceRef,
		DedupKey:    firstNonEmpty(input.DedupKey, input.Scenario+":"+input.Title+":"+input.FailedStep),
	})
}

func buildTypedFindingTitle(finding TypedFinding) string {
	if finding.Title != "" {
		return finding.Title
	}
	parts := []string{}
	if finding.FeatureID != "" {
		parts = append(parts, finding.FeatureID)
	}
	if finding.WSID != "" {
		parts = append(parts, finding.WSID)
	}
	prefix := cmp.Or(strings.Join(parts, " "), strings.ToUpper(string(finding.Source)))
	if finding.Summary != "" {
		return fmt.Sprintf("%s: %s", prefix, truncate(finding.Summary, 72))
	}
	return fmt.Sprintf("%s: finding", prefix)
}

func buildTypedFindingDescription(finding TypedFinding, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "**Source:** %s\n", finding.Source)
	if finding.FeatureID != "" {
		fmt.Fprintf(&buf, "**Feature:** %s\n", finding.FeatureID)
	}
	if finding.WSID != "" {
		fmt.Fprintf(&buf, "**Workstream:** %s\n", finding.WSID)
	}
	fmt.Fprintf(&buf, "**Blocking:** %t\n", finding.Blocking)
	if finding.Severity != "" {
		fmt.Fprintf(&buf, "**Severity:** %s\n", finding.Severity)
	}
	if finding.Priority > 0 {
		fmt.Fprintf(&buf, "**Priority:** P%d\n", finding.Priority)
	}
	buf.WriteString("\n")
	if finding.Summary != "" {
		fmt.Fprintf(&buf, "**Summary:** %s\n\n", finding.Summary)
	}
	if finding.Description != "" {
		fmt.Fprintf(&buf, "**Description:** %s\n\n", finding.Description)
	}
	if finding.EvidenceRef != "" || finding.ArtifactRef != "" || finding.PRURL != "" || finding.TraceRef != "" || finding.DriftVerdict != "" {
		buf.WriteString("**References:**\n")
		if finding.EvidenceRef != "" {
			fmt.Fprintf(&buf, "- Evidence: %s\n", finding.EvidenceRef)
		}
		if finding.ArtifactRef != "" {
			fmt.Fprintf(&buf, "- Artifact: %s\n", finding.ArtifactRef)
		}
		if finding.PRURL != "" {
			fmt.Fprintf(&buf, "- PR: %s\n", finding.PRURL)
		}
		if finding.TraceRef != "" {
			fmt.Fprintf(&buf, "- Trace: %s\n", finding.TraceRef)
		}
		if finding.DriftVerdict != "" {
			fmt.Fprintf(&buf, "- Drift: %s\n", finding.DriftVerdict)
		}
		buf.WriteString("\n")
	}
	fmt.Fprintf(&buf, "**Finding Hash:** `%s`\n", findingHash)
	fmt.Fprintf(&buf, "**Payload Hash:** `%s`\n", payloadHash)

	return buf.String()
}

func buildReviewSummary(input ReviewFindingInput) string {
	if input.Summary != "" {
		return input.Summary
	}
	if input.Role != "" && input.Title != "" {
		return fmt.Sprintf("%s review finding: %s", input.Role, input.Title)
	}
	return input.Title
}

func buildReviewDescription(input ReviewFindingInput) string {
	var buf bytes.Buffer
	if input.Role != "" {
		fmt.Fprintf(&buf, "Reviewer role: %s\n", input.Role)
	}
	if input.Description != "" {
		buf.WriteString(strings.TrimSpace(input.Description))
	}
	return strings.TrimSpace(buf.String())
}

func buildQASummary(input QAFindingInput) string {
	if input.Summary != "" {
		return input.Summary
	}
	parts := []string{}
	if input.Scenario != "" {
		parts = append(parts, input.Scenario)
	}
	if input.FailedStep != "" {
		parts = append(parts, "failed at "+input.FailedStep)
	}
	if input.Title != "" {
		parts = append(parts, input.Title)
	}
	return strings.Join(parts, ": ")
}

func buildQADescription(input QAFindingInput) string {
	var lines []string
	if input.Description != "" {
		lines = append(lines, strings.TrimSpace(input.Description))
	}
	if input.Scenario != "" {
		lines = append(lines, "Scenario: "+strings.TrimSpace(input.Scenario))
	}
	if input.FailedStep != "" {
		lines = append(lines, "Failed step: "+strings.TrimSpace(input.FailedStep))
	}
	if input.ExpectedOutcome != "" {
		lines = append(lines, "Expected: "+strings.TrimSpace(input.ExpectedOutcome))
	}
	if input.ActualOutcome != "" {
		lines = append(lines, "Actual: "+strings.TrimSpace(input.ActualOutcome))
	}
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
