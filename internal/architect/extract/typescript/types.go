package typescript

// Known blind spots (documented per WS-08 acceptance criteria):
//   - Path aliases (@/...) are collected from tsconfig.json but not resolved
//     during import graph construction.
//   - Barrel re-export chains (index.ts -> index.ts) are detected as re-exports
//     but the transitive closure is not computed.
//   - Dynamic import() with variable arguments is not handled.
//   - CommonJS require() with computed paths (require(variable)) is not handled.
//   - Webpack module federation is out of scope.
//   - Multi-line import statements spanned across lines may be missed by
//     line-based regex parsing.

// TSImportKind classifies the type of import statement.
type TSImportKind string

const (
	TSImportESModule   TSImportKind = "es_module"
	TSImportCommonJS   TSImportKind = "commonjs"
	TSImportSideEffect TSImportKind = "side_effect"
	TSImportReExport   TSImportKind = "re_export"
	TSImportDynamic    TSImportKind = "dynamic"
)

// TSImportEdge represents a directed import from one file to another.
type TSImportEdge struct {
	From     string       `json:"from"`
	To       string       `json:"to"`
	Kind     TSImportKind `json:"kind"`
	Line     int          `json:"line,omitempty"`
	Resolved bool         `json:"resolved"` // true if specifier was resolved to a local path
}

// TSPackageNode describes a single TS/JS module discovered during extraction.
type TSPackageNode struct {
	Path        string `json:"path"`
	RelPath     string `json:"rel_path"`
	IsBarrel    bool   `json:"is_barrel"`
	IsGenerated bool   `json:"is_generated"`
	Cluster     string `json:"cluster"`
}

// TSDetectedFramework records a TS/JS framework detected from imports and config.
type TSDetectedFramework struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// TSDependencyEntry represents a single dependency from package.json.
type TSDependencyEntry struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Dev     bool   `json:"dev,omitempty"`
}

// TSWorkspaceInfo describes a detected monorepo workspace package.
type TSWorkspaceInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// TSPathAlias maps a tsconfig path alias prefix to its resolved directory.
type TSPathAlias struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
}

// TSImportGraph is the result of running TSExtractor against a TS/JS project.
type TSImportGraph struct {
	Nodes            []TSPackageNode       `json:"nodes"`
	Edges            []TSImportEdge        `json:"edges"`
	Clusters         []string              `json:"clusters"`
	BarrelFiles      []string              `json:"barrel_files"`
	Frameworks       []TSDetectedFramework `json:"frameworks,omitempty"`
	Dependencies     []TSDependencyEntry   `json:"dependencies,omitempty"`
	Workspaces       []TSWorkspaceInfo     `json:"workspaces,omitempty"`
	PathAliases      []TSPathAlias         `json:"path_aliases,omitempty"`
	IsMonorepo       bool                  `json:"is_monorepo"`
	MonorepoTool     string                `json:"monorepo_tool,omitempty"`
	ExtractionMethod string                `json:"extraction_method"`
	AccuracyEstimate float64               `json:"accuracy_estimate"`
}

// skipDirs are directories that should never be traversed.
var skipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
	".output":      true,
	"coverage":     true,
	".git":         true,
}

// extensions lists file extensions treated as TypeScript/JavaScript source.
var extensions = map[string]bool{
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".mjs":    true,
	".cjs":    true,
	".vue":    true,
	".svelte": true,
}

// generatedSuffixes lists file suffixes that indicate generated TS/JS files.
var generatedSuffixes = []string{
	".generated.ts",
	".generated.js",
	".gen.ts",
	".gen.js",
	".pb.ts",
	".pb.js",
	".d.ts", // type declarations are typically generated
}

// generatedPaths lists path substrings that indicate generated code.
var generatedPaths = []string{
	"__generated__/",
	"generated/",
}

// barrelFileNames are filenames treated as barrel (re-export) files.
var barrelFileNames = map[string]bool{
	"index.ts":  true,
	"index.tsx": true,
	"index.js":  true,
	"index.jsx": true,
}
