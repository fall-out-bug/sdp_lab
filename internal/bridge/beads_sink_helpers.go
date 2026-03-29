package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"
)

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
		return "", fmt.Errorf("parse issue template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute issue template: %w", err)
	}

	return buf.String(), nil
}
