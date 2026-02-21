package safeid

import "testing"

func TestValidateIssueID(t *testing.T) {
	tests := []struct {
		name    string
		issueID string
		wantErr bool
	}{
		{"valid", "sdp_dev-4pg", false},
		{"valid with dot", "sdp_dev-2aq.1", false},
		{"empty", "", true},
		{"slash", "sdp_dev/evil", true},
		{"backslash", "sdp_dev\\evil", true},
		{"parent dir", "..", true},
		{"parent in middle", "sdp_dev/../etc/passwd", true},
		{"trailing slash", "sdp_dev-4pg/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIssueID(tt.issueID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIssueID(%q) err = %v, wantErr %v", tt.issueID, err, tt.wantErr)
			}
		})
	}
}
