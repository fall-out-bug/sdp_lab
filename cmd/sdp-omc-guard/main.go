// Package main implements sdp-omc-guard, a pre-tool-call guard hook for OhMyOpenCode.
//
// The guard reads PreToolUseInput from stdin, extracts file paths from tool_input,
// checks them against the declared scope, and returns exit codes:
//   - 0: allow (all files in scope)
//   - 1: ask (some files uncertain)
//   - 2: deny (files out of scope)
//
// Usage:
//
//	sdp-omc-guard --ws 00-059-01 --session-id abc123
//
// Input (stdin): PreToolUseInput JSON from OhMyOpenCode
// Output (stdout): GuardResult JSON
// Errors (stderr): Human-readable error messages
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/guard"
	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
	"github.com/fall-out-bug/sdp_lab/internal/session"
)

// PreToolUseInput is the input structure from OhMyOpenCode PreToolUse hook.
type PreToolUseInput struct {
	SessionID      string                 `json:"session_id"`
	TranscriptPath string                 `json:"transcript_path,omitempty"`
	CWD            string                 `json:"cwd"`
	PermissionMode string                 `json:"permission_mode,omitempty"`
	HookEventName  string                 `json:"hook_event_name"`
	ToolName       string                 `json:"tool_name"`
	ToolInput      map[string]interface{} `json:"tool_input"`
	ToolUseID      string                 `json:"tool_use_id,omitempty"`
	HookSource     string                 `json:"hook_source,omitempty"`
}

// GuardResult is the output structure for the guard decision.
type GuardResult struct {
	Decision   string   `json:"decision"`   // allow, deny, ask
	Reason     string   `json:"reason"`     // Human-readable reason
	WSID       string   `json:"ws_id"`      // Workstream ID checked
	SessionID  string   `json:"session_id"` // Session ID
	Files      []string `json:"files"`      // Files checked
	Violations []string `json:"violations,omitempty"`
	Allowed    bool     `json:"allowed"`
}

func main() {
	wsID := flag.String("ws", "", "Workstream ID (e.g. 00-059-01)")
	sessionID := flag.String("session-id", "", "Session ID for evidence logging")
	emitEvidence := flag.Bool("emit-evidence", true, "Emit guard_check event to session log")
	flag.Parse()

	if *wsID == "" {
		fmt.Fprintln(os.Stderr, "error: --ws is required")
		flag.Usage()
		os.Exit(2)
	}

	// Read input from stdin
	var input PreToolUseInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "error: decode input: %v\n", err)
		os.Exit(2)
	}

	// Use session ID from flag or input
	sid := *sessionID
	if sid == "" {
		sid = input.SessionID
	}

	// Determine project root
	cwd := input.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err)
			os.Exit(2)
		}
	}

	projectRoot, err := orchestrate.FindProjectRoot(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err)
		os.Exit(2)
	}

	// Extract files from tool input
	files := extractFilesFromToolInput(input.ToolName, input.ToolInput)

	// Check scope
	result := GuardResult{
		WSID:      *wsID,
		SessionID: sid,
		Files:     files,
	}

	// Read operations are allowed without scope restriction (they don't modify files)
	// but are still logged for evidence
	isReadOperation := strings.HasPrefix(input.ToolName, "read")

	if len(files) == 0 || isReadOperation {
		// No files to check or read operation - allow
		result.Decision = "allow"
		result.Allowed = true
		if isReadOperation {
			result.Reason = "read operation - allowed without scope restriction"
		} else {
			result.Reason = "no files to check"
		}
	} else {
		verdict, err := checkFilesAgainstScope(projectRoot, *wsID, files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: check scope: %v\n", err)
			os.Exit(2)
		}

		result.Violations = verdict.Violations
		result.Allowed = verdict.Pass

		if verdict.Pass {
			result.Decision = "allow"
			if len(verdict.Warnings) > 0 {
				result.Reason = fmt.Sprintf("allowed with %d warnings (allowlisted files)", len(verdict.Warnings))
			} else {
				result.Reason = "all files in scope"
			}
		} else {
			result.Decision = "deny"
			result.Reason = fmt.Sprintf("scope violation: %d files out of scope", len(verdict.Violations))
		}
	}

	// Emit evidence if requested
	if *emitEvidence && sid != "" {
		if err := emitGuardCheckEvent(projectRoot, sid, result); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "warning: emit evidence: %v\n", err)
		}
	}

	// Output result as JSON
	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal result: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(string(output))

	// Exit with appropriate code
	switch result.Decision {
	case "allow":
		os.Exit(0)
	case "ask":
		os.Exit(1)
	case "deny":
		os.Exit(2)
	default:
		os.Exit(2)
	}
}

// extractFilesFromToolInput extracts file paths from tool input based on tool type.
func extractFilesFromToolInput(toolName string, input map[string]interface{}) []string {
	var files []string

	switch {
	case strings.HasPrefix(toolName, "edit"):
		// edit tool: file_path
		if fp, ok := input["file_path"].(string); ok && fp != "" {
			files = append(files, fp)
		}
	case toolName == "write":
		// write tool: file_path
		if fp, ok := input["file_path"].(string); ok && fp != "" {
			files = append(files, fp)
		}
	case toolName == "bash":
		// bash tool: extract file paths from command (best effort)
		// This is a heuristic - we look for common file operations
		if cmd, ok := input["command"].(string); ok {
			files = extractFilesFromBashCommand(cmd)
		}
	case strings.HasPrefix(toolName, "read"):
		// read tool: file_path - typically allowed without restriction
		// but we still track it for evidence
		if fp, ok := input["file_path"].(string); ok && fp != "" {
			files = append(files, fp)
		}
	}

	return files
}

// extractFilesFromBashCommand extracts file paths from bash commands.
// This is a heuristic that looks for common patterns.
func extractFilesFromBashCommand(cmd string) []string {
	var files []string

	// Look for common file operations
	// This is intentionally conservative - we only extract obvious file paths
	patterns := []string{
		`> `,         // redirect output
		`>> `,        // append output
		` < `,        // redirect input
		` -o `,       // output file flag
		` --output `, // output file flag
	}

	for _, pattern := range patterns {
		if idx := strings.Index(cmd, pattern); idx >= 0 {
			// Extract the word after the pattern
			rest := strings.TrimSpace(cmd[idx+len(pattern):])
			if spaceIdx := strings.Index(rest, " "); spaceIdx > 0 {
				candidate := rest[:spaceIdx]
				if looksLikeFilePath(candidate) {
					files = append(files, candidate)
				}
			} else if rest != "" && looksLikeFilePath(rest) {
				files = append(files, rest)
			}
		}
	}

	return files
}

// looksLikeFilePath returns true if the string looks like a file path.
func looksLikeFilePath(s string) bool {
	// Skip obvious non-files
	if s == "" || s == "|" || s == "&" || s == "&&" || s == "||" {
		return false
	}
	// Skip flags
	if strings.HasPrefix(s, "-") {
		return false
	}
	// Skip common bash keywords
	switch s {
	case "then", "else", "fi", "do", "done", "if", "while", "for":
		return false
	}
	// Must contain a path separator or look like a filename
	return strings.Contains(s, "/") || strings.Contains(s, ".")
}

// checkFilesAgainstScope checks files against the declared scope for a workstream.
func checkFilesAgainstScope(projectRoot, wsID string, files []string) (*guard.ScopeVerdict, error) {
	// Load scope from workstream file
	wsPath := filepath.Join(projectRoot, "docs", "workstreams", "backlog", wsID+".md")
	scopePaths, err := guard.ParseScopeFiles(wsPath)
	if err != nil {
		return nil, fmt.Errorf("parse scope files: %w", err)
	}

	// Build scope set
	scopeSet := make(map[string]bool)
	for _, p := range scopePaths {
		scopeSet[p] = true
	}

	// Load allowlist
	allowlist, err := guard.LoadAllowlist(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load allowlist: %w", err)
	}

	// Check each file
	var violations, warnings []string
	for _, f := range files {
		// Make path relative to project root if absolute
		if filepath.IsAbs(f) {
			rel, err := filepath.Rel(projectRoot, f)
			if err == nil {
				f = rel
			}
		}

		if isInScope(f, scopeSet) {
			continue
		}
		if guard.IsAllowlisted(f, allowlist) {
			warnings = append(warnings, f)
			continue
		}
		violations = append(violations, f)
	}

	return &guard.ScopeVerdict{
		Pass:       len(violations) == 0,
		Violations: violations,
		Warnings:   warnings,
	}, nil
}

// isInScope checks if a file is in the declared scope.
func isInScope(file string, scopeSet map[string]bool) bool {
	// Exact match
	if scopeSet[file] {
		return true
	}
	// Prefix match (for directories)
	for scopePath := range scopeSet {
		if strings.HasPrefix(file, scopePath+"/") || strings.HasPrefix(file, scopePath) {
			return true
		}
	}
	return false
}

// emitGuardCheckEvent emits a guard_check event to the session log.
func emitGuardCheckEvent(projectRoot, sessionID string, result GuardResult) error {
	paths := session.NewPaths(projectRoot)
	if err := paths.EnsureLogDir(); err != nil {
		return err
	}

	writer, err := session.NewWriter(projectRoot, sessionID)
	if err != nil {
		return err
	}
	defer func() { _ = writer.Close() }()

	_, err = writer.AppendGuardCheck(
		result.WSID,
		"pre-tool-call",
		result.Files,
		result.Allowed,
		result.Violations,
		result.Reason,
	)
	return err
}
