package strataudit

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func loadRegressionFixtureConfig(t *testing.T) (*Config, string) {
	t.Helper()

	cfg, fixtureRoot, err := loadRegressionFixtureConfigForOutput(filepath.Join(t.TempDir(), ".strataudit"))
	if err != nil {
		t.Fatalf("load regression fixture config: %v", err)
	}
	return cfg, fixtureRoot
}

func newRegressionMockLLMClient(t *testing.T) *regressionFixtureRuntime {
	t.Helper()

	runtime := newRegressionFixtureRuntime()
	assertRegressionRuntimeStats(t, runtime)
	return runtime
}

func newRegressionStore(t *testing.T, cfg *Config) *SQLiteStore {
	t.Helper()

	store, err := openRegressionFixtureStore(cfg)
	if err != nil {
		t.Fatalf("open regression fixture store: %v", err)
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

func runRegressionPipelineWithRuntime(t *testing.T, runtime ModelRuntime) (*Config, *SQLiteStore, *PipelineResult) {
	t.Helper()

	cfg, _ := loadRegressionFixtureConfig(t)
	store := newRegressionStore(t, cfg)

	result, err := RunPipeline(context.Background(), cfg, store, runtime, PipelineOpts{})
	if err != nil {
		t.Fatalf("RunPipeline: %v", err)
	}
	return cfg, store, result
}

func newRegressionZeroTraceRuntime(t *testing.T) ModelRuntime {
	t.Helper()

	runtime := newRegressionFixtureRuntime()
	assertRegressionRuntimeStats(t, runtime)

	return FunctionalRuntime{
		ChatFunc: func(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
			resp, err := runtime.Chat(ctx, req)
			if err != nil {
				return nil, err
			}
			if strings.EqualFold(strings.TrimSpace(req.Stage), "verify") {
				resp = &LLMResponse{
					Content:       `{"related": false, "confidence": 0.19, "relation": "none", "justification": "Evidence quotes do not prove a strategic relation."}`,
					ContentSource: resp.ContentSource,
					PromptHash:    resp.PromptHash,
					TokensIn:      resp.TokensIn,
					TokensOut:     resp.TokensOut,
					Model:         resp.Model,
				}
			}
			return resp, nil
		},
		EmbedFunc: runtime.Embed,
	}
}

func assertRegressionRuntimeStats(t *testing.T, runtime *regressionFixtureRuntime) {
	t.Helper()

	t.Cleanup(func() {
		stats := runtime.Stats()
		if stats.ChatCalls == 0 {
			t.Error("regression mock llm was never called for chat")
		}
		if stats.VerifyCalls == 0 {
			t.Error("regression mock llm never handled trace verification")
		}
		if stats.EmbedCalls == 0 {
			t.Error("regression mock llm was never called for embeddings")
		}
	})
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
