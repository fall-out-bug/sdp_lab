// Package cli provides shim wrappers for backward compatibility during CLI migration.
// F137-04: Shim wrappers + deprecation warnings
package cli

import (
	"fmt"
	"os"
	"strings"
)

// DeprecatedCommand records a deprecated command pattern and its replacement.
type DeprecatedCommand struct {
	OldPattern     string // Old command pattern
	NewCommand     string // New replacement command
	MigrationGuide string // Additional migration guidance
	RemovedIn      string // Version when this will be removed (optional)
}

// deprecatedPatterns tracks deprecated command patterns.
var deprecatedPatterns = []DeprecatedCommand{
	// Example: Old flag-based commands moved to subcommands
	{
		OldPattern:     "sdp --project",
		NewCommand:     "sdp card --project",
		MigrationGuide: "Global --project flag has been removed. Use command-specific subcommands.",
		RemovedIn:      "v2.0.0",
	},
}

// ShimCommand wraps a command handler with deprecation checking.
// If the command is deprecated, it prints a warning and may suggest alternatives.
type ShimCommand struct {
	Name        string
	Handler     func(args []string)
	Deprecated  bool
	Replacement string
	RemovedIn   string
}

// Execute runs the command with deprecation warnings if applicable.
func (s *ShimCommand) Execute(args []string) {
	if s.Deprecated {
		printDeprecationWarning(s.Name, s.Replacement, s.RemovedIn)
	}
	s.Handler(args)
}

// printDeprecationWarning prints a formatted deprecation warning.
func printDeprecationWarning(oldCmd, newCmd, removedIn string) {
	fmt.Fprintf(os.Stderr, "\n  ⚠️  DEPRECATED: sdp %s\n\n", oldCmd)
	fmt.Fprintf(os.Stderr, "  This command is deprecated and will be removed in %s\n\n", removedIn)
	if newCmd != "" {
		fmt.Fprintf(os.Stderr, "  Replacement: sdp %s\n\n", newCmd)
	}
	fmt.Fprintf(os.Stderr, "  Please update your scripts and workflows.\n\n")
}

// CheckForDeprecatedPatterns checks if args contain deprecated patterns.
// Returns a list of warnings for any deprecated patterns found.
func CheckForDeprecatedPatterns(args []string) []string {
	var warnings []string

	for _, pattern := range deprecatedPatterns {
		if containsPattern(args, pattern.OldPattern) {
			warning := fmt.Sprintf("Deprecated pattern '%s' detected. %s", pattern.OldPattern, pattern.MigrationGuide)
			if pattern.NewCommand != "" {
				warning += fmt.Sprintf("\n  Use: %s", pattern.NewCommand)
			}
			if pattern.RemovedIn != "" {
				warning += fmt.Sprintf("\n  Removed in: %s", pattern.RemovedIn)
			}
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

// containsPattern checks if args contain a specific pattern.
// For multi-word patterns, it checks if the args start with the pattern words.
func containsPattern(args []string, pattern string) bool {
	patternParts := strings.Split(pattern, " ")

	// If pattern is a single word, check exact match or prefix
	if len(patternParts) == 1 {
		for _, arg := range args {
			if arg == pattern || len(arg) > len(pattern) && arg[:len(pattern)] == pattern {
				return true
			}
		}
		return false
	}

	// For multi-word patterns, check if args start with the pattern words
	if len(args) < len(patternParts) {
		return false
	}

	for i := 0; i <= len(args)-len(patternParts); i++ {
		matches := true
		for j, part := range patternParts {
			if args[i+j] != part && (!strings.HasPrefix(args[i+j], part) || len(args[i+j]) <= len(part)) {
				matches = false
				break
			}
			// For args that are longer than the pattern part, check prefix
			if len(args[i+j]) > len(part) && !strings.HasPrefix(args[i+j], part) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}

	return false
}

// PrintDeprecatedWarnings prints all deprecation warnings to stderr.
func PrintDeprecatedWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, "\n⚠️  Deprecation Warnings:")
	fmt.Fprintln(os.Stderr)
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "  %s\n", warning)
		fmt.Fprintln(os.Stderr)
	}
}

// RegisterDeprecatedCommand registers a deprecated command in the registry.
// This is used during migration to maintain backward compatibility while guiding users to new commands.
func RegisterDeprecatedCommand(oldName, newName, removedIn, description string) error {
	cmd := &CommandMetadata{
		Name:              oldName,
		Category:          "Deprecated",
		Description:       description,
		Deprecated:        true,
		DeprecationMessage: fmt.Sprintf("Use '%s' instead", newName),
		Hidden:            false, // Show in help but marked as deprecated
		IntroducedIn:      "v1.0.0",
	}

	return RegisterCommand(cmd)
}

// MigrateLegacyArgs transforms legacy argument patterns to modern equivalents.
// This allows old scripts to continue working while guiding users to new patterns.
func MigrateLegacyArgs(args []string) ([]string, bool) {
	// Example migrations
	// No migrations needed currently, but this function provides a hook
	// for future compatibility shims

	migrated := false
	newArgs := make([]string, len(args))
	copy(newArgs, args)

	// Add migration logic here as needed
	// Example: if args[0] == "old-command" {
	//            newArgs[0] = "new-command"
	//            migrated = true
	//          }

	return newArgs, migrated
}

// ShimWrapper wraps a command function with deprecation checking and argument migration.
// It returns a new function that can be used in place of the original command handler.
func ShimWrapper(originalHandler func(args []string), deprecatedCmd string) func(args []string) {
	return func(args []string) {
		// Check for deprecated patterns
		warnings := CheckForDeprecatedPatterns(args)
		if len(warnings) > 0 {
			PrintDeprecatedWarnings(warnings)
		}

		// Migrate legacy arguments if needed
		newArgs, wasMigrated := MigrateLegacyArgs(args)
		if wasMigrated {
			fmt.Fprintf(os.Stderr, "Note: Arguments were migrated to modern format.\n")
			fmt.Fprintf(os.Stderr, "Please update your scripts to use the new format.\n\n")
		}

		// Execute the original handler
		originalHandler(newArgs)
	}
}

// ValidateCommandForDeprecated checks if a specific command name is deprecated.
// Returns (isDeprecated, metadata) where metadata contains deprecation info.
func ValidateCommandForDeprecated(cmdName string) (bool, *CommandMetadata) {
	registry := GetRegistry()
	metadata, exists := registry.Lookup(cmdName)

	if !exists {
		return false, nil
	}

	if metadata.Deprecated {
		return true, metadata
	}

	return false, metadata // Return metadata even for non-deprecated commands
}

// GetMigrationPath returns the recommended migration path for a deprecated command.
// Returns empty string if the command is not deprecated or has no clear migration path.
func GetMigrationPath(cmdName string) string {
	isDeprecated, metadata := ValidateCommandForDeprecated(cmdName)

	if !isDeprecated {
		return ""
	}

	// Extract the replacement from the deprecation message
	// Format: "Use 'new-command' instead"
	msg := metadata.DeprecationMessage
	if len(msg) > 5 && msg[:4] == "Use " {
		// Extract command between quotes
		// Skip past the opening quote
		start := 5 // Position after "Use '"
		end := start
		for end < len(msg) && msg[end] != '\'' {
			end++
		}
		if end < len(msg) {
			return msg[start:end]
		}
	}

	return ""
}
