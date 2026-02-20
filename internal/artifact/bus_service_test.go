package artifact

import "testing"

func TestBusServiceIngestAndRetrieveWithHashChain(t *testing.T) {
	bus := NewBusService()

	first, err := bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.16.3",
		ArtifactID:    "intent-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:00:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("ingest first: %v", err)
	}
	if first.Provenance.Sequence != 0 {
		t.Fatalf("expected genesis sequence 0, got %d", first.Provenance.Sequence)
	}
	if first.Provenance.HashPrev != "" {
		t.Fatalf("expected empty genesis hash_prev, got %q", first.Provenance.HashPrev)
	}

	second, err := bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.16.3",
		ArtifactID:    "plan-001",
		ArtifactClass: "execution-plan",
		Phase:         "plan",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:01:00Z",
		Payload:       map[string]any{"depends_on": []string{"sdp_dev-2aq.16.2"}},
	})
	if err != nil {
		t.Fatalf("ingest second: %v", err)
	}
	if second.Provenance.Sequence != 1 {
		t.Fatalf("expected sequence 1, got %d", second.Provenance.Sequence)
	}
	if second.Provenance.HashPrev != first.Provenance.Hash {
		t.Fatalf("expected hash_prev %s, got %s", first.Provenance.Hash, second.Provenance.HashPrev)
	}

	byID, ok := bus.GetByIssueArtifactID("sdp_dev-2aq.16.3", "plan-001")
	if !ok {
		t.Fatal("expected retrieval by issue/artifact id to succeed")
	}
	if byID.Provenance.Hash != second.Provenance.Hash {
		t.Fatalf("unexpected hash for retrieved artifact: %s", byID.Provenance.Hash)
	}

	bySeq, ok := bus.GetByIssueSequence("sdp_dev-2aq.16.3", 0)
	if !ok {
		t.Fatal("expected retrieval by sequence to succeed")
	}
	if bySeq.ArtifactID != "intent-001" {
		t.Fatalf("expected intent-001 for sequence lookup, got %s", bySeq.ArtifactID)
	}

	latest, ok := bus.LatestByIssue("sdp_dev-2aq.16.3")
	if !ok {
		t.Fatal("expected latest retrieval for issue to succeed")
	}
	if latest.ArtifactID != second.ArtifactID {
		t.Fatalf("expected latest artifact %s, got %s", second.ArtifactID, latest.ArtifactID)
	}

	byHash, ok := bus.GetByHash(first.Provenance.Hash)
	if !ok {
		t.Fatal("expected retrieval by provenance hash to succeed")
	}
	if byHash.ArtifactID != "intent-001" {
		t.Fatalf("expected intent-001 from hash lookup, got %s", byHash.ArtifactID)
	}

	index := bus.ProvenanceIndex("sdp_dev-2aq.16.3")
	if len(index) != 2 {
		t.Fatalf("expected 2 provenance index rows, got %d", len(index))
	}
	if index[0].ContractVersion != ProvenanceContractVersion || index[0].HashAlgorithm != ProvenanceHashAlgorithm {
		t.Fatalf("unexpected provenance contract metadata in index row: %+v", index[0])
	}
	if index[1].Role != "planner" {
		t.Fatalf("expected role planner in provenance index, got %s", index[1].Role)
	}
	if index[1].HashPrev != index[0].Hash {
		t.Fatalf("expected hash chain in provenance index, got %s prev %s", index[1].Hash, index[1].HashPrev)
	}

	metadata, ok := bus.ChainMetadata("sdp_dev-2aq.16.3")
	if !ok {
		t.Fatal("expected chain metadata for issue")
	}
	if metadata.RecordCount != 2 || metadata.LastSequence != 1 || metadata.HeadHash != second.Provenance.Hash || metadata.GenesisHash != first.Provenance.Hash {
		t.Fatalf("unexpected chain metadata: %+v", metadata)
	}
}

func TestBusServiceRejectsDuplicateArtifactIDWithinIssue(t *testing.T) {
	bus := NewBusService()
	_, err := bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.16.3",
		ArtifactID:    "code-001",
		ArtifactClass: "code-diff",
		Phase:         "execute",
		Role:          "executor",
		CapturedAt:    "2026-02-20T19:02:00Z",
		Payload:       map[string]any{"paths_touched": []string{"internal/artifact/bus_service.go"}},
	})
	if err != nil {
		t.Fatalf("initial ingest: %v", err)
	}

	_, err = bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.16.3",
		ArtifactID:    "code-001",
		ArtifactClass: "code-diff",
		Phase:         "execute",
		Role:          "executor",
		CapturedAt:    "2026-02-20T19:03:00Z",
		Payload:       map[string]any{"paths_touched": []string{"internal/artifact/bus_service_test.go"}},
	})
	if err == nil {
		t.Fatal("expected duplicate artifact id ingest to fail")
	}
}

func TestBusServiceStartsIndependentChainPerIssue(t *testing.T) {
	bus := NewBusService()
	left, err := bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.16.3",
		ArtifactID:    "left-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:04:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("left ingest: %v", err)
	}
	right, err := bus.Ingest(IngestRequest{
		IssueID:       "sdp_dev-2aq.99.1",
		ArtifactID:    "right-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:05:00Z",
		Payload:       map[string]any{"trigger": "manual"},
	})
	if err != nil {
		t.Fatalf("right ingest: %v", err)
	}

	if left.Provenance.Sequence != 0 || right.Provenance.Sequence != 0 {
		t.Fatalf("expected both streams to start at genesis, got %d and %d", left.Provenance.Sequence, right.Provenance.Sequence)
	}
	if right.Provenance.HashPrev != "" {
		t.Fatalf("expected independent genesis hash_prev empty, got %q", right.Provenance.HashPrev)
	}

	if _, ok := bus.GetByIssueArtifactID("missing", "artifact"); ok {
		t.Fatal("expected missing issue lookup to fail")
	}
	if _, ok := bus.GetByIssueSequence("missing", 0); ok {
		t.Fatal("expected missing issue sequence lookup to fail")
	}
	if _, ok := bus.GetByIssueSequence("sdp_dev-2aq.16.3", 4); ok {
		t.Fatal("expected out-of-range sequence lookup to fail")
	}
	if _, ok := bus.GetByHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); ok {
		t.Fatal("expected missing hash lookup to fail")
	}
	if _, ok := bus.LatestByIssue("missing"); ok {
		t.Fatal("expected missing latest lookup to fail")
	}
}
