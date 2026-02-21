package main

import (
	"testing"
)

func TestHasLabel(t *testing.T) {
	tests := []struct {
		labels []string
		name   string
		want   bool
	}{
		{[]string{"autonomy", "strict-evidence"}, "autonomy", true},
		{[]string{"autonomy", "strict-evidence"}, "strict-evidence", true},
		{[]string{"autonomy"}, "strict-evidence", false},
		{[]string{}, "autonomy", false},
		{[]string{"workstream:generic"}, "workstream:generic", true},
	}
	for _, tt := range tests {
		got := hasLabel(tt.labels, tt.name)
		if got != tt.want {
			t.Errorf("hasLabel(%v, %q) = %v, want %v", tt.labels, tt.name, got, tt.want)
		}
	}
}

func TestHasWorkstreamLabel(t *testing.T) {
	// supportedWorkstreams default includes workstream:generic
	tests := []struct {
		labels []string
		want   bool
	}{
		{[]string{"workstream:generic"}, true},
		{[]string{"workstream:policy-slugify-trim"}, true},
		{[]string{"autonomy"}, false},
		{[]string{}, false},
	}
	for _, tt := range tests {
		got := hasWorkstreamLabel(tt.labels)
		if got != tt.want {
			t.Errorf("hasWorkstreamLabel(%v) = %v, want %v", tt.labels, got, tt.want)
		}
	}
}

func TestLaneFromLabels(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
	}{
		{[]string{"lane:commit"}, "commit"},
		{[]string{"lane:explore"}, "explore"},
		{[]string{"lane:other"}, "commit"}, // invalid fallback
		{[]string{}, "commit"},
		{[]string{"autonomy", "lane:explore"}, "explore"},
	}
	for _, tt := range tests {
		got := laneFromLabels(tt.labels)
		if got != tt.want {
			t.Errorf("laneFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestAllowedPrefixesFromLabels(t *testing.T) {
	restricted := []string{"internal/policy/", "internal/evidence/", "cmd/", "docs/", "specs/", "scripts/"}
	forbidden := []string{"internal/", "cmd/", "docs/", "specs/", "scripts/", "deploy/"}
	tests := []struct {
		labels []string
		want   []string
	}{
		{[]string{"workstream:policy-slugify-trim"}, restricted},
		{[]string{"workstream:generic"}, forbidden},
		{[]string{"workstream:self-improvement"}, forbidden},
		{[]string{}, forbidden},
	}
	for _, tt := range tests {
		got := allowedPrefixesFromLabels(tt.labels)
		if len(got) != len(tt.want) {
			t.Errorf("allowedPrefixesFromLabels(%v) = %v, want %v", tt.labels, got, tt.want)
		}
	}
}

func TestDepsSatisfied(t *testing.T) {
	closed := issue{ID: "dep-1", Status: "closed"}
	open := issue{ID: "dep-2", Status: "open"}
	byID := map[string]issue{"dep-1": closed, "dep-2": open}

	tests := []struct {
		name string
		it   issue
		want bool
	}{
		{"no deps", issue{ID: "x", Dependencies: []dep{}}, true},
		{"parent-child skip", issue{ID: "x", Dependencies: []dep{{Type: "parent-child"}}}, true},
		{"dep closed", issue{ID: "x", Dependencies: []dep{{DependsOnID: "dep-1"}}}, true},
		{"dep open", issue{ID: "x", Dependencies: []dep{{DependsOnID: "dep-2"}}}, false},
		{"dep missing", issue{ID: "x", Dependencies: []dep{{DependsOnID: "missing"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := depsSatisfied(tt.it, byID)
			if got != tt.want {
				t.Errorf("depsSatisfied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelFromLabels(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
		wantErr bool
	}{
		{[]string{}, "glm-5", false}, // default
		{[]string{"model:glm-5"}, "glm-5", false},
		{[]string{"model:glm-4.7"}, "glm-4.7", false},
	}
	for _, tt := range tests {
		got, err := modelFromLabels(tt.labels)
		if (err != nil) != tt.wantErr {
			t.Errorf("modelFromLabels(%v) err = %v, wantErr %v", tt.labels, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("modelFromLabels(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"pure json", []byte(`{"id":"x"}`), `{"id":"x"}`},
		{"leading noise", []byte(`some output\n{"id":"x"}`), `{"id":"x"}`},
		{"array", []byte(`[{"id":"x"}]`), `[{"id":"x"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(extractJSON(tt.in))
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
