package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// BeadsSink creates and updates Beads tasks from findings.
type BeadsSink struct {
	mu       sync.RWMutex
	prefix   string // Issue prefix (e.g., "sdplab-")
	dryRun   bool
	labels   []string
	stats    SyncStats
	findings map[string]bool // Track existing findings by key
}

// NewBeadsSink creates a new Beads sink.
func NewBeadsSink(prefix string, dryRun bool, defaultLabels []string) *BeadsSink {
	return &BeadsSink{
		prefix:   prefix,
		dryRun:   dryRun,
		labels:   defaultLabels,
		findings: make(map[string]bool),
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
	// Query existing issues with finding_key label
	cmd := exec.CommandContext(ctx, "bd", "list", "-l", "ci-finding", "--json")
	output, err := cmd.Output()
	if err != nil {
		// May not have any existing findings, that's OK
		return nil
	}

	var issues []map[string]interface{}
	if err := json.Unmarshal(output, &issues); err != nil {
		// Log parse error but don't fail - existing findings are optional
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, issue := range issues {
		if id, ok := issue["id"].(string); ok {
			s.findings[id] = true
		}
	}

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

	// Generate unique key for deduplication
	key := fmt.Sprintf("finding-%s", f.FindingKey)
	s.mu.RLock()
	exists := s.findings[key]
	s.mu.RUnlock()
	if exists {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	// Build title
	title := fmt.Sprintf("[%s] %s: %s", f.Category, f.Code, truncate(f.Message, 60))

	// Build description
	desc := s.buildProtocolDescription(f, source)

	// Build labels
	labels := s.buildLabels(f.Severity, f.Category, f.Context.FeatureID, f.Context.WSID)

	// Determine priority
	priority := severityToPriority(f.Severity)

	// Create the issue
	if s.dryRun {
		fmt.Printf("[DRY-RUN] Would create: %s\n", title)
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
		return nil
	}

	if err := s.createBeadsIssue(ctx, title, desc, priority, labels); err != nil {
		return err
	}

	s.mu.Lock()
	s.stats.Created++
	s.findings[key] = true
	s.mu.Unlock()
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

	// Generate unique key for deduplication
	key := fmt.Sprintf("finding-%s", f.FindingKey)
	s.mu.RLock()
	exists := s.findings[key]
	s.mu.RUnlock()
	if exists {
		s.mu.Lock()
		s.stats.Skipped++
		s.mu.Unlock()
		return nil
	}

	// Build title
	title := fmt.Sprintf("[docs:%s] %s", f.Category, truncate(f.Message, 60))

	// Build description
	desc := s.buildDocsDescription(f, source)

	// Build labels
	labels := s.buildLabels(f.Severity, f.Category, "", "")

	// Determine priority
	priority := severityToPriority(f.Severity)

	// Create the issue
	if s.dryRun {
		fmt.Printf("[DRY-RUN] Would create: %s\n", title)
		s.mu.Lock()
		s.stats.Created++
		s.mu.Unlock()
		return nil
	}

	if err := s.createBeadsIssue(ctx, title, desc, priority, labels); err != nil {
		return err
	}

	s.mu.Lock()
	s.stats.Created++
	s.findings[key] = true
	s.mu.Unlock()
	return nil
}

func (s *BeadsSink) buildProtocolDescription(f *ProtocolFinding, source *FindingsSource) string {
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

	buf.WriteString(fmt.Sprintf("---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID))

	return buf.String()
}

func (s *BeadsSink) buildDocsDescription(f *DocsFinding, source *FindingsSource) string {
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

	buf.WriteString(fmt.Sprintf("---\n*Source: %s (run %d)*\n", source.CheckName, source.RunID))

	return buf.String()
}

func (s *BeadsSink) buildLabels(severity, category, featureID, wsID string) []string {
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

	// Add default labels
	labels = append(labels, s.labels...)

	return labels
}

func (s *BeadsSink) createBeadsIssue(ctx context.Context, title, description string, priority int, labels []string) error {
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
