package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/bootstrap"
	"sdp_dev/internal/harnessadapter"
	"sdp_dev/internal/harnesscfg"
	"sdp_dev/internal/rules"
	"sdp_dev/internal/scout"
)

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "show what would be generated without writing")
	force := fs.Bool("force", false, "overwrite existing artifacts")
	noVerify := fs.Bool("no-verify", false, "skip build/test/lint verification")
	beads := fs.Bool("beads", false, "enable beads initialization (opt-in)")
	yes := fs.Bool("yes", false, "CI automation: approve final artifacts without DRAFT prefix")
	autoCurate := fs.Bool("auto-curate", false, "CI automation: bypass DRAFT prefix and produce final artifacts")
	format := fs.String("format", "text", "output format: json, text")
	onlyStr := fs.String("only", "", "generate only these artifacts (comma-separated: claude-md,agents-md,policies,hooks,beads)")
	conventions := fs.Bool("conventions", false, "extract conventions via scout and generate harness-specific rule configs")
	mode := fs.String("mode", "", "bootstrap mode: greenfield (new project) or brownfield (existing project delta)")
	preset := fs.String("preset", "", "greenfield preset (go-web-service, go-cli, go-library)")

	_ = fs.Parse(args)

	// Determine subcommand: "status" or repo path.
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp bootstrap [--dry-run] [--force] [--beads] [--yes] [--auto-curate] [--only TYPES] [--conventions] [--mode greenfield|brownfield] [--preset NAME] <repo-path>")
		fmt.Fprintln(os.Stderr, "       sdp bootstrap status <repo-path>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Modes:")
		fmt.Fprintln(os.Stderr, "  greenfield  Interactive bootstrap for new projects (--preset for non-interactive)")
		fmt.Fprintln(os.Stderr, "  brownfield  Delta analysis for existing projects (ADDED/MODIFIED/REMOVED markers)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "CI automation flags (--yes, --auto-curate) bypass DRAFT prefix for unattended runs.")
		os.Exit(2)
	}

	// Handle "status" subcommand.
	if fs.Arg(0) == "status" {
		if fs.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: sdp bootstrap status <repo-path>")
			os.Exit(2)
		}
		runBootstrapStatus(fs.Arg(1), *format)
		return
	}

	repoPath := fs.Arg(0)
	validateFormat(*format)

	// Default: UseDraft=true (DRAFT-prefixed files). CI flags bypass this.
	useDraft := !(*yes || *autoCurate)

	cfg := bootstrap.BootstrapConfig{
		RepoPath: repoPath,
		DryRun:   *dryRun,
		Force:    *force,
		NoVerify: *noVerify,
		Beads:    *beads,
		UseDraft: useDraft,
	}

	if *onlyStr != "" {
		cfg.Only = strings.Split(*onlyStr, ",")
	}

	// Early mode handling: greenfield/brownfield bypass the standard planner
	// entirely, since the planner expects an existing project with files.
	if *mode == "greenfield" || *mode == "brownfield" {
		report := &bootstrap.BootstrapReport{
			Repo:        repoPath,
			Version:     "1.0.0",
			DataSources: make(map[string]bool),
		}
		switch *mode {
		case "greenfield":
			appendGreenfieldArtifacts(report, repoPath, useDraft, *preset)
		case "brownfield":
			appendBrownfieldArtifacts(report, repoPath, useDraft)
		}
		renderBootstrapReport(report, *format)
		return
	}

	planner := bootstrap.NewPlanner(cfg)

	if *dryRun {
		report, err := planner.DryRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		renderBootstrapReport(report, *format)
		return
	}

	report, err := planner.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Optional: conventions + rules + adapter pipeline.
	if *conventions {
		appendConventionsArtifacts(report, repoPath, useDraft, *force)
	}

	renderBootstrapReport(report, *format)
}

// appendConventionsArtifacts runs the scout -> rules -> adapter pipeline and
// appends the resulting adapter outputs as DRAFT artifacts to the report.
func appendConventionsArtifacts(report *bootstrap.BootstrapReport, repoPath string, useDraft bool, force bool) {
	card, err := scout.Run(repoPath)
	if err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: "conventions", Status: "error",
			Message: fmt.Sprintf("scout failed: %v", err),
		})
		return
	}
	report.DataSources["conventions"] = true

	var generatedRules []rules.Rule
	evidenceDir := filepath.Join(repoPath, ".sdp", "evidence")
	if dirExists(evidenceDir) {
		gen := rules.NewGenerator(evidenceDir)
		generatedRules, err = gen.Generate()
		if err != nil {
			report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
				Type: "conventions", Status: "error",
				Message: fmt.Sprintf("rules generation failed: %v", err),
			})
		} else if len(generatedRules) > 0 {
			report.DataSources["evidence"] = true
			report.Notes = append(report.Notes,
				fmt.Sprintf("conventions: %d rule(s) from evidence", len(generatedRules)))
		}
	}

	manifestPath := filepath.Join(repoPath, ".sdp", "harness-config.json")
	manifest := loadManifestOrDefault("", repoPath)
	if _, serr := os.Stat(manifestPath); serr == nil {
		report.DataSources["manifest"] = true
	}

	registry := harnessadapter.NewRegistry(manifest)
	rendered, err := registry.RenderAll(card, generatedRules)
	if err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: "conventions", Status: "error",
			Message: fmt.Sprintf("adapter render failed: %v", err),
		})
		return
	}

	for adapterName, content := range rendered {
		harness := manifestHarness(manifest, adapterName)
		filename := bootstrapRulesArtifactName(harness, useDraft)
		fullPath := filepath.Join(repoPath, filename)

		var finalContent string
		if useDraft {
			finalContent = bootstrap.DraftHeader(
				time.Now().UTC().Format("2006-01-02")) + string(content)
		} else {
			finalContent = string(content)
		}

		// Preserve existing content unless --force.
		existing, readErr := os.ReadFile(fullPath)
		if readErr == nil {
			if string(existing) == finalContent {
				report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
					Type: "adapter", Path: filename,
					Action: "skip", Status: "ok",
					Message: fmt.Sprintf("conventions: %s unchanged", adapterName),
				})
				continue
			}
			if !force {
				report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
					Type: "adapter", Path: filename,
					Action: "skip", Status: "ok",
					Message: fmt.Sprintf("conventions: %s exists with different content (use --force to overwrite)", adapterName),
				})
				continue
			}
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
				Type: "adapter", Path: filename,
				Status: "error", Message: err.Error(),
			})
			continue
		}
		if err := os.WriteFile(fullPath, []byte(finalContent), 0o644); err != nil {
			report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
				Type: "adapter", Path: filename,
				Status: "error", Message: err.Error(),
			})
			continue
		}

		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type:    "adapter",
			Path:    filename,
			Action:  "create",
			Status:  "ok",
			Message: fmt.Sprintf("conventions: %s adapter (%d bytes)", adapterName, len(finalContent)),
		})
	}
}

// appendGreenfieldArtifacts runs the greenfield bootstrap flow:
// preset/interactive → principles + agents rules → split → roadmap.
// All outputs are DRAFT-prefixed and appended to the bootstrap report.
func appendGreenfieldArtifacts(report *bootstrap.BootstrapReport, repoPath string, useDraft bool, presetName string) {
	var result *bootstrap.BootstrapResult
	var err error

	if presetName != "" {
		result, err = bootstrap.RunGreenfieldFromPreset(presetName)
		if err != nil {
			report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
				Type: "greenfield", Status: "error",
				Message: fmt.Sprintf("preset %q: %v", presetName, err),
			})
			return
		}
	} else {
		// Default: use go-web-service preset for non-interactive mode.
		result, err = bootstrap.RunGreenfieldFromPreset("go-web-service")
		if err != nil {
			report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
				Type: "greenfield", Status: "error",
				Message: fmt.Sprintf("greenfield: %v", err),
			})
			return
		}
		report.Notes = append(report.Notes, "greenfield: using default preset go-web-service (use --preset to change)")
	}

	// Split principles from rules.
	split := bootstrap.SplitContent(result.PrinciplesContent + "\n" + result.AgentsContent)

	// Write DRAFT-PRINCIPLES.md.
	principlesContent := bootstrap.RenderPrinciplesFile(split.Principles)
	writeDraftArtifact(report, repoPath, "PRINCIPLES.md", principlesContent, useDraft, "greenfield-principles")

	// Write DRAFT-AGENTS.md rules section.
	agentsContent := bootstrap.RenderRulesSection(split.Rules)
	writeDraftArtifact(report, repoPath, "AGENTS-RULES.md", agentsContent, useDraft, "greenfield-rules")

	// Write DRAFT-ROADMAP.md from scout data.
	card, scoutErr := scout.Run(repoPath)
	if scoutErr != nil {
		report.Notes = append(report.Notes, fmt.Sprintf("greenfield: scout failed, skipping roadmap: %v", scoutErr))
		return
	}
	roadmap := bootstrap.GenerateRoadmap(card)
	roadmapContent := bootstrap.RenderRoadmapMarkdown(roadmap)
	writeDraftArtifact(report, repoPath, "ROADMAP.md", roadmapContent, useDraft, "greenfield-roadmap")

	// Save bootstrap answers for reproducibility.
	// Resolve the actual preset name used (may differ from CLI arg when default is applied).
	resolvedPreset := presetName
	if resolvedPreset == "" {
		resolvedPreset = "go-web-service"
	}
	var presetCfg bootstrap.GreenfieldConfig
	if pc, ok := bootstrap.Presets[resolvedPreset]; ok {
		presetCfg = pc
	}
	answersPath := filepath.Join(repoPath, ".sdp", "bootstrap-answers.json")
	if cfgBytes, jsonErr := bootstrap.MarshalAnswers(presetCfg); jsonErr == nil {
		if mkdirErr := os.MkdirAll(filepath.Dir(answersPath), 0o755); mkdirErr == nil {
			if writeErr := os.WriteFile(answersPath, cfgBytes, 0o644); writeErr != nil {
				report.Notes = append(report.Notes, fmt.Sprintf("greenfield: failed to save answers: %v", writeErr))
			}
		}
	}
}

// appendBrownfieldArtifacts runs delta analysis: scout → compare with existing → report deltas.
func appendBrownfieldArtifacts(report *bootstrap.BootstrapReport, repoPath string, useDraft bool) {
	card, err := scout.Run(repoPath)
	if err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: "brownfield", Status: "error",
			Message: fmt.Sprintf("scout failed: %v", err),
		})
		return
	}

	// Read existing rules from AGENTS.md or similar.
	existingRules := readExistingRules(repoPath)
	result := bootstrap.RunBrownfield(card, existingRules)

	if len(result.Deltas) == 0 {
		report.Notes = append(report.Notes, "brownfield: no deltas detected — project rules are up to date")
		return
	}

	// Write delta report.
	deltaJSON, err := bootstrap.MarshalBrownfieldResult(result)
	if err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: "brownfield", Status: "error",
			Message: fmt.Sprintf("marshal delta: %v", err),
		})
		return
	}

	deltaPath := "bootstrap-delta.json"
	if useDraft {
		deltaPath = bootstrap.DraftPath(deltaPath)
	}
	writeDraftArtifact(report, repoPath, deltaPath, string(deltaJSON), false, "brownfield-delta")

	for _, d := range result.Deltas {
		report.Notes = append(report.Notes,
			fmt.Sprintf("brownfield: %s %s", d.ChangeType, d.Section))
	}
}

// readExistingRules reads existing rule files from the repo for brownfield comparison.
// Reads all enabled harness config files from the manifest, falling back to
// AGENTS.md and CLAUDE.md when no manifest is available.
func readExistingRules(repoPath string) map[string]string {
	existing := make(map[string]string)

	// Determine which files to read from manifest.
	manifest := loadManifestOrDefault("", repoPath)
	var files []string
	for _, h := range manifest.Harnesses {
		if h.ConfigFile != "" {
			files = append(files, h.ConfigFile)
		}
	}
	// Fallback: read conventional files if no manifest harnesses found.
	if len(files) == 0 {
		files = []string{"AGENTS.md", "CLAUDE.md"}
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(repoPath, f))
		if err != nil {
			continue
		}
		// Simple section extraction: split by ## headers.
		prefix := f + ":"
		section := prefix + "header"
		var content strings.Builder
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "## ") {
				existing[section] = content.String()
				section = prefix + strings.TrimSpace(strings.TrimPrefix(line, "## "))
				content.Reset()
			} else {
				content.WriteString(line)
				content.WriteByte('\n')
			}
		}
		existing[section] = content.String()
	}
	return existing
}

// writeDraftArtifact writes a single DRAFT artifact and appends to the report.
func writeDraftArtifact(report *bootstrap.BootstrapReport, repoPath, basename, content string, useDraft bool, artifactType string) {
	var filename string
	if useDraft {
		filename = bootstrap.DraftPath(basename)
	} else {
		filename = basename
	}
	fullPath := filepath.Join(repoPath, filename)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: artifactType, Path: filename,
			Status: "error", Message: err.Error(),
		})
		return
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
			Type: artifactType, Path: filename,
			Status: "error", Message: err.Error(),
		})
		return
	}

	report.Artifacts = append(report.Artifacts, bootstrap.ArtifactResult{
		Type: artifactType, Path: filename,
		Action: "create", Status: "ok",
		Message: fmt.Sprintf("%d bytes", len(content)),
	})
}
// bootstrapRulesArtifactName derives a rules-specific filename from the manifest
// harness ConfigFile. Handles dotfiles like .cursorrules correctly.
// When useDraft is true, files get a DRAFT- prefix.
func bootstrapRulesArtifactName(h *harnesscfg.Harness, useDraft bool) string {
	var base string
	if h != nil && h.ConfigFile != "" {
		ext := filepath.Ext(h.ConfigFile)
		nameWithoutExt := strings.TrimSuffix(h.ConfigFile, ext)
		if nameWithoutExt == "" {
			// Dotfile like .cursorrules: strip dot, append -rules.md
			b := strings.TrimPrefix(filepath.Base(h.ConfigFile), ".")
			dir := filepath.Dir(h.ConfigFile)
			if dir == "." {
				base = b + "-rules.md"
			} else {
				base = dir + "/" + b + "-rules.md"
			}
		} else {
			base = nameWithoutExt + "-rules" + ext
		}
	} else {
		base = "harness-rules.md"
	}
	if useDraft {
		return bootstrap.DraftPath(base)
	}
	return base
}

func runBootstrapStatus(repoPath string, format string) {
	cfg := bootstrap.BootstrapConfig{RepoPath: repoPath}
	planner := bootstrap.NewPlanner(cfg)
	status, err := planner.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "json":
		out, jerr := json.MarshalIndent(status, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	default:
		fmt.Print(bootstrap.FormatStatusText(status))
	}
}

func renderBootstrapReport(report *bootstrap.BootstrapReport, format string) {
	switch format {
	case "json":
		out, err := bootstrap.FormatReportJSON(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)
	default:
		// Text format: show notes then artifacts.
		fmt.Fprintf(os.Stdout, "Bootstrap Report — %s\n", report.Repo)
		fmt.Fprintf(os.Stdout, "Version: %s | Duration: %dms\n\n", report.Version, report.DurationMs)

		if len(report.Notes) > 0 {
			for _, n := range report.Notes {
				fmt.Fprintf(os.Stdout, "  %s\n", n)
			}
			fmt.Fprintln(os.Stdout)
		}

		for _, a := range report.Artifacts {
			mark := "[ok]"
			switch a.Status {
			case "dry_run":
				mark = "[plan]"
			case "skipped":
				mark = "[skip]"
			case "error":
				mark = "[err]"
			}
			fmt.Fprintf(os.Stdout, "  %s %-20s %s\n", mark, a.Path, a.Message)
		}

		// Data sources summary.
		fmt.Fprintln(os.Stdout, "\nData Sources:")
		for src, found := range report.DataSources {
			label := "no"
			if found {
				label = "yes"
			}
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", src+":", label)
		}
	}
}

func validateFormat(format string) {
	switch format {
	case "json", "text":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", format)
		os.Exit(2)
	}
}
