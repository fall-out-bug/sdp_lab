package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/c4"
	"sdp_dev/internal/architect/eval"
	"sdp_dev/internal/architect/extract"
)

func runArchitect(args []string) {
	if len(args) < 1 {
		architectUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "analyze":
		runArchitectAnalyze(args[1:])
	case "c4":
		runArchitectC4(args[1:])
	case "eval":
		runArchitectEval(args[1:])
	case "render":
		runArchitectRender(args[1:])
	case "contracts":
		fmt.Fprintln(os.Stderr, "sdp architect contracts: not implemented yet (Phase B)")
	case "conform":
		fmt.Fprintln(os.Stderr, "sdp architect conform: not implemented yet (Phase C)")
	case "greenfield":
		fmt.Fprintln(os.Stderr, "sdp architect greenfield: not implemented yet (Phase C)")
	default:
		architectUsage()
		os.Exit(2)
	}
}

// --- analyze subcommand ---

func runArchitectAnalyze(args []string) {
	fs := flag.NewFlagSet("architect analyze", flag.ExitOnError)
	allowExtLLM := fs.Bool("allow-external-llm", false, "allow sending sanitized data to cloud LLMs")
	noLLM := fs.Bool("no-llm", false, "disable all LLM enrichment")
	tierFlag := fs.Int("tier", 2, "analysis depth: 1 (system), 2 (container), 3 (component)")
	extractorsFlag := fs.String("extractors", "", "comma-separated list of extractors (default: all)")
	formatFlag := fs.String("format", "json", "output format: json, text, mermaid")
	sectionFlag := fs.String("section", "", "output only specific section: profile, report, model, diagrams, summary")
	timeoutFlag := fs.Duration("timeout", 5*time.Minute, "total session timeout")
	outputFlag := fs.String("output", "", "output file path (default: stdout)")
	verboseFlag := fs.Bool("verbose", false, "show per-extractor timing")
	fs.BoolVar(verboseFlag, "v", false, "shorthand for --verbose")
	skipGit := fs.Bool("skip-git", false, "skip git history analysis")
	langFilter := fs.String("language", "", "comma-separated language filter (e.g. go,python)")
	writeArtifacts := fs.Bool("write-artifacts", false, "write .sdp/architecture/ artifact files")

	// Reorder args: move flags before positional args so flag.FlagSet
	// doesn't stop parsing at the first non-flag argument.
	args = reorderFlags(args)

	if err := fs.Parse(args); err != nil {
		log.Fatalf("flag parse error: %v", err)
	}

	repoPath := fs.Arg(0)
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect analyze [flags] <repo-path>")
		fs.PrintDefaults()
		os.Exit(2)
	}

	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatalf("failed to resolve repo path: %v", err)
	}
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		log.Fatalf("repo path does not exist: %s", repoRoot)
	}

	// Build extractor list
	extractors := extract.DefaultExtractors()
	if *skipGit {
		filtered := make([]architect.Extractor, 0, len(extractors))
		for _, ext := range extractors {
			if ext.Name() != "git_history" {
				filtered = append(filtered, ext)
			}
		}
		extractors = filtered
	}
	if *langFilter != "" {
		languages := strings.Split(*langFilter, ",")
		for i, lang := range languages {
			languages[i] = strings.TrimSpace(strings.ToLower(lang))
		}
		extractors = filterExtractorsByLanguage(extractors, languages)
	}
	if *extractorsFlag != "" {
		names := strings.Split(*extractorsFlag, ",")
		for i, n := range names {
			names[i] = strings.TrimSpace(n)
		}
		extractors = filterExtractorsByName(extractors, names)
	}

	tier := architect.TierLevel(*tierFlag)
	if tier < architect.Tier1 || tier > architect.Tier3 {
		log.Fatalf("invalid tier: %d (must be 1, 2, or 3)", *tierFlag)
	}

	config := architect.PipelineConfig{
		RepoRoot:         repoRoot,
		Tier:             tier,
		AllowExternalLLM: *allowExtLLM,
		NoLLM:            *noLLM,
		Timeout:          *timeoutFlag,
		Format:           *formatFlag,
		C4Level:          *tierFlag,
		Verbose:          *verboseFlag,
	}

	// Create pipeline with progress reporting
	reporter := architect.NewProgressReporter(*verboseFlag)
	pipeline := architect.NewPipeline(config, extractors)
	pipeline.SetProgressCallback(reporter.Callback())

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}

	// Generate C4 diagrams in the CLI layer (avoids import cycle)
	var diagrams []*c4.DiagramResult
	if result.ReferenceModel != nil {
		reporter.Report("Generating C4 diagrams...")
		diagrams = generateC4Diagrams(result.ReferenceModel, *tierFlag)
		reporter.Report(fmt.Sprintf("Generated %d diagrams", len(diagrams)))
	}

	// Write output — apply section filter if requested
	var output string
	if *sectionFlag != "" {
		output = formatSection(result, diagrams, *sectionFlag, *formatFlag)
	} else {
		output = formatAnalyzeResult(result, diagrams, *formatFlag)
	}
	if *outputFlag != "" {
		if err := os.WriteFile(*outputFlag, []byte(output), 0644); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", *outputFlag)
	} else {
		fmt.Println(output)
	}

	// Write artifact files only when --write-artifacts is explicitly set.
	// --output alone should only write to the specified file, not mutate the repo.
	if *writeArtifacts {
		if err := writeArtifactFiles(repoRoot, result, diagrams); err != nil {
			log.Printf("Warning: failed to write artifact files: %v", err)
		}
	}

	// Print verbose summary to stderr
	if *verboseFlag {
		reporter.Summary()
	}
}

// --- c4 subcommand ---

func runArchitectC4(args []string) {
	fs := flag.NewFlagSet("architect c4", flag.ExitOnError)
	levelFlag := fs.Int("level", 0, "C4 diagram level: 1 (system), 2 (container), 3 (component). Default: all")
	outputFlag := fs.String("output", "", "output directory for .mmd files (default: stdout)")
	extractorsFlag := fs.String("extractors", "", "comma-separated list of extractors (default: all)")
	timeoutFlag := fs.Duration("timeout", 5*time.Minute, "total session timeout")
	verboseFlag := fs.Bool("verbose", false, "show detailed output")
	fs.BoolVar(verboseFlag, "v", false, "shorthand for --verbose")
	formatFlag := fs.String("format", "mermaid", "output format: mermaid, json")

	args = reorderFlags(args)

	if err := fs.Parse(args); err != nil {
		log.Fatalf("flag parse error: %v", err)
	}

	repoPath := fs.Arg(0)
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect c4 [flags] <repo-path>")
		fs.PrintDefaults()
		os.Exit(2)
	}

	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatalf("failed to resolve repo path: %v", err)
	}
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		log.Fatalf("repo path does not exist: %s", repoRoot)
	}

	extractors := extract.DefaultExtractors()
	if *extractorsFlag != "" {
		names := strings.Split(*extractorsFlag, ",")
		for i, n := range names {
			names[i] = strings.TrimSpace(n)
		}
		extractors = filterExtractorsByName(extractors, names)
	}

	c4Level := *levelFlag
	if c4Level < 0 || c4Level > 3 {
		log.Fatalf("invalid C4 level: %d (must be 0 for all, or 1, 2, or 3)", c4Level)
	}

	config := architect.PipelineConfig{
		RepoRoot: repoRoot,
		Tier:     architect.Tier2,
		NoLLM:    true, // C4 generation does not need LLM
		Timeout:  *timeoutFlag,
		C4Level:  c4Level,
		Verbose:  *verboseFlag,
	}

	reporter := architect.NewProgressReporter(*verboseFlag)
	pipeline := architect.NewPipeline(config, extractors)
	pipeline.SetProgressCallback(reporter.Callback())

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}

	if result.ReferenceModel == nil {
		fmt.Fprintln(os.Stderr, "No reference model could be generated (insufficient data)")
		os.Exit(0)
	}

	diagrams := generateC4Diagrams(result.ReferenceModel, c4Level)
	if len(diagrams) == 0 {
		fmt.Fprintln(os.Stderr, "No C4 diagrams could be generated (insufficient data)")
		os.Exit(0)
	}

	if *outputFlag != "" {
		// Write individual .mmd files
		if err := os.MkdirAll(*outputFlag, 0755); err != nil {
			log.Fatalf("failed to create output directory: %v", err)
		}
		for _, d := range diagrams {
			filename := fmt.Sprintf("c4-%s.mmd", d.Level)
			if d.Level == c4.Level3 {
				filename = fmt.Sprintf("c4-L3-component-%d.mmd", d.NodeCount)
			}
			path := filepath.Join(*outputFlag, filename)
			if err := os.WriteFile(path, []byte(d.MermaidCode), 0644); err != nil {
				log.Printf("failed to write %s: %v", path, err)
			} else if *verboseFlag {
				fmt.Fprintf(os.Stderr, "  wrote %s (%d nodes, %d edges)\n", path, d.NodeCount, d.EdgeCount)
			}
		}
		fmt.Fprintf(os.Stderr, "C4 diagrams written to %s\n", *outputFlag)
	} else {
		// Output to stdout
		switch *formatFlag {
		case "json":
			type diagramOutput struct {
				Level       string `json:"level"`
				MermaidCode string `json:"mermaid_code"`
				NodeCount   int    `json:"node_count"`
				EdgeCount   int    `json:"edge_count"`
			}
			output := make([]diagramOutput, 0, len(diagrams))
			for _, d := range diagrams {
				output = append(output, diagramOutput{
					Level:       d.Level.String(),
					MermaidCode: d.MermaidCode,
					NodeCount:   d.NodeCount,
					EdgeCount:   d.EdgeCount,
				})
			}
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
		default:
			// Mermaid output (default for c4 command)
			for _, d := range diagrams {
				fmt.Printf("%% %s\n%s\n\n", d.Level, d.MermaidCode)
			}
		}
	}
}

// --- eval subcommand ---

func runArchitectEval(args []string) {
	fs := flag.NewFlagSet("architect eval", flag.ExitOnError)
	groundTruthFlag := fs.String("ground-truth", "", "path to ground truth JSON file (required)")
	outputFlag := fs.String("output", "", "output file path (default: stdout)")
	fs.StringVar(outputFlag, "o", "", "shorthand for --output")
	formatFlag := fs.String("format", "text", "output format: json, text")
	timeoutFlag := fs.Duration("timeout", 5*time.Minute, "total session timeout")
	verboseFlag := fs.Bool("verbose", false, "show per-extractor timing")
	fs.BoolVar(verboseFlag, "v", false, "shorthand for --verbose")

	args = reorderFlags(args)

	if err := fs.Parse(args); err != nil {
		log.Fatalf("flag parse error: %v", err)
	}

	repoPath := fs.Arg(0)
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect eval <repo-path> --ground-truth <file>")
		fs.PrintDefaults()
		os.Exit(2)
	}
	if *groundTruthFlag == "" {
		fmt.Fprintln(os.Stderr, "error: --ground-truth flag is required")
		fs.PrintDefaults()
		os.Exit(2)
	}

	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatalf("failed to resolve repo path: %v", err)
	}
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		log.Fatalf("repo path does not exist: %s", repoRoot)
	}

	// Load ground truth
	gtData, err := os.ReadFile(*groundTruthFlag)
	if err != nil {
		log.Fatalf("failed to read ground truth file: %v", err)
	}

	var groundTruth eval.GroundTruth
	if err := json.Unmarshal(gtData, &groundTruth); err != nil {
		log.Fatalf("failed to parse ground truth JSON: %v", err)
	}

	// Run extractors to produce actual fragment
	extractors := extract.DefaultExtractors()
	config := architect.PipelineConfig{
		RepoRoot: repoRoot,
		Tier:     architect.Tier2,
		NoLLM:    true, // eval does not need LLM
		Timeout:  *timeoutFlag,
		Verbose:  *verboseFlag,
	}

	reporter := architect.NewProgressReporter(*verboseFlag)
	pipeline := architect.NewPipeline(config, extractors)
	pipeline.SetProgressCallback(reporter.Callback())

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		log.Fatalf("pipeline failed: %v", err)
	}

	// Build a ProfileFragment from the assembled profile for evaluation
	fragment := profileToFragment(result.Profile)

	// Run evaluation
	harness := eval.NewHarness([]eval.GroundTruth{groundTruth})
	evalResult, err := harness.Evaluate(groundTruth.RepoName, "pipeline", fragment)
	if err != nil {
		log.Fatalf("evaluation failed: %v", err)
	}

	// Format output
	var output string
	switch *formatFlag {
	case "json":
		data, err := json.MarshalIndent(evalResult, "", "  ")
		if err != nil {
			log.Fatalf("failed to marshal eval result: %v", err)
		}
		output = string(data)
	default:
		output = eval.FormatReport(evalResult)
	}

	if *outputFlag != "" {
		if err := os.WriteFile(*outputFlag, []byte(output), 0644); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Evaluation report written to %s\n", *outputFlag)
	} else {
		fmt.Println(output)
	}

	// Exit with non-zero if F1 is below threshold
	if evalResult.OverallF1() < 0.5 {
		fmt.Fprintf(os.Stderr, "Warning: overall F1 score (%.3f) is below 0.5 threshold\n", evalResult.OverallF1())
	}
}

// --- C4 diagram generation (in CLI layer to avoid import cycle) ---

func generateC4Diagrams(model *architect.ReferenceModel, level int) []*c4.DiagramResult {
	opts := c4.RenderOptions{
		Direction: "TB",
		Theme:     "default",
	}

	var diagrams []*c4.DiagramResult

	if level == 0 || level >= 1 {
		l1, err := c4.RenderL1(model, opts)
		if err == nil {
			diagrams = append(diagrams, l1)
		}
	}

	if level == 0 || level >= 2 {
		l2, err := c4.RenderL2(model, opts)
		if err == nil {
			diagrams = append(diagrams, l2)
		}
	}

	if level == 0 || level >= 3 {
		for _, container := range model.Containers {
			l3, err := c4.RenderL3(model, container.ID, opts)
			if err == nil {
				diagrams = append(diagrams, l3)
			}
		}
	}

	return diagrams
}

// --- Output formatting ---

// analyzeOutput is the JSON output structure for the analyze command.
type analyzeOutput struct {
	Version        string                    `json:"version"`
	AnalyzedAt     time.Time                 `json:"analyzed_at"`
	RepoRoot       string                    `json:"repo_root"`
	DurationMs     int64                     `json:"duration_ms"`
	Profile        *architect.CodebaseProfile `json:"profile"`
	Report         *architect.ArchitectureReport `json:"report"`
	ReferenceModel *architect.ReferenceModel  `json:"reference_model,omitempty"`
	Diagrams       []diagramOutput           `json:"diagrams,omitempty"`
	Errors         []string                  `json:"errors,omitempty"`
}

type diagramOutput struct {
	Level       string `json:"level"`
	MermaidCode string `json:"mermaid_code"`
	NodeCount   int    `json:"node_count"`
	EdgeCount   int    `json:"edge_count"`
}

func formatAnalyzeResult(result *architect.PipelineResult, diagrams []*c4.DiagramResult, format string) string {
	switch format {
	case "text":
		return formatTextResult(result, diagrams)
	case "mermaid":
		return formatMermaidResult(diagrams)
	default: // json
		return formatJSONResult(result, diagrams)
	}
}

func formatJSONResult(result *architect.PipelineResult, diagrams []*c4.DiagramResult) string {
	out := analyzeOutput{
		Version:        "1.0.0",
		AnalyzedAt:     time.Now(),
		RepoRoot:       result.Report.RepoRoot,
		DurationMs:     result.Duration.Milliseconds(),
		Profile:        result.Profile,
		Report:         result.Report,
		ReferenceModel: result.ReferenceModel,
	}

	for _, d := range diagrams {
		out.Diagrams = append(out.Diagrams, diagramOutput{
			Level:       d.Level.String(),
			MermaidCode: d.MermaidCode,
			NodeCount:   d.NodeCount,
			EdgeCount:   d.EdgeCount,
		})
	}

	for _, e := range result.Errors {
		out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", e.Extractor, e.Err))
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "marshal failed: %v"}`, err)
	}
	return string(data)
}

func formatTextResult(result *architect.PipelineResult, diagrams []*c4.DiagramResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Architecture Analysis: %s\n", result.Report.RepoRoot)
	fmt.Fprintf(&b, "Duration: %s\n\n", result.Duration.Round(time.Millisecond))

	if result.Profile != nil {
		fmt.Fprintf(&b, "Files: %d  |  LOC: %d  |  Languages: %d\n",
			result.Profile.Metrics.TotalFiles, result.Profile.Metrics.TotalLOC, result.Profile.Metrics.LanguagesCount)
		fmt.Fprintf(&b, "Containers: %d  |  Components: %d  |  Specs: %d\n",
			result.Profile.Metrics.ContainersDetected, result.Profile.Metrics.ComponentsDetected,
			result.Profile.Metrics.ContractsDiscovered)
	}

	if result.ReferenceModel != nil {
		fmt.Fprintf(&b, "\nC4 Model: %d containers, %d relationships\n",
			len(result.ReferenceModel.Containers), len(result.ReferenceModel.Relationships))
		for _, c := range result.ReferenceModel.Containers {
			fmt.Fprintf(&b, "  - %s (%s)\n", c.Name, c.Technology)
		}
	}

	if len(diagrams) > 0 {
		fmt.Fprintf(&b, "\nDiagrams: %d generated\n", len(diagrams))
		for _, d := range diagrams {
			fmt.Fprintf(&b, "  - %s: %d nodes, %d edges\n", d.Level, d.NodeCount, d.EdgeCount)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(&b, "\nWarnings: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(&b, "  - %s: %v\n", e.Extractor, e.Err)
		}
	}

	return b.String()
}

func formatMermaidResult(diagrams []*c4.DiagramResult) string {
	var parts []string
	for _, d := range diagrams {
		parts = append(parts, fmt.Sprintf("%% %s\n%s", d.Level, d.MermaidCode))
	}
	return strings.Join(parts, "\n\n")
}

// --- Section filtering ---

// formatSection outputs a single section of the analysis result.
// Supported sections: profile, report, model, diagrams, summary.
func formatSection(result *architect.PipelineResult, diagrams []*c4.DiagramResult, section, format string) string {
	switch strings.ToLower(section) {
	case "profile":
		return marshalOrText(result.Profile, format, func() string {
			return formatProfileText(result.Profile)
		})
	case "report":
		return marshalOrText(result.Report, format, func() string {
			return formatReportText(result.Report)
		})
	case "model":
		return marshalOrText(result.ReferenceModel, format, func() string {
			return formatModelText(result.ReferenceModel)
		})
	case "diagrams":
		if format == "json" {
			type d struct {
				Level     string `json:"level"`
				Mermaid   string `json:"mermaid_code"`
				NodeCount int    `json:"node_count"`
				EdgeCount int    `json:"edge_count"`
			}
			out := make([]d, 0, len(diagrams))
			for _, diag := range diagrams {
				out = append(out, d{Level: diag.Level.String(), Mermaid: diag.MermaidCode, NodeCount: diag.NodeCount, EdgeCount: diag.EdgeCount})
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			return string(data)
		}
		return formatMermaidResult(diagrams)
	case "summary":
		return formatSummaryText(result, diagrams)
	default:
		return fmt.Sprintf("unknown section %q (valid: profile, report, model, diagrams, summary)", section)
	}
}

func marshalOrText(v interface{}, format string, textFn func() string) string {
	if format == "json" {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return string(data)
	}
	return textFn()
}

func formatProfileText(p *architect.CodebaseProfile) string {
	if p == nil {
		return "(no profile)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== Codebase Profile: %s ===\n", p.Name)
	fmt.Fprintf(&b, "Files: %d  |  LOC: %d  |  Languages: %d\n", p.Metrics.TotalFiles, p.Metrics.TotalLOC, p.Metrics.LanguagesCount)
	fmt.Fprintf(&b, "Containers: %d  |  Components: %d  |  Specs: %d\n",
		p.Metrics.ContainersDetected, p.Metrics.ComponentsDetected, p.Metrics.ContractsDiscovered)

	if p.Dependencies.Language != "" {
		fmt.Fprintf(&b, "\nPrimary language: %s\n", p.Dependencies.Language)
	}
	if len(p.Dependencies.Manifests) > 0 {
		fmt.Fprintf(&b, "Manifests: %d\n", len(p.Dependencies.Manifests))
		for _, m := range p.Dependencies.Manifests {
			fmt.Fprintf(&b, "  - %s (%s, %d deps)\n", m.Path, m.Language, m.DepsCount)
		}
	}
	if len(p.Infra.Containers) > 0 {
		fmt.Fprintf(&b, "\nInfra containers: %d\n", len(p.Infra.Containers))
		for _, c := range p.Infra.Containers {
			img := c.Image
			if img == "" {
				img = c.Source
			}
			fmt.Fprintf(&b, "  - %s [%s] %s\n", c.Name, c.Type, img)
		}
	}
	if len(p.Infra.ModuleBoundaries) > 0 {
		fmt.Fprintf(&b, "\nModule boundaries:\n")
		for _, mb := range p.Infra.ModuleBoundaries {
			fmt.Fprintf(&b, "  %s (%s): %d children\n", mb.Path, mb.BuildSystem, len(mb.Children))
			for _, ch := range mb.Children {
				fmt.Fprintf(&b, "    - %s\n", ch)
			}
		}
	}
	if p.ImportGraph.Nodes > 0 {
		fmt.Fprintf(&b, "\nImport graph: %d nodes, %d edges, %d clusters\n",
			p.ImportGraph.Nodes, p.ImportGraph.Edges, len(p.ImportGraph.Clusters))
		for _, cl := range p.ImportGraph.Clusters {
			fmt.Fprintf(&b, "  cluster %s: %d pkgs, %d internal, %d external edges\n",
				cl.ID, len(cl.Packages), cl.InternalEdges, cl.ExternalEdges)
		}
	}
	return b.String()
}

func formatReportText(r *architect.ArchitectureReport) string {
	if r == nil {
		return "(no report)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== Architecture Report ===\n")
	fmt.Fprintf(&b, "Repo: %s\n", r.RepoRoot)
	fmt.Fprintf(&b, "Analyzed at: %s\n", r.AnalyzedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Duration: %.1fs\n", r.AnalysisDurationS)
	if r.Languages.Primary != "" {
		fmt.Fprintf(&b, "Primary language: %s\n", r.Languages.Primary)
	}
	if len(r.StyleHypothesis.Styles) > 0 {
		fmt.Fprintf(&b, "Styles:\n")
		for _, s := range r.StyleHypothesis.Styles {
			fmt.Fprintf(&b, "  - %s (confidence: %.2f)\n", s.Style, s.Confidence)
		}
	}
	if len(r.PatternsDetected) > 0 {
		fmt.Fprintf(&b, "\nPatterns: %d\n", len(r.PatternsDetected))
		for _, p := range r.PatternsDetected {
			fmt.Fprintf(&b, "  - %s (confidence: %.2f)\n", p.Name, p.Confidence)
		}
	}
	if len(r.Risks) > 0 {
		fmt.Fprintf(&b, "\nRisks: %d\n", len(r.Risks))
		for _, risk := range r.Risks {
			fmt.Fprintf(&b, "  [%s] %s: %s\n", risk.Severity, risk.Category, risk.Description)
		}
	}
	return b.String()
}

func formatModelText(m *architect.ReferenceModel) string {
	if m == nil {
		return "(no model)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== C4 Reference Model ===\n")
	fmt.Fprintf(&b, "System: %s\n", m.System.Name)
	fmt.Fprintf(&b, "State: %s  |  Version: %s\n", m.State, m.Version)

	if len(m.Actors) > 0 {
		fmt.Fprintf(&b, "\nActors: %d\n", len(m.Actors))
		for _, a := range m.Actors {
			fmt.Fprintf(&b, "  - %s: %s\n", a.ID, a.Description)
		}
	}
	if len(m.ExternalSystems) > 0 {
		fmt.Fprintf(&b, "\nExternal systems: %d\n", len(m.ExternalSystems))
		for _, es := range m.ExternalSystems {
			fmt.Fprintf(&b, "  - %s (%s): %s\n", es.ID, es.Technology, es.Description)
		}
	}

	fmt.Fprintf(&b, "\nContainers: %d\n", len(m.Containers))
	for _, c := range m.Containers {
		fmt.Fprintf(&b, "  [%s] %s — %s (deploy: %s)\n", c.ID, c.Name, c.Technology, c.Deploy)
		for _, comp := range c.Components {
			fmt.Fprintf(&b, "    component: %s (%s) confidence=%.2f\n", comp.ID, comp.Path, comp.Confidence)
		}
	}

	fmt.Fprintf(&b, "\nRelationships: %d\n", len(m.Relationships))
	for _, r := range m.Relationships {
		fmt.Fprintf(&b, "  %s -> %s: %s\n", r.From, r.To, r.Description)
	}
	return b.String()
}

func formatSummaryText(result *architect.PipelineResult, diagrams []*c4.DiagramResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== Architecture Summary ===\n")
	fmt.Fprintf(&b, "Duration: %s\n\n", result.Duration.Round(time.Millisecond))

	if p := result.Profile; p != nil {
		fmt.Fprintf(&b, "Codebase: %s\n", p.Name)
		fmt.Fprintf(&b, "  Files: %d  |  LOC: %d  |  Languages: %d\n",
			p.Metrics.TotalFiles, p.Metrics.TotalLOC, p.Metrics.LanguagesCount)
		if p.Dependencies.Language != "" {
			fmt.Fprintf(&b, "  Primary: %s\n", p.Dependencies.Language)
		}
		fmt.Fprintf(&b, "  Deployment: %s\n", p.Infra.DeploymentType)
		if len(p.Infra.ModuleBoundaries) > 0 {
			total := 0
			for _, mb := range p.Infra.ModuleBoundaries {
				total += len(mb.Children)
			}
			fmt.Fprintf(&b, "  Modules: %d (across %d build files)\n", total, len(p.Infra.ModuleBoundaries))
		}
		if p.ImportGraph.Nodes > 0 {
			fmt.Fprintf(&b, "  Import graph: %d nodes, %d edges, %d clusters\n",
				p.ImportGraph.Nodes, p.ImportGraph.Edges, len(p.ImportGraph.Clusters))
		}
	}

	if m := result.ReferenceModel; m != nil {
		fmt.Fprintf(&b, "\nC4 Model: %s\n", m.System.Name)
		fmt.Fprintf(&b, "  Containers: %d\n", len(m.Containers))
		for _, c := range m.Containers {
			compCount := len(c.Components)
			fmt.Fprintf(&b, "    - %s (%s) [%d components]\n", c.Name, c.Technology, compCount)
		}
		fmt.Fprintf(&b, "  Relationships: %d\n", len(m.Relationships))
		fmt.Fprintf(&b, "  Actors: %d  |  External systems: %d\n", len(m.Actors), len(m.ExternalSystems))
	}

	if len(diagrams) > 0 {
		fmt.Fprintf(&b, "\nDiagrams: %d\n", len(diagrams))
		for _, d := range diagrams {
			fmt.Fprintf(&b, "  - %s: %d nodes, %d edges\n", d.Level, d.NodeCount, d.EdgeCount)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(&b, "\nWarnings: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(&b, "  - %s: %v\n", e.Extractor, e.Err)
		}
	}
	return b.String()
}

// --- Helper functions ---

// reorderFlags moves flag arguments (starting with - or --) before positional
// arguments so that flag.FlagSet.Parse doesn't stop at the first non-flag arg.
// This allows both "sdp architect analyze --tier 3 ./repo" and
// "sdp architect analyze ./repo --tier 3" to work correctly.
func reorderFlags(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			// If this flag takes a value and the next arg is not a flag, grab it too
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Check if it's a boolean flag like --verbose, --no-llm, --skip-git, --write-artifacts
				// Boolean flags don't consume the next argument
				if !isBoolFlag(args[i]) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			positionals = append(positionals, args[i])
		}
	}
	return append(flags, positionals...)
}

// isBoolFlag returns true for flags that don't consume the next argument.
func isBoolFlag(arg string) bool {
	flags := map[string]bool{
		"--allow-external-llm": true,
		"--no-llm":            true,
		"--verbose":           true,
		"-v":                  true,
		"--skip-git":          true,
		"--write-artifacts":   true,
	}
	return flags[arg]
}

func filterExtractorsByName(extractors []architect.Extractor, names []string) []architect.Extractor {
	if len(names) == 0 {
		return extractors
	}
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}

	filtered := make([]architect.Extractor, 0, len(extractors))
	for _, ext := range extractors {
		if nameSet[strings.ToLower(ext.Name())] {
			filtered = append(filtered, ext)
		}
	}
	return filtered
}

func filterExtractorsByLanguage(extractors []architect.Extractor, languages []string) []architect.Extractor {
	if len(languages) == 0 {
		return extractors
	}

	agnosticExtractors := map[string]bool{
		"filetree": true, "deps": true, "specs": true,
		"generated": true, "infra": true, "git_history": true,
	}

	langToExtractor := map[string]string{
		"go": "go", "golang": "go",
		"java": "java", "kotlin": "java",
		"python": "python", "py": "python",
		"typescript": "TypeScript", "ts": "TypeScript",
		"javascript": "TypeScript", "js": "TypeScript",
	}

	filtered := make([]architect.Extractor, 0, len(extractors))
	for _, ext := range extractors {
		name := ext.Name()
		if agnosticExtractors[name] {
			filtered = append(filtered, ext)
			continue
		}
		for _, lang := range languages {
			if extractorName, ok := langToExtractor[strings.ToLower(lang)]; ok {
				if name == extractorName {
					filtered = append(filtered, ext)
					break
				}
			}
		}
	}
	return filtered
}

// profileToFragment converts a CodebaseProfile back to a ProfileFragment for evaluation.
func profileToFragment(profile *architect.CodebaseProfile) *architect.ProfileFragment {
	if profile == nil {
		return nil
	}
	frag := &architect.ProfileFragment{
		FileTree:    &profile.FileTree,
		Specs:       profile.Specs,
		SQLAnalysis: profile.SQLAnalysis,
		GitAnalysis: profile.GitAnalysis,
		Metrics:     &profile.Metrics,
	}

	if len(profile.Dependencies.Manifests) > 0 || len(profile.Dependencies.NotableDeps) > 0 {
		frag.Dependencies = []architect.DependencyInfo{profile.Dependencies}
	}

	if profile.ImportGraph.Nodes > 0 || profile.ImportGraph.Edges > 0 {
		frag.ImportGraph = &profile.ImportGraph
	}

	if len(profile.Infra.Containers) > 0 {
		frag.Infra = &profile.Infra
	}

	return frag
}

func architectUsage() {
	fmt.Fprintln(os.Stderr, "usage: sdp architect <subcommand> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  analyze <repo-path>           Full architecture analysis pipeline")
	fmt.Fprintln(os.Stderr, "  c4 <repo-path>                Generate C4 diagrams")
	fmt.Fprintln(os.Stderr, "  eval <repo-path>              Run evaluation against ground truth")
	fmt.Fprintln(os.Stderr, "  render <report.md>            Render markdown report to interactive HTML")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Analyze flags:")
	fmt.Fprintln(os.Stderr, "  --allow-external-llm          Enable cloud LLM enrichment")
	fmt.Fprintln(os.Stderr, "  --no-llm                      Disable all LLM calls")
	fmt.Fprintln(os.Stderr, "  --tier <1|2|3>                Analysis depth (default: 2)")
	fmt.Fprintln(os.Stderr, "  --extractors <list>           Comma-separated extractor names (default: all)")
	fmt.Fprintln(os.Stderr, "  --format <json|text|mermaid>  Output format (default: json)")
	fmt.Fprintln(os.Stderr, "  --section <name>              Output only: profile, report, model, diagrams, summary")
	fmt.Fprintln(os.Stderr, "  --timeout <duration>          Total session timeout (default: 5m)")
	fmt.Fprintln(os.Stderr, "  -o, --output <path>           Output file path (default: stdout)")
	fmt.Fprintln(os.Stderr, "  -v, --verbose                 Show per-extractor timing")
	fmt.Fprintln(os.Stderr, "  --write-artifacts             Write .sdp/architecture/ artifact files")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Render flags:")
	fmt.Fprintln(os.Stderr, "  -o, --output <path>           Output HTML path (default: same name .html)")
	fmt.Fprintln(os.Stderr, "  --open                        Open in browser after rendering")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "C4 flags:")
	fmt.Fprintln(os.Stderr, "  --level <1|2|3>               C4 diagram level (default: all)")
	fmt.Fprintln(os.Stderr, "  -o, --output <dir>            Output directory for .mmd files")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Eval flags:")
	fmt.Fprintln(os.Stderr, "  --ground-truth <file>         Path to ground truth JSON (required)")
	fmt.Fprintln(os.Stderr, "  --format <json|text>          Output format (default: text)")
}

// writeArtifactFiles writes analysis artifacts to .sdp/architecture/ directory.
func writeArtifactFiles(repoRoot string, result *architect.PipelineResult, diagrams []*c4.DiagramResult) error {
	// Create .sdp/architecture/ directory
	archDir := filepath.Join(repoRoot, ".sdp", "architecture")
	if err := os.MkdirAll(archDir, 0755); err != nil {
		return fmt.Errorf("failed to create architecture directory: %w", err)
	}

	// Write profile.json
	if result.Profile != nil {
		profileData, err := json.MarshalIndent(result.Profile, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal profile: %w", err)
		}
		profilePath := filepath.Join(archDir, "profile.json")
		if err := os.WriteFile(profilePath, profileData, 0644); err != nil {
			return fmt.Errorf("failed to write profile.json: %w", err)
		}
	}

	// Write report.json
	if result.Report != nil {
		reportData, err := json.MarshalIndent(result.Report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		reportPath := filepath.Join(archDir, "report.json")
		if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
			return fmt.Errorf("failed to write report.json: %w", err)
		}
	}

	// Write C4 diagrams to c4/ subdirectory
	if len(diagrams) > 0 {
		c4Dir := filepath.Join(archDir, "c4")
		if err := os.MkdirAll(c4Dir, 0755); err != nil {
			return fmt.Errorf("failed to create c4 directory: %w", err)
		}

		for _, diag := range diagrams {
			filename := fmt.Sprintf("c4-%s.mmd", diag.Level)
			diagPath := filepath.Join(c4Dir, filename)
			if err := os.WriteFile(diagPath, []byte(diag.MermaidCode), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", filename, err)
			}
		}
	}

	return nil
}
