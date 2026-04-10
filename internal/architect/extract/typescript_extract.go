package extract

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// TypeScript/JavaScript domain types
// ---------------------------------------------------------------------------

// ImportKind classifies the type of import statement.
type ImportKind string

const (
	ImportESModule   ImportKind = "es_module"
	ImportCommonJS   ImportKind = "commonjs"
	ImportSideEffect ImportKind = "side_effect"
	ImportReExport   ImportKind = "re_export"
)

// ImportEntry records a single import found in a source file.
type ImportEntry struct {
	Specifier string     `json:"specifier"`
	Kind      ImportKind `json:"kind"`
}

// TSExtractionResult is the output of the TypeScriptExtractor.
type TSExtractionResult struct {
	Language         string                    `json:"language"`
	ExtractionMethod string                    `json:"extraction_method"`
	AccuracyEstimate float64                   `json:"accuracy_estimate"`
	Imports          map[string][]ImportEntry   `json:"imports,omitempty"`
	Dependencies     []TSDependency            `json:"dependencies,omitempty"`
	Workspaces       []string                  `json:"workspaces,omitempty"`
	PathAliases      map[string]string         `json:"path_aliases,omitempty"`
	Frameworks       []TSFramework             `json:"frameworks,omitempty"`
}

// TSDependency is a single dependency from package.json.
type TSDependency struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Dev     bool   `json:"dev,omitempty"`
}

// TSFramework describes a detected TypeScript/JavaScript framework.
type TSFramework struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"` // "high", "medium", "low"
	Evidence   string `json:"evidence"`
}

// tsSkipDirs are directories that should never be traversed for TypeScript projects.
var tsSkipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".next":        true,
}

// tsExtensions lists file extensions treated as TypeScript/JavaScript.
var tsExtensions = map[string]bool{
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
	".mjs": true,
	".cjs": true,
	".vue": true,
}

// Import regexes — compiled once.
var (
	reESModule   = regexp.MustCompile(`^import\s+.*\s+from\s+['"]([^'"]+)['"]`)
	reSideEffect = regexp.MustCompile(`^import\s+['"]([^'"]+)['"]`)
	reCommonJS   = regexp.MustCompile(`const\s+\w+\s*=\s*require\(\s*['"]([^'"]+)['"]\s*\)`)
	reReExport   = regexp.MustCompile(`^export\s+.*\s+from\s+['"]([^'"]+)['"]`)
)

// Framework detection regexes.
var (
	reNestModule     = regexp.MustCompile(`@Module\s*\(`)
	reNestController = regexp.MustCompile(`@Controller\s*\(`)
	reExpressApp     = regexp.MustCompile(`app\.(get|post|put|delete|patch|use)\s*\(`)
)

// TypeScriptExtractor implements Extractor for TypeScript and JavaScript projects.
type TypeScriptExtractor struct{}

// NewTypeScriptExtractor returns a ready-to-use TypeScript extractor.
func NewTypeScriptExtractor() *TypeScriptExtractor {
	return &TypeScriptExtractor{}
}

// Name returns the extractor name.
func (e *TypeScriptExtractor) Name() string { return "TypeScript" }

// Detect returns true if rootDir contains tsconfig.json, package.json with TS/JS
// files, or any .ts/.tsx files.
func (e *TypeScriptExtractor) Detect(rootDir string) bool {
	// Fast checks: config files in root.
	for _, f := range []string{"tsconfig.json", "package.json"} {
		if fileExists(filepath.Join(rootDir, f)) {
			return true
		}
	}
	// Fallback: scan for any TS/JS file (limited depth).
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if tsExtensions[filepath.Ext(path)] {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// Extract walks rootDir, parses imports, package.json, tsconfig.json, and
// detects frameworks.
func (e *TypeScriptExtractor) Extract(rootDir string) (*TSExtractionResult, error) {
	result := &TSExtractionResult{
		Language:         "typescript",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.60,
		Imports:          make(map[string][]ImportEntry),
		PathAliases:      make(map[string]string),
	}

	// 1. Parse tsconfig.json path aliases.
	aliases := parseTSConfig(filepath.Join(rootDir, "tsconfig.json"))
	result.PathAliases = aliases

	// 2. Parse package.json for dependencies and workspaces.
	deps, workspaces := parseTSPackageJSON(filepath.Join(rootDir, "package.json"))
	result.Dependencies = deps
	result.Workspaces = workspaces

	// 3. Walk source files and extract imports.
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // skip unreadable entries
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if !tsExtensions[ext] {
			return nil
		}
		imports, scanErr := extractImports(path)
		if scanErr != nil {
			return nil //nolint:nilerr // skip unreadable files
		}
		if len(imports) > 0 {
			rel, relErr := filepath.Rel(rootDir, path)
			if relErr != nil {
				rel = path
			}
			result.Imports[rel] = imports
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 4. Detect frameworks.
	result.Frameworks = detectTSFrameworks(rootDir, result)

	return result, nil
}

// extractImports scans a single file and returns all import entries found.
func extractImports(path string) ([]ImportEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []ImportEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Try re-export first (more specific: starts with "export ... from").
		if m := reReExport.FindStringSubmatch(line); m != nil {
			imports = append(imports, ImportEntry{Specifier: m[1], Kind: ImportReExport})
			continue
		}

		// ES module import (import X from "Y").
		if m := reESModule.FindStringSubmatch(line); m != nil {
			imports = append(imports, ImportEntry{Specifier: m[1], Kind: ImportESModule})
			continue
		}

		// Side-effect import (import "Y").
		if m := reSideEffect.FindStringSubmatch(line); m != nil {
			imports = append(imports, ImportEntry{Specifier: m[1], Kind: ImportSideEffect})
			continue
		}

		// CommonJS require.
		if m := reCommonJS.FindStringSubmatch(line); m != nil {
			imports = append(imports, ImportEntry{Specifier: m[1], Kind: ImportCommonJS})
			continue
		}
	}
	return imports, scanner.Err()
}

// parseTSConfig reads tsconfig.json and extracts compilerOptions.paths aliases.
// Returns a map from alias prefix (e.g. "@/") to resolved path (e.g. "src/").
func parseTSConfig(path string) map[string]string {
	aliases := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		return aliases
	}

	var cfg struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return aliases
	}

	baseURL := cfg.CompilerOptions.BaseURL
	if baseURL == "" {
		baseURL = "."
	}

	for alias, targets := range cfg.CompilerOptions.Paths {
		if len(targets) == 0 {
			continue
		}
		// Strip trailing wildcard from alias: "@/*" -> "@/"
		cleanAlias := strings.TrimSuffix(alias, "*")
		// Strip trailing wildcard from target: "src/*" -> "src/"
		cleanTarget := strings.TrimSuffix(targets[0], "*")
		// Resolve relative to baseUrl.
		// Use path.Join-like logic but preserve trailing slash so callers
		// can do simple prefix replacement.
		resolved := filepath.Join(baseURL, cleanTarget)
		if strings.HasSuffix(cleanTarget, "/") && !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		aliases[cleanAlias] = resolved
	}
	return aliases
}

// packageJSON is a minimal representation of package.json.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Workspaces      interface{}       `json:"workspaces"` // string[] or {packages: string[]}
}

// parseTSPackageJSON reads package.json and returns dependency list and workspace patterns.
func parseTSPackageJSON(path string) ([]TSDependency, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, nil
	}

	var deps []TSDependency
	for name, version := range pkg.Dependencies {
		deps = append(deps, TSDependency{Name: name, Version: version, Dev: false})
	}
	for name, version := range pkg.DevDependencies {
		deps = append(deps, TSDependency{Name: name, Version: version, Dev: true})
	}

	workspaces := parseWorkspaces(pkg.Workspaces)

	return deps, workspaces
}

// parseWorkspaces handles both array and object forms of the "workspaces" field.
func parseWorkspaces(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	// Array form: ["packages/*", "apps/*"]
	if arr, ok := raw.([]interface{}); ok {
		var out []string
		for _, v := range arr {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	// Object form: {"packages": ["packages/*"]}
	if obj, ok := raw.(map[string]interface{}); ok {
		if pkgs, ok := obj["packages"]; ok {
			return parseWorkspaces(pkgs)
		}
	}
	return nil
}

// detectTSFrameworks inspects configuration files and import patterns to identify
// which frameworks the project uses.
func detectTSFrameworks(rootDir string, result *TSExtractionResult) []TSFramework {
	var frameworks []TSFramework

	// --- Next.js ---
	hasNextConfig := fileExists(filepath.Join(rootDir, "next.config.js")) ||
		fileExists(filepath.Join(rootDir, "next.config.mjs")) ||
		fileExists(filepath.Join(rootDir, "next.config.ts"))
	hasPagesDir := dirExists(filepath.Join(rootDir, "pages"))
	hasAppDir := dirExists(filepath.Join(rootDir, "app"))
	hasNextDep := hasTSDependency(result.Dependencies, "next")

	if hasNextConfig && (hasPagesDir || hasAppDir) {
		frameworks = append(frameworks, TSFramework{
			Name:       "Next.js",
			Confidence: "high",
			Evidence:   "next.config.* + pages/ or app/ directory",
		})
	} else if hasNextConfig || hasNextDep {
		frameworks = append(frameworks, TSFramework{
			Name:       "Next.js",
			Confidence: "medium",
			Evidence:   evidenceForNext(hasNextConfig, hasNextDep),
		})
	}

	// --- NestJS ---
	hasNestDep := hasTSDependency(result.Dependencies, "@nestjs/core")
	nestDecoratorFound := scanFilesForPattern(rootDir, reNestModule) ||
		scanFilesForPattern(rootDir, reNestController)

	if hasNestDep && nestDecoratorFound {
		frameworks = append(frameworks, TSFramework{
			Name:       "NestJS",
			Confidence: "high",
			Evidence:   "@nestjs/core dependency + @Module/@Controller decorators",
		})
	} else if hasNestDep {
		frameworks = append(frameworks, TSFramework{
			Name:       "NestJS",
			Confidence: "medium",
			Evidence:   "@nestjs/core dependency",
		})
	} else if nestDecoratorFound {
		frameworks = append(frameworks, TSFramework{
			Name:       "NestJS",
			Confidence: "low",
			Evidence:   "@Module/@Controller decorators found",
		})
	}

	// --- Express ---
	hasExpressDep := hasTSDependency(result.Dependencies, "express")
	expressPatternFound := scanFilesForPattern(rootDir, reExpressApp)

	if hasExpressDep && expressPatternFound {
		frameworks = append(frameworks, TSFramework{
			Name:       "Express",
			Confidence: "high",
			Evidence:   "express dependency + app.get/post/use patterns",
		})
	} else if hasExpressDep {
		frameworks = append(frameworks, TSFramework{
			Name:       "Express",
			Confidence: "medium",
			Evidence:   "express dependency",
		})
	}

	// --- Vue ---
	hasVueDep := hasTSDependency(result.Dependencies, "vue")
	hasVueFiles := hasFileWithExtension(rootDir, ".vue")

	if hasVueDep && hasVueFiles {
		frameworks = append(frameworks, TSFramework{
			Name:       "Vue",
			Confidence: "high",
			Evidence:   "vue dependency + .vue files",
		})
	} else if hasVueDep || hasVueFiles {
		evidence := "vue dependency"
		if hasVueFiles {
			evidence = ".vue files"
		}
		frameworks = append(frameworks, TSFramework{
			Name:       "Vue",
			Confidence: "medium",
			Evidence:   evidence,
		})
	}

	// --- Angular ---
	hasAngularJSON := fileExists(filepath.Join(rootDir, "angular.json"))
	hasAngularDep := hasTSDependency(result.Dependencies, "@angular/core")

	if hasAngularJSON && hasAngularDep {
		frameworks = append(frameworks, TSFramework{
			Name:       "Angular",
			Confidence: "high",
			Evidence:   "angular.json + @angular/core dependency",
		})
	} else if hasAngularJSON {
		frameworks = append(frameworks, TSFramework{
			Name:       "Angular",
			Confidence: "high",
			Evidence:   "angular.json configuration file",
		})
	} else if hasAngularDep {
		frameworks = append(frameworks, TSFramework{
			Name:       "Angular",
			Confidence: "medium",
			Evidence:   "@angular/core dependency",
		})
	}

	return frameworks
}

// --- Helpers ---

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasTSDependency(deps []TSDependency, name string) bool {
	for _, d := range deps {
		if d.Name == name {
			return true
		}
	}
	return false
}

// scanFilesForPattern walks TS/JS files and returns true if any line matches re.
func scanFilesForPattern(rootDir string, re *regexp.Regexp) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !tsExtensions[filepath.Ext(path)] {
			return nil
		}
		f, fErr := os.Open(path)
		if fErr != nil {
			return nil //nolint:nilerr
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if re.MatchString(scanner.Text()) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

// hasFileWithExtension returns true if any file with the given extension exists
// under rootDir (skipping skipDirs).
func hasFileWithExtension(rootDir string, ext string) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ext {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func evidenceForNext(hasConfig, hasDep bool) string {
	switch {
	case hasConfig:
		return "next.config.* found"
	case hasDep:
		return "next dependency in package.json"
	default:
		return "next indicators"
	}
}
