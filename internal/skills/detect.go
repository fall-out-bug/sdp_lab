package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// DetectStack examines the project directory to determine the primary tech stack.
// Returns the path to the appropriate stack config, or an error if no stack detected.
func DetectStack(projectRoot string) (string, error) {
	// Check for Go project (highest priority)
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
		stackConfigPath := filepath.Join(projectRoot, "configs", "stack-configs", "go-default.json")
		if _, err := os.Stat(stackConfigPath); err == nil {
			return stackConfigPath, nil
		}
	}

	// Check for Python project (medium priority)
	hasRequirementsTxt := false
	hasPyprojectToml := false
	if _, err := os.Stat(filepath.Join(projectRoot, "requirements.txt")); err == nil {
		hasRequirementsTxt = true
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "pyproject.toml")); err == nil {
		hasPyprojectToml = true
	}
	if hasRequirementsTxt || hasPyprojectToml {
		stackConfigPath := filepath.Join(projectRoot, "configs", "stack-configs", "python-default.json")
		if _, err := os.Stat(stackConfigPath); err == nil {
			return stackConfigPath, nil
		}
	}

	// Check for TypeScript/JavaScript project (lowest priority)
	hasPackageJson := false
	hasTsconfigJson := false
	if _, err := os.Stat(filepath.Join(projectRoot, "package.json")); err == nil {
		hasPackageJson = true
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "tsconfig.json")); err == nil {
		hasTsconfigJson = true
	}
	if hasPackageJson && hasTsconfigJson {
		stackConfigPath := filepath.Join(projectRoot, "configs", "stack-configs", "typescript-default.json")
		if _, err := os.Stat(stackConfigPath); err == nil {
			return stackConfigPath, nil
		}
	}
	// JavaScript projects (package.json without tsconfig.json) also use TypeScript config
	if hasPackageJson {
		stackConfigPath := filepath.Join(projectRoot, "configs", "stack-configs", "typescript-default.json")
		if _, err := os.Stat(stackConfigPath); err == nil {
			return stackConfigPath, nil
		}
	}

	return "", fmt.Errorf("no supported stack detected")
}
