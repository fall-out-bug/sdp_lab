package architect

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PipelineConfig holds configuration for the analysis pipeline.
type PipelineConfig struct {
	// RepoRoot is the absolute path to the repository to analyze.
	RepoRoot string

	// Tier controls depth: 1 (system overview), 2 (container detail), 3 (component source).
	Tier TierLevel

	// ExtractorNames is a comma-separated list of extractor names. Empty means all.
	ExtractorNames []string

	// AllowExternalLLM enables cloud LLM enrichment.
	AllowExternalLLM bool

	// NoLLM disables all LLM calls (overrides AllowExternalLLM).
	NoLLM bool

	// Timeout is the total session timeout. 0 means no timeout.
	Timeout time.Duration

	// Format is the output format: "json", "text", "mermaid".
	Format string

	// C4Level controls which C4 diagram levels to generate: 0=all, 1, 2, or 3.
	C4Level int

	// GroundTruthPath is the path to a ground truth JSON file for evaluation.
	GroundTruthPath string

	// Verbose enables per-extractor timing output.
	Verbose bool
}

// PipelineResult holds the complete output of a pipeline run.
type PipelineResult struct {
	Profile        *CodebaseProfile    `json:"profile"`
	Report         *ArchitectureReport `json:"report"`
	ReferenceModel *ReferenceModel     `json:"reference_model,omitempty"`
	Duration       time.Duration       `json:"duration"`
	Errors         []ExtractorError    `json:"errors,omitempty"`
	Diagrams       []DiagramResult    `json:"diagrams,omitempty"`
}

// DiagramResult holds a rendered C4 diagram.
type DiagramResult struct {
	Level       string `json:"level"`
	MermaidCode string `json:"mermaid_code"`
	NodeCount   int    `json:"node_count"`
	EdgeCount   int    `json:"edge_count"`
}

// ExtractorTiming records the duration of a single extractor run.
type ExtractorTiming struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
}

// PipelineStage represents a named stage in the pipeline.
type PipelineStage string

const (
	StageExtract  PipelineStage = "extract"
	StageAssemble PipelineStage = "assemble"
	StageFilter   PipelineStage = "filter"
	StageEnrich   PipelineStage = "enrich"
	StageModel    PipelineStage = "model"
	StageOutput   PipelineStage = "output"
)

// ProgressCallback is invoked when a pipeline stage starts or completes.
type ProgressCallback func(stage PipelineStage, msg string, timing *ExtractorTiming)

// Pipeline orchestrates the full analysis: extract -> assemble -> filter ->
// enrich -> build model -> output.
type Pipeline struct {
	config     PipelineConfig
	extractors []Extractor
	secFilter  *SecurityFilter
	progress   ProgressCallback
}

// NewPipeline creates a pipeline with the given config and extractors.
func NewPipeline(config PipelineConfig, extractors []Extractor) *Pipeline {
	sf := NewSecurityFilter()
	sf.AllowExternalLLM = config.AllowExternalLLM && !config.NoLLM

	return &Pipeline{
		config:     config,
		extractors: extractors,
		secFilter:  sf,
		progress:   func(stage PipelineStage, msg string, timing *ExtractorTiming) {},
	}
}

// SetProgressCallback sets the progress reporting callback.
func (p *Pipeline) SetProgressCallback(cb ProgressCallback) {
	if cb != nil {
		p.progress = cb
	}
}

// Run executes the full pipeline and returns the result.
func (p *Pipeline) Run(ctx context.Context) (*PipelineResult, error) {
	if p.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}

	start := time.Now()
	result := &PipelineResult{}

	// Stage 1: Extract
	p.progress(StageExtract, fmt.Sprintf("Running %d extractors...", len(p.extractors)), nil)
	profile, timings, extractErrs := p.runExtractors(ctx)
	result.Errors = extractErrs

	if profile == nil {
		return nil, fmt.Errorf("all extractors failed: %d errors", len(extractErrs))
	}
	result.Profile = profile

	// Report timings
	for _, t := range timings {
		p.progress(StageExtract, fmt.Sprintf("  %s: %s", t.Name, t.Duration.Round(time.Millisecond)), &t)
	}

	// Stage 2: Assemble (already done by ProfileAssembler)
	p.progress(StageAssemble, fmt.Sprintf("Assembled profile: %d files, %d LOC",
		profile.Metrics.TotalFiles, profile.Metrics.TotalLOC), nil)

	// Stage 3: Security filter
	p.progress(StageFilter, "Applying security filter...", nil)
	_, secretsFound := p.secFilter.Sanitize(profile)
	if secretsFound.Count > 0 {
		p.progress(StageFilter, fmt.Sprintf("Security filter: %d secrets redacted (types: %v)",
			secretsFound.Count, secretsFound.Types), nil)
	}

	// Stage 4: LLM enrichment (optional)
	if p.config.AllowExternalLLM && !p.config.NoLLM {
		p.progress(StageEnrich, "Running LLM enrichment...", nil)
		enrichmentResult := p.runEnrichment(ctx, profile)
		if enrichmentResult != nil && !enrichmentResult.Completed {
			p.progress(StageEnrich, fmt.Sprintf("LLM enrichment partially completed: %d failures",
				len(enrichmentResult.Failed)), nil)
		} else if enrichmentResult != nil {
			p.progress(StageEnrich, fmt.Sprintf("LLM enrichment completed: %d nodes enriched",
				len(enrichmentResult.Enrichment)), nil)
		}
	} else {
		p.progress(StageEnrich, "LLM enrichment disabled", nil)
	}

	// Stage 5: Build reference model
	p.progress(StageModel, "Building reference model...", nil)
	refModel := BuildReferenceModelFromProfile(profile)
	result.ReferenceModel = refModel
	p.progress(StageModel, fmt.Sprintf("Model: %d containers, %d relationships",
		len(refModel.Containers), len(refModel.Relationships)), nil)

	// Build report
	report := &ArchitectureReport{
		Version:           "1.0.0",
		AnalyzedAt:        time.Now(),
		RepoRoot:          p.config.RepoRoot,
		AnalysisDurationS: time.Since(start).Seconds(),
		LLMCostUSD:        0,
		Metrics:           profile.Metrics,
		SpecsDiscovered:   profile.Specs,
		ConfidenceSummary: ConfidenceSummary{Note: "structural analysis only"},
	}
	result.Report = report

	result.Duration = time.Since(start)
	p.progress(StageOutput, fmt.Sprintf("Pipeline complete in %s",
		result.Duration.Round(time.Millisecond)), nil)

	return result, nil
}

// runExtractors runs all configured extractors and collects fragments.
func (p *Pipeline) runExtractors(ctx context.Context) (*CodebaseProfile, []ExtractorTiming, []ExtractorError) {
	assembler := NewProfileAssembler(p.config.Tier, p.extractors)
	profile, err := assembler.Assemble(ctx, p.config.RepoRoot)
	if err != nil {
		return nil, nil, []ExtractorError{{Extractor: "assembler", Err: err}}
	}

	timings := make([]ExtractorTiming, 0)
	for _, e := range assembler.Errors() {
		timings = append(timings, ExtractorTiming{
			Name:    e.Extractor,
			Success: false,
			Error:   e.Err.Error(),
		})
	}

	return profile, timings, assembler.Errors()
}

// runEnrichment performs LLM enrichment on profile nodes.
func (p *Pipeline) runEnrichment(ctx context.Context, profile *CodebaseProfile) *EnrichmentResult {
	if !p.secFilter.ExternalLLMAllowed() {
		return &EnrichmentResult{
			Completed: false,
			Failed: []EnrichmentError{
				{Stage: "policy", Retriable: false, Err: fmt.Errorf("external LLM not allowed")},
			},
		}
	}

	inputs := make([]EnrichmentInput, 0, len(profile.Infra.Containers))
	for _, c := range profile.Infra.Containers {
		content := fmt.Sprintf("Container: %s\nType: %s\nImage: %s\nSource: %s",
			c.Name, c.Type, c.Image, c.Source)
		if len(c.Ports) > 0 {
			content += fmt.Sprintf("\nPorts: %s", strings.Join(c.Ports, ", "))
		}
		inputs = append(inputs, EnrichmentInput{
			NodeID:  c.Name,
			Content: content,
		})
	}

	if len(inputs) == 0 {
		return &EnrichmentResult{Completed: true, Enrichment: make(map[string]LLMEnrichment)}
	}

	llmCfg := DefaultLLMConfig()
	client := NewLLMClient(llmCfg, p.secFilter)
	enricher := NewSecureEnricher(client, p.secFilter)

	result := enricher.EnrichNodes(ctx, inputs)
	return &result
}

// BuildReferenceModelFromProfile creates a C4 ReferenceModel from a profile.
// This is exported so the CLI layer can call it without importing the c4 package.
func BuildReferenceModelFromProfile(profile *CodebaseProfile) *ReferenceModel {
	model := &ReferenceModel{
		Version:     "1.0.0",
		State:       ModelObserved,
		GeneratedAt: time.Now().Format(time.RFC3339),
		System: SystemInfo{
			Name: profile.Name,
		},
		Actors:          make([]Actor, 0),
		ExternalSystems: make([]ExternalSystem, 0),
		Containers:      make([]C4Container, 0),
		Relationships:   make([]C4Relationship, 0),
	}

	if model.System.Name == "" {
		model.System.Name = "unknown-system"
	}

	// Infer actors from context
	isLibrary := false
	for _, spec := range profile.Specs {
		if spec.Kind == "openapi" || spec.Kind == "graphql" || spec.Kind == "protobuf" {
			isLibrary = true
			break
		}
	}
	if isLibrary {
		model.Actors = append(model.Actors, Actor{
			ID:          "developer",
			Description: "Developer using this library/API",
		})
	}

	// Convert infrastructure containers to C4 containers
	for i, infraContainer := range profile.Infra.Containers {
		c4Container := C4Container{
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
			model.Relationships = append(model.Relationships, C4Relationship{
				From:        fromID,
				To:          toID,
				Description: "depends on",
				Type:        "sync",
			})
		}
	}

	// Add import graph clusters as containers
	for _, cluster := range profile.ImportGraph.Clusters {
		containerID := fmt.Sprintf("cluster_%s", cluster.ID)
		found := false
		for _, c := range model.Containers {
			if c.ID == containerID {
				found = true
				break
			}
		}
		if !found {
			model.Containers = append(model.Containers, C4Container{
				ID:          containerID,
				Name:        cluster.ID,
				Description: fmt.Sprintf("Module cluster with %d packages", len(cluster.Packages)),
			})
		}
	}

	// Infer external systems from notable dependencies
	depSystemMap := map[string]string{
		"kafka": "message broker", "rabbitmq": "message broker",
		"redis": "cache", "postgres": "database", "mysql": "database",
		"mongodb": "database", "elasticsearch": "search engine",
		"s3": "object storage", "dynamodb": "database",
	}
	seenSystems := make(map[string]bool)
	for _, dep := range profile.Dependencies.NotableDeps {
		lowerName := strings.ToLower(dep.Name)
		for sysPrefix, sysType := range depSystemMap {
			if strings.Contains(lowerName, sysPrefix) && !seenSystems[sysPrefix] {
				model.ExternalSystems = append(model.ExternalSystems, ExternalSystem{
					ID:          sysPrefix,
					Description: sysPrefix + " " + sysType,
					Technology:  sysPrefix,
					Evidence:    "inferred from dependency: " + dep.Name,
				})
				seenSystems[sysPrefix] = true
			}
		}
	}

	return model
}

// PipelineOutput represents the complete JSON output of a pipeline run.
type PipelineOutput struct {
	Version        string              `json:"version"`
	AnalyzedAt     time.Time           `json:"analyzed_at"`
	RepoRoot       string              `json:"repo_root"`
	DurationMs     int64               `json:"duration_ms"`
	Profile        *CodebaseProfile    `json:"profile"`
	Report         *ArchitectureReport `json:"report"`
	ReferenceModel *ReferenceModel     `json:"reference_model,omitempty"`
	Errors         []string            `json:"errors,omitempty"`
}

// ToOutput converts a PipelineResult to a PipelineOutput for serialization.
func (r *PipelineResult) ToOutput() *PipelineOutput {
	out := &PipelineOutput{
		Version:        "1.0.0",
		AnalyzedAt:     time.Now(),
		RepoRoot:       r.Report.RepoRoot,
		DurationMs:     r.Duration.Milliseconds(),
		Profile:        r.Profile,
		Report:         r.Report,
		ReferenceModel: r.ReferenceModel,
	}

	for _, e := range r.Errors {
		out.Errors = append(out.Errors, fmt.Sprintf("%s: %v", e.Extractor, e.Err))
	}

	return out
}

// ToJSON serializes the pipeline result to JSON.
func (r *PipelineResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r.ToOutput(), "", "  ")
}

// ToText generates a human-readable text summary.
func (r *PipelineResult) ToText() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Architecture Analysis: %s\n", r.Report.RepoRoot)
	fmt.Fprintf(&b, "Duration: %s\n\n", r.Duration.Round(time.Millisecond))

	if r.Profile != nil {
		fmt.Fprintf(&b, "Files: %d  |  LOC: %d  |  Languages: %d\n",
			r.Profile.Metrics.TotalFiles, r.Profile.Metrics.TotalLOC, r.Profile.Metrics.LanguagesCount)
		fmt.Fprintf(&b, "Containers: %d  |  Components: %d  |  Specs: %d\n",
			r.Profile.Metrics.ContainersDetected, r.Profile.Metrics.ComponentsDetected,
			r.Profile.Metrics.ContractsDiscovered)
	}

	if r.ReferenceModel != nil {
		fmt.Fprintf(&b, "\nC4 Model: %d containers, %d relationships\n",
			len(r.ReferenceModel.Containers), len(r.ReferenceModel.Relationships))
		for _, c := range r.ReferenceModel.Containers {
			fmt.Fprintf(&b, "  - %s (%s)\n", c.Name, c.Technology)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Fprintf(&b, "\nWarnings: %d\n", len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "  - %s: %v\n", e.Extractor, e.Err)
		}
	}

	return b.String()
}

// ToMermaid returns all diagrams as concatenated Mermaid text.
func (r *PipelineResult) ToMermaid() string {
	var b strings.Builder
	for _, d := range r.Diagrams {
		fmt.Fprintf(&b, "%% %s\n%s\n\n", d.Level, d.MermaidCode)
	}
	return b.String()
}
