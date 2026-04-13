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
