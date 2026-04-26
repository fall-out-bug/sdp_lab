// Package registry provides CLI registry discovery for MCP tool generation.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Discovery discovers CLI commands and their metadata for MCP tool generation.
type Discovery struct {
	binaryPath string
	repoRoot   string
}

// NewDiscovery creates a new CLI registry discovery.
func NewDiscovery(binaryPath, repoRoot string) *Discovery {
	return &Discovery{
		binaryPath: binaryPath,
		repoRoot:   repoRoot,
	}
}

// BinaryPath returns the binary path for the discovery.
func (d *Discovery) BinaryPath() string {
	return d.binaryPath
}

// RepoRoot returns the repository root for the discovery.
func (d *Discovery) RepoRoot() string {
	return d.repoRoot
}

// CommandInfo represents information about a CLI command.
type CommandInfo struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Flags       []FlagInfo        `json:"flags"`
	Arguments   []ArgumentInfo    `json:"arguments"`
	Source      string            `json:"source"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FlagInfo represents information about a CLI flag.
type FlagInfo struct {
	Name        string      `json:"name"`
	Short       string      `json:"short,omitempty"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	EnumValues  []string    `json:"enum_values,omitempty"`
}

// ArgumentInfo represents information about a CLI argument.
type ArgumentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	Required    bool   `json:"required"`
	Variadic    bool   `json:"variadic"`
}

// DiscoverAll discovers all CLI commands available in the registry.
func (d *Discovery) DiscoverAll() ([]CommandInfo, error) {
	// Try to get help in JSON format
	commands, err := d.discoverFromHelpJSON()
	if err == nil && len(commands) > 0 {
		return commands, nil
	}

	// Fall back to parsing help text
	return d.discoverFromHelpText()
}

// discoverFromHelpJSON discovers commands using `--help --json` if available.
func (d *Discovery) discoverFromHelpJSON() ([]CommandInfo, error) {
	cmd := exec.Command(d.binaryPath, "--help", "--json")
	cmd.Dir = d.repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get help JSON: %w", err)
	}

	var helpOutput struct {
		Commands []struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Flags       []FlagInfo        `json:"flags"`
			Metadata    map[string]string `json:"metadata,omitempty"`
		} `json:"commands"`
	}

	if err := json.Unmarshal(output, &helpOutput); err != nil {
		return nil, fmt.Errorf("parse help JSON: %w", err)
	}

	commands := make([]CommandInfo, len(helpOutput.Commands))
	for i, cmd := range helpOutput.Commands {
		commands[i] = CommandInfo{
			Name:        cmd.Name,
			Path:        cmd.Name,
			Description: cmd.Description,
			Flags:       cmd.Flags,
			Metadata:    cmd.Metadata,
		}
	}

	return commands, nil
}

// discoverFromHelpText discovers commands by parsing text help output.
func (d *Discovery) discoverFromHelpText() ([]CommandInfo, error) {
	cmd := exec.Command(d.binaryPath, "--help")
	cmd.Dir = d.repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get help text: %w", err)
	}

	// Parse help text to extract commands
	// This is a simplified parser - real implementation would be more robust
	lines := strings.Split(string(output), "\n")
	var commands []CommandInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for command definitions (format: "  commandName  description")
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				commands = append(commands, CommandInfo{
					Name:        parts[0],
					Path:        parts[0],
					Description: strings.Join(parts[1:], " "),
				})
			}
		}
	}

	return commands, nil
}

// DiscoverCommand discovers detailed information about a specific command.
func (d *Discovery) DiscoverCommand(commandPath string) (*CommandInfo, error) {
	// Split command path into parts (e.g., "index build" -> ["index", "build"])
	parts := strings.Fields(commandPath)

	// Get command-specific help
	args := append(parts, "--help")
	if jsonSupported() {
		args = append(args, "--json")
	}

	cmd := exec.Command(d.binaryPath, args...)
	cmd.Dir = d.repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get command help for %s: %w", commandPath, err)
	}

	// Try JSON first
	if jsonSupported() {
		var cmdHelp struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Flags       []FlagInfo        `json:"flags"`
			Arguments   []ArgumentInfo    `json:"arguments"`
			Metadata    map[string]string `json:"metadata,omitempty"`
		}

		if err := json.Unmarshal(output, &cmdHelp); err == nil {
			return &CommandInfo{
				Name:        cmdHelp.Name,
				Path:        commandPath,
				Description: cmdHelp.Description,
				Flags:       cmdHelp.Flags,
				Arguments:   cmdHelp.Arguments,
				Metadata:    cmdHelp.Metadata,
			}, nil
		}
	}

	// Fall back to text parsing
	return d.parseCommandHelp(commandPath, string(output))
}

// parseCommandHelp parses command help text to extract flags and arguments.
func (d *Discovery) parseCommandHelp(commandPath, helpText string) (*CommandInfo, error) {
	info := &CommandInfo{
		Name: filepath.Base(commandPath),
		Path: commandPath,
	}

	lines := strings.Split(helpText, "\n")
	var inFlags bool
	var inUsage bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract description from first paragraph
		if info.Description == "" && line != "" && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "Usage:") {
			info.Description = line
		}

		// Parse usage line for arguments
		if strings.HasPrefix(line, "Usage:") {
			inUsage = true
			continue
		}

		if inUsage && line != "" {
			// Parse arguments from usage line
			usageParts := strings.Fields(line)
			for i, part := range usageParts {
				if strings.HasPrefix(part, "<") && strings.HasSuffix(part, ">") {
					info.Arguments = append(info.Arguments, ArgumentInfo{
						Name:     strings.Trim(part, "<>"),
						Position: i,
						Required: true,
					})
				} else if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
					if strings.Contains(part, "<") {
						info.Arguments = append(info.Arguments, ArgumentInfo{
							Name:     strings.Trim(strings.Trim(part, "[]"), "<>"),
							Position: i,
							Required: false,
						})
					}
				}
			}
			inUsage = false
		}

		// Parse flags
		if strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "Options:") {
			inFlags = true
			continue
		}

		if inFlags {
			if line == "" {
				inFlags = false
				continue
			}

			flagInfo := d.parseFlagLine(line)
			if flagInfo != nil {
				info.Flags = append(info.Flags, *flagInfo)
			}
		}
	}

	return info, nil
}

// parseFlagLine parses a single flag definition line.
func (d *Discovery) parseFlagLine(line string) *FlagInfo {
	// Flag format: "-f, --flag    description (default: value)"
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	flagNames := parts[0]
	description := strings.Join(parts[1:], " ")

	var short, long string
	for _, name := range strings.Split(flagNames, ",") {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(name, "--") {
			long = strings.TrimPrefix(name, "--")
		} else if strings.HasPrefix(name, "-") && len(name) == 2 {
			short = strings.TrimPrefix(name, "-")
		}
	}

	if long == "" {
		return nil
	}

	flagType := "string"
	required := false
	var defaultValue interface{}

	// Detect type and default from description
	if strings.Contains(description, "(default:") {
		// Extract default value
		defaultStart := strings.Index(description, "(default:")
		defaultEnd := strings.Index(description[defaultStart:], ")")
		if defaultEnd > 0 {
			defaultStr := strings.TrimSpace(description[defaultStart+9 : defaultStart+defaultEnd])
			defaultValue = defaultStr

			// Infer type from default value
			if strings.EqualFold(defaultStr, "true") || strings.EqualFold(defaultStr, "false") {
				flagType = "boolean"
			} else if strings.Contains(defaultStr, ".") {
				flagType = "number"
			}
		}
	}

	// Detect enum type
	if strings.Contains(description, "one of:") || strings.Contains(description, "( ") {
		enumStart := strings.Index(description, "( ")
		if enumStart >= 0 {
			enumEnd := strings.Index(description[enumStart:], ")")
			if enumEnd > 0 {
				enumStr := strings.TrimSpace(description[enumStart+2 : enumStart+enumEnd])
				enumValues := strings.Split(enumStr, "|")
				if len(enumValues) > 1 {
					flagType = "enum"
					return &FlagInfo{
						Name:        long,
						Short:       short,
						Description: description,
						Type:        flagType,
						Required:    required,
						EnumValues:  enumValues,
					}
				}
			}
		}
	}

	return &FlagInfo{
		Name:        long,
		Short:       short,
		Description: description,
		Type:        flagType,
		Required:    required,
		Default:     defaultValue,
	}
}

// GetSourceLocation finds the source file where a command is implemented.
func (d *Discovery) GetSourceLocation(commandPath string) (string, error) {
	// Convert command path to potential source file names
	parts := strings.Fields(commandPath)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command path")
	}

	// Try different patterns
	patterns := []string{
		fmt.Sprintf("cmd/sdp/cmd_%s.go", parts[0]),
		fmt.Sprintf("cmd/sdp/%s.go", parts[0]),
		fmt.Sprintf("cmd/sdp-%s/main.go", parts[0]),
	}

	// Try relative to repo root
	for _, pattern := range patterns {
		path := filepath.Join(d.repoRoot, pattern)
		if _, err := os.Stat(path); err == nil {
			return pattern, nil
		}
	}

	// Try in cmd/sdp/ directory
	cmdDir := filepath.Join(d.repoRoot, "cmd", "sdp")
	for _, pattern := range patterns {
		path := filepath.Join(cmdDir, pattern)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("source location not found for command: %s", commandPath)
}

// Snapshot creates a registry snapshot for hash generation.
func (d *Discovery) Snapshot() (*Snapshot, error) {
	commands, err := d.DiscoverAll()
	if err != nil {
		return nil, fmt.Errorf("discover commands: %w", err)
	}

	entries := make([]CommandEntry, len(commands))
	for i, cmd := range commands {
		entries[i] = CommandEntry{
			Name:        cmd.Name,
			Flags:       convertFlags(cmd.Flags),
			Source:      cmd.Source,
			Metadata:    cmd.Metadata,
		}
	}

	return &Snapshot{
		Commands: entries,
	}, nil
}

// Snapshot represents a serializable registry snapshot.
type Snapshot struct {
	Commands []CommandEntry `json:"commands"`
}

// CommandEntry represents a command in the snapshot.
type CommandEntry struct {
	Name     string            `json:"name"`
	Flags    []FlagEntry       `json:"flags"`
	Source   string            `json:"source"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// FlagEntry represents a flag in the snapshot.
type FlagEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Default  interface{} `json:"default,omitempty"`
}

func convertFlags(flags []FlagInfo) []FlagEntry {
	result := make([]FlagEntry, len(flags))
	for i, f := range flags {
		result[i] = FlagEntry{
			Name:     f.Name,
			Type:     f.Type,
			Required: f.Required,
			Default:  f.Default,
		}
	}
	return result
}

func jsonSupported() bool {
	// Check if the CLI supports JSON output
	// This is a heuristic - real implementation might check version or features
	return true
}