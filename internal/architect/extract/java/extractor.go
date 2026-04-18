// Package java provides tree-sitter-based extraction for Java/Kotlin projects.
package java

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Extractor performs tree-sitter and regex-based extraction of Java/Kotlin projects.
type Extractor struct {
	// Root directory of the project
	RootDir string
	// Accuracy estimate for extraction (0.0-1.0)
	AccuracyEstimate float64
}

// NewExtractor creates a new Java/Kotlin extractor.
func NewExtractor(rootDir string) *Extractor {
	return &Extractor{
		RootDir:          rootDir,
		AccuracyEstimate: 0.75, // 75% accuracy with tree-sitter queries + regex fallback
	}
}

// JavaExtractionResult is the main result type from extraction.
type JavaExtractionResult struct {
	Language         string
	ImportGraph      JavaImportGraph
	Frameworks       []JavaFramework
	BuildSystem      *JavaBuildSystem
	PomProperties    map[string]string
	SubmoduleDeps    []JavaSubmoduleBuildDeps
	Modules          []string
	SpringEndpoints  []JavaSpringEndpoint
	Annotations      []JavaAnnotationSighting
	RuntimeCouplings []JavaRuntimeCouplingSighting
	PackageStructure *JavaPackageStructure
	ExtractionMethod string
	AccuracyEstimate float64
	Metadata         map[string]string
}

// JavaImportGraph groups imports by package directory.
type JavaImportGraph struct {
	PackageImports map[string][]string
}

// JavaFramework describes a detected framework.
type JavaFramework struct {
	Name       string
	Confidence float64
	Evidence   []string
}

// JavaBuildSystem captures build system metadata.
type JavaBuildSystem struct {
	Type         string
	Dependencies []JavaDependency
}

// JavaDependency is a single resolved dependency.
type JavaDependency struct {
	Group    string
	Artifact string
	Scope    string
}

// JavaSubmoduleBuildDeps captures submodule dependencies.
type JavaSubmoduleBuildDeps struct {
	ModuleDir    string
	ArtifactID   string
	Dependencies []JavaDependency
	BuildType    string
}

// JavaSpringEndpoint represents a Spring MVC endpoint.
type JavaSpringEndpoint struct {
	HTTPMethod string
	Path       string
	File       string
	LineNumber int
}

// JavaAnnotationSighting records an annotation.
type JavaAnnotationSighting struct {
	Annotation string
	File       string
	LineNumber int
}

// JavaRuntimeCouplingSighting records runtime coupling.
type JavaRuntimeCouplingSighting struct {
	Type     string
	File     string
	Line     int
	Evidence string
}

// JavaPackageStructure describes package layout.
type JavaPackageStructure struct {
	SourceDirs   []string
	TestDirs     []string
	HasKotlin    bool
	RootPackages []string
	BuildTool    string
	MultiModule  bool
}

// Extract performs the extraction analysis.
func (e *Extractor) Extract() (*JavaExtractionResult, error) {
	result := &JavaExtractionResult{
		Language:         "java/kotlin",
		ExtractionMethod: "tree-sitter+regex",
		AccuracyEstimate: e.AccuracyEstimate,
		ImportGraph: JavaImportGraph{
			PackageImports: make(map[string][]string),
		},
		PackageStructure: &JavaPackageStructure{},
		Metadata:         make(map[string]string),
	}

	// Detect package structure and build tool
	result.PackageStructure = detectPackageStructure(e.RootDir)

	// Parse settings.gradle for Gradle multi-project
	gradleSubprojects := parseSettingsGradle(e.RootDir)
	if len(gradleSubprojects) > 0 {
		result.PackageStructure.MultiModule = true
		result.Modules = append(result.Modules, gradleSubprojects...)
		result.Metadata["gradle_multi_project"] = "true"
	}

	// Extract properties from root pom.xml
	if rootPomData, err := os.ReadFile(filepath.Join(e.RootDir, "pom.xml")); err == nil {
		result.PomProperties = extractPomProperties(string(rootPomData))
	}

	// Collect all annotations and mappings
	classLevelMappings := make(map[string]string)
	classLevelMappingLines := make(map[string]int)
	allAnnotations := make(map[string]bool)
	lombokAnnotations := make(map[string]bool)
	javaFiles := 0
	kotlinFiles := 0
	foundSource := false

	// Walk the directory tree
	err := filepath.Walk(e.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Skip common non-source directories
			if info.Name() == "target" || info.Name() == "build" || info.Name() == "out" ||
			   info.Name() == ".gradle" || info.Name() == ".idea" || info.Name() == ".mvn" ||
			   info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, relErr := filepath.Rel(e.RootDir, path)
		if relErr != nil {
			return nil
		}

		switch {
		case strings.HasSuffix(rel, ".java"):
			foundSource = true
			javaFiles++
			imports, annotations, pkgDecl, scanErr := scanJavaFile(path, rel)
			if scanErr != nil {
				return nil
			}
			pkgDir := filepath.Dir(rel)
			result.ImportGraph.PackageImports[pkgDir] = append(
				result.ImportGraph.PackageImports[pkgDir], imports...)
			for _, a := range annotations {
				allAnnotations[a.Annotation] = true
				result.Annotations = append(result.Annotations, a)
				if lombokAnnotationLabels[a.Annotation] != "" {
					lombokAnnotations[a.Annotation] = true
				}
			}
			if pkgDecl != "" {
				recordRootPackage(result.PackageStructure, pkgDecl)
			}

			// Detect class-level @RequestMapping
			prefix, prefixLine := extractClassLevelMapping(path)
			if prefix != "" {
				classLevelMappings[rel] = prefix
				classLevelMappingLines[rel] = prefixLine
			}

			// Extract Spring endpoints
			endpoints := extractSpringEndpoints(path, rel, classLevelMappings[rel], classLevelMappingLines[rel])
			result.SpringEndpoints = append(result.SpringEndpoints, endpoints...)

		case strings.HasSuffix(rel, ".kt") && !strings.HasSuffix(rel, ".kts"):
			foundSource = true
			kotlinFiles++
			imports, annotations, pkgDecl, scanErr := scanKotlinFile(path, rel)
			if scanErr != nil {
				return nil
			}
			pkgDir := filepath.Dir(rel)
			result.ImportGraph.PackageImports[pkgDir] = append(
				result.ImportGraph.PackageImports[pkgDir], imports...)
			for _, a := range annotations {
				allAnnotations[a.Annotation] = true
				result.Annotations = append(result.Annotations, a)
			}
			if pkgDecl != "" {
				recordRootPackage(result.PackageStructure, pkgDecl)
			}

			// Detect Kotlin-specific patterns
			detectKotlinPatterns(path, result)

		case filepath.Base(rel) == "pom.xml":
			bs, artifactID, parseErr := parsePomXMLWithMeta(path)
			if parseErr == nil && bs != nil {
				if result.BuildSystem == nil {
					result.BuildSystem = bs
					result.PackageStructure.BuildTool = "maven"
				} else {
					result.BuildSystem.Dependencies = append(
						result.BuildSystem.Dependencies, bs.Dependencies...)
				}

				moduleDir := filepath.Dir(rel)
				if moduleDir == "." {
					moduleDir = ""
				}

				resolvedDeps := make([]JavaDependency, len(bs.Dependencies))
				for i, dep := range bs.Dependencies {
					dep.Artifact = resolvePomProperties(dep.Artifact, result.PomProperties)
					dep.Group = resolvePomProperties(dep.Group, result.PomProperties)
					resolvedDeps[i] = dep
				}
				resolvedArtifactID := resolvePomProperties(artifactID, result.PomProperties)
				result.SubmoduleDeps = append(result.SubmoduleDeps, JavaSubmoduleBuildDeps{
					ModuleDir:    filepath.ToSlash(moduleDir),
					ArtifactID:   resolvedArtifactID,
					Dependencies: resolvedDeps,
					BuildType:    "maven",
				})
			}

			modules := parseModules(path)
			if len(modules) > 0 {
				result.Modules = append(result.Modules, modules...)
				result.PackageStructure.MultiModule = true
				result.Metadata["multi_module"] = "true"
			}

		case filepath.Base(rel) == "build.gradle" || filepath.Base(rel) == "build.gradle.kts":
			bs, parseErr := parseBuildGradle(path)
			if parseErr == nil && bs != nil {
				if result.BuildSystem == nil {
					result.BuildSystem = bs
					if result.PackageStructure.BuildTool == "" {
						result.PackageStructure.BuildTool = "gradle"
					}
				} else {
					result.BuildSystem.Dependencies = append(
						result.BuildSystem.Dependencies, bs.Dependencies...)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", e.RootDir, err)
	}

	if !foundSource {
		return nil, fmt.Errorf("no Java or Kotlin source files found in %s", e.RootDir)
	}

	// Post-processing
	for pkg, imports := range result.ImportGraph.PackageImports {
		result.ImportGraph.PackageImports[pkg] = dedup(imports)
	}

	result.PackageStructure.HasKotlin = kotlinFiles > 0
	if kotlinFiles > 0 {
		result.Metadata["kotlin_files"] = fmt.Sprintf("%d", kotlinFiles)
	}
	if javaFiles > 0 {
		result.Metadata["java_files"] = fmt.Sprintf("%d", javaFiles)
	}

	result.Frameworks = detectFrameworks(result.ImportGraph, allAnnotations, lombokAnnotations)

	sort.Slice(result.SpringEndpoints, func(i, j int) bool {
		if result.SpringEndpoints[i].Path != result.SpringEndpoints[j].Path {
			return result.SpringEndpoints[i].Path < result.SpringEndpoints[j].Path
		}
		return result.SpringEndpoints[i].HTTPMethod < result.SpringEndpoints[j].HTTPMethod
	})

	if len(result.Modules) > 0 {
		result.Modules = dedup(result.Modules)
	}

	result.PackageStructure.RootPackages = dedup(result.PackageStructure.RootPackages)
	sort.Strings(result.PackageStructure.RootPackages)

	return result, nil
}

// Helper functions

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

func recordRootPackage(ps *JavaPackageStructure, pkgDecl string) {
	if ps == nil || pkgDecl == "" {
		return
	}
	segments := strings.Split(pkgDecl, ".")
	root := pkgDecl
	if len(segments) >= 2 {
		root = segments[0] + "." + segments[1]
	}
	for _, existing := range ps.RootPackages {
		if existing == root {
			return
		}
	}
	ps.RootPackages = append(ps.RootPackages, root)
}

// detectPackageStructure checks for standard Maven/Gradle source layouts.
func detectPackageStructure(rootDir string) *JavaPackageStructure {
	ps := &JavaPackageStructure{}

	sourceDirs := []string{
		"src/main/java",
		"src/test/java",
		"src/main/kotlin",
		"src/test/kotlin",
		"src/main/resources",
		"src/test/resources",
	}

	for _, dir := range sourceDirs {
		fullPath := filepath.Join(rootDir, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			if strings.Contains(dir, "test") {
				ps.TestDirs = append(ps.TestDirs, dir)
			} else if !strings.Contains(dir, "resources") {
				ps.SourceDirs = append(ps.SourceDirs, dir)
			}
			if strings.Contains(dir, "kotlin") {
				ps.HasKotlin = true
			}
		}
	}

	return ps
}

// Regex patterns
var (
	javaPackageRe   = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)\s*;`)
	kotlinPackageRe = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)`)
	springAnnotRe   = regexp.MustCompile(`@(RestController|Service|Repository|Component|Configuration|Entity|SpringBootApplication)`)
	springBeanRe    = regexp.MustCompile(`@Bean`)
	lombokAnnotRe   = regexp.MustCompile(`@(Data|Builder|Getter|Setter|AllArgsConstructor|NoArgsConstructor|RequiredArgsConstructor|Value|With|Slf4j|Log|Log4j2|Cleanup|Synchronized|SneakyThrows|CustomLog)`)
)

// Spring and Lombok annotation labels
var springAnnotations = map[string]string{
	"@RestController":        "Spring MVC REST controller",
	"@Service":               "Spring service component",
	"@Repository":            "Spring data repository",
	"@Component":             "Spring component",
	"@Configuration":         "Spring configuration class",
	"@SpringBootApplication": "Spring Boot application",
	"@Entity":                "JPA/Hibernate entity",
	"@Bean":                  "Spring bean definition",
}

var lombokAnnotationLabels = map[string]string{
	"@Data":                    "Lombok @Data (generates getters/setters/equals/hashCode/toString)",
	"@Builder":                 "Lombok @Builder (builder pattern)",
	"@Getter":                  "Lombok @Getter",
	"@Setter":                  "Lombok @Setter",
	"@AllArgsConstructor":      "Lombok @AllArgsConstructor",
	"@NoArgsConstructor":       "Lombok @NoArgsConstructor",
	"@RequiredArgsConstructor": "Lombok @RequiredArgsConstructor",
	"@Value":                   "Lombok @Value (immutable holder)",
	"@Slf4j":                   "Lombok @Slf4j (logger)",
}
