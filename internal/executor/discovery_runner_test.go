package executor

import (
	"testing"

	"sdp_dev/internal/control"
)

func TestExtractDiscoveryIdea(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Discovery: automate product discovery", "automate product discovery"},
		{"Discovery: ", ""},
		{"No prefix here", "No prefix here"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractDiscoveryIdea(tt.input)
		if got != tt.want {
			t.Errorf("extractDiscoveryIdea(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsDiscoveryCard(t *testing.T) {
	tests := []struct {
		name string
		card *control.FeatureCard
		want bool
	}{
		{
			name: "nil card",
			card: nil,
			want: false,
		},
		{
			name: "discovery type",
			card: &control.FeatureCard{IssueType: "discovery"},
			want: true,
		},
		{
			name: "other type",
			card: &control.FeatureCard{IssueType: "feature"},
			want: false,
		},
		{
			name: "empty type",
			card: &control.FeatureCard{IssueType: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDiscoveryCard(tt.card)
			if got != tt.want {
				t.Errorf("isDiscoveryCard(%v) = %v, want %v", tt.card, got, tt.want)
			}
		})
	}
}
