package bridge

import (
	"strings"
	"testing"
)

func TestTypedFindingHashesStableNormalization(t *testing.T) {
	a := TypedFinding{
		Source:       FindingSourceReview,
		FeatureID:    " F054 ",
		WSID:         "00-054-03",
		Blocking:     true,
		Title:        " Missing WS Verdict Evidence ",
		Summary:      " review found missing evidence ",
		Description:  " Required ws-verdict artifacts are missing. ",
		Severity:     " P1 ",
		Priority:     1,
		PRURL:        " https://github.com/org/repo/pull/123 ",
		EvidenceRef:  " docs/reviews/F054-REVIEW-SUMMARY.md ",
		TraceRef:     " trace://f054 ",
		DriftVerdict: " no_drift ",
		DedupKey:     " WS-VERDICT-MISSING ",
	}
	b := TypedFinding{
		Source:       FindingSourceReview,
		FeatureID:    "f054",
		WSID:         "00-054-03",
		Blocking:     true,
		Title:        "missing ws verdict evidence",
		Summary:      "Review found missing evidence",
		Description:  "required ws-verdict artifacts are missing.",
		Severity:     "p1",
		Priority:     1,
		PRURL:        "https://github.com/org/repo/pull/123",
		EvidenceRef:  "docs/reviews/F054-REVIEW-SUMMARY.md",
		TraceRef:     "trace://f054",
		DriftVerdict: "no_drift",
		DedupKey:     "ws-verdict-missing",
	}

	identityA, payloadA := TypedFindingHashes(a)
	identityB, payloadB := TypedFindingHashes(b)

	if identityA == "" || payloadA == "" {
		t.Fatalf("expected non-empty hashes, got identity=%q payload=%q", identityA, payloadA)
	}
	if identityA != identityB {
		t.Fatalf("expected stable identity hash, got %q vs %q", identityA, identityB)
	}
	if payloadA != payloadB {
		t.Fatalf("expected stable payload hash, got %q vs %q", payloadA, payloadB)
	}
}

func TestBuildTypedFindingDescriptionIncludesCanonicalFields(t *testing.T) {
	finding := TypedFinding{
		Source:       FindingSourceQA,
		FeatureID:    "F054",
		WSID:         "00-054-03",
		Blocking:     true,
		Summary:      "UAT failed on acceptance step 3",
		Description:  "Observed behavior does not match the feature intent.",
		Severity:     "P1",
		Priority:     1,
		PRURL:        "https://github.com/org/repo/pull/123",
		ArtifactRef:  ".sdp/evidence/f054-uat.json",
		EvidenceRef:  "docs/reviews/F054-UAT.md",
		TraceRef:     ".sdp/runs/f054.json",
		DriftVerdict: "accepted_drift",
	}

	description := buildTypedFindingDescription(finding, "abc123", "def456")

	checks := []string{
		"**Source:** qa",
		"**Feature:** F054",
		"**Workstream:** 00-054-03",
		"**Blocking:** true",
		"**Summary:** UAT failed on acceptance step 3",
		"- PR: https://github.com/org/repo/pull/123",
		"- Artifact: .sdp/evidence/f054-uat.json",
		"- Evidence: docs/reviews/F054-UAT.md",
		"- Trace: .sdp/runs/f054.json",
		"- Drift: accepted_drift",
		"**Finding Hash:** `abc123`",
	}

	for _, check := range checks {
		if !strings.Contains(description, check) {
			t.Fatalf("expected description to contain %q, got:\n%s", check, description)
		}
	}
}

func TestBuildLabelsForTypedFinding(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, []string{"autonomy", "review-finding"})
	labels := sink.buildLabels(FindingSourceReview, "P1", "docs", "F054", "00-054-03", true, "hash1", "hash2")

	expected := []string{
		"sdp-finding",
		"review-finding",
		"p1",
		"blocking",
		"docs",
		"F054",
		"00-054-03",
		findingHashLabel("hash1"),
		payloadHashLabel("hash2"),
		"autonomy",
	}

	for _, want := range expected {
		found := false
		for _, got := range labels {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected label %q in %v", want, labels)
		}
	}

	count := 0
	for _, label := range labels {
		if label == "review-finding" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected deduplicated review-finding label, got %v", labels)
	}
}

func TestCreateReviewFindingUsesReviewSourceAndRoleSummary(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	_, err := sink.CreateReviewFinding(ctx, ReviewFindingInput{
		FeatureID:   "F054",
		WSID:        "00-054-03",
		Blocking:    true,
		Role:        "security",
		Title:       "unsafe shell execution",
		Description: "Reviewer found unsafe shell execution in hooks.",
		Severity:    "P1",
		Priority:    1,
		PRURL:       "https://github.com/org/repo/pull/123",
	})
	if err != nil {
		t.Fatalf("CreateReviewFinding returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Created != 1 {
		t.Fatalf("expected one dry-run creation, got %+v", stats)
	}

	identityA, payloadA := TypedFindingHashes(TypedFinding{
		Source:      FindingSourceReview,
		FeatureID:   "F054",
		WSID:        "00-054-03",
		Blocking:    true,
		Title:       "unsafe shell execution",
		Summary:     "security review finding: unsafe shell execution",
		Description: "Reviewer role: security\nReviewer found unsafe shell execution in hooks.",
		Severity:    "P1",
		Priority:    1,
		PRURL:       "https://github.com/org/repo/pull/123",
		DedupKey:    "security:unsafe shell execution",
	})

	decision := sink.dedupe.Decide(identityA, payloadA)
	if decision.Action != DedupeSkip {
		t.Fatalf("expected created review finding to be tracked in dedupe store, got %s", decision.Action)
	}
}

func TestCreateQAFindingUsesScenarioAndStep(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	_, err := sink.CreateQAFinding(ctx, QAFindingInput{
		FeatureID:       "F054",
		WSID:            "00-054-03",
		Blocking:        true,
		Scenario:        "user submits review form",
		FailedStep:      "submit",
		Title:           "validation message missing",
		Description:     "Form submits successfully instead of rejecting invalid input.",
		Severity:        "P1",
		Priority:        1,
		ExpectedOutcome: "submission is blocked with validation message",
		ActualOutcome:   "submission succeeds",
	})
	if err != nil {
		t.Fatalf("CreateQAFinding returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Created != 1 {
		t.Fatalf("expected one dry-run creation, got %+v", stats)
	}

	identityA, payloadA := TypedFindingHashes(TypedFinding{
		Source:      FindingSourceQA,
		FeatureID:   "F054",
		WSID:        "00-054-03",
		Blocking:    true,
		Title:       "validation message missing",
		Summary:     "user submits review form: failed at submit: validation message missing",
		Description: "Form submits successfully instead of rejecting invalid input.\nScenario: user submits review form\nFailed step: submit\nExpected: submission is blocked with validation message\nActual: submission succeeds",
		Severity:    "P1",
		Priority:    1,
		DedupKey:    "user submits review form:validation message missing:submit",
	})

	decision := sink.dedupe.Decide(identityA, payloadA)
	if decision.Action != DedupeSkip {
		t.Fatalf("expected created QA finding to be tracked in dedupe store, got %s", decision.Action)
	}
}
