package extract

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sdp_dev/internal/architect"
)

// Known Python standard library top-level modules (subset for heuristic classification).
var pythonStdlib = map[string]bool{
	"abc": true, "argparse": true, "ast": true, "asyncio": true,
	"base64": true, "binascii": true, "bisect": true,
	"calendar": true, "cgi": true, "collections": true, "colorsys": true,
	"concurrent": true, "configparser": true, "contextlib": true, "copy": true,
	"csv": true, "ctypes": true, "curses": true,
	"dataclasses": true, "datetime": true, "dbm": true, "decimal": true,
	"difflib": true, "dis": true, "distutils": true,
	"email": true, "encodings": true, "enum": true, "errno": true,
	"fcntl": true, "filecmp": true, "fileinput": true, "fnmatch": true,
	"fractions": true, "ftplib": true, "functools": true,
	"gc": true, "getopt": true, "getpass": true, "gettext": true,
	"glob": true, "grp": true, "gzip": true,
	"hashlib": true, "heapq": true, "hmac": true, "html": true, "http": true,
	"imaplib": true, "importlib": true, "inspect": true, "io": true,
	"ipaddress": true, "itertools": true,
	"json": true,
	"keyword": true,
	"linecache": true, "locale": true, "logging": true, "lzma": true,
	"mailbox": true, "math": true, "mimetypes": true, "mmap": true,
	"multiprocessing": true,
	"numbers": true,
	"operator": true, "optparse": true, "os": true,
	"pathlib": true, "pdb": true, "pickle": true, "pkgutil": true,
	"platform": true, "plistlib": true, "poplib": true, "posixpath": true,
	"pprint": true, "profile": true, "pstats": true, "pty": true,
	"pwd": true, "py_compile": true, "pydoc": true,
	"queue": true, "quopri": true,
	"random": true, "re": true, "readline": true, "reprlib": true,
	"resource": true, "rlcompleter": true, "runpy": true,
	"sched": true, "secrets": true, "select": true, "selectors": true,
	"shelve": true, "shlex": true, "shutil": true, "signal": true, "site": true,
	"smtplib": true, "socket": true, "socketserver": true, "sqlite3": true,
	"ssl": true, "stat": true, "statistics": true, "string": true,
	"struct": true, "subprocess": true, "sys": true, "sysconfig": true, "syslog": true,
	"tarfile": true, "tempfile": true, "termios": true, "test": true,
	"textwrap": true, "threading": true, "time": true, "timeit": true,
	"tkinter": true, "token": true, "tokenize": true, "tomllib": true,
	"trace": true, "traceback": true, "tracemalloc": true, "tty": true,
	"types": true, "typing": true,
	"unicodedata": true, "unittest": true, "urllib": true, "uuid": true,
	"venv": true,
	"warnings": true, "wave": true, "weakref": true, "webbrowser": true,
	"xml": true, "xmlrpc": true,
	"zipfile": true, "zipimport": true, "zlib": true,
	"_thread": true, "__future__": true,
}

// pythonSkipDirs lists directories to skip when walking a Python project.
var pythonSkipDirs = map[string]bool{
	"venv":          true,
	".venv":         true,
	"env":           true,
	"__pycache__":   true,
	"node_modules":  true,
	".git":          true,
	".tox":          true,
	".mypy_cache":   true,
	".pytest_cache": true,
}

var (
	reAbsoluteImport = regexp.MustCompile(`^import\s+(\S+)`)
	reFromImport     = regexp.MustCompile(`^from\s+(\S+)\s+import\s+(\S+)`)

	// Framework detection patterns.
	reFlaskRoute   = regexp.MustCompile(`@app\.route\(`)
	reFastAPIRoute = regexp.MustCompile(`@app\.(get|post|put|delete|patch)\(`)
	reDjangoApps   = regexp.MustCompile(`INSTALLED_APPS\s*=`)

	// requirements.txt line: package==version or package>=version etc.
	reRequirement = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)`)

	// pyproject.toml dependencies line inside [project] or [tool.poetry.dependencies]
	rePyprojectDep = regexp.MustCompile(`^\s*"?([A-Za-z0-9][A-Za-z0-9._-]*)"?\s*[>=<~!]`)
	// Simple key = "version" style (poetry)
	rePoetryDep = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)
)

// PythonExtractor implements architect.Extractor for Python projects using regex.
type PythonExtractor struct{}

// Language returns "python".
func (p *PythonExtractor) Language() string { return "python" }

// Extract walks rootDir, parses .py files for imports, reads requirements.txt
// and pyproject.toml, and detects frameworks.
func (p *PythonExtractor) Extract(ctx context.Context, rootDir string) (*architect.ExtractionResult, error) {
	result := &architect.ExtractionResult{
		Language:         "python",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.55,
	}

	seen := make(map[string]bool)     // dedup key: "source:name"
	frameworks := make(map[string]architect.Framework)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		// Check context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			if pythonSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(rootDir, path)

		switch {
		case strings.HasSuffix(info.Name(), ".py"):
			deps, fws, err := parsePythonFile(path, rel)
			if err != nil {
				return nil // skip unreadable files
			}
			result.FileCount++
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}
			for _, fw := range fws {
				if existing, ok := frameworks[fw.Name]; !ok || fw.Confidence > existing.Confidence {
					frameworks[fw.Name] = fw
				}
			}

		case info.Name() == "requirements.txt":
			deps := parseRequirementsTxt(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}

		case info.Name() == "pyproject.toml":
			deps := parsePyprojectToml(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, fw := range frameworks {
		result.Frameworks = append(result.Frameworks, fw)
	}

	return result, nil
}

// parsePythonFile extracts imports and detects frameworks from a single .py file.
func parsePythonFile(path, relPath string) ([]architect.Dependency, []architect.Framework, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var deps []architect.Dependency
	var fws []architect.Framework

	scanner := bufio.NewScanner(f)
	inTripleQuote := false
	tripleChar := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle triple-quote strings.
		if inTripleQuote {
			if strings.Contains(trimmed, tripleChar) {
				inTripleQuote = false
			}
			continue
		}

		// Check for triple-quote start (not already inside one).
		if countAndToggleTriple(trimmed, &inTripleQuote, &tripleChar) {
			continue
		}

		// Skip single-line comments.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Framework detection on every non-comment, non-string line.
		if reFlaskRoute.MatchString(trimmed) {
			fws = append(fws, architect.Framework{
				Name:       "Flask",
				Confidence: 0.9,
				Evidence:   "@app.route",
			})
		}
		if reFastAPIRoute.MatchString(trimmed) {
			fws = append(fws, architect.Framework{
				Name:       "FastAPI",
				Confidence: 0.9,
				Evidence:   "@app.get/post/put/delete/patch",
			})
		}
		if reDjangoApps.MatchString(trimmed) {
			fws = append(fws, architect.Framework{
				Name:       "Django",
				Confidence: 0.85,
				Evidence:   "INSTALLED_APPS",
			})
		}

		// Import extraction.
		if m := reFromImport.FindStringSubmatch(trimmed); m != nil {
			modName := m[1]
			importedName := m[2]
			dep := resolveImport(modName, importedName, relPath)
			deps = append(deps, dep)
			continue
		}
		if m := reAbsoluteImport.FindStringSubmatch(trimmed); m != nil {
			raw := m[1]
			// Handle "import a, b, c" — split on commas.
			for _, part := range strings.Split(raw, ",") {
				name := strings.TrimSpace(part)
				if name == "" {
					continue
				}
				dep := classifyImport(name)
				deps = append(deps, dep)
			}
		}
	}

	return deps, fws, scanner.Err()
}

// countAndToggleTriple detects triple-quote boundaries. Returns true if the
// line is consumed by (or starts) a triple-quote block.
func countAndToggleTriple(line string, inTriple *bool, tripleChar *string) bool {
	for _, tq := range []string{`"""`, `'''`} {
		count := strings.Count(line, tq)
		if count == 0 {
			continue
		}
		if count == 1 {
			// Either opening or closing. If it's a standalone docstring on one
			// line (e.g., """docstring"""), count would be 2.
			*inTriple = true
			*tripleChar = tq
			return true
		}
		if count%2 == 0 {
			// Even number of triple quotes on one line means they open and close
			// on the same line. The line is consumed (it's a string literal).
			return true
		}
		// Odd number > 1 means one is left open.
		*inTriple = true
		*tripleChar = tq
		return true
	}
	return false
}

// resolveImport resolves a Python import (potentially relative) to an absolute
// module path and classifies it.
//
// modName is the module reference (e.g. ".", "..", ".core"),
// importedName is what follows "import" (e.g. "utils" in "from . import utils"),
// relPath is the file path relative to the project root.
func resolveImport(modName, importedName, relPath string) architect.Dependency {
	if !strings.HasPrefix(modName, ".") {
		return classifyImport(modName)
	}

	// Relative import: count leading dots.
	dots := 0
	for _, ch := range modName {
		if ch == '.' {
			dots++
		} else {
			break
		}
	}
	suffix := modName[dots:] // e.g. "" for ".", "core" for "..core"

	// Determine the package directory from the file's relative path.
	dir := filepath.Dir(relPath)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	// Go up (dots - 1) levels from the current package.
	ups := dots - 1
	if ups > len(parts) {
		ups = len(parts)
	}
	if ups > 0 {
		parts = parts[:len(parts)-ups]
	}

	// Build the resolved module path.
	// If suffix is empty (pure dots like "from . import X"), append importedName.
	if suffix == "" {
		suffix = importedName
	}

	var resolved string
	base := strings.Join(parts, ".")
	if base == "." || base == "" {
		resolved = suffix
	} else if suffix == "" {
		resolved = base
	} else {
		resolved = base + "." + suffix
	}

	if resolved == "" {
		resolved = modName // fallback
	}

	return architect.Dependency{
		Name:   resolved,
		Source: "from-import",
		Kind:   "relative",
	}
}

// classifyImport decides if a module is stdlib, third-party, or local.
func classifyImport(name string) architect.Dependency {
	top := name
	if idx := strings.Index(name, "."); idx > 0 {
		top = name[:idx]
	}

	kind := "third-party"
	if pythonStdlib[top] {
		kind = "stdlib"
	}

	source := "import"
	return architect.Dependency{
		Name:   name,
		Source: source,
		Kind:   kind,
	}
}

// parseRequirementsTxt reads a requirements.txt file.
func parseRequirementsTxt(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if m := reRequirement.FindStringSubmatch(line); m != nil {
			deps = append(deps, architect.Dependency{
				Name:   m[1],
				Source: "requirements.txt",
				Kind:   "third-party",
			})
		}
	}
	return deps
}

// parsePyprojectToml does a best-effort regex parse of pyproject.toml dependencies.
func parsePyprojectToml(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	inDeps := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect dependency sections.
		if strings.HasPrefix(trimmed, "[") {
			lower := strings.ToLower(trimmed)
			inDeps = strings.Contains(lower, "dependencies") ||
				strings.Contains(lower, "tool.poetry.dependencies")
			continue
		}

		if !inDeps {
			continue
		}

		// Array-style: "flask>=2.0",
		if m := rePyprojectDep.FindStringSubmatch(trimmed); m != nil {
			name := strings.Trim(m[1], `"`)
			if name != "" && name != "python" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
			continue
		}

		// Poetry-style: flask = "^2.0"
		if m := rePoetryDep.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			if name != "" && name != "python" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}
