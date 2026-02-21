package swarm

import (
	"testing"
)

func TestNewCoordinator(t *testing.T) {
	c := NewCoordinator()
	if c == nil {
		t.Fatal("NewCoordinator returned nil")
	}
}

func TestCoordinator_Claim(t *testing.T) {
	c := NewCoordinator()
	s := c.Claim("p1", "i1", "run-1")
	if s == nil {
		t.Fatal("Claim returned nil")
	}
	if s.Phase != PhaseClaimed || s.ProjectID != "p1" || s.IssueID != "i1" {
		t.Errorf("Claim: %+v", s)
	}
}

func TestCoordinator_Transition(t *testing.T) {
	c := NewCoordinator()
	c.Claim("p1", "i1", "r1")
	s := c.Transition("p1", "i1", PhaseAnalyzing)
	if s == nil || s.Phase != PhaseAnalyzing {
		t.Errorf("Transition: %+v", s)
	}
	if c.Transition("p2", "i2", PhaseClaimed) != nil {
		t.Error("Transition on unknown key should return nil")
	}
}

func TestCoordinator_Get(t *testing.T) {
	c := NewCoordinator()
	c.Claim("p1", "i1", "r1")
	s := c.Get("p1", "i1")
	if s == nil || s.RunID != "r1" {
		t.Errorf("Get: %+v", s)
	}
	if c.Get("p2", "i2") != nil {
		t.Error("Get unknown should return nil")
	}
}
