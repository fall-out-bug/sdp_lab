//go:build integration

package control

import (
	"os"
	"testing"
)

// TestR6_ContractEvidenceLinking tests contract/evidence/provenance metadata linking.
func TestR6_ContractEvidenceLinking(t *testing.T) {
	if os.Getenv("SDP_TEST_BEADS") == "" {
		t.Skip("set SDP_TEST_BEADS=1 to run")
	}

	repo := NewBeadsCardRepository("", nil)

	// Create a test card
	card := &FeatureCard{
		ProjectID:        "test",
		Title:            "[R6 TEST] Contract/evidence linking",
		NormalizedIntent: "Testing R6 metadata operations",
		ExecutionMode:    "2",
		TaskType:         "build",
	}
	if err := repo.CreateCard("test", card); err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	id := card.ID
	t.Logf("Created: %s", id)

	// Link contract
	if err := repo.LinkContract(id, "CTR-R6-001", "sha256:abc123"); err != nil {
		t.Fatalf("LinkContract: %v", err)
	}
	t.Log("✓ Linked contract")

	// Link evidence
	if err := repo.LinkEvidence(id, "build", []string{".sdp/evidence/test/build.json"}); err != nil {
		t.Fatalf("LinkEvidence: %v", err)
	}
	t.Log("✓ Linked evidence")

	// Set provenance
	if err := repo.SetProvenance(id, "sha256:pkg123", "sha256:prm123"); err != nil {
		t.Fatalf("SetProvenance: %v", err)
	}
	t.Log("✓ Set provenance")

	// Set executor state
	if err := repo.SetExecutorState(id, "omo-implementation", "ses_test123", "running"); err != nil {
		t.Fatalf("SetExecutorState: %v", err)
	}
	t.Log("✓ Set executor state")

	// Cleanup
	_ = repo.ResolveGate(id) // best effort, it's not a gate
	_ = repo.SaveCard(&FeatureCard{ID: id, Status: "closed"})
	t.Logf("Cleaned up: %s", id)
}
