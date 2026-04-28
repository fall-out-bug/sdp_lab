package orchestrate_test

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func TestFormatNextAction(t *testing.T) {
	tests := []struct {
		name string
		action *orchestrate.NextAction
		want   string
	}{
		{
			name:   "init",
			action: &orchestrate.NextAction{Action: "init"},
			want:   "Phase: init",
		},
		{
			name:   "build",
			action: &orchestrate.NextAction{Action: "build", WSID: "00-004-01", Feature: "F004"},
			want:   "00-004-01",
		},
		{
			name:   "review",
			action: &orchestrate.NextAction{Action: "review", Feature: "F004"},
			want:   "review",
		},
		{
			name:   "pr",
			action: &orchestrate.NextAction{Action: "pr", Feature: "F004"},
			want:   "pr",
		},
		{
			name:   "ci-loop with PR",
			action: &orchestrate.NextAction{Action: "ci-loop", Feature: "F004", PR: 42},
			want:   "PR #42",
		},
		{
			name:   "qa",
			action: &orchestrate.NextAction{Action: "qa", Feature: "F004"},
			want:   "qa",
		},
		{
			name:   "done",
			action: &orchestrate.NextAction{Action: "done"},
			want:   "done",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orchestrate.FormatNextAction(tt.action)
			if !strings.Contains(got, tt.want) {
				t.Errorf("FormatNextAction(%+v) = %q, should contain %q", tt.action, got, tt.want)
			}
		})
	}
}

func TestFormatCheckpointStatus(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID: "F004",
		Phase:     orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{ID: "00-004-01", Status: "done"},
			{ID: "00-004-02", Status: "pending"},
		},
	}
	action := &orchestrate.NextAction{Action: "build", WSID: "00-004-02", Feature: "F004"}
	got := orchestrate.FormatCheckpointStatus("F004", cp, []string{"00-004-01", "00-004-02"}, action)

	if !strings.Contains(got, "F004") {
		t.Error("should contain feature ID")
	}
	if !strings.Contains(got, "build") {
		t.Error("should contain phase")
	}
	if !strings.Contains(got, "00-004-01") {
		t.Error("should contain workstream ID")
	}
	if !strings.Contains(got, "1/2") {
		t.Error("should contain progress count")
	}
}

func TestFormatCheckpointStatus_NilCheckpoint(t *testing.T) {
	action := &orchestrate.NextAction{Action: "init"}
	got := orchestrate.FormatCheckpointStatus("F004", nil, []string{"00-004-01"}, action)
	if !strings.Contains(got, "00-004-01") {
		t.Error("should list pending workstreams when cp is nil")
	}
}
