// Package workstream provides workstream template generation.
package workstream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/beads"
)

// WorkstreamTemplate generates workstream files from formulas.
type WorkstreamTemplate struct {
	projectID string
	featureID string
	outputDir string
}

// TemplateConfig configures the workstream template generator.
type TemplateConfig struct {
	// ProjectID is the project prefix (default: "00").
	ProjectID string

	// FeatureID is the feature identifier (e.g., "F061").
	FeatureID string

	// OutputDir is where workstream files are written.
	OutputDir string
}

// NewWorkstreamTemplate creates a new template generator.
func NewWorkstreamTemplate(cfg TemplateConfig) *WorkstreamTemplate {
	if cfg.ProjectID == "" {
		cfg.ProjectID = "00"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "docs/workstreams/backlog"
	}

	return &WorkstreamTemplate{
		projectID: cfg.ProjectID,
		featureID: cfg.FeatureID,
		outputDir: cfg.OutputDir,
	}
}

// Generate generates workstream files from a formula.
func (wt *WorkstreamTemplate) Generate(formula *beads.Formula, vars map[string]string) ([]string, error) {
	var generatedFiles []string

	// Resolve variables (merge defaults with provided)
	resolvedVars := wt.resolveVariables(formula, vars)

	// Generate a feature ID if not provided
	featureID := wt.featureID
	if featureID == "" {
		featureID = fmt.Sprintf("F%s", strings.ToUpper(strings.ReplaceAll(formula.Name, "-", "")))
	}

	// Generate workstream for each step
	for i, step := range formula.Steps {
		wsID := fmt.Sprintf("%s-%s-%02d", wt.projectID, featureID, i+1)

		content, err := wt.generateWorkstream(formula, step, wsID, featureID, resolvedVars, i)
		if err != nil {
			return nil, fmt.Errorf("generate step %d: %w", i, err)
		}

		// Write file
		filename := fmt.Sprintf("%s.md", wsID)
		path := filepath.Join(wt.outputDir, filename)

		if err := os.MkdirAll(wt.outputDir, 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}

		if err := os.WriteFile(path, content, 0644); err != nil {
			return nil, fmt.Errorf("write file: %w", err)
		}

		generatedFiles = append(generatedFiles, path)
	}

	return generatedFiles, nil
}

// resolveVariables merges formula defaults with provided variables.
func (wt *WorkstreamTemplate) resolveVariables(formula *beads.Formula, vars map[string]string) map[string]interface{} {
	resolved := make(map[string]interface{})

	// Set defaults
	for name, v := range formula.Variables {
		if v.Default != nil {
			resolved[name] = v.Default
		}
	}

	// Override with provided
	for name, value := range vars {
		resolved[name] = value
	}

	return resolved
}

// generateWorkstream generates a single workstream file.
func (wt *WorkstreamTemplate) generateWorkstream(
	formula *beads.Formula,
	step beads.FormulaStep,
	wsID, featureID string,
	vars map[string]interface{},
	stepIndex int,
) ([]byte, error) {
	// Substitute variables in step
	step = wt.substituteStepVars(step, vars)

	// Build dependencies
	var dependsOn []string
	for _, dep := range step.Dependencies {
		// Convert step name to WS ID
		for i, s := range formula.Steps {
			if s.Name == dep {
				dependsOn = append(dependsOn, fmt.Sprintf("%s-%s-%02d", wt.projectID, featureID, i+1))
				break
			}
		}
	}

	// Build template data
	data := map[string]interface{}{
		"WSID":               wsID,
		"FeatureID":          featureID,
		"StepName":           step.Name,
		"Title":              step.Title,
		"Description":        step.Description,
		"Type":               step.Type,
		"Priority":           step.Priority,
		"Size":               step.Size,
		"DependsOn":          dependsOn,
		"ScopeFiles":         step.ScopeFiles,
		"AcceptanceCriteria": step.AcceptanceCriteria,
		"FormulaName":        formula.Name,
		"FormulaVersion":     formula.Version,
		"FormulaHash":        wt.formulaHash(formula),
		"GeneratedAt":        time.Now().Format(time.RFC3339),
	}

	// Set defaults
	if data["Type"] == "" {
		data["Type"] = "task"
	}
	if data["Priority"] == 0 {
		data["Priority"] = 2
	}
	if data["Size"] == "" {
		data["Size"] = "M"
	}
	if step.Title == "" {
		data["Title"] = step.Name
	}

	// Execute template
	var buf bytes.Buffer
	if err := wsTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// substituteStepVars substitutes variables in step fields.
func (wt *WorkstreamTemplate) substituteStepVars(step beads.FormulaStep, vars map[string]interface{}) beads.FormulaStep {
	// Substitute in title
	step.Title = wt.substituteVars(step.Title, vars)

	// Substitute in description
	step.Description = wt.substituteVars(step.Description, vars)

	// Substitute in scope files
	for i, f := range step.ScopeFiles {
		step.ScopeFiles[i] = wt.substituteVars(f, vars)
	}

	// Substitute in acceptance criteria
	for i, ac := range step.AcceptanceCriteria {
		step.AcceptanceCriteria[i] = wt.substituteVars(ac, vars)
	}

	return step
}

// substituteVars substitutes {{var}} and {{.var}} patterns in a string.
func (wt *WorkstreamTemplate) substituteVars(s string, vars map[string]interface{}) string {
	for name, value := range vars {
		// Handle both {{var}} and {{.var}} patterns
		placeholder1 := fmt.Sprintf("{{%s}}", name)
		placeholder2 := fmt.Sprintf("{{.%s}}", name)
		s = strings.ReplaceAll(s, placeholder1, fmt.Sprintf("%v", value))
		s = strings.ReplaceAll(s, placeholder2, fmt.Sprintf("%v", value))
	}
	return s
}

// formulaHash returns a hash of the formula for versioning.
func (wt *WorkstreamTemplate) formulaHash(formula *beads.Formula) string {
	h := sha256.New()
	h.Write([]byte(formula.Name))
	h.Write([]byte(formula.Version))
	for _, step := range formula.Steps {
		h.Write([]byte(step.Name))
	}
	hash := hex.EncodeToString(h.Sum(nil))
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

// wsTemplate is the workstream markdown template.
var wsTemplate = template.Must(template.New("workstream").Parse(`---
ws_id: {{.WSID}}
feature_id: {{.FeatureID}}
status: backlog
priority: P{{.Priority}}
size: {{.Size}}
{{- if .DependsOn}}
depends_on: [{{range $i, $d := .DependsOn}}{{if $i}}, {{end}}{{$d}}{{end}}]
{{- end}}
ws_kind: leaf
parent_ws_id: null
dispatch_lifecycle: active
formula:
  name: {{.FormulaName}}
  version: {{.FormulaVersion}}
  hash: {{.FormulaHash}}
generated: {{.GeneratedAt}}
---

# {{.WSID}}: {{.Title}}

Feature: {{.FeatureID}} ({{.FormulaName}})

## Goal

{{if .Description}}{{.Description}}{{else}}Complete the {{.StepName}} step.{{end}}

## Scope Files

{{- range .ScopeFiles}}
- ` + "`{{.}}`" + `
{{- else}}
- TBD
{{- end}}

## Acceptance Criteria

{{- range .AcceptanceCriteria}}
- [ ] {{.}}
{{- else}}
- [ ] Implementation complete
- [ ] Tests pass
{{- end}}

## Beads

## Reference

- Formula: {{.FormulaName}} v{{.FormulaVersion}}
`))
