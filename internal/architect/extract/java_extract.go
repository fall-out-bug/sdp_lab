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

	"sdp_dev/internal/architect"
)

// ---------------------------------------------------------------------------
// Java/Kotlin domain types
// ---------------------------------------------------------------------------

// JavaExtractionResult is the output of the JavaExtractor.
type JavaExtractionResult struct {
	Language         string                    `json:"language"`
	ImportGraph      JavaImportGraph           `json:"import_graph"`
	Frameworks       []JavaFramework           `json:"frameworks"`
	BuildSystem      *BuildSystem              `json:"build_system,omitempty"`
	Modules          []string                  `json:"modules,omitempty"`
	SpringEndpoints  []SpringEndpoint          `json:"spring_endpoints,omitempty"`
	Annotations      []AnnotationSighting      `json:"annotations,omitempty"`
	RuntimeCouplings []RuntimeCouplingSighting `json:"runtime_couplings,omitempty"`
	PackageStructure *PackageStructure         `json:"package_structure,omitempty"`
	ExtractionMethod string                    `json:"extraction_method"`
	AccuracyEstimate float64                   `json:"accuracy_estimate"`
	Metadata         map[string]string         `json:"metadata,omitempty"`
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

// SpringEndpoint represents a detected Spring MVC REST endpoint.
type SpringEndpoint struct {
	HTTPMethod string `json:"http_method"` // GET, POST, PUT, DELETE, PATCH
	Path       string `json:"path"`
	File       string `json:"file,omitempty"`
	LineNumber int    `json:"line_number,omitempty"`
}

// AnnotationSighting records a single annotation found in a source file.
type AnnotationSighting struct {
	Annotation string `json:"annotation"` // e.g. "@RestController", "@Data"
	File       string `json:"file"`
	LineNumber int    `json:"line_number,omitempty"`
}

// RuntimeCouplingSighting records a Java runtime bridge or RPC signal.
type RuntimeCouplingSighting struct {
	Type     string `json:"type"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Evidence string `json:"evidence"`
}

// PackageStructure describes the detected Maven/Gradle source layout.
type PackageStructure struct {
	SourceDirs   []string `json:"source_dirs"`   // detected source directories
	TestDirs     []string `json:"test_dirs"`     // detected test directories
	HasKotlin    bool     `json:"has_kotlin"`    // src/main/kotlin detected
	RootPackages []string `json:"root_packages"` // top-level Java packages (e.g. "com.example")
	BuildTool    string   `json:"build_tool"`    // "maven", "gradle", or ""
	MultiModule  bool     `json:"multi_module"`  // multi-module project detected
}

// ---------------------------------------------------------------------------
// javaSkipDirs — directories to skip when walking Java/Kotlin projects
// ---------------------------------------------------------------------------

var javaSkipDirs = map[string]bool{
	"target":       true, // Maven build output
	"build":        true, // Gradle build output
	"out":          true, // IntelliJ output
	".gradle":      true, // Gradle cache
	".idea":        true, // IntelliJ config
	".mvn":         true, // Maven wrapper config
	".git":         true,
	"node_modules": true,
	".settings":    true, // Eclipse
	".classpath":   true,
	".project":     true,
	"bin":          true,
}

// ---------------------------------------------------------------------------
// Regex patterns — compiled once
// ---------------------------------------------------------------------------

var (
	// Import patterns.
	javaImportRe   = regexp.MustCompile(`^import\s+(static\s+)?([a-zA-Z0-9_.]+\*?);`)
	kotlinImportRe = regexp.MustCompile(`^import\s+([a-zA-Z0-9_.]+)`)

	// Java package declaration.
	javaPackageRe   = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)\s*;`)
	kotlinPackageRe = regexp.MustCompile(`^package\s+([a-zA-Z0-9_.]+)`)

	// Spring annotation patterns.
	springAnnotRe = regexp.MustCompile(`@(RestController|Service|Repository|Component|Configuration|Entity|SpringBootApplication)`)
	springBeanRe  = regexp.MustCompile(`@Bean`)

	// Spring endpoint annotations.
	requestMappingRe    = regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	getMappingRe        = regexp.MustCompile(`@GetMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	postMappingRe       = regexp.MustCompile(`@PostMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	putMappingRe        = regexp.MustCompile(`@PutMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	deleteMappingRe     = regexp.MustCompile(`@DeleteMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	patchMappingRe      = regexp.MustCompile(`@PatchMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	classLevelMappingRe = regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)

	// Lombok annotations.
	lombokAnnotRe   = regexp.MustCompile(`@(Data|Builder|Getter|Setter|AllArgsConstructor|NoArgsConstructor|RequiredArgsConstructor|Value|With|Slf4j|Log|Log4j2|Cleanup|Synchronized|SneakyThrows|CustomLog)`)
	reNettyRpcEnv   = regexp.MustCompile(`(?:NettyRpcEnv|RpcEnv\s*\.\s*create|TransportContext\s*\()`)
	reGatewayServer = regexp.MustCompile(`GatewayServer\s*[\.(]`)
	reRpcImport     = regexp.MustCompile(`org\.apache\.spark\.rpc\.`)

	// Kotlin-specific patterns.
	kotlinDataClassRe    = regexp.MustCompile(`^data\s+class\s+`)
	kotlinCompanionObjRe = regexp.MustCompile(`^\s*companion\s+object`)
	kotlinExtFunRe       = regexp.MustCompile(`^fun\s+\w+\.\w+`)

	// Maven pom.xml patterns.
	pomDepBlockRe = regexp.MustCompile(`(?s)<dependency>(.*?)</dependency>`)
	pomGroupRe    = regexp.MustCompile(`<groupId>\s*(.*?)\s*</groupId>`)
	pomArtifactRe = regexp.MustCompile(`<artifactId>\s*(.*?)\s*</artifactId>`)
	pomScopeRe    = regexp.MustCompile(`<scope>\s*(.*?)\s*</scope>`)
	pomModuleRe   = regexp.MustCompile(`<module>\s*(.*?)\s*</module>`)
	pomModulesRe  = regexp.MustCompile(`(?s)<modules>(.*?)</modules>`)

	// Gradle dependency patterns (Groovy DSL: single/double quotes; Kotlin DSL: parentheses).
	gradleDepRe = regexp.MustCompile(
		`(?:implementation|api|compile|compileOnly|runtimeOnly|testImplementation|testRuntimeOnly)\s*\(?` +
			`['"]([a-zA-Z0-9._-]+):([a-zA-Z0-9._-]+)(?::([a-zA-Z0-9._-]+))?['"]\)?`)

	// Gradle settings include patterns.
	gradleIncludeRe       = regexp.MustCompile(`include\s*\(\s*['"]([^'"]+)['"]`)
	gradleIncludeSingleRe = regexp.MustCompile(`include\s+['"]([^'"]+)['"]`)

	// Known source directory patterns (Maven/Gradle standard layout).
	javaSourceDirs = []string{
		"src/main/java",
		"src/test/java",
		"src/main/kotlin",
		"src/test/kotlin",
		"src/main/resources",
		"src/test/resources",
	}
)

// ---------------------------------------------------------------------------
// Spring annotation definitions
// ---------------------------------------------------------------------------

// springAnnotations maps annotation names to the framework evidence string.
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

// lombokAnnotationLabels maps Lombok annotations to descriptions.
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

// ---------------------------------------------------------------------------
// JavaExtractor
// ---------------------------------------------------------------------------

// JavaExtractor performs regex-based extraction of Java and Kotlin projects.
type JavaExtractor struct{}

// Name returns the extractor identifier.
func (e *JavaExtractor) Name() string { return "java" }

// Extract walks rootDir, collecting Java/Kotlin imports, parsing build files,
// detecting frameworks (Spring Boot, Lombok), extracting endpoints, and
// identifying multi-module Maven/Gradle layouts.
func (e *JavaExtractor) Extract(rootDir string) (*JavaExtractionResult, error) {
	result := &JavaExtractionResult{
		Language:         "java/kotlin",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.70,
		ImportGraph: JavaImportGraph{
			PackageImports: make(map[string][]string),
		},
		PackageStructure: &PackageStructure{},
		Metadata:         make(map[string]string),
	}

	// Pre-scan: detect package structure and build tool.
	result.PackageStructure = detectPackageStructure(rootDir)

	// Pre-scan: parse settings.gradle for Gradle multi-project.
	gradleSubprojects := parseSettingsGradle(rootDir)
	if len(gradleSubprojects) > 0 {
		result.PackageStructure.MultiModule = true
		result.Modules = append(result.Modules, gradleSubprojects...)
		result.Metadata["gradle_multi_project"] = "true"
	}

	// Collect unique class-level RequestMapping values for endpoint prefixing.
	classLevelMappings := make(map[string]string)  // file -> prefix path
	classLevelMappingLines := make(map[string]int) // file -> line number of class-level mapping
	// Collect all annotations for framework detection.
	allAnnotations := make(map[string]bool)
	// Collect Lombok annotations.
	lombokAnnotations := make(map[string]bool)
	// Source file counters.
	javaFiles := 0
	kotlinFiles := 0
	foundSource := false

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		// Skip known non-source directories.
		if info.IsDir() {
			if javaSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}

		switch {
		case isJavaFile(rel):
			foundSource = true
			javaFiles++
			imports, annotations, pkgDecl, scanErr := scanJavaFile(path, rel, &result.RuntimeCouplings)
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

			// Detect class-level @RequestMapping for endpoint prefixing.
			prefix, prefixLine := extractClassLevelMapping(path)
			if prefix != "" {
				classLevelMappings[rel] = prefix
				classLevelMappingLines[rel] = prefixLine
			}

			// Extract Spring endpoints.
			endpoints := extractSpringEndpoints(path, rel, classLevelMappings[rel], classLevelMappingLines[rel])
			result.SpringEndpoints = append(result.SpringEndpoints, endpoints...)

		case isKotlinFile(rel):
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
				if lombokAnnotationLabels[a.Annotation] != "" {
					lombokAnnotations[a.Annotation] = true
				}
			}
			if pkgDecl != "" {
				recordRootPackage(result.PackageStructure, pkgDecl)
			}

			// Detect Kotlin-specific patterns.
			detectKotlinPatterns(path, result)

		case filepath.Base(rel) == "pom.xml":
			bs, parseErr := parsePomXML(path)
			if parseErr == nil && bs != nil {
				if result.BuildSystem == nil {
					result.BuildSystem = bs
					result.PackageStructure.BuildTool = "maven"
				} else {
					result.BuildSystem.Dependencies = append(
						result.BuildSystem.Dependencies, bs.Dependencies...)
				}
			}
			// Check for multi-module Maven project.
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
		return nil, fmt.Errorf("walk %s: %w", rootDir, err)
	}

	if !foundSource {
		return nil, fmt.Errorf("no Java or Kotlin source files found in %s", rootDir)
	}

	// Deduplicate imports within each package directory.
	for pkg, imports := range result.ImportGraph.PackageImports {
		result.ImportGraph.PackageImports[pkg] = dedup(imports)
	}

	// Set Kotlin flag.
	result.PackageStructure.HasKotlin = kotlinFiles > 0
	if kotlinFiles > 0 {
		result.Metadata["kotlin_files"] = fmt.Sprintf("%d", kotlinFiles)
	}
	if javaFiles > 0 {
		result.Metadata["java_files"] = fmt.Sprintf("%d", javaFiles)
	}

	// Detect frameworks from collected imports AND annotations.
	result.Frameworks = detectJavaFrameworks(result.ImportGraph, allAnnotations, lombokAnnotations)

	// Sort endpoints for deterministic output.
	sort.Slice(result.SpringEndpoints, func(i, j int) bool {
		if result.SpringEndpoints[i].Path != result.SpringEndpoints[j].Path {
			return result.SpringEndpoints[i].Path < result.SpringEndpoints[j].Path
		}
		return result.SpringEndpoints[i].HTTPMethod < result.SpringEndpoints[j].HTTPMethod
	})

	// Deduplicate modules.
	if len(result.Modules) > 0 {
		result.Modules = dedup(result.Modules)
	}

	// Finalize root packages.
	result.PackageStructure.RootPackages = dedup(result.PackageStructure.RootPackages)
	sort.Strings(result.PackageStructure.RootPackages)

	return result, nil
}

// ---------------------------------------------------------------------------
// Import and annotation scanning
// ---------------------------------------------------------------------------

// scanJavaImports reads only import statements from a Java file (lightweight).
func scanJavaImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := javaImportRe.FindStringSubmatch(line)
		if m != nil && len(m) > 2 {
			imports = append(imports, m[2])
		}
	}
	return imports, scanner.Err()
}

// scanJavaFile performs full scanning of a Java file: imports, annotations,
// and package declaration.
func scanJavaFile(path, relPath string, runtimeCouplings ...*[]RuntimeCouplingSighting) ([]string, []AnnotationSighting, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer f.Close()

	return scanJavaFromReader(f, relPath, runtimeCouplings...)
}

// scanJavaFromReader reads from a bufio-ready reader and extracts imports,
// annotations, and the package declaration.
func scanJavaFromReader(f *os.File, relPath string, runtimeCouplings ...*[]RuntimeCouplingSighting) ([]string, []AnnotationSighting, string, error) {
	var imports []string
	var annotations []AnnotationSighting
	var packageDecl string
	lineNum := 0
	var couplingSink *[]RuntimeCouplingSighting
	if len(runtimeCouplings) > 0 {
		couplingSink = runtimeCouplings[0]
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}

		if couplingSink != nil {
			if reNettyRpcEnv.MatchString(line) {
				*couplingSink = append(*couplingSink, RuntimeCouplingSighting{
					Type:     "netty_rpc",
					File:     relPath,
					Line:     lineNum,
					Evidence: line,
				})
			}
			if reGatewayServer.MatchString(line) {
				*couplingSink = append(*couplingSink, RuntimeCouplingSighting{
					Type:     "py4j_gateway",
					File:     relPath,
					Line:     lineNum,
					Evidence: line,
				})
			}
			if reRpcImport.MatchString(line) {
				*couplingSink = append(*couplingSink, RuntimeCouplingSighting{
					Type:     "spark_rpc",
					File:     relPath,
					Line:     lineNum,
					Evidence: line,
				})
			}
		}

		// Package declaration.
		if m := javaPackageRe.FindStringSubmatch(line); m != nil {
			packageDecl = m[1]
			continue
		}

		// Import statements.
		if m := javaImportRe.FindStringSubmatch(line); m != nil && len(m) > 2 {
			imports = append(imports, m[2])
			continue
		}

		// Spring annotations.
		for _, match := range springAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, AnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// @Bean detection.
		if springBeanRe.MatchString(line) {
			annotations = append(annotations, AnnotationSighting{
				Annotation: "@Bean",
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// Lombok annotations.
		for _, match := range lombokAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, AnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}
	}

	return imports, annotations, packageDecl, scanner.Err()
}

// scanKotlinImports reads only import statements from a Kotlin file (lightweight).
func scanKotlinImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := kotlinImportRe.FindStringSubmatch(line)
		if m != nil {
			imports = append(imports, m[1])
		}
	}
	return imports, scanner.Err()
}

// scanKotlinFile performs full scanning of a Kotlin file: imports, annotations,
// and package declaration.
func scanKotlinFile(path, relPath string) ([]string, []AnnotationSighting, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer f.Close()

	var imports []string
	var annotations []AnnotationSighting
	var packageDecl string
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}

		// Package declaration.
		if m := kotlinPackageRe.FindStringSubmatch(line); m != nil {
			packageDecl = m[1]
			continue
		}

		// Import statements.
		if m := kotlinImportRe.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
			continue
		}

		// Spring annotations (Kotlin uses same annotations).
		for _, match := range springAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, AnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// @Bean detection.
		if springBeanRe.MatchString(line) {
			annotations = append(annotations, AnnotationSighting{
				Annotation: "@Bean",
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// Lombok is not used in Kotlin, but check anyway for mixed projects.
	}

	return imports, annotations, packageDecl, scanner.Err()
}

// ---------------------------------------------------------------------------
// Kotlin-specific pattern detection
// ---------------------------------------------------------------------------

// detectKotlinPatterns scans a Kotlin file for data classes, companion objects,
// and extension functions.
func detectKotlinPatterns(path string, result *JavaExtractionResult) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		if kotlinDataClassRe.MatchString(line) {
			result.Metadata["kotlin_data_class"] = "true"
		}
		if kotlinCompanionObjRe.MatchString(line) {
			result.Metadata["kotlin_companion_object"] = "true"
		}
		if kotlinExtFunRe.MatchString(line) {
			result.Metadata["kotlin_extension_function"] = "true"
		}
	}
}

// ---------------------------------------------------------------------------
// Package structure detection
// ---------------------------------------------------------------------------

// detectPackageStructure checks for standard Maven/Gradle source layouts.
func detectPackageStructure(rootDir string) *PackageStructure {
	ps := &PackageStructure{}

	for _, dir := range javaSourceDirs {
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

// recordRootPackage extracts the root package (first two segments) from a
// full package declaration and records it in the PackageStructure if not
// already present.
func recordRootPackage(ps *PackageStructure, pkgDecl string) {
	if ps == nil || pkgDecl == "" {
		return
	}
	segments := strings.Split(pkgDecl, ".")
	root := pkgDecl
	if len(segments) >= 2 {
		root = segments[0] + "." + segments[1]
	}
	// Check for duplicates.
	for _, existing := range ps.RootPackages {
		if existing == root {
			return
		}
	}
	ps.RootPackages = append(ps.RootPackages, root)
}

// ---------------------------------------------------------------------------
// Spring endpoint extraction
// ---------------------------------------------------------------------------

// extractSpringEndpoints scans a file for Spring endpoint annotations and
// returns the detected endpoints. classPrefix is the class-level @RequestMapping
// value (if any). classPrefixLine is the line number where the class-level
// mapping was found (0 if none); lines at that number are skipped.
func extractSpringEndpoints(path, relPath, classPrefix string, classPrefixLine int) []SpringEndpoint {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var endpoints []SpringEndpoint
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}

		// Skip the class-level @RequestMapping line (already extracted as prefix).
		if lineNum == classPrefixLine {
			continue
		}

		if m := getMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: "GET",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := postMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: "POST",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := putMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: "PUT",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := deleteMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: "DELETE",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := patchMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: "PATCH",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}

		// @RequestMapping at method level — default to GET unless method specified.
		if m := requestMappingRe.FindStringSubmatch(line); m != nil {
			method := "GET" // default
			if strings.Contains(line, "method") {
				if strings.Contains(line, "POST") {
					method = "POST"
				} else if strings.Contains(line, "PUT") {
					method = "PUT"
				} else if strings.Contains(line, "DELETE") {
					method = "DELETE"
				} else if strings.Contains(line, "PATCH") {
					method = "PATCH"
				}
			}
			endpoints = append(endpoints, SpringEndpoint{
				HTTPMethod: method,
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
		}
	}

	return endpoints
}

// extractClassLevelMapping returns the path and line number from a class-level
// @RequestMapping annotation in the file. Returns ("", 0) if none found.
func extractClassLevelMapping(path string) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()

	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		// Look for @RequestMapping at class level (before any method definitions).
		if m := classLevelMappingRe.FindStringSubmatch(line); m != nil {
			return m[1], lineNum
		}
		// Once we hit a method definition, stop looking for class-level mapping.
		if strings.Contains(line, "public ") && strings.Contains(line, "(") && !strings.Contains(line, "@") {
			break
		}
	}
	return "", 0
}

// joinPaths combines a class-level prefix with a method-level path.
func joinPaths(prefix, suffix string) string {
	if prefix == "" {
		return suffix
	}
	if suffix == "" || suffix == "/" {
		return prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")
	suffix = strings.TrimPrefix(suffix, "/")
	return prefix + "/" + suffix
}

// ---------------------------------------------------------------------------
// pom.xml parsing (regex-based, no XML library)
// ---------------------------------------------------------------------------

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
// settings.gradle parsing (Gradle multi-project detection)
// ---------------------------------------------------------------------------

// parseSettingsGradle reads settings.gradle or settings.gradle.kts and
// extracts include directives to identify subprojects.
func parseSettingsGradle(rootDir string) []string {
	var subprojects []string
	for _, filename := range []string{"settings.gradle", "settings.gradle.kts"} {
		path := filepath.Join(rootDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		// include "project-name" style
		for _, m := range gradleIncludeRe.FindAllStringSubmatch(content, -1) {
			subprojects = append(subprojects, m[1])
		}
		// include 'project-name' style (single quote, no parens)
		for _, m := range gradleIncludeSingleRe.FindAllStringSubmatch(content, -1) {
			subprojects = append(subprojects, m[1])
		}
		break // only read the first one found
	}
	return subprojects
}

// ---------------------------------------------------------------------------
// Framework detection (Spring Boot, Lombok)
// ---------------------------------------------------------------------------

func detectJavaFrameworks(ig JavaImportGraph, annotations map[string]bool, lombokAnnotations map[string]bool) []JavaFramework {
	var frameworks []JavaFramework

	// Collect all imports into a single set for efficient lookup.
	allImports := make(map[string]bool)
	for _, imports := range ig.PackageImports {
		for _, imp := range imports {
			allImports[imp] = true
		}
	}

	// Spring Boot detection: check for Spring-related import prefixes AND
	// annotation sightings from source file scanning.
	springEvidence := detectSpringEvidence(allImports, annotations)
	if len(springEvidence) > 0 {
		confidence := float64(len(springEvidence)) / float64(len(springAnnotations))
		if confidence > 1.0 {
			confidence = 1.0
		}
		// Minimum confidence of 0.5 if we have any evidence.
		if confidence < 0.5 {
			confidence = 0.5
		}
		frameworks = append(frameworks, JavaFramework{
			Name:       "Spring Boot",
			Confidence: confidence,
			Evidence:   springEvidence,
		})
	}

	// Lombok detection.
	if len(lombokAnnotations) > 0 {
		var evidence []string
		for annot := range lombokAnnotations {
			if desc, ok := lombokAnnotationLabels[annot]; ok {
				evidence = append(evidence, desc)
			}
		}
		sort.Strings(evidence)
		confidence := float64(len(lombokAnnotations)) / float64(len(lombokAnnotationLabels))
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.5 {
			confidence = 0.5
		}
		frameworks = append(frameworks, JavaFramework{
			Name:       "Lombok",
			Confidence: confidence,
			Evidence:   evidence,
		})
	}

	// Hibernate/JPA detection via imports.
	hibernateEvidence := detectHibernateEvidence(allImports)
	if len(hibernateEvidence) > 0 {
		frameworks = append(frameworks, JavaFramework{
			Name:       "Hibernate",
			Confidence: 0.8,
			Evidence:   hibernateEvidence,
		})
	}

	return frameworks
}

// detectSpringEvidence scans the import set and annotations for Spring-related patterns.
func detectSpringEvidence(imports map[string]bool, annotations map[string]bool) []string {
	var evidence []string
	seen := make(map[string]bool)

	// Evidence from import paths.
	springImportPrefixes := map[string]string{
		"org.springframework.web.bind.annotation":                      "@RestController",
		"org.springframework.stereotype.Service":                       "@Service",
		"org.springframework.stereotype.Repository":                    "@Repository",
		"org.springframework.stereotype.Component":                     "@Component",
		"org.springframework.context.annotation.Configuration":         "@Configuration",
		"org.springframework.boot.autoconfigure.SpringBootApplication": "@SpringBootApplication",
		"org.springframework.boot.SpringApplication":                   "@SpringBootApplication",
		"org.springframework.context.annotation.Bean":                  "@Bean",
		"javax.persistence.Entity":                                     "@Entity",
		"jakarta.persistence.Entity":                                   "@Entity",
	}

	for imp := range imports {
		for prefix, annotation := range springImportPrefixes {
			if strings.HasPrefix(imp, prefix) || imp == prefix {
				if !seen[annotation] {
					seen[annotation] = true
					if desc, ok := springAnnotations[annotation]; ok {
						evidence = append(evidence, desc)
					}
				}
			}
		}
	}

	// Evidence from directly-observed annotations in source files.
	// Use a separate seen map so import evidence and annotation evidence
	// are both emitted.
	annotSeen := make(map[string]bool)
	for annot := range annotations {
		if desc, ok := springAnnotations[annot]; ok {
			if !annotSeen[annot] {
				annotSeen[annot] = true
				evidence = append(evidence, desc+" (annotation in source)")
			}
		}
	}

	sort.Strings(evidence)
	return evidence
}

// detectHibernateEvidence scans imports for Hibernate/JPA patterns.
func detectHibernateEvidence(imports map[string]bool) []string {
	var evidence []string
	seen := make(map[string]bool)

	hibernatePatterns := map[string]string{
		"org.hibernate":                "Hibernate session/entity imports",
		"javax.persistence":            "JPA persistence annotations",
		"jakarta.persistence":          "Jakarta persistence annotations",
		"org.springframework.data.jpa": "Spring Data JPA",
	}

	for imp := range imports {
		for prefix, desc := range hibernatePatterns {
			if strings.HasPrefix(imp, prefix) && !seen[prefix] {
				seen[prefix] = true
				evidence = append(evidence, desc)
			}
		}
	}

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
	return strings.HasSuffix(rel, ".kt")
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

// scanImportsWithRegex is a generic helper that scans a file for imports
// matching a given regex pattern. Provided for backwards compatibility.
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

// Ensure JavaExtractor satisfies architect.Extractor at compile time (optional,
// since the adapter pattern is used instead).
var _ architect.Extractor = (*JavaAdapter)(nil)
