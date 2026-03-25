package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/control"
	"sdp_dev/internal/executor/omoclient"
	"sdp_dev/internal/orchestrate"
)

// ServeBridge connects SDP dispatch to OmO via opencode serve (REST+SSE).
// This replaces ExecutorBridge which used exec.CommandContext.
type ServeBridge struct {
	Store           *control.Store
	ProjectRoot     string
	OmOServeURL     string           // base URL for opencode serve (default "http://127.0.0.1:4096")
	MaxConcurrent   int              // max concurrent dispatches (default 3)
	Governance      *omoclient.GovernanceConfig
}

// NewServeBridge creates a new serve-mode bridge.
func NewServeBridge(store *control.Store, projectRoot string) *ServeBridge {
	return &ServeBridge{
		Store:       store,
		ProjectRoot: projectRoot,
		OmOServeURL: os.Getenv("OMO_SERVE_URL"),
		MaxConcurrent: 3,
		Governance:   omoclient.DefaultGovernanceConfig(),
	}
}

// serveURL returns the configured or default opencode serve URL.
func (b *ServeBridge) serveURL() string {
	if b.OmOServeURL != "" {
		return b.OmOServeURL
	}
	return "http://127.0.0.1:4096"
}

// DispatchBeads dispatches the next ready Beads issue through OmO.
// Returns the card ID dispatched, or empty string if nothing ready.
func (b *ServeBridge) DispatchBeads(ctx context.Context) (string, error) {
	beadsRepo := b.Store.BeadsRepo()
	if beadsRepo == nil {
		return "", fmt.Errorf("DispatchBeads requires beads or dual mode (current: %s)", b.Store.RepoMode)
	}

	// Query ready queue from Beads
	ready, err := beadsRepo.QueryReady()
	if err != nil {
		return "", fmt.Errorf("query ready: %w", err)
	}
	if len(ready) == 0 {
		return "", nil // nothing to dispatch
	}

	// Pick first ready item (TODO: policy ranking)
	card := ready[0]
	return card.ID, nil
}

// DispatchAndRun executes a card through OmO serve with full governance.
func (b *ServeBridge) DispatchAndRun(ctx context.Context, projectID, cardID string) (*control.ExecutorResultPacket, error) {
	if b == nil || b.Store == nil {
		return nil, fmt.Errorf("nil serve bridge/store")
	}

	// Load card from primary repo
	card, err := b.Store.LoadCard(projectID, cardID)
	if err != nil {
		return nil, fmt.Errorf("load card: %w", err)
	}

	// Build execution packet from card
	packet, err := b.buildPacket(card)
	if err != nil {
		return nil, fmt.Errorf("build packet: %w", err)
	}

	// Build governed prompt using GovernancePromptBuilder
	envelope := b.cardToEnvelope(card)
	promptBuilder := omoclient.NewGovernancePromptBuilder()
	governedPrompt := promptBuilder.BuildFullPrompt(envelope, buildDispatchPrompt(packet))

	// Record provenance
	if err := RecordDispatchProvenance(b.ProjectRoot, card, packet, governedPrompt); err != nil {
		return nil, fmt.Errorf("record provenance: %w", err)
	}

	// Create OmO session
	logger := log.New(log.Writer(), "[serve-bridge] ", log.LstdFlags)
	client := omoclient.NewClient(b.serveURL(), logger)
	govWrapper := omoclient.NewGovernanceWrapper(client, omoclient.DefaultStrikePolicy(), true)

	// Pre-call gate
	if err := govWrapper.PreCall(ctx, envelope); err != nil {
		return nil, fmt.Errorf("governance pre-call: %w", err)
	}

	// Set executor state in Beads
	beadsRepo := b.Store.BeadsRepo()
	if beadsRepo != nil {
		now := time.Now().UTC()
		sessionID := fmt.Sprintf("omo-%d", now.UnixNano())
		_ = beadsRepo.SetExecutorState(cardID, "omo-implementation", sessionID, "running")
	}

	// Invoke via ServeInvoker (or fallback to default)
	invoker := orchestrate.DefaultLLMInvoker
	agent := mapExecutorRoleToSisyphus(packet.ExecutorRole)

	output, exitCode, invokeErr := invoker.Invoke(ctx, b.ProjectRoot, agent, governedPrompt)

	// Build result
	result := translateResult(packet, output, exitCode)

	// Evidence capture
	if beadsRepo != nil {
		evidencePath := filepath.Join(b.ProjectRoot, ".sdp", "artifacts", cardID, "build.json")
		evidenceJSON, _ := json.MarshalIndent(map[string]any{
			"phase":         "build",
			"card_id":       cardID,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
			"executor":      agent,
			"exit_code":     exitCode,
			"status":        result.Status,
			"summary":       result.Summary,
			"files_changed": result.Artifacts,
		}, "", "  ")
		_ = os.MkdirAll(filepath.Dir(evidencePath), 0o755)
		_ = os.WriteFile(evidencePath, evidenceJSON, 0o644)

		// Link evidence in Beads metadata
		_ = beadsRepo.LinkEvidence(cardID, "build", []string{evidencePath})
	}

	// Route findings
	if invokeErr == nil {
		if err := RouteFindingsToCard(b.Store, projectID, cardID, result); err != nil {
			return nil, fmt.Errorf("route findings: %w", err)
		}
	}

	// Update card state
	card, _ = b.Store.LoadCard(projectID, cardID)
	completedAt := time.Now().UTC()
	card.LastExecutorHeartbeatAt = completedAt.Format(time.RFC3339)
	card.ExecutorProgressSummary = result.Summary
	card.ExecutorResult = summarizeResult(result, completedAt)
	if result.Status == control.ResultStatusSuccess {
		card.ExecutorRuntimeState = control.ExecutorRuntimeCompleted
	} else {
		card.ExecutorRuntimeState = "failed"
	}
	_ = b.Store.SaveCard(card)

	// Set final executor state in Beads
	if beadsRepo != nil {
		state := "completed"
		if result.Status != control.ResultStatusSuccess {
			state = "failed"
		}
		_ = beadsRepo.SetExecutorState(cardID, "omo-implementation", "", state)
	}

	return result, invokeErr
}

// buildPacket creates an ExecutionPacket from a FeatureCard.
func (b *ServeBridge) buildPacket(card *control.FeatureCard) (*control.ExecutionPacket, error) {
	return &control.ExecutionPacket{
		ParentFeatureID: card.ID,
		ProjectID:       card.ProjectID,
		Objective:       card.NormalizedIntent,
		ScopeIn:         card.ScopeIn,
		ScopeOut:        card.ScopeOut,
		Constraints:     card.LinkedArtifacts, // constraints from linked artifacts
		ExecutorRole:    string(control.ExecutorRoleOmOImplementation),
	}, nil
}

// cardToEnvelope converts a FeatureCard to a TaskEnvelope for governance.
func (b *ServeBridge) cardToEnvelope(card *control.FeatureCard) omoclient.TaskEnvelope {
	return omoclient.TaskEnvelope{
		TaskID:      card.ID,
		Phase:       card.TaskType,
		EntryAgent:  "sisyphus",
		Objective:   card.NormalizedIntent,
		ScopeIn:     card.ScopeIn,
		ScopeOut:    card.ScopeOut,
		Constraints: card.LinkedArtifacts,
		Governance:  b.Governance,
		Provenance:  map[string]any{},
	}
}

// mapExecutorRoleToSisyphus always routes through sisyphus (orchestrator).
func mapExecutorRoleToSisyphus(role string) string {
	// OmO decision: always sisyphus, never SDP-local agents
	return "sisyphus"
}
