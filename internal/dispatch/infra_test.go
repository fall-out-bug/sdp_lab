package dispatch

import (
	"strings"
	"testing"
)

func TestDetectInfrastructure(t *testing.T) {
	infra := DetectInfrastructure()
	if infra == nil {
		t.Fatal("DetectInfrastructure returned nil")
	}
	name := infra.Name()
	if name != "gastown" && name != "standalone" {
		t.Fatalf("unexpected Name() %q: must be gastown or standalone", name)
	}
}

func TestStandaloneInfra_Convoy(t *testing.T) {
	s := &StandaloneInfra{}

	id, err := s.CreateConvoy("myconvoy", []string{"ISSUE-1", "ISSUE-2"})
	if err != nil {
		t.Fatalf("CreateConvoy error: %v", err)
	}
	if !strings.HasPrefix(id, "standalone-") {
		t.Fatalf("expected id to start with standalone-, got %q", id)
	}
	if id != "standalone-myconvoy" {
		t.Fatalf("expected standalone-myconvoy, got %q", id)
	}

	status, err := s.ConvoyStatus("any-id")
	if err != nil {
		t.Fatalf("ConvoyStatus error: %v", err)
	}
	if status != "unknown" {
		t.Fatalf("expected unknown, got %q", status)
	}
}

func TestStandaloneInfra_Sling(t *testing.T) {
	s := &StandaloneInfra{}
	err := s.Sling("ISSUE-1", "rig-a", "agent-x")
	if err == nil {
		t.Fatal("expected error from Sling, got nil")
	}
	if !strings.Contains(err.Error(), "standalone mode") {
		t.Fatalf("expected error to contain 'standalone mode', got %q", err.Error())
	}
}
