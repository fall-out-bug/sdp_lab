package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/bootstrap"
	"sdp_dev/internal/harnessadapter"
	"sdp_dev/internal/harnesscfg"
	"sdp_dev/internal/rules"
	"sdp_dev/internal/scout"
)

func runRules(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp rules <update> [flags] <repo-path>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Rules commands:")
		fmt.Fprintln(os.Stderr, "  sdp rules update <repo-path> [--source-evidence=<dir>] [--manifest=<file>] [--format json|text]")
		os.Exit(2)
	}

	switch args[0] {
	case "update":
		runRulesUpdate(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown rules subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runRulesUpdate(args []string) {
	fs := flag.NewFlagSet("rules update", flag.ExitOnError)
	sourceEvidence := fs.String("source-evidence", "", "evidence directory (default: <repo>/.sdp/evidence/)")
	manifestFile := fs.String("manifest", "", "harness config manifest JSON file (default: <repo>/.sdp/harness-config.json)")
	format := fs.String("format", "text", "output format: json, text")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp rules update <repo-path> [--source-evidence=<dir>] [--manifest=<file>]")
		os.Exit(2)
	}

	validateFormat(*format)
	repoPath := fs.Arg(0)

	if _, err := os.Stat(repoPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	evidenceDir := *sourceEvidence
	if evidenceDir == "" {
		evidenceDir = filepath.Join(repoPath, ".sdp", "evidence")
	}

	report, err := executeRulesUpdate(repoPath, evidenceDir, *manifestFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	renderRulesReport(report, *format)
}

// rulesReport captures the result of a rules update run.
type rulesReport struct {
	Repo        string              `json:"repo"`
	Version     string              `json:"version"`
	GeneratedAt time.Time           `json:"generated_at"`
	DurationMs  int64               `json:"duration_ms"`
	RulesFound  int                 `json:"rules_found"`
	Adapters    map[string]adapterResult `json:"adapters"`
	Sources     map[string]bool     `json:"sources"`
	Notes       []string            `json:"notes,omitempty"`
}

// adapterResult records the output for a single adapter.
type adapterResult struct {
	File    string `json:"file"`
	Size    int    `json:"size"`
	Message string `json:"message"`
}

func executeRulesUpdate(repoPath, evidenceDir, manifestFile string) (*rulesReport, error) {
	start := time.Now()
	report := &rulesReport{
		Repo:        repoPath,
		Version:     "1.0.0",
		GeneratedAt: start.UTC(),
		Sources:     make(map[string]bool),
		Adapters:    make(map[string]adapterResult),
	}

	// Step 1: Run scout.
	card, err := scout.Run(repoPath)
	if err != nil {
		return nil, fmt.Errorf("scout failed: %w", err)
	}
	report.Sources["scout"] = true

	// Step 2: Generate rules from evidence (optional).
	var generatedRules []rules.Rule
	if dirExists(evidenceDir) {
		gen := rules.NewGenerator(evidenceDir)
		generatedRules, err = gen.Generate()
		if err != nil {
			return nil, fmt.Errorf("rule generation failed: %w", err)
		}
		report.Sources["evidence"] = true
		report.RulesFound = len(generatedRules)
		if len(generatedRules) > 0 {
			report.Notes = append(report.Notes,
				fmt.Sprintf("Generated %d rule(s) from evidence", len(generatedRules)))
		} else {
			report.Notes = append(report.Notes, "Evidence directory found but no failure patterns detected")
		}
	} else {
		report.Sources["evidence"] = false
		report.Notes = append(report.Notes, "No evidence directory found; rules generation skipped")
	}

	// Step 3: Load manifest and create adapter registry.
	manifest := loadManifestOrDefault(manifestFile, repoPath)
	registry := harnessadapter.NewRegistry(manifest)

	// Step 4: Render via adapters and write DRAFT files.
	rendered, err := registry.RenderAll(card, generatedRules)
	if err != nil {
		return nil, fmt.Errorf("adapter render failed: %w", err)
	}

	if len(rendered) == 0 {
		report.Notes = append(report.Notes, "No adapters registered; no output files written")
		report.DurationMs = time.Since(start).Milliseconds()
		return report, nil
	}

	for adapterName, content := range rendered {
		filename := draftFilename(adapterName)
		fullPath := filepath.Join(repoPath, filename)

		header := bootstrap.DraftHeader(start.UTC().Format("2006-01-02"))
		finalContent := header + string(content)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", filename, err)
		}
		if err := os.WriteFile(fullPath, []byte(finalContent), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", filename, err)
		}

		report.Adapters[adapterName] = adapterResult{
			File:    filename,
			Size:    len(finalContent),
			Message: "Generated DRAFT file",
		}
	}

	report.DurationMs = time.Since(start).Milliseconds()
	return report, nil
}

// draftFilename maps a harness name to a DRAFT-prefixed output filename.
func draftFilename(harnessName string) string {
	switch harnessName {
	case "claude-code":
		return "DRAFT-CLAUDE-RULES.md"
	case "cursor":
		return "DRAFT-.cursorrules"
	default:
		return "DRAFT-" + harnessName + "-rules.md"
	}
}

// loadManifestOrDefault loads a manifest from the given file path, or creates
// a default manifest for the repo. Returns nil-safe default on any error.
func loadManifestOrDefault(manifestFile, repoPath string) *harnesscfg.Manifest {
	path := manifestFile
	if path == "" {
		path = filepath.Join(repoPath, ".sdp", "harness-config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultManifest()
	}

	var m harnesscfg.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Fprintf(os.Stderr, "warning: manifest parse error: %v; using defaults\n", err)
		return defaultManifest()
	}
	return &m
}

// defaultManifest creates a manifest with claude-code enabled as a sensible default.
func defaultManifest() *harnesscfg.Manifest {
	return &harnesscfg.Manifest{
		Version:        "0.1.0",
		LifecycleStage: harnesscfg.StageGreenfieldStr,
		Harnesses: []harnesscfg.Harness{
			{Name: "claude-code", ConfigFile: "CLAUDE.md"},
		},
	}
}

// dirExists reports whether the path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// renderRulesReport outputs the rules report in the requested format.
func renderRulesReport(report *rulesReport, format string) {
	switch format {
	case "json":
		out, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	default:
		fmt.Fprintf(os.Stdout, "Rules Update Report — %s\n", report.Repo)
		fmt.Fprintf(os.Stdout, "Version: %s | Duration: %dms\n\n", report.Version, report.DurationMs)

		for _, n := range report.Notes {
			fmt.Fprintf(os.Stdout, "  %s\n", n)
		}
		if len(report.Notes) > 0 {
			fmt.Fprintln(os.Stdout)
		}

		fmt.Fprintln(os.Stdout, "Data Sources:")
		for src, found := range report.Sources {
			label := "no"
			if found {
				label = "yes"
			}
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", src+":", label)
		}

		if len(report.Adapters) > 0 {
			fmt.Fprintln(os.Stdout, "\nGenerated Files:")
			for name, ar := range report.Adapters {
				fmt.Fprintf(os.Stdout, "  [ok] %-20s %s (%d bytes)\n", ar.File, name, ar.Size)
			}
		}
	}
}
