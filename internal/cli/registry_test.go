package cli

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	stats := r.Stats()
	if stats["total_commands"] != 0 {
		t.Errorf("expected 0 commands, got %v", stats["total_commands"])
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()

	cmd := &CommandMetadata{
		Name:        "test",
		Category:    "Test commands",
		Description: "A test command",
		Usage:       "sdp test <arg>",
	}

	err := r.Register(cmd)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify command was registered
	retrieved, exists := r.Lookup("test")
	if !exists {
		t.Fatal("command not found after registration")
	}

	if retrieved.Name != "test" {
		t.Errorf("expected name 'test', got %q", retrieved.Name)
	}

	if retrieved.Category != "Test commands" {
		t.Errorf("expected category 'Test commands', got %q", retrieved.Category)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	cmd1 := &CommandMetadata{
		Name:     "duplicate",
		Category: "Test",
	}
	cmd2 := &CommandMetadata{
		Name:     "duplicate",
		Category: "Test",
	}

	err := r.Register(cmd1)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err = r.Register(cmd2)
	if err == nil {
		t.Error("expected error when registering duplicate command, got nil")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	r := NewRegistry()

	cmd := &CommandMetadata{
		Name:     "",
		Category: "Test",
	}

	err := r.Register(cmd)
	if err == nil {
		t.Error("expected error for empty command name, got nil")
	}
}

func TestLookup(t *testing.T) {
	r := NewRegistry()

	cmd := &CommandMetadata{
		Name:        "lookup_test",
		Category:    "Test",
		Description: "Test lookup functionality",
	}

	r.Register(cmd)

	// Test existing command
	retrieved, exists := r.Lookup("lookup_test")
	if !exists {
		t.Fatal("command not found")
	}
	if retrieved.Description != "Test lookup functionality" {
		t.Errorf("wrong description: %q", retrieved.Description)
	}

	// Test non-existing command
	_, exists = r.Lookup("nonexistent")
	if exists {
		t.Error("found non-existent command")
	}
}

func TestList(t *testing.T) {
	r := NewRegistry()

	// Register mix of hidden and visible commands
	r.Register(&CommandMetadata{Name: "visible1", Category: "Test"})
	r.Register(&CommandMetadata{Name: "visible2", Category: "Test"})
	r.Register(&CommandMetadata{Name: "hidden", Category: "Test", Hidden: true})

	list := r.List()

	if len(list) != 2 {
		t.Errorf("expected 2 visible commands, got %d", len(list))
	}
}

func TestByCategory(t *testing.T) {
	r := NewRegistry()

	r.Register(&CommandMetadata{Name: "card", Category: "Card commands"})
	r.Register(&CommandMetadata{Name: "board", Category: "Board commands"})
	r.Register(&CommandMetadata{Name: "doctor", Category: "Doctor commands"})
	r.Register(&CommandMetadata{Name: "dispatch", Category: "Dispatch commands"})

	categories := r.ByCategory()

	if len(categories) != 4 {
		t.Errorf("expected 4 categories, got %d", len(categories))
	}

	cardCmds, ok := categories["Card commands"]
	if !ok {
		t.Fatal("Card commands category not found")
	}
	if len(cardCmds) != 1 {
		t.Errorf("expected 1 card command, got %d", len(cardCmds))
	}
}

func TestDeprecatedCommands(t *testing.T) {
	r := NewRegistry()

	r.Register(&CommandMetadata{Name: "active", Category: "Test"})
	r.Register(&CommandMetadata{
		Name:              "deprecated",
		Category:          "Test",
		Deprecated:        true,
		DeprecationMessage: "Use 'active' instead",
	})

	deprecated := r.DeprecatedCommands()

	if len(deprecated) != 1 {
		t.Errorf("expected 1 deprecated command, got %d", len(deprecated))
	}

	if deprecated[0].Name != "deprecated" {
		t.Errorf("expected deprecated command name 'deprecated', got %q", deprecated[0].Name)
	}

	if deprecated[0].DeprecationMessage != "Use 'active' instead" {
		t.Errorf("wrong deprecation message: %q", deprecated[0].DeprecationMessage)
	}
}

func TestGenerateHelp(t *testing.T) {
	r := NewRegistry()

	r.Register(&CommandMetadata{
		Name:        "card",
		Category:    "Card commands",
		Usage:       "sdp card <create|show|ready>",
		Description: "Manage feature cards",
	})

	r.Register(&CommandMetadata{
		Name:     "dispatch",
		Category: "Dispatch commands",
		Usage:    "sdp dispatch <card|next>",
	})

	help := r.GenerateHelp()

	if !contains(help, "usage: sdp <command>") {
		t.Error("help text missing usage header")
	}
	if !contains(help, "Card commands:") {
		t.Error("help text missing Card commands section")
	}
	if !contains(help, "sdp card <create|show|ready>") {
		t.Error("help text missing card usage")
	}
}

func TestGenerateCommandHelp(t *testing.T) {
	r := NewRegistry()

	r.Register(&CommandMetadata{
		Name:              "old",
		Category:          "Test",
		Deprecated:        true,
		DeprecationMessage: "Use 'new' instead",
		Description:       "Old command",
		Usage:             "sdp old <args>",
		Subcommands:       []string{"sub1", "sub2"},
		Examples:          []string{"sdp old sub1", "sdp old sub2"},
		IntroducedIn:      "v1.0.0",
		Aliases:           []string{"o", "oldcmd"},
	})

	help, err := r.GenerateCommandHelp("old")
	if err != nil {
		t.Fatalf("GenerateCommandHelp failed: %v", err)
	}

	if !contains(help, "DEPRECATED") {
		t.Error("help missing deprecation notice")
	}
	if !contains(help, "Use 'new' instead") {
		t.Error("help missing deprecation message")
	}
	if !contains(help, "Description: Old command") {
		t.Error("help missing description")
	}
	if !contains(help, "Subcommands:") {
		t.Error("help missing subcommands section")
	}
	if !contains(help, "Examples:") {
		t.Error("help missing examples section")
	}
	if !contains(help, "Introduced in: v1.0.0") {
		t.Error("help missing version info")
	}
	if !contains(help, "Aliases:") {
		t.Error("help missing aliases")
	}
}

func TestGenerateCommandHelpNotFound(t *testing.T) {
	r := NewRegistry()

	_, err := r.GenerateCommandHelp("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent command, got nil")
	}
}

func TestValidate(t *testing.T) {
	r := NewRegistry()

	// Test valid registration
	r.Register(&CommandMetadata{
		Name:     "valid",
		Category: "Test",
	})

	errors := r.Validate()
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got %d", len(errors))
	}

	// Test missing category
	r.Register(&CommandMetadata{
		Name:     "no_category",
		Category: "",
	})

	errors = r.Validate()
	if len(errors) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(errors))
	}
}

func TestStats(t *testing.T) {
	r := NewRegistry()

	r.Register(&CommandMetadata{Name: "active", Category: "Test"})
	r.Register(&CommandMetadata{Name: "deprecated", Category: "Test", Deprecated: true})
	r.Register(&CommandMetadata{Name: "hidden", Category: "Test", Hidden: true})

	stats := r.Stats()

	if stats["total_commands"] != 3 {
		t.Errorf("expected 3 total commands, got %v", stats["total_commands"])
	}
	if stats["deprecated_commands"] != 1 {
		t.Errorf("expected 1 deprecated command, got %v", stats["deprecated_commands"])
	}
	if stats["hidden_commands"] != 1 {
		t.Errorf("expected 1 hidden command, got %v", stats["hidden_commands"])
	}
	if stats["categories"] != 1 {
		t.Errorf("expected 1 category, got %v", stats["categories"])
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	cmd := &CommandMetadata{
		Name:     "global_test",
		Category: "Global",
	}

	err := RegisterCommand(cmd)
	if err != nil {
		t.Fatalf("RegisterCommand failed: %v", err)
	}

	retrieved, exists := GetRegistry().Lookup("global_test")
	if !exists {
		t.Fatal("command not found in global registry")
	}

	if retrieved.Name != "global_test" {
		t.Errorf("expected name 'global_test', got %q", retrieved.Name)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
