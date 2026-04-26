package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestShimCommandExecute(t *testing.T) {
	called := false
	handler := func(args []string) {
		called = true
	}

	// Test non-deprecated command
	shim := &ShimCommand{
		Name:    "test",
		Handler: handler,
	}
	shim.Execute(nil)

	if !called {
		t.Error("handler was not called for non-deprecated command")
	}
}

func TestShimCommandExecuteDeprecated(t *testing.T) {
	called := false
	handler := func(args []string) {
		called = true
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	shim := &ShimCommand{
		Name:        "old-cmd",
		Handler:     handler,
		Deprecated:  true,
		Replacement: "new-cmd",
		RemovedIn:   "v2.0.0",
	}
	shim.Execute(nil)

	w.Close()
	os.Stderr = oldStderr

	// Read captured stderr
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !called {
		t.Error("handler was not called for deprecated command")
	}

	if !strings.Contains(output, "DEPRECATED") {
		t.Error("deprecation warning not printed")
	}
	if !strings.Contains(output, "new-cmd") {
		t.Error("replacement command not shown")
	}
	if !strings.Contains(output, "v2.0.0") {
		t.Error("removal version not shown")
	}
}

func TestCheckForDeprecatedPatterns(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()
	initDeprecatedPatterns()

	tests := []struct {
		name        string
		args        []string
		wantWarning bool
	}{
		{
			name:        "no deprecated patterns",
			args:        []string{"card", "show"},
			wantWarning: false,
		},
		{
			name:        "has deprecated pattern",
			args:        []string{"sdp", "--project", "test"},
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := CheckForDeprecatedPatterns(tt.args)
			hasWarning := len(warnings) > 0

			if hasWarning != tt.wantWarning {
				t.Errorf("CheckForDeprecatedPatterns() warning = %v, want %v", hasWarning, tt.wantWarning)
			}
		})
	}
}

func TestPrintDeprecatedWarnings(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	warnings := []string{
		"Warning 1",
		"Warning 2",
	}
	PrintDeprecatedWarnings(warnings)

	w.Close()
	os.Stderr = oldStderr

	// Read captured stderr
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Deprecation Warnings") {
		t.Error("header not printed")
	}
	if !strings.Contains(output, "Warning 1") {
		t.Error("warning 1 not printed")
	}
	if !strings.Contains(output, "Warning 2") {
		t.Error("warning 2 not printed")
	}
}

func TestRegisterDeprecatedCommand(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	err := RegisterDeprecatedCommand("old-command", "new-command", "v2.0.0", "Old command description")
	if err != nil {
		t.Fatalf("RegisterDeprecatedCommand failed: %v", err)
	}

	// Verify command was registered
	cmd, exists := GetRegistry().Lookup("old-command")
	if !exists {
		t.Fatal("deprecated command not registered")
	}

	if !cmd.Deprecated {
		t.Error("command not marked as deprecated")
	}

	if !strings.Contains(cmd.DeprecationMessage, "new-command") {
		t.Error("deprecation message missing replacement command")
	}
}

func TestMigrateLegacyArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMigrate bool
	}{
		{
			name:        "no migration needed",
			args:        []string{"card", "show"},
			wantMigrate: false,
		},
		{
			name:        "migration needed (if implemented)",
			args:        []string{"old-command"},
			wantMigrate: false, // No migrations implemented yet
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newArgs, wasMigrated := MigrateLegacyArgs(tt.args)

			if wasMigrated != tt.wantMigrate {
				t.Errorf("MigrateLegacyArgs() migrated = %v, want %v", wasMigrated, tt.wantMigrate)
			}

			if !wasMigrated && len(newArgs) != len(tt.args) {
				t.Error("args length changed when no migration occurred")
			}
		})
	}
}

func TestShimWrapper(t *testing.T) {
	called := false
	calledWithArgs := false
	handler := func(args []string) {
		called = true
		if len(args) > 0 {
			calledWithArgs = true
		}
	}

	wrapped := ShimWrapper(handler, "test")

	// Capture stderr to suppress warnings in test output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	wrapped([]string{"arg1", "arg2"})

	w.Close()
	os.Stderr = oldStderr
	_ = r // Discard captured stderr

	if !called {
		t.Error("handler was not called")
	}

	if !calledWithArgs {
		t.Error("handler was not called with args")
	}
}

func TestValidateCommandForDeprecated(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	// Register a deprecated command
	RegisterDeprecatedCommand("deprecated-cmd", "new-cmd", "v2.0.0", "Test")
	// Register a non-deprecated command
	RegisterCommand(&CommandMetadata{
		Name:     "active-cmd",
		Category: "Test",
	})

	tests := []struct {
		name            string
		cmdName         string
		wantDeprecated  bool
		wantMetadata    bool
	}{
		{
			name:           "deprecated command",
			cmdName:        "deprecated-cmd",
			wantDeprecated: true,
			wantMetadata:   true,
		},
		{
			name:           "active command",
			cmdName:        "active-cmd",
			wantDeprecated: false,
			wantMetadata:   true,
		},
		{
			name:           "non-existent command",
			cmdName:        "non-existent",
			wantDeprecated: false,
			wantMetadata:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify command was actually registered
			_, exists := globalRegistry.Lookup(tt.cmdName)
			t.Logf("cmd=%s exists in globalRegistry: %v", tt.cmdName, exists)

			isDeprecated, metadata := ValidateCommandForDeprecated(tt.cmdName)

			t.Logf("cmd=%s, isDeprecated=%v, metadata=%v", tt.cmdName, isDeprecated, metadata != nil)

			if isDeprecated != tt.wantDeprecated {
				t.Errorf("ValidateCommandForDeprecated() deprecated = %v, want %v", isDeprecated, tt.wantDeprecated)
			}

			hasMetadata := metadata != nil
			if hasMetadata != tt.wantMetadata {
				t.Errorf("ValidateCommandForDeprecated() metadata = %v, want %v", hasMetadata, tt.wantMetadata)
			}
		})
	}
}

func TestGetMigrationPath(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	// Register a deprecated command
	RegisterDeprecatedCommand("old-cmd", "new-cmd", "v2.0.0", "Test")

	tests := []struct {
		name          string
		cmdName       string
		wantMigration string
	}{
		{
			name:          "deprecated command with migration",
			cmdName:       "old-cmd",
			wantMigration: "new-cmd",
		},
		{
			name:          "non-existent command",
			cmdName:       "non-existent",
			wantMigration: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migration := GetMigrationPath(tt.cmdName)

			if migration != tt.wantMigration {
				t.Errorf("GetMigrationPath() = %q, want %q", migration, tt.wantMigration)
			}
		})
	}
}

// Helper function to initialize deprecated patterns for testing
func initDeprecatedPatterns() {
	// This is a no-op now, but provides a hook if we need to reset patterns
}
