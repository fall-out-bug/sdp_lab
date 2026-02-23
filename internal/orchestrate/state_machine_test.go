package orchestrate_test

import (
	"testing"

	"sdp_dev/internal/orchestrate"
)

func TestAdvanceInitToBuild(t *testing.T) {
	cp := orchestrate.CreateInitialCheckpoint("F004", "feature/F004-x", []string{"00-004-01", "00-004-02"})
	if cp.Phase != orchestrate.PhaseInit {
		t.Errorf("expected init phase, got %s", cp.Phase)
	}
	err := orchestrate.Advance(cp, []string{"00-004-01", "00-004-02"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if cp.Phase != orchestrate.PhaseBuild {
		t.Errorf("expected build phase, got %s", cp.Phase)
	}
	if len(cp.Workstreams) != 2 {
		t.Errorf("expected 2 workstreams, got %d", len(cp.Workstreams))
	}
}

func TestAdvanceBuildToReview(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID:   "F004",
		Phase:       orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{ID: "00-004-01", Status: "done"},
			{ID: "00-004-02", Status: "done"},
		},
	}
	err := orchestrate.Advance(cp, []string{"00-004-01", "00-004-02"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if cp.Phase != orchestrate.PhaseReview {
		t.Errorf("expected review phase, got %s", cp.Phase)
	}
}
