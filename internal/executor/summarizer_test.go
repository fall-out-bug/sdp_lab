package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/control"
)

func TestSummarizeCard_Evaluation(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Починить executor loop", "почини цикл")
	if err != nil {
		t.Fatal(err)
	}
	card.NormalizedIntent = "Исправить обработку evaluation summary"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	artifactDir := filepath.Join(store.ProjectRoot, ".sdp", "artifacts", card.ID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	buildPayload := map[string]any{
		"phase":         "build",
		"card_id":       card.ID,
		"exit_code":     0,
		"status":        control.ResultStatusSuccess,
		"files_changed": []string{"internal/executor/loop_v2.go", "internal/executor/summarizer.go"},
	}
	writeJSONFile(t, filepath.Join(artifactDir, "build.json"), buildPayload)
	writeJSONFile(t, filepath.Join(artifactDir, "evaluation.json"), map[string]any{
		"phase":     "evaluation",
		"card_id":   card.ID,
		"timestamp": "2026-03-25T16:00:00Z",
		"verdict":   "needs_review",
		"score":     0.65,
		"findings":  []string{"нет явного покрытия тестами", "нужно проверить scope"},
		"passed": map[string]bool{
			"tests_pass":      true,
			"scope_adherence": false,
		},
	})

	result, err := SummarizeCard(context.Background(), store.ProjectRoot, card.ID)
	if err != nil {
		t.Fatalf("SummarizeCard error: %v", err)
	}
	if result.Phase != "evaluation" {
		t.Fatalf("phase = %s, want evaluation", result.Phase)
	}
	if !strings.Contains(result.Text, "📋 Починить executor loop") {
		t.Fatalf("summary missing title: %s", result.Text)
	}
	if !strings.Contains(result.Text, "Вердикт: needs_review | Score: 0.65/1.0 | Тесты: ✓ | Scope: ✗") {
		t.Fatalf("summary missing verdict line: %s", result.Text)
	}
	if !strings.Contains(result.Text, "Проблемы: нет явного покрытия тестами, нужно проверить scope") {
		t.Fatalf("summary missing findings: %s", result.Text)
	}
	if !strings.Contains(result.Text, "Файлы: 2 changed") {
		t.Fatalf("summary missing file count: %s", result.Text)
	}
}

func TestSummarizeCard_Clarification(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Новый summarizer", "сделай summarizer для evidence")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	artifactDir := filepath.Join(store.ProjectRoot, ".sdp", "artifacts", card.ID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, filepath.Join(artifactDir, "clarification.json"), map[string]any{
		"phase":     "clarification",
		"card_id":   card.ID,
		"timestamp": "2026-03-25T16:05:00Z",
		"status":    "needs_clarification",
		"questions": []string{"Какие артефакты читать?", "Нужен ли русский вывод?"},
		"card": map[string]any{
			"id":          card.ID,
			"project_id":  card.ProjectID,
			"title":       card.Title,
			"raw_request": card.RawRequest,
		},
	})

	result, err := SummarizeCard(context.Background(), store.ProjectRoot, card.ID)
	if err != nil {
		t.Fatalf("SummarizeCard error: %v", err)
	}
	if result.Phase != "clarification" {
		t.Fatalf("phase = %s, want clarification", result.Phase)
	}
	if !strings.Contains(result.Text, "⚠️ Требуется уточнение") {
		t.Fatalf("summary missing clarification status: %s", result.Text)
	}
	if !strings.Contains(result.Text, "Q: Какие артефакты читать?") || !strings.Contains(result.Text, "Q: Нужен ли русский вывод?") {
		t.Fatalf("summary missing questions: %s", result.Text)
	}
}

func TestSummarizeCard_NoEvidence(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Пустая карточка", "ничего нет")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	result, err := SummarizeCard(context.Background(), store.ProjectRoot, card.ID)
	if err != nil {
		t.Fatalf("SummarizeCard error: %v", err)
	}
	if result.Verdict != "blocked" {
		t.Fatalf("verdict = %s, want blocked", result.Verdict)
	}
	if !strings.Contains(result.Text, "Нет evaluation.json или clarification.json") {
		t.Fatalf("summary = %s", result.Text)
	}
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
