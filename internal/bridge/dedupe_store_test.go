package bridge

import (
	"testing"
)

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

// TestDocsFindingHashesStableNormalization verifies deterministic hash output
// for DocsFinding across cosmetic input differences.
func TestDocsFindingHashesStableNormalization(t *testing.T) {
	sourceA := FindingsSource{
		CheckName:  " SDP-DOC-SYNC ",
		Workflow:   " CI ",
		Repository: "Fall-Out-Bug/SDP_LAB",
		Branch:     " Dev ",
	}
	sourceB := FindingsSource{
		CheckName:  "sdp-doc-sync",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "dev",
	}

	findingA := DocsFinding{
		FindingKey: "XYZ789",
		Severity:   " Error ",
		Category:   " Consistency ",
		Code:       " BROKEN_LINK ",
		File:       "docs\\runbooks\\CI_LOCAL_BRIDGE.md",
		Line:       12,
		Column:     5,
		Message:    " Link   target   missing ",
		Remediation: &Remediation{
			Hint: " Fix the link ",
		},
		Context: DocsContext{
			LinkTarget: " docs/missing.md ",
			LinkText:   " missing doc ",
		},
	}

	findingB := DocsFinding{
		FindingKey: "xyz789",
		Severity:   "error",
		Category:   "consistency",
		Code:       "broken_link",
		File:       "docs/runbooks/ci_local_bridge.md",
		Line:       12,
		Column:     5,
		Message:    "link target missing",
		Remediation: &Remediation{
			Hint: "fix the link",
		},
		Context: DocsContext{
			LinkTarget: "docs/missing.md",
			LinkText:   "missing doc",
		},
	}

	identityA, payloadA := DocsFindingHashes(sourceA, findingA)
	identityB, payloadB := DocsFindingHashes(sourceB, findingB)

	if identityA == "" || payloadA == "" {
		t.Fatalf("expected non-empty docs hashes, got identity=%q payload=%q", identityA, payloadA)
	}
	if identityA != identityB {
		t.Fatalf("expected stable docs identity hash, got %q vs %q", identityA, identityB)
	}
	if payloadA != payloadB {
		t.Fatalf("expected stable docs payload hash, got %q vs %q", payloadA, payloadB)
	}
}

// TestTypedFindingHashesStable verifies deterministic hash for TypedFinding.
func TestTypedFindingHashesStable(t *testing.T) {
	fA := TypedFinding{
		DedupKey: " DEDUP-KEY-1 ",
		Source:   " PROTOCOL ",
		Title:    " Missing Roadmap Entry ",
		Severity: " Warning ",
		Priority: 3,
		Blocking: true,
	}
	fB := TypedFinding{
		DedupKey: "dedup-key-1",
		Source:   "protocol",
		Title:    "missing roadmap entry",
		Severity: "warning",
		Priority: 3,
		Blocking: true,
	}

	identA, payloadA := TypedFindingHashes(fA)
	identB, payloadB := TypedFindingHashes(fB)

	if identA == "" || payloadA == "" {
		t.Fatalf("expected non-empty typed finding hashes, got identity=%q payload=%q", identA, payloadA)
	}
	if identA != identB {
		t.Fatalf("expected stable typed identity hash, got %q vs %q", identA, identB)
	}
	if payloadA != payloadB {
		t.Fatalf("expected stable typed payload hash, got %q vs %q", payloadA, payloadB)
	}
}

// TestClosedFindingNotReopened tests that a closed finding with unchanged
// payload stays closed when reprocessed through the full Decide flow.
func TestClosedFindingNotReopened(t *testing.T) {
	store := NewDedupeStore()
	source := FindingsSource{
		CheckName:  "sdp-protocol-check",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "main",
	}
	finding := ProtocolFinding{
		FindingKey: "CLOSED-001",
		Severity:   "error",
		Category:   "roadmap",
		Code:       "missing_entry",
		File:       "docs/roadmap.md",
		Line:       10,
		Message:    "Missing F077 in roadmap",
	}

	findingHash, payloadHash := ProtocolFindingHashes(source, finding)

	// Simulate: CI found this, we created an issue, then it was resolved and closed.
	decision := store.Decide(findingHash, payloadHash)
	if decision.Action != DedupeCreate {
		t.Fatalf("initial finding should create, got %s", decision.Action)
	}
	store.RecordCreated(findingHash, payloadHash, "sdplab-closed-issue")
	store.RecordClosed(findingHash)

	// The same CI run delivers the finding again (at-least-once).
	// Since the issue is closed and the payload is identical, it must stay closed.
	for i := 0; i < 5; i++ {
		decision = store.Decide(findingHash, payloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("closed finding with same payload should skip on redelivery %d, got %s", i+1, decision.Action)
		}
		if decision.IssueID != "sdplab-closed-issue" {
			t.Fatalf("expected original issue ID on redelivery, got %s", decision.IssueID)
		}
	}
}

// TestClosedFindingReopenedOnPayloadChange tests that a closed finding is
// reopened ONLY when the payload actually changes.
func TestClosedFindingReopenedOnPayloadChange(t *testing.T) {
	store := NewDedupeStore()
	source := FindingsSource{
		CheckName:  "sdp-protocol-check",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "main",
	}

	originalFinding := ProtocolFinding{
		FindingKey: "REOPEN-001",
		Severity:   "error",
		Category:   "roadmap",
		Code:       "missing_entry",
		File:       "docs/roadmap.md",
		Line:       10,
		Message:    "Missing F077 in roadmap",
	}

	findingHash, payloadHashV1 := ProtocolFindingHashes(source, originalFinding)
	store.RecordCreated(findingHash, payloadHashV1, "sdplab-reopen-1")
	store.RecordClosed(findingHash)

	// New CI run produces the same finding key but the message changed.
	changedFinding := originalFinding
	changedFinding.Message = "Missing F077 and F078 in roadmap"
	_, payloadHashV2 := ProtocolFindingHashes(source, changedFinding)

	decision := store.Decide(findingHash, payloadHashV2)
	if decision.Action != DedupeReopenUpdate {
		t.Fatalf("changed payload on closed finding should reopen, got %s", decision.Action)
	}
	if decision.IssueID != "sdplab-reopen-1" {
		t.Fatalf("should reference original issue, got %s", decision.IssueID)
	}
}

// TestProtocolFindingAtLeastOnceDelivery simulates at-least-once delivery
// using ProtocolFindingHashes to verify end-to-end idempotency.
func TestProtocolFindingAtLeastOnceDelivery(t *testing.T) {
	store := NewDedupeStore()
	source := FindingsSource{
		CheckName:  "sdp-protocol-check",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "dev",
	}
	finding := ProtocolFinding{
		FindingKey: "ATLEAST-ONCE-PROTOCOL",
		Severity:   "warning",
		Category:   "roadmap",
		Code:       "drift",
		File:       "docs/roadmap.md",
		Line:       30,
		Message:    "F077 roadmap entry drifted from spec",
		Context: ProtocolContext{
			FeatureID: "F077",
			WSID:      "00-077-03",
		},
	}

	findingHash, payloadHash := ProtocolFindingHashes(source, finding)

	// First delivery creates the issue.
	decision := store.Decide(findingHash, payloadHash)
	if decision.Action != DedupeCreate {
		t.Fatalf("first delivery should create, got %s", decision.Action)
	}
	store.RecordCreated(findingHash, payloadHash, "sdplab-proto-1")

	// Simulate 10 redeliveries (at-least-once semantics).
	for i := 0; i < 10; i++ {
		decision = store.Decide(findingHash, payloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("redelivery %d should skip, got %s", i+1, decision.Action)
		}
		if decision.IssueID != "sdplab-proto-1" {
			t.Fatalf("redelivery %d should reference existing issue, got %s", i+1, decision.IssueID)
		}
	}

	// A new CI run detects a change in the finding.
	finding.Message = "F077 roadmap entry still drifted — severity increased"
	_, newPayloadHash := ProtocolFindingHashes(source, finding)

	decision = store.Decide(findingHash, newPayloadHash)
	if decision.Action != DedupeUpdate {
		t.Fatalf("changed payload should update, got %s", decision.Action)
	}
	if decision.IssueID != "sdplab-proto-1" {
		t.Fatalf("update should reference existing issue, got %s", decision.IssueID)
	}

	// Record the update and simulate more redeliveries with the new payload.
	store.RecordUpdated(findingHash, newPayloadHash)
	for i := 0; i < 10; i++ {
		decision = store.Decide(findingHash, newPayloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("post-update redelivery %d should skip, got %s", i+1, decision.Action)
		}
	}
}

// TestTypedFindingAtLeastOnceDelivery simulates at-least-once delivery
// for TypedFinding to verify full lifecycle idempotency.
func TestTypedFindingAtLeastOnceDelivery(t *testing.T) {
	store := NewDedupeStore()

	finding := TypedFinding{
		DedupKey:   "typed-dedup-001",
		Source:     "protocol",
		Title:      "Spec drift in SDP pipeline",
		Severity:   "warning",
		Priority:   2,
		Blocking:   false,
		Summary:    "The spec and implementation have diverged",
		FeatureID:  "F077",
		WSID:       "00-077-03",
	}

	findingHash, payloadHash := TypedFindingHashes(finding)

	// First delivery.
	decision := store.Decide(findingHash, payloadHash)
	if decision.Action != DedupeCreate {
		t.Fatalf("first delivery should create, got %s", decision.Action)
	}
	store.RecordCreated(findingHash, payloadHash, "sdplab-typed-1")

	// Repeated redeliveries.
	for i := 0; i < 5; i++ {
		decision = store.Decide(findingHash, payloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("typed redelivery %d should skip, got %s", i+1, decision.Action)
		}
	}

	// Close the issue, then redeliver the same payload — must stay closed.
	store.RecordClosed(findingHash)
	for i := 0; i < 5; i++ {
		decision = store.Decide(findingHash, payloadHash)
		if decision.Action != DedupeSkip {
			t.Fatalf("closed typed redelivery %d should skip, got %s", i+1, decision.Action)
		}
	}

	// New data arrives — payload changes, should reopen.
	finding.Summary = "Updated: spec and impl still diverge, now blocking"
	finding.Blocking = true
	_, newPayloadHash := TypedFindingHashes(finding)

	decision = store.Decide(findingHash, newPayloadHash)
	if decision.Action != DedupeReopenUpdate {
		t.Fatalf("changed payload on closed typed finding should reopen, got %s", decision.Action)
	}
}

// TestHashDeterminismAcrossMultipleCalls verifies that the same inputs
// always produce the same hash, even when called many times.
func TestHashDeterminismAcrossMultipleCalls(t *testing.T) {
	source := FindingsSource{
		CheckName:  "sdp-determinism-check",
		Workflow:   "ci",
		Repository: "fall-out-bug/sdp_lab",
		Branch:     "main",
	}
	finding := ProtocolFinding{
		FindingKey: "DETERM-001",
		Severity:   "error",
		Category:   "protocol",
		Code:       "violation",
		File:       "docs/spec.md",
		Line:       100,
		Message:    "Spec violation detected",
	}

	var lastIdent, lastPayload string
	for i := 0; i < 100; i++ {
		ident, payload := ProtocolFindingHashes(source, finding)
		if lastIdent != "" && ident != lastIdent {
			t.Fatalf("identity hash changed on iteration %d: %q vs %q", i, lastIdent, ident)
		}
		if lastPayload != "" && payload != lastPayload {
			t.Fatalf("payload hash changed on iteration %d: %q vs %q", i, lastPayload, payload)
		}
		lastIdent = ident
		lastPayload = payload
	}
}
