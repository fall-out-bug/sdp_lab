package runtimeparity

import "testing"

func TestCompareEqual(t *testing.T) {
	a := CapabilitySet{Operations: []string{"a", "b"}, States: []string{"open"}, EvidenceKeys: []string{"intent"}, AllowedModels: []string{"glm-5"}}
	b := CapabilitySet{Operations: []string{"b", "a"}, States: []string{"open"}, EvidenceKeys: []string{"intent"}, AllowedModels: []string{"glm-5"}}
	res := Compare(a, b)
	if !res.Equal {
		t.Fatalf("expected equal, got %+v", res)
	}
}

func TestCompareMismatch(t *testing.T) {
	a := CapabilitySet{Operations: []string{"a"}, States: []string{"open"}, EvidenceKeys: []string{"intent"}, AllowedModels: []string{"glm-5"}}
	b := CapabilitySet{Operations: []string{"a", "b"}, States: []string{"open"}, EvidenceKeys: []string{"intent"}, AllowedModels: []string{"glm-5"}}
	res := Compare(a, b)
	if res.Equal {
		t.Fatalf("expected mismatch")
	}
}
