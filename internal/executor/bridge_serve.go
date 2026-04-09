package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/control"
	"sdp_dev/internal/deploy"
	"sdp_dev/internal/kernel"
	"sdp_dev/internal/executor/omoclient"
	)

// ServeBridge connects SDP dispatch to OmO via opencode serve (REST+SSE).
// This replaces ExecutorBridge which used exec.CommandContext.
type ServeBridge struct {
	Store         *control.Store
	ProjectRoot   string
	OmOServeURL   string // base URL for opencode serve (default "http://127.0.0.1:4096")
	MaxConcurrent int    // max concurrent dispatches (default 3)
	Governance    *omoclient.GovernanceConfig
	Evaluator     EvaluatorConfig
	Clarifier     ClarifierConfig
	Planner       PlannerConfig
}

// NewServeBridge creates a new serve-mode bridge.
func NewServeBridge(store *control.Store, projectRoot string) *ServeBridge {
	return &ServeBridge{
		Store:         store,
		ProjectRoot:   projectRoot,
		OmOServeURL:   os.Getenv("OMO_SERVE_URL"),
		MaxConcurrent: 3,
		Governance:    omoclient.DefaultGovernanceConfig(),
		Evaluator:     DefaultEvaluatorConfig(),
		Clarifier:     DefaultClarifierConfig(),
		Planner:       DefaultPlannerConfig(),
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

	// Pick best ready item using ranking policy
	cardID := RankAndPick(ready, DefaultRankingPolicy(), nil)
	if cardID == "" {
		return "", nil // all candidates exhausted
	}
	return cardID, nil
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

	// Create governance wrapper
	var govWrapper *omoclient.GovernanceWrapper
	if b.serveURL() != "" {
		logger := log.New(log.Writer(), "[serve-bridge] ", log.LstdFlags)
		client := omoclient.NewClient(b.serveURL(), logger)
		govWrapper = omoclient.NewGovernanceWrapper(client, omoclient.DefaultStrikePolicy(), true)
	}

	// Pre-call gate
	if govWrapper != nil {
		if err := govWrapper.PreCall(ctx, envelope); err != nil {
			return nil, fmt.Errorf("governance pre-call: %w", err)
		}
	}

	// Set executor state in Beads
	beadsRepo := b.Store.BeadsRepo()
	if beadsRepo != nil {
		now := time.Now().UTC()
		sessionID := fmt.Sprintf("omo-%d", now.UnixNano())
		if err := beadsRepo.SetExecutorState(cardID, "omo-implementation", sessionID, "running"); err != nil {
				log.Printf("warn: set executor state: %v", err)
			}
	}

	phase := card.TaskType
	if strings.TrimSpace(phase) == "" {
		phase = "build"
	}
	agent := ResolveAgent(phase)

	runtimeResult, invokeErr := InvokeWithFallback(ctx, kernel.RuntimeInvocation{
		WorkDir: b.ProjectRoot,
		Agent:   agent,
		Prompt:  governedPrompt,
	})

	// Build result
	result := translateResult(packet, runtimeResult.Output, runtimeResult.ExitCode)

	// Evidence capture
	if beadsRepo != nil {
		evidencePath := filepath.Join(b.ProjectRoot, ".sdp", "artifacts", cardID, "build.json")
		evidenceJSON, _ := json.MarshalIndent(map[string]any{
			"phase":         "build",
			"card_id":       cardID,
			"timestamp":     time.Now().UTC().Format(time.RFC3339),
			"executor":      agent,
			"exit_code":     runtimeResult.ExitCode,
			"status":        result.Status,
			"summary":       result.Summary,
			"files_changed": extractArtifactReferences(result.Artifacts),
			"artifacts":     result.Artifacts,
			"findings":      result.Findings,
		}, "", "  ")
		if err := os.MkdirAll(filepath.Dir(evidencePath), 0o755); err != nil {
				log.Printf("warn: create evidence dir: %v", err)
			}
		_ = os.WriteFile(evidencePath, evidenceJSON, 0o644)

		// Link evidence in Beads metadata
		if err := beadsRepo.LinkEvidence(cardID, "build", []string{evidencePath}); err != nil {
				log.Printf("warn: link evidence: %v", err)
			}
	}

	// Route findings
	if invokeErr == nil {
		if err := RouteFindingsToCard(b.Store, projectID, cardID, result); err != nil {
			return nil, fmt.Errorf("route findings: %w", err)
		}
	}

	// Update card state
	card, loadErr := b.Store.LoadCard(projectID, cardID)
		if loadErr != nil {
			return nil, fmt.Errorf("reload card after execution: %w", loadErr)
		}
	completedAt := time.Now().UTC()
	card.LastExecutorHeartbeatAt = completedAt.Format(time.RFC3339)
	card.ExecutorProgressSummary = result.Summary
	card.ExecutorResult = summarizeResult(result, completedAt)
	if result.Status == control.ResultStatusSuccess {
		card.ExecutorRuntimeState = control.ExecutorRuntimeCompleted
	} else {
		card.ExecutorRuntimeState = "failed"
	}
	if err := b.Store.SaveCard(card); err != nil {
			log.Printf("warn: save card after execution: %v", err)
		}

	// Set final executor state in Beads
	if beadsRepo != nil {
		state := "completed"
		if result.Status != control.ResultStatusSuccess {
			state = "failed"
		}
		if err := beadsRepo.SetExecutorState(cardID, "omo-implementation", "", state); err != nil {
				log.Printf("warn: set final executor state: %v", err)
			}
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
	phase := card.TaskType
	if phase == "" {
		phase = "build"
	}
	return omoclient.TaskEnvelope{
		TaskID:      card.ID,
		Phase:       phase,
		EntryAgent:  ResolveAgent(phase),
		Objective:   card.NormalizedIntent,
		ScopeIn:     card.ScopeIn,
		ScopeOut:    card.ScopeOut,
		Constraints: card.LinkedArtifacts,
		Governance:  b.Governance,
		Provenance:  map[string]any{},
	}
}

// Set via OMO_DEPLOY_ROOT env var, defaults to ServeBridge.ProjectRoot.
func (b *ServeBridge) deployProjectRoot() string {
	if root := os.Getenv("OMO_DEPLOY_ROOT"); root != "" {
		return root
	}
	return b.ProjectRoot
}

// TryDeployPhase checks if a card is ready for deploy and initiates staging.
// Deploy is a lifecycle phase, not a separate command.
func (b *ServeBridge) Evaluate(ctx context.Context, cardID string) (EvalResult, error) {
	card, err := b.Store.LoadCardByID(cardID)
	if err != nil {
		return EvalResult{}, fmt.Errorf("load card for evaluation: %w", err)
	}
	result, err := EvaluateBuild(ctx, b.ProjectRoot, card, b.Evaluator)
	if err != nil {
		return EvalResult{}, err
	}
	if result.Verdict == evalVerdictBlocked {
		return result, nil
	}
	if path, saveErr := saveEvaluationEvidence(b.ProjectRoot, cardID, result); saveErr == nil {
		if beadsRepo := b.Store.BeadsRepo(); beadsRepo != nil {
			_ = beadsRepo.LinkEvidence(cardID, "evaluation", []string{path})
		}
	}
	return result, nil
}

func (b *ServeBridge) RecordEvalFindings(cardID string, result EvalResult) error {
	card, err := b.Store.LoadCardByID(cardID)
	if err != nil {
		return fmt.Errorf("load card for evaluation findings: %w", err)
	}
	if len(result.Findings) > 0 {
		card.AdminActionRequired = appendUnique(card.AdminActionRequired, result.Findings...)
		card.BlockingReasons = appendUnique(card.BlockingReasons, result.Findings...)
	}
	card.RecommendedNextAction = "retry_dispatch"
	card.RecommendedNextReason = fmt.Sprintf("evaluation verdict=%s score=%.2f", result.Verdict, result.Score)
	card.ExecutorProgressSummary = card.RecommendedNextReason
	if err := b.Store.SaveCard(card); err != nil {
		return fmt.Errorf("save evaluation findings: %w", err)
	}
	return nil
}

func (b *ServeBridge) Summarize(ctx context.Context, cardID string) (SummaryResult, error) {
	if b == nil {
		return SummaryResult{}, fmt.Errorf("nil serve bridge")
	}
	summary, err := SummarizeCard(ctx, b.ProjectRoot, cardID)
	if err != nil {
		return SummaryResult{}, err
	}
	if err := saveSummary(b.ProjectRoot, cardID, summary); err != nil {
		return SummaryResult{}, err
	}
	return summary, nil
}

func extractArtifactReferences(artifacts []control.ExecutorArtifact) []string {
	refs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Reference) != "" {
			refs = append(refs, strings.TrimSpace(artifact.Reference))
		}
	}
	return refs
}

func (b *ServeBridge) TryDeployPhase(ctx context.Context, cardID, projectRoot string) error {
	beadsRepo := b.Store.BeadsRepo()
	if beadsRepo == nil {
		return fmt.Errorf("deploy requires beads mode")
	}

	// Check if card has deploy-staging label (set by review/QA phase)
	card, err := b.Store.LoadCard("", cardID)
	if err != nil {
		return fmt.Errorf("load card for deploy check: %w", err)
	}

	// Only auto-deploy cards with deploy-ready state
	if card.ExecutorRuntimeState != "deploy-ready" {
		return nil // not ready for deploy, skip silently
	}

	// Check deploy gates
	ciGate, humanGate, err := b.Store.DeployGate(cardID, control.DeployPhaseStaging)
	if err != nil {
		return fmt.Errorf("create deploy gates: %w", err)
	}

	// Execute staging deploy
	deployCfg := deploy.DefaultConfig(b.deployProjectRoot())
	deployResult, err := deploy.Staging(ctx, deployCfg, "latest")
	if err != nil {
		// Mark deploy failed, don't block the loop
		_ = beadsRepo.SetExecutorState(cardID, "deploy-staging", "", "failed")
		_ = b.Store.RecordDeployEvidence(cardID, control.DeployPhaseStaging, map[string]any{
			"error": err.Error(),
			"phase": "staging",
		})
		return fmt.Errorf("staging deploy: %w", err)
	}

	// Record deploy evidence
	_ = b.Store.RecordDeployEvidence(cardID, control.DeployPhaseStaging, map[string]any{
		"image_tag":  deployResult.ImageTag,
		"duration":   deployResult.Duration,
		"smoke_test": deployResult.SmokeTest,
		"containers": deployResult.Containers,
		"ci_gate":    ciGate,
		"human_gate": humanGate,
	})

	_ = beadsRepo.SetExecutorState(cardID, "deploy-staging", "", "staging-complete")
	_ = beadsRepo.LinkEvidence(cardID, "deploy-staging", []string{
		fmt.Sprintf("ci_gate=%s", ciGate),
		fmt.Sprintf("human_gate=%s", humanGate),
	})

	return nil
}
