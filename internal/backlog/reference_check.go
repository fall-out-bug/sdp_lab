// Package backlog provides reference integrity checking for SDP workstreams.
// F100-01: Reference Integrity CI Gate
package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ReferenceType defines the type of reference.
type ReferenceType string

const (
	// RefTypeWorkstream references another workstream document
	RefTypeWorkstream ReferenceType = "workstream"
	// RefTypeFeature references a feature ID (e.g., F042)
	RefTypeFeature ReferenceType = "feature"
	// RefTypeBead references a bead ID (e.g., sdplab-xxx)
	RefTypeBead ReferenceType = "bead"
	// RefTypeFile references a file in the repository
	RefTypeFile ReferenceType = "file"
	// RefTypeExternal references an external resource (URL, issue tracker, etc.)
	RefTypeExternal ReferenceType = "external"
)

// Reference represents a cross-reference found in workstream documentation.
type Reference struct {
	// Type is the kind of reference
	Type ReferenceType

	// Source is the file where the reference was found
	Source string

	// LineNumber is the line number where the reference appears (1-indexed)
	LineNumber int

	// Raw is the raw reference text as found in the document
	Raw string

	// Target is the resolved target (e.g., file path, feature ID)
	Target string

	// Context provides surrounding context for the reference
	Context string
}

// ReferenceIssue represents a problem found with a reference.
type ReferenceIssue struct {
	// Reference is the problematic reference
	Reference Reference

	// Severity is the issue severity (error, warning, info)
	Severity string

	// Message describes the issue
	Message string

	// Suggestion provides a suggested fix (if applicable)
	Suggestion string
}

// CheckResult contains the results of a reference integrity check.
type CheckResult struct {
	// TotalReferences is the total number of references found
	TotalReferences int

	// ValidReferences is the number of valid references
	ValidReferences int

	// Issues contains all found issues
	Issues []ReferenceIssue

	// CheckedFiles is the number of files checked
	CheckedFiles int

	// SkippedFiles is the number of files skipped (e.g., not found, parse errors)
	SkippedFiles int
}

// CheckOptions configures reference integrity checking.
type CheckOptions struct {
	// RepoRoot is the repository root directory
	RepoRoot string

	// StrictMode enables strict checking (warnings become errors)
	StrictMode bool

	// CheckExternal verifies external references (slow, requires network)
	CheckExternal bool

	// AllowedSchemes lists allowed URL schemes for external references
	AllowedSchemes []string

	// WorkstreamDir is the directory containing workstream documents
	WorkstreamDir string

	// ExcludePatterns contains file patterns to exclude from checking
	ExcludePatterns []string
}

// DefaultCheckOptions returns default check options.
func DefaultCheckOptions(repoRoot string) CheckOptions {
	return CheckOptions{
		RepoRoot:       repoRoot,
		StrictMode:     false,
		CheckExternal:  false,
		AllowedSchemes: []string{"http", "https", "mailto"},
		WorkstreamDir:  filepath.Join(repoRoot, "docs", "workstreams", "backlog"),
		ExcludePatterns: []string{
			"node_modules",
			".git",
			"vendor",
			".sdp",
		},
	}
}

// CheckReferenceIntegrity performs a comprehensive reference integrity check.
func CheckReferenceIntegrity(opts CheckOptions) (*CheckResult, error) {
	result := &CheckResult{
		Issues: []ReferenceIssue{},
	}

	// Find all workstream documents
	wsFiles, err := findWorkstreamFiles(opts.WorkstreamDir)
	if err != nil {
		return nil, fmt.Errorf("find workstream files: %w", err)
	}

	// Check each workstream file
	for _, wsFile := range wsFiles {
		if shouldExclude(wsFile, opts.ExcludePatterns) {
			result.SkippedFiles++
			continue
		}

		fileResult, err := checkFileReferences(wsFile, opts)
		if err != nil {
			result.SkippedFiles++
			continue
		}

		result.TotalReferences += fileResult.TotalReferences
		result.ValidReferences += fileResult.ValidReferences
		result.Issues = append(result.Issues, fileResult.Issues...)
		result.CheckedFiles++
	}

	return result, nil
}

// findWorkstreamFiles finds all workstream markdown files.
func findWorkstreamFiles(wsDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(wsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// shouldExclude checks if a file should be excluded based on patterns.
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// checkFileReferences checks all references in a single file.
func checkFileReferences(filePath string, opts CheckOptions) (*CheckResult, error) {
	result := &CheckResult{
		Issues: []ReferenceIssue{},
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	relPath, _ := filepath.Rel(opts.RepoRoot, filePath)

	// Find all references in the file
	for lineNum, line := range lines {
		refs := extractReferences(line, relPath, lineNum+1)

		for _, ref := range refs {
			result.TotalReferences++

			// Validate the reference
			issue := validateReference(ref, opts)
			if issue != nil {
				result.Issues = append(result.Issues, *issue)
			} else {
				result.ValidReferences++
			}
		}
	}

	return result, nil
}

// Reference patterns
var (
	// Workstream reference: [WS Name](../../workstreams/backlog/00-XXX-XX.md)
	wsRefRe = regexp.MustCompile(`\[([^\]]+)\]\((\.\.\/.*\/workstreams\/backlog\/[0-9]{2}-[0-9]{3}(?:-[0-9]{2})?\.md)\)`)

	// Feature reference: F042, F101-02
	featureRefRe = regexp.MustCompile(`\bF([0-9]{3})(?:-([0-9]{2}))?\b`)

	// Bead reference: sdplab-xxx
	beadRefRe = regexp.MustCompile(`\bsdplab-[a-z0-9]+\b`)

	// File reference: [text](path/to/file.ext)
	fileRefRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.[a-z]{2,4})\)`)

	// URL reference: https?://...
	urlRefRe = regexp.MustCompile(`https?://[^\s\)]+`)
)

// extractReferences finds all references in a line of text.
func extractReferences(line, source string, lineNum int) []Reference {
	var refs []Reference

	// Check for workstream references
	if matches := wsRefRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 3 {
				refs = append(refs, Reference{
					Type:       RefTypeWorkstream,
					Source:     source,
					LineNumber: lineNum,
					Raw:        match[0],
					Target:     match[2],
					Context:    getContext(line, match[0]),
				})
			}
		}
	}

	// Check for feature references
	if matches := featureRefRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 2 {
				target := "F" + match[1]
				if match[2] != "" {
					target += "-" + match[2]
				}
				refs = append(refs, Reference{
					Type:       RefTypeFeature,
					Source:     source,
					LineNumber: lineNum,
					Raw:        match[0],
					Target:     target,
					Context:    getContext(line, match[0]),
				})
			}
		}
	}

	// Check for bead references
	if matches := beadRefRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		for _, match := range matches {
			refs = append(refs, Reference{
				Type:       RefTypeBead,
				Source:     source,
				LineNumber: lineNum,
				Raw:        match[0],
				Target:     match[0],
				Context:    getContext(line, match[0]),
			})
		}
	}

	// Check for file references (excluding workstream refs already captured)
	if matches := fileRefRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 3 && !strings.Contains(match[2], "workstreams/backlog") {
				refs = append(refs, Reference{
					Type:       RefTypeFile,
					Source:     source,
					LineNumber: lineNum,
					Raw:        match[0],
					Target:     match[2],
					Context:    getContext(line, match[0]),
				})
			}
		}
	}

	// Check for URL references
	if matches := urlRefRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		for _, match := range matches {
			refs = append(refs, Reference{
				Type:       RefTypeExternal,
				Source:     source,
				LineNumber: lineNum,
				Raw:        match[0],
				Target:     match[0],
				Context:    getContext(line, match[0]),
			})
		}
	}

	return refs
}

// getContext returns surrounding context for a reference.
func getContext(line, ref string) string {
	idx := strings.Index(line, ref)
	if idx == -1 {
		return line
	}

	start := idx - 20
	if start < 0 {
		start = 0
	}

	end := idx + len(ref) + 20
	if end > len(line) {
		end = len(line)
	}

	context := line[start:end]
	if start > 0 {
		context = "..." + context
	}
	if end < len(line) {
		context = context + "..."
	}

	return context
}

// validateReference checks if a reference is valid.
func validateReference(ref Reference, opts CheckOptions) *ReferenceIssue {
	switch ref.Type {
	case RefTypeWorkstream:
		return validateWorkstreamReference(ref, opts)
	case RefTypeFeature:
		return validateFeatureReference(ref, opts)
	case RefTypeFile:
		return validateFileReference(ref, opts)
	case RefTypeExternal:
		return validateExternalReference(ref, opts)
	case RefTypeBead:
		// Bead references are harder to validate without access to beads
		// For now, just check format
		if strings.HasPrefix(ref.Target, "sdplab-") && len(ref.Target) > 8 {
			return nil
		}
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "warning",
			Message:   "Invalid bead ID format",
		}
	}

	return nil
}

// validateWorkstreamReference checks if a workstream file exists.
func validateWorkstreamReference(ref Reference, opts CheckOptions) *ReferenceIssue {
	// Resolve the path relative to the source file
	sourceDir := filepath.Dir(ref.Source)
	targetPath := filepath.Join(opts.RepoRoot, sourceDir, ref.Target)

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "error",
			Message:   fmt.Sprintf("Workstream file does not exist: %s", ref.Target),
			Suggestion: fmt.Sprintf("Create the file at %s or remove the reference", targetPath),
		}
	}

	return nil
}

// validateFeatureReference checks if a feature reference is valid.
func validateFeatureReference(ref Reference, opts CheckOptions) *ReferenceIssue {
	// Check if there's a workstream file for this feature
	wsDir := filepath.Join(opts.RepoRoot, "docs", "workstreams", "backlog")

	// Parse feature ID
	fidRe := regexp.MustCompile(`^F([0-9]{3})(?:-([0-9]{2}))?$`)
	match := fidRe.FindStringSubmatch(ref.Target)
	if match == nil {
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "error",
			Message:   fmt.Sprintf("Invalid feature ID format: %s", ref.Target),
		}
	}

	epicNum := match[1]
	subNum := match[2]

	var wsPattern string
	if subNum == "" {
		// Epic-level reference
		wsPattern = filepath.Join(wsDir, fmt.Sprintf("00-%s-*.md", epicNum))
	} else {
		// Sub-feature reference
		wsPattern = filepath.Join(wsDir, fmt.Sprintf("00-%s-%s.md", epicNum, subNum))
	}

	matches, _ := filepath.Glob(wsPattern)
	if len(matches) == 0 {
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "warning",
			Message:   fmt.Sprintf("No workstream file found for feature %s", ref.Target),
			Suggestion: fmt.Sprintf("Create a workstream file matching pattern: %s", wsPattern),
		}
	}

	return nil
}

// validateFileReference checks if a file reference exists.
func validateFileReference(ref Reference, opts CheckOptions) *ReferenceIssue {
	// Resolve the path relative to the source file
	sourceDir := filepath.Dir(ref.Source)
	targetPath := filepath.Join(opts.RepoRoot, sourceDir, ref.Target)

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "error",
			Message:   fmt.Sprintf("Referenced file does not exist: %s", ref.Target),
			Suggestion: fmt.Sprintf("Create the file at %s or update the reference", targetPath),
		}
	}

	return nil
}

// validateExternalReference checks if an external URL is valid.
func validateExternalReference(ref Reference, opts CheckOptions) *ReferenceIssue {
	// Check URL scheme
	hasScheme := false
	for _, scheme := range opts.AllowedSchemes {
		if strings.HasPrefix(ref.Target, scheme+"://") {
			hasScheme = true
			break
		}
	}

	if !hasScheme {
		return &ReferenceIssue{
			Reference: ref,
			Severity:  "warning",
			Message:   fmt.Sprintf("URL scheme not in allowed list: %s", ref.Target),
			Suggestion: "Use one of: " + strings.Join(opts.AllowedSchemes, ", "),
		}
	}

	// In strict mode with external checking enabled, we could verify the URL
	// For now, just check format
	return nil
}

// FormatCheckReport formats the check result as a human-readable report.
func FormatCheckReport(result *CheckResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Reference Integrity Check\n"))
	sb.WriteString(fmt.Sprintf("========================\n\n"))
	sb.WriteString(fmt.Sprintf("Files checked: %d\n", result.CheckedFiles))
	sb.WriteString(fmt.Sprintf("Files skipped: %d\n", result.SkippedFiles))
	sb.WriteString(fmt.Sprintf("Total references: %d\n", result.TotalReferences))
	sb.WriteString(fmt.Sprintf("Valid references: %d\n", result.ValidReferences))
	sb.WriteString(fmt.Sprintf("Issues found: %d\n\n", len(result.Issues)))

	if len(result.Issues) == 0 {
		sb.WriteString("✓ All references are valid!\n")
		return sb.String()
	}

	// Group issues by severity
	errors := []ReferenceIssue{}
	warnings := []ReferenceIssue{}

	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			errors = append(errors, issue)
		} else {
			warnings = append(warnings, issue)
		}
	}

	// Print errors
	if len(errors) > 0 {
		sb.WriteString(fmt.Sprintf("Errors (%d):\n", len(errors)))
		for _, err := range errors {
			sb.WriteString(fmt.Sprintf("  ✗ %s:%d: %s\n", err.Reference.Source, err.Reference.LineNumber, err.Message))
			if err.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", err.Suggestion))
			}
		}
		sb.WriteString("\n")
	}

	// Print warnings
	if len(warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings (%d):\n", len(warnings)))
		for _, warn := range warnings {
			sb.WriteString(fmt.Sprintf("  ⚠ %s:%d: %s\n", warn.Reference.Source, warn.Reference.LineNumber, warn.Message))
			if warn.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", warn.Suggestion))
			}
		}
	}

	return sb.String()
}

// ExitStatusForCheck returns the appropriate exit code for a check result.
func ExitStatusForCheck(result *CheckResult, strictMode bool) int {
	if len(result.Issues) == 0 {
		return 0
	}

	// In strict mode, any issue fails
	if strictMode {
		return 1
	}

	// Otherwise, only errors fail
	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			return 1
		}
	}

	return 0
}
