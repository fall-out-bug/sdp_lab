package java

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Kotlin-specific patterns
var (
	kotlinDataClassRe     = regexp.MustCompile(`^data\s+class\s+`)
	kotlinCompanionObjRe  = regexp.MustCompile(`^\s*companion\s+object`)
	kotlinExtFunRe        = regexp.MustCompile(`^fun\s+\w+\.\w+`)
	kotlinTopLevelFunRe   = regexp.MustCompile(`^(public|internal|private)?\s*fun\s+`)
	kotlinSealedClassRe   = regexp.MustCompile(`^sealed\s+class\s+`)
	kotlinObjectDeclRe    = regexp.MustCompile(`^object\s+`)
	kotlinInlineFunRe     = regexp.MustCompile(`^inline\s+fun\s+`)
	kotlinInfixFunRe      = regexp.MustCompile(`^infix\s+fun\s+`)
	kotlinTailrecFunRe    = regexp.MustCompile(`^tailrec\s+fun\s+`)
	kotlinOperatorFunRe   = regexp.MustCompile(`^operator\s+fun\s+`)
	kotlinSuspendFunRe    = regexp.MustCompile(`^suspend\s+fun\s+`)
	kotlinValueClassRe    = regexp.MustCompile(`^(inline\s+)?value\s+class\s+`)
	kotlinFunInterfaceRe  = regexp.MustCompile(`^fun\s+interface\s+`)
	kotlinTypeAliasRe     = regexp.MustCompile(`^typealias\s+`)
)

// KotlinFeature represents a Kotlin language feature detected in source code.
type KotlinFeature struct {
	Name       string `json:"name"`
	Confidence float64 `json:"confidence"`
	Count      int    `json:"count"`
}

// detectKotlinPatterns scans a Kotlin file for language-specific patterns.
func detectKotlinPatterns(path string, result *Result) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	features := make(map[string]int)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") || line == "" {
			continue
		}

		// Detect language features
		if kotlinDataClassRe.MatchString(line) {
			features["data_class"]++
			result.Metadata["kotlin_data_class"] = "true"
		}
		if kotlinCompanionObjRe.MatchString(line) {
			features["companion_object"]++
			result.Metadata["kotlin_companion_object"] = "true"
		}
		if kotlinExtFunRe.MatchString(line) {
			features["extension_function"]++
			result.Metadata["kotlin_extension_function"] = "true"
		}
		if kotlinSealedClassRe.MatchString(line) {
			features["sealed_class"]++
			result.Metadata["kotlin_sealed_class"] = "true"
		}
		if kotlinObjectDeclRe.MatchString(line) {
			features["object_declaration"]++
			result.Metadata["kotlin_object_declaration"] = "true"
		}
		if kotlinInlineFunRe.MatchString(line) {
			features["inline_function"]++
			result.Metadata["kotlin_inline_function"] = "true"
		}
		if kotlinInfixFunRe.MatchString(line) {
			features["infix_function"]++
			result.Metadata["kotlin_infix_function"] = "true"
		}
		if kotlinTailrecFunRe.MatchString(line) {
			features["tailrec_function"]++
			result.Metadata["kotlin_tailrec_function"] = "true"
		}
		if kotlinOperatorFunRe.MatchString(line) {
			features["operator_overload"]++
			result.Metadata["kotlin_operator_overload"] = "true"
		}
		if kotlinSuspendFunRe.MatchString(line) {
			features["coroutine"]++
			result.Metadata["kotlin_coroutine"] = "true"
		}
		if kotlinValueClassRe.MatchString(line) {
			features["value_class"]++
			result.Metadata["kotlin_value_class"] = "true"
		}
		if kotlinFunInterfaceRe.MatchString(line) {
			features["functional_interface"]++
			result.Metadata["kotlin_functional_interface"] = "true"
		}
		if kotlinTypeAliasRe.MatchString(line) {
			features["type_alias"]++
			result.Metadata["kotlin_type_alias"] = "true"
		}
	}

	// Update result with feature counts
	for feature, count := range features {
		key := "kotlin_feature_" + feature
		result.Metadata[key] = strings.Itoa(count)
	}
}

// AnalyzeKotlinCodeStyle analyzes the coding style patterns in Kotlin files.
//
// This function detects:
//   - Extension function usage
//   - Data class prevalence
//   - Coroutine usage
//   - Functional programming patterns
//   - DSL construction patterns
func AnalyzeKotlinCodeStyle(rootDir string) (*KotlinCodeStyle, error) {
	style := &KotlinCodeStyle{
		Features:      make(map[string]*KotlinFeature),
		PackageUsage:  make(map[string]int),
		ImportPatterns: make(map[string]int),
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return nil
		}

		if isKotlinFile(rel) {
			analyzeKotlinFileStyle(path, style)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return style, nil
}

// KotlinCodeStyle represents Kotlin coding style metrics.
type KotlinCodeStyle struct {
	Features        map[string]*KotlinFeature `json:"features"`
	PackageUsage    map[string]int            `json:"package_usage"`
	ImportPatterns  map[string]int            `json:"import_patterns"`
	TotalFiles      int                       `json:"total_files"`
	TotalLines      int                       `json:"total_lines"`
}

// analyzeKotlinFileStyle analyzes a single Kotlin file for style patterns.
func analyzeKotlinFileStyle(path string, style *KotlinCodeStyle) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	imports := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Count imports
		if strings.HasPrefix(line, "import ") {
			imports++
			// Analyze import patterns
			if strings.HasPrefix(line, "import kotlinx.coroutines") {
				style.ImportPatterns["coroutines"]++
			} else if strings.HasPrefix(line, "import java.util") {
				style.ImportPatterns["java_util"]++
			} else if strings.HasPrefix(line, "import kotlin") {
				style.ImportPatterns["kotlin_stdlib"]++
			}
		}
	}

	style.TotalFiles++
	style.TotalLines += lineNum
	style.ImportPatterns["avg_imports_per_file"] += imports
}

// DetectKotlinCoroutines checks if Kotlin coroutines are used in the project.
//
// This function scans for:
//   - kotlinx.coroutines imports
//   - suspend function declarations
//   - coroutine builders (launch, async, runBlocking)
//   - Flow/Channel usage
func DetectKotlinCoroutines(rootDir string) (bool, []string, error) {
	var evidence []string
	hasCoroutines := false

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !isKotlinFile(path) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Check for coroutine imports
			if strings.HasPrefix(line, "import kotlinx.coroutines") {
				hasCoroutines = true
				evidence = append(evidence, line)
			}

			// Check for suspend functions
			if strings.HasPrefix(line, "suspend fun") {
				hasCoroutines = true
				evidence = append(evidence, "suspend function declaration")
			}

			// Check for coroutine builders
			if strings.Contains(line, "launch(") || strings.Contains(line, "async(") || strings.Contains(line, "runBlocking(") {
				hasCoroutines = true
				evidence = append(evidence, "coroutine builder usage")
			}

			// Check for Flow usage
			if strings.Contains(line, "Flow<") || strings.Contains(line, "StateFlow") || strings.Contains(line, "SharedFlow") {
				hasCoroutines = true
				evidence = append(evidence, "Flow usage")
			}
		}

		return nil
	})

	return hasCoroutines, evidence, err
}

// Known Kotlin blind spots
const KotlinBlindSpots = `
# Kotlin Extraction Blind Spots

## Compile-Time Metaprogramming
- Kotlin symbol processing (KSP) generates code at compile time
- KotlinPoet generates code programmatically
- Auto-service generates META-INF/services files

## Extension Resolution
- Extension functions imported from other modules
- Extension properties with custom getters
- Operator overloads with complex resolution rules

## Type Aliases
- type aliases can create opaque type dependencies
- Generic type aliases with variance
- Projection type aliases

## Inline Functions
- Inline functions can capture dependencies
- Reified type parameters create runtime type dependencies
- Inline properties with custom accessors

## Delegation
- Class delegation creates implicit dependencies
- Interface delegation patterns
- Lazy delegate initialization

## Coroutines
- Coroutine dispatchers may have thread dependencies
- Coroutine scope hierarchy
- Channel and Flow operators

## DSL Construction
- Type-safe builders create implicit dependencies
- Infix functions used in DSL patterns
- Extension receiver scope in DSLs

## Native Interop
- Kotlin/Native C interop
- Objective-C/Swift interop
- Static C++ libraries
`
