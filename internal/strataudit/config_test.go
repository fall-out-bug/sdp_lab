package strataudit

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidYAML(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("testdata", "valid-config.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Project.Name != "Test Project" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "Test Project")
	}
	if len(cfg.Levels) != 3 {
		t.Fatalf("len(Levels) = %d, want 3", len(cfg.Levels))
	}
	if cfg.Levels[0].Name != "vision" {
		t.Errorf("Levels[0].Name = %q, want %q", cfg.Levels[0].Name, "vision")
	}
	if cfg.Levels[0].Rank != 0 {
		t.Errorf("Levels[0].Rank = %d, want 0", cfg.Levels[0].Rank)
	}
	if cfg.Thresholds.Similarity != 0.5 {
		t.Errorf("Thresholds.Similarity = %f, want 0.5", cfg.Thresholds.Similarity)
	}
	if cfg.Runtime.Provider != "openrouter" {
		t.Errorf("Runtime.Provider = %q, want openrouter", cfg.Runtime.Provider)
	}
	if cfg.Runtime.APIKeyEnv != "OPENROUTER_API_KEY" {
		t.Errorf("Runtime.APIKeyEnv = %q, want OPENROUTER_API_KEY", cfg.Runtime.APIKeyEnv)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg, _ := LoadConfig(filepath.Join("testdata", "valid-config.yaml"))
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_ValidateDuplicateRanks(t *testing.T) {
	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "a", Rank: 0},
			{Name: "b", Rank: 0},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for duplicate ranks")
	}
}

func TestConfig_SortedLevels(t *testing.T) {
	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "task", Rank: 2},
			{Name: "vision", Rank: 0},
			{Name: "strategy", Rank: 1},
		},
	}
	sorted := cfg.SortedLevels()
	if sorted[0].Name != "vision" || sorted[1].Name != "strategy" || sorted[2].Name != "task" {
		t.Errorf("SortedLevels order: %v", sorted)
	}
}

func TestConfig_TemperatureForStage(t *testing.T) {
	cfg := &Config{
		LLM: LLMConfig{
			Temperature: 0.1,
			Temperatures: map[string]float64{
				"classify": 0.0,
				"extract":  0.1,
				"verify":   0.0,
				"infer":    0.3,
			},
		},
	}
	if v := cfg.TemperatureForStage("infer"); v != 0.3 {
		t.Errorf("TemperatureForStage(infer) = %f, want 0.3", v)
	}
	if v := cfg.TemperatureForStage("unknown"); v != 0.1 {
		t.Errorf("TemperatureForStage(unknown) = %f, want 0.1 (default)", v)
	}
}

func TestConfig_SetDefaultsRuntime(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	if cfg.Runtime.Provider != "openrouter" {
		t.Fatalf("Runtime.Provider = %q, want openrouter", cfg.Runtime.Provider)
	}
	if cfg.Runtime.BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("Runtime.BaseURL = %q", cfg.Runtime.BaseURL)
	}
	if cfg.Runtime.APIKeyEnv != "OPENROUTER_API_KEY" {
		t.Fatalf("Runtime.APIKeyEnv = %q", cfg.Runtime.APIKeyEnv)
	}
}

func TestDefaultConfigYAML_IsFullyMaterialized(t *testing.T) {
	cfg := DefaultConfigYAML()

	if cfg.Output.Lang != "ru" {
		t.Fatalf("Output.Lang = %q, want ru", cfg.Output.Lang)
	}
	if !cfg.Thresholds.EmitDistribution {
		t.Fatal("EmitDistribution should be true in generated template")
	}
	if cfg.Thresholds.AutoVerifySimilarity != 0.85 {
		t.Fatalf("AutoVerifySimilarity = %v, want 0.85", cfg.Thresholds.AutoVerifySimilarity)
	}
	if cfg.Thresholds.LLMVerifyBudget != 50 {
		t.Fatalf("LLMVerifyBudget = %d, want 50", cfg.Thresholds.LLMVerifyBudget)
	}
	if cfg.Thresholds.MaxChunksPerDocument != 100 {
		t.Fatalf("MaxChunksPerDocument = %d, want 100", cfg.Thresholds.MaxChunksPerDocument)
	}
}
