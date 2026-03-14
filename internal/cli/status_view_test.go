package cli

import (
	"strings"
	"testing"
)

func TestNewStatusViewFromBeads(t *testing.T) {
	tests := []struct {
		name           string
		items          []BeadsItem
		wantReady      int
		wantBlocked    int
		wantInProgress int
		wantRecommend  string
	}{
		{
			name:           "empty items",
			items:          []BeadsItem{},
			wantReady:      0,
			wantBlocked:    0,
			wantInProgress: 0,
			wantRecommend:  "No action required",
		},
		{
			name: "in_progress takes priority",
			items: []BeadsItem{
				{ID: "sdplab-1", Title: "Ready task", Status: "open", Priority: 1, BlockedBy: nil},
				{ID: "sdplab-2", Title: "In progress task", Status: "in_progress", Priority: 2, BlockedBy: nil},
			},
			wantReady:      1,
			wantInProgress: 1,
			wantRecommend:  "Continue sdplab-2",
		},
		{
			name: "ready items when no in_progress",
			items: []BeadsItem{
				{ID: "sdplab-1", Title: "High priority", Status: "open", Priority: 1, BlockedBy: nil},
				{ID: "sdplab-2", Title: "Low priority", Status: "open", Priority: 3, BlockedBy: nil},
			},
			wantReady:     2,
			wantRecommend: "Start sdplab-1",
		},
		{
			name: "blocked items show blockers",
			items: []BeadsItem{
				{ID: "sdplab-1", Title: "Blocked task", Status: "open", Priority: 1, BlockedBy: []string{"sdplab-0"}},
			},
			wantBlocked:   1,
			wantRecommend: "Resolve blockers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewStatusViewFromBeads(tt.items)
			if view.ReadyCount != tt.wantReady {
				t.Errorf("ReadyCount = %d, want %d", view.ReadyCount, tt.wantReady)
			}
			if view.BlockedCount != tt.wantBlocked {
				t.Errorf("BlockedCount = %d, want %d", view.BlockedCount, tt.wantBlocked)
			}
			if view.InProgressCount != tt.wantInProgress {
				t.Errorf("InProgressCount = %d, want %d", view.InProgressCount, tt.wantInProgress)
			}
			if !strings.Contains(view.NextAction.Recommended, tt.wantRecommend) {
				t.Errorf("Recommended = %q, want to contain %q", view.NextAction.Recommended, tt.wantRecommend)
			}
		})
	}
}

func TestStatusViewRenderText(t *testing.T) {
	items := []BeadsItem{
		{ID: "sdplab-1", Title: "Ready task", Status: "open", Priority: 1, BlockedBy: nil},
		{ID: "sdplab-2", Title: "Blocked task", Status: "open", Priority: 2, BlockedBy: []string{"sdplab-0"}},
	}
	view := NewStatusViewFromBeads(items)

	output := view.RenderText()
	if !strings.Contains(output, "Ready: 1") {
		t.Error("Text output should contain ready count")
	}
	if !strings.Contains(output, "Blocked: 1") {
		t.Error("Text output should contain blocked count")
	}
	if !strings.Contains(output, "sdplab-1") {
		t.Error("Text output should contain item ID")
	}
	if !strings.Contains(output, "Next action:") {
		t.Error("Text output should contain next action section")
	}
}

func TestStatusViewRenderJSON(t *testing.T) {
	items := []BeadsItem{
		{ID: "sdplab-1", Title: "Test task", Status: "open", Priority: 1, BlockedBy: nil},
	}
	view := NewStatusViewFromBeads(items)

	json, err := view.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}
	if !strings.Contains(json, `"spec_version"`) {
		t.Error("JSON output should contain spec_version")
	}
	if !strings.Contains(json, `"sdplab-1"`) {
		t.Error("JSON output should contain item ID")
	}
	if !strings.Contains(json, `"next_action"`) {
		t.Error("JSON output should contain next_action")
	}
}

func TestInstructionPayloadRenderText(t *testing.T) {
	status := &StatusView{
		SpecVersion:     "v1.0",
		ReadyCount:      1,
		BlockedCount:    0,
		InProgressCount: 0,
		Items: []StatusItem{
			{ID: "sdplab-1", Title: "Test", Status: "ready", Priority: 1},
		},
		NextAction: StatusNextStep{
			Recommended: "Start sdplab-1",
			Reason:      "Highest priority",
			Command:     "bd update sdplab-1 --status in_progress",
		},
	}

	payload := NewInstructionPayloadForAction("start", status)
	output := payload.RenderText()

	if !strings.Contains(output, "Instructions:") {
		t.Error("Text output should contain Instructions header")
	}
	if !strings.Contains(output, "sdplab-1") {
		t.Error("Text output should contain item ID")
	}
}

func TestInstructionPayloadRenderJSON(t *testing.T) {
	status := &StatusView{
		SpecVersion: "v1.0",
		NextAction:  StatusNextStep{Recommended: "Test", Reason: "Test reason"},
	}

	payload := NewInstructionPayloadForAction("check_status", status)
	json, err := payload.RenderJSON()

	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}
	if !strings.Contains(json, `"spec_version"`) {
		t.Error("JSON should contain spec_version")
	}
	if !strings.Contains(json, `"instructions"`) {
		t.Error("JSON should contain instructions array")
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		status      string
		hasBlockers bool
		want        string
	}{
		{"in_progress", false, "in_progress"},
		{"in_progress", true, "in_progress"},
		{"blocked", false, "blocked"},
		{"open", true, "blocked"},
		{"open", false, "ready"},
		{"ready", false, "ready"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := normalizeStatus(tt.status, tt.hasBlockers)
			if got != tt.want {
				t.Errorf("normalizeStatus(%q, %v) = %q, want %q", tt.status, tt.hasBlockers, got, tt.want)
			}
		})
	}
}

func TestStatusOrder(t *testing.T) {
	tests := []struct {
		status string
		want   int
	}{
		{"in_progress", 0},
		{"ready", 1},
		{"blocked", 2},
		{"unknown", 3},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := statusOrder(tt.status); got != tt.want {
				t.Errorf("statusOrder(%q) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}
