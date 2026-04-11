package architect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ExtractorConfig controls concurrency and timeout behaviour for the assembler.
// All fields have sensible defaults; a zero-value ExtractorConfig is valid.
type ExtractorConfig struct {
	MaxExtractorTimeout time.Duration // per-extractor deadline (default 30s)
	MaxConcurrency      int           // max goroutines for extractor group (default runtime.NumCPU())
	MaxTotalTime        time.Duration // overall assembly deadline (default 5m)
}

// defaults fills zero-valued fields with spec Section 7.4 defaults.
func (c ExtractorConfig) defaults() ExtractorConfig {
	if c.MaxExtractorTimeout == 0 {
		c.MaxExtractorTimeout = 30 * time.Second
	}
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = runtime.NumCPU()
	}
	if c.MaxTotalTime == 0 {
		c.MaxTotalTime = 5 * time.Minute
	}
	return c
}

// TierLevel controls how much detail to include in the assembled profile.
type TierLevel int

const (
	Tier1 TierLevel = iota + 1 // ~2K tokens: system overview
	Tier2                       // ~5-15K tokens: per-container detail
	Tier3                       // on-demand: source code snippets
)

// extractorPriority maps extractor names to merge precedence.
// Higher number = higher precedence (wins on conflicts for scalar fields).
// Aligned with spec Section 4 merge order.
var extractorPriority = map[string]int{
	"filetree":   1,  // FileTreeAnalyzer
	"deps":       2,  // DependencyManifestParser
	"specs":      3,  // SpecInventoryScanner
	"infra":      4,  // InfraExtractor
	"go":         5,  // Go adapter
	"python":     6,  // Python adapter
	"java":       7,  // Java adapter
	"typescript": 8,  // TypeScript adapter
	"sql":        9,  // SQL extractor
	"generated":  10, // GeneratedCodeDetector
}

// defaultExtractorPriority is used when an extractor name is not in the map.
const defaultExtractorPriority = 0

// extractorPriorityOf returns the precedence for the given extractor name.
func extractorPriorityOf(name string) int {
	if p, ok := extractorPriority[name]; ok {
		return p
	}
	return defaultExtractorPriority
}

// priorityFragment pairs a fragment with its extractor priority for merge ordering.
type priorityFragment struct {
	fragment *ProfileFragment
	priority int
	name     string
}

// ProfileAssembler collects fragments from extractors and merges them into
// a CodebaseProfile suitable for LLM consumption.
type ProfileAssembler struct {
	extractors []Extractor
	tier       TierLevel
	config     ExtractorConfig
	errors     []ExtractorError
	mu         sync.Mutex
}

// ExtractorError records a non-fatal extractor failure.
type ExtractorError struct {
	Extractor string
	Err       error
}

// NewProfileAssembler creates an assembler with the given extractors and tier.
// An optional ExtractorConfig can be provided; a zero-value config uses defaults.
func NewProfileAssembler(tier TierLevel, extractors []Extractor, cfg ...ExtractorConfig) *ProfileAssembler {
	var c ExtractorConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c = c.defaults()

	return &ProfileAssembler{
		extractors: extractors,
		tier:       tier,
		config:     c,
		errors:     make([]ExtractorError, 0),
	}
}

// Assemble runs all extractors, collects fragments, and merges them into
// a CodebaseProfile. Non-fatal extractor errors are logged and collected.
func (pa *ProfileAssembler) Assemble(ctx context.Context, repoRoot string) (*CodebaseProfile, error) {
	startTime := TimeNow()
	log.Printf("[assembler] starting assembly with %d extractors at tier %d", len(pa.extractors), pa.tier)

	// Apply overall assembly timeout from config.
	totalCtx, totalCancel := context.WithTimeout(ctx, pa.config.MaxTotalTime)
	defer totalCancel()

	fragments := make([]*ProfileFragment, len(pa.extractors))
	g, gctx := errgroup.WithContext(totalCtx)
	g.SetLimit(pa.config.MaxConcurrency)

	// Run extractors concurrently.
	for i, ext := range pa.extractors {
		i, ext := i, ext
		g.Go(func() error {
			// Per-extractor timeout.
			extCtx, cancel := context.WithTimeout(gctx, pa.config.MaxExtractorTimeout)
			defer cancel()

			log.Printf("[assembler] running extractor: %s", ext.Name())
			frag, err := ext.Extract(extCtx, repoRoot)
			if err != nil {
				// Non-fatal: record error and continue
				pa.mu.Lock()
				pa.errors = append(pa.errors, ExtractorError{
					Extractor: ext.Name(),
					Err:       err,
				})
				pa.mu.Unlock()
				log.Printf("[assembler] extractor %s failed (non-fatal): %v", ext.Name(), err)
				return nil // Continue with other extractors
			}
			fragments[i] = frag
			log.Printf("[assembler] extractor %s completed", ext.Name())
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		// Context was cancelled or fatal error
		return nil, fmt.Errorf("extractor group failed: %w", err)
	}

	// Filter out nil fragments and pair with priorities.
	priorityFragments := make([]priorityFragment, 0, len(fragments))
	for i, frag := range fragments {
		if frag != nil {
			name := pa.extractors[i].Name()
			priorityFragments = append(priorityFragments, priorityFragment{
				fragment: frag,
				priority: extractorPriorityOf(name),
				name:     name,
			})
		}
	}

	profile := pa.mergeFragments(priorityFragments)

	// Build validFragments for computeMetrics (only non-nil).
	validFragments := make([]*ProfileFragment, 0, len(priorityFragments))
	for _, pf := range priorityFragments {
		validFragments = append(validFragments, pf.fragment)
	}

	// Compute aggregate metrics
	pa.computeMetrics(profile, validFragments)

	// Estimate tokens
	tokenCount := EstimateTokens(profile)
	log.Printf("[assembler] estimated token count: %d", tokenCount)

	// Apply tier filtering
	pa.applyTierFilter(profile)

	// Generate summary for Tier1
	if pa.tier == Tier1 {
		profile.Summary = pa.TierSummary(profile)
	}

	// Set metadata
	if profile.Metadata == nil {
		profile.Metadata = make(map[string]string)
	}
	profile.Metadata["tier"] = fmt.Sprintf("%d", pa.tier)
	profile.Metadata["estimated_tokens"] = fmt.Sprintf("%d", tokenCount)
	profile.Metadata["assembly_duration_ms"] = fmt.Sprintf("%d", TimeSince(startTime))
	profile.Metadata["extractors_run"] = fmt.Sprintf("%d", len(pa.extractors))
	profile.Metadata["extractors_succeeded"] = fmt.Sprintf("%d", len(validFragments))

	log.Printf("[assembler] assembly complete: %d files, %d LOC, %d languages",
		profile.Metrics.TotalFiles, profile.Metrics.TotalLOC, profile.Metrics.LanguagesCount)

	return profile, nil
}

// mergeFragments combines multiple priority-tagged fragments into a single CodebaseProfile.
// Legacy fields use simple first-non-nil or append-dedup semantics.
// Canonical data-model fields (Modules, Edges, APISurfaces, Boundaries, Layers)
// are merged using priority-based precedence: higher priority wins on scalar
// conflicts, slice fields are merged and deduplicated.
func (pa *ProfileAssembler) mergeFragments(fragments []priorityFragment) *CodebaseProfile {
	profile := &CodebaseProfile{
		FileTree: FileTreeInfo{
			ExtCounts: make(map[string]int),
		},
		Dependencies: DependencyInfo{
			Manifests:   make([]ManifestInfo, 0),
			NotableDeps: make([]NotableDep, 0),
		},
		ImportGraph: ImportGraph{
			Clusters:             make([]ImportCluster, 0),
			CircularDependencies: make([]CircularDep, 0),
		},
		Infra: InfraInfo{
			Containers:   make([]ContainerInfo, 0),
			Deployment:   DeploymentInfo{},
			BaseImages:   make([]string, 0),
			ExposedPorts: make([]string, 0),
			Services:     make([]ServiceDep, 0),
			Resources:    make([]ResourceInfo, 0),
		},
		Specs:        make([]SpecArtifact, 0),
		SQLAnalysis:  nil,
		GitAnalysis:  nil,
		Metrics:      CodeMetrics{},
		Files:        make(map[string]string),
		Metadata:     make(map[string]string),
		Summary:      "",
	}

	// Track unique values for deduplication (legacy fields)
	seenNotableDeps := make(map[string]bool)
	seenBaseImages := make(map[string]bool)
	seenPorts := make(map[string]bool)
	seenSpecs := make(map[string]bool)

	for _, pf := range fragments {
		frag := pf.fragment
		if frag == nil {
			continue
		}

		// FileTree: first non-nil wins
		if frag.FileTree != nil && profile.FileTree.TotalFiles == 0 {
			profile.FileTree = *frag.FileTree
		}

		// Dependencies: aggregate manifests and notable deps
		for _, depInfo := range frag.Dependencies {
			if depInfo.File != "" {
				profile.Dependencies.Manifests = append(profile.Dependencies.Manifests, ManifestInfo{
					Path:      depInfo.File,
					Language:  depInfo.Language,
					DepsCount: depInfo.DepCount,
				})
			}
			for _, notable := range depInfo.NotableDeps {
				if !seenNotableDeps[notable.Name] {
					seenNotableDeps[notable.Name] = true
					profile.Dependencies.NotableDeps = append(profile.Dependencies.NotableDeps, notable)
				}
			}
		}

		// ImportGraph: merge clusters and circular deps
		if frag.ImportGraph != nil {
			if frag.ImportGraph.ExtractionMethod != "" && profile.ImportGraph.ExtractionMethod == "" {
				profile.ImportGraph.ExtractionMethod = frag.ImportGraph.ExtractionMethod
			}
			if frag.ImportGraph.AccuracyEstimate > 0 && profile.ImportGraph.AccuracyEstimate == 0 {
				profile.ImportGraph.AccuracyEstimate = frag.ImportGraph.AccuracyEstimate
			}
			profile.ImportGraph.Nodes += frag.ImportGraph.Nodes
			profile.ImportGraph.Edges += frag.ImportGraph.Edges
			profile.ImportGraph.Clusters = append(profile.ImportGraph.Clusters, frag.ImportGraph.Clusters...)
			profile.ImportGraph.CircularDependencies = append(profile.ImportGraph.CircularDependencies, frag.ImportGraph.CircularDependencies...)
		}

		// Infra: merge containers and deployment info
		if frag.Infra != nil {
			profile.Infra.Containers = append(profile.Infra.Containers, frag.Infra.Containers...)
			if frag.Infra.Deployment.Type != "" && profile.Infra.Deployment.Type == "" {
				profile.Infra.Deployment = frag.Infra.Deployment
			}
			if frag.Infra.DeploymentType != "" && profile.Infra.DeploymentType == "" {
				profile.Infra.DeploymentType = frag.Infra.DeploymentType
			}
			for _, img := range frag.Infra.BaseImages {
				if !seenBaseImages[img] {
					seenBaseImages[img] = true
					profile.Infra.BaseImages = append(profile.Infra.BaseImages, img)
				}
			}
			for _, port := range frag.Infra.ExposedPorts {
				if !seenPorts[port] {
					seenPorts[port] = true
					profile.Infra.ExposedPorts = append(profile.Infra.ExposedPorts, port)
				}
			}
			profile.Infra.Services = append(profile.Infra.Services, frag.Infra.Services...)
			profile.Infra.Resources = append(profile.Infra.Resources, frag.Infra.Resources...)
		}

		// Specs: deduplicate by path
		for _, spec := range frag.Specs {
			if !seenSpecs[spec.Path] {
				seenSpecs[spec.Path] = true
				profile.Specs = append(profile.Specs, spec)
			}
		}

		// SQL: first non-nil wins
		if frag.SQLAnalysis != nil && profile.SQLAnalysis == nil {
			profile.SQLAnalysis = frag.SQLAnalysis
		}

		// Git: first non-nil wins
		if frag.GitAnalysis != nil && profile.GitAnalysis == nil {
			profile.GitAnalysis = frag.GitAnalysis
		}

		// Metrics: aggregate values
		if frag.Metrics != nil {
			if frag.Metrics.TotalFiles > 0 && profile.Metrics.TotalFiles == 0 {
				profile.Metrics.TotalFiles = frag.Metrics.TotalFiles
			}
			if frag.Metrics.TotalLOC > 0 && profile.Metrics.TotalLOC == 0 {
				profile.Metrics.TotalLOC = frag.Metrics.TotalLOC
			}
			if frag.Metrics.TestRatio > 0 && profile.Metrics.TestRatio == 0 {
				profile.Metrics.TestRatio = frag.Metrics.TestRatio
			}
		}

		// Generated files
		profile.Metrics.GeneratedExcluded += len(frag.Generated)
	}

	// --- Priority-based merge for canonical data-model fields ---
	// Sort fragments by priority (ascending) so that higher priority values
	// overwrite lower priority ones when iterating the sorted slice.
	sorted := make([]priorityFragment, len(fragments))
	copy(sorted, fragments)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].priority < sorted[j].priority
	})

	// Modules: union by ID, highest priority wins on conflict.
	modulesByID := make(map[string]Module)
	modulesPrio := make(map[string]int)
	for _, pf := range sorted {
		for _, m := range pf.fragment.Modules {
			if existingPrio, ok := modulesPrio[m.ID]; !ok || pf.priority > existingPrio {
				modulesByID[m.ID] = m
				modulesPrio[m.ID] = pf.priority
			}
		}
	}
	profile.Modules = make([]Module, 0, len(modulesByID))
	for _, m := range modulesByID {
		profile.Modules = append(profile.Modules, m)
	}

	// Edges: union by (source, target, kind, protocol), increment weight, keep higher confidence.
	type edgeKey struct {
		Source, Target, Kind, Protocol string
	}
	edgesByKey := make(map[edgeKey]StructuralEdge)
	for _, pf := range sorted {
		for _, e := range pf.fragment.Edges {
			key := edgeKey{e.Source, e.Target, string(e.Kind), e.Protocol}
			existing, ok := edgesByKey[key]
			if !ok {
				edgesByKey[key] = e
			} else {
				existing.Weight += e.Weight
				if e.Confidence > existing.Confidence {
					existing.Confidence = e.Confidence
				}
				edgesByKey[key] = existing
			}
		}
	}
	profile.Edges = make([]StructuralEdge, 0, len(edgesByKey))
	for _, e := range edgesByKey {
		profile.Edges = append(profile.Edges, e)
	}

	// APISurfaces: union by (path, method), highest priority wins.
	type apiSurfKey struct {
		Path, Method string
	}
	apiByPath := make(map[apiSurfKey]APISurface)
	apiPriority := make(map[apiSurfKey]int)
	for _, pf := range sorted {
		for _, a := range pf.fragment.APISurfaces {
			key := apiSurfKey{a.Path, a.Method}
			if _, ok := apiByPath[key]; !ok || pf.priority > apiPriority[key] {
				apiByPath[key] = a
				apiPriority[key] = pf.priority
			}
		}
	}
	profile.APISurfaces = make([]APISurface, 0, len(apiByPath))
	for _, a := range apiByPath {
		profile.APISurfaces = append(profile.APISurfaces, a)
	}

	// Boundaries: union by name, merge entry_files and public_interfaces.
	boundaryMap := make(map[string]ModuleBoundary)
	boundaryPrio := make(map[string]int)
	for _, pf := range sorted {
		for _, b := range pf.fragment.Boundaries {
			existing, ok := boundaryMap[b.Name]
			if !ok {
				boundaryMap[b.Name] = b
				boundaryPrio[b.Name] = pf.priority
			} else {
				existing.EntryFiles = mergeStringSlices(existing.EntryFiles, b.EntryFiles)
				existing.PublicInterfaces = mergeStringSlices(existing.PublicInterfaces, b.PublicInterfaces)
				if pf.priority > boundaryPrio[b.Name] {
					existing.Pattern = b.Pattern
					boundaryPrio[b.Name] = pf.priority
				}
				boundaryMap[b.Name] = existing
			}
		}
	}
	profile.Boundaries = make([]ModuleBoundary, 0, len(boundaryMap))
	for _, b := range boundaryMap {
		profile.Boundaries = append(profile.Boundaries, b)
	}

	// Layers: union by layer name, keep highest confidence per layer.
	layerMap := make(map[string]LayerAssignment)
	for _, pf := range sorted {
		for _, l := range pf.fragment.Layers {
			existing, ok := layerMap[l.Layer]
			if !ok || l.Confidence > existing.Confidence {
				layerMap[l.Layer] = l
			}
		}
	}
	profile.Layers = make([]LayerAssignment, 0, len(layerMap))
	for _, l := range layerMap {
		profile.Layers = append(profile.Layers, l)
	}

	return profile
}

// mergeStringSlices returns the union of two string slices, deduplicated.
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// computeMetrics calculates aggregate metrics from fragments.
func (pa *ProfileAssembler) computeMetrics(profile *CodebaseProfile, fragments []*ProfileFragment) {
	// Count languages
	languageSet := make(map[string]bool)
	for _, frag := range fragments {
		for _, lang := range frag.Languages {
			if lang.Primary != "" {
				languageSet[lang.Primary] = true
			}
			for _, l := range lang.All {
				languageSet[l] = true
			}
		}
	}
	profile.Metrics.LanguagesCount = len(languageSet)

	// Pull FileTree metrics if not already set
	if profile.Metrics.TotalFiles == 0 && profile.FileTree.TotalFiles > 0 {
		profile.Metrics.TotalFiles = profile.FileTree.TotalFiles
	}

	// Containers detected
	profile.Metrics.ContainersDetected = len(profile.Infra.Containers)

	// Components detected (from clusters)
	profile.Metrics.ComponentsDetected = len(profile.ImportGraph.Clusters)

	// Contracts discovered
	profile.Metrics.ContractsDiscovered = len(profile.Specs)

	// Estimate missing contracts (heuristic)
	profile.Metrics.ContractsMissing = 0
	if profile.Metrics.ComponentsDetected > 0 && profile.Metrics.ContractsDiscovered < profile.Metrics.ComponentsDetected {
		profile.Metrics.ContractsMissing = profile.Metrics.ComponentsDetected - profile.Metrics.ContractsDiscovered
	}
}

// applyTierFilter filters the profile based on tier level.
// Tier1 (~2K tokens): system overview -- containers, languages, external deps, spec inventory.
// Tier2 (~5-15K tokens): full detail without source code.
// Tier3: include everything (source code on demand).
func (pa *ProfileAssembler) applyTierFilter(profile *CodebaseProfile) {
	switch pa.tier {
	case Tier1:
		// Keep: container names/types, languages, dependency signals, spec list.
		// Strip: import graph edges/clusters, deployment evidence, resource details.
		profile.ImportGraph = ImportGraph{
			ExtractionMethod: profile.ImportGraph.ExtractionMethod,
			AccuracyEstimate: profile.ImportGraph.AccuracyEstimate,
			Nodes:            profile.ImportGraph.Nodes,
			Edges:            profile.ImportGraph.Edges,
		}
		profile.Infra = InfraInfo{
			Containers: profile.Infra.Containers,
		}
		profile.Dependencies = DependencyInfo{
			NotableDeps: profile.Dependencies.NotableDeps, // keep signals
		}
		profile.GitAnalysis = nil
		profile.SQLAnalysis = nil
		profile.Files = nil
	case Tier2:
		// Full detail, but no source code
		profile.Files = nil
	case Tier3:
		// Include everything (source code would be added separately)
	}
}

// EstimateTokens provides a rough token count for the profile.
func EstimateTokens(profile *CodebaseProfile) int {
	data, err := json.Marshal(profile)
	if err != nil {
		return 0
	}
	// Rough approximation: 4 bytes per token
	return len(data) / 4
}

// ContentHash returns a SHA-256 hash of the profile's canonical JSON.
func (pa *ProfileAssembler) ContentHash(profile *CodebaseProfile) string {
	data, err := json.Marshal(profile)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Errors returns non-fatal extractor errors.
func (pa *ProfileAssembler) Errors() []ExtractorError {
	return pa.errors
}

// TierSummary generates a human-readable summary for Tier1 (~2K tokens).
func (pa *ProfileAssembler) TierSummary(profile *CodebaseProfile) string {
	var b strings.Builder

	b.WriteString("## Codebase Profile Summary\n\n")

	// Basic stats
	b.WriteString(fmt.Sprintf("**Files**: %d files across %d directories\n", profile.FileTree.TotalFiles, profile.FileTree.TotalDirs))
	b.WriteString(fmt.Sprintf("**Depth**: Max %d levels\n\n", profile.FileTree.MaxDepth))

	// Languages
	if profile.Metrics.LanguagesCount > 0 {
		b.WriteString(fmt.Sprintf("**Languages**: %d detected\n", profile.Metrics.LanguagesCount))
		if len(profile.FileTree.ExtCounts) > 0 {
			b.WriteString("  Primary extensions: ")
			first := true
			for ext, count := range profile.FileTree.ExtCounts {
				if count > 10 { // Only show significant extensions
					if !first {
						b.WriteString(", ")
					}
					b.WriteString(fmt.Sprintf(".%s (%d)", ext, count))
					first = false
				}
			}
			b.WriteString("\n\n")
		}
	}

	// Dependencies
	if len(profile.Dependencies.Manifests) > 0 {
		b.WriteString(fmt.Sprintf("**Dependencies**: %d manifest files found\n", len(profile.Dependencies.Manifests)))
		for _, m := range profile.Dependencies.Manifests {
			b.WriteString(fmt.Sprintf("  - %s (%s): %d deps\n", m.Path, m.Language, m.DepsCount))
		}
		b.WriteString("\n")
	}

	// Infrastructure
	if len(profile.Infra.Containers) > 0 {
		b.WriteString(fmt.Sprintf("**Containers**: %d detected\n", len(profile.Infra.Containers)))
		for _, c := range profile.Infra.Containers {
			b.WriteString(fmt.Sprintf("  - %s (%s) from %s\n", c.Name, c.Type, c.Source))
		}
		b.WriteString("\n")
	}

	if profile.Infra.Deployment.Type != "" {
		b.WriteString(fmt.Sprintf("**Deployment**: %s\n\n", profile.Infra.Deployment.Type))
	}

	// Architecture
	if profile.Metrics.ComponentsDetected > 0 {
		b.WriteString(fmt.Sprintf("**Components**: %d clusters detected\n\n", profile.Metrics.ComponentsDetected))
	}

	// Specs
	if len(profile.Specs) > 0 {
		b.WriteString(fmt.Sprintf("**Specs**: %d artifacts found\n", len(profile.Specs)))
		for _, s := range profile.Specs {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", s.Kind, s.Path))
		}
		b.WriteString("\n")
	}

	// Patterns
	if len(profile.FileTree.Patterns) > 0 {
		b.WriteString("**Patterns detected**: ")
		b.WriteString(strings.Join(profile.FileTree.Patterns, ", "))
		b.WriteString("\n\n")
	}

	// Metrics summary
	b.WriteString("**Metrics**:\n")
	b.WriteString(fmt.Sprintf("  - LOC: %d\n", profile.Metrics.TotalLOC))
	b.WriteString(fmt.Sprintf("  - Test ratio: %.1f%%\n", profile.Metrics.TestRatio*100))
	b.WriteString(fmt.Sprintf("  - Contracts: %d found, ~%d missing\n",
		profile.Metrics.ContractsDiscovered, profile.Metrics.ContractsMissing))

	return b.String()
}

// TimeNow returns the current time in milliseconds (extracted for testability).
var TimeNow = func() int64 {
	return time.Now().UnixMilli()
}

// TimeSince returns milliseconds since a timestamp.
func TimeSince(start int64) int64 {
	return TimeNow() - start
}
