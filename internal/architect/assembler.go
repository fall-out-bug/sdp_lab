package architect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

// TierLevel controls how much detail to include in the assembled profile.
type TierLevel int

const (
	Tier1 TierLevel = iota + 1 // ~2K tokens: system overview
	Tier2                       // ~5-15K tokens: per-container detail
	Tier3                       // on-demand: source code snippets
)

// ProfileAssembler collects fragments from extractors and merges them into
// a CodebaseProfile suitable for LLM consumption.
type ProfileAssembler struct {
	extractors []Extractor
	tier       TierLevel
	errors     []ExtractorError
	mu         sync.Mutex
}

// ExtractorError records a non-fatal extractor failure.
type ExtractorError struct {
	Extractor string
	Err       error
}

// NewProfileAssembler creates an assembler with the given extractors and tier.
func NewProfileAssembler(tier TierLevel, extractors ...Extractor) *ProfileAssembler {
	return &ProfileAssembler{
		extractors: extractors,
		tier:       tier,
		errors:     make([]ExtractorError, 0),
	}
}

// Assemble runs all extractors, collects fragments, and merges them into
// a CodebaseProfile. Non-fatal extractor errors are logged and collected.
func (pa *ProfileAssembler) Assemble(ctx context.Context, repoRoot string) (*CodebaseProfile, error) {
	startTime := TimeNow()
	log.Printf("[assembler] starting assembly with %d extractors at tier %d", len(pa.extractors), pa.tier)

	fragments := make([]*ProfileFragment, len(pa.extractors))
	g, gctx := errgroup.WithContext(ctx)

	// Run extractors concurrently.
	for i, ext := range pa.extractors {
		i, ext := i, ext
		g.Go(func() error {
			log.Printf("[assembler] running extractor: %s", ext.Name())
			frag, err := ext.Extract(gctx, repoRoot)
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

	// Filter out nil fragments
	validFragments := make([]*ProfileFragment, 0, len(fragments))
	for _, frag := range fragments {
		if frag != nil {
			validFragments = append(validFragments, frag)
		}
	}

	profile := pa.mergeFragments(validFragments)

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

// mergeFragments combines multiple fragments into a single CodebaseProfile.
func (pa *ProfileAssembler) mergeFragments(fragments []*ProfileFragment) *CodebaseProfile {
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
			Containers:     make([]ContainerInfo, 0),
			Deployment:     DeploymentInfo{},
			BaseImages:     make([]string, 0),
			ExposedPorts:   make([]string, 0),
			Services:       make([]ServiceDep, 0),
			Resources:      make([]ResourceInfo, 0),
		},
		Specs:        make([]SpecArtifact, 0),
		SQLAnalysis:  nil,
		GitAnalysis:  nil,
		Metrics:      CodeMetrics{},
		Files:        make(map[string]string),
		Metadata:     make(map[string]string),
		Summary:      "",
	}

	// Track unique values for deduplication
	seenNotableDeps := make(map[string]bool)
	seenBaseImages := make(map[string]bool)
	seenPorts := make(map[string]bool)
	seenSpecs := make(map[string]bool)

	for _, frag := range fragments {
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
				// Single manifest info
				profile.Dependencies.Manifests = append(profile.Dependencies.Manifests, ManifestInfo{
					Path:      depInfo.File,
					Language:  depInfo.Language,
					DepsCount: depInfo.DepCount,
				})
			}
		}
		for _, notable := range profile.Dependencies.NotableDeps {
			if !seenNotableDeps[notable.Name] {
				seenNotableDeps[notable.Name] = true
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
		if frag.SQL != nil && profile.SQLAnalysis == nil {
			profile.SQLAnalysis = frag.SQL
		}
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

	return profile
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
func (pa *ProfileAssembler) applyTierFilter(profile *CodebaseProfile) {
	switch pa.tier {
	case Tier1:
		// Summary only: strip detailed fields
		profile.ImportGraph = ImportGraph{}
		profile.Infra = InfraInfo{
			Containers: profile.Infra.Containers, // Keep container list for summary
		}
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

// TimeNow returns the current time (extracted for testability).
var TimeNow = func() int64 {
	return 0 // Will be set in init
}

// TimeSince returns milliseconds since a timestamp.
func TimeSince(start int64) int64 {
	// This is a placeholder; actual implementation would use time.Now()
	return 0
}

func init() {
	TimeNow = func() int64 {
		return 0 // Placeholder for testing
	}
}
