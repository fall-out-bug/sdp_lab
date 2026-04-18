package architect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Diagrams       []DiagramResult     `json:"diagrams,omitempty"`
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

	// Report timings
	for _, t := range timings {
		p.progress(StageExtract, fmt.Sprintf("  %s: %s", t.Name, t.Duration.Round(time.Millisecond)), &t)
	}

	// Stage 2: Assemble (already done by ProfileAssembler)
	p.progress(StageAssemble, fmt.Sprintf("Assembled profile: %d files, %d LOC",
		profile.Metrics.TotalFiles, profile.Metrics.TotalLOC), nil)

	// Infer system name from repoRoot (with access to filesystem).
	if profile.Name == "" {
		profile.Name = inferSystemName(profile, p.config.RepoRoot)
	}

	// Stage 3: Security filter
	p.progress(StageFilter, "Applying security filter...", nil)
	sanitizedProfile, secretsFound := p.secFilter.Sanitize(profile)
	if secretsFound.Count > 0 {
		p.progress(StageFilter, fmt.Sprintf("Security filter: %d secrets redacted (types: %v)",
			secretsFound.Count, secretsFound.Types), nil)
	}
	// Use sanitized profile for all subsequent stages
	profile = sanitizedProfile
	result.Profile = profile

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
			// Store enrichment results in profile for hypothesis/pattern/risk outputs
			profile.Enrichment = enrichmentResult.Enrichment
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

		// Update container count to reflect the actual reference model
		// (assembler only counts Dockerfile containers, but the model includes
		// Maven modules, import clusters, etc.).
		profile.Metrics.ContainersDetected = len(refModel.Containers)

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

func pipelineSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-").Replace(s)

	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == '-':
			if !lastHyphen {
				b.WriteRune(r)
				lastHyphen = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "unnamed"
	}
	return slug
}

func parentPrefix(path string) string {
	normalized := strings.Trim(filepath.ToSlash(filepath.Clean(path)), "/")
	if normalized == "" || normalized == "." {
		return ""
	}

	for _, part := range strings.Split(normalized, "/") {
		if part != "" && part != "." {
			return part
		}
	}
	return ""
}

// BuildReferenceModelFromProfile creates a C4 ReferenceModel from a profile.
// This is exported so the CLI layer can call it without importing the c4 package.
func BuildReferenceModelFromProfile(profile *CodebaseProfile) *ReferenceModel {
	// Infer system name if not already set on the profile.
	if profile.Name == "" {
		// repoRoot is not available here; use metadata or manifest-based heuristics.
		profile.Name = inferSystemName(profile, "")
	}

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
		if spec.Kind == "openapi" || spec.Kind == "graphql" || spec.Kind == "protobuf" ||
			spec.Kind == "thrift" || spec.Kind == "connect_proto" {
			isLibrary = true
			break
		}
	}
	// Also detect frameworks/libraries: Maven modules + no web framework signals
	// a library rather than a deployable service.
	if !isLibrary && len(profile.Infra.ModuleBoundaries) > 0 {
		hasWebFramework := false
		for _, dep := range profile.Dependencies.NotableDeps {
			if dep.Signal == "web_framework" {
				hasWebFramework = true
				break
			}
		}
		if !hasWebFramework {
			isLibrary = true
		}
	}
	if isLibrary {
		model.Actors = append(model.Actors, Actor{
			ID:          "developer",
			Description: "Developer using this library/framework",
		})
	}

	// Convert infrastructure containers to C4 containers (skip CI-only images)
	containerIdx := 0
	seenNames := make(map[string]int)
	for _, infraContainer := range profile.Infra.Containers {
		if isCIPipelineContainer(infraContainer) {
			continue
		}
		containerIdx++
		c4Container := C4Container{
			ID:         fmt.Sprintf("container_%d", containerIdx),
			Name:       infraContainer.Name,
			Technology: infraContainer.Type,
			Source:     infraContainer.Source,
		}
		if infraContainer.Image != "" {
			c4Container.Technology = infraContainer.Image
		}

		baseName := c4Container.Name
		baseCount := seenNames[baseName]
		if baseCount > 0 {
			suffix := strings.TrimSuffix(filepath.Base(infraContainer.Source), filepath.Ext(infraContainer.Source))
			if suffix == "" || strings.EqualFold(suffix, "Dockerfile") || pipelineSlug(suffix) == pipelineSlug(baseName) {
				suffix = fmt.Sprintf("%d", baseCount+1)
			}
			candidateName := fmt.Sprintf("%s-%s", baseName, suffix)
			if seenNames[candidateName] > 0 {
				suffix = fmt.Sprintf("%d", baseCount+1)
				candidateName = fmt.Sprintf("%s-%s", baseName, suffix)
			}
			c4Container.Name = candidateName
			c4Container.ID = fmt.Sprintf("%s-%s", c4Container.ID, pipelineSlug(suffix))
		}
		seenNames[baseName] = baseCount + 1
		if c4Container.Name != baseName {
			seenNames[c4Container.Name]++
		}

		model.Containers = append(model.Containers, c4Container)
	}

	// Add Maven/Gradle module boundaries as containers
	containerSeen := make(map[string]bool)
	for _, c := range model.Containers {
		containerSeen[c.Name] = true
	}
	for _, mb := range profile.Infra.ModuleBoundaries {
		for _, child := range mb.Children {
			childName := filepath.Base(child)
			// Use the full child path as display name to avoid basename collisions
			// (e.g. "core" and "sql/core" both have basename "core").
			displayName := childName
			if strings.Contains(child, "/") {
				displayName = filepath.ToSlash(child)
			}
			if childName == "" || childName == "." {
				continue
			}
			if containerSeen[displayName] {
				continue
			}
			containerSeen[displayName] = true
			containerIdx++
			model.Containers = append(model.Containers, C4Container{
				ID:          pipelineSlug(displayName),
				Name:        displayName,
				Technology:  mb.BuildSystem,
				Source:      mb.Path,
				Description: mb.BuildSystem + " module: " + child,
			})
		}
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

	// Add relationships from directed module dependency edges (import-graph derived).
	// EdgeSync sources use module slugs (e.g. "spark-sql-core" from module "sql/core").
	// Build a mapping from module child paths to container IDs using the module boundary data.
	nameToID := make(map[string]string)
	for _, c := range model.Containers {
		nameToID[c.Name] = c.ID
	}
		// Map each module child path to its slug and associate with container.
		// For ambiguous basenames (e.g. "core" from both "core" and "sql/core"),
		// use the container whose Description contains the full child path.
		for _, mb := range profile.Infra.ModuleBoundaries {
			for _, child := range mb.Children {
				parts := strings.Split(filepath.ToSlash(child), "/")
				slug := "spark-" + strings.Join(parts, "-")
				childName := filepath.Base(child)
				if id, ok := nameToID[childName]; ok {
					nameToID[slug] = id
				}
				// Also try matching by Description (contains full module path).
				for _, c := range model.Containers {
					if strings.Contains(c.Description, child) {
						nameToID[slug] = c.ID
						break
					}
				}
			}
		}
	for _, edge := range profile.Edges {
		if edge.Kind != EdgeSync {
			continue
		}
		idA := nameToID[edge.Source]
		idB := nameToID[edge.Target]
		if idA == "" || idB == "" {
			continue
		}
		model.Relationships = append(model.Relationships, C4Relationship{
			From:        idA,
			To:          idB,
			Description: "declared module dependency",
			Type:        "sync",
		})
	}

	// Add import graph clusters as containers (only substantial clusters:
	// must have >1 package and at least some internal edges to avoid noise).
	for _, cluster := range profile.ImportGraph.Clusters {
		if len(cluster.Packages) <= 1 {
			continue
		}
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
		"hadoop": "distributed filesystem", "hdfs": "distributed filesystem",
		"yarn": "cluster manager", "mesos": "cluster manager",
		"zookeeper": "coordination service",
		"hive":      "data warehouse", "spark": "data processing",
		"flink": "stream processing", "storm": "stream processing",
		"cassandra": "database", "presto": "query engine", "trino": "query engine",
		"kubernetes": "container orchestration", "minio": "object storage",
		"airflow": "workflow orchestration",
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
	if model.System.Name != "" && len(model.ExternalSystems) > 0 {
		systemName := strings.ToLower(model.System.Name)
		filtered := make([]ExternalSystem, 0, len(model.ExternalSystems))
		for _, ext := range model.ExternalSystems {
			if strings.Contains(systemName, strings.ToLower(ext.ID)) {
				continue
			}
			filtered = append(filtered, ext)
		}
		model.ExternalSystems = filtered
	}

	// Add runtime coupling edges as relationships and external systems.
	// Internal protocols (spark-rpc, py4j) are NOT added as external systems.
	seenRuntimeEdges := make(map[string]bool)
	internalProtocols := map[string]bool{"spark-rpc": true, "py4j": true, "spark-connect": true, "spark": true, "grpc": true}

	for _, edge := range profile.Edges {
		if edge.Kind != EdgeRuntimeBridge && edge.Kind != EdgeRPC {
			continue
		}

		// Map source file to container
		fromID := ""
		for _, c := range model.Containers {
			if strings.Contains(edge.Source, c.Name) || strings.Contains(c.Source, edge.Source) {
				fromID = c.ID
				break
			}
		}

		desc := edge.Protocol + " runtime coupling"
		edgeKey := fromID + "->" + edge.Protocol
		if !seenRuntimeEdges[edgeKey] {
			seenRuntimeEdges[edgeKey] = true
			if fromID != "" {
				model.Relationships = append(model.Relationships, C4Relationship{
					From:        fromID,
					To:          edge.Protocol,
					Description: desc,
					Type:        "runtime",
					Contract:    edge.Method,
				})
			}
		}

		// Only add as external system if not an internal protocol
		sysID := edge.Protocol
		if !internalProtocols[sysID] && !seenSystems[sysID] && edge.Protocol != "" {
			model.ExternalSystems = append(model.ExternalSystems, ExternalSystem{
				ID:          sysID,
				Description: edge.Protocol + " runtime bridge",
				Technology:  edge.Protocol,
				Evidence:    edge.Path + ": " + edge.Method,
			})
			seenSystems[sysID] = true
		}
	}

	// Phantom container filtering: keep only containers that have real architectural
	// significance. A container is kept if ANY of:
	//   - Is a Maven/Gradle module (in module boundaries)
	//   - Is a Dockerfile-derived container
	//   - Has edges to/from it in the relationship graph
	//   - Is a Python package cluster with internal cohesion (internal edges > 0)
	//   - Has technology set (non-empty, meaning it was detected from a manifest)
	containerEdgeCount := make(map[string]int)
	for _, r := range model.Relationships {
		containerEdgeCount[r.From]++
		containerEdgeCount[r.To]++
	}
		mavenModuleSet := make(map[string]bool)
		for _, b := range profile.Infra.ModuleBoundaries {
			for _, child := range b.Children {
				childName := filepath.Base(child)
				mavenModuleSet[childName] = true
				if strings.Contains(child, "/") {
					mavenModuleSet[filepath.ToSlash(child)] = true
				}
			}
		}
		// Build set of cluster IDs that represent real architectural boundaries.
		// Only include clusters that correspond to actual Maven modules or are
		// Python packages. Java clusters from adaptive splitting (org.apache.spark.*)
		// are excluded because they overlap with Maven module containers.
		significantClusters := make(map[string]bool)
		// Build a set of module slugs that we already have as containers.
		moduleSlugSet := make(map[string]bool)
		for _, b := range profile.Infra.ModuleBoundaries {
			for _, child := range b.Children {
				slug := pipelineSlug(filepath.ToSlash(child))
				if slug != "" {
					moduleSlugSet[slug] = true
				}
			}
		}
		for _, cl := range profile.ImportGraph.Clusters {
			cid := fmt.Sprintf("cluster_%s", cl.ID)
			isPython := strings.Contains(cl.ID, "pyspark") || strings.Contains(cl.ID, "python.")
			// Include module-derived clusters (those whose ID matches a module slug).
			isModuleCluster := moduleSlugSet[cl.ID]
			if isPython || isModuleCluster {
				significantClusters[cid] = true
			}
		}
		filtered := make([]C4Container, 0, len(model.Containers))
		for _, c := range model.Containers {
			isDockerfile := strings.Contains(strings.ToLower(c.Source), "dockerfile")
			hasTech := c.Technology != ""
			_, isMavenMod := mavenModuleSet[c.Name]
			_, isSignificant := significantClusters[c.ID]
			_, hasEdges := containerEdgeCount[c.ID]
			// Maven modules always pass — they represent real architectural
			// boundaries from the build system, even without detected edges.
			keepMaven := isMavenMod
			if keepMaven || (!isMavenMod && (isDockerfile || hasEdges || hasTech || isSignificant)) {
				filtered = append(filtered, c)
			}
		}
	model.Containers = filtered

	// Assign components to containers from import graph clusters (for L3 diagrams).
	assignComponentsFromClusters(profile, model)

	return model
}

// assignComponentsFromClusters populates C4Component entries within each container
// based on import graph clusters. This enables L3 diagram rendering.
func assignComponentsFromClusters(profile *CodebaseProfile, model *ReferenceModel) {
	if len(profile.ImportGraph.Clusters) == 0 {
		return
	}

	// Build container ID set for exact matching.
	containerByID := make(map[string]int, len(model.Containers))
	for i, c := range model.Containers {
		containerByID[c.ID] = i
	}

	// Map each cluster to the best-matching container.
	for _, cluster := range profile.ImportGraph.Clusters {
		containerIdx := -1

		// Pass 1: exact match by container ID.
		if idx, ok := containerByID[cluster.ID]; ok {
			containerIdx = idx
		}

		// Pass 2: fuzzy name matching.
		if containerIdx < 0 {
			bestScore := 0
			clusterLower := strings.ToLower(cluster.ID)
			for i, c := range model.Containers {
				nameLower := strings.ToLower(c.Name)
				score := 0
				if clusterLower == nameLower {
					score = 100
				} else if strings.Contains(clusterLower, nameLower) || strings.Contains(nameLower, clusterLower) {
					score = 50
				}
				if score > bestScore {
					bestScore = score
					containerIdx = i
				}
			}
		}

		if containerIdx < 0 {
			continue
		}

		// Build component from cluster.
		parts := strings.Split(cluster.ID, "/")
		shortName := parts[len(parts)-1]
		if shortName == "" {
			shortName = cluster.ID
		}

		comp := C4Component{
			ID:          model.Containers[containerIdx].ID + "-" + pipelineSlug(shortName),
			Path:        cluster.ID,
			Description: shortName + " component",
			Confidence:  clusterComponentConfidence(cluster),
		}
		model.Containers[containerIdx].Components = append(
			model.Containers[containerIdx].Components, comp,
		)
	}
}

// clusterComponentConfidence returns a confidence score for a component
// derived from an import cluster's edge density.
func clusterComponentConfidence(cluster ImportCluster) float64 {
	total := cluster.InternalEdges + cluster.ExternalEdges
	if total >= 5 {
		return 0.85
	}
	if total >= 2 {
		return 0.70
	}
	return 0.50
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

// inferSystemName attempts to determine the project name from multiple sources.
// It tries, in order: metadata, README.md heading, pom.xml <artifactId>,
// pom.xml <name>, directory basename.
func inferSystemName(profile *CodebaseProfile, repoRoot string) string {
	// 1. Check metadata
	if profile.Metadata != nil {
		if name := strings.TrimSpace(profile.Metadata["project_name"]); name != "" {
			return name
		}
	}

	// 2. Check README.md first heading (usually the best human-readable name)
	if repoRoot != "" {
		if name := extractReadmeTitle(filepath.Join(repoRoot, "README.md")); name != "" {
			return name
		}
	}

	// 3. Check pom.xml <artifactId> (prefer over <name> which is often verbose
	// like "Spark Project Parent POM" rather than "spark-parent").
	for _, m := range profile.Dependencies.Manifests {
		if strings.HasSuffix(m.Path, "pom.xml") {
			fullPath := filepath.Join(repoRoot, m.Path)
			if name := extractPomField(fullPath, "artifactId"); name != "" {
				return name
			}
		}
	}
	if repoRoot != "" {
		if name := extractPomField(filepath.Join(repoRoot, "pom.xml"), "artifactId"); name != "" {
			return name
		}
	}

	// 4. Check pom.xml <name> as fallback
	for _, m := range profile.Dependencies.Manifests {
		if strings.HasSuffix(m.Path, "pom.xml") {
			fullPath := filepath.Join(repoRoot, m.Path)
			if name := extractPomName(fullPath); name != "" {
				return name
			}
		}
	}
	if repoRoot != "" {
		if name := extractPomName(filepath.Join(repoRoot, "pom.xml")); name != "" {
			return name
		}
	}

	// 5. Directory basename
	if repoRoot != "" {
		base := filepath.Base(repoRoot)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}

	return "unknown-system"
}

// pomNameRe extracts the content of the first top-level <name>...</name> in a pom.xml.
// We match non-greedily and take only the first occurrence to avoid picking up names
// from dependency/dependencyManagement blocks.
var pomNameRe = regexp.MustCompile(`(?s)<name>\s*(.*?)\s*</name>`)

// extractPomName reads a pom.xml file and returns the first <name> value.
func extractPomName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	matches := pomNameRe.FindSubmatch(data)
	if len(matches) >= 2 {
		return strings.TrimSpace(string(matches[1]))
	}
	return ""
}

// pomFieldRe extracts a top-level XML element by name from pom.xml.
var pomFieldRe = regexp.MustCompile(`(?s)<([a-zA-Z-]+)>\s*(.*?)\s*</(?:[a-zA-Z-]+)>`)

// extractPomField reads a pom.xml file and returns the first top-level
// element matching the given tag name (e.g. "artifactId").
func extractPomField(path, tag string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	pattern := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(tag) + `>\s*(.*?)\s*</` + regexp.QuoteMeta(tag) + `>`)
	matches := pattern.FindSubmatch(data)
	if len(matches) >= 2 {
		return strings.TrimSpace(string(matches[1]))
	}
	return ""
}

// extractReadmeTitle reads a README file and returns the first Markdown heading text.
func extractReadmeTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
	}
	return ""
}

// isCIPipelineContainer returns true if the container appears to be a CI/build-time
// image rather than a runtime deploy unit.
func isCIPipelineContainer(ci ContainerInfo) bool {
	src := strings.ToLower(ci.Source)
	name := strings.ToLower(ci.Name)

	ciPaths := []string{
		".github/", ".ci/", "ci/", ".circleci/", ".gitlab/",
		"dev/docker/", "dev/spark-test-image/",
		"-test-image/", "test/docker/", "testing/",
	}
	for _, p := range ciPaths {
		if strings.Contains(src, p) {
			return true
		}
	}

	ciNames := []string{
		"lint", "test", "docs", "binder", "check", "build",
		"coverage", "python", "pypy",
	}
	for _, ciName := range ciNames {
		if name == ciName || strings.HasPrefix(name, ciName+"-") {
			return true
		}
	}

	return false
}
