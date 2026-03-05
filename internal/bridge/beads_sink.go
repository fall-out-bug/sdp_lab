package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"
)

// SyncStats tracks synchronization statistics.
type SyncStats struct {
	Processed int `json:"processed"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

type beadsIssueSummary struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
}

// BeadsSink creates and updates Beads tasks from findings.
type BeadsSink struct {
	mu     sync.RWMutex
	prefix string // Issue prefix (e.g., "sdplab-")
	dryRun bool
	labels []string
	stats  SyncStats
	dedupe *DedupeStore
}

// NewBeadsSink creates a new Beads sink.
func NewBeadsSink(prefix string, dryRun bool, defaultLabels []string) *BeadsSink {
	return &BeadsSink{
		prefix: prefix,
		dryRun: dryRun,
		labels: defaultLabels,
		dedupe: NewDedupeStore(),
	}
}

// GetStats returns the current sync statistics.
func (s *BeadsSink) GetStats() SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// LoadExistingFindings loads existing findings keys from Beads.
func (s *BeadsSink) LoadExistingFindings(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "bd", "list", "--all", "-l", "ci-finding", "--json", "-n", "0")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list existing ci findings: %w", err)
	}

	var issues []beadsIssueSummary
	if err := json.Unmarshal(output, &issues); err != nil {
		return fmt.Errorf("parse existing ci findings: %w", err)
	}

	existing := make([]ExistingIssue, 0, len(issues))
	for _, issue := range issues {
		existing = append(existing, ExistingIssue{
			ID:     issue.ID,
			Status: issue.Status,
			Labels: issue.Labels,
		})
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

	// Build title
	title := fmt.Sprintf("[%s] %s: %s", f.Category, f.Code, truncate(f.Message, 60))

	// Build description
	desc := s.buildProtocolDescription(f, source, findingHash, payloadHash)

	// Build labels
	labels := s.buildLabels(f.Severity, f.Category, f.Context.FeatureID, f.Context.WSID, findingHash, payloadHash)

	// Determine priority
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
		return err
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

	// Build title
	title := fmt.Sprintf("[docs:%s] %s", f.Category, truncate(f.Message, 60))

	// Build description
	desc := s.buildDocsDescription(f, source, findingHash, payloadHash)

	// Build labels
	labels := s.buildLabels(f.Severity, f.Category, "", "", findingHash, payloadHash)

	// Determine priority
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
		return err
	}
	return nil
}

func (s *BeadsSink) buildProtocolDescription(f *ProtocolFinding, source *FindingsSource, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("**Category:** %s\n", f.Category))
	buf.WriteString(fmt.Sprintf("**Severity:** %s\n", f.Severity))
	buf.WriteString(fmt.Sprintf("**File:** `%s`", f.File))
	if f.Line > 0 {
		buf.WriteString(fmt.Sprintf(":%d", f.Line))
	}
	buf.WriteString("\n\n")
	buf.WriteString(fmt.Sprintf("**Message:** %s\n\n", f.Message))

	if f.Remediation != nil {
		buf.WriteString("**Remediation:**\n")
		if f.Remediation.Hint != "" {
			buf.WriteString(fmt.Sprintf("- %s\n", f.Remediation.Hint))
		}
		if f.Remediation.Template != "" {
			buf.WriteString(fmt.Sprintf("```\n%s\n```\n", f.Remediation.Template))
		}
		if f.Remediation.DocURL != "" {
			buf.WriteString(fmt.Sprintf("- Docs: %s\n", f.Remediation.DocURL))
		}
		buf.WriteString("\n")
	}

	if f.Context.FeatureID != "" || f.Context.WSID != "" {
		buf.WriteString("**Context:**\n")
		if f.Context.FeatureID != "" {
			buf.WriteString(fmt.Sprintf("- Feature: %s\n", f.Context.FeatureID))
		}
		if f.Context.WSID != "" {
			buf.WriteString(fmt.Sprintf("- Workstream: %s\n", f.Context.WSID))
		}
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("**Finding Hash:** `%s`\n", findingHash))
	buf.WriteString(fmt.Sprintf("**Payload Hash:** `%s`\n\n", payloadHash))
	buf.WriteString(fmt.Sprintf("---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID))

	return buf.String()
}

func (s *BeadsSink) buildDocsDescription(f *DocsFinding, source *FindingsSource, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("**Category:** %s\n", f.Category))
	buf.WriteString(fmt.Sprintf("**Severity:** %s\n", f.Severity))
	buf.WriteString(fmt.Sprintf("**File:** `%s`", f.File))
	if f.Line > 0 {
		buf.WriteString(fmt.Sprintf(":%d", f.Line))
	}
	buf.WriteString("\n\n")
	buf.WriteString(fmt.Sprintf("**Message:** %s\n\n", f.Message))

	if f.Context.LinkTarget != "" {
		buf.WriteString(fmt.Sprintf("**Link Target:** `%s`\n", f.Context.LinkTarget))
		if f.Context.LinkText != "" {
			buf.WriteString(fmt.Sprintf("**Link Text:** %s\n", f.Context.LinkText))
		}
		buf.WriteString("\n")
	}

	if f.Remediation != nil {
		buf.WriteString("**Remediation:**\n")
		if f.Remediation.Hint != "" {
			buf.WriteString(fmt.Sprintf("- %s\n", f.Remediation.Hint))
		}
		if f.Remediation.SuggestedFix != "" {
			buf.WriteString(fmt.Sprintf("Suggested: `%s`\n", f.Remediation.SuggestedFix))
		}
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("**Finding Hash:** `%s`\n", findingHash))
	buf.WriteString(fmt.Sprintf("**Payload Hash:** `%s`\n\n", payloadHash))
	buf.WriteString(fmt.Sprintf("---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID))

	return buf.String()
}

func (s *BeadsSink) buildLabels(severity, category, featureID, wsID, findingHash, payloadHash string) []string {
	labels := []string{"ci-finding", severity}

	if category != "" {
		labels = append(labels, category)
	}

	if featureID != "" {
		labels = append(labels, featureID)
	}

	if wsID != "" {
		labels = append(labels, wsID)
	}

	if findingHash != "" {
		labels = append(labels, findingHashLabel(findingHash))
	}

	if payloadHash != "" {
		labels = append(labels, payloadHashLabel(payloadHash))
	}

	// Add default labels
	labels = append(labels, s.labels...)

	return labels
}

func (s *BeadsSink) createBeadsIssue(ctx context.Context, title, description string, priority int, labels []string) (string, error) {
	args := []string{
		"create",
		"--silent",
		"--prefix", s.prefix,
		"-p", fmt.Sprintf("%d", priority),
		"-t", "task",
		"-l", strings.Join(labels, ","),
		"-d", description,
		title,
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	issueID := strings.TrimSpace(string(output))
	if issueID == "" {
		return "", fmt.Errorf("bd create returned empty issue id")
	}

	return issueID, nil
}

func (s *BeadsSink) updateBeadsIssue(ctx context.Context, issueID, title, description string, priority int, labels []string) error {
	args := []string{"update", issueID, "--title", title, "-d", description, "-p", fmt.Sprintf("%d", priority)}
	for _, label := range labels {
		args = append(args, "--add-label", label)
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd update %s failed: %w: %s", issueID, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *BeadsSink) reopenBeadsIssue(ctx context.Context, issueID, reason string) error {
	args := []string{"reopen", issueID}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	cmd := exec.CommandContext(ctx, "bd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd reopen %s failed: %w: %s", issueID, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *BeadsSink) handleDryRunDecision(decision DedupeDecision, findingHash, payloadHash, title string) {
	switch decision.Action {
	case DedupeCreate:
		fmt.Printf("[DRY-RUN] Would create: %s\n", title)
		s.dedupe.RecordCreated(findingHash, payloadHash, "dry-run")
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
	case DedupeUpdate:
		fmt.Printf("[DRY-RUN] Would update: %s\n", decision.IssueID)
		s.dedupe.RecordUpdated(findingHash, payloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
	case DedupeReopenUpdate:
		fmt.Printf("[DRY-RUN] Would reopen+update: %s\n", decision.IssueID)
		s.dedupe.RecordUpdated(findingHash, payloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
	default:
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
	}
}

func (s *BeadsSink) applyDecision(ctx context.Context, decision DedupeDecision, title, description string, priority int, labels []string) error {
	switch decision.Action {
	case DedupeCreate:
		issueID, err := s.createBeadsIssue(ctx, title, description, priority, labels)
		if err != nil {
			return err
		}
		s.dedupe.RecordCreated(decision.FindingHash, decision.PayloadHash, issueID)
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
		return nil
	case DedupeUpdate:
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return nil
	case DedupeReopenUpdate:
		if err := s.reopenBeadsIssue(ctx, decision.IssueID, "finding payload changed"); err != nil {
			return err
		}
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return nil
	default:
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}
}

// severityToPriority maps finding severity to Beads priority.
func severityToPriority(severity string) int {
	switch severity {
	case "error":
		return 1 // P1
	case "warning":
		return 2 // P2
	case "info":
		return 3 // P3
	default:
		return 4 // P4
	}
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PrintSummary prints a summary of the sync.
func (s *BeadsSink) PrintSummary() {
	fmt.Printf("\nSync Summary:\n")
	fmt.Printf("  Processed: %d\n", s.stats.Processed)
	fmt.Printf("  Created:   %d\n", s.stats.Created)
	fmt.Printf("  Updated:   %d\n", s.stats.Updated)
	fmt.Printf("  Skipped:   %d\n", s.stats.Skipped)
	fmt.Printf("  Failed:    %d\n", s.stats.Failed)
}

// GenerateReport generates a JSON report of the sync.
func (s *BeadsSink) GenerateReport() ([]byte, error) {
	report := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.stats,
	}
	return json.MarshalIndent(report, "", "  ")
}

// IssueTemplateData contains data for issue templates.
type IssueTemplateData struct {
	Category  string
	Severity  string
	File      string
	Line      int
	Message   string
	Hint      string
	FeatureID string
	WSID      string
	CheckName string
	RunID     int64
}

// DefaultIssueTemplate is the default template for issue descriptions.
const DefaultIssueTemplate = `**Category:** {{.Category}}
**Severity:** {{.Severity}}
**File:** {{.File}}{{if .Line}}:{{.Line}}{{end}}

**Message:** {{.Message}}

{{if .Hint}}**Remediation:** {{.Hint}}{{end}}

---
*Source: {{.CheckName}} (run {{.RunID}})*
`

// RenderIssueTemplate renders an issue description from a template.
func RenderIssueTemplate(tmpl string, data *IssueTemplateData) (string, error) {
	t, err := template.New("issue").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
