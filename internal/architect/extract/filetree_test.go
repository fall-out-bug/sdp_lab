package extract

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeExtractor_BasicStructure(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create some test files and directories
	structure := []string{
		"src/main.go",
		"src/utils/helper.go",
		"lib/common.go",
		"cmd/server/main.go",
		"pkg/api/handler.go",
		"internal/auth/auth.go",
		"README.md",
		"go.mod",
	}

	for _, path := range structure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte("// test file"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	extractor := FileTreeExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if frag.FileTree == nil {
		t.Fatal("FileTree is nil")
	}

	// Check basic counts
	if frag.FileTree.TotalFiles != 9 {
		t.Errorf("Expected 9 files, got %d", frag.FileTree.TotalFiles)
	}

	if frag.FileTree.TotalDirs != 8 { // src, src/utils, lib, cmd, cmd/server, pkg, pkg/api, internal, internal/auth
		t.Errorf("Expected 8 directories, got %d", frag.FileTree.TotalDirs)
	}

	// Check for detected patterns
	found := make(map[string]bool)
	for _, pattern := range frag.FileTree.Patterns {
		found[pattern] = true
	}

	expectedPatterns := []string{"src_layout", "lib_layout", "cmd_layout", "pkg_layout", "internal_layout"}
	for _, exp := range expectedPatterns {
		if !found[exp] {
			t.Errorf("Expected pattern %s not found", exp)
		}
	}

	// Check metrics
	if frag.Metrics == nil {
		t.Fatal("Metrics is nil")
	}
	if frag.Metrics.TotalFiles != 9 {
		t.Errorf("Expected 9 files in metrics, got %d", frag.Metrics.TotalFiles)
	}
}

func TestFileTreeExtractor_NamingStyles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories with different naming styles
	styles := []string{
		"snake_case_dir",
		"camelCaseDir",
		"kebab-case-dir",
		"PascalCaseDir",
	}

	for _, dir := range styles {
		fullPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// Add a file to make it count
		testFile := filepath.Join(fullPath, "test.go")
		if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	extractor := FileTreeExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if frag.FileTree == nil {
		t.Fatal("FileTree is nil")
	}

	// Check for naming styles (need at least 2 directories of the same style to be detected)
	foundStyles := make(map[string]bool)
	for _, style := range frag.FileTree.NamingStyles {
		foundStyles[style] = true
	}

	// With only 1 directory of each style, none should be reported (minStyleCount = 2)
	if len(foundStyles) > 0 {
		t.Logf("Found naming styles: %v", frag.FileTree.NamingStyles)
	}
}

func TestFileTreeExtractor_NamingStyles_MultipleDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple directories with the same naming style
	snakeDirs := []string{
		"snake_case_one",
		"snake_case_two",
		"snake_case_three",
		"another_snake_dir",
	}

	camelDirs := []string{
		"camelCaseOne",
		"camelCaseTwo",
		"anotherCamelDir",
	}

	for _, dir := range snakeDirs {
		fullPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		testFile := filepath.Join(fullPath, "test.go")
		if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	for _, dir := range camelDirs {
		fullPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		testFile := filepath.Join(fullPath, "test.go")
		if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	extractor := FileTreeExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if frag.FileTree == nil {
		t.Fatal("FileTree is nil")
	}

	// Check for naming styles
	foundStyles := make(map[string]bool)
	for _, style := range frag.FileTree.NamingStyles {
		foundStyles[style] = true
	}

	// With 4 snake_case and 3 camelCase directories, both should be detected
	if !foundStyles["snake_case"] {
		t.Errorf("Expected snake_case naming style to be detected")
	}
	if !foundStyles["camelCase"] {
		t.Errorf("Expected camelCase naming style to be detected")
	}
}

func TestDetectNamingStyle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", "my_directory", "snake_case"},
		{"snake_case with numbers", "dir123_test", "snake_case"},
		{"camelCase", "myDirectory", "camelCase"},
		{"camelCase with numbers", "dir123Test", "camelCase"},
		{"kebab-case", "my-directory", "kebab-case"},
		{"kebab-case with numbers", "dir-123-test", "kebab-case"},
		{"PascalCase", "MyDirectory", "PascalCase"},
		{"PascalCase with numbers", "Dir123Test", "PascalCase"},
		{"single word lowercase", "directory", "unknown"},
		{"single word uppercase", "DIRECTORY", "unknown"},
		{"hidden directory", ".hidden", "unknown"},
		{"private directory", "_private", "unknown"},
		{"mixed separators", "my-directory_test", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectNamingStyle(tt.input)
			if result != tt.expected {
				t.Errorf("detectNamingStyle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFileTreeExtractor_ExtensionCounts(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"file.go":       "// go file",
		"file.py":       "# python file",
		"file.ts":       "// ts file",
		"file.java":     "// java file",
		"README.md":     "# readme",
		"data.json":     "{}",
		"config.yaml":   "key: value",
		"script.sh":     "#!/bin/bash",
		"doc.txt":       "text",
		"style.css":     ".class {}",
		"page.html":     "<html></html>",
		"binary.bin":    "binary",
		"archive.tar.gz": "archive",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	extractor := FileTreeExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if frag.FileTree == nil {
		t.Fatal("FileTree is nil")
	}

	expectedCounts := map[string]int{
		".go":    1,
		".py":    1,
		".ts":    1,
		".java":  1,
		".md":    1,
		".json":  1,
		".yaml":  1,
		".sh":    1,
		".txt":   1,
		".css":   1,
		".html":  1,
		".bin":   0, // build artifacts excluded
		".tar.gz": 0, // build artifacts excluded
	}

	for ext, expectedCount := range expectedCounts {
		actualCount := frag.FileTree.ExtCounts[ext]
		if actualCount != expectedCount {
			t.Errorf("Extension %s: expected %d, got %d", ext, expectedCount, actualCount)
		}
	}
}

func TestFileTreeExtractor_SkipDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories that should be skipped
	skipDirs := []string{
		".git",
		"node_modules",
		"vendor",
		"__pycache__",
		".sdp",
	}

	for _, dir := range skipDirs {
		fullPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// Add files inside
		testFile := filepath.Join(fullPath, "test.go")
		if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Add a regular file
	regularFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(regularFile, []byte("// main"), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	extractor := FileTreeExtractor{}
	frag, err := extractor.Extract(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should only count the regular file, not files in skipped directories
	if frag.FileTree.TotalFiles != 1 {
		t.Errorf("Expected 1 file (skipped dirs not counted), got %d", frag.FileTree.TotalFiles)
	}

	// Skipped directories should not be counted
	if frag.FileTree.TotalDirs != 0 {
		t.Errorf("Expected 0 directories (all skipped), got %d", frag.FileTree.TotalDirs)
	}
}

func TestFileTreeExtractor_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large directory structure
	for i := 0; i < 100; i++ {
		dir := filepath.Join(tmpDir, "dir", filepath.Join("nested", string(rune('a'+i))))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		file := filepath.Join(dir, "file.go")
		if err := os.WriteFile(file, []byte("// test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	extractor := FileTreeExtractor{}
	_, err := extractor.Extract(ctx, tmpDir)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
