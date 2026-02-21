package adapter

import (
	"testing"
)

func TestNewPolicyGate(t *testing.T) {
	g := NewPolicyGate()
	if g == nil {
		t.Fatal("NewPolicyGate returned nil")
	}
}

func TestPolicyGate_PreDispatchModelAllowlist(t *testing.T) {
	g := NewPolicyGate()

	r := g.PreDispatchModelAllowlist("glm-5")
	if !r.Passed {
		t.Errorf("glm-5 should pass: %+v", r)
	}

	r = g.PreDispatchModelAllowlist("unknown-model-xyz")
	if r.Passed {
		t.Error("unknown model should fail")
	}
}

func TestPolicyGate_PreCloseRiskThreshold(t *testing.T) {
	g := NewPolicyGate()

	r := g.PreCloseRiskThreshold("low")
	if !r.Passed {
		t.Errorf("low risk should pass: %+v", r)
	}

	r = g.PreCloseRiskThreshold("critical")
	if r.Passed {
		t.Error("critical risk should fail")
	}
}

func TestPolicyGate_PrePublishGoNoGo(t *testing.T) {
	g := NewPolicyGate()

	r := g.PrePublishGoNoGo(true)
	if !r.Passed {
		t.Errorf("evidence complete should pass: %+v", r)
	}

	r = g.PrePublishGoNoGo(false)
	if r.Passed {
		t.Error("evidence incomplete should fail")
	}
}
