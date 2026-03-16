package orchestrate

import "testing"

func TestRedirectToBuildForBlockingFindingsFromReview(t *testing.T) {
	cp := &Checkpoint{
		FeatureID: "F090",
		Phase:     PhaseReview,
		Workstreams: []WSStatus{
			{ID: "00-090-01", Status: "done"},
			{ID: "00-090-02", Status: "done"},
			{ID: "00-090-03", Status: "done"},
		},
		Review: &ReviewStatus{Iteration: 2, Status: "approved"},
	}
	findings := []BlockingFinding{{ID: "sdplab-find-1", Labels: []string{"sdp-finding", "blocking", "00-090-03"}}}

	targetWS, ids, err := RedirectToBuildForBlockingFindings(cp, PhaseReview, findings)
	if err != nil {
		t.Fatalf("RedirectToBuildForBlockingFindings returned error: %v", err)
	}
	if targetWS != "00-090-03" {
		t.Fatalf("targetWS = %q, want 00-090-03", targetWS)
	}
	if cp.Phase != PhaseBuild {
		t.Fatalf("phase = %q, want build", cp.Phase)
	}
	if cp.Workstreams[2].Status != "pending" {
		t.Fatalf("target ws status = %q, want pending", cp.Workstreams[2].Status)
	}
	if cp.Review == nil || cp.Review.Status != "pending" {
		t.Fatalf("review status not reset: %+v", cp.Review)
	}
	if len(ids) != 1 || ids[0] != "sdplab-find-1" {
		t.Fatalf("unexpected finding ids: %v", ids)
	}
}

func TestRedirectToBuildForBlockingFindingsFromQAFallbacksToCheckpointWS(t *testing.T) {
	cp := &Checkpoint{
		FeatureID: "F090",
		Phase:     PhaseQA,
		Workstreams: []WSStatus{
			{ID: "00-090-01", Status: "done"},
			{ID: "00-090-02", Status: "done"},
			{ID: "00-090-03", Status: "done"},
		},
		QA: &QAStatus{Iteration: 1, Status: "passed"},
	}
	findings := []BlockingFinding{{ID: "sdplab-find-2", Labels: []string{"sdp-finding", "blocking"}}}

	targetWS, _, err := RedirectToBuildForBlockingFindings(cp, PhaseQA, findings)
	if err != nil {
		t.Fatalf("RedirectToBuildForBlockingFindings returned error: %v", err)
	}
	if targetWS != "00-090-03" {
		t.Fatalf("targetWS = %q, want checkpoint fallback workstream", targetWS)
	}
	if cp.Phase != PhaseBuild {
		t.Fatalf("phase = %q, want build", cp.Phase)
	}
	if cp.QA == nil || cp.QA.Status != "pending" {
		t.Fatalf("qa status not reset: %+v", cp.QA)
	}
	if cp.Workstreams[2].Status != "pending" {
		t.Fatalf("target ws status = %q, want pending", cp.Workstreams[2].Status)
	}
}
