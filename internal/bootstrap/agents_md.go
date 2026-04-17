package bootstrap

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/agents_md.tmpl
var agentsMDTemplateFS embed.FS

// GenerateAgentsMD produces a complete AGENTS.md from the collected data sources.
// It uses the embedded template and falls back gracefully when optional data is missing.
func GenerateAgentsMD(repoPath string, ds DataSourceInfo, cmds BuildCommands) (string, error) {
	data := buildAgentsMDData(repoPath, ds, cmds)

	tmpl, err := parseAgentsMDTemplate()
	if err != nil {
		return "", fmt.Errorf("agents_md: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("agents_md: execute template: %w", err)
	}

	return strings.TrimSpace(buf.String()) + "\n", nil
}

// MergeAgentsMD merges generated content into an existing AGENTS.md.
// User-authored content is preserved; generated sections are updated within markers.
func MergeAgentsMD(existing string, repoPath string, ds DataSourceInfo, cmds BuildCommands) (string, error) {
	generated, err := GenerateAgentsMD(repoPath, ds, cmds)
	if err != nil {
		return "", err
	}

	// Extract generated sections from the newly generated content.
	genSections := extractGeneratedSections(generated)

	// Replace only the generated blocks within the existing content.
	result := existing

	for name, content := range genSections {
		marker := sectionMarker(name)
		markerRegex := sectionMarkerRegexForName(name)

		if markerRegex.MatchString(result) {
			// Replace existing generated block for this section.
			result = markerRegex.ReplaceAllString(result, marker+"\n"+content+"\n"+endMarker())
		} else {
			// Section is new — insert at the right position.
			result = insertSection(result, name, content)
		}
	}

	return result, nil
}

// parseAgentsMDTemplate parses the embedded AGENTS.md template.
func parseAgentsMDTemplate() (*template.Template, error) {
	raw, err := agentsMDTemplateFS.ReadFile("templates/agents_md.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read embedded template: %w", err)
	}
	return template.New("agents_md").Parse(string(raw))
}

// buildAgentsMDData assembles the template data from all available sources.
func buildAgentsMDData(repoPath string, ds DataSourceInfo, cmds BuildCommands) AgentsMDTemplateData {
	name := filepath.Base(repoPath)

	data := AgentsMDTemplateData{
		Name:          name,
		Description:   fmt.Sprintf("Cross-harness agent instructions for %s", name),
		Agents:        defaultAgents(),
		CommitStyle:   "Conventional commits recommended",
		BranchPattern: "feature/<ticket>-<short-description>",
	}

	if ds.Scout != nil {
		data.Description = fmt.Sprintf("Cross-harness agent instructions for %s (%s project)", name, ds.Scout.PrimaryLanguage)
	}

	if ds.Metrics != nil {
		data.CommitStyle = detectCommitStyle(ds.Metrics)
		data.BranchPattern = detectBranchPattern(ds.Metrics)
	}

	// Build agent-specific notes based on available data.
	data.AgentNotes = buildAgentNotes(ds, cmds)

	return data
}

// defaultAgents returns the standard set of supported agents.
func defaultAgents() []AgentInfo {
	return []AgentInfo{
		{
			Name:        "Claude Code",
			Description: "Anthropic Claude CLI for code generation, review, and refactoring",
			BestFor:     "Complex code generation, architecture decisions, multi-file refactoring",
			ConfigPath:  "CLAUDE.md",
		},
		{
			Name:        "Codex CLI",
			Description: "OpenAI Codex agent for code tasks",
			BestFor:     "Code generation, test writing, documentation",
			ConfigPath:  "AGENTS.md",
		},
		{
			Name:        "Cursor",
			Description: "Cursor IDE with AI-powered editing",
			BestFor:     "Inline code editing, quick fixes, pair programming",
			ConfigPath:  ".cursorrules",
		},
		{
			Name:        "OpenCode",
			Description: "Terminal-based coding agent",
			BestFor:     "Terminal-driven development, script automation",
			ConfigPath:  "AGENTS.md",
		},
	}
}

// detectBranchPattern returns the branch naming pattern from metrics.
func detectBranchPattern(metrics *MetricsData) string {
	if metrics.BusFactor == 1 {
		return "feature/<short-description> or direct to main"
	}
	return "feature/<ticket>-<short-description>"
}

// buildAgentNotes creates agent-specific notes based on project data.
func buildAgentNotes(ds DataSourceInfo, cmds BuildCommands) []AgentNote {
	var notes []AgentNote

	// Build common notes from available data.
	var claudeNotes []string
	var codexNotes []string
	var sharedNotes []string

	if ds.Scout != nil {
		langNote := fmt.Sprintf("Primary language: %s", ds.Scout.PrimaryLanguage)
		claudeNotes = append(claudeNotes, langNote)
		codexNotes = append(codexNotes, langNote)
	}

	if cmds.Test != "" {
		testNote := fmt.Sprintf("Run tests before committing: `%s`", cmds.Test)
		sharedNotes = append(sharedNotes, testNote)
	}

	if cmds.Lint != "" {
		lintNote := fmt.Sprintf("Run lint before pushing: `%s`", cmds.Lint)
		sharedNotes = append(sharedNotes, lintNote)
	}

	if ds.Scout != nil && ds.Scout.HasCI {
		ciNote := fmt.Sprintf("CI system: %s — ensure changes pass CI before merging", ds.Scout.CISystem)
		sharedNotes = append(sharedNotes, ciNote)
	}

	claudeNotes = append(claudeNotes, "Read AGENTS.md for shared operator rules")
	claudeNotes = append(claudeNotes, "Use `rtk` prefix for shell commands to optimize token usage")

	codexNotes = append(codexNotes, "Follow conventions in AGENTS.md")
	codexNotes = append(codexNotes, "Use full commands (no rtk prefix)")

	if len(claudeNotes) > 0 {
		notes = append(notes, AgentNote{
			Agent: "Claude Code",
			Notes: "- " + strings.Join(claudeNotes, "\n- "),
		})
	}

	if len(codexNotes) > 0 {
		notes = append(notes, AgentNote{
			Agent: "Codex CLI",
			Notes: "- " + strings.Join(codexNotes, "\n- "),
		})
	}

	if len(sharedNotes) > 0 {
		notes = append(notes, AgentNote{
			Agent: "All Agents",
			Notes: "- " + strings.Join(sharedNotes, "\n- "),
		})
	}

	return notes
}
