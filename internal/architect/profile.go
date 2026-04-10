package architect

// ProfileFragment is a partial contribution from one extractor.
type ProfileFragment struct {
	Languages      []LanguageInfo     `json:"languages,omitempty"`
	Dependencies   []DependencyInfo   `json:"dependencies,omitempty"`
	ImportGraph    *ImportGraph       `json:"import_graph,omitempty"`
	Infra          *InfraInfo         `json:"infra,omitempty"`
	FileTree       *FileTreeInfo      `json:"file_tree,omitempty"`
	Specs          []SpecArtifact     `json:"specs,omitempty"`
	SQL            *SQLAnalysis       `json:"sql,omitempty"`
	SQLAnalysis    *SQLAnalysis       `json:"sql_analysis,omitempty"`
	GitAnalysis    *GitAnalysis       `json:"git_analysis,omitempty"`
	Metrics        *CodeMetrics       `json:"metrics,omitempty"`
	InfraArtifacts []string           `json:"infra_artifacts,omitempty"`
	Generated      []GeneratedFile    `json:"generated,omitempty"`
}

// CodebaseProfile is the assembled structural snapshot of a repository.
// It serves as the information bottleneck between deterministic extraction
// and LLM interpretation.
type CodebaseProfile struct {
	Name         string            `json:"name,omitempty"`
	FileTree     FileTreeInfo      `json:"file_tree"`
	Dependencies DependencyInfo    `json:"dependencies"`
	ImportGraph  ImportGraph       `json:"import_graph"`
	Infra        InfraInfo         `json:"infra"`
	SQLAnalysis  *SQLAnalysis      `json:"sql_analysis,omitempty"`
	GitAnalysis  *GitAnalysis      `json:"git_analysis,omitempty"`
	Specs        []SpecArtifact    `json:"specs,omitempty"`
	Metrics      CodeMetrics       `json:"metrics"`
	Files        map[string]string `json:"files,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Summary      string            `json:"summary,omitempty"`
}

// FileTreeInfo describes the directory structure of the repository.
type FileTreeInfo struct {
	TotalFiles     int               `json:"total_files"`
	TotalDirs      int               `json:"total_dirs"`
	MaxDepth       int               `json:"max_depth"`
	TopLevel       []string          `json:"top_level,omitempty"`
	NamingPatterns map[string]int    `json:"naming_patterns,omitempty"` // e.g. "controller": 5
	Patterns       []string          `json:"patterns,omitempty"`        // detected naming convention labels
	ExtCounts      map[string]int    `json:"ext_counts,omitempty"`      // extension -> file count
}

// ManifestInfo describes a single dependency manifest file.
type ManifestInfo struct {
	Path      string `json:"path"`
	Language  string `json:"language"`
	DepsCount int    `json:"deps_count"`
}

// NotableDep is a dependency that carries architectural signal.
type NotableDep struct {
	Name    string `json:"name"`
	FoundIn int    `json:"found_in"` // number of manifests containing it
	Signal  string `json:"signal"`   // e.g. "event_driven", "orm"
}

// DependencyInfo aggregates dependency manifest data.
// When used as a per-manifest entry (from DependencyManifestParser),
// the File/Language/DepCount/Signals fields are populated.
type DependencyInfo struct {
	Manifests   []ManifestInfo `json:"manifests,omitempty"`
	NotableDeps []NotableDep   `json:"notable_deps,omitempty"`
	File        string         `json:"file,omitempty"`
	Language    string         `json:"language,omitempty"`
	DepCount    int            `json:"dep_count,omitempty"`
	Signals     []string       `json:"signals,omitempty"`
}

// ImportEdge represents a single import relationship.
type ImportEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Language string `json:"language"`
	Kind     string `json:"kind"` // "import", "require", "use"
}

// ImportCluster is a group of tightly-coupled modules.
type ImportCluster struct {
	ID            string   `json:"id"`
	Packages      []string `json:"packages"`
	InternalEdges int      `json:"internal_edges"`
	ExternalEdges int      `json:"external_edges"`
}

// CircularDep describes a circular dependency between modules.
type CircularDep struct {
	A        string `json:"a"`
	B        string `json:"b"`
	EdgeType string `json:"edge_type,omitempty"`
}

// ImportGraph holds the module dependency graph.
type ImportGraph struct {
	ExtractionMethod     string          `json:"extraction_method"` // "tree-sitter", "go/packages", "regex"
	AccuracyEstimate     float64         `json:"accuracy_estimate"`
	Nodes                int             `json:"nodes"`
	Edges                int             `json:"edges"`
	Clusters             []ImportCluster `json:"clusters,omitempty"`
	CircularDependencies []CircularDep   `json:"circular_dependencies,omitempty"`
}

// ContainerInfo describes a deployable unit detected from infrastructure.
type ContainerInfo struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`          // e.g. "services/auth/Dockerfile"
	Type   string   `json:"type"`            // "service", "database", "message_broker", "cache"
	Image  string   `json:"image,omitempty"` // container image reference
	Ports  []string `json:"ports,omitempty"` // exposed ports
}

// DeploymentInfo describes the deployment strategy.
type DeploymentInfo struct {
	Type     string   `json:"type"`               // "kubernetes", "docker-compose", "serverless", "bare"
	Evidence []string `json:"evidence,omitempty"`
}

// ServiceDep represents a dependency between two services.
type ServiceDep struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ResourceInfo describes a Terraform or IaC resource.
type ResourceInfo struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
	Source   string `json:"source,omitempty"`
}

// InfraInfo aggregates infrastructure-level information.
type InfraInfo struct {
	Containers     []ContainerInfo `json:"containers,omitempty"`
	Deployment     DeploymentInfo  `json:"deployment"`
	DeploymentType string          `json:"deployment_type,omitempty"` // "kubernetes", "docker-compose", "serverless", "bare"
	BaseImages     []string        `json:"base_images,omitempty"`
	ExposedPorts   []string        `json:"exposed_ports,omitempty"`
	Services       []ServiceDep    `json:"services,omitempty"`
	Resources      []ResourceInfo  `json:"resources,omitempty"`
}

// GeneratedFile records a file detected as machine-generated.
type GeneratedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
