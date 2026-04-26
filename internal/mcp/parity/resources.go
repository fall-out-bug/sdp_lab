// Package parity provides resource parity alignment with normalized catalog.
package parity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResourceDefinition defines an MCP resource with parity tracking.
type ResourceDefinition struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	MIMEType    string                 `json:"mime_type"`
	Path        string                 `json:"path"`        // Relative path in .sdp/
	SourceCLI   string                 `json:"source_cli"`  // CLI command that generates it
	HintTool    string                 `json:"hint_tool"`   // MCP tool to suggest when missing
	ParityStatus ParityStatus          `json:"parity_status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceRegistry manages resource definitions with parity tracking.
type ResourceRegistry struct {
	resources map[string]*ResourceDefinition
}

// NewResourceRegistry creates a new resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]*ResourceDefinition),
	}
}

// Register registers a resource definition.
func (r *ResourceRegistry) Register(resource *ResourceDefinition) error {
	if resource.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}
	if resource.Name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}
	if resource.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}
	if resource.MIMEType == "" {
		return fmt.Errorf("MIME type cannot be empty")
	}
	if resource.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if resource.SourceCLI == "" {
		return fmt.Errorf("source CLI cannot be empty")
	}
	if resource.ParityStatus == "" {
		return fmt.Errorf("parity status cannot be empty")
	}

	r.resources[resource.URI] = resource
	return nil
}

// Get retrieves a resource by URI.
func (r *ResourceRegistry) Get(uri string) (*ResourceDefinition, bool) {
	resource, ok := r.resources[uri]
	return resource, ok
}

// List returns all registered resources.
func (r *ResourceRegistry) List() []*ResourceDefinition {
	resources := make([]*ResourceDefinition, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}
	return resources
}

// GetByParityStatus returns resources filtered by parity status.
func (r *ResourceRegistry) GetByParityStatus(status ParityStatus) []*ResourceDefinition {
	var result []*ResourceDefinition
	for _, resource := range r.resources {
		if resource.ParityStatus == status {
			result = append(result, resource)
		}
	}
	return result
}

// ValidateParity validates that all core resources have full parity.
func (r *ResourceRegistry) ValidateParity() error {
	coreResources := []string{
		"sdp://scout",
		"sdp://architect",
		"sdp://metrics",
		"sdp://spec",
	}

	for _, uri := range coreResources {
		resource, ok := r.Get(uri)
		if !ok {
			return fmt.Errorf("missing core resource: %s", uri)
		}

		if resource.ParityStatus != ParityFull {
			return fmt.Errorf("resource %s does not have full parity (status: %s)",
				uri, resource.ParityStatus)
		}
	}

	return nil
}

// CheckAvailability checks if resources are available on disk.
func (r *ResourceRegistry) CheckAvailability(repoRoot string) map[string]bool {
	availability := make(map[string]bool)

	for uri, resource := range r.resources {
		path := filepath.Join(repoRoot, resource.Path)
		if _, err := os.Stat(path); err == nil {
			availability[uri] = true
		} else {
			availability[uri] = false
		}
	}

	return availability
}

// GetMissingResources returns resources that are not available on disk.
func (r *ResourceRegistry) GetMissingResources(repoRoot string) []*ResourceDefinition {
	var missing []*ResourceDefinition
	availability := r.CheckAvailability(repoRoot)

	for uri, resource := range r.resources {
		if !availability[uri] {
			missing = append(missing, resource)
		}
	}

	return missing
}

// DefaultResources returns the default set of resources aligned with CLI outputs.
func DefaultResources() []*ResourceDefinition {
	return []*ResourceDefinition{
		{
			URI:         "sdp://manifest",
			Name:        "SDP Manifest",
			Description: "Project context primer. Use the CLI directly: `sdp index manifest <repo-path>`.",
			MIMEType:    "text/markdown",
			Path:        ".sdp/manifest.md",
			SourceCLI:   "index manifest",
			HintTool:    "sdp index manifest (CLI)",
			ParityStatus: ParityForward,
			Metadata: map[string]interface{}{
				"category": "project",
				"format":   "markdown",
			},
		},
		{
			URI:         "sdp://scout",
			Name:        "Scout Report",
			Description: "Project card with languages, frameworks, structure, and health signals.",
			MIMEType:    "application/json",
			Path:        ".sdp/scout.json",
			SourceCLI:   "scout",
			HintTool:    "sdp_scout",
			ParityStatus: ParityFull,
			Metadata: map[string]interface{}{
				"category": "analysis",
				"format":   "json",
			},
		},
		{
			URI:         "sdp://architect",
			Name:        "Architecture Report",
			Description: "C4 models, dependency graphs, and quality metrics from architecture analysis.",
			MIMEType:    "application/json",
			Path:        ".sdp/architect/report.json",
			SourceCLI:   "architect analyze",
			HintTool:    "sdp_architect",
			ParityStatus: ParityFull,
			Metadata: map[string]interface{}{
				"category": "analysis",
				"format":   "json",
			},
		},
		{
			URI:         "sdp://metrics",
			Name:        "Metrics Report",
			Description: "Git-derived process health metrics: activity, hygiene, release patterns.",
			MIMEType:    "application/json",
			Path:        ".sdp/metrics/report.json",
			SourceCLI:   "metrics",
			HintTool:    "sdp_metrics",
			ParityStatus: ParityFull,
			Metadata: map[string]interface{}{
				"category": "analysis",
				"format":   "json",
			},
		},
		{
			URI:         "sdp://spec",
			Name:        "Spec Report",
			Description: "Recovered API contracts, business rules, invariants, and SLA parameters.",
			MIMEType:    "application/json",
			Path:        ".sdp/specs/spec.json",
			SourceCLI:   "spec",
			HintTool:    "sdp_spec",
			ParityStatus: ParityFull,
			Metadata: map[string]interface{}{
				"category": "analysis",
				"format":   "json",
			},
		},
		{
			URI:         "sdp://bootstrap",
			Name:        "Bootstrap Report",
			Description: "Agent-ready setup artifacts report (AGENTS.md, hooks, policies). Forward-compatible: bootstrap writes config files, not a JSON report.",
			MIMEType:    "application/json",
			Path:        ".sdp/bootstrap/report.json",
			SourceCLI:   "bootstrap",
			HintTool:    "requires a future CLI enhancement",
			ParityStatus: ParityForward,
			Metadata: map[string]interface{}{
				"category": "setup",
				"format":   "json",
				"note":     "Forward-compatible: bootstrap currently writes config files",
			},
		},
		{
			URI:         "sdp://index/modules",
			Name:        "Index Module List",
			Description: "Module list with metadata from the codebase index. Planned: will be populated by a future index tool enhancement.",
			MIMEType:    "application/json",
			Path:        ".sdp/index/modules.json",
			SourceCLI:   "index",
			HintTool:    "requires a future CLI enhancement",
			ParityStatus: ParityForward,
			Metadata: map[string]interface{}{
				"category": "index",
				"format":   "json",
				"note":     "Planned for future index tool enhancement",
			},
		},
		{
			URI:         "sdp://index/stats",
			Name:        "Index Statistics",
			Description: "Index statistics: file counts, symbol counts, index freshness. Planned: will be populated by a future index tool enhancement.",
			MIMEType:    "application/json",
			Path:        ".sdp/index/stats.json",
			SourceCLI:   "index",
			HintTool:    "requires a future CLI enhancement",
			ParityStatus: ParityForward,
			Metadata: map[string]interface{}{
				"category": "index",
				"format":   "json",
				"note":     "Planned for future index tool enhancement",
			},
		},
	}
}

// ResourcePath represents a normalized resource path for parity checking.
type ResourcePath struct {
	URI         string
	RelativePath string
	Exists      bool
	Size        int64
	LastModified int64
}

// ScanResources scans the .sdp/ directory for available resources.
func ScanResources(repoRoot string) ([]*ResourcePath, error) {
	sdpDir := filepath.Join(repoRoot, ".sdp")
	if _, err := os.Stat(sdpDir); os.IsNotExist(err) {
		return []*ResourcePath{}, nil
	}

	var resources []*ResourcePath

	err := filepath.Walk(sdpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}

		// Convert to resource URI
		uri := pathToURI(relPath)

		resources = append(resources, &ResourcePath{
			URI:          uri,
			RelativePath: relPath,
			Exists:       true,
			Size:         info.Size(),
			LastModified: info.ModTime().Unix(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk .sdp directory: %w", err)
	}

	return resources, nil
}

// pathToURI converts a relative file path to a resource URI.
func pathToURI(path string) string {
	// Remove .sdp/ prefix
	if strings.HasPrefix(path, ".sdp/") {
		path = strings.TrimPrefix(path, ".sdp/")
	}

	// Extract base name for core resources
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Map to known resource URIs
	uriMap := map[string]string{
		"manifest":  "sdp://manifest",
		"scout":     "sdp://scout",
		"report":    "sdp://architect", // .sdp/architect/report.json
		"spec":      "sdp://spec",
		"modules":   "sdp://index/modules",
		"stats":     "sdp://index/stats",
	}

	// Check directory context
	if strings.Contains(path, "architect") {
		return "sdp://architect"
	}
	if strings.Contains(path, "metrics") {
		return "sdp://metrics"
	}
	if strings.Contains(path, "specs") {
		return "sdp://spec"
	}
	if strings.Contains(path, "bootstrap") {
		return "sdp://bootstrap"
	}
	if strings.Contains(path, filepath.Join("index", "modules")) {
		return "sdp://index/modules"
	}
	if strings.Contains(path, filepath.Join("index", "stats")) {
		return "sdp://index/stats"
	}

	// Try direct mapping
	if uri, ok := uriMap[base]; ok {
		return uri
	}

	// Default URI format
	return fmt.Sprintf("sdp://%s", strings.ReplaceAll(base, "_", "-"))
}