package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/executor/omoclient"
	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

const clarificationBlockingReason = "needs_clarification"

type ClarifyResult struct {
	Card        *control.FeatureCard `json:"card,omitempty"`
	Status      string               `json:"status"`
	Questions   []string             `json:"questions,omitempty"`
	RawFeedback string               `json:"raw_feedback,omitempty"`
}

type ClarifierConfig struct {
	Enabled     bool
	Model       string
	AutoApprove bool
}

type llmClarifyPayload struct {
	NormalizedIntent    string   `json:"normalized_intent"`
	ScopeIn             []string `json:"scope_in"`
	ScopeOut            []string `json:"scope_out"`
	Phase               string   `json:"phase"`
	RiskLevel           string   `json:"risk_level"`
	ClarificationNeeded bool     `json:"clarification_needed"`
	Questions           []string `json:"questions"`
	EstimatedComplexity string   `json:"estimated_complexity"`
}

func DefaultClarifierConfig() ClarifierConfig {
	model := strings.TrimSpace(os.Getenv("SDP_CLARIFY_MODEL"))
	if model == "" {
		model = "default"
	}
	return ClarifierConfig{Enabled: true, Model: model}
}

func hasBlockingReason(card *control.FeatureCard, reason string) bool {
	if card == nil {
		return false
	}
	for _, item := range card.BlockingReasons {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(reason)) {
			return true
		}
	}
	return false
}

func isAlreadyClarified(card *control.FeatureCard) bool {
	if card == nil {
		return false
	}
	if strings.TrimSpace(card.NormalizedIntent) == "" {
		return false
	}
	if len(card.ScopeIn) == 0 && len(card.ScopeOut) == 0 {
		return false
	}
	return !hasBlockingReason(card, clarificationBlockingReason)
}

func sanitizePhase(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "build", "fix", "refactor", "feature", "research":
		return strings.ToLower(strings.TrimSpace(phase))
	default:
		return "feature"
	}
}

func sanitizeRisk(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(risk))
	default:
		return "medium"
	}
}

func removeStringsLocal(from []string, remove []string) []string {
	if len(from) == 0 || len(remove) == 0 {
		return from
	}
	removeSet := make(map[string]struct{}, len(remove))
	for _, item := range remove {
		item = strings.TrimSpace(item)
		if item != "" {
			removeSet[item] = struct{}{}
		}
	}
	if len(removeSet) == 0 {
		return from
	}
	result := make([]string, 0, len(from))
	for _, item := range from {
		trimmed := strings.TrimSpace(item)
		if _, ok := removeSet[trimmed]; ok {
			continue
		}
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func ClarifyIntent(ctx context.Context, projectRoot, rawIntent string, existingCard *control.FeatureCard) (ClarifyResult, error) {
	return ClarifyIntentWithConfig(ctx, projectRoot, rawIntent, existingCard, DefaultClarifierConfig())
}

func ClarifyIntentWithConfig(ctx context.Context, projectRoot, rawIntent string, existingCard *control.FeatureCard, cfg ClarifierConfig) (ClarifyResult, error) {
	if existingCard == nil {
		return ClarifyResult{Status: "error"}, fmt.Errorf("nil card")
	}
	if isAlreadyClarified(existingCard) {
		return ClarifyResult{Card: existingCard, Status: "ready"}, nil
	}
	if !cfg.Enabled {
		return ClarifyResult{Card: existingCard, Status: "error", Questions: []string{"clarifier is disabled"}}, nil
	}

	prompt := BuildClarificationPrompt(projectRoot, rawIntent)
	baseURL := strings.TrimSpace(os.Getenv("OMO_SERVE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:4096"
	}
	client := omoclient.NewClient(baseURL)
	if _, err := client.ListSessions(); err != nil {
		result := ClarifyResult{Card: existingCard, Status: "error", Questions: []string{"OmO serve unavailable — clarification blocked"}}
		return result, nil
	}
	runtimeResult, invokeErr := InvokeWithFallback(ctx, kernel.RuntimeInvocation{
		WorkDir: projectRoot,
		Agent:   "sisyphus",
		Prompt:  prompt,
	})
	if invokeErr != nil || runtimeResult.ExitCode != 0 {
		result := ClarifyResult{Card: existingCard, Status: "error", RawFeedback: runtimeResult.Output, Questions: []string{fmt.Sprintf("OmO serve clarification failed: %v", invokeErr)}}
		return result, nil
	}

	payload := llmClarifyPayload{}
	if err := json.NewDecoder(strings.NewReader(extractJSONObject(runtimeResult.Output))).Decode(&payload); err != nil {
		return ClarifyResult{Card: existingCard, Status: "error", RawFeedback: runtimeResult.Output}, nil
	}

	cardCopy := *existingCard
	cardCopy.NormalizedIntent = strings.TrimSpace(payload.NormalizedIntent)
	cardCopy.ScopeIn = dedupeStrings(payload.ScopeIn)
	cardCopy.ScopeOut = dedupeStrings(payload.ScopeOut)
	cardCopy.TaskType = sanitizePhase(payload.Phase)
	cardCopy.RiskLevel = sanitizeRisk(payload.RiskLevel)
	cardCopy.ExecutionMode = payload.EstimatedComplexity
	cardCopy.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	cardCopy.BlockingReasons = removeStringsLocal(cardCopy.BlockingReasons, []string{clarificationBlockingReason})

	if payload.ClarificationNeeded {
		questions := dedupeStrings(payload.Questions)
		cardCopy.OpenQuestions = dedupeStrings(append(cardCopy.OpenQuestions, questions...))
		cardCopy.BlockingReasons = dedupeStrings(append(cardCopy.BlockingReasons, clarificationBlockingReason))
		return ClarifyResult{Card: &cardCopy, Status: "needs_clarification", Questions: questions, RawFeedback: runtimeResult.Output}, nil
	}

	cardCopy.OpenQuestions = removeStringsLocal(cardCopy.OpenQuestions, payload.Questions)
	return ClarifyResult{Card: &cardCopy, Status: "ready", RawFeedback: runtimeResult.Output}, nil
}

func (b *ServeBridge) Clarify(ctx context.Context, card *control.FeatureCard) (ClarifyResult, error) {
	if card == nil {
		return ClarifyResult{Status: "error"}, fmt.Errorf("nil card")
	}
	intent := card.RawRequest
	if strings.TrimSpace(intent) == "" {
		intent = card.NormalizedIntent
	}
	if strings.TrimSpace(intent) == "" {
		intent = card.Title
	}
	return ClarifyIntentWithConfig(ctx, b.ProjectRoot, intent, card, DefaultClarifierConfig())
}

func (b *ServeBridge) RecordClarification(cardID string, result ClarifyResult) error {
	if b == nil || b.Store == nil {
		return fmt.Errorf("nil serve bridge/store")
	}
	if result.Card == nil {
		return fmt.Errorf("clarify result card is required")
	}
	artifactPath := filepath.Join(b.ProjectRoot, ".sdp", "artifacts", cardID, "clarification.json")
	payload := map[string]any{
		"phase":        "clarification",
		"card_id":      cardID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"status":       result.Status,
		"questions":    result.Questions,
		"raw_feedback": result.RawFeedback,
		"card":         result.Card,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal clarification evidence: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		return fmt.Errorf("mkdir clarification artifact dir: %w", err)
	}
	if err := os.WriteFile(artifactPath, data, 0o644); err != nil {
		return fmt.Errorf("write clarification artifact: %w", err)
	}

	card := result.Card
	if result.Status == "needs_clarification" {
		card.Status = "needs_input"
		card.WaitingOn = dedupeStrings(append(card.WaitingOn, "human"))
		card.NeedsFeedbackFrom = dedupeStrings(append(card.NeedsFeedbackFrom, "human"))
		card.FeedbackRequest = dedupeStrings(append(card.FeedbackRequest, result.Questions...))
		card.BlockingReasons = dedupeStrings(append(card.BlockingReasons, clarificationBlockingReason))
		card.RecommendedNextAction = "await_human_input"
		card.RecommendedNextReason = "Clarifier found ambiguity that requires human input before dispatch"
	} else {
		card.BlockingReasons = removeStringsLocal(card.BlockingReasons, []string{clarificationBlockingReason})
		card.FeedbackRequest = removeStringsLocal(card.FeedbackRequest, result.Questions)
	}
	if err := b.Store.SaveCard(card); err != nil {
		return fmt.Errorf("save clarified card: %w", err)
	}
	if beadsRepo := b.Store.BeadsRepo(); beadsRepo != nil {
		_ = beadsRepo.LinkEvidence(cardID, "clarification", []string{artifactPath})
	}
	if card.ProjectID != "" {
		_, _ = b.Store.BuildProjectSnapshot(card.ProjectID)
	}
	_, _ = b.Store.BuildPortfolioSnapshot()
	return nil
}
