package typescript

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// tsFrameworkSignals maps dependency names to framework metadata.
var tsFrameworkSignals = []struct {
	Name       string
	DepName    string
	Confidence float64
	FileCheck  string // optional config file basename
}{
	{Name: "Next.js", DepName: "next", Confidence: 0.90, FileCheck: "next.config.js"},
	{Name: "React", DepName: "react", Confidence: 0.90},
	{Name: "Express", DepName: "express", Confidence: 0.90},
	{Name: "NestJS", DepName: "@nestjs/core", Confidence: 0.90},
	{Name: "Vue", DepName: "vue", Confidence: 0.90},
	{Name: "Angular", DepName: "@angular/core", Confidence: 0.90, FileCheck: "angular.json"},
	{Name: "Svelte", DepName: "svelte", Confidence: 0.90},
	{Name: "Nuxt", DepName: "nuxt", Confidence: 0.90},
	{Name: "Fastify", DepName: "fastify", Confidence: 0.85},
	{Name: "Koa", DepName: "koa", Confidence: 0.85},
	{Name: "Hapi", DepName: "@hapi/hapi", Confidence: 0.85},
}

// Framework detection regexes.
var (
	reNestModule     = regexp.MustCompile(`@Module\s*\(`)
	reNestController = regexp.MustCompile(`@Controller\s*\(`)
	reNestInjectable = regexp.MustCompile(`@Injectable\s*\(`)
	reExpressApp     = regexp.MustCompile(`(?:app|router)\.(get|post|put|delete|patch|use)\s*\(`)
	reReactComponent = regexp.MustCompile(`(?:function|const)\s+\w+\s*(?:=\s*)?=>\s*(?:<[\w\s,]*>)?\s*\(|export\s+(?:default\s+)?(?:function|class|const)`)
	reVueComponent   = regexp.MustCompile(`export\s+default\s+(?:defineComponent|\{\s*name:)`)
	reAngularInject  = regexp.MustCompile(`@Injectable\s*\(`)
)

// detectTSFrameworksV2 scans dependencies, config files, and source patterns.
func detectTSFrameworksV2(rootDir string, deps []TSDependencyEntry, nodeMap map[string]*TSPackageNode) []TSDetectedFramework {
	depSet := make(map[string]string, len(deps))
	for _, d := range deps {
		depSet[d.Name] = d.Version
	}

	var frameworks []TSDetectedFramework
	seen := make(map[string]bool)

	for _, sig := range tsFrameworkSignals {
		if seen[sig.Name] {
			continue
		}

		confidence := 0.0
		var evidence []string

		// Check dependency presence.
		if _, ok := depSet[sig.DepName]; ok {
			confidence += 0.5
			evidence = append(evidence, sig.DepName+" in dependencies")
		}

		// Check config file.
		if sig.FileCheck != "" {
			// Check common variants (.js, .mjs, .cjs, .ts).
			base := strings.TrimSuffix(sig.FileCheck, filepath.Ext(sig.FileCheck))
			for _, ext := range []string{".js", ".mjs", ".cjs", ".ts"} {
				if fileExists(filepath.Join(rootDir, base+ext)) {
					confidence += 0.3
					evidence = append(evidence, base+ext+" config found")
					break
				}
			}
		}

		// Framework-specific additional signals.
		switch sig.Name {
		case "Next.js":
			if dirExists(filepath.Join(rootDir, "pages")) || dirExists(filepath.Join(rootDir, "app")) {
				confidence += 0.2
				evidence = append(evidence, "pages/ or app/ directory present")
			}
		case "React":
			if hasFileWithExtension(rootDir, ".tsx") || hasFileWithExtension(rootDir, ".jsx") {
				confidence += 0.2
				evidence = append(evidence, "TSX/JSX files found")
			}
		case "NestJS":
			if scanForPattern(rootDir, reNestModule) || scanForPattern(rootDir, reNestController) || scanForPattern(rootDir, reNestInjectable) {
				confidence += 0.3
				evidence = append(evidence, "@Module/@Controller/@Injectable decorators found")
			}
		case "Express":
			if scanForPattern(rootDir, reExpressApp) {
				confidence += 0.3
				evidence = append(evidence, "app.get/post/use patterns found")
			}
		case "Svelte":
			if hasFileWithExtension(rootDir, ".svelte") {
				confidence += 0.3
				evidence = append(evidence, ".svelte files found")
			}
			if fileExists(filepath.Join(rootDir, "svelte.config.js")) {
				confidence += 0.2
				evidence = append(evidence, "svelte.config.js found")
			}
		case "Vue":
			if hasFileWithExtension(rootDir, ".vue") {
				confidence += 0.2
				evidence = append(evidence, ".vue files found")
			}
			if scanForPattern(rootDir, reVueComponent) {
				confidence += 0.2
				evidence = append(evidence, "defineComponent patterns found")
			}
		case "Angular":
			if fileExists(filepath.Join(rootDir, "angular.json")) {
				confidence += 0.3
				evidence = append(evidence, "angular.json found")
			}
			if scanForPattern(rootDir, reAngularInject) {
				confidence += 0.2
				evidence = append(evidence, "@Injectable decorators found")
			}
		}

		// Only report frameworks with at least some evidence.
		if confidence > 0.2 {
			frameworks = append(frameworks, TSDetectedFramework{
				Name:       sig.Name,
				Confidence: minFloat(confidence, 1.0),
				Evidence:   strings.Join(evidence, "; "),
			})
			seen[sig.Name] = true
		}
	}

	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].Name < frameworks[j].Name
	})
	return frameworks
}

// scanForPattern walks TS/JS files and returns true if any line matches the regex.
func scanForPattern(rootDir string, re *regexp.Regexp) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !extensions[filepath.Ext(path)] {
			return nil
		}
		f, fErr := os.Open(path)
		if fErr != nil {
			return nil
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

// minFloat returns the smaller of two floats.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
