package architect

// ProfileFragment is a partial contribution from one extractor.
type ProfileFragment struct {
	Languages      []LanguageInfo  `json:"languages,omitempty"`
	Dependencies   *DependencyInfo `json:"dependencies,omitempty"`
	ImportGraph    *ImportGraph    `json:"import_graph,omitempty"`
	Infra          *InfraInfo      `json:"infra,omitempty"`
	FileTree       *FileTreeInfo   `json:"file_tree,omitempty"`
	Specs          []SpecArtifact  `json:"specs,omitempty"`
	SQLAnalysis    *SQLAnalysis    `json:"sql_analysis,omitempty"`
	GitAnalysis    *GitAnalysis    `json:"git_analysis,omitempty"`
	Metrics        *CodeMetrics    `json:"metrics,omitempty"`
	InfraArtifacts []InfraArtifact `json:"infra_artifacts,omitempty"`
}

// CodebaseProfile is the assembled structural snapshot of a repository.
// It serves as the information bottleneck between deterministic extraction
// and LLM interpretation.
type CodebaseProfile struct {
	FileTree     FileTreeInfo   `json:"file_tree"`
	Dependencies DependencyInfo `json:"dependencies"`
	ImportGraph  ImportGraph    `json:"import_graph"`
	Infra        InfraInfo      `json:"infra"`
	SQLAnalysis  *SQLAnalysis   `json:"sql_analysis,omitempty"`
	GitAnalysis  *GitAnalysis   `json:"git_analysis,omitempty"`
	Specs        []SpecArtifact `json:"specs,omitempty"`
	Metrics      CodeMetrics    `json:"metrics"`
}

// FileTreeInfo describes the directory structure of the repository.
type FileTreeInfo struct {
	TotalFiles     int               `json:"total_files"`
	TotalDirs      int               `json:"total_dirs"`
	MaxDepth       int               `json:"max_depth"`
	TopLevel       []string          `json:"top_level,omitempty"`
	NamingPatterns map[string]int    `json:"naming_patterns,omitempty"` // e.g. "controller": 5
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
type DependencyInfo struct {
	Manifests   []ManifestInfo `json:"manifests,omitempty"`
	NotableDeps []NotableDep   `json:"notable_deps,omitempty"`
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
	Name   string `json:"name"`
	Source string `json:"source"` // e.g. "services/auth/Dockerfile"
	Type   string `json:"type"`   // "service", "database", "message_broker", "cache"
}

// DeploymentInfo describes the deployment strategy.
type DeploymentInfo struct {
	Type     string   `json:"type"`               // "kubernetes", "docker-compose", "serverless", "bare"
	Evidence []string `json:"evidence,omitempty"`
}

// InfraInfo aggregates infrastructure-level information.
type InfraInfo struct {
	Containers []ContainerInfo `json:"containers,omitempty"`
	Deployment DeploymentInfo  `json:"deployment"`
}
