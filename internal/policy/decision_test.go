package policy

import "testing"

func TestDecideDefaultModel(t *testing.T) {
	res := Decide(DecisionRequest{IssueID: "id-1", Title: "Update CLI", ChangedPaths: []string{"src/cli/main.go"}})
	if res.PolicyVerdict != "allow" {
		t.Fatalf("expected allow, got %s", res.PolicyVerdict)
	}
	if res.SelectedModel != "glm-5" {
		t.Fatalf("expected glm-5, got %s", res.SelectedModel)
	}
}

func TestDecideForbiddenModelEscalates(t *testing.T) {
	res := Decide(DecisionRequest{IssueID: "id-2", Title: "Model check", PreferredModel: "gpt-5", ChangedPaths: []string{"src/runtime/run.go"}})
	if res.PolicyVerdict != "escalate" {
		t.Fatalf("expected escalate, got %s", res.PolicyVerdict)
	}
}

func TestDecideCriticalEscalates(t *testing.T) {
	res := Decide(DecisionRequest{IssueID: "id-3", Title: "Auth changes", PreferredModel: "glm-4.7", ChangedPaths: []string{"security/auth/policy.yaml"}})
	if res.RiskClass != "critical" {
		t.Fatalf("expected critical risk, got %s", res.RiskClass)
	}
	if res.PolicyVerdict != "escalate" {
		t.Fatalf("expected escalate, got %s", res.PolicyVerdict)
	}
}
