package python

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// ImportRecord represents a single import extracted from a Python file.
type ImportRecord struct {
	Name           string
	Source         string
	Kind           string
	ResolvedModule string
}

// FrameworkRecord represents a detected framework from source analysis.
type FrameworkRecord struct {
	Name       string
	Confidence float64
	Evidence   string
}

// ParseImports extracts imports and detects frameworks from a single .py file.
func ParseImports(path string) ([]ImportRecord, []FrameworkRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var imports []ImportRecord
	var frameworks []FrameworkRecord

	scanner := bufio.NewScanner(f)
	inTripleQuote := false
	tripleChar := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if inTripleQuote {
			if strings.Contains(trimmed, tripleChar) {
				inTripleQuote = false
			}
			continue
		}

		if countAndToggleTriple(trimmed, &inTripleQuote, &tripleChar) {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		detectFrameworksFromLine(trimmed, &frameworks)

		if m := reFromImport.FindStringSubmatch(trimmed); m != nil {
			modName := m[1]
			importedName := m[2]
			rec := resolveImport(modName, importedName)
			imports = append(imports, rec)
			continue
		}

		if m := reAbsoluteImport.FindStringSubmatch(trimmed); m != nil {
			raw := m[1]
			for _, part := range strings.Split(raw, ",") {
				name := strings.TrimSpace(part)
				if name == "" {
					continue
				}
				rec := classifyImport(name)
				imports = append(imports, rec)
			}
		}
	}

	return imports, frameworks, scanner.Err()
}

func resolveImport(modName, importedName string) ImportRecord {
	if !strings.HasPrefix(modName, ".") {
		rec := classifyImport(modName)
		rec.ResolvedModule = modName
		return rec
	}

	dots := 0
	for _, ch := range modName {
		if ch == '.' {
			dots++
		} else {
			break
		}
	}
	suffix := modName[dots:]

	var resolved string
	if suffix == "" {
		resolved = importedName
	} else {
		resolved = suffix + "." + importedName
	}

	if resolved == "" {
		resolved = modName
	}

	return ImportRecord{
		Name:           resolved,
		Source:         "from-import",
		Kind:           "relative",
		ResolvedModule: resolved,
	}
}

func classifyImport(name string) ImportRecord {
	top := name
	if idx := strings.Index(name, "."); idx > 0 {
		top = name[:idx]
	}

	kind := "third-party"
	if stdlib[top] {
		kind = "stdlib"
	}

	return ImportRecord{
		Name:           name,
		Source:         "import",
		Kind:           kind,
		ResolvedModule: name,
	}
}

func detectFrameworksFromLine(trimmed string, frameworks *[]FrameworkRecord) {
	if reFlaskApp.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Flask",
			Confidence: 0.95,
			Evidence:   "Flask app instantiation",
		})
	}
	if reFlaskRoute.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Flask",
			Confidence: 0.90,
			Evidence:   "@app.route decorator",
		})
	}
	if reFlaskBlueprint.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Flask",
			Confidence: 0.85,
			Evidence:   "Blueprint registration",
		})
	}

	if reFastAPIApp.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "FastAPI",
			Confidence: 0.95,
			Evidence:   "FastAPI app instantiation",
		})
	}
	if reFastAPIDecor.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "FastAPI",
			Confidence: 0.90,
			Evidence:   "FastAPI route decorator",
		})
	}
	if reFastAPIRouter.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "FastAPI",
			Confidence: 0.85,
			Evidence:   "APIRouter usage",
		})
	}

	if reDjangoApps.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Django",
			Confidence: 0.95,
			Evidence:   "INSTALLED_APPS",
		})
	}
	if reDjangoURLs.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Django",
			Confidence: 0.90,
			Evidence:   "urlpatterns",
		})
	}
	if reDjangoModel.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Django",
			Confidence: 0.85,
			Evidence:   "Django model class",
		})
	}
	if reDjangoConfig.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Django",
			Confidence: 0.90,
			Evidence:   "AppConfig subclass",
		})
	}

	if reCeleryApp.MatchString(trimmed) {
		*frameworks = append(*frameworks, FrameworkRecord{
			Name:       "Celery",
			Confidence: 0.90,
			Evidence:   "Celery app instantiation",
		})
	}
}

func countAndToggleTriple(line string, inTriple *bool, tripleChar *string) bool {
	for _, tq := range []string{`"""`, `'''`} {
		count := strings.Count(line, tq)
		if count == 0 {
			continue
		}
		if count == 1 {
			*inTriple = true
			*tripleChar = tq
			return true
		}
		if count%2 == 0 {
			return true
		}
		*inTriple = true
		*tripleChar = tq
		return true
	}
	return false
}

var (
	reAbsoluteImport = regexp.MustCompile(`^import\s+(\S+)`)
	reFromImport     = regexp.MustCompile(`^from\s+(\S+)\s+import\s+(\S+)`)
	reFlaskApp       = regexp.MustCompile(`(?:app|application)\s*=\s*Flask\s*\(`)
	reFlaskRoute     = regexp.MustCompile(`@(?:\w+)\.route\s*\(`)
	reFlaskBlueprint = regexp.MustCompile(`Blueprint\s*\(`)
	reFastAPIDecor   = regexp.MustCompile(`@(?:\w+)\.(get|post|put|delete|patch|api_route)\s*\(`)
	reFastAPIApp     = regexp.MustCompile(`(?:app|application)\s*=\s*FastAPI\s*\(`)
	reFastAPIRouter  = regexp.MustCompile(`APIRouter\s*\(`)
	reDjangoApps     = regexp.MustCompile(`INSTALLED_APPS\s*=`)
	reDjangoURLs     = regexp.MustCompile(`urlpatterns\s*=`)
	reDjangoModel    = regexp.MustCompile(`class\s+\w+\s*\(\s*(?:models\.)?Model\s*\)`)
	reDjangoConfig   = regexp.MustCompile(`class\s+\w+Config\s*\(\s*(?:apps\.)?AppConfig\s*\)`)
	reCeleryApp      = regexp.MustCompile(`(?:celery|app)\s*=\s*Celery\s*\(`)
)

var stdlib = map[string]bool{
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
	"json": true, "keyword": true, "linecache": true, "locale": true,
	"logging": true, "lzma": true,
	"mailbox": true, "math": true, "mimetypes": true, "mmap": true,
	"multiprocessing": true, "numbers": true, "operator": true, "optparse": true,
	"os": true, "pathlib": true, "pdb": true, "pickle": true, "pkgutil": true,
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
	"venv": true, "warnings": true, "wave": true, "weakref": true,
	"webbrowser": true, "xml": true, "xmlrpc": true,
	"zipfile": true, "zipimport": true, "zlib": true,
	"_thread": true, "__future__": true,
}
