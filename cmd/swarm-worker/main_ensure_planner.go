package main

import (
	"os"
	"path/filepath"
)

func ensurePlannerEnvelopeFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "planner", "envelope.go")
	testPath := filepath.Join(repo, "internal", "planner", "envelope_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package planner

import "fmt"

type PlanningInput struct {
	FeatureText string   ` + "`json:\"feature_text\"`" + `
	Repo        string   ` + "`json:\"repo\"`" + `
	RiskClass   string   ` + "`json:\"risk_class\"`" + `
	Lane        string   ` + "`json:\"lane\"`" + `
	Model       string   ` + "`json:\"model\"`" + `
	Boundaries  []string ` + "`json:\"boundaries\"`" + `
}

type ConstraintEnvelope struct {
	FeatureText string   ` + "`json:\"feature_text\"`" + `
	Repo        string   ` + "`json:\"repo\"`" + `
	RiskClass   string   ` + "`json:\"risk_class\"`" + `
	Lane        string   ` + "`json:\"lane\"`" + `
	Model       string   ` + "`json:\"model\"`" + `
	Boundaries  []string ` + "`json:\"boundaries\"`" + `
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
`
	test := `package planner

import "testing"

func TestBuildConstraintEnvelopeDefaults(t *testing.T) {
	out, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "Add parallel swarm", Repo: "fall-out-bug/sdp_private", Boundaries: []string{"internal/", "cmd/"}})
	if err != nil {
		t.Fatalf("build envelope: %v", err)
	}
	if out.RiskClass != "medium" || out.Lane != "commit" || out.Model != "glm-5" {
		t.Fatalf("unexpected defaults: %#v", out)
	}
	if len(out.Boundaries) != 2 {
		t.Fatalf("unexpected boundaries: %#v", out.Boundaries)
	}
}

func TestBuildConstraintEnvelopeRequiresInputs(t *testing.T) {
	if _, err := BuildConstraintEnvelope(PlanningInput{Repo: "fall-out-bug/sdp_private"}); err == nil {
		t.Fatal("expected feature_text validation error")
	}
	if _, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "x"}); err == nil {
		t.Fatal("expected repo validation error")
	}
}
`
	if err := os.WriteFile(corePath, []byte(core), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(testPath, []byte(test), 0o644); err != nil {
		return err
	}
	return nil
}
