package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectStack_GoProject(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir := t.TempDir()
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create the expected stack config path INSIDE the project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	stackConfigPath := filepath.Join(configsDir, "go-default.json")

	// Create the stack config file
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(stackConfigPath, []byte(`{"stack": "go", "display_name": "Go"}`), 0644); err != nil {
		t.Fatalf("failed to create stack config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	// The result should be the absolute path to the stack config
	expected := stackConfigPath
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDetectStack_PythonProject(t *testing.T) {
	tmpDir := t.TempDir()
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte("requests==2.31.0\n"), 0644); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// Create stack config inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	stackConfigPath := filepath.Join(configsDir, "python-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(stackConfigPath, []byte(`{"stack": "python", "display_name": "Python"}`), 0644); err != nil {
		t.Fatalf("failed to create stack config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	expected := stackConfigPath
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDetectStack_PythonProject_Pyproject(t *testing.T) {
	tmpDir := t.TempDir()
	pyprojectPath := filepath.Join(tmpDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte("[project]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatalf("failed to create pyproject.toml: %v", err)
	}

	// Create stack config inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	stackConfigPath := filepath.Join(configsDir, "python-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(stackConfigPath, []byte(`{"stack": "python", "display_name": "Python"}`), 0644); err != nil {
		t.Fatalf("failed to create stack config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	expected := stackConfigPath
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDetectStack_TypeScriptProject(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions": {}}`), 0644); err != nil {
		t.Fatalf("failed to create tsconfig.json: %v", err)
	}

	// Create stack config inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	stackConfigPath := filepath.Join(configsDir, "typescript-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(stackConfigPath, []byte(`{"stack": "typescript", "display_name": "TypeScript"}`), 0644); err != nil {
		t.Fatalf("failed to create stack config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	expected := stackConfigPath
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDetectStack_JavaScriptProject(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	// Create stack config inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	stackConfigPath := filepath.Join(configsDir, "typescript-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(stackConfigPath, []byte(`{"stack": "typescript", "display_name": "TypeScript"}`), 0644); err != nil {
		t.Fatalf("failed to create stack config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	expected := stackConfigPath
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDetectStack_NoStack(t *testing.T) {
	tmpDir := t.TempDir()
	// Empty directory with no project markers

	_, err := DetectStack(tmpDir)
	if err == nil {
		t.Fatal("expected error for no stack detected, got nil")
	}
	if err.Error() != "no supported stack detected" {
		t.Errorf("expected 'no supported stack detected', got %q", err.Error())
	}
}

func TestDetectStack_Priority(t *testing.T) {
	// Go should win when both Go and Python markers present
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create requirements.txt
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte("requests==2.31.0\n"), 0644); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// Create stack configs inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	goConfigPath := filepath.Join(configsDir, "go-default.json")
	pyConfigPath := filepath.Join(configsDir, "python-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(goConfigPath, []byte(`{"stack": "go", "display_name": "Go"}`), 0644); err != nil {
		t.Fatalf("failed to create go config: %v", err)
	}
	if err := os.WriteFile(pyConfigPath, []byte(`{"stack": "python", "display_name": "Python"}`), 0644); err != nil {
		t.Fatalf("failed to create python config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	// Should detect Go (higher priority than Python)
	expected := goConfigPath
	if result != expected {
		t.Errorf("expected %q (Go), got %q", expected, result)
	}
}

func TestDetectStack_PythonOverTypeScript(t *testing.T) {
	// Python should win when both Python and TypeScript markers present
	tmpDir := t.TempDir()

	// Create requirements.txt
	reqPath := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqPath, []byte("requests==2.31.0\n"), 0644); err != nil {
		t.Fatalf("failed to create requirements.txt: %v", err)
	}

	// Create package.json and tsconfig.json
	pkgPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(`{"compilerOptions": {}}`), 0644); err != nil {
		t.Fatalf("failed to create tsconfig.json: %v", err)
	}

	// Create stack configs inside project root
	configsDir := filepath.Join(tmpDir, "configs", "stack-configs")
	pyConfigPath := filepath.Join(configsDir, "python-default.json")
	tsConfigPath := filepath.Join(configsDir, "typescript-default.json")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}
	if err := os.WriteFile(pyConfigPath, []byte(`{"stack": "python", "display_name": "Python"}`), 0644); err != nil {
		t.Fatalf("failed to create python config: %v", err)
	}
	if err := os.WriteFile(tsConfigPath, []byte(`{"stack": "typescript", "display_name": "TypeScript"}`), 0644); err != nil {
		t.Fatalf("failed to create typescript config: %v", err)
	}

	result, err := DetectStack(tmpDir)
	if err != nil {
		t.Fatalf("DetectStack failed: %v", err)
	}

	// Should detect Python (higher priority than TypeScript)
	expected := pyConfigPath
	if result != expected {
		t.Errorf("expected %q (Python), got %q", expected, result)
	}
}
