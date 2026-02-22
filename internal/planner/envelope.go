package planner

import "fmt"

type PlanningInput struct {
	FeatureText string   `json:"feature_text"`
	Repo        string   `json:"repo"`
	RiskClass   string   `json:"risk_class"`
	Lane        string   `json:"lane"`
	Model       string   `json:"model"`
	Boundaries  []string `json:"boundaries"`
}

type ConstraintEnvelope struct {
	FeatureText string   `json:"feature_text"`
	Repo        string   `json:"repo"`
	RiskClass   string   `json:"risk_class"`
	Lane        string   `json:"lane"`
	Model       string   `json:"model"`
	Boundaries  []string `json:"boundaries"`
}

func BuildConstraintEnvelope(in PlanningInput) (ConstraintEnvelope, error) {
	if in.FeatureText == "" {
		return ConstraintEnvelope{}, fmt.Errorf("feature_text is required")
	}
	if in.Repo == "" {
		return ConstraintEnvelope{}, fmt.Errorf("repo is required")
	}
	if in.RiskClass == "" {
		in.RiskClass = "medium"
	}
	if in.Lane == "" {
		in.Lane = "commit"
	}
	if in.Model == "" {
		in.Model = "glm-5"
	}
	return ConstraintEnvelope{
		FeatureText: in.FeatureText,
		Repo:        in.Repo,
		RiskClass:   in.RiskClass,
		Lane:        in.Lane,
		Model:       in.Model,
		Boundaries:  in.Boundaries,
	}, nil
}
