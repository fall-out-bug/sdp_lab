package intake

import (
	"testing"
)

func TestNormalize_nil(t *testing.T) {
	if err := Normalize(nil); err != nil {
		t.Errorf("Normalize(nil) should not error: %v", err)
	}
}

func TestNormalize_defaults(t *testing.T) {
	req := &Request{}
	if err := Normalize(req); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if req.ProjectID != "default" {
		t.Errorf("ProjectID = %q, want default", req.ProjectID)
	}
	if req.Priority != 1 {
		t.Errorf("Priority = %d, want 1", req.Priority)
	}
	if req.Source != "api" {
		t.Errorf("Source = %q, want api", req.Source)
	}
}

func TestNormalize_preserves(t *testing.T) {
	req := &Request{
		ProjectID: "my-proj",
		Priority:  5,
		Source:    "telegram",
	}
	if err := Normalize(req); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if req.ProjectID != "my-proj" || req.Priority != 5 || req.Source != "telegram" {
		t.Errorf("Normalize should preserve: %+v", req)
	}
}
