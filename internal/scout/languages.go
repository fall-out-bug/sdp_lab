package scout

import (
	"path/filepath"
	"strings"
)

// extToLanguage maps file extensions to language names.
var extToLanguage = map[string]string{
	".go":    "go",
	".ts":    "typescript",
	".tsx":   "typescript",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".cjs":   "javascript",
	".py":    "python",
	".pyi":   "python",
	".java":  "java",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".scala": "scala",
	".rs":    "rust",
	".rb":    "ruby",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".hpp":   "cpp",
	".hxx":   "cpp",
	".cs":    "csharp",
	".swift": "swift",
	".dart":  "dart",
	".lua":   "lua",
	".r":     "r",
	".R":     "r",
	".php":   "php",
	".sh":    "shell",
	".bash":  "shell",
	".zsh":   "shell",
	".fish":  "shell",
	".ps1":   "powershell",
	".ex":    "elixir",
	".exs":   "elixir",
	".erl":   "erlang",
	".hrl":   "erlang",
	".clj":   "clojure",
	".cljs":  "clojure",
	".hs":    "haskell",
	".ml":    "ocaml",
	".mli":   "ocaml",
	".zig":   "zig",
	".nim":   "nim",
	".v":     "vlang",
	".sql":   "sql",
	".html":  "html",
	".css":   "css",
	".scss":  "scss",
	".less":  "less",
	".vue":   "vue",
	".svelte":"svelte",
	".yaml":  "yaml",
	".yml":   "yaml",
	".json":  "json",
	".xml":   "xml",
	".toml":  "toml",
	".ini":   "ini",
	".cfg":   "ini",
	".md":    "markdown",
	".tf":    "terraform",
	".proto": "protobuf",
	".graphql":"graphql",
	".gql":   "graphql",
}

// ExtToLanguage returns the language name for a file extension (e.g. ".go" → "go").
func ExtToLanguage(ext string) (string, bool) {
	lang, ok := extToLanguage[ext]
	return lang, ok
}

// testFilePatterns matches common test file naming conventions.
var testFilePatterns = []string{
	"_test.go",     // Go
	".test.ts",     // TypeScript/JavaScript
	".test.tsx",
	".test.js",
	".test.jsx",
	".spec.ts",
	".spec.tsx",
	".spec.js",
	".spec.jsx",
	"test_",        // Python
	"_test.py",
	"Test.java",    // Java
	"Tests.java",
	"Spec.scala",   // Scala
	"Test.scala",
	"Test.kt",      // Kotlin
	"Tests.kt",
	"_test.rs",     // Rust
	"Test.rs",      // Rust alternative
	"Test.cs",      // C#
	"_test.rb",     // Ruby
	"Test.php",     // PHP
	"*_test.exs",   // Elixir
}

// IsTestFile reports whether a filename follows common test naming patterns.
func IsTestFile(name string) bool {
	for _, pattern := range testFilePatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	// Also check for files in test directories
	base := filepath.Base(filepath.Dir(name))
	if base == "testdata" || base == "fixtures" {
		return false
	}
	return false
}

// buildSystemFiles maps known build/config files to their system identifier.
var buildSystemFiles = map[string]string{
	"go.mod":           "go-modules",
	"pom.xml":          "maven",
	"build.gradle":     "gradle",
	"build.gradle.kts": "gradle",
	"build.sbt":        "sbt",
	"package.json":     "npm",
	"Cargo.toml":       "cargo",
	"mix.exs":          "mix",
	"CMakeLists.txt":   "cmake",
	"Makefile":         "make",
	"BUILD":            "bazel",
	"WORKSPACE":        "bazel",
	"pyproject.toml":   "python",
	"setup.py":         "python",
	"Gemfile":          "bundler",
}

// DetectBuildSystem returns the build system identifier for a known build file.
func DetectBuildSystem(filename string) (string, bool) {
	sys, ok := buildSystemFiles[filename]
	return sys, ok
}
