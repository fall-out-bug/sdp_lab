package workstream

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sdp_dev/internal/sdputil"
)

type ValidationIssue struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type ValidationReport struct {
	Issues []ValidationIssue `json:"issues"`
}

func (r ValidationReport) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

func ValidateProtocol(projectRoot string, strictBeads, strictAll bool) (ValidationReport, error) {
	report := ValidationReport{Issues: []ValidationIssue{}}

	indexPath := filepath.Join(projectRoot, "docs", "workstreams", "INDEX.md")
	roadmapPath := filepath.Join(projectRoot, "docs", "roadmap", "ROADMAP.md")
	backlogDir := filepath.Join(projectRoot, "docs", "workstreams", "backlog")

	indexContent, roadmapContent, entries, err := loadProtocolFiles(indexPath, roadmapPath, backlogDir)
	if err != nil {
		return report, err
	}

	indexFeatures := extractFeatures(indexContent)
	roadmapFeatures := extractFeatures(roadmapContent)
	indexWSIDs := extractWSIDs(indexContent)

	wsFeatures := map[string]bool{}
	wsFiles := map[string]bool{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(backlogDir, entry.Name())
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			report.Issues = append(report.Issues, ValidationIssue{Severity: "error", File: rel(projectRoot, path), Message: fmt.Sprintf("read file: %v", err)})
			continue
		}
		content := string(contentBytes)

		meta, metaIssues := validateWorkstreamFile(projectRoot, entry.Name(), content, strictBeads, strictAll)
		report.Issues = append(report.Issues, metaIssues...)

		if meta.WSID != "" {
			wsFiles[meta.WSID] = true
		}
		if meta.FeatureID != "" {
			wsFeatures[meta.FeatureID] = true
		}
	}

	report.Issues = append(report.Issues, validateFeatureReferences(projectRoot, strictAll, wsFeatures, indexFeatures, roadmapFeatures, indexPath, roadmapPath)...)
	report.Issues = append(report.Issues, validateIndexWorkstreamReferences(projectRoot, indexWSIDs, wsFiles, indexPath)...)

	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Severity == report.Issues[j].Severity {
			if report.Issues[i].File == report.Issues[j].File {
				return report.Issues[i].Message < report.Issues[j].Message
			}
			return report.Issues[i].File < report.Issues[j].File
		}
		return report.Issues[i].Severity < report.Issues[j].Severity
	})

	return report, nil
}

func loadProtocolFiles(indexPath, roadmapPath, backlogDir string) (string, string, []os.DirEntry, error) {
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("read INDEX.md: %w", err)
	}

	roadmapContent, err := os.ReadFile(roadmapPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("read ROADMAP.md: %w", err)
	}

	entries, err := os.ReadDir(backlogDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("read backlog directory: %w", err)
	}

	return string(indexContent), string(roadmapContent), entries, nil
}

func validateFeatureReferences(projectRoot string, strictAll bool, wsFeatures, indexFeatures, roadmapFeatures map[string]bool, indexPath, roadmapPath string) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	for feature := range wsFeatures {
		severity := severityForStrict(strictAll, isLegacyFeatureID(feature))
		if !indexFeatures[feature] {
			issues = append(issues, ValidationIssue{Severity: severity, File: rel(projectRoot, indexPath), Message: fmt.Sprintf("feature %s referenced by backlog WS but missing in INDEX.md", feature)})
		}
		if !roadmapFeatures[feature] {
			issues = append(issues, ValidationIssue{Severity: severity, File: rel(projectRoot, roadmapPath), Message: fmt.Sprintf("feature %s referenced by backlog WS but missing in ROADMAP.md", feature)})
		}
	}
	return issues
}

func validateIndexWorkstreamReferences(projectRoot string, indexWSIDs, wsFiles map[string]bool, indexPath string) []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	for wsid := range indexWSIDs {
		if !wsFiles[wsid] {
			issues = append(issues, ValidationIssue{Severity: "warning", File: rel(projectRoot, indexPath), Message: fmt.Sprintf("INDEX references %s but backlog file not found", wsid)})
		}
	}
	return issues
}

type wsMeta struct {
	WSID      string
	FeatureID string
}

func validateWorkstreamFile(projectRoot, filename, content string, strictBeads, strictAll bool) (wsMeta, []ValidationIssue) {
	issues := []ValidationIssue{}
	file := rel(projectRoot, filepath.Join(projectRoot, "docs", "workstreams", "backlog", filename))
	meta := wsMeta{}
	legacyWS := isLegacyWorkstreamFilename(filename)

	fm := parseFrontmatter(content)
	frontmatterIssues, earlyExit := validateFrontmatterAndMeta(file, filename, fm, strictAll, legacyWS, &meta)
	issues = append(issues, frontmatterIssues...)
	if earlyExit {
		return meta, issues
	}

	issues = append(issues, validateBeadsSection(file, content, strictBeads, strictAll, legacyWS)...)
	issues = append(issues, validateAcceptanceCriteriaSection(file, content, strictAll, legacyWS)...)

	return meta, issues
}

func validateFrontmatterAndMeta(file, filename string, fm map[string]string, strictAll, legacyWS bool, meta *wsMeta) ([]ValidationIssue, bool) {
	issues := []ValidationIssue{}
	required := []string{"ws_id", "feature_id", "status", "priority", "size", "depends_on"}
	missing := 0
	for _, k := range required {
		if _, ok := fm[k]; !ok {
			missing++
		}
	}

	if missing == len(required) {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "legacy workstream format detected (no protocol frontmatter); strict checks skipped"})
		return issues, true
	}

	for _, k := range required {
		if _, ok := fm[k]; !ok {
			issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: fmt.Sprintf("missing frontmatter key %q", k)})
		}
	}

	if wsid, ok := fm["ws_id"]; ok {
		meta.WSID = wsid
		if err := sdputil.ValidateWSID(wsid); err != nil {
			issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: err.Error()})
		}
		base := strings.TrimSuffix(filename, ".md")
		if wsid != base {
			issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: fmt.Sprintf("ws_id %q does not match filename %q", wsid, base)})
		}
	}

	if fid, ok := fm["feature_id"]; ok {
		meta.FeatureID = fid
		if err := sdputil.ValidateFeatureID(fid); err != nil {
			issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: err.Error()})
		}
	}

	return issues, false
}

var strictBeadsIDPattern = regexp.MustCompile(`sdplab-[a-z0-9]+`)

func validateBeadsSection(file, content string, strictBeads, strictAll, legacyWS bool) []ValidationIssue {
	issues := []ValidationIssue{}
	beadsSection, hasBeads := extractSection(content, "Beads")
	if !hasBeads {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "missing section '## Beads'"})
		return issues
	}

	beadsItems := checkboxOrBulletItems(beadsSection, false)
	if len(beadsItems) == 0 {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "section '## Beads' must contain at least one bullet item"})
		return issues
	}

	valid := false
	for _, item := range beadsItems {
		if !strings.Contains(item, "sdplab-") {
			continue
		}

		valid = true
		if strictBeads {
			lowerItem := strings.ToLower(item)
			if strings.Contains(lowerItem, "sdplab-xx") || strings.Contains(lowerItem, "sdplab-xxx") || strings.Contains(lowerItem, "sdplab-placeholder") {
				issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: "Beads entry must reference concrete issue id (sdplab-<id>) in strict mode - placeholder detected"})
			} else if !strictBeadsIDPattern.MatchString(lowerItem) {
				issues = append(issues, ValidationIssue{Severity: "error", File: file, Message: "Beads entry must reference concrete issue id (sdplab-<id>) in strict mode"})
			}
		} else if strings.Contains(item, "sdplab-XX") {
			issues = append(issues, ValidationIssue{Severity: "warning", File: file, Message: "Beads entry uses placeholder id sdplab-XX"})
		}
	}

	if !valid {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "Beads section must include sdplab-* reference"})
	}

	return issues
}

func validateAcceptanceCriteriaSection(file, content string, strictAll, legacyWS bool) []ValidationIssue {
	issues := []ValidationIssue{}
	acSection, hasAC := extractSection(content, "Acceptance Criteria")
	if !hasAC {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "missing section '## Acceptance Criteria'"})
		return issues
	}

	if ac := checkboxOrBulletItems(acSection, true); len(ac) == 0 {
		issues = append(issues, ValidationIssue{Severity: severityForStrict(strictAll, legacyWS), File: file, Message: "Acceptance Criteria section must contain at least one checkbox item"})
	}

	return issues
}

func severityForStrict(strictAll, legacy bool) string {
	if strictAll && !legacy {
		return "error"
	}
	return "warning"
}

func parseFrontmatter(content string) map[string]string {
	result := map[string]string{}
	if !strings.HasPrefix(content, "---\n") {
		return result
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return result
	}
	body := content[4 : 4+end]
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"`)
		result[k] = v
	}
	return result
}

func extractSection(content, title string) (string, bool) {
	re := regexp.MustCompile(`(?ms)^##\s+` + regexp.QuoteMeta(title) + `\s*\n(.*?)(?:\n##\s+|\z)`)
	m := re.FindStringSubmatch(content)
	if len(m) < 2 {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

func checkboxOrBulletItems(section string, checkboxOnly bool) []string {
	items := []string{}
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if checkboxOnly {
			if strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ") || strings.HasPrefix(line, "- [X] ") {
				items = append(items, strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "- [ ]"), "- [x]"), "- [X]")))
			}
			continue
		}
		if strings.HasPrefix(line, "- ") {
			items = append(items, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}
	return items
}

func extractFeatures(content string) map[string]bool {
	result := map[string]bool{}
	re := regexp.MustCompile(`\*\*(F[0-9]{3,4})\*\*`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			result[m[1]] = true
		}
	}
	return result
}

func extractWSIDs(content string) map[string]bool {
	result := map[string]bool{}
	re := regexp.MustCompile(`\b00-[0-9]{3}-[0-9]{2}\b`)
	for _, m := range re.FindAllString(content, -1) {
		result[m] = true
	}
	return result
}

func rel(projectRoot, path string) string {
	if rp, err := filepath.Rel(projectRoot, path); err == nil {
		return rp
	}
	return path
}

func isLegacyFeatureID(featureID string) bool {
	if !strings.HasPrefix(featureID, "F") {
		return false
	}
	var n int
	if _, err := fmt.Sscanf(featureID, "F%d", &n); err != nil {
		return false
	}
	return n < 60
}

func isLegacyWorkstreamFilename(filename string) bool {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	var prefix, feature, seq int
	if _, err := fmt.Sscanf(base, "%d-%d-%d", &prefix, &feature, &seq); err != nil {
		return false
	}
	_ = prefix
	_ = seq
	return feature < 60
}
