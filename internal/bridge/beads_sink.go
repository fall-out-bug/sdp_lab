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

type FindingSourceType string

const (
	FindingSourceReview FindingSourceType = "review"
	FindingSourceCI     FindingSourceType = "ci"
	FindingSourceDrift  FindingSourceType = "drift"
	FindingSourceQA     FindingSourceType = "qa"
)

type TypedFinding struct {
	Source       FindingSourceType
	FeatureID    string
	WSID         string
	Blocking     bool
	Title        string
	Summary      string
	Description  string
	Severity     string
	Priority     int
	PRURL        string
	ArtifactRef  string
	EvidenceRef  string
	TraceRef     string
	DriftVerdict string
	DedupKey     string
}

type ReviewFindingInput struct {
	FeatureID    string
	WSID         string
	Blocking     bool
	Role         string
	Title        string
	Summary      string
	Description  string
	Severity     string
	Priority     int
	PRURL        string
	ArtifactRef  string
	EvidenceRef  string
	TraceRef     string
	DriftVerdict string
	DedupKey     string
}

type QAFindingInput struct {
	FeatureID       string
	WSID            string
	Blocking        bool
	Scenario        string
	FailedStep      string
	Title           string
	Summary         string
	Description     string
	Severity        string
	Priority        int
	PRURL           string
	ArtifactRef     string
	EvidenceRef     string
	TraceRef        string
	ExpectedOutcome string
	ActualOutcome   string
	DedupKey        string
}

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
	cmd := exec.CommandContext(ctx, "bd", "list", "--all", "--json", "-n", "0")
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

	title := fmt.Sprintf("[%s] %s: %s", f.Category, f.Code, truncate(f.Message, 60))
	desc := s.buildProtocolDescription(f, source, findingHash, payloadHash)
	labels := s.buildLabels(FindingSourceCI, f.Severity, f.Category, f.Context.FeatureID, f.Context.WSID, true, findingHash, payloadHash)
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if _, err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
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

	title := fmt.Sprintf("[docs:%s] %s", f.Category, truncate(f.Message, 60))
	desc := s.buildDocsDescription(f, source, findingHash, payloadHash)
	labels := s.buildLabels(FindingSourceCI, f.Severity, f.Category, "", "", true, findingHash, payloadHash)
	priority := severityToPriority(f.Severity)

	if s.dryRun {
		s.handleDryRunDecision(decision, findingHash, payloadHash, title)
		return nil
	}

	if _, err := s.applyDecision(ctx, decision, title, desc, priority, labels); err != nil {
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
	prefix := strings.Join(parts, " ")
	if prefix == "" {
		prefix = strings.ToUpper(string(finding.Source))
	}
	if finding.Summary != "" {
		return fmt.Sprintf("%s: %s", prefix, truncate(finding.Summary, 72))
	}
	return fmt.Sprintf("%s: finding", prefix)
}

func buildTypedFindingDescription(finding TypedFinding, findingHash, payloadHash string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("**Source:** %s\n", finding.Source))
	if finding.FeatureID != "" {
		buf.WriteString(fmt.Sprintf("**Feature:** %s\n", finding.FeatureID))
	}
	if finding.WSID != "" {
		buf.WriteString(fmt.Sprintf("**Workstream:** %s\n", finding.WSID))
	}
	buf.WriteString(fmt.Sprintf("**Blocking:** %t\n", finding.Blocking))
	if finding.Severity != "" {
		buf.WriteString(fmt.Sprintf("**Severity:** %s\n", finding.Severity))
	}
	if finding.Priority > 0 {
		buf.WriteString(fmt.Sprintf("**Priority:** P%d\n", finding.Priority))
	}
	buf.WriteString("\n")
	if finding.Summary != "" {
		buf.WriteString(fmt.Sprintf("**Summary:** %s\n\n", finding.Summary))
	}
	if finding.Description != "" {
		buf.WriteString(fmt.Sprintf("**Description:** %s\n\n", finding.Description))
	}
	if finding.EvidenceRef != "" || finding.ArtifactRef != "" || finding.PRURL != "" || finding.TraceRef != "" || finding.DriftVerdict != "" {
		buf.WriteString("**References:**\n")
		if finding.EvidenceRef != "" {
			buf.WriteString(fmt.Sprintf("- Evidence: %s\n", finding.EvidenceRef))
		}
		if finding.ArtifactRef != "" {
			buf.WriteString(fmt.Sprintf("- Artifact: %s\n", finding.ArtifactRef))
		}
		if finding.PRURL != "" {
			buf.WriteString(fmt.Sprintf("- PR: %s\n", finding.PRURL))
		}
		if finding.TraceRef != "" {
			buf.WriteString(fmt.Sprintf("- Trace: %s\n", finding.TraceRef))
		}
		if finding.DriftVerdict != "" {
			buf.WriteString(fmt.Sprintf("- Drift: %s\n", finding.DriftVerdict))
		}
		buf.WriteString("\n")
	}
	buf.WriteString(fmt.Sprintf("**Finding Hash:** `%s`\n", findingHash))
	buf.WriteString(fmt.Sprintf("**Payload Hash:** `%s`\n", payloadHash))

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
		buf.WriteString(fmt.Sprintf("Reviewer role: %s\n", input.Role))
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

func (s *BeadsSink) buildLabels(source FindingSourceType, severity, category, featureID, wsID string, blocking bool, findingHash, payloadHash string) []string {
	labels := []string{"sdp-finding", sourceFindingLabel(source)}
	if severity != "" {
		labels = append(labels, normalizeValue(severity))
	}
	if blocking {
		labels = append(labels, "blocking")
	} else {
		labels = append(labels, "non-blocking")
	}

	if category != "" {
		labels = append(labels, normalizeValue(category))
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

	labels = append(labels, s.labels...)

	return uniqueLabels(labels)
}

func sourceFindingLabel(source FindingSourceType) string {
	switch source {
	case FindingSourceReview:
		return "review-finding"
	case FindingSourceDrift:
		return "drift-finding"
	case FindingSourceQA:
		return "qa-finding"
	case FindingSourceCI:
		fallthrough
	default:
		return "ci-finding"
	}
}

func uniqueLabels(labels []string) []string {
	seen := make(map[string]struct{}, len(labels))
	result := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	return result
}

func (s *BeadsSink) createBeadsIssue(ctx context.Context, title, description string, priority int, labels []string) (string, error) {
	args := []string{
		"create",
		"--silent",
		"-p", fmt.Sprintf("%d", priority),
		"-t", "task",
		"-l", strings.Join(labels, ","),
		"-d", description,
	}
	if strings.TrimSpace(s.prefix) != "" {
		args = append(args, "--prefix", s.prefix)
	}
	args = append(args, title)

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

func (s *BeadsSink) applyDecision(ctx context.Context, decision DedupeDecision, title, description string, priority int, labels []string) (string, error) {
	switch decision.Action {
	case DedupeCreate:
		issueID, err := s.createBeadsIssue(ctx, title, description, priority, labels)
		if err != nil {
			return "", err
		}
		s.dedupe.RecordCreated(decision.FindingHash, decision.PayloadHash, issueID)
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
		return issueID, nil
	case DedupeUpdate:
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return "", err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return decision.IssueID, nil
	case DedupeReopenUpdate:
		if err := s.reopenBeadsIssue(ctx, decision.IssueID, "finding payload changed"); err != nil {
			return "", err
		}
		if err := s.updateBeadsIssue(ctx, decision.IssueID, title, description, priority, labels); err != nil {
			return "", err
		}
		s.dedupe.RecordUpdated(decision.FindingHash, decision.PayloadHash)
		s.mu.Lock()
		s.stats.Updated++
		s.mu.Unlock()
		return decision.IssueID, nil
	default:
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return decision.IssueID, nil
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
