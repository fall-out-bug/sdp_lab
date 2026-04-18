package python

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonExtractor_Extract(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string]string
		wantFileCount  int
		wantMinDeps    int
		wantFrameworks int
	}{
		{
			name: "simple flask project",
			files: map[string]string{
				"requirements.txt": "flask==2.0.0\nrequests>=2.25.0\n",
				"app.py": `
from flask import Flask, Blueprint

app = Flask(__name__)

@app.route('/')
def hello():
    return 'Hello'

api = Blueprint('api', __name__)
`,
			},
			wantFileCount:  1,
			wantMinDeps:    3, // flask (from import), flask (from requirements.txt), requests
			wantFrameworks: 1, // Flask
		},
		{
			name: "non-python project",
			files: map[string]string{
				"README.md": "# My Project\n",
				"go.mod":    "module myapp\n",
			},
			wantFileCount:  0,
			wantMinDeps:    0,
			wantFrameworks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.files {
				fullPath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(fullPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			extractor := &PythonExtractor{}
			ctx := context.Background()
			result, err := extractor.Extract(ctx, tmpDir)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			if result.FileCount != tt.wantFileCount {
				t.Errorf("Extract() FileCount = %d, want %d", result.FileCount, tt.wantFileCount)
			}

			if len(result.Dependencies) < tt.wantMinDeps {
				t.Errorf("Extract() Dependencies = %d, want >= %d", len(result.Dependencies), tt.wantMinDeps)
			}

			if len(result.Frameworks) != tt.wantFrameworks {
				t.Errorf("Extract() Frameworks = %d, want %d", len(result.Frameworks), tt.wantFrameworks)
			}
		})
	}
}

func TestPythonExtractor_Language(t *testing.T) {
	extractor := &PythonExtractor{}
	if got := extractor.Language(); got != "python" {
		t.Errorf("PythonExtractor.Language() = %q, want 'python'", got)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	content := `
flask==2.0.0
requests>=2.25.0
# This is a comment
django>=3.2
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "requirements.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	deps := ParseRequirementsTxt(tmpFile)
	if len(deps) != 3 {
		t.Errorf("ParseRequirementsTxt() returned %d deps, want 3", len(deps))
	}

	expectedNames := []string{"flask", "requests", "django"}
	for i, wantName := range expectedNames {
		if i >= len(deps) {
			t.Errorf("missing dep at index %d", i)
			continue
		}
		if deps[i].Name != wantName {
			t.Errorf("deps[%d].Name = %q, want %q", i, deps[i].Name, wantName)
		}
	}
}
