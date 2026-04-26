package strataudit

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngest_BasicFiles(t *testing.T) {
	// Create temp project structure
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	// Write test files
	os.WriteFile(filepath.Join(stratDir, "vision-statement.md"), []byte("# Our Vision\nBecome the market leader in AI tools"), 0644) //nolint:errcheck
	os.WriteFile(filepath.Join(stratDir, "strategy-2026.md"), []byte("# Strategy\nExpand into Southeast Asian markets"), 0644)       //nolint:errcheck
	os.WriteFile(filepath.Join(stratDir, "task-backlog.md"), []byte("# Tasks\n- Hire country manager\n- Set up office"), 0644)       //nolint:errcheck
	os.WriteFile(filepath.Join(stratDir, "random.tmp"), []byte("temp file"), 0644)                                                   //nolint:errcheck

	cfg := &Config{
		Project: ProjectConfig{
			SourceDirs: []string{stratDir},
			Exclude:    []string{"*.tmp"},
		},
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}},
			{Name: "strategy", Rank: 1, Patterns: []string{"*strategy*"}},
			{Name: "task", Rank: 2, Patterns: []string{"*task*"}},
		},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close() //nolint:errcheck

	result, err := Ingest(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	if result.New != 3 {
		t.Errorf("New = %d, want 3", result.New)
	}
	if result.Unchanged != 0 {
		t.Errorf("Unchanged = %d, want 0", result.Unchanged)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors: %v", result.Errors)
	}

	// Verify levels saved
	levels, _ := store.LoadLevels(context.Background())
	if len(levels) != 3 {
		t.Fatalf("levels: got %d, want 3", len(levels))
	}

	// Verify documents stored
	doc, err := store.DocumentByPath(context.Background(), filepath.Join(stratDir, "vision-statement.md"))
	if err != nil {
		t.Fatalf("DocumentByPath: %v", err)
	}
	if doc == nil {
		t.Fatal("document not found")
	}
	if doc.LevelID != "vision" {
		t.Errorf("LevelID = %q, want vision", doc.LevelID)
	}
	if doc.ContentHash == "" {
		t.Error("ContentHash is empty")
	}
}

func TestIngest_Deduplication(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	os.WriteFile(filepath.Join(stratDir, "vision.md"), []byte("Be the leader"), 0644) //nolint:errcheck

	cfg := &Config{
		Project:    ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:     []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	dbPath := filepath.Join(dir, "test.db")
	store, _ := NewSQLiteStore(dbPath)
	defer store.Close() //nolint:errcheck

	// First run
	r1, _ := Ingest(context.Background(), cfg, store)
	if r1.New != 1 {
		t.Fatalf("first run: New = %d, want 1", r1.New)
	}

	// Second run — same content
	r2, _ := Ingest(context.Background(), cfg, store)
	if r2.Unchanged != 1 {
		t.Fatalf("second run: Unchanged = %d, want 1", r2.Unchanged)
	}
	if r2.New != 0 {
		t.Fatalf("second run: New = %d, want 0", r2.New)
	}
}

func TestIngest_UpdateTriggersVersionBump(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	visionPath := filepath.Join(stratDir, "vision.md")
	os.WriteFile(visionPath, []byte("Version 1"), 0644) //nolint:errcheck

	cfg := &Config{
		Project:    ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:     []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	dbPath := filepath.Join(dir, "test.db")
	store, _ := NewSQLiteStore(dbPath)
	defer store.Close() //nolint:errcheck

	// First run
	// First run
	_, _ = Ingest(context.Background(), cfg, store)

	// Modify file
	os.WriteFile(visionPath, []byte("Version 2 updated"), 0644) //nolint:errcheck

	// Second run
	r2, _ := Ingest(context.Background(), cfg, store)
	if r2.Updated != 1 {
		t.Fatalf("updated run: Updated = %d, want 1", r2.Updated)
	}

	doc, _ := store.DocumentByPath(context.Background(), visionPath)
	if doc.Version != 2 {
		t.Errorf("Version = %d, want 2", doc.Version)
	}
	if doc.Content != "Version 2 updated" {
		t.Errorf("Content = %q, want updated content", doc.Content)
	}
}

func TestIngest_PersistsSectionsWithOffsets(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	visionPath := filepath.Join(stratDir, "vision.md")
	content := "Наша стратегия — лидерство в платежах. " +
		"Мы усиливаем продукт, каналы и международную экспансию. " +
		"Это требует отдельного ownership и KPI."
	os.WriteFile(visionPath, []byte(content), 0644) //nolint:errcheck

	cfg := &Config{
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    8,
			ChunkOverlapTokens: 2,
		},
	}

	store := setupTestStore(t)
	if _, err := Ingest(context.Background(), cfg, store); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	doc, err := store.DocumentByPath(context.Background(), visionPath)
	if err != nil {
		t.Fatalf("DocumentByPath: %v", err)
	}
	sections, err := store.SectionsByDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("SectionsByDocument: %v", err)
	}
	if len(sections) < 2 {
		t.Fatalf("len(sections) = %d, want >= 2", len(sections))
	}
	if sections[0].DocumentID != doc.ID {
		t.Fatalf("sections[0].DocumentID = %q, want %q", sections[0].DocumentID, doc.ID)
	}
	if sections[0].CharStart != 0 {
		t.Fatalf("sections[0].CharStart = %d, want 0", sections[0].CharStart)
	}
	if sections[0].CharEnd <= sections[0].CharStart {
		t.Fatalf("invalid first section offsets: %+v", sections[0])
	}
	if sections[0].Preview == "" {
		t.Fatal("expected section preview to be populated")
	}
	if len(sections[0].QualityFlags) == 0 || sections[0].QualityFlags[0] != "section_parse_fallback" {
		t.Fatalf("unexpected quality flags: %+v", sections[0].QualityFlags)
	}
}

func TestIngest_UpdateReplacesOldSections(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	visionPath := filepath.Join(stratDir, "vision.md")
	os.WriteFile(visionPath, []byte("Первая версия стратегии с длинным хвостом для нескольких чанков."), 0644) //nolint:errcheck

	cfg := &Config{
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    6,
			ChunkOverlapTokens: 1,
		},
	}

	store := setupTestStore(t)
	if _, err := Ingest(context.Background(), cfg, store); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	doc, err := store.DocumentByPath(context.Background(), visionPath)
	if err != nil {
		t.Fatalf("DocumentByPath: %v", err)
	}
	firstSections, err := store.SectionsByDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("SectionsByDocument first: %v", err)
	}
	if len(firstSections) == 0 {
		t.Fatal("expected initial sections")
	}

	os.WriteFile(visionPath, []byte("Короткая вторая версия."), 0644) //nolint:errcheck
	if _, err := Ingest(context.Background(), cfg, store); err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	updatedDoc, err := store.DocumentByPath(context.Background(), visionPath)
	if err != nil {
		t.Fatalf("DocumentByPath updated: %v", err)
	}
	updatedSections, err := store.SectionsByDocument(context.Background(), updatedDoc.ID)
	if err != nil {
		t.Fatalf("SectionsByDocument updated: %v", err)
	}
	if len(updatedSections) != 1 {
		t.Fatalf("len(updatedSections) = %d, want 1", len(updatedSections))
	}
	if updatedSections[0].Content != "Короткая вторая версия." {
		t.Fatalf("unexpected updated section content: %q", updatedSections[0].Content)
	}
}

func TestClassifyLevel(t *testing.T) {
	levels := []LevelConfig{
		{Name: "vision", Rank: 0, Patterns: []string{"*vision*", "*mission*"}},
		{Name: "strategy", Rank: 1, Patterns: []string{"*strategy*"}},
	}

	tests := []struct {
		path string
		want string
	}{
		{"strategy/our-vision.md", "vision"},
		{"strategy/mission-statement.txt", "vision"},
		{"plans/strategy-2026.md", "strategy"},
		{"plans/random.md", ""},
	}
	for _, tt := range tests {
		levelMatchers := buildLevelMatchers(levels)
		got := classifyLevelOptimized(tt.path, levelMatchers)
		if got != tt.want {
			t.Errorf("classifyLevel(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestClassifyLevel_Deterministic_LowestRankWins(t *testing.T) {
	levels := []LevelConfig{
		{Name: "strategy", Rank: 0, Patterns: []string{"*стратег*"}},
		{Name: "architecture", Rank: 1, Patterns: []string{"*HLD*", "*Архитекту*"}},
		{Name: "design", Rank: 2, Patterns: []string{"*LLD*", "*ТЗ*"}},
		{Name: "implementation", Rank: 3, Patterns: []string{"*API*", "*Подключ*", "*Платеж*СКМ*", "*Смена*счета*"}},
	}

	tests := []struct {
		path string
		want string
	}{
		// Implementation patterns
		{"API+→+Digital+back+→+Платежи.doc", "implementation"},
		{"Подключение+СБП.doc", "implementation"},
		{"Платежи+СКМ.doc", "implementation"},
		{"Смена+основного+счета.doc", "implementation"},
		// Design patterns
		{"LLD-MBK-106+Online+платежи.doc", "design"},
		{"ТЗ+для+МП.doc", "design"},
		// Architecture patterns
		{"Архитектура+решения.doc", "architecture"},
		{"[HLD-DPT-006]+Накопительный.doc", "architecture"},
		// Strategy
		{"Стратегия+развития.doc", "strategy"},
		// Unmatched
		{"random.doc", ""},
		// Multi-match: LLD file with API keyword → design wins (lower rank)
		{"LLD-MBK-132+API+интеграция.doc", "design"},
	}

	for _, tt := range tests {
		levelMatchers := buildLevelMatchers(levels)
		got := classifyLevelOptimized(tt.path, levelMatchers)
		if got != tt.want {
			t.Errorf("classifyLevel(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}

	// Determinism: run 10 times, must get same results
	levelMatchers := buildLevelMatchers(levels)
	for i := 0; i < 10; i++ {
		for _, tt := range tests {
			got := classifyLevelOptimized(tt.path, levelMatchers)
			if got != tt.want {
				t.Fatalf("determinism check failed on iteration %d: classifyLevel(%q) = %q, want %q", i, tt.path, got, tt.want)
			}
		}
	}
}

func TestChunkContent(t *testing.T) {
	content := "Hello world. "            // 13 chars, ~3 tokens
	chunks := ChunkContent(content, 2, 1) // 2 tokens ≈ 8 chars
	if len(chunks) < 1 {
		t.Fatal("expected at least 1 chunk")
	}

	// Small content should not be chunked
	single := ChunkContent("short", 3000, 500)
	if len(single) != 1 {
		t.Errorf("short content: got %d chunks, want 1", len(single))
	}

	// Long content should be chunked
	longContent := ""
	for i := 0; i < 1000; i++ {
		longContent += "This is a test sentence for chunking. "
	}
	chunks = ChunkContent(longContent, 1000, 200)
	if len(chunks) <= 1 {
		t.Errorf("long content: got %d chunks, want >1", len(chunks))
	}
}

func TestIsSupportedExt(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"doc.md", true},
		{"doc.txt", true},
		{"doc.pdf", true},
		{"doc.docx", true},
		{"deck.pptx", true},
		{"doc.html", false},
		{"doc.xlsx", false},
	}
	for _, tt := range tests {
		got := isSupportedExt(tt.path)
		if got != tt.want {
			t.Errorf("isSupportedExt(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIngest_ExcludeMatchesNestedDirectorySegments(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "corpus")
	nested := filepath.Join(root, "Downloads", "nested")
	keep := filepath.Join(root, "strategy")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keep, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(nested, "strategy.md"), []byte("must be excluded"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keep, "strategy.md"), []byte("must stay"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Project: ProjectConfig{
			SourceDirs: []string{root},
			Exclude:    []string{"Downloads"},
		},
		Levels:     []LevelConfig{{Name: "strategy", Rank: 0, Patterns: []string{"*strategy*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	store := setupTestStore(t)
	result, err := Ingest(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.New != 1 {
		t.Fatalf("result.New = %d, want 1", result.New)
	}

	excludedDoc, err := store.DocumentByPath(context.Background(), filepath.Join(nested, "strategy.md"))
	if err != nil {
		t.Fatalf("DocumentByPath excluded: %v", err)
	}
	if excludedDoc != nil {
		t.Fatalf("expected nested Downloads document to be excluded, got %+v", excludedDoc)
	}
}

func TestIngest_NativePPTXExtraction(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}

	pptxPath := filepath.Join(root, "Стратегия-команды.pptx")
	if err := os.WriteFile(pptxPath, buildTestPPTX(t, map[string]string{
		"ppt/slides/slide1.xml":           pptxTextXML("Технологическая стратегия", "Ключевая инициатива"),
		"ppt/notesSlides/notesSlide1.xml": pptxTextXML("Детали в notes"),
	}), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Project:    ProjectConfig{SourceDirs: []string{root}},
		Levels:     []LevelConfig{{Name: "strategy", Rank: 0, Patterns: []string{"*стратег*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	store := setupTestStore(t)
	result, err := Ingest(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.New != 1 {
		t.Fatalf("result.New = %d, want 1", result.New)
	}

	doc, err := store.DocumentByPath(context.Background(), pptxPath)
	if err != nil {
		t.Fatalf("DocumentByPath: %v", err)
	}
	if doc == nil {
		t.Fatal("expected pptx document to be stored")
	}
	if !strings.Contains(doc.Content, "Технологическая стратегия") {
		t.Fatalf("pptx content missing slide text: %q", doc.Content)
	}
	if !strings.Contains(doc.Content, "Детали в notes") {
		t.Fatalf("pptx content missing notes text: %q", doc.Content)
	}
}

func buildTestPPTX(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func pptxTextXML(values ...string) string {
	var b strings.Builder
	b.WriteString(`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree>`)
	for _, value := range values {
		b.WriteString(`<p:sp><p:txBody><a:p><a:r><a:t>`)
		b.WriteString(value)
		b.WriteString(`</a:t></a:r></a:p></p:txBody></p:sp>`)
	}
	b.WriteString(`</p:spTree></p:cSld></p:sld>`)
	return b.String()
}

func TestSHA256Hash(t *testing.T) {
	h1 := sha256Hash([]byte("hello"))
	h2 := sha256Hash([]byte("hello"))
	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	h3 := sha256Hash([]byte("world"))
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestIngest_SkipEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	os.WriteFile(filepath.Join(stratDir, "vision.md"), []byte("   \n\n  "), 0644) //nolint:errcheck

	cfg := &Config{
		Project:    ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:     []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	dbPath := filepath.Join(dir, "test.db")
	store, _ := NewSQLiteStore(dbPath)
	defer store.Close() //nolint:errcheck

	result, _ := Ingest(context.Background(), cfg, store)
	if result.New != 0 {
		t.Errorf("empty file: New = %d, want 0", result.New)
	}
}

func TestIngest_SkipNoLevelMatch(t *testing.T) {
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	os.WriteFile(filepath.Join(stratDir, "random.md"), []byte("Some content"), 0644) //nolint:errcheck

	cfg := &Config{
		Project:    ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:     []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
		Thresholds: ThresholdConfig{ChunkTokenLimit: 3000},
	}

	dbPath := filepath.Join(dir, "test.db")
	store, _ := NewSQLiteStore(dbPath)
	defer store.Close() //nolint:errcheck

	result, _ := Ingest(context.Background(), cfg, store)
	if result.New != 0 {
		t.Errorf("no match: New = %d, want 0", result.New)
	}
}
