package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/control"
)

type SummaryResult struct {
	Text      string `json:"text"`
	Verdict   string `json:"verdict,omitempty"`
	CardID    string `json:"card_id"`
	Phase     string `json:"phase"`
	Timestamp string `json:"timestamp"`
}

type evaluationSummaryEvidence struct {
	Phase     string          `json:"phase"`
	CardID    string          `json:"card_id"`
	Timestamp string          `json:"timestamp"`
	Verdict   string          `json:"verdict"`
	Score     float64         `json:"score"`
	Findings  []string        `json:"findings"`
	Passed    map[string]bool `json:"passed"`
	Result    *EvalResult     `json:"result"`
}

type clarificationSummaryEvidence struct {
	Phase     string               `json:"phase"`
	CardID    string               `json:"card_id"`
	Timestamp string               `json:"timestamp"`
	Status    string               `json:"status"`
	Questions []string             `json:"questions"`
	Card      *control.FeatureCard `json:"card"`
}

func SummarizeCard(ctx context.Context, projectRoot, cardID string) (SummaryResult, error) {
	_ = ctx
	card, err := loadSummaryCard(projectRoot, cardID)
	if err != nil {
		return SummaryResult{}, err
	}

	artifactDir := filepath.Join(projectRoot, ".sdp", "artifacts", cardID)
	if clarification, ok, err := loadClarificationSummaryEvidence(filepath.Join(artifactDir, "clarification.json")); err != nil {
		return SummaryResult{}, err
	} else if ok {
		return buildClarificationSummary(cardID, card, clarification), nil
	}

	if evaluation, ok, err := loadEvaluationSummaryEvidence(filepath.Join(artifactDir, "evaluation.json")); err != nil {
		return SummaryResult{}, err
	} else if ok {
		return buildEvaluationSummary(projectRoot, cardID, card, evaluation), nil
	}

	return SummaryResult{
		Text:      "⚠️ Сводка недоступна\nНет evaluation.json или clarification.json\nСначала запусти clarify/eval для этой карточки",
		Verdict:   "blocked",
		CardID:    cardID,
		Phase:     "unknown",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func loadSummaryCard(projectRoot, cardID string) (*control.FeatureCard, error) {
	store, err := control.OpenFromEnv(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("open control store: %w", err)
	}
	card, err := store.LoadCardByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("load card: %w", err)
	}
	return card, nil
}

func loadEvaluationSummaryEvidence(path string) (evaluationSummaryEvidence, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return evaluationSummaryEvidence{}, false, nil
		}
		return evaluationSummaryEvidence{}, false, fmt.Errorf("read evaluation evidence: %w", err)
	}
	var payload evaluationSummaryEvidence
	if err := json.Unmarshal(data, &payload); err != nil {
		return evaluationSummaryEvidence{}, false, fmt.Errorf("parse evaluation evidence: %w", err)
	}
	if payload.Result != nil {
		if payload.Verdict == "" {
			payload.Verdict = payload.Result.Verdict
		}
		if payload.Score == 0 {
			payload.Score = payload.Result.Score
		}
		if len(payload.Findings) == 0 {
			payload.Findings = payload.Result.Findings
		}
		if len(payload.Passed) == 0 {
			payload.Passed = payload.Result.Passed
		}
	}
	return payload, true, nil
}

func loadClarificationSummaryEvidence(path string) (clarificationSummaryEvidence, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return clarificationSummaryEvidence{}, false, nil
		}
		return clarificationSummaryEvidence{}, false, fmt.Errorf("read clarification evidence: %w", err)
	}
	var payload clarificationSummaryEvidence
	if err := json.Unmarshal(data, &payload); err != nil {
		return clarificationSummaryEvidence{}, false, fmt.Errorf("parse clarification evidence: %w", err)
	}
	return payload, true, nil
}

func buildEvaluationSummary(projectRoot, cardID string, card *control.FeatureCard, evidence evaluationSummaryEvidence) SummaryResult {
	verdict := strings.TrimSpace(evidence.Verdict)
	if verdict == "" {
		verdict = "needs_review"
	}
	passed := evidence.Passed
	if passed == nil {
		passed = map[string]bool{}
	}
	lines := []string{
		fmt.Sprintf("📋 %s — %s", summaryTitle(card), summaryIntent(card)),
		fmt.Sprintf("Вердикт: %s | Score: %.2f/1.0 | Тесты: %s | Scope: %s", verdict, evidence.Score, passMark(passed["tests_pass"]), passMark(passed["scope_adherence"])),
	}
	if shouldShowFindings(verdict) {
		findings := limitStrings(evidence.Findings, 3)
		if len(findings) > 0 {
			lines = append(lines, fmt.Sprintf("Проблемы: %s", strings.Join(findings, ", ")))
		}
	}
	lines = append(lines, fmt.Sprintf("Файлы: %d changed", changedFileCount(projectRoot, cardID)))
	lines = limitLines(lines, 5)
	return SummaryResult{
		Text:      strings.Join(lines, "\n"),
		Verdict:   verdict,
		CardID:    cardID,
		Phase:     "evaluation",
		Timestamp: summaryTimestamp(evidence.Timestamp),
	}
}

func buildClarificationSummary(cardID string, card *control.FeatureCard, evidence clarificationSummaryEvidence) SummaryResult {
	activeCard := card
	if evidence.Card != nil {
		activeCard = evidence.Card
	}
	statusLine := "✅ Готово к работе"
	verdict := "ready"
	if strings.TrimSpace(evidence.Status) == "needs_clarification" {
		statusLine = "⚠️ Требуется уточнение"
		verdict = "needs_review"
	}
	lines := []string{
		fmt.Sprintf("📋 %s — %s", summaryTitle(activeCard), summaryRawIntent(activeCard)),
		statusLine,
	}
	for _, q := range limitStrings(evidence.Questions, 3) {
		lines = append(lines, "Q: "+q)
	}
	lines = limitLines(lines, 5)
	return SummaryResult{
		Text:      strings.Join(lines, "\n"),
		Verdict:   verdict,
		CardID:    cardID,
		Phase:     "clarification",
		Timestamp: summaryTimestamp(evidence.Timestamp),
	}
}

func summaryTitle(card *control.FeatureCard) string {
	if card == nil || strings.TrimSpace(card.Title) == "" {
		return "Без названия"
	}
	return strings.TrimSpace(card.Title)
}

func summaryIntent(card *control.FeatureCard) string {
	if card == nil {
		return "без описания"
	}
	if v := strings.TrimSpace(card.NormalizedIntent); v != "" {
		return truncateInline(v, 80)
	}
	if v := strings.TrimSpace(card.RawRequest); v != "" {
		return truncateInline(v, 80)
	}
	return "без описания"
}

func summaryRawIntent(card *control.FeatureCard) string {
	if card == nil {
		return "без описания"
	}
	if v := strings.TrimSpace(card.RawRequest); v != "" {
		return truncateInline(v, 80)
	}
	return summaryIntent(card)
}

func truncateInline(s string, max int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return strings.TrimSpace(s[:max-1]) + "…"
}

func passMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func shouldShowFindings(verdict string) bool {
	switch verdict {
	case evalVerdictFail, evalVerdictBlocked, evalVerdictNeedsReview:
		return true
	default:
		return false
	}
}

func changedFileCount(projectRoot, cardID string) int {
	evidence, _, err := loadBuildEvidence(projectRoot, cardID)
	if err != nil || evidence == nil {
		return 0
	}
	return len(dedupeStrings(evidence.FilesChanged))
}

func limitStrings(items []string, max int) []string {
	if max <= 0 || len(items) == 0 {
		return nil
	}
	out := make([]string, 0, max)
	for _, item := range items {
		item = truncateInline(item, 80)
		if item == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func limitLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	return lines[:max]
}

func summaryTimestamp(ts string) string {
	if strings.TrimSpace(ts) != "" {
		return ts
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func saveSummary(projectRoot, cardID string, summary SummaryResult) error {
	path := filepath.Join(projectRoot, ".sdp", "artifacts", cardID, "summary.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir summary dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(summary.Text+"\n"), 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}
