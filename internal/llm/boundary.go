package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BoundarySpec defines path constraints for LLM execution.
type BoundarySpec struct {
	AllowedPathPrefixes   []string
	ControlPathPrefixes   []string
	ForbiddenPathPrefixes []string
}

// workstreamConfig is the YAML structure for specs/workstream-config.yaml.
type workstreamConfig struct {
	Workstreams []struct {
		Label        string   `yaml:"label"`
		PathPrefixes []string `yaml:"path_prefixes"`
	} `yaml:"workstreams"`
}

// DefaultControlPaths are paths the system manages; LLM must not modify.
var DefaultControlPaths = []string{".beads/", ".sdp/"}

// DefaultForbiddenPaths are paths the LLM must never touch.
var DefaultForbiddenPaths = []string{".git/"}

// LoadBoundary loads boundary spec for the given workstream from workstream-config.yaml.
// root is the repo root; config is read from root/specs/workstream-config.yaml.
// workstream is the short name (e.g. "generic", "builder").
func LoadBoundary(root, workstream string) (BoundarySpec, error) {
	cfgPath := filepath.Join(root, "specs", "workstream-config.yaml")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return BoundarySpec{}, fmt.Errorf("read workstream config: %w", err)
	}
	var cfg workstreamConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return BoundarySpec{}, fmt.Errorf("parse workstream config: %w", err)
	}
	label := "workstream:" + workstream
	for _, w := range cfg.Workstreams {
		if w.Label == label {
			return BoundarySpec{
				AllowedPathPrefixes:   append([]string{}, w.PathPrefixes...),
				ControlPathPrefixes:   append([]string{}, DefaultControlPaths...),
				ForbiddenPathPrefixes: append([]string{}, DefaultForbiddenPaths...),
			}, nil
		}
	}
	// Fallback for generic/builder: use same as workstream:generic
	if workstream == "builder" {
		return LoadBoundary(root, "generic")
	}
	return BoundarySpec{}, fmt.Errorf("workstream %q not found in config", workstream)
}

// ValidateChangedPaths checks that all changed paths comply with the boundary.
// Returns an error listing any paths that violate the boundary.
func ValidateChangedPaths(changed []string, spec BoundarySpec) error {
	var violations []string
	for _, p := range changed {
		if hasPrefixAny(p, spec.ControlPathPrefixes) {
			continue
		}
		if hasPrefixAny(p, spec.ForbiddenPathPrefixes) {
			violations = append(violations, p)
			continue
		}
		if len(spec.AllowedPathPrefixes) > 0 && !hasPrefixAny(p, spec.AllowedPathPrefixes) {
			violations = append(violations, p)
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return fmt.Errorf("boundary violation: %s", strings.Join(violations, ", "))
}

func hasPrefixAny(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) || strings.HasPrefix(filepath.ToSlash(path), prefix) {
			return true
		}
	}
	return false
}
