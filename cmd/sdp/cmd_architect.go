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
	"sdp_dev/internal/architect/classify"
	"sdp_dev/internal/architect/extract"
	"sdp_dev/internal/discovery"
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
		fmt.Println("sdp architect c4: not implemented yet")
	case "contracts":
		fmt.Println("sdp architect contracts: not implemented yet")
	case "conform":
		fmt.Println("sdp architect conform: not implemented yet")
	case "greenfield":
		fmt.Println("sdp architect greenfield: not implemented yet")
	default:
		architectUsage()
		os.Exit(2)
	}
}

func runArchitectAnalyze(args []string) {
	fs := flag.NewFlagSet("architect analyze", flag.ExitOnError)
	allowExtLLM := fs.Bool("allow-external-llm", false, "allow sending sanitized data to cloud LLMs")
	skipGit := fs.Bool("skip-git", false, "skip git history analysis")
	langFilter := fs.String("language", "", "comma-separated language filter (e.g. go,python)")
	outputDir := fs.String("output-dir", "", "output directory (default: <repo>/.sdp/architecture)")
	tierFlag := fs.Int("tier", 2, "analysis depth: 1, 2, or 3")
	model := fs.String("model", "google/gemini-2.0-flash-001", "LLM model")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("flag parse error: %v", err)
	}

	repoPath := fs.Arg(0)
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect analyze [flags] <repo-path>")
		fs.PrintDefaults()
		os.Exit(2)
	}

	// 1. Resolve repo path
	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		log.Fatalf("failed to resolve repo path: %v", err)
	}

	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		log.Fatalf("repo path does not exist: %s", repoRoot)
	}

	fmt.Printf("AI Architect: analyzing %s\n", repoRoot)

	// 2. Create SecurityFilter
	secFilter := architect.NewSecurityFilter()
	secFilter.AllowExternalLLM = *allowExtLLM

	// 3. Build extractor list
	extractors := extract.DefaultExtractors()

	// Add GitHistoryExtractor conditionally
	if *skipGit {
		fmt.Println("  skipping git history analysis")
		// Remove git_history extractor if present
		filtered := make([]architect.Extractor, 0, len(extractors))
		for _, ext := range extractors {
			if ext.Name() != "git_history" {
				filtered = append(filtered, ext)
			}
		}
		extractors = filtered
	}

	// Filter by language if specified
	if *langFilter != "" {
		languages := strings.Split(*langFilter, ",")
		for i, lang := range languages {
			languages[i] = strings.TrimSpace(strings.ToLower(lang))
		}
		extractors = filterExtractors(extractors, languages)
		fmt.Printf("  language filter: %s\n", languages)
	}

	// 4. Create ProfileAssembler with tier
	tier := architect.TierLevel(*tierFlag)
	if tier < architect.Tier1 || tier > architect.Tier3 {
		log.Fatalf("invalid tier: %d (must be 1, 2, or 3)", *tierFlag)
	}

	assembler := architect.NewProfileAssembler(tier, extractors...)

	// 5. Assemble profile
	ctx := context.Background()
	startTime := time.Now()

	fmt.Printf("  running %d extractors at tier %d...\n", len(extractors), tier)
	profile, err := assembler.Assemble(ctx, repoRoot)
	if err != nil {
		log.Fatalf("failed to assemble profile: %v", err)
	}

	analysisDuration := time.Since(startTime).Seconds()

	// 6. Print profile stats
	fmt.Printf("  profile assembled:\n")
	fmt.Printf("    files: %d\n", profile.Metrics.TotalFiles)
	fmt.Printf("    LOC: %d\n", profile.Metrics.TotalLOC)
	fmt.Printf("    containers: %d\n", profile.Metrics.ContainersDetected)
	fmt.Printf("    languages: %d\n", profile.Metrics.LanguagesCount)

	// Print any extractor errors
	extractorErrors := assembler.Errors()
	if len(extractorErrors) > 0 {
		fmt.Printf("  warnings:\n")
		for _, e := range extractorErrors {
			fmt.Printf("    %s: %v\n", e.Extractor, e.Err)
		}
	}

	// 7. LLM analysis (if allowed)
	var hypothesisResult *classify.HypothesisResult
	llmCostUSD := 0.0

	if *allowExtLLM {
		if !secFilter.ExternalLLMAllowed() {
			log.Fatal("security filter blocked external LLM access (use --allow-external-llm)")
		}

		// Sanitize profile
		sanitizedProfile := secFilter.Sanitize(profile)

		// Create LLM client
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			log.Fatal("OPENROUTER_API_KEY environment variable not set")
		}
		llmClient := discovery.NewLLMClient(apiKey, "https://openrouter.ai/api/v1")

		// Run Hypothesizer
		fmt.Printf("  running LLM analysis with model %s...\n", *model)
		hypothesizer := classify.NewHypothesizer(llmClient, *model)

		hypothesisResult, err = hypothesizer.Analyze(ctx, sanitizedProfile)
		if err != nil {
			log.Printf("LLM analysis failed (continuing without LLM results): %v", err)
			hypothesisResult = nil
		} else {
			llmCostUSD = hypothesisResult.TotalCostUSD
			fmt.Printf("    LLM cost: $%.4f\n", llmCostUSD)
		}
	} else {
		fmt.Println("  skipping LLM analysis (use --allow-external-llm to enable)")
	}

	// 8. Build ArchitectureReport
	report := buildReport(repoRoot, profile, hypothesisResult, analysisDuration, llmCostUSD)

	// 9. Compute confidence summary
	report.ConfidenceSummary = computeConfidence(profile, hypothesisResult)

	// 10. Create output directory
	if *outputDir == "" {
		*outputDir = filepath.Join(repoRoot, ".sdp", "architecture")
	}

	c4Dir := filepath.Join(*outputDir, "c4")
	if err := os.MkdirAll(c4Dir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}
	fmt.Printf("  writing artifacts to %s\n", *outputDir)

	// 11. Write artifacts

	// report.json
	if err := writeArtifact(*outputDir, "report.json", report); err != nil {
		log.Fatalf("failed to write report.json: %v", err)
	}

	// codebase-profile.json
	if err := writeArtifact(*outputDir, "codebase-profile.json", profile); err != nil {
		log.Fatalf("failed to write codebase-profile.json: %v", err)
	}

	// reference-model.json (if LLM was used or if we have enough data)
	if hypothesisResult != nil || len(profile.Infra.Containers) > 0 {
		refModel := buildReferenceModel(profile)
		if err := writeArtifact(*outputDir, "reference-model.json", refModel); err != nil {
			log.Fatalf("failed to write reference-model.json: %v", err)
		}

		// Generate C4 diagrams
		fmt.Println("  generating C4 diagrams...")
		opts := c4.RenderOptions{
			Direction: "TB",
			MaxNodes:  0,
			Theme:     "default",
		}

		// L1
		l1Result, err := c4.RenderL1(refModel, opts)
		if err != nil {
			log.Printf("failed to render L1: %v", err)
		} else {
			if err := writeMermaid(c4Dir, "c4-L1-context.mmd", l1Result.MermaidCode); err != nil {
				log.Printf("failed to write L1 diagram: %v", err)
			}
			fmt.Printf("    L1: %d nodes, %d edges\n", l1Result.NodeCount, l1Result.EdgeCount)
		}

		// L2
		l2Result, err := c4.RenderL2(refModel, opts)
		if err != nil {
			log.Printf("failed to render L2: %v", err)
		} else {
			if err := writeMermaid(c4Dir, "c4-L2-containers.mmd", l2Result.MermaidCode); err != nil {
				log.Printf("failed to write L2 diagram: %v", err)
			}
			fmt.Printf("    L2: %d nodes, %d edges\n", l2Result.NodeCount, l2Result.EdgeCount)
		}

		// L3 for each container
		for _, container := range refModel.Containers {
			l3Result, err := c4.RenderL3(refModel, container.ID, opts)
			if err != nil {
				log.Printf("failed to render L3 for %s: %v", container.ID, err)
				continue
			}
			filename := fmt.Sprintf("c4-L3-%s.mmd", sanitizeFilename(container.ID))
			if err := writeMermaid(c4Dir, filename, l3Result.MermaidCode); err != nil {
				log.Printf("failed to write L3 diagram for %s: %v", container.ID, err)
			}
			fmt.Printf("    L3-%s: %d nodes, %d edges\n", container.ID, l3Result.NodeCount, l3Result.EdgeCount)
		}
	}

	// 12. Print summary
	fmt.Println("\nAnalysis complete!")
	fmt.Printf("  Report: %s\n", filepath.Join(*outputDir, "report.json"))
	fmt.Printf("  Profile: %s\n", filepath.Join(*outputDir, "codebase-profile.json"))
	fmt.Printf("  C4 diagrams: %s\n", c4Dir)

	// Print style hypothesis if available
	if len(report.StyleHypothesis.Styles) > 0 {
		fmt.Println("\nTop architectural styles:")
		for i, style := range report.StyleHypothesis.Styles {
			if i >= 3 {
				break
			}
			fmt.Printf("  %d. %s (%.0f%% confidence)\n", i+1, style.Style, style.Confidence*100)
		}
	}

	if len(report.Risks) > 0 {
		fmt.Printf("\nRisks detected: %d\n", len(report.Risks))
		for _, risk := range report.Risks {
			if risk.Severity == architect.SeverityHigh {
				fmt.Printf("  [HIGH] %s: %s\n", risk.Category, risk.Description)
			}
		}
	}
}

// buildReport creates an ArchitectureReport from profile and optional LLM results
func buildReport(repoRoot string, profile *architect.CodebaseProfile, hypothesisResult *classify.HypothesisResult, durationS, llmCost float64) *architect.ArchitectureReport {
	report := &architect.ArchitectureReport{
		Version:           "1.0.0",
		AnalyzedAt:        time.Now(),
		RepoRoot:          repoRoot,
		AnalysisDurationS: durationS,
		LLMCostUSD:        llmCost,
		Metrics:           profile.Metrics,
	}

	// Extract language info
	report.Languages = extractLanguageInfo(profile)

	// Style hypothesis
	if hypothesisResult != nil {
		report.StyleHypothesis = hypothesisResult.StyleHypothesis
		report.PatternsDetected = hypothesisResult.Patterns
		report.Risks = hypothesisResult.Risks
	} else {
		// Default hypothesis
		report.StyleHypothesis = architect.StyleHypothesis{
			Styles: []architect.StyleScore{
				{Style: architect.StyleLayered, Confidence: 0.5, Evidence: []string{"default hypothesis"}},
			},
		}
	}

	// Specs discovered
	report.SpecsDiscovered = profile.Specs

	return report
}

// extractLanguageInfo derives LanguageInfo from a profile
func extractLanguageInfo(profile *architect.CodebaseProfile) architect.LanguageInfo {
	info := architect.LanguageInfo{
		All:          make([]string, 0),
		Distribution: make(map[string]float64),
	}

	// Count files by extension
	totalFiles := 0
	langCounts := make(map[string]int)

	for ext, count := range profile.FileTree.ExtCounts {
		if count > 0 {
			langCounts[ext] = count
			totalFiles += count
		}
	}

	// Map extensions to languages
	langMap := map[string]string{
		"go":   "go",
		"py":   "python",
		"js":   "javascript",
		"ts":   "typescript",
		"java": "java",
		"rs":   "rust",
		"rb":   "ruby",
		"php":  "php",
		"cs":   "c#",
		"cpp":  "c++",
		"c":    "c",
		"cc":   "c++",
		"h":    "c",
		"hh":   "c++",
	}

	seenLangs := make(map[string]bool)
	for ext, count := range langCounts {
		if lang, ok := langMap[ext]; ok {
			if !seenLangs[lang] {
				info.All = append(info.All, lang)
				seenLangs[lang] = true
			}
			if info.Distribution != nil {
				info.Distribution[lang] += float64(count)
			}
		}
	}

	// Find primary language
	maxCount := 0
	for lang, total := range info.Distribution {
		if total > float64(maxCount) {
			info.Primary = lang
			maxCount = int(total)
		}
	}

	// Convert to percentages
	if totalFiles > 0 {
		for lang := range info.Distribution {
			info.Distribution[lang] = info.Distribution[lang] / float64(totalFiles)
		}
	}

	return info
}

// buildReferenceModel derives a C4 ReferenceModel from a CodebaseProfile
func buildReferenceModel(profile *architect.CodebaseProfile) *architect.ReferenceModel {
	model := &architect.ReferenceModel{
		Version:     "1.0.0",
		State:       architect.ModelObserved,
		GeneratedAt: time.Now().Format(time.RFC3339),
		System: architect.SystemInfo{
			Name: filepath.Base(profile.Name),
		},
		Actors:          make([]architect.Actor, 0),
		ExternalSystems: make([]architect.ExternalSystem, 0),
		Containers:      make([]architect.C4Container, 0),
		Relationships:   make([]architect.C4Relationship, 0),
	}

	// Infer actors from context (heuristic)
	// If this looks like a library/API, add "developer" actor
	isLibrary := false
	for _, spec := range profile.Specs {
		if spec.Kind == "openapi" || spec.Kind == "graphql" || spec.Kind == "protobuf" {
			isLibrary = true
			break
		}
	}
	if isLibrary {
		model.Actors = append(model.Actors, architect.Actor{
			ID:          "developer",
			Description: "Developer using this library/API",
		})
	}

	// Infer external systems from notable dependencies
	depSystemMap := map[string]string{
		"kafka":      "message broker",
		"rabbitmq":   "message broker",
		"redis":      "cache",
		"postgres":   "database",
		"mysql":      "database",
		"mongodb":    "database",
		"elasticsearch": "search engine",
		"s3":         "object storage",
		"dynamodb":   "database",
		"sns":        "pub/sub",
		"sqs":        "message queue",
	}

	seenSystems := make(map[string]bool)
	for _, dep := range profile.Dependencies.NotableDeps {
		lowerName := strings.ToLower(dep.Name)
		for sysPrefix, sysType := range depSystemMap {
			if strings.Contains(lowerName, sysPrefix) && !seenSystems[sysPrefix] {
				model.ExternalSystems = append(model.ExternalSystems, architect.ExternalSystem{
					ID:          sysPrefix,
					Description: sysPrefix + " " + sysType,
					Technology:  sysPrefix,
					Evidence:    "inferred from dependency: " + dep.Name,
				})
				seenSystems[sysPrefix] = true
			}
		}
	}

	// Convert containers from Infra
	for i, infraContainer := range profile.Infra.Containers {
		c4Container := architect.C4Container{
			ID:         fmt.Sprintf("container_%d", i+1),
			Name:       infraContainer.Name,
			Technology: infraContainer.Type,
			Source:     infraContainer.Source,
		}
		if infraContainer.Image != "" {
			c4Container.Technology = infraContainer.Image
		}
		model.Containers = append(model.Containers, c4Container)
	}

	// Add relationships from service dependencies
	for _, svcDep := range profile.Infra.Services {
		// Find container IDs
		var fromID, toID string
		for _, c := range model.Containers {
			if c.Name == svcDep.From {
				fromID = c.ID
			}
			if c.Name == svcDep.To {
				toID = c.ID
			}
		}
		if fromID != "" && toID != "" {
			model.Relationships = append(model.Relationships, architect.C4Relationship{
				From:        fromID,
				To:          toID,
				Description: "depends on",
				Type:        "sync",
			})
		}
	}

	// Add relationships from import graph clusters
	for _, cluster := range profile.ImportGraph.Clusters {
		// Create a container for each cluster if not already present
		containerID := fmt.Sprintf("cluster_%s", sanitizeFilename(cluster.ID))
		found := false
		for _, c := range model.Containers {
			if c.ID == containerID {
				found = true
				break
			}
		}
		if !found {
			model.Containers = append(model.Containers, architect.C4Container{
				ID:          containerID,
				Name:        cluster.ID,
				Description: fmt.Sprintf("Module cluster with %d packages", len(cluster.Packages)),
			})
		}
	}

	return model
}

// computeConfidence calculates confidence scores from profile and LLM results
func computeConfidence(profile *architect.CodebaseProfile, result *classify.HypothesisResult) architect.ConfidenceSummary {
	summary := architect.ConfidenceSummary{
		Overall:            0.0,
		StructuralAnalysis: 0.0,
		StyleHypothesis:    0.0,
		ContractCoverage:   0.0,
	}

	// Structural analysis: based on extractor coverage
	// Count how many data types we have meaningful data for
	dataPoints := 0
	totalDataPoints := 5 // filetree, deps, infra, specs, import

	if profile.FileTree.TotalFiles > 0 {
		dataPoints++
	}
	if len(profile.Dependencies.Manifests) > 0 {
		dataPoints++
	}
	if len(profile.Infra.Containers) > 0 {
		dataPoints++
	}
	if len(profile.Specs) > 0 {
		dataPoints++
	}
	if profile.ImportGraph.Nodes > 0 {
		dataPoints++
	}

	summary.StructuralAnalysis = float64(dataPoints) / float64(totalDataPoints)

	// Style hypothesis: from LLM result average confidence
	if result != nil && len(result.StyleHypothesis.Styles) > 0 {
		totalConf := 0.0
		for _, style := range result.StyleHypothesis.Styles {
			totalConf += style.Confidence
		}
		summary.StyleHypothesis = totalConf / float64(len(result.StyleHypothesis.Styles))
	}

	// Contract coverage: specs / containers
	if profile.Metrics.ContainersDetected > 0 {
		summary.ContractCoverage = float64(profile.Metrics.ContractsDiscovered) / float64(profile.Metrics.ContainersDetected)
		if summary.ContractCoverage > 1.0 {
			summary.ContractCoverage = 1.0
		}
	}

	// Overall: weighted average
	summary.Overall = (summary.StructuralAnalysis * 0.4) + (summary.StyleHypothesis * 0.4) + (summary.ContractCoverage * 0.2)

	// Add note if no LLM analysis was performed
	if result == nil {
		summary.Note = "Style hypothesis is default; run with --allow-external-llm for AI-powered analysis"
	}

	return summary
}

// filterExtractors filters extractors by language
func filterExtractors(extractors []architect.Extractor, languages []string) []architect.Extractor {
	if len(languages) == 0 {
		return extractors
	}

	// Language-agnostic extractors that should always be included
	agnosticExtractors := map[string]bool{
		"filetree": true,
		"deps":     true,
		"specs":    true,
		"generated": true,
		"infra":    true,
		"git_history": true,
	}

	// Map languages to extractor names
	langToExtractor := map[string]string{
		"go":         "go",
		"golang":     "go",
		"java":       "java",
		"kotlin":     "java",
		"python":     "python",
		"py":         "python",
		"typescript": "TypeScript",
		"ts":         "TypeScript",
		"javascript": "TypeScript",
		"js":         "TypeScript",
	}

	filtered := make([]architect.Extractor, 0, len(extractors))
	for _, ext := range extractors {
		name := ext.Name()
		// Always include language-agnostic extractors
		if agnosticExtractors[name] {
			filtered = append(filtered, ext)
			continue
		}
		// Check if extractor matches any requested language
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

// writeArtifact writes a JSON artifact to a file
func writeArtifact(dir, name string, data interface{}) error {
	path := filepath.Join(dir, name)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// writeMermaid writes a Mermaid diagram to a file
func writeMermaid(dir, name, code string) error {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// sanitizeFilename creates a filesystem-safe filename from a string
func sanitizeFilename(s string) string {
	// Replace special characters with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
	return result
}

func architectUsage() {
	fmt.Fprintln(os.Stderr, "usage: sdp architect <analyze|c4|contracts|conform|greenfield> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  analyze <repo-path>   Full architecture analysis (probe mode)")
	fmt.Fprintln(os.Stderr, "  c4 <repo-path>        Generate C4 diagrams only")
	fmt.Fprintln(os.Stderr, "  contracts <repo-path>  Discover integration contracts")
	fmt.Fprintln(os.Stderr, "  conform <repo-path>    Run conformance check")
	fmt.Fprintln(os.Stderr, "  greenfield             Guided architecture conversation")
}
