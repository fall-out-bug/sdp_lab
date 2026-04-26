package glob

import (
	"path/filepath"
	"testing"
)

// Benchmark comparing filepath.Match vs CompiledPattern
func BenchmarkLiteralMatch(b *testing.B) {
	pattern := "main.go"
	path := "main.go"
	cp := Compile(pattern)

	b.Run("filepath.Match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			filepath.Match(pattern, path)
		}
	})

	b.Run("CompiledPattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cp.MatchString(path)
		}
	})
}

func BenchmarkWildcardMatch(b *testing.B) {
	pattern := "*.go"
	path := "main.go"
	cp := Compile(pattern)

	b.Run("filepath.Match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			filepath.Match(pattern, path)
		}
	})

	b.Run("CompiledPattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cp.MatchString(path)
		}
	})
}

func BenchmarkComplexPatternMatch(b *testing.B) {
	pattern := "internal/**/test_*.go"
	path := "internal/strataudit/ingest_test.go"
	cp := Compile(pattern)

	b.Run("filepath.Match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			filepath.Match(pattern, path)
		}
	})

	b.Run("CompiledPattern", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cp.MatchString(path)
		}
	})
}

// Benchmark matching many files against many patterns (simulating large codebase)
func BenchmarkMatcherManyPatterns(b *testing.B) {
	patterns := []string{
		"*.go",
		"*_test.go",
		"internal/**/*.go",
		"cmd/**/*.go",
		"pkg/**/*.go",
		"*.md",
		"*.yaml",
		"*.yml",
		"*.json",
		"Dockerfile",
		"Makefile",
		"*.proto",
	}
	files := []string{
		"main.go",
		"main_test.go",
		"internal/strataudit/ingest.go",
		"internal/strataudit/ingest_test.go",
		"cmd/sdp/main.go",
		"README.md",
		"config.yaml",
		"package.json",
		"Dockerfile",
		"Makefile",
		"api/service.proto",
		"docs/guide.md",
	}

	matcher := NewMatcher(patterns)

	b.Run("Naive_loop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				for _, pattern := range patterns {
					filepath.Match(pattern, file)
				}
			}
		}
	})

	b.Run("Matcher_MatchAny", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				matcher.MatchAny(file)
			}
		}
	})
}

// Benchmark with realistic large codebase simulation
func BenchmarkLargeCodebase(b *testing.B) {
	// Simulate 1000 files
	patterns := []string{
		"*.go",
		"*_test.go",
		"internal/**/*.go",
		"cmd/**/*.go",
		"pkg/**/*.go",
		"api/**/*.go",
		"*.md",
		"*.yaml",
		"*.yml",
		"*.json",
		"Dockerfile*",
		"Makefile",
		"*.proto",
		"*.sql",
		"migrations/*.sql",
		"config/**",
	}

	files := make([]string, 1000)
	// Generate realistic file paths
	for i := 0; i < 1000; i++ {
		switch i % 10 {
		case 0:
			files[i] = "internal/strataudit/ingest.go"
		case 1:
			files[i] = "internal/strataudit/ingest_test.go"
		case 2:
			files[i] = "cmd/sdp/main.go"
		case 3:
			files[i] = "pkg/api/service.go"
		case 4:
			files[i] = "README.md"
		case 5:
			files[i] = "config/development.yaml"
		case 6:
			files[i] = "migrations/001_init.up.sql"
		case 7:
			files[i] = "api/v1/service.proto"
		case 8:
			files[i] = "docs/guides/setup.md"
		case 9:
			files[i] = "Makefile"
		}
	}

	matcher := NewMatcher(patterns)

	b.Run("Naive_loop", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				for _, pattern := range patterns {
					filepath.Match(pattern, file)
				}
			}
		}
	})

	b.Run("Matcher_MatchAny", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				matcher.MatchAny(file)
			}
		}
	})
}

// Benchmark prefix rejection optimization
func BenchmarkPrefixRejection(b *testing.B) {
	patterns := []string{
		"internal/**/*.go",
		"cmd/**/*.go",
		"pkg/**/*.go",
	}
	files := []string{
		"docs/readme.md",      // Should be quickly rejected
		"test/file.go",         // Should be quickly rejected
		"external/lib.go",      // Should be quickly rejected
		"README.md",            // Should be quickly rejected
		"internal/core/file.go", // Should match
	}

	matcher := NewMatcher(patterns)

	b.Run("Naive_loop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				for _, pattern := range patterns {
					filepath.Match(pattern, file)
				}
			}
		}
	})

	b.Run("Matcher_MatchAny", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				matcher.MatchAny(file)
			}
		}
	})
}

// Benchmark suffix optimization for "*.ext" patterns
func BenchmarkSuffixOptimization(b *testing.B) {
	patterns := []string{
		"*.go",
		"*.md",
		"*.yaml",
		"*.json",
	}
	files := []string{
		"main.go",
		"README.md",
		"config.yaml",
		"package.json",
		"test.txt", // No match
		"file.bin", // No match
	}

	matcher := NewMatcher(patterns)

	b.Run("Naive_loop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				for _, pattern := range patterns {
					filepath.Match(pattern, file)
				}
			}
		}
	})

	b.Run("Matcher_MatchAny", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, file := range files {
				matcher.MatchAny(file)
			}
		}
	})
}

// Test correctness
func TestCompile(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{"literal", "main.go", "main.go", true},
		{"literal no match", "main.go", "test.go", false},
		{"wildcard", "*.go", "main.go", true},
		{"wildcard no match", "*.go", "main.txt", false},
		{"complex", "internal/**/*.go", "internal/test/file.go", true},
		{"suffix", "*_test.go", "file_test.go", true},
		{"prefix", "cmd/*", "cmd/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := Compile(tt.pattern)
			result := cp.MatchString(tt.path)
			if result != tt.expected {
				t.Errorf("Compile(%q).Match(%q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatcher(t *testing.T) {
	patterns := []string{"*.go", "*.md", "internal/**"}
	matcher := NewMatcher(patterns)

	if !matcher.MatchAny("main.go") {
		t.Error("Expected main.go to match")
	}
	if !matcher.MatchAny("README.md") {
		t.Error("Expected README.md to match")
	}
	if !matcher.MatchAny("internal/test/file.go") {
		t.Error("Expected internal/test/file.go to match")
	}
	if matcher.MatchAny("test.txt") {
		t.Error("Expected test.txt to not match")
	}
}

func TestCaseInsensitiveMatcher(t *testing.T) {
	patterns := []string{"*.GO", "README.MD"}
	matcher := NewCaseInsensitiveMatcher(patterns)

	if !matcher.Match("main.go") {
		t.Error("Expected main.go to match *.GO (case-insensitive)")
	}
	if !matcher.Match("readme.md") {
		t.Error("Expected readme.md to match README.MD (case-insensitive)")
	}
}
