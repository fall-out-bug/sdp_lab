package scout

import "testing"

func TestExtToLanguage(t *testing.T) {
	cases := map[string]string{
		".go":   "go",
		".ts":   "typescript",
		".tsx":  "typescript",
		".py":   "python",
		".java": "java",
		".rs":   "rust",
		".rb":   "ruby",
		".js":   "javascript",
		".jsx":  "javascript",
		".scala":"scala",
		".kt":   "kotlin",
		".c":    "c",
		".cpp":  "cpp",
		".h":    "c",
		".hpp":  "cpp",
		".sh":   "shell",
		".yaml": "yaml",
		".yml":  "yaml",
		".json": "json",
		".xml":  "xml",
		".md":   "markdown",
		".tf":   "terraform",
		".ex":   "elixir",
		".exs":  "elixir",
		".erl":  "erlang",
	}
	for ext, want := range cases {
		got, ok := ExtToLanguage(ext)
		if !ok {
			t.Errorf("ExtToLanguage(%q): not found", ext)
		} else if got != want {
			t.Errorf("ExtToLanguage(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestExtToLanguageUnknown(t *testing.T) {
	_, ok := ExtToLanguage(".zzz_unknown")
	if ok {
		t.Error("expected unknown extension to return false")
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []string{
		"foo_test.go",
		"bar.test.ts",
		"bar.test.tsx",
		"test_main.py",
		"MainTest.java",
		"FooSpec.scala",
		"FooTest.kt",
		"FooTest.rs",
	}
	for _, f := range tests {
		if !IsTestFile(f) {
			t.Errorf("IsTestFile(%q) = false, want true", f)
		}
	}
}

func TestIsTestFileNegative(t *testing.T) {
	notTests := []string{
		"main.go",
		"tester.py",
		"testimony.go",
		"foo.go",
		"README.md",
	}
	for _, f := range notTests {
		if IsTestFile(f) {
			t.Errorf("IsTestFile(%q) = true, want false", f)
		}
	}
}

func TestBuildSystemDetection(t *testing.T) {
	cases := map[string]string{
		"go.mod":              "go-modules",
		"pom.xml":             "maven",
		"build.gradle":        "gradle",
		"build.gradle.kts":    "gradle",
		"build.sbt":           "sbt",
		"package.json":        "npm",
		"Cargo.toml":          "cargo",
		"mix.exs":             "mix",
		"CMakeLists.txt":      "cmake",
		"Makefile":            "make",
		"BUILD":               "bazel",
		"WORKSPACE":           "bazel",
		"pyproject.toml":      "python",
		"setup.py":            "python",
		"Gemfile":             "bundler",
	}
	for file, want := range cases {
		got, ok := DetectBuildSystem(file)
		if !ok {
			t.Errorf("DetectBuildSystem(%q): not found", file)
		} else if got != want {
			t.Errorf("DetectBuildSystem(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestBuildSystemUnknown(t *testing.T) {
	_, ok := DetectBuildSystem("random.txt")
	if ok {
		t.Error("expected unknown file to return false")
	}
}
