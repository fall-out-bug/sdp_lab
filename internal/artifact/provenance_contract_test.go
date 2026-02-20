package artifact

import "testing"

func TestCanonicalJSONStableAcrossMapOrder(t *testing.T) {
	left := map[string]any{"b": "two", "a": []any{3.0, map[string]any{"z": true, "x": "ok"}}}
	right := map[string]any{"a": []any{3.0, map[string]any{"x": "ok", "z": true}}, "b": "two"}

	leftDigest, err := DigestHex(left)
	if err != nil {
		t.Fatalf("digest left: %v", err)
	}
	rightDigest, err := DigestHex(right)
	if err != nil {
		t.Fatalf("digest right: %v", err)
	}
	if leftDigest != rightDigest {
		t.Fatalf("expected deterministic digest, got %s != %s", leftDigest, rightDigest)
	}
}

func TestBuildProvenanceRecordDeterministic(t *testing.T) {
	in := ProvenanceInput{
		IssueID:       "sdp_dev-2aq.16.2",
		ArtifactID:    "artifact-001",
		ArtifactClass: "execution-plan",
		Phase:         "plan",
		Role:          "planner",
		CapturedAt:    "2026-02-20T20:00:00Z",
		Sequence:      0,
	}
	payload := map[string]any{"ordering": []string{"intake", "plan", "execute"}}

	one, err := BuildProvenanceRecord(in, payload)
	if err != nil {
		t.Fatalf("build record one: %v", err)
	}
	two, err := BuildProvenanceRecord(in, payload)
	if err != nil {
		t.Fatalf("build record two: %v", err)
	}
	if one.Hash != two.Hash {
		t.Fatalf("expected deterministic hash, got %s != %s", one.Hash, two.Hash)
	}
	if err := ValidateAppend(nil, one); err != nil {
		t.Fatalf("validate genesis: %v", err)
	}
}

func TestValidateAppendRequiresMonotonicSequenceAndHashLink(t *testing.T) {
	first, err := BuildProvenanceRecord(ProvenanceInput{
		IssueID:       "sdp_dev-2aq.16.2",
		ArtifactID:    "artifact-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T20:00:00Z",
		Sequence:      0,
	}, map[string]any{"trigger": "agent"})
	if err != nil {
		t.Fatalf("build first: %v", err)
	}

	second, err := BuildProvenanceRecord(ProvenanceInput{
		IssueID:       "sdp_dev-2aq.16.2",
		ArtifactID:    "artifact-002",
		ArtifactClass: "execution-plan",
		Phase:         "plan",
		Role:          "planner",
		CapturedAt:    "2026-02-20T20:01:00Z",
		Sequence:      1,
		HashPrev:      first.Hash,
	}, map[string]any{"depends_on": []string{"sdp_dev-2aq.16.1"}})
	if err != nil {
		t.Fatalf("build second: %v", err)
	}
	if err := ValidateAppend(&first, second); err != nil {
		t.Fatalf("validate append: %v", err)
	}

	broken := second
	broken.Sequence = 3
	if err := ValidateAppend(&first, broken); err == nil {
		t.Fatal("expected sequence validation error")
	}

	tampered := second
	tampered.HashPrev = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tampered.Hash = second.Hash
	if err := ValidateAppend(&first, tampered); err == nil {
		t.Fatal("expected hash link validation error")
	}
}
