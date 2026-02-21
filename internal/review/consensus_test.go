package review

import (
	"testing"
)

func TestConsensus_empty(t *testing.T) {
	r := Consensus(nil)
	if r.Consensus != "needs_changes" {
		t.Errorf("empty: Consensus=%q", r.Consensus)
	}
}

func TestConsensus_approve(t *testing.T) {
	v := []ReviewVerdict{
		{PersonaID: "a", Verdict: "approve"},
		{PersonaID: "b", Verdict: "approve"},
	}
	r := Consensus(v)
	if !r.Approved || r.Consensus != "approve" {
		t.Errorf("approve: %+v", r)
	}
}

func TestConsensus_reject(t *testing.T) {
	v := []ReviewVerdict{
		{PersonaID: "a", Verdict: "approve"},
		{PersonaID: "b", Verdict: "reject"},
	}
	r := Consensus(v)
	if !r.Rejected || r.Consensus != "reject" {
		t.Errorf("reject: %+v", r)
	}
}

func TestConsensus_needsChanges(t *testing.T) {
	// threshold=(3+1)/2=2; approveCount=1 < 2 -> needs_changes
	v := []ReviewVerdict{
		{PersonaID: "a", Verdict: "approve"},
		{PersonaID: "b", Verdict: "needs_changes"},
		{PersonaID: "c", Verdict: "needs_changes"},
	}
	r := Consensus(v)
	if !r.NeedsChanges || r.Consensus != "needs_changes" {
		t.Errorf("needs_changes: %+v", r)
	}
}

func TestConsensus_approveThreshold(t *testing.T) {
	// 3 approve, threshold=2 -> approve
	v := []ReviewVerdict{
		{PersonaID: "a", Verdict: "approve"},
		{PersonaID: "b", Verdict: "approve"},
		{PersonaID: "c", Verdict: "approve"},
	}
	r := Consensus(v)
	if !r.Approved || r.Consensus != "approve" {
		t.Errorf("approve threshold: %+v", r)
	}
}

func TestConsensus_dissenting(t *testing.T) {
	v := []ReviewVerdict{
		{PersonaID: "sec", Verdict: "reject"},
		{PersonaID: "dx", Verdict: "approve"},
	}
	r := Consensus(v)
	if len(r.Dissenting) == 0 || !r.Rejected {
		t.Errorf("dissenting: %+v", r)
	}
}
