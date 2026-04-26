// Package cli provides a command registry for SDP CLI surface normalization.
// F137-02: Registry core + discovery contract
package cli

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CommandMetadata describes a CLI command for discovery and documentation.
type CommandMetadata struct {
	// Name is the command name (e.g., "card", "dispatch")
	Name string

	// Category groups related commands (e.g., "Card commands", "Doctor commands")
	Category string

	// Description is a short one-line description
	Description string

	// Usage shows the command syntax (e.g., "sdp card <create|show|ready>")
	Usage string

	// Deprecated indicates if this command is deprecated
	Deprecated bool

	// DeprecationMessage provides migration guidance if Deprecated is true
	DeprecationMessage string

	// Subcommands lists available subcommands (e.g., ["create", "show", "ready"])
	Subcommands []string

	// Examples shows common usage patterns
	Examples []string

	// Hidden indicates if command should be hidden from help output
	Hidden bool

	// Version when this command was introduced
	IntroducedIn string

	// Aliases are alternative names for this command
	Aliases []string
}

// Registry maintains a registry of all CLI commands.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*CommandMetadata
	byCat    map[string][]*CommandMetadata
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]*CommandMetadata),
		byCat:    make(map[string][]*CommandMetadata),
	}
	return r
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd *CommandMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cmd.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if _, exists := r.commands[cmd.Name]; exists {
		return fmt.Errorf("command %q already registered", cmd.Name)
	}

	r.commands[cmd.Name] = cmd
	r.byCat[cmd.Category] = append(r.byCat[cmd.Category], cmd)

	return nil
}

// Lookup retrieves a command by name.
func (r *Registry) Lookup(name string) (*CommandMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cmd, exists := r.commands[name]
	return cmd, exists
}

// List returns all registered commands.
func (r *Registry) List() []*CommandMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*CommandMetadata, 0, len(r.commands))
	for _, cmd := range r.commands {
		if !cmd.Hidden {
			result = append(result, cmd)
		}
	}
	return result
}

// ByCategory returns commands grouped by category.
func (r *Registry) ByCategory() map[string][]*CommandMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a deep copy to avoid race conditions
	result := make(map[string][]*CommandMetadata)
	for cat, cmds := range r.byCat {
		filtered := make([]*CommandMetadata, 0, len(cmds))
		for _, cmd := range cmds {
			if !cmd.Hidden {
				filtered = append(filtered, cmd)
			}
		}
		if len(filtered) > 0 {
			result[cat] = filtered
		}
	}
	return result
}

// DeprecatedCommands returns all deprecated commands.
func (r *Registry) DeprecatedCommands() []*CommandMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*CommandMetadata
	for _, cmd := range r.commands {
		if cmd.Deprecated {
			result = append(result, cmd)
		}
	}
	return result
}

// GenerateHelp generates a help text for all commands.
func (r *Registry) GenerateHelp() string {
	var sb strings.Builder

	sb.WriteString("usage: sdp <command> [subcommand] [flags]\n\n")

	// Get categories and sort them
	categories := r.ByCategory()
	catNames := make([]string, 0, len(categories))
	for name := range categories {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	// Output each category
	for _, catName := range catNames {
		cmds := categories[catName]
		sb.WriteString(fmt.Sprintf("%s:\n", catName))

		// Sort commands within category
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].Name < cmds[j].Name
		})

		for _, cmd := range cmds {
			if cmd.Deprecated {
				continue
			}
			if cmd.Usage != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", cmd.Usage))
			} else {
				sb.WriteString(fmt.Sprintf("  sdp %s\n", cmd.Name))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateCommandHelp generates detailed help for a specific command.
func (r *Registry) GenerateCommandHelp(name string) (string, error) {
	cmd, exists := r.Lookup(name)
	if !exists {
		return "", fmt.Errorf("command %q not found", name)
	}

	var sb strings.Builder

	if cmd.Deprecated {
		sb.WriteString(fmt.Sprintf("DEPRECATED: %s\n", cmd.DeprecationMessage))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Command: sdp %s\n\n", cmd.Name))

	if cmd.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n\n", cmd.Description))
	}

	if cmd.Usage != "" {
		sb.WriteString(fmt.Sprintf("Usage: %s\n\n", cmd.Usage))
	}

	if len(cmd.Subcommands) > 0 {
		sb.WriteString("Subcommands:\n")
		for _, sub := range cmd.Subcommands {
			sb.WriteString(fmt.Sprintf("  - %s\n", sub))
		}
		sb.WriteString("\n")
	}

	if len(cmd.Examples) > 0 {
		sb.WriteString("Examples:\n")
		for _, ex := range cmd.Examples {
			sb.WriteString(fmt.Sprintf("  %s\n", ex))
		}
		sb.WriteString("\n")
	}

	if len(cmd.Aliases) > 0 {
		sb.WriteString(fmt.Sprintf("Aliases: %s\n\n", strings.Join(cmd.Aliases, ", ")))
	}

	if cmd.IntroducedIn != "" {
		sb.WriteString(fmt.Sprintf("Introduced in: %s\n", cmd.IntroducedIn))
	}

	return sb.String(), nil
}

// Validate checks for common issues in the registry.
func (r *Registry) Validate() []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errors []error

	// Check for empty names
	for name, cmd := range r.commands {
		if cmd.Name != name {
			errors = append(errors, fmt.Errorf("command key %q doesn't match metadata name %q", name, cmd.Name))
		}
		if cmd.Category == "" {
			errors = append(errors, fmt.Errorf("command %q has no category", name))
		}
	}

	// Check for duplicate subcommands across categories
	seenSubs := make(map[string]string) // subcommand -> command
	for _, cmd := range r.commands {
		for _, sub := range cmd.Subcommands {
			key := cmd.Name + " " + sub
			if existing, ok := seenSubs[key]; ok {
				errors = append(errors, fmt.Errorf("duplicate subcommand %q (in %s and %s)", key, existing, cmd.Name))
			}
			seenSubs[key] = cmd.Name
		}
	}

	return errors
}

// Stats returns registry statistics.
func (r *Registry) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_commands":      len(r.commands),
		"deprecated_commands": 0,
		"hidden_commands":     0,
		"categories":          len(r.byCat),
	}

	for _, cmd := range r.commands {
		if cmd.Deprecated {
			stats["deprecated_commands"] = stats["deprecated_commands"].(int) + 1
		}
		if cmd.Hidden {
			stats["hidden_commands"] = stats["hidden_commands"].(int) + 1
		}
	}

	return stats
}

// Global registry instance
var globalRegistry = NewRegistry()

// GetRegistry returns the global command registry.
func GetRegistry() *Registry {
	return globalRegistry
}

// RegisterCommand is a convenience function to register with the global registry.
func RegisterCommand(cmd *CommandMetadata) error {
	return globalRegistry.Register(cmd)
}
