// Package extract provides language-specific code extractors for AI Architect.
package extract

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Java/Kotlin domain types
// ---------------------------------------------------------------------------

// JavaExtractionResult is the output of the JavaExtractor.
type JavaExtractionResult struct {
	Language         string            `json:"language"`
	ImportGraph      JavaImportGraph   `json:"import_graph"`
	Frameworks       []JavaFramework   `json:"frameworks"`
	BuildSystem      *BuildSystem      `json:"build_system,omitempty"`
	Modules          []string          `json:"modules,omitempty"`
	ExtractionMethod string            `json:"extraction_method"`
	AccuracyEstimate float64           `json:"accuracy_estimate"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// JavaImportGraph groups imports by the package directory in which they appear.
type JavaImportGraph struct {
	// PackageImports maps a relative package directory (e.g. "src/main/java/com/example/service")
	// to the list of import strings found in files within that directory.
	PackageImports map[string][]string `json:"package_imports"`
}

// JavaFramework describes a detected framework or annotation pattern.
type JavaFramework struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

// BuildSystem captures metadata extracted from pom.xml or build.gradle.
type BuildSystem struct {
	Type         string           `json:"type"` // "maven" | "gradle"
	Dependencies []JavaDependency `json:"dependencies"`
}

// JavaDependency is a single resolved dependency from a build descriptor.
type JavaDependency struct {
	Group    string `json:"group"`
	Artifact string `json:"artifact"`
	Scope    string `json:"scope,omitempty"`
}

// ---------------------------------------------------------------------------
// JavaExtractor
// ---------------------------------------------------------------------------

// JavaExtractor performs regex-based extraction of Java and Kotlin projects.
type JavaExtractor struct{}

// Name returns the extractor identifier.
func (e *JavaExtractor) Name() string { return "java" }

// Extract walks rootDir, collecting Java/Kotlin imports, parsing build files,
// detecting frameworks, and identifying multi-module Maven layouts.
func (e *JavaExtractor) Extract(rootDir string) (*JavaExtractionResult, error) {
	result := &JavaExtractionResult{
		Language:         "java/kotlin",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.70,
		ImportGraph: JavaImportGraph{
			PackageImports: make(map[string][]string),
		},
		Metadata: make(map[string]string),
	}

	foundSource := false

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}

		switch {
		case isJavaFile(rel):
			foundSource = true
			imports, scanErr := scanJavaImports(path)
			if scanErr != nil {
				return nil
			}
			pkgDir := filepath.Dir(rel)
			result.ImportGraph.PackageImports[pkgDir] = append(
				result.ImportGraph.PackageImports[pkgDir], imports...)

		case isKotlinFile(rel):
			foundSource = true
			imports, scanErr := scanKotlinImports(path)
			if scanErr != nil {
				return nil
			}
			pkgDir := filepath.Dir(rel)
			result.ImportGraph.PackageImports[pkgDir] = append(
				result.ImportGraph.PackageImports[pkgDir], imports...)

		case filepath.Base(rel) == "pom.xml":
			bs, parseErr := parsePomXML(path)
			if parseErr == nil && bs != nil {
				if result.BuildSystem == nil {
					result.BuildSystem = bs
				} else {
					// Merge dependencies from additional pom.xml files.
					result.BuildSystem.Dependencies = append(
						result.BuildSystem.Dependencies, bs.Dependencies...)
				}
			}
			// Check for multi-module Maven project.
			modules := parseModules(path)
			if len(modules) > 0 {
				result.Modules = append(result.Modules, modules...)
				result.Metadata["multi_module"] = "true"
			}

		case filepath.Base(rel) == "build.gradle" || filepath.Base(rel) == "build.gradle.kts":
			bs, parseErr := parseBuildGradle(path)
			if parseErr == nil && bs != nil {
				if result.BuildSystem == nil {
					result.BuildSystem = bs
				} else {
					result.BuildSystem.Dependencies = append(
						result.BuildSystem.Dependencies, bs.Dependencies...)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", rootDir, err)
	}

	if !foundSource {
		return nil, fmt.Errorf("no Java or Kotlin source files found in %s", rootDir)
	}

	// Deduplicate imports within each package directory.
	for pkg, imports := range result.ImportGraph.PackageImports {
		result.ImportGraph.PackageImports[pkg] = dedup(imports)
	}

	// Detect frameworks from collected imports.
	result.Frameworks = detectJavaFrameworks(result.ImportGraph)

	return result, nil
}

// ---------------------------------------------------------------------------
// Import scanning
// ---------------------------------------------------------------------------

var (
	javaImportRe  = regexp.MustCompile(`^import\s+(static\s+)?([a-zA-Z0-9_.]+\*?);`)
	kotlinImportRe = regexp.MustCompile(`^import\s+([a-zA-Z0-9_.]+)`)
)

func scanJavaImports(path string) ([]string, error) {
	return scanImportsWithRegex(path, javaImportRe, 2)
}

func scanKotlinImports(path string) ([]string, error) {
	return scanImportsWithRegex(path, kotlinImportRe, 1)
}

func scanImportsWithRegex(path string, re *regexp.Regexp, group int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := re.FindStringSubmatch(line)
		if m != nil && group < len(m) {
			imports = append(imports, m[group])
		}
	}
	return imports, scanner.Err()
}

// ---------------------------------------------------------------------------
// pom.xml parsing (regex-based, no XML library)
// ---------------------------------------------------------------------------

var (
	pomDepBlockRe = regexp.MustCompile(`(?s)<dependency>(.*?)</dependency>`)
	pomGroupRe    = regexp.MustCompile(`<groupId>\s*(.*?)\s*</groupId>`)
	pomArtifactRe = regexp.MustCompile(`<artifactId>\s*(.*?)\s*</artifactId>`)
	pomScopeRe    = regexp.MustCompile(`<scope>\s*(.*?)\s*</scope>`)
	pomModuleRe   = regexp.MustCompile(`<module>\s*(.*?)\s*</module>`)
	pomModulesRe  = regexp.MustCompile(`(?s)<modules>(.*?)</modules>`)
)

func parsePomXML(path string) (*BuildSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	blocks := pomDepBlockRe.FindAllStringSubmatch(content, -1)
	if len(blocks) == 0 {
		return &BuildSystem{Type: "maven"}, nil
	}

	bs := &BuildSystem{Type: "maven"}
	for _, block := range blocks {
		dep := JavaDependency{}
		if m := pomGroupRe.FindStringSubmatch(block[1]); m != nil {
			dep.Group = m[1]
		}
		if m := pomArtifactRe.FindStringSubmatch(block[1]); m != nil {
			dep.Artifact = m[1]
		}
		if m := pomScopeRe.FindStringSubmatch(block[1]); m != nil {
			dep.Scope = m[1]
		}
		if dep.Group != "" || dep.Artifact != "" {
			bs.Dependencies = append(bs.Dependencies, dep)
		}
	}
	return bs, nil
}

func parseModules(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	modulesBlock := pomModulesRe.FindStringSubmatch(content)
	if modulesBlock == nil {
		return nil
	}

	matches := pomModuleRe.FindAllStringSubmatch(modulesBlock[1], -1)
	var modules []string
	for _, m := range matches {
		modules = append(modules, m[1])
	}
	return modules
}

// ---------------------------------------------------------------------------
// build.gradle parsing (regex-based)
// ---------------------------------------------------------------------------

var gradleDepRe = regexp.MustCompile(
	`(?:implementation|api|compile|compileOnly|runtimeOnly|testImplementation)\s+` +
		`['"]([a-zA-Z0-9._-]+):([a-zA-Z0-9._-]+)(?::([a-zA-Z0-9._-]+))?['"]`)

func parseBuildGradle(path string) (*BuildSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	matches := gradleDepRe.FindAllStringSubmatch(content, -1)
	bs := &BuildSystem{Type: "gradle"}
	for _, m := range matches {
		dep := JavaDependency{
			Group:    m[1],
			Artifact: m[2],
		}
		bs.Dependencies = append(bs.Dependencies, dep)
	}
	return bs, nil
}

// ---------------------------------------------------------------------------
// Spring Boot detection
// ---------------------------------------------------------------------------

// springAnnotations maps annotation names to the framework evidence string.
var springAnnotations = map[string]string{
	"@RestController":          "Spring MVC REST controller",
	"@Service":                 "Spring service component",
	"@Repository":              "Spring data repository",
	"@SpringBootApplication":   "Spring Boot application",
	"@Entity":                  "JPA/Hibernate entity",
}

func detectJavaFrameworks(ig JavaImportGraph) []JavaFramework {
	var frameworks []JavaFramework

	// Collect all imports into a single set for efficient lookup.
	allImports := make(map[string]bool)
	for _, imports := range ig.PackageImports {
		for _, imp := range imports {
			allImports[imp] = true
		}
	}

	// Spring Boot detection: check for spring-related import prefixes
	// AND annotation-bearing evidence (we infer annotations from import paths).
	springEvidence := detectSpringEvidence(allImports)
	if len(springEvidence) > 0 {
		confidence := float64(len(springEvidence)) / float64(len(springAnnotations))
		if confidence > 1.0 {
			confidence = 1.0
		}
		frameworks = append(frameworks, JavaFramework{
			Name:       "Spring Boot",
			Confidence: confidence,
			Evidence:   springEvidence,
		})
	}

	return frameworks
}

// detectSpringEvidence scans the import set for Spring-related patterns and
// returns human-readable evidence strings.
func detectSpringEvidence(imports map[string]bool) []string {
	var evidence []string

	springImportPrefixes := map[string]string{
		"org.springframework.web.bind.annotation":            "@RestController",
		"org.springframework.stereotype.Service":             "@Service",
		"org.springframework.stereotype.Repository":          "@Repository",
		"org.springframework.boot.autoconfigure.SpringBootApplication": "@SpringBootApplication",
		"org.springframework.boot.SpringApplication":         "@SpringBootApplication",
		"javax.persistence.Entity":                           "@Entity",
		"jakarta.persistence.Entity":                         "@Entity",
	}

	seen := make(map[string]bool)
	for imp := range imports {
		for prefix, annotation := range springImportPrefixes {
			if strings.HasPrefix(imp, prefix) || imp == prefix {
				if !seen[annotation] {
					seen[annotation] = true
					desc, ok := springAnnotations[annotation]
					if ok {
						evidence = append(evidence, desc)
					}
				}
			}
		}
	}

	// Sort for deterministic output.
	sort.Strings(evidence)
	return evidence
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isJavaFile(rel string) bool {
	return strings.HasSuffix(rel, ".java")
}

func isKotlinFile(rel string) bool {
	return strings.HasSuffix(rel, ".kt") || strings.HasSuffix(rel, ".kts")
}

func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
