package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "1.0.0"

// Planner generates bootstrap plans and reports without mutating files.
type Planner struct {
	// Config holds the user-supplied bootstrap configuration.
	Config BootstrapConfig
	// Collector reads .sdp/ data for the repo.
	Collector *Collector
}

// NewPlanner creates a Planner for the given configuration.
func NewPlanner(cfg BootstrapConfig) *Planner {
	return &Planner{
		Config:    cfg,
		Collector: NewCollector(cfg.RepoPath),
	}
}

// Plan generates a BootstrapPlan describing what bootstrap will do.
// This is a read-only operation; no files are created or modified.
func (p *Planner) Plan() (*BootstrapPlan, error) {
	ds, err := p.Collector.Collect()
	if err != nil {
		return nil, err
	}

	lang := ""
	if ds.Scout != nil {
		lang = ds.Scout.PrimaryLanguage
	}

	cmds := DetectBuildCommands(p.Config.RepoPath, lang)
	plan := &BootstrapPlan{
		DataSources: *ds,
		Commands:    cmds,
	}

	onlyMap := onlySet(p.Config.Only)

	// Plan CLAUDE.md
	if shouldGenerate("claude-md", onlyMap) {
		p.planArtifact(plan, "claude_md", "CLAUDE.md",
			"SDP harness configuration for Claude Code")
	}

	// Plan AGENTS.md (independent from claude-md)
	if shouldGenerate("agents-md", onlyMap) {
		p.planArtifact(plan, "agents_md", "AGENTS.md",
			"Cross-harness agent instructions")
	}

	// Plan policies
	if shouldGenerate("policies", onlyMap) {
		p.planArtifact(plan, "policy", ".sdp/policies",
			"SDP policy configuration directory")
	}

	// Plan hooks
	if shouldGenerate("hooks", onlyMap) {
		p.planArtifact(plan, "hook", ".claude/hooks",
			"Claude Code hooks directory")
	}

	// Plan beads (opt-in only — requires --beads flag or explicit --only beads)
	if p.Config.Beads || (len(onlyMap) > 0 && onlyMap["beads"]) {
		p.planArtifact(plan, "beads", ".beads",
			"Beads issue tracking directory")
	}

	return plan, nil
}

// DryRun executes a plan and returns a report without writing any files.
func (p *Planner) DryRun() (*BootstrapReport, error) {
	start := time.Now()

	plan, err := p.Plan()
	if err != nil {
		return nil, err
	}

	report := &BootstrapReport{
		Version:     version,
		GeneratedAt: start.UTC(),
		Repo:        p.Config.RepoPath,
		DataSources: p.Collector.DataSourcesAvailable(),
		Confidence:  computeConfidence(plan),
		Notes:       generateNotes(plan),
	}

	// Convert plan items to artifact results with "dry_run" status
	// (since we're in dry-run mode, nothing is actually written).
	for _, a := range plan.WillCreate {
		report.Artifacts = append(report.Artifacts, ArtifactResult{
			Type:    a.Type,
			Path:    a.Path,
			Action:  a.Action,
			Status:  "dry_run",
			Message: a.Description,
		})
	}
	for _, a := range plan.WillMerge {
		report.Artifacts = append(report.Artifacts, ArtifactResult{
			Type:    a.Type,
			Path:    a.Path,
			Action:  a.Action,
			Status:  "dry_run",
			Message: a.Description,
		})
	}
	for _, a := range plan.WillSkip {
		report.Artifacts = append(report.Artifacts, ArtifactResult{
			Type:    a.Type,
			Path:    a.Path,
			Action:  a.Action,
			Status:  "skipped",
			Message: a.Description,
		})
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

// Execute runs the bootstrap plan and writes files.
// Returns a report of what was done. Uses idempotent writes — only writes
// files whose content actually changed. Runs verification unless --no-verify
// is set.
func (p *Planner) Execute() (*BootstrapReport, error) {
	start := time.Now()

	plan, err := p.Plan()
	if err != nil {
		return nil, err
	}

	report := &BootstrapReport{
		Version:     version,
		GeneratedAt: start.UTC(),
		Repo:        p.Config.RepoPath,
		DataSources: p.Collector.DataSourcesAvailable(),
		Confidence:  computeConfidence(plan),
		Notes:       generateNotes(plan),
	}

	// Create artifacts.
	for _, a := range plan.WillCreate {
		result := p.executeArtifact(a)
		report.Artifacts = append(report.Artifacts, result)
		trackKeptOrUpdated(&result, report)
	}

	// Merge artifacts.
	for _, a := range plan.WillMerge {
		result := p.executeArtifact(a)
		report.Artifacts = append(report.Artifacts, result)
		trackKeptOrUpdated(&result, report)
	}

	// Skip artifacts.
	for _, a := range plan.WillSkip {
		report.Artifacts = append(report.Artifacts, ArtifactResult{
			Type:    a.Type,
			Path:    a.Path,
			Action:  "skip",
			Status:  "skipped",
			Message: a.Description,
		})
		report.Kept = append(report.Kept, a.Path)
	}

	// Run verification unless --no-verify is set.
	if !p.Config.NoVerify {
		lang := ""
		if plan.DataSources.Scout != nil {
			lang = plan.DataSources.Scout.PrimaryLanguage
		}
		cmds := DetectBuildCommands(p.Config.RepoPath, lang)
		if cmds.Build != "" || cmds.Test != "" || cmds.Lint != "" {
			results := VerifyCommands(context.Background(), cmds, p.Config.RepoPath)
			report.Verification = results

			if !AllPassed(results) {
				failed := UnverifiedCommands(results)
				report.Notes = append(report.Notes,
					fmt.Sprintf("VERIFICATION FAILED: %d command(s) failed: %s",
						len(failed), strings.Join(failed, ", ")))
				report.Notes = append(report.Notes,
					"See verification results for recovery steps")
				report.Notes = append(report.Notes,
					"Rollback: git checkout -- CLAUDE.md AGENTS.md .sdp/policies .claude/hooks")
				report.Notes = append(report.Notes,
					"Re-run with --no-verify to skip verification")

					report.DurationMs = time.Since(start).Milliseconds()
					return report, fmt.Errorf("verification failed: %d command(s) did not pass: %s",
						len(failed), strings.Join(failed, ", "))
			}
		}
	}

	// Summarize kept vs updated in notes.
	if len(report.Kept) > 0 {
		report.Notes = append(report.Notes,
			fmt.Sprintf("Kept %d unchanged artifact(s): %s",
				len(report.Kept), strings.Join(report.Kept, ", ")))
	}
	if len(report.Updated) > 0 {
		report.Notes = append(report.Notes,
			fmt.Sprintf("Updated %d artifact(s): %s",
				len(report.Updated), strings.Join(report.Updated, ", ")))
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

// trackKeptOrUpdated categorizes an artifact result into kept or updated lists.
func trackKeptOrUpdated(result *ArtifactResult, report *BootstrapReport) {
	if result.Status != "ok" {
		return
	}
	if strings.Contains(result.Message, "unchanged") || strings.Contains(result.Message, "unchanged (idempotent)") {
		report.Kept = append(report.Kept, result.Path)
	} else {
		report.Updated = append(report.Updated, result.Path)
	}
}

// Status reports the current bootstrap state of the repository.
func (p *Planner) Status() (*BootstrapStatus, error) {
	_ = p.Collector.ExistingConfig()
	avail := p.Collector.DataSourcesAvailable()

	expected := []struct {
		path string
		name string
	}{
		{"CLAUDE.md", "CLAUDE.md"},
		{"AGENTS.md", "AGENTS.md"},
		{".sdp/policies", ".sdp/policies"},
		{".claude/hooks", ".claude/hooks"},
	}

	// Only count .beads when beads is opted-in.
	if p.Config.Beads {
		expected = append(expected, struct {
			path string
			name string
		}{".beads", ".beads"})
	}

	var existingFiles, missingFiles []string
	for _, exp := range expected {
		full := filepath.Join(p.Config.RepoPath, exp.path)
		if _, err := os.Stat(full); err == nil {
			existingFiles = append(existingFiles, exp.name)
		} else {
			missingFiles = append(missingFiles, exp.name)
		}
	}

	suggestions := []string{}
	if !avail["scout"] {
		suggestions = append(suggestions, "Run 'sdp scout --output . <repo>' to generate scout.json")
	}
	if !avail["architect"] {
		suggestions = append(suggestions, "Run 'sdp architect <repo>' to generate architecture analysis")
	}
	if len(missingFiles) > 0 {
		suggestions = append(suggestions, "Run 'sdp bootstrap <repo>' to generate missing files")
	}

	// Bootstrapped requires at least CLAUDE.md + one other required artifact
	// (AGENTS.md, policies, or hooks). .beads is only counted when opted-in.
	bootstrapped := false
	var hasClaudeMD bool
	var hasOtherRequired bool
	for _, name := range existingFiles {
		switch name {
		case "CLAUDE.md":
			hasClaudeMD = true
		case "AGENTS.md", ".sdp/policies", ".claude/hooks", ".beads":
			hasOtherRequired = true
		}
	}
	bootstrapped = hasClaudeMD && hasOtherRequired

	return &BootstrapStatus{
		RepoPath:      p.Config.RepoPath,
		Bootstrapped:  bootstrapped,
		ExistingFiles: existingFiles,
		MissingFiles:  missingFiles,
		DataSources:   avail,
		Suggestions:   suggestions,
	}, nil
}

// planArtifact adds a planned artifact to the appropriate list (create/merge/skip).
func (p *Planner) planArtifact(plan *BootstrapPlan, artifactType, relPath, desc string) {
	fullPath := filepath.Join(p.Config.RepoPath, relPath)
	exists := pathExists(fullPath)

	artifact := PlannedArtifact{
		Type:        artifactType,
		Path:        relPath,
		Description: desc,
	}

	if !exists {
		artifact.Action = "create"
		plan.WillCreate = append(plan.WillCreate, artifact)
	} else if p.Config.Force {
		artifact.Action = "merge"
		plan.WillMerge = append(plan.WillMerge, artifact)
	} else {
		artifact.Action = "skip"
		plan.WillSkip = append(plan.WillSkip, artifact)
	}
}

// executeArtifact writes a single planned artifact to disk.
// Uses idempotent writes: only writes if content actually changed.
func (p *Planner) executeArtifact(a PlannedArtifact) ArtifactResult {
	result := ArtifactResult{
		Type:   a.Type,
		Path:   a.Path,
		Action: a.Action,
	}

	fullPath := filepath.Join(p.Config.RepoPath, a.Path)

	// Collect data sources for template generation.
	ds := p.Collector.CollectOptional()
	lang := ""
	if ds.Scout != nil {
		lang = ds.Scout.PrimaryLanguage
	}
	cmds := DetectBuildCommands(p.Config.RepoPath, lang)

	switch a.Type {
	case "claude_md":
		var content string
		var err error
		if a.Action == "merge" && pathExists(fullPath) {
			existingBytes, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				result.Status = "error"
				result.Message = readErr.Error()
				return result
			}
			content, err = MergeClaudeMD(string(existingBytes), p.Config.RepoPath, *ds, cmds)
		} else {
			content, err = GenerateClaudeMD(p.Config.RepoPath, *ds, cmds)
		}
		if err != nil {
			result.Status = "error"
			result.Message = err.Error()
		} else if written, werr := writeFileIdempotent(fullPath, content); werr != nil {
			result.Status = "error"
			result.Message = werr.Error()
		} else if !written {
			result.Status = "ok"
			result.Message = "CLAUDE.md unchanged (idempotent)"
		} else {
			result.Status = "ok"
			result.Message = "Generated CLAUDE.md"
		}
	case "agents_md":
		var content string
		var err error
		if a.Action == "merge" && pathExists(fullPath) {
			existingBytes, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				result.Status = "error"
				result.Message = readErr.Error()
				return result
			}
			content, err = MergeAgentsMD(string(existingBytes), p.Config.RepoPath, *ds, cmds)
		} else {
			content, err = GenerateAgentsMD(p.Config.RepoPath, *ds, cmds)
		}
		if err != nil {
			result.Status = "error"
			result.Message = err.Error()
		} else if written, werr := writeFileIdempotent(fullPath, content); werr != nil {
			result.Status = "error"
			result.Message = werr.Error()
		} else if !written {
			result.Status = "ok"
			result.Message = "AGENTS.md unchanged (idempotent)"
		} else {
			result.Status = "ok"
			result.Message = "Generated AGENTS.md"
		}
	case "policy":
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			result.Status = "error"
			result.Message = err.Error()
		} else {
			policyInput := BuildPolicyInput(ds, p.Config.RepoPath, cmds)
			policyPath := filepath.Join(fullPath, "main.rego")
			if err := GeneratePolicyToDir(policyInput, fullPath); err != nil {
				result.Status = "error"
				result.Message = err.Error()
			} else {
				result.Status = "ok"
				result.Message = fmt.Sprintf("Generated %s", policyPath)
			}
		}
	case "hook":
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			result.Status = "error"
			result.Message = err.Error()
		} else {
			hookInput := BuildHookInput(ds, cmds, p.Config.RepoPath)
			hookResults, err := GenerateHooksToDir(hookInput, fullPath)
			if err != nil {
				result.Status = "error"
				result.Message = err.Error()
			} else {
				var names []string
				for _, hr := range hookResults {
					names = append(names, hr.Name)
				}
				result.Status = "ok"
				result.Message = fmt.Sprintf("Generated hooks: %s", strings.Join(names, ", "))
			}
		}
	case "beads":
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			result.Status = "error"
			result.Message = err.Error()
		} else {
			result.Status = "ok"
			result.Message = "Created beads directory"
		}
	default:
		result.Status = "skipped"
		result.Message = fmt.Sprintf("Unknown artifact type: %s", a.Type)
	}

	return result
}

// writeFileIdempotent writes content to path only if it differs from the
// existing file content. Returns true if the file was actually written
// (content changed or file was new), false if it was already up to date.
func writeFileIdempotent(path string, content string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil {
		// File exists — compare content.
		if !ContentChanged(string(existing), content) {
			return false, nil
		}
	}

	// File is new or content changed — write it.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// computeConfidence assigns confidence scores to the plan based on data availability.
func computeConfidence(plan *BootstrapPlan) map[string]float64 {
	c := make(map[string]float64)
	ds := plan.DataSources

	// Base confidence from scout (required)
	c["overall"] = 0.3

	if ds.Scout != nil {
		c["scout"] = 1.0
		c["overall"] += 0.3
	}
	if ds.Architect != nil {
		c["architect"] = 0.9
		c["overall"] += 0.15
	}
	if ds.Metrics != nil {
		c["metrics"] = 0.85
		c["overall"] += 0.1
	}
	if ds.Spec != nil {
		c["spec"] = 0.8
		c["overall"] += 0.05
	}
	if ds.Index != nil {
		c["index"] = 0.75
		c["overall"] += 0.1
	}

	// Cap at 1.0
	if c["overall"] > 1.0 {
		c["overall"] = 1.0
	}

	return c
}

// generateNotes produces human-readable notes about the plan.
func generateNotes(plan *BootstrapPlan) []string {
	var notes []string
	ds := plan.DataSources

	if ds.Scout == nil {
		notes = append(notes, "WARNING: No scout.json found — bootstrap will use defaults")
	} else {
		notes = append(notes, fmt.Sprintf("Primary language: %s", ds.Scout.PrimaryLanguage))
		if ds.Scout.BuildSystem != "" {
			notes = append(notes, fmt.Sprintf("Build system: %s", ds.Scout.BuildSystem))
		}
	}

	if ds.Architect != nil {
		notes = append(notes, fmt.Sprintf("Architecture: %d components, %d decisions",
			len(ds.Architect.Components), len(ds.Architect.Decisions)))
	} else {
		notes = append(notes, "No architect data — policy generation will use defaults")
	}

	if ds.Metrics != nil {
		notes = append(notes, fmt.Sprintf("Health: bus_factor=%d, staleness=%s",
			ds.Metrics.BusFactor, ds.Metrics.Staleness))
	}

	if len(plan.WillCreate) > 0 {
		notes = append(notes, fmt.Sprintf("Will create %d new artifact(s)", len(plan.WillCreate)))
	}
	if len(plan.WillMerge) > 0 {
		notes = append(notes, fmt.Sprintf("Will merge %d existing artifact(s)", len(plan.WillMerge)))
	}
	if len(plan.WillSkip) > 0 {
		notes = append(notes, fmt.Sprintf("Will skip %d existing artifact(s) (use --force to overwrite)",
			len(plan.WillSkip)))
	}

	return notes
}

// onlySet converts the --only flag values into a lookup set.
// An empty set means "generate everything".
func onlySet(only []string) map[string]bool {
	m := make(map[string]bool)
	for _, o := range only {
		m[o] = true
	}
	return m
}

// shouldGenerate reports whether a given artifact type should be generated
// based on the --only filter.
func shouldGenerate(artifact string, onlyMap map[string]bool) bool {
	if len(onlyMap) == 0 {
		return true
	}
	return onlyMap[artifact]
}

// FormatPlanText renders a BootstrapPlan as human-readable text.
func FormatPlanText(plan *BootstrapPlan) string {
	out := "Bootstrap Plan\n"
	out += "==============\n\n"

	out += "Data Sources:\n"
	if plan.DataSources.Scout != nil {
		out += fmt.Sprintf("  scout:     %s (build=%s)\n", plan.DataSources.Scout.PrimaryLanguage, plan.DataSources.Scout.BuildSystem)
	} else {
		out += "  scout:     (none)\n"
	}
	out += fmt.Sprintf("  architect: %v\n", plan.DataSources.Architect != nil)
	out += fmt.Sprintf("  metrics:   %v\n", plan.DataSources.Metrics != nil)
	out += fmt.Sprintf("  spec:      %v\n", plan.DataSources.Spec != nil)
	out += fmt.Sprintf("  index:     %v\n\n", plan.DataSources.Index != nil)

	if plan.Commands.Build != "" {
		out += "Detected Commands:\n"
		out += fmt.Sprintf("  build: %s\n", plan.Commands.Build)
		out += fmt.Sprintf("  test:  %s\n", plan.Commands.Test)
		out += fmt.Sprintf("  lint:  %s\n", plan.Commands.Lint)
		if plan.Commands.Run != "" {
			out += fmt.Sprintf("  run:   %s\n", plan.Commands.Run)
		}
		out += "\n"
	}

	if len(plan.WillCreate) > 0 {
		out += "Will Create:\n"
		for _, a := range plan.WillCreate {
			out += fmt.Sprintf("  [create] %s — %s\n", a.Path, a.Description)
		}
		out += "\n"
	}

	if len(plan.WillMerge) > 0 {
		out += "Will Merge (--force):\n"
		for _, a := range plan.WillMerge {
			out += fmt.Sprintf("  [merge]  %s — %s\n", a.Path, a.Description)
		}
		out += "\n"
	}

	if len(plan.WillSkip) > 0 {
		out += "Will Skip (already exists):\n"
		for _, a := range plan.WillSkip {
			out += fmt.Sprintf("  [skip]   %s — %s\n", a.Path, a.Description)
		}
		out += "\n"
	}

	return out
}

// FormatStatusText renders a BootstrapStatus as human-readable text.
func FormatStatusText(status *BootstrapStatus) string {
	out := fmt.Sprintf("Bootstrap Status: %s\n", status.RepoPath)
	out += fmt.Sprintf("Bootstrapped: %v\n\n", status.Bootstrapped)

	if len(status.ExistingFiles) > 0 {
		out += "Existing Files:\n"
		for _, f := range status.ExistingFiles {
			out += fmt.Sprintf("  [ok]   %s\n", f)
		}
		out += "\n"
	}

	if len(status.MissingFiles) > 0 {
		out += "Missing Files:\n"
		for _, f := range status.MissingFiles {
			out += fmt.Sprintf("  [miss] %s\n", f)
		}
		out += "\n"
	}

	out += "Data Sources:\n"
	for src, found := range status.DataSources {
		mark := "no"
		if found {
			mark = "yes"
		}
		out += fmt.Sprintf("  %-12s %s\n", src+":", mark)
	}
	out += "\n"

	if len(status.Suggestions) > 0 {
		out += "Suggestions:\n"
		for _, s := range status.Suggestions {
			out += fmt.Sprintf("  - %s\n", s)
		}
	}

	return out
}

// FormatReportJSON renders a BootstrapReport as JSON.
func FormatReportJSON(report *BootstrapReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
