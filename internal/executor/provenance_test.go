package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/control"
)

func TestRecordDispatchProvenanceWritesFile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".sdp", "control", "projects", "openclaw", "dispatches"), 0o755); err != nil {
		t.Fatal(err)
	}

	card := &control.FeatureCard{ID: "FC-001", ProjectID: "openclaw", DispatchedPacketPath: filepath.Join(projectRoot, ".sdp", "control", "projects", "openclaw", "dispatches", "FC-001.json")}
	packet := &control.ExecutionPacket{
		ParentFeatureID:   card.ID,
		ProjectID:         card.ProjectID,
		ExecutorRole:      string(control.ExecutorRoleOmOImplementation),
		Objective:         "Implement provenance",
		ScopeIn:           []string{"write bridge provenance"},
		RequiredArtifacts: []string{"dispatch provenance file"},
	}

	contractBytes, err := json.Marshal(map[string]any{"id": card.ID, "objective": "Implement provenance"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "contracts", card.ID+".json"), contractBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	packetBytes, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(card.DispatchedPacketPath, packetBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	prompt := "Objective:\nImplement provenance"
	if err := RecordDispatchProvenance(projectRoot, card, packet, prompt); err != nil {
		t.Fatalf("RecordDispatchProvenance: %v", err)
	}

	path := filepath.Join(projectRoot, ".sdp", "dispatch-provenance-"+card.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read provenance file: %v", err)
	}

	var got DispatchProvenance
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal provenance: %v", err)
	}

	if got.CardID != card.ID {
		t.Fatalf("card_id = %q, want %q", got.CardID, card.ID)
	}
	if got.Timestamp == "" || got.ContractHash == "" || got.PacketHash == "" || got.PromptHash == "" {
		t.Fatalf("expected non-empty hashes/timestamp: %+v", got)
	}
	if len(got.ContextSources) != 3 {
		t.Fatalf("context_sources len = %d, want 3", len(got.ContextSources))
	}
	if got.ContextSources[0].Type != "contract" || got.ContextSources[0].Path != "contracts/FC-001.json" || got.ContextSources[0].Hash == "" {
		t.Fatalf("unexpected contract source: %+v", got.ContextSources[0])
	}
	if got.ContextSources[1].Type != "execution_packet" || got.ContextSources[1].Hash == "" {
		t.Fatalf("unexpected execution packet source: %+v", got.ContextSources[1])
	}
	if got.ContextSources[2].Type != "feature_card" || got.ContextSources[2].Hash == "" {
		t.Fatalf("unexpected feature card source: %+v", got.ContextSources[2])
	}
}

func TestRecordDispatchProvenanceMissingContract(t *testing.T) {
	projectRoot := t.TempDir()
	card := &control.FeatureCard{ID: "FC-002", ProjectID: "openclaw"}
	packet := &control.ExecutionPacket{
		ParentFeatureID: card.ID,
		ProjectID:       card.ProjectID,
		Objective:       "Implement provenance without contract file",
	}

	if err := RecordDispatchProvenance(projectRoot, card, packet, "prompt"); err != nil {
		t.Fatalf("RecordDispatchProvenance: %v", err)
	}

	path := filepath.Join(projectRoot, ".sdp", "dispatch-provenance-"+card.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read provenance file: %v", err)
	}

	var got DispatchProvenance
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal provenance: %v", err)
	}
	if got.ContractHash != "" {
		t.Fatalf("contract_hash = %q, want empty", got.ContractHash)
	}
	if len(got.ContextSources) == 0 || got.ContextSources[0].Type != "contract" || got.ContextSources[0].Hash != "" {
		t.Fatalf("unexpected contract context source: %+v", got.ContextSources)
	}
	if got.PacketHash == "" || got.PromptHash == "" {
		t.Fatalf("expected packet/prompt hashes: %+v", got)
	}
}
