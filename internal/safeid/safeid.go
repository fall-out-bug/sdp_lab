package safeid

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrInvalidIssueID is returned when issueID is unsafe for path construction.
var ErrInvalidIssueID = errors.New("issue_id contains path traversal characters or is empty")

// ValidateIssueID rejects issueIDs that could cause path traversal when used
// in filepath.Join(".sdp", "evidence", issueID+".json") or similar.
// Beads IDs are typically alphanumeric with hyphens/dots (e.g. sdp_dev-4pg).
func ValidateIssueID(issueID string) error {
	if issueID == "" {
		return ErrInvalidIssueID
	}
	if strings.Contains(issueID, "/") || strings.Contains(issueID, "\\") {
		return ErrInvalidIssueID
	}
	if strings.Contains(issueID, "..") {
		return ErrInvalidIssueID
	}
	// Ensure resolved path stays under base (no absolute or drive-relative)
	if filepath.IsAbs(issueID) {
		return ErrInvalidIssueID
	}
	return nil
}
