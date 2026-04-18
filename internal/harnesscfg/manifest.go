// Package harnesscfg provides types and validation for the harness config manifest.
//
// The manifest describes which AI coding harnesses are active in a project, their
// config file paths, and the project lifecycle stage. Defined by JSON Schema at
// schema/harness-config-manifest.schema.json.
package harnesscfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

const (
	StageGreenfieldStr       = "greenfield"
	StageBrownfieldNewStr    = "brownfield-new"
	StageBrownfieldMatureStr = "brownfield-mature"
)

var validStages = map[string]bool{
	StageGreenfieldStr:       true,
	StageBrownfieldNewStr:    true,
	StageBrownfieldMatureStr: true,
}

var validHarnessNames = map[string]bool{
	"claude-code": true, "codex-cli": true, "cursor": true,
	"opencode": true, "copilot": true, "zed": true, "warp": true,
}

var semverRE = regexp.MustCompile(
	`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`,
)

// Manifest is the in-memory representation of a harness config manifest.
type Manifest struct {
	Version        string    `json:"version"`
	LifecycleStage string    `json:"lifecycle_stage"`
	Harnesses      []Harness `json:"harnesses"`
	Language       string    `json:"language,omitempty"`
	RulesFile      string    `json:"rules_file,omitempty"`
}

// Harness describes a single AI coding harness and its config file.
type Harness struct {
	Name       string `json:"name"`
	ConfigFile string `json:"config_file"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

// Schema holds parsed structural metadata from the JSON Schema file.
type Schema struct {
	properties map[string]json.RawMessage
}

// ParseSchema decodes JSON Schema bytes and extracts structural metadata.
func ParseSchema(data []byte) (*Schema, error) {
	var raw struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("harnesscfg: parse schema: %w", err)
	}
	return &Schema{properties: raw.Properties}, nil
}

// HasField reports whether the schema declares a top-level property with the given name.
func (s *Schema) HasField(name string) bool {
	_, ok := s.properties[name]
	return ok
}

// Validate checks that m satisfies the constraints expressed by the schema.
func Validate(_ *Schema, m Manifest) error {
	var errs []error
	if m.Version == "" {
		errs = append(errs, errors.New("version is required"))
	} else if !semverRE.MatchString(m.Version) {
		errs = append(errs, fmt.Errorf("version %q is not a valid semver string", m.Version))
	}
	if m.LifecycleStage == "" {
		errs = append(errs, errors.New("lifecycle_stage is required"))
	} else if !validStages[m.LifecycleStage] {
		errs = append(errs, fmt.Errorf("lifecycle_stage %q must be one of: greenfield, brownfield-new, brownfield-mature", m.LifecycleStage))
	}
	if len(m.Harnesses) == 0 {
		errs = append(errs, errors.New("harnesses must have at least one item"))
	}
	for i, h := range m.Harnesses {
		if !validHarnessNames[h.Name] {
			errs = append(errs, fmt.Errorf("harnesses[%d].name %q is not a recognized harness", i, h.Name))
		}
		if h.ConfigFile == "" {
			errs = append(errs, fmt.Errorf("harnesses[%d].config_file is required", i))
		}
	}
	return errors.Join(errs...)
}
