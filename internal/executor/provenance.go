package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/control"
	"sdp_dev/internal/orchestrate"
)

// DispatchProvenance records the provenance for a dispatch bridge execution.
type DispatchProvenance struct {
	CardID         string                      `json:"card_id"`
	Timestamp      string                      `json:"timestamp"`
	ContractHash   string                      `json:"contract_hash"`
	PacketHash     string                      `json:"packet_hash"`
	PromptHash     string                      `json:"prompt_hash"`
	ContextSources []orchestrate.ContextSource `json:"context_sources"`
}

// RecordDispatchProvenance records the provenance for a dispatch bridge execution.
// It chains: contract hash → execution packet hash → prompt hash → context sources.
func RecordDispatchProvenance(projectRoot string, card *control.FeatureCard, packet *control.ExecutionPacket, prompt string) error {
	if card == nil {
		return fmt.Errorf("nil feature card")
	}
	if packet == nil {
		return fmt.Errorf("nil execution packet")
	}

	contractRel := filepath.Join("contracts", card.ID+".json")
	contractPath := filepath.Join(projectRoot, contractRel)
	contractHash, err := hashFileIfExists(contractPath)
	if err != nil {
		return fmt.Errorf("hash contract: %w", err)
	}

	packetBytes, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("marshal execution packet: %w", err)
	}
	packetHash := digestBytes(packetBytes)

	cardBytes, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal feature card: %w", err)
	}
	cardHash := digestBytes(cardBytes)

	promptHash := orchestrate.ComputePromptHash(prompt)
	packetPath := strings.TrimSpace(card.DispatchedPacketPath)
	if packetPath == "" {
		packetPath = filepath.Join(projectRoot, ".sdp", "control", "projects", card.ProjectID, "dispatches", card.ID+".json")
	}

	sources := []orchestrate.ContextSource{
		{Type: "contract", Path: relativePath(projectRoot, contractPath), Hash: contractHash},
		{Type: "execution_packet", Path: relativePath(projectRoot, packetPath), Hash: packetHash},
		{Type: "feature_card", Hash: cardHash},
	}

	body := DispatchProvenance{
		CardID:         card.ID,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		ContractHash:   contractHash,
		PacketHash:     packetHash,
		PromptHash:     promptHash,
		ContextSources: sources,
	}

	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dispatch provenance: %w", err)
	}

	sdpDir := filepath.Join(projectRoot, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		return fmt.Errorf("mkdir .sdp: %w", err)
	}
	path := filepath.Join(sdpDir, fmt.Sprintf("dispatch-provenance-%s.json", card.ID))
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write dispatch provenance: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename dispatch provenance: %w", err)
	}
	return nil
}

func hashFileIfExists(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return digestBytes(b), nil
}

func digestBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func relativePath(root, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
