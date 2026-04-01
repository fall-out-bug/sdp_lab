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
	"sdp_dev/internal/executor/omoclient"
	"sdp_dev/internal/kernel"
)

const planPendingBlockingReason = "plan_pending_approval"

type PlanResult struct {
	CardID          string   `json:"card_id"`
	Status          string   `json:"status"`
	Approach        string   `json:"approach,omitempty"`
	FilesToChange   []string `json:"files_to_change,omitempty"`
	TestsToWrite    []string `json:"tests_to_write,omitempty"`
	RiskAssessment  string   `json:"risk_assessment,omitempty"`
	EstimatedSteps  int      `json:"estimated_steps,omitempty"`
	RawPlan         string   `json:"raw_plan,omitempty"`
	ApprovalPending bool     `json:"approval_pending,omitempty"`
}

type PlannerConfig struct {
	Enabled     bool
	AutoApprove []string
	Model       string
}

type llmPlanPayload struct {
	Approach        string   `json:"approach"`
	FilesToChange   []string `json:"files_to_change"`
	TestsToWrite    []string `json:"tests_to_write"`
	RiskAssessment  string   `json:"risk_assessment"`
	EstimatedSteps  int      `json:"estimated_steps"`
}

func DefaultPlannerConfig() PlannerConfig {
	model := strings.TrimSpace(os.Getenv("SDP_PLAN_MODEL"))
	if model == "" {
		model = "default"
	}
	return PlannerConfig{
		Enabled:     true,
		AutoApprove: []string{"low"},
		Model:       model,
	}
}

func isPlanApproved(card *control.FeatureCard) bool {
	if card == nil {
		return false
	}
	if strings.TrimSpace(card.ExecutorRuntimeState) == "plan-approved" {
		return true
	}
	return !hasBlockingReason(card, planPendingBlockingReason)
}

func isAutoApproveRisk(riskLevel string, autoApprove []string) bool {
	riskLevel = strings.ToLower(strings.TrimSpace(riskLevel))
	for _, allowed := range autoApprove {
		if strings.ToLower(strings.TrimSpace(allowed)) == riskLevel {
			return true
		}
	}
	return false
}

func GeneratePlan(ctx context.Context, projectRoot string, card *control.FeatureCard, cfg PlannerConfig) (PlanResult, error) {
	if card == nil {
		return PlanResult{Status: "error"}, fmt.Errorf("nil card")
	}

	if isPlanApproved(card) {
		planPath := filepath.Join(projectRoot, ".sdp", "artifacts", card.ID, "plan.json")
		if data, err := os.ReadFile(planPath); err == nil {
			var existing PlanResult
			if err := json.Unmarshal(data, &existing); err == nil {
				existing.Status = "approved"
				return existing, nil
			}
		}
		return PlanResult{CardID: card.ID, Status: "approved"}, nil
	}

	if !cfg.Enabled {
		return PlanResult{CardID: card.ID, Status: "error"}, fmt.Errorf("planner is disabled")
	}

	prompt := BuildPlanPrompt(projectRoot, card)
	logger := log.New(log.Writer(), "[planner] ", log.LstdFlags)
	baseURL := strings.TrimSpace(os.Getenv("OMO_SERVE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:4096"
	}
	client := omoclient.NewClient(baseURL, logger)
	if _, err := client.ListSessions(); err != nil {
		return PlanResult{CardID: card.ID, Status: "error"}, fmt.Errorf("OmO serve unavailable — plan generation blocked")
	}

	invoker := omoclient.NewServeInvoker(baseURL, logger)
	runtimeResult, invokeErr := invoker.Invoke(ctx, kernel.RuntimeInvocation{
		WorkDir: projectRoot,
		Agent:   "sisyphus",
		Prompt:  prompt,
	})
	if invokeErr != nil || runtimeResult.ExitCode != 0 {
		return PlanResult{CardID: card.ID, Status: "error", RawPlan: runtimeResult.Output}, fmt.Errorf("OmO serve plan failed: %v", invokeErr)
	}

	payload := llmPlanPayload{}
	if err := json.NewDecoder(strings.NewReader(extractJSONObject(runtimeResult.Output))).Decode(&payload); err != nil {
		return PlanResult{CardID: card.ID, Status: "error", RawPlan: runtimeResult.Output}, fmt.Errorf("parse plan response: %w", err)
	}

	result := PlanResult{
		CardID:         card.ID,
		Status:         "generated",
		Approach:       strings.TrimSpace(payload.Approach),
		FilesToChange:  dedupeStrings(payload.FilesToChange),
		TestsToWrite:   dedupeStrings(payload.TestsToWrite),
		RiskAssessment: strings.TrimSpace(payload.RiskAssessment),
		EstimatedSteps: payload.EstimatedSteps,
		RawPlan:        runtimeResult.Output,
	}

	planPath := filepath.Join(projectRoot, ".sdp", "artifacts", card.ID, "plan.json")
	planData, _ := json.MarshalIndent(result, "", "  ")
	_ = os.MkdirAll(filepath.Dir(planPath), 0o755)
	_ = os.WriteFile(planPath, planData, 0o644)

	if isAutoApproveRisk(card.RiskLevel, cfg.AutoApprove) {
		result.Status = "approved"
		return result, nil
	}

	result.Status = "pending_approval"
	result.ApprovalPending = true
	return result, nil
}

func (b *ServeBridge) GeneratePlan(ctx context.Context, card *control.FeatureCard) (PlanResult, error) {
	if card == nil {
		return PlanResult{Status: "error"}, fmt.Errorf("nil card")
	}
	return GeneratePlan(ctx, b.ProjectRoot, card, b.Planner)
}

func (b *ServeBridge) RecordPlan(cardID string, result PlanResult) error {
	if b == nil || b.Store == nil {
		return fmt.Errorf("nil serve bridge/store")
	}

	planPath := filepath.Join(b.ProjectRoot, ".sdp", "artifacts", cardID, "plan.json")
	payload := map[string]any{
		"phase":            "plan",
		"card_id":          cardID,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"status":           result.Status,
		"approach":         result.Approach,
		"files_to_change":  result.FilesToChange,
		"tests_to_write":   result.TestsToWrite,
		"risk_assessment":  result.RiskAssessment,
		"estimated_steps":  result.EstimatedSteps,
		"raw_plan":         result.RawPlan,
		"approval_pending": result.ApprovalPending,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan artifact: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		return fmt.Errorf("mkdir plan artifact dir: %w", err)
	}
	if err := os.WriteFile(planPath, data, 0o644); err != nil {
		return fmt.Errorf("write plan artifact: %w", err)
	}

	card, err := b.Store.LoadCardByID(cardID)
	if err != nil {
		return fmt.Errorf("load card for plan: %w", err)
	}

	if result.Status == "pending_approval" || result.ApprovalPending {
		card.BlockingReasons = dedupeStrings(append(card.BlockingReasons, planPendingBlockingReason))
		card.WaitingOn = dedupeStrings(append(card.WaitingOn, "human"))
		card.NeedsFeedbackFrom = dedupeStrings(append(card.NeedsFeedbackFrom, "human"))
		card.RecommendedNextAction = "await_plan_approval"
		card.RecommendedNextReason = "Plan requires human approval before execution"
	} else if result.Status == "approved" {
		card.BlockingReasons = removeStringsLocal(card.BlockingReasons, []string{planPendingBlockingReason})
		card.ExecutorRuntimeState = "plan-approved"
	}

	if err := b.Store.SaveCard(card); err != nil {
		return fmt.Errorf("save plan card: %w", err)
	}

	if beadsRepo := b.Store.BeadsRepo(); beadsRepo != nil {
		_ = beadsRepo.LinkEvidence(cardID, "plan", []string{planPath})
	}

	return nil
}

func ApprovePlan(store *control.Store, projectRoot, cardID string) error {
	card, err := store.LoadCardByID(cardID)
	if err != nil {
		return fmt.Errorf("load card: %w", err)
	}

	card.BlockingReasons = removeStringsLocal(card.BlockingReasons, []string{planPendingBlockingReason})
	card.ExecutorRuntimeState = "plan-approved"
	card.WaitingOn = removeStringsLocal(card.WaitingOn, []string{"human"})
	card.NeedsFeedbackFrom = removeStringsLocal(card.NeedsFeedbackFrom, []string{"human"})
	card.RecommendedNextAction = "dispatch"
	card.RecommendedNextReason = "Plan approved, ready for execution"

	if err := store.SaveCard(card); err != nil {
		return fmt.Errorf("save approved card: %w", err)
	}

	planPath := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "plan.json")
	if data, err := os.ReadFile(planPath); err == nil {
		var existing map[string]any
		if err := json.Unmarshal(data, &existing); err == nil {
			existing["status"] = "approved"
			existing["approval_pending"] = false
			existing["approved_at"] = time.Now().UTC().Format(time.RFC3339)
			updated, _ := json.MarshalIndent(existing, "", "  ")
			_ = os.WriteFile(planPath, updated, 0o644)
		}
	}

	return nil
}

func LoadPlan(projectRoot, cardID string) (*PlanResult, error) {
	planPath := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "plan.json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	var result PlanResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}
	return &result, nil
}
