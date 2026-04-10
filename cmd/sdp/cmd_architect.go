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
	timeoutFlag := fs.Duration("timeout", 5*time.Minute, "total session timeout")
	outputFlag := fs.String("output", "", "output file path (default: stdout)")
	verboseFlag := fs.Bool("verbose", false, "show per-extractor timing")
	fs.BoolVar(verboseFlag, "v", false, "shorthand for --verbose")
	skipGit := fs.Bool("skip-git", false, "skip git history analysis")
	langFilter := fs.String("language", "", "comma-separated language filter (e.g. go,python)")

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

	// Write output
	output := formatAnalyzeResult(result, diagrams, *formatFlag)
	if *outputFlag != "" {
		if err := os.WriteFile(*outputFlag, []byte(output), 0644); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", *outputFlag)
	} else {
		fmt.Println(output)
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

// --- Helper functions ---

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
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Analyze flags:")
	fmt.Fprintln(os.Stderr, "  --allow-external-llm          Enable cloud LLM enrichment")
	fmt.Fprintln(os.Stderr, "  --no-llm                      Disable all LLM calls")
	fmt.Fprintln(os.Stderr, "  --tier <1|2|3>                Analysis depth (default: 2)")
	fmt.Fprintln(os.Stderr, "  --extractors <list>           Comma-separated extractor names (default: all)")
	fmt.Fprintln(os.Stderr, "  --format <json|text|mermaid>  Output format (default: json)")
	fmt.Fprintln(os.Stderr, "  --timeout <duration>          Total session timeout (default: 5m)")
	fmt.Fprintln(os.Stderr, "  -o, --output <path>           Output file path (default: stdout)")
	fmt.Fprintln(os.Stderr, "  -v, --verbose                 Show per-extractor timing")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "C4 flags:")
	fmt.Fprintln(os.Stderr, "  --level <1|2|3>               C4 diagram level (default: all)")
	fmt.Fprintln(os.Stderr, "  -o, --output <dir>            Output directory for .mmd files")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Eval flags:")
	fmt.Fprintln(os.Stderr, "  --ground-truth <file>         Path to ground truth JSON (required)")
	fmt.Fprintln(os.Stderr, "  --format <json|text>          Output format (default: text)")
}
