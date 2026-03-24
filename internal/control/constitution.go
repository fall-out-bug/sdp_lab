package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Constitution is the foundational constraint set for a project.
type Constitution struct {
	Vision           string   `yaml:"vision" json:"vision"`
	Principles       []string `yaml:"principles" json:"principles"`
	NonNegotiables   []string `yaml:"non_negotiables" json:"non_negotiables"`
	AllowedTaskTypes []string `yaml:"allowed_task_types" json:"allowed_task_types"`
	MaxRiskDefault   string   `yaml:"max_risk_default" json:"max_risk_default"`
}

func defaultConstitution() *Constitution {
	return &Constitution{
		Vision: "SDP provides a spec-driven pipeline from intent to deploy with provenance, evidence, and trace.",
		Principles: []string{
			"Code is derivable. Contracts are not.",
			"Every agent is opaque — communicates via declared contracts only.",
			"No agent holds shadow state of another agent.",
			"Every dispatch produces traceable provenance.",
			"Evidence is mandatory, not optional.",
		},
		NonNegotiables: []string{
			"Security policy must be set for high-risk tasks",
			"Contract gates cannot be bypassed",
		},
		AllowedTaskTypes: []string{"feature", "bugfix", "refactor", "infra", "docs"},
		MaxRiskDefault:   "medium",
	}
}

// LoadConstitution reads constitution.yaml or constitution.json from the provided root.
// If no constitution file exists, it returns the embedded default constitution.
func LoadConstitution(controlRoot string) (*Constitution, error) {
	constitution := defaultConstitution()
	for _, candidate := range []string{
		filepath.Join(controlRoot, "constitution.yaml"),
		filepath.Join(controlRoot, "constitution.json"),
	} {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read constitution %s: %w", candidate, err)
		}
		loaded := &Constitution{}
		switch filepath.Ext(candidate) {
		case ".json":
			if err := json.Unmarshal(data, loaded); err != nil {
				return nil, fmt.Errorf("parse constitution %s: %w", candidate, err)
			}
		default:
			if err := yaml.Unmarshal(data, loaded); err != nil {
				return nil, fmt.Errorf("parse constitution %s: %w", candidate, err)
			}
		}
		return mergeConstitutionDefaults(loaded), nil
	}
	return constitution, nil
}

func mergeConstitutionDefaults(loaded *Constitution) *Constitution {
	if loaded == nil {
		return defaultConstitution()
	}
	merged := defaultConstitution()
	if strings.TrimSpace(loaded.Vision) != "" {
		merged.Vision = strings.TrimSpace(loaded.Vision)
	}
	if len(cleanList(loaded.Principles)) > 0 {
		merged.Principles = cleanList(loaded.Principles)
	}
	if len(cleanList(loaded.NonNegotiables)) > 0 {
		merged.NonNegotiables = cleanList(loaded.NonNegotiables)
	}
	if len(cleanList(loaded.AllowedTaskTypes)) > 0 {
		merged.AllowedTaskTypes = cleanList(loaded.AllowedTaskTypes)
	}
	if strings.TrimSpace(loaded.MaxRiskDefault) != "" {
		merged.MaxRiskDefault = strings.ToLower(strings.TrimSpace(loaded.MaxRiskDefault))
	}
	return merged
}

func (c *Constitution) ValidateCard(card *FeatureCard) []string {
	if card == nil {
		return nil
	}
	constitution := mergeConstitutionDefaults(c)
	var warnings []string

	if taskType := strings.ToLower(strings.TrimSpace(card.TaskType)); taskType != "" && len(constitution.AllowedTaskTypes) > 0 {
		allowed := false
		for _, candidate := range constitution.AllowedTaskTypes {
			if strings.EqualFold(strings.TrimSpace(candidate), taskType) {
				allowed = true
				break
			}
		}
		if !allowed {
			warnings = append(warnings, fmt.Sprintf("task_type %q is not listed in constitution allowed_task_types", card.TaskType))
		}
	}

	if cardRisk, ok := riskRank(card.RiskLevel); ok {
		if maxRisk, ok := riskRank(constitution.MaxRiskDefault); ok && cardRisk > maxRisk {
			warnings = append(warnings, fmt.Sprintf("risk_level %q exceeds constitution max_risk_default %q", card.RiskLevel, constitution.MaxRiskDefault))
		}
	}

	return warnings
}

func riskRank(level string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return 1, true
	case "medium":
		return 2, true
	case "high":
		return 3, true
	default:
		return 0, false
	}
}
