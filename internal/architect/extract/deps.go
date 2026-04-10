package extract

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// manifestSpec describes a dependency manifest file: its basename, the
// language it implies, and how to count dependencies plus detect notable ones.
type manifestSpec struct {
	File     string
	Language string
	Counter  func(path string) (int, []string, error)
}

// notableSignals maps dependency name substrings to architectural signal
// labels.  When a dependency contains one of these substrings (case-
// insensitive) the corresponding signal is emitted.
var notableSignals = map[string]string{
	"kafka":      "event_driven",
	"nats":       "event_driven",
	"rabbitmq":   "event_driven",
	"amqp":       "event_driven",
	"prisma":     "orm",
	"gorm":       "orm",
	"sequelize":  "orm",
	"sqlalchemy": "orm",
	"typeorm":    "orm",
	"hibernate":  "orm",
	"grpc":       "grpc",
	"graphql":    "graphql",
	"redis":      "cache",
	"memcached":  "cache",
	"prometheus": "observability",
	"opentelemetry": "observability",
	"jaeger":     "observability",
	"docker":     "container",
	"kubernetes": "container",
	"terraform":  "iac",
	"pulumi":     "iac",
}

// manifests lists the dependency files we know how to parse.
var manifests = []manifestSpec{
	{File: "go.mod", Language: "go", Counter: countGoMod},
	{File: "package.json", Language: "javascript", Counter: countPackageJSON},
	{File: "requirements.txt", Language: "python", Counter: countLineFile},
	{File: "pyproject.toml", Language: "python", Counter: countLineFile},
	{File: "Cargo.toml", Language: "rust", Counter: countLineFile},
	{File: "pom.xml", Language: "java", Counter: countXMLDeps},
	{File: "build.gradle", Language: "java", Counter: countLineFile},
	{File: "build.gradle.kts", Language: "kotlin", Counter: countLineFile},
	{File: "Gemfile", Language: "ruby", Counter: countGemfile},
}

// DependencyManifestParser detects dependency manifest files and extracts
// language, dependency count, and notable signals from each.
type DependencyManifestParser struct{}

// Name implements architect.Extractor.
func (DependencyManifestParser) Name() string { return "deps" }

// Extract implements architect.Extractor.
func (DependencyManifestParser) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	var deps []architect.DependencyInfo

	for _, m := range manifests {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		path := filepath.Join(repoRoot, m.File)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		count, signals, err := m.Counter(path)
		if err != nil {
			// Non-fatal: skip manifests we cannot parse.
			continue
		}
		deps = append(deps, architect.DependencyInfo{
			File:     m.File,
			Language: m.Language,
			DepCount: count,
			Signals:  signals,
		})
	}

	// Walk for *.csproj files (C# projects) since they are not fixed-name.
	csprojDeps, err := scanCsprojFiles(ctx, repoRoot)
	if err != nil {
		// Non-fatal: continue with other deps.
		return &architect.ProfileFragment{Dependencies: deps}, nil
	}
	deps = append(deps, csprojDeps...)

	return &architect.ProfileFragment{
		Dependencies: deps,
	}, nil
}

// ---------------------------------------------------------------------------
// Counting helpers
// ---------------------------------------------------------------------------

// countGoMod counts `require` directives in a go.mod file.
func countGoMod(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count   int
		inBlock bool
		seen    = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "require (") || line == "require (" {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if line != "" && !strings.HasPrefix(line, "//") {
				count++
				detectSignals(line, seen)
			}
			continue
		}
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
			count++
			detectSignals(line, seen)
		}
	}
	return count, mapKeys(seen), sc.Err()
}

// countPackageJSON does a simple line-scan for dependency entries. It counts
// lines inside "dependencies" and "devDependencies" blocks by looking for
// colon-separated key-value pairs.
func countPackageJSON(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count   int
		inDeps  bool
		seen    = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.Contains(line, `"dependencies"`) || strings.Contains(line, `"devDependencies"`) {
			inDeps = true
			continue
		}
		if inDeps {
			if strings.Contains(line, "}") {
				inDeps = false
				continue
			}
			if strings.Contains(line, ":") {
				count++
				detectSignals(line, seen)
			}
		}
	}
	return count, mapKeys(seen), sc.Err()
}

// countLineFile counts non-blank, non-comment lines. Used for
// requirements.txt, pyproject.toml, Cargo.toml, build.gradle.
func countLineFile(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count int
		seen  = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		count++
		detectSignals(line, seen)
	}
	return count, mapKeys(seen), sc.Err()
}

// countXMLDeps counts <dependency> tags in a pom.xml.
func countXMLDeps(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count int
		seen  = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.Contains(line, "<dependency>") {
			count++
		}
		detectSignals(line, seen)
	}
	return count, mapKeys(seen), sc.Err()
}

// detectSignals checks a line for notable dependency keywords.
func detectSignals(line string, seen map[string]bool) {
	lower := strings.ToLower(line)
	for substr, signal := range notableSignals {
		if strings.Contains(lower, substr) {
			seen[signal] = true
		}
	}
}

// mapKeys returns sorted keys of a bool map.
func mapKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

// countGemfile counts gem lines in a Ruby Gemfile, skipping comments,
// blanks, source/group/group-end lines.
func countGemfile(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count int
		seen  = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "source ") ||
			strings.HasPrefix(line, "ruby ") || strings.HasPrefix(line, "group ") ||
			line == "end" || strings.HasPrefix(line, "git ") || strings.HasPrefix(line, "platforms:") {
			continue
		}
		// Only count lines that look like gem declarations or similar.
		if strings.HasPrefix(line, "gem ") || strings.Contains(line, ",") {
			count++
			detectSignals(line, seen)
		}
	}
	return count, mapKeys(seen), sc.Err()
}

// scanCsprojFiles walks the repo for *.csproj files and counts
// <PackageReference> tags in each.
func scanCsprojFiles(ctx context.Context, repoRoot string) ([]architect.DependencyInfo, error) {
	var deps []architect.DependencyInfo

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".csproj") {
			return nil
		}

		count, signals, parseErr := countCsproj(path)
		if parseErr != nil {
			return nil // Non-fatal
		}

		rel, _ := filepath.Rel(repoRoot, path)
		deps = append(deps, architect.DependencyInfo{
			File:     rel,
			Language: "csharp",
			DepCount: count,
			Signals:  signals,
		})
		return nil
	})
	return deps, err
}

// countCsproj counts <PackageReference> tags in a .csproj file.
func countCsproj(path string) (int, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer f.Close()

	var (
		count int
		seen  = make(map[string]bool)
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.Contains(line, "<PackageReference") {
			count++
			detectSignals(line, seen)
		}
	}
	return count, mapKeys(seen), sc.Err()
}
