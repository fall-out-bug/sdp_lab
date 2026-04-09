package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// LoadExistingFindings loads existing findings keys from Beads.
func (s *BeadsSink) LoadExistingFindings(ctx context.Context) error {
	output, err := bdListAll(ctx)
	if err != nil {
		return fmt.Errorf("list existing ci findings: %w", err)
	}

	var issues []beadsIssueSummary
	if err := json.Unmarshal(output, &issues); err != nil {
		return fmt.Errorf("parse existing ci findings: %w", err)
	}

	existing := make([]ExistingIssue, 0, len(issues))
	for _, issue := range issues {
		existing = append(existing, ExistingIssue(issue))
	}
	s.dedupe.ImportExisting(existing)

	return nil
}

// SyncProtocolFindings syncs protocol findings to Beads.
func (s *BeadsSink) SyncProtocolFindings(ctx context.Context, findings *ProtocolFindings) error {
	s.mu.Lock()
	s.stats.Processed += len(findings.Findings)
	s.mu.Unlock()

	for _, f := range findings.Findings {
		if err := s.syncProtocolFinding(ctx, &f, &findings.Source); err != nil {
			s.mu.Lock()
			s.stats.Failed++
			s.mu.Unlock()
			continue
		}
	}

	return nil
}

// SyncDocsFindings syncs docs findings to Beads.
func (s *BeadsSink) SyncDocsFindings(ctx context.Context, findings *DocsFindings) error {
	s.mu.Lock()
	s.stats.Processed += len(findings.Findings)
	s.mu.Unlock()

	for _, f := range findings.Findings {
		if err := s.syncDocsFinding(ctx, &f, &findings.Source); err != nil {
			s.mu.Lock()
			s.stats.Failed++
			s.mu.Unlock()
			continue
		}
	}

	return nil
}

func (s *BeadsSink) syncProtocolFinding(ctx context.Context, f *ProtocolFinding, source *FindingsSource) error {
	// Only create tasks for errors and warnings
	if f.Severity != "error" && f.Severity != "warning" {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	findingHash, payloadHash := ProtocolFindingHashes(*source, *f)
	decision := s.dedupe.Decide(findingHash, payloadHash)
	if decision.Action == DedupeSkip {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}
	if (decision.Action == DedupeUpdate || decision.Action == DedupeReopenUpdate) && decision.IssueID == "" {
		decision.Action = DedupeCreate
	}

	title := fmt.Sprintf("[%s] %s: %s", f.Category, f.Code, truncate(f.Message, 60))
	desc := s.buildProtocolDescription(f, source, findingHash, payloadHash)
	labels := s.buildLabels(FindingSourceCI, f.Severity, f.Category, f.Context.FeatureID, f.Context.WSID, true, findingHash, payloadHash)
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if _, err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
		return fmt.Errorf("sync protocol finding: %w", err)
	}
	return nil
}

func (s *BeadsSink) syncDocsFinding(ctx context.Context, f *DocsFinding, source *FindingsSource) error {
	// Only create tasks for errors and warnings
	if f.Severity != "error" && f.Severity != "warning" {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	findingHash, payloadHash := DocsFindingHashes(*source, *f)
	decision := s.dedupe.Decide(findingHash, payloadHash)
	if decision.Action == DedupeSkip {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}
	if (decision.Action == DedupeUpdate || decision.Action == DedupeReopenUpdate) && decision.IssueID == "" {
		decision.Action = DedupeCreate
	}

	title := fmt.Sprintf("[docs:%s] %s", f.Category, truncate(f.Message, 60))
	desc := s.buildDocsDescription(f, source, findingHash, payloadHash)
	labels := s.buildLabels(FindingSourceCI, f.Severity, f.Category, "", "", true, findingHash, payloadHash)
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if _, err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
		return fmt.Errorf("sync docs finding: %w", err)
	}
	return nil
}

func (s *BeadsSink) buildProtocolDescription(f *ProtocolFinding, source *FindingsSource, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "**Category:** %s\n", f.Category)
	fmt.Fprintf(&buf, "**Severity:** %s\n", f.Severity)
	fmt.Fprintf(&buf, "**File:** `%s`", f.File)
	if f.Line > 0 {
		fmt.Fprintf(&buf, ":%d", f.Line)
	}
	buf.WriteString("\n\n")
	fmt.Fprintf(&buf, "**Message:** %s\n\n", f.Message)

	if f.Remediation != nil {
		buf.WriteString("**Remediation:**\n")
		if f.Remediation.Hint != "" {
			fmt.Fprintf(&buf, "- %s\n", f.Remediation.Hint)
		}
		if f.Remediation.Template != "" {
			fmt.Fprintf(&buf, "```\n%s\n```\n", f.Remediation.Template)
		}
		if f.Remediation.DocURL != "" {
			fmt.Fprintf(&buf, "- Docs: %s\n", f.Remediation.DocURL)
		}
		buf.WriteString("\n")
	}

	if f.Context.FeatureID != "" || f.Context.WSID != "" {
		buf.WriteString("**Context:**\n")
		if f.Context.FeatureID != "" {
			fmt.Fprintf(&buf, "- Feature: %s\n", f.Context.FeatureID)
		}
		if f.Context.WSID != "" {
			fmt.Fprintf(&buf, "- Workstream: %s\n", f.Context.WSID)
		}
		buf.WriteString("\n")
	}

	fmt.Fprintf(&buf, "**Finding Hash:** `%s`\n", findingHash)
	fmt.Fprintf(&buf, "**Payload Hash:** `%s`\n\n", payloadHash)
	fmt.Fprintf(&buf, "---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID)

	return buf.String()
}

func (s *BeadsSink) buildDocsDescription(f *DocsFinding, source *FindingsSource, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "**Category:** %s\n", f.Category)
	fmt.Fprintf(&buf, "**Severity:** %s\n", f.Severity)
	fmt.Fprintf(&buf, "**File:** `%s`", f.File)
	if f.Line > 0 {
		fmt.Fprintf(&buf, ":%d", f.Line)
	}
	buf.WriteString("\n\n")
	fmt.Fprintf(&buf, "**Message:** %s\n\n", f.Message)

	if f.Context.LinkTarget != "" {
		fmt.Fprintf(&buf, "**Link Target:** `%s`\n", f.Context.LinkTarget)
		if f.Context.LinkText != "" {
			fmt.Fprintf(&buf, "**Link Text:** %s\n", f.Context.LinkText)
		}
		buf.WriteString("\n")
	}

	if f.Remediation != nil {
		buf.WriteString("**Remediation:**\n")
		if f.Remediation.Hint != "" {
			fmt.Fprintf(&buf, "- %s\n", f.Remediation.Hint)
		}
		if f.Remediation.SuggestedFix != "" {
			fmt.Fprintf(&buf, "Suggested: `%s`\n", f.Remediation.SuggestedFix)
		}
		buf.WriteString("\n")
	}

	fmt.Fprintf(&buf, "**Finding Hash:** `%s`\n", findingHash)
	fmt.Fprintf(&buf, "**Payload Hash:** `%s`\n\n", payloadHash)
	fmt.Fprintf(&buf, "---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID)

	return buf.String()
}
