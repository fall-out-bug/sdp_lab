package strataudit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIngest_BasicFiles(t *testing.T) {
	// Create temp project structure
	dir := t.TempDir()
	stratDir := filepath.Join(dir, "strategy")
	os.MkdirAll(stratDir, 0755) //nolint:errcheck

	// Write test files
		os.WriteFile(filepath.Join(stratDir, "vision-statement.md"), []byte("# Our Vision\nBecome the market leader in AI tools"), 0644) //nolint:errcheck
		os.WriteFile(filepath.Join(stratDir, "strategy-2026.md"), []byte("# Strategy\nExpand into Southeast Asian markets"), 0644) //nolint:errcheck
		os.WriteFile(filepath.Join(stratDir, "task-backlog.md"), []byte("# Tasks\n- Hire country manager\n- Set up office"), 0644) //nolint:errcheck
		os.WriteFile(filepath.Join(stratDir, "random.tmp"), []byte("temp file"), 0644) //nolint:errcheck

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
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
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
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
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

func TestClassifyLevel(t *testing.T) {
	levelMap := map[string]LevelConfig{
		"vision":   {Name: "vision", Rank: 0, Patterns: []string{"*vision*", "*mission*"}},
		"strategy": {Name: "strategy", Rank: 1, Patterns: []string{"*strategy*"}},
	}

	tests := []struct {
		path    string
		want    string
	}{
		{"strategy/our-vision.md", "vision"},
		{"strategy/mission-statement.txt", "vision"},
		{"plans/strategy-2026.md", "strategy"},
		{"plans/random.md", ""},
	}
	for _, tt := range tests {
		got := classifyLevel(tt.path, levelMap)
		if got != tt.want {
			t.Errorf("classifyLevel(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestChunkContent(t *testing.T) {
	content := "Hello world. " // 13 chars, ~3 tokens
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
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
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
		Project: ProjectConfig{SourceDirs: []string{stratDir}},
		Levels:  []LevelConfig{{Name: "vision", Rank: 0, Patterns: []string{"*vision*"}}},
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
