package bridge

import "testing"

func TestProtocolFindingHashesStableNormalization(t *testing.T) {
	sourceA := FindingsSource{
		CheckName:  " SDP-PROTOCOL-CHECK ",
		Workflow:   " CI ",
		Repository: "Fall-Out-Bug/SDP_LAB",
		Branch:     " Dev ",
	}
	sourceB := FindingsSource{
		CheckName:  "sdp-protocol-check",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "dev",
	}

	findingA := ProtocolFinding{
		FindingKey: "ABC123",
		Severity:   " Warning ",
		Category:   " Roadmap ",
		Code:       " MISSING_FEATURE ",
		File:       "docs\\roadmap\\ROADMAP.md",
		Line:       42,
		Message:    " Feature   F001 missing ",
		Remediation: &Remediation{
			Hint: " Add roadmap entry ",
		},
		Context: ProtocolContext{
			FeatureID: "F001",
			WSID:      "00-001-01",
		},
	}

	findingB := ProtocolFinding{
		FindingKey: "abc123",
		Severity:   "warning",
		Category:   "roadmap",
		Code:       "missing_feature",
		File:       "docs/roadmap/roadmap.md",
		Line:       42,
		Message:    "feature f001    missing",
		Remediation: &Remediation{
			Hint: "add roadmap entry",
		},
		Context: ProtocolContext{
			FeatureID: "f001",
			WSID:      "00-001-01",
		},
	}

	identityA, payloadA := ProtocolFindingHashes(sourceA, findingA)
	identityB, payloadB := ProtocolFindingHashes(sourceB, findingB)

	if identityA == "" || payloadA == "" {
		t.Fatalf("expected non-empty protocol hashes, got identity=%q payload=%q", identityA, payloadA)
	}
	if identityA != identityB {
		t.Fatalf("expected stable protocol identity hash, got %q vs %q", identityA, identityB)
	}
	if payloadA != payloadB {
		t.Fatalf("expected stable protocol payload hash, got %q vs %q", payloadA, payloadB)
	}
}

func TestDedupeStoreDecisionFlow(t *testing.T) {
	store := NewDedupeStore()
	findingHash := "1111111111111111"
	payloadA := "aaaaaaaaaaaaaaaa"
	payloadB := "bbbbbbbbbbbbbbbb"

	decision := store.Decide(findingHash, payloadA)
	if decision.Action != DedupeCreate {
		t.Fatalf("expected create action, got %s", decision.Action)
	}

	store.RecordCreated(findingHash, payloadA, "sdplab-1")

	decision = store.Decide(findingHash, payloadA)
	if decision.Action != DedupeSkip {
		t.Fatalf("expected skip for unchanged payload, got %s", decision.Action)
	}

	decision = store.Decide(findingHash, payloadB)
	if decision.Action != DedupeUpdate {
		t.Fatalf("expected update for changed payload, got %s", decision.Action)
	}

	store.RecordUpdated(findingHash, payloadB)
	store.RecordClosed(findingHash)

	decision = store.Decide(findingHash, payloadB)
	if decision.Action != DedupeSkip {
		t.Fatalf("expected skip for unchanged closed finding, got %s", decision.Action)
	}

	decision = store.Decide(findingHash, payloadA)
	if decision.Action != DedupeReopenUpdate {
		t.Fatalf("expected reopen+update for changed closed finding, got %s", decision.Action)
	}
}

func TestDedupeStoreImportExistingPrefersOpenRecord(t *testing.T) {
	store := NewDedupeStore()
	findingHash := "2222222222222222"
	payloadClosed := "cccccccccccccccc"
	payloadOpen := "dddddddddddddddd"

	store.ImportExisting([]ExistingIssue{
		{
			ID:     "sdplab-closed",
			Status: "closed",
			Labels: []string{"ci-finding", findingHashLabel(findingHash), payloadHashLabel(payloadClosed)},
		},
		{
			ID:     "sdplab-open",
			Status: "open",
			Labels: []string{"ci-finding", findingHashLabel(findingHash), payloadHashLabel(payloadOpen)},
		},
	})

	decision := store.Decide(findingHash, payloadOpen)
	if decision.Action != DedupeSkip {
		t.Fatalf("expected skip for matching open record, got %s", decision.Action)
	}
	if decision.IssueID != "sdplab-open" {
		t.Fatalf("expected open issue id to win, got %s", decision.IssueID)
	}

	decision = store.Decide(findingHash, payloadClosed)
	if decision.Action != DedupeUpdate {
		t.Fatalf("expected update against open record for changed payload, got %s", decision.Action)
	}
}

func TestDedupeStoreAtLeastOnceDeliveryBehavior(t *testing.T) {
	store := NewDedupeStore()
	source := FindingsSource{
		CheckName:  "sdp-doc-sync",
		Workflow:   "CI",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "dev",
	}
	finding := DocsFinding{
		FindingKey: "abc123def4567890",
		Severity:   "error",
		Category:   "consistency",
		Code:       "BROKEN_LINK",
		File:       "docs/runbooks/CI_LOCAL_BRIDGE.md",
		Line:       12,
		Message:    "Link target does not exist",
	}

	findingHash, payloadHash := DocsFindingHashes(source, finding)
	decision := store.Decide(findingHash, payloadHash)
	if decision.Action != DedupeCreate {
		t.Fatalf("first delivery should create, got %s", decision.Action)
	}

	store.RecordCreated(findingHash, payloadHash, "sdplab-9")

	for i := 0; i < 3; i++ {
		decision = store.Decide(findingHash, payloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("duplicate delivery %d should skip, got %s", i+1, decision.Action)
		}
	}

	finding.Message = "Link target still missing after docs update"
	_, changedPayloadHash := DocsFindingHashes(source, finding)
	decision = store.Decide(findingHash, changedPayloadHash)
	if decision.Action != DedupeUpdate {
		t.Fatalf("changed payload should update existing issue, got %s", decision.Action)
	}
}
