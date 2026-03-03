package harness

import (
	"testing"
	"time"
)

func TestClassifyClarification(t *testing.T) {
	t.Run("additive", func(t *testing.T) {
		decision := ClassifyClarification(&ClarificationChange{
			AddMetrics: []RequiredMetric{{Name: "latency_ms"}},
		})
		if decision.Classification != ClarificationAdditive {
			t.Fatalf("expected additive, got %s", decision.Classification)
		}
		if decision.RequiresApproval {
			t.Fatal("additive clarification should not require approval")
		}
	})

	t.Run("reductive", func(t *testing.T) {
		decision := ClassifyClarification(&ClarificationChange{
			RemoveMetrics: []string{"coverage"},
		})
		if decision.Classification != ClarificationReductive {
			t.Fatalf("expected reductive, got %s", decision.Classification)
		}
		if !decision.RequiresApproval || !decision.Blocking {
			t.Fatal("reductive clarification must require approval and be blocking")
		}
	})

	t.Run("policy-sensitive", func(t *testing.T) {
		decision := ClassifyClarification(&ClarificationChange{PolicySensitive: true})
		if decision.Classification != ClarificationPolicySensitive {
			t.Fatalf("expected policy_sensitive, got %s", decision.Classification)
		}
		if !decision.RequiresApproval || !decision.Blocking {
			t.Fatal("policy-sensitive clarification must require approval and be blocking")
		}
	})
}

func TestClassifyClarificationText(t *testing.T) {
	t.Run("additive text", func(t *testing.T) {
		decision := ClassifyClarificationText("добавь еще метрику latency")
		if decision.Classification != ClarificationAdditive {
			t.Fatalf("expected additive, got %s", decision.Classification)
		}
	})

	t.Run("reductive text", func(t *testing.T) {
		decision := ClassifyClarificationText("убери lint gate")
		if decision.Classification != ClarificationReductive {
			t.Fatalf("expected reductive, got %s", decision.Classification)
		}
		if !decision.RequiresApproval {
			t.Fatal("expected approval requirement for reductive text")
		}
	})

	t.Run("policy text", func(t *testing.T) {
		decision := ClassifyClarificationText("нужно поменять security policy")
		if decision.Classification != ClarificationPolicySensitive {
			t.Fatalf("expected policy-sensitive, got %s", decision.Classification)
		}
	})
}

func TestApplyClarification(t *testing.T) {
	contract := baseContract()
	change := &ClarificationChange{
		ID:                 "CR-42",
		Reason:             "User added latency requirement",
		AddMetrics:         []RequiredMetric{{Name: "latency_ms", Direction: "at_most", Target: 500}},
		AddEvidence:        []string{"trace_link"},
		EnableQualityGates: []string{"typecheck"},
	}

	decision, err := ApplyClarification(contract, change, "", time.Time{})
	if err != nil {
		t.Fatalf("apply additive clarification: %v", err)
	}
	if decision.Classification != ClarificationAdditive {
		t.Fatalf("expected additive classification, got %s", decision.Classification)
	}
	if contract.Version != "v2" {
		t.Fatalf("expected version v2 after apply, got %s", contract.Version)
	}

	hasLatency := false
	for _, metric := range contract.RequiredMetrics {
		if metric.Name == "latency_ms" {
			hasLatency = true
			break
		}
	}
	if !hasLatency {
		t.Fatal("expected latency_ms metric to be added")
	}

	if len(contract.ChangeRequests) != 1 {
		t.Fatalf("expected one change request, got %d", len(contract.ChangeRequests))
	}
}

func TestApplyClarificationRequiresApprovalForReductive(t *testing.T) {
	contract := baseContract()
	change := &ClarificationChange{RemoveMetrics: []string{"coverage"}}

	_, err := ApplyClarification(contract, change, "", time.Time{})
	if err == nil {
		t.Fatal("expected error for reductive clarification without approval")
	}

	_, err = ApplyClarification(contract, change, "tech-lead", time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected reductive clarification with approval to pass: %v", err)
	}

	for _, metric := range contract.RequiredMetrics {
		if metric.Name == "coverage" {
			t.Fatal("expected coverage metric to be removed")
		}
	}
}
