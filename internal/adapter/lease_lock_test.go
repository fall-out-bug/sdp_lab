package adapter

import (
	"testing"
)

func TestLeaseName(t *testing.T) {
	tests := []struct {
		issueID string
		want   string
	}{
		{"sdp_dev-4pg", "sdp-ar-sdp-dev-4pg"},
		{"sdp_dev-5l9.3", "sdp-ar-sdp-dev-5l9-3"},
		{"ABC_xyz", "sdp-ar-abc-xyz"},
		{"a", "sdp-ar-a"},
	}
	for _, tt := range tests {
		got := leaseName(tt.issueID)
		if got != tt.want {
			t.Errorf("leaseName(%q) = %q, want %q", tt.issueID, got, tt.want)
		}
		// DNS-1123: lowercase, alphanumeric, hyphens
		for _, r := range got {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				t.Errorf("leaseName(%q) produced invalid char %q in %q", tt.issueID, r, got)
			}
		}
		if len(got) > 63 {
			t.Errorf("leaseName(%q) = %q exceeds 63 chars", tt.issueID, got)
		}
	}
}
