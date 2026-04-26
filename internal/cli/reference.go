// Package cli provides CLI reference documentation generation and parity checking.
// F137-05: CLI reference + parity gate
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

// ReferenceDoc represents the full CLI reference documentation.
type ReferenceDoc struct {
	GeneratedAt string
	Version     string
	Categories  []CategorySection
	Stats       RegistryStats
}

// CategorySection groups commands by category.
type CategorySection struct {
	Name        string
	Description string
	Commands    []CommandReference
}

// CommandReference provides detailed information about a command.
type CommandReference struct {
	Name              string
	Description       string
	Usage             string
	Subcommands       []string
	Examples          []string
	Aliases           []string
	Deprecated        bool
	DeprecationMessage string
	IntroducedIn      string
	Hidden            bool
}

// RegistryStats provides statistics about the CLI registry.
type RegistryStats struct {
	TotalCommands      int
	DeprecatedCommands int
	HiddenCommands     int
	Categories         int
}

// GenerateReferenceDoc generates complete CLI reference documentation.
func GenerateReferenceDoc(version string) *ReferenceDoc {
	registry := GetRegistry()

	doc := &ReferenceDoc{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Version:     version,
	}

	// Get statistics
	stats := registry.Stats()
	doc.Stats = RegistryStats{
		TotalCommands:      safeInt(stats, "total_commands"),
		DeprecatedCommands: safeInt(stats, "deprecated_commands"),
		HiddenCommands:     safeInt(stats, "hidden_commands"),
		Categories:         safeInt(stats, "categories"),
	}

	// Build categories
	categories := registry.ByCategory()
	catNames := make([]string, 0, len(categories))
	for name := range categories {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	for _, catName := range catNames {
		cmds := categories[catName]
		section := CategorySection{
			Name:        catName,
			Description: getCategoryDescription(catName),
		}

		for _, cmd := range cmds {
			ref := CommandReference{
				Name:              cmd.Name,
				Description:       cmd.Description,
				Usage:             cmd.Usage,
				Subcommands:       cmd.Subcommands,
				Examples:          cmd.Examples,
				Aliases:           cmd.Aliases,
				Deprecated:        cmd.Deprecated,
				DeprecationMessage: cmd.DeprecationMessage,
				IntroducedIn:      cmd.IntroducedIn,
				Hidden:            cmd.Hidden,
			}
			section.Commands = append(section.Commands, ref)
		}

		// Sort commands within category
		sort.Slice(section.Commands, func(i, j int) bool {
			return section.Commands[i].Name < section.Commands[j].Name
		})

		doc.Categories = append(doc.Categories, section)
	}

	return doc
}

// getCategoryDescription returns a description for a category.
func getCategoryDescription(category string) string {
	descriptions := map[string]string{
		"Card commands":                         "Manage feature cards through their lifecycle",
		"Board commands":                        "Manage kanban boards for projects",
		"Doctor commands":                       "Diagnose issues with SDP control state, adapters, and backlog",
		"Dispatch commands":                     "Dispatch cards for execution",
		"Result commands":                       "Ingest and manage execution results",
		"Orchestrate commands":                  "Run orchestration loop for result processing",
		"Query commands (require beads/dual mode)": "Query cards and traces (requires beads or dual mode)",
		"Deploy commands":                       "Manage deployments to staging and production",
		"Discovery commands (Stage 0)":          "Run discovery pipeline for new features",
		"Pipeline commands":                     "Pipeline operations for card processing",
		"Phase commands":                        "Run phase-specific operations",
		"Scout commands":                        "Scout repository for code patterns",
		"Spec commands":                         "Extract specifications from repository",
		"Index commands":                        "Build and query repository indexes",
		"Bootstrap commands":                    "Initialize repository with SDP workstreams",
		"Rules commands":                        "Update rules from evidence sources",
		"Build commands":                        "Run build pipeline for feature ideas",
		"Reset commands":                        "Reset checkpoints for features",
		"Coverage commands":                     "Scan code coverage",
		"Skills commands":                       "Manage and augment skills",
		"Deprecated":                            "Deprecated commands (use alternatives)",
	}

	if desc, ok := descriptions[category]; ok {
		return desc
	}
	return ""
}

// WriteMarkdown writes the CLI reference as a Markdown document.
func (doc *ReferenceDoc) WriteMarkdown(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Write markdown content
	fmt.Fprintf(file, "# SDP CLI Reference\n\n")
	fmt.Fprintf(file, "**Generated:** %s\n", doc.GeneratedAt)
	fmt.Fprintf(file, "**Version:** %s\n\n", doc.Version)

	// Write statistics
	fmt.Fprintf(file, "## Statistics\n\n")
	fmt.Fprintf(file, "- **Total Commands:** %d\n", doc.Stats.TotalCommands)
	fmt.Fprintf(file, "- **Deprecated Commands:** %d\n", doc.Stats.DeprecatedCommands)
	fmt.Fprintf(file, "- **Hidden Commands:** %d\n", doc.Stats.HiddenCommands)
	fmt.Fprintf(file, "- **Categories:** %d\n\n", doc.Stats.Categories)

	// Write table of contents
	fmt.Fprintf(file, "## Table of Contents\n\n")
	for _, cat := range doc.Categories {
		anchor := strings.ToLower(strings.ReplaceAll(cat.Name, " ", "-"))
		fmt.Fprintf(file, "- [%s](#%s)\n", cat.Name, anchor)
	}
	fmt.Fprintf(file, "\n")

	// Write categories
	for _, cat := range doc.Categories {
		fmt.Fprintf(file, "## %s\n\n", cat.Name)
		if cat.Description != "" {
			fmt.Fprintf(file, "%s\n\n", cat.Description)
		}

		for _, cmd := range cat.Commands {
			fmt.Fprintf(file, "### %s\n\n", cmd.Name)

			if cmd.Deprecated {
				fmt.Fprintf(file, "**⚠️ DEPRECATED**\n\n")
				if cmd.DeprecationMessage != "" {
					fmt.Fprintf(file, "%s\n\n", cmd.DeprecationMessage)
				}
			}

			if cmd.Description != "" {
				fmt.Fprintf(file, "%s\n\n", cmd.Description)
			}

			if cmd.Usage != "" {
				fmt.Fprintf(file, "**Usage:**\n```\n%s\n```\n\n", cmd.Usage)
			}

			if len(cmd.Subcommands) > 0 {
				fmt.Fprintf(file, "**Subcommands:**\n")
				for _, sub := range cmd.Subcommands {
					fmt.Fprintf(file, "- `%s`\n", sub)
				}
				fmt.Fprintf(file, "\n")
			}

			if len(cmd.Examples) > 0 {
				fmt.Fprintf(file, "**Examples:**\n")
				for _, ex := range cmd.Examples {
					fmt.Fprintf(file, "```bash\n%s\n```\n", ex)
				}
				fmt.Fprintf(file, "\n")
			}

			if len(cmd.Aliases) > 0 {
				fmt.Fprintf(file, "**Aliases:** %s\n\n", strings.Join(cmd.Aliases, ", "))
			}

			if cmd.IntroducedIn != "" {
				fmt.Fprintf(file, "**Introduced in:** %s\n\n", cmd.IntroducedIn)
			}
		}
	}

	return nil
}

// WriteJSON writes the CLI reference as a JSON document.
func (doc *ReferenceDoc) WriteJSON(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// For simplicity, we'll use the markdown writer for now
	// In a real implementation, you'd use json.Marshal here
	return fmt.Errorf("JSON output not implemented yet")
}

// ParityCheckResult represents the result of a parity check between harnesses.
type ParityCheckResult struct {
	Passed       bool
	Missing      []string // Commands missing in target
	Extra        []string // Commands extra in target
	Deprecated   []string // Deprecated commands still in use
	VersionDiff  []string // Commands with version differences
	Summary      string
}

// CheckHarnessParity compares the current CLI registry against a reference harness.
// This is used in CI gates to ensure parity across different harness implementations.
func CheckHarnessParity(referenceCommands []string) *ParityCheckResult {
	registry := GetRegistry()
	result := &ParityCheckResult{
		Passed:      true,
		Missing:     []string{},
		Extra:       []string{},
		Deprecated:  []string{},
		VersionDiff: []string{},
	}

	currentCommands := registry.List()

	// Build a map of current commands
	currentMap := make(map[string]bool)
	for _, cmd := range currentCommands {
		currentMap[cmd.Name] = true
	}

	// Build a map of reference commands
	referenceMap := make(map[string]bool)
	for _, cmd := range referenceCommands {
		referenceMap[cmd] = true
	}

	// Check for missing commands
	for _, refCmd := range referenceCommands {
		if !currentMap[refCmd] {
			// Check if this is a deprecated command that we know about
			cmd, exists := registry.Lookup(refCmd)
			if exists && cmd.Deprecated {
				// Deprecated commands in reference don't fail parity, just note them
				result.Deprecated = append(result.Deprecated, refCmd)
			} else {
				// Truly missing commands fail parity
				result.Missing = append(result.Missing, refCmd)
				result.Passed = false
			}
		}
	}

	// Check for extra commands
	for _, currCmd := range currentCommands {
		if !referenceMap[currCmd.Name] && !currCmd.Hidden {
			result.Extra = append(result.Extra, currCmd.Name)
			// Extra commands don't fail parity check, they're logged
		}
	}

	// Check for deprecated commands still in use (that exist in both)
	deprecated := registry.DeprecatedCommands()
	for _, cmd := range deprecated {
		if referenceMap[cmd.Name] && currentMap[cmd.Name] {
			result.Deprecated = append(result.Deprecated, cmd.Name)
		}
	}

	// Generate summary
	result.generateSummary()

	return result
}

// generateSummary creates a human-readable summary of the parity check.
func (r *ParityCheckResult) generateSummary() {
	var parts []string

	if len(r.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing command(s)", len(r.Missing)))
	}

	if len(r.Extra) > 0 {
		parts = append(parts, fmt.Sprintf("%d extra command(s)", len(r.Extra)))
	}

	if len(r.Deprecated) > 0 {
		parts = append(parts, fmt.Sprintf("%d deprecated command(s) still in reference", len(r.Deprecated)))
	}

	if r.Passed {
		r.Summary = fmt.Sprintf("Parity check passed: %s", strings.Join(parts, ", "))
	} else {
		r.Summary = fmt.Sprintf("Parity check failed: %s", strings.Join(parts, ", "))
	}
}

// FormatParityReport returns a formatted parity check report.
func (r *ParityCheckResult) FormatParityReport() string {
	var sb strings.Builder

	sb.WriteString(r.Summary)
	sb.WriteString("\n")

	if len(r.Missing) > 0 {
		sb.WriteString("\nMissing commands:\n")
		for _, cmd := range r.Missing {
			sb.WriteString(fmt.Sprintf("  - %s\n", cmd))
		}
	}

	if len(r.Extra) > 0 {
		sb.WriteString("\nExtra commands (not in reference):\n")
		for _, cmd := range r.Extra {
			sb.WriteString(fmt.Sprintf("  - %s\n", cmd))
		}
	}

	if len(r.Deprecated) > 0 {
		sb.WriteString("\nDeprecated commands in reference:\n")
		for _, cmd := range r.Deprecated {
			sb.WriteString(fmt.Sprintf("  - %s\n", cmd))
		}
	}

	return sb.String()
}

// ExitStatusForParity returns the appropriate exit code for a parity check result.
func (r *ParityCheckResult) ExitStatusForParity() int {
	if r.Passed {
		return 0
	}
	return 1
}

// GenerateManPage generates a Unix man page for a command.
func GenerateManPage(commandName string) (string, error) {
	registry := GetRegistry()
	cmd, exists := registry.Lookup(commandName)

	if !exists {
		return "", fmt.Errorf("command %q not found", commandName)
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(".TH SDP %s \"User Commands\" \"SDP Manual\"\n", commandName))
	sb.WriteString(".SH NAME\n")
	sb.WriteString(fmt.Sprintf("%s \\- %s\n", commandName, cmd.Description))
	sb.WriteString(".SH SYNOPSIS\n")
	sb.WriteString(fmt.Sprintf(".B %s\n\n", cmd.Usage))
	sb.WriteString(".SH DESCRIPTION\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))

	if len(cmd.Subcommands) > 0 {
		sb.WriteString(".SH SUBCOMMANDS\n")
		for _, sub := range cmd.Subcommands {
			sb.WriteString(fmt.Sprintf(".TP\n.B %s\n%s\n", sub, sub))
		}
	}

	if len(cmd.Examples) > 0 {
		sb.WriteString(".SH EXAMPLES\n")
		for _, ex := range cmd.Examples {
			sb.WriteString(fmt.Sprintf(".PP\n.nf\n%s\n.fi\n", ex))
		}
	}

	if cmd.Deprecated {
		sb.WriteString(".SH DEPRECATED\n")
		sb.WriteString(fmt.Sprintf("This command is deprecated: %s\n", cmd.DeprecationMessage))
	}

	return sb.String(), nil
}

// ReferenceTemplate is the template for generating reference documentation.
var ReferenceTemplate = template.Must(template.New("cli-reference").Parse(`
# SDP CLI Reference

Generated: {{.GeneratedAt}}
Version: {{.Version}}

## Statistics

- Total Commands: {{.Stats.TotalCommands}}
- Deprecated Commands: {{.Stats.DeprecatedCommands}}
- Hidden Commands: {{.Stats.HiddenCommands}}
- Categories: {{.Stats.Categories}}

{{range .Categories}}
## {{.Name}}

{{.Description}}

{{range .Commands}}
### {{.Name}}

{{if .Deprecated}}**DEPRECATED**: {{.DeprecationMessage}}{{end}}

{{.Description}}

**Usage:**
` + "```" + `
{{.Usage}}
` + "```" + `

{{if .Subcommands}}**Subcommands:**
{{range .Subcommands}}
- {{.}}
{{end}}{{end}}

{{if .Examples}}**Examples:**
{{range .Examples}}
` + "```bash" + `
{{.}}
` + "```" + `
{{end}}{{end}}

{{end}}
{{end}}
`))

// safeInt safely extracts an int from a map[string]interface{} with a default value of 0
func safeInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}
