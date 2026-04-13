package strataudit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadRegressionFixtureConfig(t *testing.T) (*Config, string) {
	t.Helper()

	cfgPath, err := filepath.Abs(filepath.Join("testdata", "regression_corpus", "strataudit.yaml"))
	if err != nil {
		t.Fatalf("Abs(config): %v", err)
	}
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", cfgPath, err)
	}

	fixtureRoot := filepath.Dir(cfgPath)
	for i, src := range cfg.Project.SourceDirs {
		cfg.Project.SourceDirs[i] = filepath.Join(fixtureRoot, src)
	}
	cfg.Output.Dir = filepath.Join(t.TempDir(), cfg.Output.Dir)
	return cfg, fixtureRoot
}

func newRegressionMockLLMClient(t *testing.T) *LLMClient {
	t.Helper()

	var chatCalls, verifyCalls, embedCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll(%s): %v", r.URL.Path, err)
		}
		defer func() { _ = r.Body.Close() }()

		switch r.URL.Path {
		case "/chat/completions":
			var req struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode chat request: %v", err)
			}
			user := ""
			if len(req.Messages) > 0 {
				user = req.Messages[len(req.Messages)-1].Content
			}
			chatCalls++

			response := routeRegressionChatResponse(t, user, &verifyCalls)
			payload := fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":4,"cost":0.0001}}`, response)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(payload))
			return

		case "/embeddings":
			var req struct {
				Input []string `json:"input"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode embedding request: %v", err)
			}
			embedCalls++

			type item struct {
				Embedding []float32 `json:"embedding"`
			}
			resp := struct {
				Data []item `json:"data"`
			}{
				Data: make([]item, len(req.Input)),
			}
			for i, input := range req.Input {
				resp.Data[i] = item{Embedding: regressionEmbeddingFor(input)}
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode embedding response: %v", err)
			}
			return

		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(func() {
		srv.Close()
		if chatCalls == 0 {
			t.Error("regression mock llm was never called for chat/completions")
		}
		if verifyCalls == 0 {
			t.Error("regression mock llm never handled trace verification")
		}
		if embedCalls == 0 {
			t.Error("regression mock llm was never called for embeddings")
		}
	})

	client := NewLLMClient("test-key", srv.URL)
	client.SetRateLimit(1200)
	client.SetRetryConfig(0, 0)
	return client
}

func routeRegressionChatResponse(t *testing.T, user string, verifyCalls *int) string {
	t.Helper()
	user = normalizeRegressionPrompt(user)

	switch {
	case containsNormalizedAll(user,
		"Document level: vision",
		"Наша цель: создать единый платежный контур для всех продуктов компании.",
	):
		return `{"entities":[{"type":"goal","title_original":"Единый платежный контур","description_original":"Создать единый платежный контур для всех продуктов компании.","source_quote":"Наша цель: создать единый платежный контур для всех продуктов компании."}]}`

	case containsNormalizedAll(user,
		"Document level: strategy",
		"Наша программа: программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур.",
	):
		return `{"entities":[{"type":"objective","title_original":"Программа платежного хаба","description_original":"Программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур.","source_quote":"Наша программа: программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур."}]}`

	case containsNormalizedAll(user, "Document level: vision", "Template memo."):
		return `{"entities":[{"type":"goal","title_original":"Return valid JSON only","description_original":"Never fabricate quotes","source_quote":"Template memo."}]}`

	case containsNormalizedAll(user,
		"Assess whether the lower-level entity is meaningfully related to the upper-level entity",
		"Программа платежного хаба",
		"Единый платежный контур",
	):
		*verifyCalls = *verifyCalls + 1
		return `{"related": true, "confidence": 0.91, "relation": "contributes_to", "justification": "Нижняя инициатива прямо поддерживает верхнюю цель по evidence quotes."}`

	default:
		t.Fatalf("unexpected chat request in regression mock:\n%s", user)
		return ""
	}
}

func normalizeRegressionPrompt(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(SanitizeForPrompt(value))), " ")
}

func containsNormalizedAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, normalizeRegressionPrompt(needle)) {
			return false
		}
	}
	return true
}

func regressionEmbeddingFor(input string) []float32 {
	switch {
	case strings.Contains(input, "Единый платежный контур"):
		return []float32{1, 0, 0}
	case strings.Contains(input, "Программа платежного хаба"):
		return []float32{0.97, 0.03, 0}
	default:
		return []float32{0, 1, 0}
	}
}

func newRegressionStore(t *testing.T, cfg *Config) *SQLiteStore {
	t.Helper()

	dbPath := filepath.Join(cfg.Output.Dir, "strataudit.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("MkdirAll(output): %v", err)
	}
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func runRegressionPipeline(t *testing.T) (*Config, *SQLiteStore, *PipelineResult) {
	t.Helper()

	cfg, _ := loadRegressionFixtureConfig(t)
	store := newRegressionStore(t, cfg)
	llm := newRegressionMockLLMClient(t)

	result, err := RunPipeline(context.Background(), cfg, store, llm, PipelineOpts{})
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}
	return cfg, store, result
}

func containsPromptLeakMarker(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	markers := []string{
		"return valid json",
		"never fabricate quotes",
		"allowed entity types",
		"extract strategic entities",
		"document_content",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
