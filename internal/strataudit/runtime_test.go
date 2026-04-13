package strataudit

import (
	"context"
	"path/filepath"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func TestResolveRuntime_HostProviderRejected(t *testing.T) {
	cfg := DefaultConfigYAML()
	cfg.Runtime.Provider = "host"
	cfg.Runtime.BaseURL = ""
	cfg.Runtime.APIKeyEnv = ""

	_, err := cfg.ResolveRuntime()
	if err == nil {
		t.Fatal("expected host runtime rejection")
	}
	if got := err.Error(); got == "" || got == "unsupported runtime" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRuntime_UsesConfiguredAPIKeyEnv(t *testing.T) {
	t.Setenv("STRATAUDIT_TEST_API_KEY", "secret")

	cfg := DefaultConfigYAML()
	cfg.Runtime.Provider = "openai_compatible"
	cfg.Runtime.BaseURL = "https://example.invalid/v1"
	cfg.Runtime.APIKeyEnv = "STRATAUDIT_TEST_API_KEY"

	runtime, err := cfg.ResolveRuntime()
	if err != nil {
		t.Fatalf("ResolveRuntime: %v", err)
	}

	client, ok := runtime.(*LLMClient)
	if !ok {
		t.Fatalf("runtime type = %T, want *LLMClient", runtime)
	}
	if client.baseURL != "https://example.invalid/v1" {
		t.Fatalf("client.baseURL = %q", client.baseURL)
	}
}

func TestExtractEntities_UsesInjectedRuntime(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	cfg := DefaultConfigYAML()
	cfg.Levels = []LevelConfig{{Name: "strategy", Rank: 1, Description: "Strategy"}}
	cfg.EntityTypes = []string{"goal"}
	cfg.Thresholds.ChunkTokenLimit = 1000
	cfg.Thresholds.ChunkOverlapTokens = 100
	cfg.LLM.ExtractModel = "test-extract"
	cfg.LLM.EmbeddingModel = "test-embed"

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "strategy", Name: "strategy", Rank: 1},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{
			ID:          "doc1",
			Path:        "strategy.md",
			LevelID:     "strategy",
			ContentHash: "hash1",
			Content:     "Наша цель — выйти на рынок Ближнего Востока.",
		},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}

	runtime := FunctionalRuntime{
		ChatFunc: func(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
			return &LLMResponse{
				Content: `{"entities":[{"type":"goal","title":"Выход на Ближний Восток","description":"Экспансия на новый рынок","source_quote":"Наша цель — выйти на рынок Ближнего Востока."}]}`,
			}, nil
		},
		EmbedFunc: func(ctx context.Context, texts []string, model string) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{0.1, 0.2, 0.3}
			}
			return result, nil
		},
	}

	res, err := ExtractEntities(ctx, cfg, store, runtime)
	if err != nil {
		t.Fatalf("ExtractEntities: %v", err)
	}
	if res.EntitiesExtracted != 1 {
		t.Fatalf("EntitiesExtracted = %d, want 1", res.EntitiesExtracted)
	}

	entities, err := store.EntitiesByLevel(ctx, "strategy", model.Page{Limit: 10})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want 1", len(entities))
	}
	if entities[0].Title != "Выход на Ближний Восток" {
		t.Fatalf("title = %q", entities[0].Title)
	}
	if len(entities[0].Embedding) != 3 {
		t.Fatalf("embedding dims = %d, want 3", len(entities[0].Embedding))
	}
}
