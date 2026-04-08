package control

import "testing"

func TestBdToCard_MapsIssueType(t *testing.T) {
	bd := bdIssue{
		ID:    "beads-001",
		Title: "Discovery: automate product discovery",
		Type:  "discovery",
	}
	card := bdToCard(bd)
	if card.IssueType != "discovery" {
		t.Errorf("expected IssueType=discovery, got %q", card.IssueType)
	}
}
