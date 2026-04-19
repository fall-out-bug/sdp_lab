package bootstrap

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GreenfieldConfig holds answers from the interactive bootstrap flow.
// Each field controls which constitution sections are generated.
type GreenfieldConfig struct {
	ProjectType     string `json:"project_type"`     // "web-service", "cli", "library", "monorepo"
	PrimaryLanguage string `json:"primary_language"`  // "go", "python", "typescript", etc.
	TestStrategy    string `json:"test_strategy"`     // "unit", "integration", "tdd", "minimal"
	CIPreference    string `json:"ci_preference"`     // "github-actions", "gitlab-ci", "none"
	DeployTarget    string `json:"deploy_target"`     // "docker", "kubernetes", "serverless", "none"
}

// Preset configurations for non-interactive mode. Each preset maps a short
// name to a fully populated GreenfieldConfig.
var Presets = map[string]GreenfieldConfig{
	"go-web-service": {
		ProjectType:     "web-service",
		PrimaryLanguage: "go",
		TestStrategy:    "tdd",
		CIPreference:    "github-actions",
		DeployTarget:    "docker",
	},
	"go-cli": {
		ProjectType:     "cli",
		PrimaryLanguage: "go",
		TestStrategy:    "unit",
		CIPreference:    "github-actions",
		DeployTarget:    "none",
	},
	"go-library": {
		ProjectType:     "library",
		PrimaryLanguage: "go",
		TestStrategy:    "tdd",
		CIPreference:    "github-actions",
		DeployTarget:    "none",
	},
}

// BootstrapResult holds the generated artifacts from a greenfield bootstrap run.
type BootstrapResult struct {
	PrinciplesContent string // DRAFT-PRINCIPLES.md content
	AgentsContent     string // AGENTS.md rules section content
	ConfigFile        string // path to .sdp/bootstrap-answers.json
}

// RunGreenfield runs the bootstrap flow with the given config. It produces
// deterministic DRAFT artifacts that the user can review and curate.
func RunGreenfield(config GreenfieldConfig) (*BootstrapResult, error) {
	if err := validateGreenfieldConfig(config); err != nil {
		return nil, fmt.Errorf("invalid greenfield config: %w", err)
	}

	principles := renderPrinciples(config)
	agents := renderAgentsRules(config)
	configFile := renderConfigFilePath(config)

	return &BootstrapResult{
		PrinciplesContent: principles,
		AgentsContent:     agents,
		ConfigFile:        configFile,
	}, nil
}

// RunGreenfieldFromPreset runs the bootstrap flow using a named preset.
// Returns an error if the preset name is not recognized.
func RunGreenfieldFromPreset(presetName string) (*BootstrapResult, error) {
	cfg, ok := Presets[presetName]
	if !ok {
		available := presetNames()
		return nil, fmt.Errorf("unknown preset %q: available presets: %s",
			presetName, strings.Join(available, ", "))
	}
	return RunGreenfield(cfg)
}

// MarshalAnswers serializes the config as JSON for reproducibility.
// The caller can write the result to .sdp/bootstrap-answers.json.
func MarshalAnswers(config GreenfieldConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// validateGreenfieldConfig checks that all required fields are populated
// with at least one non-whitespace character.
func validateGreenfieldConfig(config GreenfieldConfig) error {
	if strings.TrimSpace(config.ProjectType) == "" {
		return fmt.Errorf("project_type is required")
	}
	if strings.TrimSpace(config.PrimaryLanguage) == "" {
		return fmt.Errorf("primary_language is required")
	}
	if strings.TrimSpace(config.TestStrategy) == "" {
		return fmt.Errorf("test_strategy is required")
	}
	if strings.TrimSpace(config.CIPreference) == "" {
		return fmt.Errorf("ci_preference is required")
	}
	if strings.TrimSpace(config.DeployTarget) == "" {
		return fmt.Errorf("deploy_target is required")
	}
	return nil
}

// presetNames returns a sorted list of available preset names.
func presetNames() []string {
	names := make([]string, 0, len(Presets))
	for k := range Presets {
		names = append(names, k)
	}
	return names
}

// renderConfigFilePath returns the canonical path for the answers JSON file.
func renderConfigFilePath(config GreenfieldConfig) string {
	return ".sdp/bootstrap-answers.json"
}
