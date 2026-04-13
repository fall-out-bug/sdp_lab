package strataudit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type RegressionDemoResult struct {
	Config          *Config
	FixtureRoot     string
	FixtureDocsDir  string
	OutputDir       string
	ReportHTMLPath  string
	ReportJSONPath  string
	ReportCompat    string
	DiagnosticsPath string
	DatabasePath    string
	Result          *PipelineResult
}

type regressionFixtureRuntimeStats struct {
	ChatCalls   int
	VerifyCalls int
	EmbedCalls  int
}

type regressionFixtureRuntime struct {
	mu    sync.Mutex
	stats regressionFixtureRuntimeStats
}

func RunRegressionDemo(ctx context.Context, outputDir string) (*RegressionDemoResult, error) {
	cfg, fixtureRoot, err := loadRegressionFixtureConfigForOutput(outputDir)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("regression demo config invalid: %w", err)
	}
	if err := prepareRegressionDemoOutput(cfg.Output.Dir); err != nil {
		return nil, err
	}

	store, err := openRegressionFixtureStore(cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = store.Close() }()

	runtime := newRegressionFixtureRuntime()
	result, err := RunPipeline(ctx, cfg, store, runtime, PipelineOpts{})
	if err != nil {
		return nil, fmt.Errorf("run regression demo: %w", err)
	}

	return &RegressionDemoResult{
		Config:          cfg,
		FixtureRoot:     fixtureRoot,
		FixtureDocsDir:  filepath.Join(fixtureRoot, "docs"),
		OutputDir:       cfg.Output.Dir,
		ReportHTMLPath:  filepath.Join(cfg.Output.Dir, "report.html"),
		ReportJSONPath:  filepath.Join(cfg.Output.Dir, "report.v2.json"),
		ReportCompat:    filepath.Join(cfg.Output.Dir, "report.json"),
		DiagnosticsPath: filepath.Join(cfg.Output.Dir, "llm_diagnostics.json"),
		DatabasePath:    filepath.Join(cfg.Output.Dir, "strataudit.db"),
		Result:          result,
	}, nil
}

func loadRegressionFixtureConfigForOutput(outputDir string) (*Config, string, error) {
	fixtureRoot, err := regressionFixtureRoot()
	if err != nil {
		return nil, "", err
	}

	cfgPath := filepath.Join(fixtureRoot, "strataudit.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return nil, "", fmt.Errorf("load regression fixture config: %w", err)
	}

	for i, src := range cfg.Project.SourceDirs {
		cfg.Project.SourceDirs[i] = filepath.Join(fixtureRoot, src)
	}

	resolvedOutputDir, err := resolveRegressionDemoOutputDir(outputDir)
	if err != nil {
		return nil, "", err
	}
	cfg.Output.Dir = resolvedOutputDir

	return cfg, fixtureRoot, nil
}

func resolveRegressionDemoOutputDir(outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		outputDir = ".strataudit-demo"
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve regression demo output dir: %w", err)
	}
	return absOutputDir, nil
}

func regressionFixtureRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve regression fixture root: runtime caller unavailable")
	}
	return filepath.Join(filepath.Dir(currentFile), "testdata", "regression_corpus"), nil
}

func prepareRegressionDemoOutput(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create regression demo output dir: %w", err)
	}

	for _, name := range []string{
		"strataudit.db",
		"report.html",
		"report.v2.json",
		"report.json",
		"similarity_distribution.json",
		"llm_diagnostics.json",
	} {
		path := filepath.Join(outputDir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale regression demo artifact %s: %w", path, err)
		}
	}

	return nil
}

func openRegressionFixtureStore(cfg *Config) (*SQLiteStore, error) {
	dbPath := filepath.Join(cfg.Output.Dir, "strataudit.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create regression fixture output dir: %w", err)
	}
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open regression fixture store: %w", err)
	}
	return store, nil
}

func newRegressionFixtureRuntime() *regressionFixtureRuntime {
	return &regressionFixtureRuntime{}
}

func (r *regressionFixtureRuntime) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	_ = ctx

	r.mu.Lock()
	r.stats.ChatCalls++
	r.mu.Unlock()

	content, verify, err := regressionFixtureChatResponse(req.User)
	if err != nil {
		return nil, err
	}
	if verify {
		r.mu.Lock()
		r.stats.VerifyCalls++
		r.mu.Unlock()
	}

	return &LLMResponse{
		Content:       content,
		ContentSource: "content",
		PromptHash:    regressionFixturePromptHash(req),
		TokensIn:      10,
		TokensOut:     4,
		Model:         req.Model,
	}, nil
}

func (r *regressionFixtureRuntime) Embed(ctx context.Context, texts []string, model string) ([][]float32, error) {
	_ = ctx
	_ = model

	r.mu.Lock()
	r.stats.EmbedCalls++
	r.mu.Unlock()

	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vectors = append(vectors, regressionEmbeddingFor(text))
	}
	return vectors, nil
}

func (r *regressionFixtureRuntime) Stats() regressionFixtureRuntimeStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

func regressionFixtureChatResponse(user string) (content string, verify bool, err error) {
	user = normalizeRegressionPrompt(user)

	switch {
	case containsNormalizedAll(user,
		"Document level: vision",
		"Наша цель: создать единый платежный контур для всех продуктов компании.",
	):
		return `{"entities":[{"type":"goal","title_original":"Единый платежный контур","description_original":"Создать единый платежный контур для всех продуктов компании.","source_quote":"Наша цель: создать единый платежный контур для всех продуктов компании."}]}`, false, nil

	case containsNormalizedAll(user,
		"Document level: strategy",
		"Наша программа: программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур.",
	):
		return `{"entities":[{"type":"objective","title_original":"Программа платежного хаба","description_original":"Программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур.","source_quote":"Наша программа: программа платежного хаба объединяет маршрутизацию платежей и поддерживает единый платежный контур."}]}`, false, nil

	case containsNormalizedAll(user, "Document level: vision", "Template memo."):
		return `{"entities":[{"type":"goal","title_original":"Return valid JSON only","description_original":"Never fabricate quotes","source_quote":"Template memo."}]}`, false, nil

	case containsNormalizedAll(user,
		"Assess whether the lower-level entity is meaningfully related to the upper-level entity",
		"Программа платежного хаба",
		"Единый платежный контур",
	):
		return `{"related": true, "confidence": 0.91, "relation": "contributes_to", "justification": "Нижняя инициатива прямо поддерживает верхнюю цель по evidence quotes."}`, true, nil

	default:
		return "", false, fmt.Errorf("unexpected regression fixture prompt")
	}
}

func regressionFixturePromptHash(req LLMRequest) string {
	return sha256Hash([]byte(fmt.Sprintf("%s|%s|%s|%f|%t", req.Model, req.System, req.User, req.Temperature, req.ReasoningFallback)))
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
