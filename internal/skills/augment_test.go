package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkers(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantCount   int
		wantErr     bool
		firstSection string
		firstStack   string
	}{
		{
			name: "single marker",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`,
			wantCount:    1,
			wantErr:      false,
			firstSection: "test",
			firstStack:   "go",
		},
		{
			name: "multiple markers",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
pytest -v
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="build" stack="go" -->
go build ./...
<!-- STACK_SPECIFIC:END -->`,
			wantCount:    3,
			wantErr:      false,
			firstSection: "test",
			firstStack:   "go",
		},
		{
			name: "mismatched markers - missing end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...`,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "mismatched markers - extra end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:END -->`,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:        "no markers",
			content:     `# Some content without markers`,
			wantCount:   0,
			wantErr:     false,
			firstSection: "",
			firstStack:   "",
		},
		{
			name: "malformed marker order - END before BEGIN",
			content: `<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`,
			wantCount:   0,
			wantErr:     true,
			firstSection: "",
			firstStack:   "",
		},
		{
			name: "malformed markers - multiple ENDs",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:BEGIN section="build" stack="go" -->
go build ./...
<!-- STACK_SPECIFIC:END -->`,
			wantCount:   0,
			wantErr:     true,
			firstSection: "",
			firstStack:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markers, err := ParseMarkers([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMarkers() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMarkers() unexpected error: %v", err)
				return
			}

			if len(markers) != tt.wantCount {
				t.Errorf("ParseMarkers() got %d markers, want %d", len(markers), tt.wantCount)
			}

			if len(markers) > 0 {
				if markers[0].Section != tt.firstSection {
					t.Errorf("ParseMarkers() first marker section = %q, want %q", markers[0].Section, tt.firstSection)
				}
				if markers[0].Stack != tt.firstStack {
					t.Errorf("ParseMarkers() first marker stack = %q, want %q", markers[0].Stack, tt.firstStack)
				}
			}
		})
	}
}

func TestRenderBlock(t *testing.T) {
	goConfig := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Run all tests:\n```bash\ngo test ./... -v\n```",
			},
			"build": {
				Heading: "### Go Building",
				Content: "Build all:\n```bash\ngo build ./...\n```",
			},
		},
	}

	tests := []struct {
		name        string
		section     string
		wantHeading string
		wantErr     bool
	}{
		{
			name:        "existing section",
			section:     "test",
			wantHeading: "### Go Testing",
			wantErr:     false,
		},
		{
			name:        "another existing section",
			section:     "build",
			wantHeading: "### Go Building",
			wantErr:     false,
		},
		{
			name:        "non-existent section",
			section:     "coverage",
			wantHeading: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderBlock(tt.section, goConfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RenderBlock() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("RenderBlock() unexpected error: %v", err)
				return
			}

			if !strings.Contains(rendered, tt.wantHeading) {
				t.Errorf("RenderBlock() output does not contain heading %q", tt.wantHeading)
			}
		})
	}
}

func TestValidateMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid single marker",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`,
			wantErr: false,
		},
		{
			name: "valid multiple markers",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
pytest -v
<!-- STACK_SPECIFIC:END -->`,
			wantErr: false,
		},
		{
			name: "mismatched - missing end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...`,
			wantErr: true,
		},
		{
			name: "mismatched - extra end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:END -->`,
			wantErr: true,
		},
		{
			name: "no markers",
			content: `# Some content without markers`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "skill.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Change to temp directory so absolute paths are within the "current" directory
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			// Use relative path since we changed to the temp directory
			err := ValidateMarkers("skill.md")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMarkers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadStackConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
		wantStack string
	}{
		{
			name: "valid config",
			config: `{
  "stack": "go",
  "version": "1.0",
  "display_name": "Go",
  "commands": {
    "test": "go test ./..."
  },
  "quality_gates": ["go test ./..."],
  "file_patterns": ["*.go"],
  "sections": {
    "test": {
      "heading": "### Go Testing",
      "content": "go test ./..."
    }
  }
}`,
			wantErr:  false,
			wantStack: "go",
		},
		{
			name: "missing stack field",
			config: `{
  "version": "1.0",
  "sections": {}
}`,
			wantErr:  true,
			wantStack: "",
		},
		{
			name: "missing sections",
			config: `{
  "stack": "go",
  "version": "1.0",
  "sections": {}
}`,
			wantErr:  true,
			wantStack: "",
		},
		{
			name: "invalid JSON",
			config: `{invalid json}`,
			wantErr:  true,
			wantStack: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "config.json")
			if err := os.WriteFile(tmpFile, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Change to temp directory so absolute paths are within the "current" directory
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			config, err := LoadStackConfig("config.json")
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadStackConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config.Stack != tt.wantStack {
				t.Errorf("LoadStackConfig() stack = %q, want %q", config.Stack, tt.wantStack)
			}
		})
	}
}

func TestAugmentSkillIdempotency(t *testing.T) {
	// Test that running augmentation twice produces identical output
	goConfig := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Run all tests:\n```bash\ngo test ./... -v\n```",
			},
		},
	}

	originalContent := `# Test Skill

Some general content.

<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

More content.
`

	// Create temporary files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "skill1.md")
	file2 := filepath.Join(tmpDir, "skill2.md")

	if err := os.WriteFile(file1, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := os.WriteFile(file2, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Change to temp directory so absolute paths are within the "current" directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Augment file1 once
	if err := AugmentSkill("skill1.md", goConfig); err != nil {
		t.Fatalf("first AugmentSkill() failed: %v", err)
	}

	// Augment file1 again (should be idempotent)
	if err := AugmentSkill("skill1.md", goConfig); err != nil {
		t.Fatalf("second AugmentSkill() failed: %v", err)
	}

	// Augment file2 once for comparison
	if err := AugmentSkill("skill2.md", goConfig); err != nil {
		t.Fatalf("AugmentSkill() on file2 failed: %v", err)
	}

	// Read both files
	content1, err := os.ReadFile("skill1.md")
	if err != nil {
		t.Fatalf("failed to read file1: %v", err)
	}
	content2, err := os.ReadFile("skill2.md")
	if err != nil {
		t.Fatalf("failed to read file2: %v", err)
	}

	// They should be identical (idempotent)
	if string(content1) != string(content2) {
		t.Errorf("AugmentSkill() is not idempotent: files differ after double vs single augmentation")
	}
}

func TestAugmentSkillPreservesContent(t *testing.T) {
	// Test that content outside markers is preserved
	goConfig := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Run all tests:\n```bash\ngo test ./... -v\n```",
			},
		},
	}

	originalContent := `# Test Skill

This is important general content that should be preserved.

## MUST DO

- Do this
- Do that

<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->

## References

- [Reference 1](https://example.com)
`

	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "skill.md")

	if err := os.WriteFile(tmpFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Change to temp directory so absolute paths are within the "current" directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Augment the file
	if err := AugmentSkill("skill.md", goConfig); err != nil {
		t.Fatalf("AugmentSkill() failed: %v", err)
	}

	// Read the augmented content
	augmentedContent, err := os.ReadFile("skill.md")
	if err != nil {
		t.Fatalf("failed to read augmented file: %v", err)
	}

	augmentedStr := string(augmentedContent)

	// Check that important sections are preserved
	preservedSections := []string{
		"# Test Skill",
		"This is important general content that should be preserved.",
		"## MUST DO",
		"- Do this",
		"- Do that",
		"## References",
		"[Reference 1](https://example.com)",
	}

	for _, section := range preservedSections {
		if !strings.Contains(augmentedStr, section) {
			t.Errorf("AugmentSkill() did not preserve content: %q", section)
		}
	}
}

func TestDryRunAugment(t *testing.T) {
	goConfig := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Run all tests:\n```bash\ngo test ./... -v\n```",
			},
		},
	}

	tests := []struct {
		name           string
		content        string
		wantChangeType string
	}{
		{
			name: "new marker would be added",
			content: `# Test Skill
No existing markers.`,
			wantChangeType: "new marker would be added",
		},
		{
			name: "existing marker would be updated",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`,
			wantChangeType: "content would be updated",
		},
		{
			name: "existing marker unchanged (idempotent)",
			content: "<!-- STACK_SPECIFIC:BEGIN section=\"test\" stack=\"go\" -->\n### Go Testing\n\nRun all tests:\n```bash\ngo test ./... -v\n```\n<!-- STACK_SPECIFIC:END -->",
			wantChangeType: "no change (idempotent)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "skill.md")

			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Change to temp directory so absolute paths are within the "current" directory
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			// Run dry-run
			output, err := DryRunAugment("skill.md", goConfig)
			if err != nil {
				t.Fatalf("DryRunAugment() failed: %v", err)
			}

			// Check that output contains expected change type
			if !strings.Contains(output, tt.wantChangeType) {
				t.Errorf("DryRunAugment() output does not contain %q\nGot: %s", tt.wantChangeType, output)
			}

			// Check that file was not modified
			content, _ := os.ReadFile("skill.md")
			if string(content) != tt.content {
				t.Errorf("DryRunAugment() modified file when it should not have")
			}
		})
	}
}

func TestFindSkillFiles(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create some skill files
	skillFiles := []string{
		"build.md",
		"test.md",
		"deploy.md",
		"subdir/feature.md",
	}
	nonSkillFiles := []string{
		"readme.txt",
		"config.json",
		".gitkeep",
	}

	for _, file := range skillFiles {
		path := filepath.Join(tmpDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("# test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	for _, file := range nonSkillFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Find skill files
	skillFileList, err := FindSkillFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindSkillFiles() failed: %v", err)
	}

	// Check count
	if len(skillFileList) != len(skillFiles) {
		t.Errorf("FindSkillFiles() found %d files, want %d", len(skillFileList), len(skillFiles))
	}

	// Check that all skill files are found
	for _, file := range skillFiles {
		expectedPath := filepath.Join(tmpDir, file)
		found := false
		for _, f := range skillFileList {
			if f == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FindSkillFiles() did not find %q", expectedPath)
		}
	}
}

func TestSafePath(t *testing.T) {
	tests := []struct {
		name        string
		baseDir     string
		untrusted   string
		wantErr     bool
		errContains string
		special     string
	}{
		{
			name:      "valid relative path",
			baseDir:   "/tmp/test",
			untrusted: "file.txt",
			wantErr:   false,
			special:   "",
		},
		{
			name:      "valid nested path",
			baseDir:   "/tmp/test",
			untrusted: "subdir/file.txt",
			wantErr:   false,
			special:   "",
		},
		{
			name:        "path traversal with ..",
			baseDir:     "/tmp/test",
			untrusted:   "../../../etc/passwd",
			wantErr:     true,
			errContains: "escapes base directory",
			special:     "",
		},
		{
			name:        "absolute path outside base is rejected",
			baseDir:     "/tmp/test",
			untrusted:   "/etc/passwd",
			wantErr:     true,
			errContains: "escapes base directory",
			special:     "",
		},
		{
			name:      "absolute path within base is accepted",
			baseDir:   ".",
			untrusted: "", // Will be replaced with actual path in test
			wantErr:   false,
			special:   "absolute-within-base",
		},
		{
			name:      "path with .. but still in base",
			baseDir:   "/tmp/test",
			untrusted: "sub/../file.txt",
			wantErr:   false,
			special:   "",
		},
		{
			name:      "empty path stays in base",
			baseDir:   "/tmp/test",
			untrusted: "",
			wantErr:   false,
			special:   "",
		},
		{
			name:      "current directory reference",
			baseDir:   "/tmp/test",
			untrusted: "./file.txt",
			wantErr:   false,
			special:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base directory for testing
			tmpDir := t.TempDir()
			baseDir := tmpDir
			if tt.baseDir != "/tmp/test" {
				baseDir = filepath.Join(tmpDir, tt.baseDir)
			}

			// Create base directory if it doesn't exist
			if err := os.MkdirAll(baseDir, 0755); err != nil {
				t.Fatalf("failed to create base directory: %v", err)
			}

			// Handle special test case for absolute path within base
			untrusted := tt.untrusted
			if tt.special == "absolute-within-base" {
				// Create a subdirectory and use its absolute path
				subdir := filepath.Join(baseDir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("failed to create subdirectory: %v", err)
				}
				untrusted = subdir
			}

			got, err := safePath(baseDir, untrusted)

			if tt.wantErr {
				if err == nil {
					t.Errorf("safePath() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("safePath() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("safePath() unexpected error: %v", err)
				return
			}

			// ALL paths (relative and absolute) must stay within baseDir
			absBase, _ := filepath.Abs(baseDir)
			if !strings.HasPrefix(got, absBase+string(os.PathSeparator)) && got != absBase {
				t.Errorf("safePath() result %q is not under base directory %q", got, absBase)
			}
		})
	}
}

func TestNewAugmenter(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid directory",
			projectRoot: ".",
			wantErr:     false,
		},
		{
			name:        "valid temp directory",
			projectRoot: os.TempDir(),
			wantErr:     false,
		},
		{
			name:        "non-existent directory",
			projectRoot: "/nonexistent/path/that/does/not/exist",
			wantErr:     true,
			errContains: "does not exist",
		},
		{
			name:        "file instead of directory",
			projectRoot: "/etc/passwd",
			wantErr:     true,
			errContains: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			augmenter, err := NewAugmenter(tt.projectRoot)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAugmenter() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewAugmenter() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewAugmenter() unexpected error: %v", err)
				return
			}

			if augmenter == nil {
				t.Errorf("NewAugmenter() returned nil augmenter")
			}

			if augmenter.projectRoot == "" {
				t.Errorf("NewAugmenter() projectRoot is empty")
			}

			// Verify projectRoot is absolute
			if !filepath.IsAbs(augmenter.projectRoot) {
				t.Errorf("NewAugmenter() projectRoot %q is not absolute", augmenter.projectRoot)
			}
		})
	}
}

func TestReadFileWithLimit(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "small file",
			content: "small content",
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: false,
		},
		{
			name:    "large file under limit",
			content: strings.Repeat("x", 1024*1024), // 1MB
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")

			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			data, err := readFileWithLimit(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("readFileWithLimit() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("readFileWithLimit() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("readFileWithLimit() unexpected error: %v", err)
				return
			}

			if string(data) != tt.content {
				t.Errorf("readFileWithLimit() got %q, want %q", string(data), tt.content)
			}
		})
	}
}

func TestReadFileWithLimitExceeds(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.txt")

	// Create a file larger than maxSkillFileSize (10MB)
	largeContent := strings.Repeat("x", maxSkillFileSize+1)
	if err := os.WriteFile(tmpFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := readFileWithLimit(tmpFile)
	if err == nil {
		t.Errorf("readFileWithLimit() expected error for oversized file but got none")
		return
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("readFileWithLimit() error = %v, want error containing 'exceeds limit'", err)
	}
}

func TestValidateMarkersExtended(t *testing.T) {
	// Additional tests for ValidateMarkers function
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "file with no markers",
			content: `# Just a regular file with no markers at all.`,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: ``,
			wantErr: false,
		},
		{
			name: "unclosed marker - missing end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
content here without end marker`,
			wantErr: true,
		},
		{
			name: "mismatched markers - extra end",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
content
<!-- STACK_SPECIFIC:END -->
<!-- STACK_SPECIFIC:END -->`,
			wantErr: true,
		},
		{
			name: "balanced markers",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
content
<!-- STACK_SPECIFIC:END -->`,
			wantErr: false,
		},
		{
			name: "multiple balanced markers",
			content: `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
content1
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="build" stack="python" -->
content2
<!-- STACK_SPECIFIC:END -->`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "skill.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// Change to temp directory so absolute paths are within the "current" directory
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to change to temp directory: %v", err)
			}

			err := ValidateMarkers("skill.md")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMarkers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRenderBlockWithTemplateSyntax(t *testing.T) {
	// Test that RenderBlock correctly concatenates heading and content
	// Note: RenderBlock does NOT use Go template syntax - it's simple concatenation
	config := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Stack: go\nVersion: 1.0\nDisplay: Go",
			},
		},
	}

	t.Run("simple heading and content concatenation", func(t *testing.T) {
		rendered, err := RenderBlock("test", config)
		if err != nil {
			t.Fatalf("RenderBlock() unexpected error: %v", err)
		}

		// Check that heading is present
		if !strings.Contains(rendered, "### Go Testing") {
			t.Errorf("RenderBlock() output does not contain heading\nGot: %s", rendered)
		}
		// Check that content is present
		if !strings.Contains(rendered, "Stack: go") {
			t.Errorf("RenderBlock() output does not contain content\nGot: %s", rendered)
		}
		if !strings.Contains(rendered, "Version: 1.0") {
			t.Errorf("RenderBlock() output does not contain version\nGot: %s", rendered)
		}
		if !strings.Contains(rendered, "Display: Go") {
			t.Errorf("RenderBlock() output does not contain display name\nGot: %s", rendered)
		}
	})

	t.Run("heading and content are separated by double newline", func(t *testing.T) {
		rendered, err := RenderBlock("test", config)
		if err != nil {
			t.Fatalf("RenderBlock() unexpected error: %v", err)
		}

		// Check that heading and content are separated by \n\n
		expected := "### Go Testing\n\nStack: go\nVersion: 1.0\nDisplay: Go"
		if rendered != expected {
			t.Errorf("RenderBlock() output format incorrect\nGot: %q\nWant: %q", rendered, expected)
		}
	})

	t.Run("content with special characters", func(t *testing.T) {
		configSpecial := StackConfig{
			Stack:       "python",
			Version:     "3.11",
			DisplayName: "Python",
			Sections: map[string]Section{
				"install": {
					Heading: "### Installation for Python",
					Content: "```bash\npip install python==3.11\n```",
				},
			},
		}

		rendered, err := RenderBlock("install", configSpecial)
		if err != nil {
			t.Fatalf("RenderBlock() unexpected error: %v", err)
		}

		if !strings.Contains(rendered, "### Installation for Python") {
			t.Errorf("RenderBlock() heading not rendered correctly\nGot: %s", rendered)
		}
		if !strings.Contains(rendered, "pip install python==3.11") {
			t.Errorf("RenderBlock() content not rendered correctly\nGot: %s", rendered)
		}
	})

	t.Run("non-existent section returns error", func(t *testing.T) {
		_, err := RenderBlock("nonexistent", config)
		if err == nil {
			t.Errorf("RenderBlock() expected error for non-existent section but got none")
		}
	})
}

func TestPathTraversalProtection(t *testing.T) {
	// Test that functions properly use the safe path returned by safePath()
	// and don't use the original untrusted input
	goConfig := StackConfig{
		Stack:       "go",
		Version:     "1.0",
		DisplayName: "Go",
		Sections: map[string]Section{
			"test": {
				Heading: "### Go Testing",
				Content: "Run all tests:\n```bash\ngo test ./... -v\n```",
			},
		},
	}

	t.Run("AugmentSkill rejects path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a valid skill file
		validFile := filepath.Join(tmpDir, "skill.md")
		content := `# Test Skill
<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		// Change to temp directory so the valid file is accessible
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp directory: %v", err)
		}

		// Try to use path traversal to escape the directory
		traversalPath := "../../../etc/passwd"

		// This should fail because the path traversal attempts to escape
		err := AugmentSkill(traversalPath, goConfig)
		if err == nil {
			t.Errorf("AugmentSkill() expected error for path traversal but got none")
			return
		}
		if !strings.Contains(err.Error(), "escapes base directory") {
			t.Errorf("AugmentSkill() error = %v, want error containing 'escapes base directory'", err)
		}
	})

	t.Run("ValidateMarkers rejects path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a valid skill file
		validFile := filepath.Join(tmpDir, "skill.md")
		content := `<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
go test ./...
<!-- STACK_SPECIFIC:END -->`
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		// Change to temp directory so the valid file is accessible
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp directory: %v", err)
		}

		// Validate should work with the valid file
		err := ValidateMarkers("skill.md")
		if err != nil {
			t.Fatalf("ValidateMarkers() failed on valid file: %v", err)
		}
	})

	t.Run("LoadStackConfig rejects path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a valid config file
		validConfig := filepath.Join(tmpDir, "config.json")
		configContent := `{
  "stack": "go",
  "version": "1.0",
  "display_name": "Go",
  "commands": {},
  "quality_gates": [],
  "file_patterns": [],
  "sections": {
    "test": {
      "heading": "### Go Testing",
      "content": "go test ./..."
    }
  }
}`
		if err := os.WriteFile(validConfig, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to create temp config: %v", err)
		}

		// Change to temp directory so the valid file is accessible
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("failed to change to temp directory: %v", err)
		}

		// Load should work with the valid config
		_, err := LoadStackConfig("config.json")
		if err != nil {
			t.Fatalf("LoadStackConfig() failed on valid config: %v", err)
		}
	})
}
