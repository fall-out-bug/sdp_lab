package strataudit

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version     string          `yaml:"version"`
	Project     ProjectConfig   `yaml:"project"`
	Levels      []LevelConfig   `yaml:"levels"`
	EntityTypes []string        `yaml:"entity_types"`
	LLM         LLMConfig       `yaml:"llm"`
	Thresholds  ThresholdConfig `yaml:"thresholds"`
	Output      OutputConfig    `yaml:"output"`
	Extractors  ExtractorsConfig `yaml:"extractors"`
}

type ProjectConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	SourceDirs  []string `yaml:"source_dirs"`
	Exclude     []string `yaml:"exclude"`
}

type LevelConfig struct {
	Name        string   `yaml:"name"`
	Rank        int      `yaml:"rank"`
	Description string   `yaml:"description"`
	Patterns    []string `yaml:"patterns"`
}

type LLMConfig struct {
	Model            string             `yaml:"model"`
	ExtractModel     string             `yaml:"extract_model"`
	EmbeddingModel   string             `yaml:"embedding_model"`
	EmbeddingDims    int                `yaml:"embedding_dims"`
	Temperature      float64            `yaml:"temperature"`
	Temperatures     map[string]float64 `yaml:"temperatures"`
	RequestsPerMin   int                `yaml:"requests_per_minute"`
	MaxConcurrent    int                `yaml:"max_concurrent"`
	MaxRetries       int                `yaml:"max_retries"`
	RetryBaseDelayMs int                `yaml:"retry_base_delay_ms"`
}

type ThresholdConfig struct {
	Similarity            float64 `yaml:"similarity"`
	TraceConfidence       float64 `yaml:"trace_confidence"`
	AutoVerifySimilarity  float64 `yaml:"auto_verify_similarity"`
	LLMVerifyBudget       int     `yaml:"llm_verify_budget"`
	CoverageWarn          float64 `yaml:"coverage_warn"`
	StaleDays             int     `yaml:"stale_days"`
	ChunkTokenLimit       int     `yaml:"chunk_token_limit"`
	ChunkOverlapTokens    int     `yaml:"chunk_overlap_tokens"`
	EmitDistribution      bool    `yaml:"emit_distribution"`
	MaxChunksPerDocument  int     `yaml:"max_chunks_per_document"`
	AdaptiveSimilarity    bool    `yaml:"adaptive_similarity"`
}

type OutputConfig struct {
	Dir     string   `yaml:"dir"`
	Formats []string `yaml:"formats"`
	Lang    string   `yaml:"lang"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.setDefaults()
	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.LLM.MaxRetries == 0 {
		c.LLM.MaxRetries = 3
	}
	if c.LLM.RequestsPerMin == 0 {
		c.LLM.RequestsPerMin = 30
	}
	if c.LLM.MaxConcurrent == 0 {
		c.LLM.MaxConcurrent = 5
	}
	if c.LLM.RetryBaseDelayMs == 0 {
		c.LLM.RetryBaseDelayMs = 1000
	}
	if c.Thresholds.Similarity == 0 {
		c.Thresholds.Similarity = 0.5
	}
	if c.Thresholds.TraceConfidence == 0 {
		c.Thresholds.TraceConfidence = 0.6
	}
	if c.Thresholds.AutoVerifySimilarity == 0 {
		c.Thresholds.AutoVerifySimilarity = 0.85
	}
	if c.Thresholds.LLMVerifyBudget == 0 {
		c.Thresholds.LLMVerifyBudget = 50
	}
	if c.Thresholds.CoverageWarn == 0 {
		c.Thresholds.CoverageWarn = 70
	}
	if c.Thresholds.StaleDays == 0 {
		c.Thresholds.StaleDays = 90
	}
	if c.Thresholds.ChunkTokenLimit == 0 {
		c.Thresholds.ChunkTokenLimit = 3000
	}
	if c.Thresholds.ChunkOverlapTokens == 0 {
		c.Thresholds.ChunkOverlapTokens = 500
	}
	if c.Thresholds.MaxChunksPerDocument == 0 {
		c.Thresholds.MaxChunksPerDocument = 100
	}
	if c.LLM.EmbeddingDims == 0 {
		c.LLM.EmbeddingDims = 1536
	}
	if c.Output.Dir == "" {
		c.Output.Dir = ".strataudit"
	}
	if c.Output.Lang == "" {
		c.Output.Lang = "ru"
	}
	if !c.Thresholds.EmitDistribution {
		c.Thresholds.EmitDistribution = true
	}
	if len(c.Output.Formats) == 0 {
		c.Output.Formats = []string{"html", "json"}
	}
	if len(c.EntityTypes) == 0 {
		c.EntityTypes = []string{"goal", "objective", "kpi", "initiative", "task", "principle", "stakeholder", "capability"}
	}
}

func (c *Config) Validate() error {
	if len(c.Levels) == 0 {
		return fmt.Errorf("at least one level must be defined")
	}
	ranks := make(map[int]string)
	for _, l := range c.Levels {
		if existing, ok := ranks[l.Rank]; ok {
			return fmt.Errorf("duplicate rank %d: %q and %q", l.Rank, existing, l.Name)
		}
		ranks[l.Rank] = l.Name
	}
	if c.Thresholds.ChunkOverlapTokens >= c.Thresholds.ChunkTokenLimit {
		return fmt.Errorf("chunk_overlap_tokens (%d) must be less than chunk_token_limit (%d)", c.Thresholds.ChunkOverlapTokens, c.Thresholds.ChunkTokenLimit)
	}
	return nil
}

func (c *Config) SortedLevels() []LevelConfig {
	sorted := make([]LevelConfig, len(c.Levels))
	copy(sorted, c.Levels)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rank < sorted[j].Rank
	})
	return sorted
}

func (c *Config) TemperatureForStage(stage string) float64 {
	if c.LLM.Temperatures != nil {
		if t, ok := c.LLM.Temperatures[stage]; ok {
			return t
		}
	}
	return c.LLM.Temperature
}

func DefaultConfigYAML() *Config {
	return &Config{
		Version: "1",
		Project: ProjectConfig{
			Name:        "My Project",
			Description: "Strategy traceability audit",
			SourceDirs:  []string{"docs/strategy", "docs/plans", "docs/roadmap"},
			Exclude:     []string{"*.tmp", ".git/**"},
		},
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0, Description: "Vision and mission", Patterns: []string{"*vision*", "*mission*"}},
			{Name: "strategy", Rank: 1, Description: "Strategic goals", Patterns: []string{"*strategy*", "*стратег*"}},
			{Name: "plan", Rank: 2, Description: "Plans and roadmaps", Patterns: []string{"*roadmap*", "*plan*"}},
			{Name: "initiative", Rank: 3, Description: "Initiatives", Patterns: []string{"*initiative*", "*project*"}},
			{Name: "task", Rank: 4, Description: "Tasks", Patterns: []string{"*sprint*", "*backlog*"}},
		},
		EntityTypes: []string{"goal", "objective", "kpi", "initiative", "task", "principle", "stakeholder", "capability"},
		LLM: LLMConfig{
			Model:          "deepseek/deepseek-v3.2",
			ExtractModel:   "deepseek/deepseek-v3.2",
			EmbeddingModel: "openai/text-embedding-3-small",
			EmbeddingDims:  1536,
			Temperature:    0.1,
			Temperatures:   map[string]float64{"classify": 0.0, "extract": 0.1, "verify": 0.0, "infer": 0.3},
			RequestsPerMin: 30, MaxConcurrent: 5, MaxRetries: 3, RetryBaseDelayMs: 1000,
		},
		Thresholds: ThresholdConfig{Similarity: 0.5, TraceConfidence: 0.6, CoverageWarn: 70, StaleDays: 90, ChunkTokenLimit: 3000, ChunkOverlapTokens: 500},
		Output:     OutputConfig{Dir: ".strataudit", Formats: []string{"html", "json"}},
	}
}
