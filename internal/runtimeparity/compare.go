package runtimeparity

import "sort"

type CapabilitySet struct {
	Runtime       string   `json:"runtime"`
	Operations    []string `json:"operations"`
	States        []string `json:"states"`
	EvidenceKeys  []string `json:"evidence_keys"`
	AllowedModels []string `json:"allowed_models"`
}

type Comparison struct {
	Equal       bool     `json:"equal"`
	Differences []string `json:"differences"`
}

func Compare(a, b CapabilitySet) Comparison {
	diffs := make([]string, 0)
	if !sameSet(a.Operations, b.Operations) {
		diffs = append(diffs, "operations mismatch")
	}
	if !sameSet(a.States, b.States) {
		diffs = append(diffs, "states mismatch")
	}
	if !sameSet(a.EvidenceKeys, b.EvidenceKeys) {
		diffs = append(diffs, "evidence_keys mismatch")
	}
	if !sameSet(a.AllowedModels, b.AllowedModels) {
		diffs = append(diffs, "allowed_models mismatch")
	}
	return Comparison{Equal: len(diffs) == 0, Differences: diffs}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string{}, a...)
	bb := append([]string{}, b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
