// Package backlog provides reference cleanup utilities for fixing broken references.
// F100-02: One-Time Reference Cleanup
package backlog

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CleanupPlan represents a plan for fixing broken references.
type CleanupPlan struct {
	// File is the path to the file to be modified
	File string

	// Issues contains the reference issues found in this file
	Issues []ReferenceIssue

	// Fixes contains the proposed fixes for each issue
	Fixes []ReferenceFix
}

// ReferenceFix represents a fix for a broken reference.
type ReferenceFix struct {
	// Issue is the issue being fixed
	Issue ReferenceIssue

	// OldContent is the original content
	OldContent string

	// NewContent is the replacement content
	NewContent string

	// AutoFix indicates if this fix can be applied automatically
	AutoFix bool

	// Reason explains why this fix was chosen
	Reason string
}

// CleanupResult represents the result of applying cleanup fixes.
type CleanupResult struct {
	// FilesScanned is the number of files scanned
	FilesScanned int

	// FilesModified is the number of files modified
	FilesModified int

	// IssuesFound is the total number of issues found
	IssuesFound int

	// IssuesFixed is the number of issues fixed
	IssuesFixed int

	// IssuesSkipped is the number of issues skipped (no fix available)
	IssuesSkipped int

	// ModifiedFiles lists the files that were modified
	ModifiedFiles []string
}

// CleanupOptions configures the cleanup process.
type CleanupOptions struct {
	// DryRun indicates if changes should be previewed without applying
	DryRun bool

	// AutoApply applies fixes that are safe to apply automatically
	AutoApply bool

	// Backup creates backup files before modifying
	Backup bool

	// Verbose prints detailed information
	Verbose bool
}

// GenerateCleanupPlans creates cleanup plans for fixing broken references.
func GenerateCleanupPlans(checkResult *CheckResult) []CleanupPlan {
	plans := make(map[string]*CleanupPlan)

	// Group issues by file
	for _, issue := range checkResult.Issues {
		file := issue.Reference.Source

		if plans[file] == nil {
			plans[file] = &CleanupPlan{
				File:   file,
				Issues: []ReferenceIssue{},
				Fixes:  []ReferenceFix{},
			}
		}

		plans[file].Issues = append(plans[file].Issues, issue)

		// Generate a fix for this issue
		fix := generateFixForIssue(issue)
		if fix != nil {
			plans[file].Fixes = append(plans[file].Fixes, *fix)
		}
	}

	// Convert map to slice
	result := make([]CleanupPlan, 0, len(plans))
	for _, plan := range plans {
		result = append(result, *plan)
	}

	return result
}

// generateFixForIssue generates a fix for a specific reference issue.
func generateFixForIssue(issue ReferenceIssue) *ReferenceFix {
	switch issue.Reference.Type {
	case RefTypeWorkstream:
		return generateWorkstreamFix(issue)
	case RefTypeFeature:
		return generateFeatureFix(issue)
	case RefTypeFile:
		return generateFileFix(issue)
	case RefTypeExternal:
		return generateExternalFix(issue)
	default:
		return nil
	}
}

// generateWorkstreamFix generates a fix for a broken workstream reference.
func generateWorkstreamFix(issue ReferenceIssue) *ReferenceFix {
	ref := issue.Reference

	// Try to find similar workstream files
	targetPath := ref.Target
	_ = filepathBase(targetPath)

	// Suggest removing the reference if it doesn't exist
	fix := &ReferenceFix{
		Issue:      issue,
		OldContent: ref.Raw,
		NewContent: fmt.Sprintf("[FIXME: broken workstream ref](%s)", targetPath),
		AutoFix:    false,
		Reason:     "Workstream file does not exist. Manual review required.",
	}

	return fix
}

// generateFeatureFix generates a fix for a broken feature reference.
func generateFeatureFix(issue ReferenceIssue) *ReferenceFix {
	ref := issue.Reference

	// Check if this is a typo or missing feature
	fidRe := regexp.MustCompile(`^F([0-9]{3})(?:-([0-9]{2}))?$`)
	match := fidRe.FindStringSubmatch(ref.Target)

	if match == nil {
		// Invalid format - can't auto-fix
		return &ReferenceFix{
			Issue:      issue,
			OldContent: ref.Raw,
			NewContent: ref.Raw, // Keep as-is
			AutoFix:    false,
			Reason:     "Invalid feature ID format - manual review required",
		}
	}

	// Feature ID is valid but workstream doesn't exist
	// Suggest creating the workstream or removing the reference
	fix := &ReferenceFix{
		Issue:      issue,
		OldContent: ref.Raw,
		NewContent: ref.Raw, // Keep the reference, it might be valid
		AutoFix:    false,
		Reason:     "Feature ID format is valid but no workstream file exists. Consider creating the workstream.",
	}

	return fix
}

// generateFileFix generates a fix for a broken file reference.
func generateFileFix(issue ReferenceIssue) *ReferenceFix {
	ref := issue.Reference

	// Try to suggest alternative paths
	fix := &ReferenceFix{
		Issue:      issue,
		OldContent: ref.Raw,
		NewContent: fmt.Sprintf("[FIXME: broken file ref](%s)", ref.Target),
		AutoFix:    false,
		Reason:     "Referenced file does not exist. Manual review required.",
	}

	return fix
}

// generateExternalFix generates a fix for a broken external URL.
func generateExternalFix(issue ReferenceIssue) *ReferenceFix {
	ref := issue.Reference

	fix := &ReferenceFix{
		Issue:      issue,
		OldContent: ref.Raw,
		NewContent: ref.Raw, // Keep URLs as-is
		AutoFix:    false,
		Reason:     "URL format issue - manual review required",
	}

	return fix
}

// ApplyCleanupPlans applies the cleanup plans to fix broken references.
func ApplyCleanupPlans(plans []CleanupPlan, opts CleanupOptions) (*CleanupResult, error) {
	result := &CleanupResult{
		ModifiedFiles: []string{},
	}

	for _, plan := range plans {
		if opts.Verbose {
			fmt.Printf("Processing: %s\n", plan.File)
		}

		result.FilesScanned++
		result.IssuesFound += len(plan.Issues)

		if len(plan.Fixes) == 0 {
			result.IssuesSkipped += len(plan.Issues)
			continue
		}

		// Apply fixes to the file
		modified, fixesApplied, err := applyFixesToFile(plan, opts)
		if err != nil {
			return nil, fmt.Errorf("apply fixes to %s: %w", plan.File, err)
		}

		if modified {
			result.FilesModified++
			result.IssuesFixed += fixesApplied
			result.ModifiedFiles = append(result.ModifiedFiles, plan.File)
		}
	}

	return result, nil
}

// applyFixesToFile applies fixes to a single file.
func applyFixesToFile(plan CleanupPlan, opts CleanupOptions) (bool, int, error) {
	if opts.DryRun {
		// In dry-run mode, just show what would be done
		fmt.Printf("  [DRY RUN] Would apply %d fix(es) to %s\n", len(plan.Fixes), plan.File)
		for i, fix := range plan.Fixes {
			fmt.Printf("    %d. %s\n", i+1, fix.Reason)
			fmt.Printf("       - %s\n", fix.OldContent)
			fmt.Printf("       + %s\n", fix.NewContent)
		}
		return false, 0, nil
	}

	// Read the file
	content, err := os.ReadFile(plan.File)
	if err != nil {
		return false, 0, fmt.Errorf("read file: %w", err)
	}

	originalContent := string(content)
	fixesApplied := 0

	// Apply each fix using line-by-line replacement
	lines := strings.Split(originalContent, "\n")
	modifiedLines := make([]string, len(lines))
	copy(modifiedLines, lines)

	for _, fix := range plan.Fixes {
		if !fix.AutoFix && !opts.AutoApply {
			continue // Skip manual fixes unless auto-apply is enabled
		}

		// Replace only on lines that match OldContent exactly
		for i, line := range modifiedLines {
			if strings.Contains(line, fix.OldContent) {
				modifiedLines[i] = strings.ReplaceAll(line, fix.OldContent, fix.NewContent)
				fixesApplied++
			}
		}
	}

	modifiedContent := strings.Join(modifiedLines, "\n")

	if fixesApplied == 0 {
		return false, 0, nil
	}

	// Create backup if requested
	if opts.Backup {
		backupPath := plan.File + ".backup"
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return false, 0, fmt.Errorf("create backup: %w", err)
		}
	}

	// Write the modified content
	if err := os.WriteFile(plan.File, []byte(modifiedContent), 0644); err != nil {
		return false, 0, fmt.Errorf("write file: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("  Applied %d fix(es) to %s\n", fixesApplied, plan.File)
	}

	return true, fixesApplied, nil
}

// FormatCleanupPlan formats a cleanup plan for display.
func FormatCleanupPlan(plan CleanupPlan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("File: %s\n", plan.File))
	sb.WriteString(fmt.Sprintf("Issues found: %d\n", len(plan.Issues)))
	sb.WriteString(fmt.Sprintf("Fixes proposed: %d\n\n", len(plan.Fixes)))

	for i, fix := range plan.Fixes {
		sb.WriteString(fmt.Sprintf("Fix #%d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Issue: %s\n", fix.Issue.Message))
		sb.WriteString(fmt.Sprintf("  Line: %d\n", fix.Issue.Reference.LineNumber))
		sb.WriteString(fmt.Sprintf("  Reason: %s\n", fix.Reason))
		sb.WriteString(fmt.Sprintf("  Auto-fix: %v\n", fix.AutoFix))
		sb.WriteString(fmt.Sprintf("  Old: %s\n", fix.OldContent))
		sb.WriteString(fmt.Sprintf("  New: %s\n\n", fix.NewContent))
	}

	return sb.String()
}

// FormatCleanupResult formats the cleanup result for display.
func FormatCleanupResult(result *CleanupResult) string {
	var sb strings.Builder

	sb.WriteString("Reference Cleanup Result\n")
	sb.WriteString("=======================\n\n")
	sb.WriteString(fmt.Sprintf("Files scanned: %d\n", result.FilesScanned))
	sb.WriteString(fmt.Sprintf("Files modified: %d\n", result.FilesModified))
	sb.WriteString(fmt.Sprintf("Issues found: %d\n", result.IssuesFound))
	sb.WriteString(fmt.Sprintf("Issues fixed: %d\n", result.IssuesFixed))
	sb.WriteString(fmt.Sprintf("Issues skipped: %d\n\n", result.IssuesSkipped))

	if len(result.ModifiedFiles) > 0 {
		sb.WriteString("Modified files:\n")
		for _, file := range result.ModifiedFiles {
			sb.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	return sb.String()
}

// QuickFix attempts to fix common reference issues automatically.
func QuickFix(filePath string) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	original := string(content)
	modified := original

	// Fix 1: Convert absolute markdown links to relative
	// [text](/docs/workstreams/backlog/00-001-01.md) -> [text](00-001-01.md)
	absLinkRe := regexp.MustCompile(`\[([^\]]+)\]\(/docs/workstreams/backlog/([0-9]{2}-[0-9]{3}(?:-[0-9]{2})?\.md)`)
	modified = absLinkRe.ReplaceAllString(modified, "[$1]($2")

	// Fix 2: Fix double slashes in relative paths
	// REMOVED: This pattern is too dangerous as it corrupts valid paths like "../../other/file.md"
	// A proper implementation would check if the path actually goes above repo root before fixing.

	// Fix 3: Normalize feature ID references
	// f042 -> F042
	featureIdRe := regexp.MustCompile(`\bf([0-9]{3})(?:-([0-9]{2}))?\b`)
	modified = featureIdRe.ReplaceAllStringFunc(modified, func(match string) string {
		parts := featureIdRe.FindStringSubmatch(match)
		if parts != nil {
			result := "F" + parts[1]
			if parts[2] != "" {
				result += "-" + parts[2]
			}
			return result
		}
		return match
	})

	if modified != original {
		if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Fixed common issues in %s\n", filePath)
	}

	return nil
}

// RemoveDeadReferences removes references to non-existent files.
func RemoveDeadReferences(filePath string, repoRoot string) (int, error) {
	opts := DefaultCheckOptions(repoRoot)

	// Check references in the file
	result, err := checkFileReferences(filePath, opts)
	if err != nil {
		return 0, fmt.Errorf("check references: %w", err)
	}

	removed := 0

	if len(result.Issues) == 0 {
		return 0, nil
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	modifiedLines := make([]string, 0, len(lines))

	for lineNum, line := range lines {
		shouldRemove := false

		// Check if this line has any error-level issues
		for _, issue := range result.Issues {
			if issue.Reference.LineNumber == lineNum+1 && issue.Severity == "error" {
				shouldRemove = true
				break
			}
		}

		if !shouldRemove {
			modifiedLines = append(modifiedLines, line)
		} else {
			removed++
		}
	}

	if removed > 0 {
		modified := strings.Join(modifiedLines, "\n")
		if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
			return 0, fmt.Errorf("write file: %w", err)
		}
	}

	return removed, nil
}

// filepathBase returns the base name of a path.
func filepathBase(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// BatchApplyCleanup applies cleanup to all workstream files in a directory.
// wsDir should be the path to the workstream directory (e.g., /path/to/repo/docs/workstreams/backlog).
func BatchApplyCleanup(wsDir string, opts CleanupOptions) (*CleanupResult, error) {
	// Extract repo root from wsDir by removing the workstream suffix
	// Assumes wsDir ends with "docs/workstreams/backlog"
	repoRoot := wsDir
	if strings.Contains(wsDir, "docs/workstreams/backlog") {
		repoRoot = strings.TrimSuffix(wsDir, "docs/workstreams/backlog")
		repoRoot = strings.TrimSuffix(repoRoot, "/")
	}

	// Run reference check
	checkOpts := DefaultCheckOptions(repoRoot)
	checkResult, err := CheckReferenceIntegrity(checkOpts)
	if err != nil {
		return nil, fmt.Errorf("check reference integrity: %w", err)
	}

	// Generate cleanup plans
	plans := GenerateCleanupPlans(checkResult)

	// Apply cleanup plans
	result, err := ApplyCleanupPlans(plans, opts)
	if err != nil {
		return nil, fmt.Errorf("apply cleanup plans: %w", err)
	}

	return result, nil
}

// InteractivelyFixReferences provides an interactive interface for fixing references.
func InteractivelyFixReferences(filePath string, repoRoot string) error {
	opts := DefaultCheckOptions(repoRoot)

	// Check references
	result, err := checkFileReferences(filePath, opts)
	if err != nil {
		return fmt.Errorf("check references: %w", err)
	}

	if len(result.Issues) == 0 {
		fmt.Println("No issues found!")
		return nil
	}

	fmt.Printf("Found %d issue(s) in %s:\n\n", len(result.Issues), filePath)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	lines := strings.Split(string(content), "\n")

	// Interactive fixing
	for i, issue := range result.Issues {
		fmt.Printf("Issue #%d:\n", i+1)
		fmt.Printf("  Line %d: %s\n", issue.Reference.LineNumber, issue.Message)
		if issue.Reference.LineNumber > 0 && issue.Reference.LineNumber <= len(lines) {
			fmt.Printf("  Context: %s\n", lines[issue.Reference.LineNumber-1])
		}

		fmt.Print("Fix? (y/n/s to skip/q to quit): ")
		var response string
		fmt.Scanln(&response)

		switch strings.ToLower(response) {
		case "y":
			fmt.Print("Enter replacement: ")
			var replacement string
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				replacement = scanner.Text()
			}
			// Apply replacement
			if issue.Reference.LineNumber > 0 && issue.Reference.LineNumber <= len(lines) {
				lines[issue.Reference.LineNumber-1] = replacement
			}
		case "q":
			fmt.Println("Quitting...")
			goto writeChanges
		case "s":
			// Skip
		}
	}

writeChanges:
	// Write modified content
	modified := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(modified), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Println("Changes applied.")
	return nil
}
