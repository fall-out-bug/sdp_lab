package orchestrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// IndexRow is one row in the workstream status table.
type IndexRow struct {
	WS     string // e.g. 00-053-01
	Feature string // e.g. F053
	Title  string
	Status string // Done, Pending, Backlog, etc.
}

var (
	reIndexWSID = regexp.MustCompile(`(?m)^ws_id:\s*(\S+)`)
	reStatus    = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
	reTitle     = regexp.MustCompile(`(?m)^#\s+(\d{2}-\d{3}-\d{2}):\s*(.+)$`)
)

// GenerateIndexTable produces INDEX.md table rows for a feature from workstream files.
// Checkpoint is optional; if present, checkpoint status overrides file status for done/in_progress.
func GenerateIndexTable(projectRoot, featureID string, cp *Checkpoint) ([]IndexRow, error) {
	fnum := strings.TrimPrefix(strings.ToUpper(featureID), "F")
	if fnum == "" {
		return nil, fmt.Errorf("invalid feature_id %q", featureID)
	}
	prefix := "00-" + fnum + "-"
	dir := filepath.Join(projectRoot, "docs", "workstreams", "backlog")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read workstreams dir: %w", err)
	}

	cpStatus := make(map[string]string)
	if cp != nil {
		for _, ws := range cp.Workstreams {
			cpStatus[ws.ID] = ws.Status
		}
	}

	var rows []IndexRow
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		row, err := parseIndexRow(path, featureID, cpStatus)
		if err != nil {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WS < rows[j].WS })
	return rows, nil
}

func parseIndexRow(path, featureID string, cpStatus map[string]string) (IndexRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return IndexRow{}, err
	}
	content := string(data)
	base := strings.TrimSuffix(filepath.Base(path), ".md")
	wsID := base
	status := "Backlog"
	title := base

	if m := reIndexWSID.FindStringSubmatch(content); len(m) > 1 {
		wsID = strings.Trim(m[1], `"`)
	}
	if m := reStatus.FindStringSubmatch(content); len(m) > 1 {
		status = strings.Trim(m[1], `"`)
	}
	if m := reTitle.FindStringSubmatch(content); len(m) > 2 {
		title = strings.TrimSpace(m[2])
	}

	// Checkpoint overrides: done, in_progress, pending
	if override, ok := cpStatus[wsID]; ok {
		switch override {
		case "done":
			status = "Done"
		case "in_progress":
			status = "In Progress"
		case "pending":
			status = "Pending"
		}
	} else {
		// Normalize file status for display
		switch strings.ToLower(status) {
		case "done":
			status = "Done"
		case "pending":
			status = "Pending"
		case "backlog":
			status = "Backlog"
		case "in_progress":
			status = "In Progress"
		default:
			status = capitalize(status)
		}
	}

	return IndexRow{WS: wsID, Feature: featureID, Title: title, Status: status}, nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// FormatIndexTable returns markdown table lines for the rows.
func FormatIndexTable(rows []IndexRow) string {
	if len(rows) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("| WS | Feature | Title | Status |\n")
	sb.WriteString("|----|---------|-------|--------|\n")
	for _, r := range rows {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", r.WS, r.Feature, r.Title, r.Status))
	}
	return sb.String()
}

// UpdateIndexFile patches docs/workstreams/INDEX.md, replacing the feature's table section.
// Section is identified by "### Phase 4 Remediation" or "### F054" style headers.
// If section not found, appends before "## Workstream ID Format".
func UpdateIndexFile(projectRoot, featureID string, cp *Checkpoint) error {
	indexPath := filepath.Join(projectRoot, "docs", "workstreams", "INDEX.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read INDEX: %w", err)
	}

	rows, err := GenerateIndexTable(projectRoot, featureID, cp)
	if err != nil {
		return err
	}
	newTable := FormatIndexTable(rows)
	if newTable == "" {
		return nil
	}

	// Find F053 section: "### Phase 4 Remediation" followed by table until next ### or ---
	content := string(data)
	sectionHeader := "### Phase 4 Remediation"
	start := strings.Index(content, sectionHeader)
	if start < 0 {
		// Try alternate header
		sectionHeader = "### F053"
		start = strings.Index(content, sectionHeader)
	}
	if start < 0 {
		return fmt.Errorf("INDEX.md: section for %s not found", featureID)
	}

	// Find end of table (next ### or ---)
	afterHeader := content[start:]
	tableStart := strings.Index(afterHeader, "| WS |")
	if tableStart < 0 {
		tableStart = strings.Index(afterHeader, "\n")
	}
	from := start + tableStart
	rest := content[from:]
	lineEnd := strings.Index(rest, "\n### ")
	if lineEnd < 0 {
		lineEnd = strings.Index(rest, "\n---")
	}
	if lineEnd < 0 {
		lineEnd = len(rest)
	}
	end := from + lineEnd

	newContent := content[:from] + "\n" + newTable + content[end:]
	return os.WriteFile(indexPath, []byte(newContent), 0o644)
}
