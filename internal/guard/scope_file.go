package guard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ScopeFile defines the guard scope for a worktree
type ScopeFile struct {
	AllowedPaths    []string `json:"allowed_paths"`
	AllowedCommands []string `json:"allowed_commands"`
	DeniedPatterns  []string `json:"denied_patterns"`
	MaxFileSize     int64    `json:"max_file_size,omitempty"`
	ReadOnlyPaths   []string `json:"read_only_paths,omitempty"`
}

// TaskMetadata from Gas Town worktree hook
type TaskMetadata struct {
	TaskID      string            `json:"task_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Priority    string            `json:"priority"`
	Constraints map[string]string `json:"constraints"`
}

// ScopeWriter writes guard scope files to worktrees
type ScopeWriter struct {
	sdpRoot string
}

// NewScopeWriter creates a new scope writer
func NewScopeWriter(sdpRoot string) *ScopeWriter {
	return &ScopeWriter{sdpRoot: sdpRoot}
}

// WriteScopeFile creates a .sdp/guard-scope.json file in the worktree
func (w *ScopeWriter) WriteScopeFile(worktreePath string, task TaskMetadata) error {
	// Create .sdp directory if it doesn't exist
	sdpDir := filepath.Join(worktreePath, ".sdp")
	if err := os.MkdirAll(sdpDir, 0755); err != nil {
		return fmt.Errorf("failed to create .sdp directory: %w", err)
	}

	// Generate scope from task constraints
	scope := w.generateScope(task)

	// Marshal to JSON
	data, err := json.MarshalIndent(scope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scope: %w", err)
	}

	// Write to file
	scopePath := filepath.Join(sdpDir, "guard-scope.json")
	if err := os.WriteFile(scopePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write scope file: %w", err)
	}

	return nil
}

// generateScope creates a scope file from task metadata
func (w *ScopeWriter) generateScope(task TaskMetadata) ScopeFile {
	scope := ScopeFile{
		AllowedPaths:    []string{"."}, // Default: allow all
		AllowedCommands: []string{"git", "go", "sdp"},
		DeniedPatterns:  []string{},
		MaxFileSize:     10 * 1024 * 1024, // 10MB default
		ReadOnlyPaths:   []string{},
	}

	// Apply constraints from task metadata
	if constraints, ok := task.Constraints["allowed_paths"]; ok {
		scope.AllowedPaths = parseStringList(constraints)
	}
	if constraints, ok := task.Constraints["allowed_commands"]; ok {
		scope.AllowedCommands = parseStringList(constraints)
	}
	if constraints, ok := task.Constraints["denied_patterns"]; ok {
		scope.DeniedPatterns = parseStringList(constraints)
	}

	return scope
}

// parseStringList parses a comma-separated string list
func parseStringList(s string) []string {
	if s == "" {
		return []string{}
	}

	// Simple split by comma
	// In production, this would handle quoted strings properly
	var result []string
	for _, item := range splitComma(s) {
		if trimmed := trimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Simple string manipulation functions to avoid dependency on strings package
func splitComma(s string) []string {
	var result []string
	current := ""
	for _, ch := range s {
		if ch == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
