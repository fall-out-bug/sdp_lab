package architect

// ProfileFragment is a partial contribution from one extractor.
// Extractors populate both legacy fields (Languages, Dependencies, etc.) and
// the canonical data-model fields below (Modules, Edges, APISurfaces, etc.).
// The assembler merges canonical fields using priority-based precedence.
type ProfileFragment struct {
	// Legacy fields — populated by existing extractors.
	Languages      []LanguageInfo     `json:"languages,omitempty"`
	Dependencies   []DependencyInfo   `json:"dependencies,omitempty"`
	ImportGraph    *ImportGraph       `json:"import_graph,omitempty"`
	Infra          *InfraInfo         `json:"infra,omitempty"`
	FileTree       *FileTreeInfo      `json:"file_tree,omitempty"`
	Specs          []SpecArtifact     `json:"specs,omitempty"`
	SQLAnalysis    *SQLAnalysis       `json:"sql_analysis,omitempty"`
	GitAnalysis    *GitAnalysis       `json:"git_analysis,omitempty"`
	Metrics        *CodeMetrics       `json:"metrics,omitempty"`
	InfraArtifacts []string           `json:"infra_artifacts,omitempty"`
	Generated      []GeneratedFile    `json:"generated,omitempty"`

	// Canonical data-model fields — populated by extractors that emit typed
	// model objects. Higher-priority extractors win on conflicts during merge.
	Modules      []Module                `json:"modules,omitempty"`
	Edges        []StructuralEdge        `json:"edges,omitempty"`
	APISurfaces  []APISurface            `json:"api_surfaces,omitempty"`
	Boundaries   []ModuleBoundary        `json:"boundaries,omitempty"`
	Layers       []LayerAssignment       `json:"layers,omitempty"`
	Correlations []DependencyCorrelation `json:",omitempty"`
	Containers   []C4Container           `json:",omitempty"`
	Components   []C4Component           `json:",omitempty"`
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
	Enrichment   map[string]LLMEnrichment `json:"enrichment,omitempty"` // keyed by node/edge ID

	// Canonical data-model fields — populated by priority-based assembler merge.
	Modules      []Module                `json:"modules,omitempty"`
	Edges        []StructuralEdge        `json:"edges,omitempty"`
	APISurfaces  []APISurface            `json:"api_surfaces,omitempty"`
	Boundaries   []ModuleBoundary        `json:"boundaries,omitempty"`
	Layers       []LayerAssignment       `json:"layers,omitempty"`
	Correlations []DependencyCorrelation `json:",omitempty"`
	Containers   []C4Container           `json:",omitempty"`
	Components   []C4Component           `json:",omitempty"`
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
	NamingStyles   []string          `json:"naming_styles,omitempty"`   // "snake_case", "camelCase", "kebab-case", "PascalCase"
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
	Name       string   `json:"name"`
	Source     string   `json:"source"`          // e.g. "services/auth/Dockerfile"
	Type       string   `json:"type"`            // "service", "database", "message_broker", "cache"
	Image      string   `json:"image,omitempty"` // container image reference
	Ports      []string `json:"ports,omitempty"` // exposed ports
	Entrypoint string   `json:"entrypoint,omitempty"` // Dockerfile ENTRYPOINT
	Cmd        string   `json:"cmd,omitempty"`        // Dockerfile CMD
	EnvRefs    []string `json:"env_refs,omitempty"`   // environment variable names (values sanitized)
	Networks   []string `json:"networks,omitempty"`   // docker-compose networks
	DependsOn  []string `json:"depends_on,omitempty"` // service dependencies
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
	Containers     []ContainerInfo     `json:"containers,omitempty"`
	Deployment     DeploymentInfo      `json:"deployment"`
	DeploymentType string              `json:"deployment_type,omitempty"` // "kubernetes", "docker-compose", "serverless", "bare"
	BaseImages     []string            `json:"base_images,omitempty"`
	ExposedPorts   []string            `json:"exposed_ports,omitempty"`
	Services       []ServiceDep        `json:"services,omitempty"`
	Resources      []ResourceInfo      `json:"resources,omitempty"`
	K8sServices    []K8sServiceInfo    `json:"k8s_services,omitempty"`
	Ingresses      []IngressInfo       `json:"ingresses,omitempty"`
	ConfigMaps     []ConfigMapInfo     `json:"configmaps,omitempty"`
	CIJobs         []CIJobInfo         `json:"ci_jobs,omitempty"`
	Networks       []string            `json:"networks,omitempty"`
	Volumes        []string            `json:"volumes,omitempty"`
	TerraformVars  []TerraformVarInfo  `json:"terraform_vars,omitempty"`
	ModuleBoundaries []ModuleBoundaryInfo `json:"module_boundaries,omitempty"`
}

// K8sServiceInfo describes a Kubernetes Service resource.
type K8sServiceInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Type      string   `json:"type,omitempty"` // "ClusterIP", "NodePort", "LoadBalancer", "ExternalName"
	Ports     []string `json:"ports,omitempty"`
	Selector  map[string]string `json:"selector,omitempty"`
	Source    string   `json:"source"`
}

// IngressInfo describes a Kubernetes Ingress resource.
type IngressInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Hosts     []string `json:"hosts,omitempty"`
	Paths     []string `json:"paths,omitempty"`
	Source    string   `json:"source"`
}

// ConfigMapInfo describes a Kubernetes ConfigMap resource.
type ConfigMapInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Keys      []string `json:"keys,omitempty"`
	Source    string   `json:"source"`
}

// CIJobInfo describes a CI/CD pipeline job or stage.
type CIJobInfo struct {
	Name            string   `json:"name"`
	Stage           string   `json:"stage,omitempty"`
	Image           string   `json:"image,omitempty"`
	Triggers        []string `json:"triggers,omitempty"`        // e.g. "push", "pull_request"
	DeployTargets   []string `json:"deploy_targets,omitempty"`  // e.g. "production", "staging"
	Source          string   `json:"source"`
}

// TerraformVarInfo describes a Terraform variable reference.
type TerraformVarInfo struct {
	Name     string `json:"name"`
	Default  string `json:"default,omitempty"`
	Type     string `json:"type,omitempty"`
	Source   string `json:"source"`
}

// ModuleBoundaryInfo describes a build-system-detected module boundary.
type ModuleBoundaryInfo struct {
	Name       string   `json:"name"`
	BuildSystem string  `json:"build_system"` // "maven", "gradle", "npm", "go"
	Path       string   `json:"path"`
	Children   []string `json:"children,omitempty"`
}

// GeneratedFile records a file detected as machine-generated.
type GeneratedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
