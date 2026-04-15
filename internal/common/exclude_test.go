package common

import "testing"

func TestDefaultExcludesMatches(t *testing.T) {
	shouldExclude := []string{
		".git",
		"vendor",
		"node_modules",
		"__pycache__",
		"target",
		"build",
		"dist",
		".next",
		".nuxt",
		".terraform",
		".gradle",
		".mvn",
	}
	for _, name := range shouldExclude {
		if !DefaultMatcher.Match(name, false) {
			t.Errorf("DefaultMatcher.Match(%q, false) = false, want true", name)
		}
	}
}

func TestDefaultExcludesGeneratedPatterns(t *testing.T) {
	shouldExclude := []string{
		"foo.pb.go",
		"bar.generated.go",
		"baz.min.js",
		"qux.min.css",
	}
	for _, name := range shouldExclude {
		if !DefaultMatcher.Match(name, false) {
			t.Errorf("DefaultMatcher.Match(%q, false) = false, want true", name)
		}
	}
}

func TestDefaultExcludesLockFiles(t *testing.T) {
	shouldExclude := []string{
		"package-lock.json",
		"go.sum",
		"poetry.lock",
	}
	for _, name := range shouldExclude {
		if !DefaultMatcher.Match(name, false) {
			t.Errorf("DefaultMatcher.Match(%q, false) = false, want true", name)
		}
	}
}

func TestDefaultExcludesDoesNotMatchSource(t *testing.T) {
	shouldPass := []string{
		"main.go",
		"app.ts",
		"index.py",
		"Makefile",
		"README.md",
		"go.mod",
		"config.yaml",
	}
	for _, name := range shouldPass {
		if DefaultMatcher.Match(name, false) {
			t.Errorf("DefaultMatcher.Match(%q, false) = true, want false", name)
		}
	}
}

func TestDefaultExcludesDirectoryMode(t *testing.T) {
	// Directories should match directory-level patterns
	if !DefaultMatcher.Match("vendor", true) {
		t.Error("vendor dir should be excluded")
	}
	if !DefaultMatcher.Match("node_modules", true) {
		t.Error("node_modules dir should be excluded")
	}
	// A file named "vendor" should still match (name-based)
	if !DefaultMatcher.Match("vendor", false) {
		t.Error("vendor file should also be excluded by name")
	}
}
