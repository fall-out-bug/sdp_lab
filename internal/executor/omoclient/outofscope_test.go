package omoclient

import (
	"context"
	"testing"
)

func TestOutOfScopeCheckerAllowed(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		denied      []string
		actualFiles []string
		expectClean bool
		expectViol  int
	}{
		{
			name:        "all allowed files",
			allowed:     []string{"*.go", "*.md"},
			denied:      []string{},
			actualFiles: []string{"main.go", "README.md", "test.go"},
			expectClean: true,
			expectViol:  0,
		},
		{
			name:        "some files not allowed",
			allowed:     []string{"src/*.go"},
			denied:      []string{},
			actualFiles: []string{"src/main.go", "test.go"},
			expectClean: false,
			expectViol:  1,
		},
		{
			name:        "empty allowed list denies all",
			allowed:     []string{},
			denied:      []string{},
			actualFiles: []string{"any.go"},
			expectClean: false,
			expectViol:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			checker := NewOutOfScopeChecker(tt.allowed, tt.denied)
			report := checker.Check(ctx, tt.actualFiles)

			if report.Clean != tt.expectClean {
				t.Errorf("Expected clean=%v, got %v", tt.expectClean, report.Clean)
			}

			if len(report.Violations) != tt.expectViol {
				t.Errorf("Expected %d violations, got %d: %v", tt.expectViol, len(report.Violations), report.Violations)
			}
		})
	}
}

func TestOutOfScopeCheckerDenied(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		denied      []string
		actualFiles []string
		expectClean bool
		expectViol  int
	}{
		{
			name:        "denied patterns",
			allowed:     []string{"*.go"},
			denied:      []string{"*_test.go", "vendor/*"},
			actualFiles: []string{"main.go", "main_test.go", "vendor/lib.go"},
			expectClean: false,
			expectViol:  2,
		},
		{
			name:        "denied directory",
			allowed:     []string{"**"},
			denied:      []string{".git/*"},
			actualFiles: []string{".git/config", ".git/HEAD", "main.go"},
			expectClean: false,
			expectViol:  2,
		},
		{
			name:        "denied matches but allowed also",
			allowed:     []string{"src/*.go"},
			denied:      []string{"*_test.go"},
			actualFiles: []string{"src/main.go", "src/main_test.go"},
			expectClean: false,
			expectViol:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			checker := NewOutOfScopeChecker(tt.allowed, tt.denied)
			report := checker.Check(ctx, tt.actualFiles)

			if report.Clean != tt.expectClean {
				t.Errorf("Expected clean=%v, got %v", tt.expectClean, report.Clean)
			}

			if len(report.Violations) != tt.expectViol {
				t.Errorf("Expected %d violations, got %d: %v", tt.expectViol, len(report.Violations), report.Violations)
			}
		})
	}
}

func TestOutOfScopeCheckerEmpty(t *testing.T) {
	ctx := context.Background()
	checker := NewOutOfScopeChecker(nil, nil)
	report := checker.Check(ctx, []string{})

	if !report.Clean {
		t.Errorf("Expected clean=true for empty input, got %v", report.Clean)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations for empty input, got %d", len(report.Violations))
	}
}

func TestOutOfScopeCheckerGlobPatterns(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		actualFiles []string
		expectClean bool
	}{
		{
			name:        "recursive pattern",
			allowed:     []string{"src/**/*.go"},
			actualFiles: []string{"src/main.go", "src/pkg/util.go", "test.go"},
			expectClean: false,
		},
		{
			name:        "question mark pattern",
			allowed:     []string{"file?.txt"},
			actualFiles: []string{"file1.txt", "file2.txt", "file.txt"},
			expectClean: false,
		},
		{
			name:        "character class pattern",
			allowed:     []string{"[abc].go"},
			actualFiles: []string{"a.go", "b.go", "c.go", "d.go"},
			expectClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			checker := NewOutOfScopeChecker(tt.allowed, nil)
			report := checker.Check(ctx, tt.actualFiles)

			if report.Clean != tt.expectClean {
				t.Errorf("Expected clean=%v, got %v", tt.expectClean, report.Clean)
			}
		})
	}
}

func TestOutOfScopeCheckerPathNormalization(t *testing.T) {
	tests := []struct {
		name        string
		pattern     []string
		actualFiles []string
		expectClean bool
	}{
		{
			name:        "windows backslash paths",
			pattern:     []string{"*.go"},
			actualFiles: []string{"src\\main.go", "pkg\\util.go"},
			expectClean: true,
		},
		{
			name:        "unix forward slash paths",
			pattern:     []string{"src/*.go"},
			actualFiles: []string{"src/main.go", "pkg/util.go"},
			expectClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			checker := NewOutOfScopeChecker(tt.pattern, nil)
			report := checker.Check(ctx, tt.actualFiles)

			if report.Clean != tt.expectClean {
				t.Errorf("Expected clean=%v, got %v", tt.expectClean, report.Clean)
			}
		})
	}
}

func TestOutOfScopeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewOutOfScopeChecker(nil, nil)
	report := checker.Check(ctx, []string{"any.go"})

	if !report.Clean {
		t.Errorf("Expected clean=true after cancellation, got %v", report.Clean)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations after cancellation, got %d", len(report.Violations))
	}
}
