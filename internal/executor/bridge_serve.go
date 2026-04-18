// Package executor provides the ServeBridge for dispatching agent work to an
// external harness (opencode serve, Claude Code, Cursor).
//
// ServeBridge is NOT the same as agentloop.Harness — they serve different roles:
//
//   - ServeBridge: SDP delegates implementation to an EXTERNAL agent (opencode/Claude/Cursor).
//     The external harness runs the code; SDP supervises and collects evidence.
//
//   - agentloop.Harness: SDP itself IS the agent, running its own internal LLM loop.
//     Used for SDP's autonomous phases (discovery analysis, planning, review).
//
// Both components are in production after F108. They are not duplicates.
package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/agentloop/livegw"
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

	// Harness fields: when harnessRouter is non-nil, DispatchAndRun uses the
	// internal agentloop.Harness path instead of the legacy OmO path.
	harnessRouter *agentloop.PhaseRouter
	harnessGate   *agentloop.GateEngine
	harnessData   string // root dir for per-card SQLite databases
}

// NewServeBridge creates a new serve-mode bridge.
//
// The internal agentloop harness path requires explicit opt-in via the
// SDP_USE_HARNESS environment variable (set to "1"). When both
// SDP_USE_HARNESS and OPENROUTER_API_KEY are present, the harness path is
// activated. Otherwise, harnessRouter stays nil and DispatchAndRun falls back
// to the legacy OmO path.
func NewServeBridge(store *control.Store, projectRoot string) *ServeBridge {
	sb := &ServeBridge{
		Store:         store,
		ProjectRoot:   projectRoot,
		OmOServeURL:   os.Getenv("OMO_SERVE_URL"),
		MaxConcurrent: 3,
		Governance:    omoclient.DefaultGovernanceConfig(),
		Evaluator:     DefaultEvaluatorConfig(),
		Clarifier:     DefaultClarifierConfig(),
		Planner:       DefaultPlannerConfig(),
		harnessData:   filepath.Join(projectRoot, ".sdp", "sessions"),
	}

	// Harness path requires explicit opt-in via SDP_USE_HARNESS=1.
	if os.Getenv("SDP_USE_HARNESS") != "1" {
		return sb
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		slog.Warn("SDP_USE_HARNESS set but OPENROUTER_API_KEY missing — harness path disabled")
		return sb
	}

	gw, err := livegw.New(apiKey, os.Getenv("OPENROUTER_BASE_URL"))
	if err != nil {
		slog.Warn("livegw init failed — harness path disabled", "error", err)
		return sb
	}
	tools := agentloop.BuildLiveTools(projectRoot, store)
	registry := agentloop.NewToolRegistry(tools)
	sb.harnessRouter = agentloop.NewPhaseRouter(
		agentloop.DefaultPhaseMap, registry, gw, nil,
	)
	sb.harnessGate = agentloop.NewGateEngine(nil, 0) // default timeout, nil contract for MVP

	return sb
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

	// Pick best ready item using ranking policy. Skip cards that are in an
	// active harness wait-state (awaiting_human/awaiting_input) to prevent
	// redispatch loops — these cards are already dispatched and pending.
	for range ready {
		cardID := RankAndPick(ready, DefaultRankingPolicy(), nil)
		if cardID == "" {
			return "", nil // all candidates exhausted
		}
		card, loadErr := b.Store.LoadCardByID(cardID)
		if loadErr != nil {
			return cardID, nil // can't check state — let dispatch proceed
		}
		switch card.ExecutorRuntimeState {
		case "awaiting_human", "awaiting_input":
			slog.Info("skipping card in harness wait-state", "card", cardID, "state", card.ExecutorRuntimeState)
			// Remove from candidates and try next
			ready = slices.DeleteFunc(ready, func(c control.FeatureCard) bool { return c.ID == cardID })
			continue
		}
		return cardID, nil
	}
	return "", nil
}

// DispatchAndRun executes a card through the internal harness (if available) or OmO serve.
func (b *ServeBridge) DispatchAndRun(ctx context.Context, projectID, cardID string) (*control.ExecutorResultPacket, error) {
	if b == nil || b.Store == nil {
		return nil, fmt.Errorf("nil serve bridge/store")
	}

	// If the harness path is available, use it as the primary execution path.
	if b.harnessRouter != nil {
		return b.runWithHarness(ctx, cardID)
	}

	// Legacy OmO path follows below.
	return b.dispatchWithOmO(ctx, projectID, cardID)
}

// recordExecutionResult writes build.json evidence, links evidence in Beads,
// routes findings, and updates card state. It is the shared bookkeeping path
// used by both the legacy OmO executor and the internal harness path.
//
// executorName identifies the source ("harness", agent name, etc.) for the
// evidence file. projectID is resolved from the card when available.
// Returns an error if mandatory bookkeeping fails (evidence write, card state).
func (b *ServeBridge) recordExecutionResult(cardID string, result *control.ExecutorResultPacket, phase, executorName string) error {
	beadsRepo := b.Store.BeadsRepo()

	// Resolve projectID from card for RouteFindings.
	var projectID string
	if card, err := b.Store.LoadCardByID(cardID); err == nil && card != nil {
		projectID = card.ProjectID
	}

	// 1. Write phase-specific evidence file.
	// Only the "build" phase writes the canonical build.json used by downstream
	// consumers (evaluator, tower). Other phases write their own evidence files
	// (discover.json, plan.json, etc.) to avoid overwriting real build output.
	evidenceName := phase + ".json"
	if phase == "" || phase == "build" {
		evidenceName = "build.json"
	}
	evidencePath := filepath.Join(b.ProjectRoot, ".sdp", "artifacts", cardID, evidenceName)
	evidenceJSON, marshalErr := json.MarshalIndent(map[string]any{
		"phase":         phase,
		"card_id":       cardID,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"executor":      executorName,
		"status":        result.Status,
		"summary":       result.Summary,
		"files_changed": extractArtifactReferences(result.Artifacts),
		"artifacts":     result.Artifacts,
		"findings":      result.Findings,
	}, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("marshal evidence: %w", marshalErr)
	}
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o755); err != nil {
		return fmt.Errorf("create evidence dir: %w", err)
	}
	if err := os.WriteFile(evidencePath, evidenceJSON, 0o644); err != nil {
		return fmt.Errorf("write evidence: %w", err)
	}

	// 2. Link evidence in Beads metadata
	if beadsRepo != nil {
		if err := beadsRepo.LinkEvidence(cardID, phase, []string{evidencePath}); err != nil {
			slog.Warn("link evidence", "card", cardID, "error", err)
		}
	}

	// 3. Route findings to card.
	// Skip routing for intermediate harness continuation (awaiting_input) to avoid
	// parking the card in the human-input lane. Findings are only routed for
	// terminal outcomes (success, failure, needs_review).
	isIntermediateContinuation := result.Status == control.ResultStatusNeedsInput && executorName == "harness"
	if !isIntermediateContinuation {
		if err := RouteFindingsToCard(b.Store, projectID, cardID, result); err != nil {
			slog.Warn("route findings", "card", cardID, "error", err)
		}
	}

	// 4. Update card state
	card, loadErr := b.Store.LoadCardByID(cardID)
	if loadErr != nil {
		return fmt.Errorf("load card for bookkeeping: %w", loadErr)
	}
	completedAt := time.Now().UTC()
	card.LastExecutorHeartbeatAt = completedAt.Format(time.RFC3339)
	card.ExecutorProgressSummary = result.Summary
	card.ExecutorResult = summarizeResult(result, completedAt)
	// Clear stale gate-specific fields from any previous NeedsReview result.
	// These are only meaningful when the card is actively awaiting human review.
	card.DecisionRequired = nil
	card.FeedbackRequest = nil
	card.NeedsFeedbackFrom = nil
	card.WaitingOn = nil

	switch result.Status {
	case control.ResultStatusSuccess:
		card.ExecutorRuntimeState = control.ExecutorRuntimeCompleted
		card.Status = "executing" // completed phase — card stays in executing until orchestrator finalizes
	case control.ResultStatusNeedsReview:
		card.ExecutorRuntimeState = "awaiting_human"
		card.Status = "needs_input"
		card.DecisionRequired = []string{"Gate approval required"}
		card.FeedbackRequest = []string{"Review gate escalation and approve or rollback"}
		card.NeedsFeedbackFrom = []string{"human"}
		card.WaitingOn = []string{"human"}
	case control.ResultStatusNeedsInput:
		card.ExecutorRuntimeState = "awaiting_input"
		card.Status = "executing" // intermediate phase — card still being processed
	default:
		card.ExecutorRuntimeState = "failed"
		card.Status = "executing" // failure recorded in state; orchestrator decides final status
	}
	if err := b.Store.SaveCard(card); err != nil {
		return fmt.Errorf("save card after execution: %w", err)
	}

	// 5. Set final executor state in Beads
	if beadsRepo != nil {
		state := "completed"
		switch result.Status {
		case control.ResultStatusNeedsReview:
			state = "awaiting_human"
		case control.ResultStatusNeedsInput:
			state = "awaiting_input"
		case control.ResultStatusFailed:
			state = "failed"
		}
		if err := beadsRepo.SetExecutorState(cardID, executorName, "", state); err != nil {
			slog.Warn("set final executor state", "card", cardID, "error", err)
		}
	}
	return nil
}

// dispatchWithOmO is the legacy OmO serve execution path (extracted from DispatchAndRun).
func (b *ServeBridge) dispatchWithOmO(ctx context.Context, projectID, cardID string) (*control.ExecutorResultPacket, error) {
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
		client := omoclient.NewClient(b.serveURL())
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
				slog.Warn("set executor state", "card", cardID, "error", err)
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

	// Shared bookkeeping: write evidence, route findings, update card state.
	// Treat bookkeeping as part of the success contract: if evidence cannot be
	// persisted, downgrade the result to failure so downstream doesn't act on
	// incomplete state.
	if bkErr := b.recordExecutionResult(cardID, result, phase, agent); bkErr != nil {
		result.Status = control.ResultStatusFailed
		result.Summary = fmt.Sprintf("bookkeeping failed: %v", bkErr)
	}

	return result, invokeErr
}

// runWithHarness executes a card through the internal agentloop.Harness.
// It creates a per-card SQLite store, restores (or creates) a Harness session,
// runs one phase turn, and returns an ExecutorResultPacket.
//
// Spec §11 bug fix: if RestoreHarness returns ErrHarnessTerminated, the old
// database is removed, a fresh store is created, and RestoreHarness is retried.
func (b *ServeBridge) runWithHarness(ctx context.Context, cardID string) (*control.ExecutorResultPacket, error) {
	// Ensure session data directory exists.
	if err := os.MkdirAll(b.harnessData, 0o755); err != nil {
		return nil, fmt.Errorf("harness mkdir: %w", err)
	}

	dbPath := filepath.Join(b.harnessData, cardID+".db")

	h, store, err := b.restoreOrCreateHarness(cardID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("harness restore: %w", err)
	}
	defer store.Close()

	// Crash reconciliation: when the agent crashes, panics, or the context
	// is cancelled, the harness must be stopped so a terminal PhaseRecord
	// is persisted in SQLite. Without this, RestoreHarness on the same cardID
	// would resume a dirty session instead of detecting termination.
	var succeeded bool
	defer func() {
		if r := recover(); r != nil {
			_ = h.Stop(context.Background(), "")
			panic(r) // re-panic after recording terminal state
		}
		if !succeeded {
			_ = h.Stop(context.Background(), "")
		}
	}()

	// Build governed prompt from card.
	card, err := b.Store.LoadCardByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("load card: %w", err)
	}

	// Capture the phase before any RunPhase call.
	phase := string(h.Phase())

	// Handle restored awaiting_human state: if the harness was restored with
	// a pending gate decision (crash recovery), skip RunPhase and return
	// the gate-pending result immediately. Without this, RunPhase would reject
	// the non-idle state and the session would fall into generic failure.
 if h.IsAwaitingHuman() {
		result := &control.ExecutorResultPacket{
			ParentFeatureID: cardID,
			Status:          control.ResultStatusNeedsReview,
			Summary:         "gate restored — awaiting human decision (crash recovery)",
		}
		if bkErr := b.recordExecutionResult(cardID, result, phase, "harness"); bkErr != nil {
			result.Status = control.ResultStatusFailed
			result.Summary = fmt.Sprintf("bookkeeping failed: %v", bkErr)
			// Bookkeeping failed — session may be inconsistent, stop it.
			_ = h.Stop(context.Background(), "")
			return result, nil
		}
		succeeded = true // bookkeeping ok — session stays alive
		return result, nil
	}

	// Build governed prompt from card — includes scope constraints, governance
	// metadata, and provenance. This mirrors the OmO path's prompt construction
	// so the harness agent also respects ScopeIn/ScopeOut and linked constraints.
	envelope := b.cardToEnvelope(card)
	promptBuilder := omoclient.NewGovernancePromptBuilder()
	governedPrompt := promptBuilder.BuildFullPrompt(envelope, card.NormalizedIntent)
	if governedPrompt == "" {
		governedPrompt = card.RawRequest
	}

	// Record provenance for harness path (same as OmO).
	packet, _ := b.buildPacket(card)
	if packet != nil {
		_ = RecordDispatchProvenance(b.ProjectRoot, card, packet, governedPrompt)
	}

	// Execute one phase turn.
	runErr := h.RunPhase(ctx, governedPrompt, "")

	// Check for gate-pending (awaiting human decision).
	if runErr == nil && h.IsAwaitingHuman() {
		result := &control.ExecutorResultPacket{
			ParentFeatureID: cardID,
			Status:          control.ResultStatusNeedsReview,
			Summary:         "gate escalated — awaiting human decision",
		}
		if err := b.recordExecutionResult(cardID, result, phase, "harness"); err != nil {
			result.Status = control.ResultStatusFailed
			result.Summary = fmt.Sprintf("bookkeeping failed: %v", err)
			_ = h.Stop(context.Background(), "")
			return result, nil
		}
		succeeded = true // bookkeeping ok — session stays alive
		return result, nil
	}

	if runErr != nil {
		// succeeded stays false → defer will call h.Stop
		result := &control.ExecutorResultPacket{
			ParentFeatureID: cardID,
			Status:          control.ResultStatusFailed,
			Summary:         runErr.Error(),
		}
		if err := b.recordExecutionResult(cardID, result, phase, "harness"); err != nil {
			slog.Error("bookkeeping failed on error path", "card", cardID, "error", err)
		}
		return result, runErr
	}

	// Distinguish between "agent finished the phase" and "agent responded
	// but didn't call completion_signal". Without this check, a plain text
	// response (no tool calls) would be incorrectly mapped to Success,
	// causing the downstream evaluator to hard-block on missing build.json.
	if !h.LastPhaseCompleted() {
		result := &control.ExecutorResultPacket{
			ParentFeatureID: cardID,
			Status:          control.ResultStatusNeedsInput,
			Summary:         "phase turn completed — awaiting next prompt",
		}
		if err := b.recordExecutionResult(cardID, result, phase, "harness"); err != nil {
			result.Status = control.ResultStatusFailed
			result.Summary = fmt.Sprintf("bookkeeping failed: %v", err)
			_ = h.Stop(context.Background(), "")
			return result, nil
		}
		succeeded = true // bookkeeping ok — session stays alive
		return result, nil
	}

	// Only return terminal Success after the final phase (RoleEval).
	// Intermediate phase completions (discover→plan→build→review) are
	// non-terminal: the harness session stays alive for the next phase,
	// and the orchestration loop should NOT trigger evaluation/deploy.
	if phase != string(agentloop.RoleEval) {
		result := &control.ExecutorResultPacket{
			ParentFeatureID: cardID,
			Status:          control.ResultStatusNeedsInput,
			Summary:         "intermediate phase completed — session continues",
		}
		if err := b.recordExecutionResult(cardID, result, phase, "harness"); err != nil {
			result.Status = control.ResultStatusFailed
			result.Summary = fmt.Sprintf("bookkeeping failed: %v", err)
			_ = h.Stop(context.Background(), "")
			return result, nil
		}
		succeeded = true // bookkeeping ok — session stays alive
		return result, nil
	}

	result := &control.ExecutorResultPacket{
		ParentFeatureID: cardID,
		Status:          control.ResultStatusSuccess,
		Summary:         "lifecycle completed — all phases done",
	}
	if err := b.recordExecutionResult(cardID, result, phase, "harness"); err != nil {
		result.Status = control.ResultStatusFailed
		result.Summary = fmt.Sprintf("bookkeeping failed: %v", err)
		_ = h.Stop(context.Background(), "")
		return result, nil
	}
	succeeded = true // terminal success — defer will skip Stop
	return result, nil
}

// restoreOrCreateHarness restores a Harness from the given dbPath, handling
// ErrHarnessTerminated by removing the old DB and retrying (spec §11).
// If the session does not exist yet (fresh card), a new session is created first.
func (b *ServeBridge) restoreOrCreateHarness(cardID, dbPath string) (*agentloop.Harness, *agentloop.SQLiteStore, error) {
	store, err := agentloop.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite open: %w", err)
	}

	h, err := agentloop.RestoreHarness(cardID, "", store, b.harnessRouter, b.harnessGate, nil)
	if err == nil {
		return h, store, nil
	}

	// Spec §11: ErrHarnessTerminated means the session was Stop()'d.
	// Remove the old DB, create a fresh store, and retry.
	if errors.Is(err, agentloop.ErrHarnessTerminated) {
		store.Close()
		return b.recreateFromScratch(cardID, dbPath)
	}

	// Session not found — this is a fresh card. Create a new session and retry.
	store.Close()
	if isSessionNotFound(err) {
		return b.createFreshHarness(cardID, dbPath)
	}

	return nil, nil, err
}

// createFreshHarness creates a new session in a fresh SQLite store and returns a Harness.
func (b *ServeBridge) createFreshHarness(cardID, dbPath string) (*agentloop.Harness, *agentloop.SQLiteStore, error) {
	store, err := agentloop.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite open (fresh): %w", err)
	}
	if _, err := agentloop.NewSession(cardID, store); err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("new session: %w", err)
	}
	h, err := agentloop.RestoreHarness(cardID, "", store, b.harnessRouter, b.harnessGate, nil)
	if err != nil {
		store.Close()
		return nil, nil, err
	}
	return h, store, nil
}

// recreateFromScratch handles spec §11: removes the terminated DB and creates fresh.
func (b *ServeBridge) recreateFromScratch(cardID, dbPath string) (*agentloop.Harness, *agentloop.SQLiteStore, error) {
	if removeErr := os.Remove(dbPath); removeErr != nil && !os.IsNotExist(removeErr) {
		return nil, nil, fmt.Errorf("remove terminated db: %w", removeErr)
	}
	return b.createFreshHarness(cardID, dbPath)
}

// isSessionNotFound checks if the error indicates a missing session.
//
// TODO(agentloop): Add exported sentinel error (e.g., ErrSessionNotFound) to
// agentloop/store so clients can use errors.Is() instead of string matching.
// The underlying store.Recover() returns fmt.Errorf("session %q not found", ...)
// which is fragile for error checking.
func isSessionNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "no rows"))
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
