package strataudit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIngest_E2E(t *testing.T) {
	dataDir := os.Getenv("STRATAUDIT_E2E_DIR")
	if dataDir == "" {
		t.Skip("STRATAUDIT_E2E_DIR not set")
	}

	if _, err := os.Stat(dataDir); err != nil {
		t.Skipf("data directory not available: %v", err)
	}

	configPath := os.Getenv("STRATAUDIT_E2E_CONFIG")
	if configPath == "" {
		t.Skip("STRATAUDIT_E2E_CONFIG not set")
	}
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Skipf("config not found: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.setDefaults()

	outRoot := os.Getenv("STRATAUDIT_E2E_OUT")
	if outRoot == "" {
		outRoot = t.TempDir()
	}

	outDir := filepath.Join(outRoot, cfg.Output.Dir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}

	dbPath := filepath.Join(outDir, "strataudit.db")
	_ = os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() {
		store.Close()
		_ = os.Remove(dbPath)
	}()

	result, err := Ingest(context.Background(), &cfg, store)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	t.Logf("Ingest: %d new, %d updated, %d unchanged, %d errors",
		result.New, result.Updated, result.Unchanged, len(result.Errors))

	if result.New == 0 && result.Updated == 0 {
		t.Error("no documents ingested")
	}

	// Check errors are not too many (< 20% of total)
	total := result.New + result.Updated + result.Unchanged
	if total > 0 && len(result.Errors) > total/5 {
		for _, e := range result.Errors {
			t.Logf("  ERR: %v", e)
		}
		t.Errorf("too many errors: %d/%d", len(result.Errors), total)
	}
}

func TestIngest_RegressionFixture(t *testing.T) {
	cfg, _ := loadRegressionFixtureConfig(t)
	store := newRegressionStore(t, cfg)

	result, err := Ingest(context.Background(), cfg, store)
	if err != nil {
		t.Fatalf("trust guarantee violated: regression fixture ingest failed: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("trust guarantee violated: regression fixture ingest produced errors: %+v", result.Errors)
	}
	if result.New != 3 {
		t.Fatalf("trust guarantee violated: regression fixture should ingest 3 synthetic documents, got new=%d updated=%d unchanged=%d", result.New, result.Updated, result.Unchanged)
	}

	docs, err := store.AllDocuments(context.Background())
	if err != nil {
		t.Fatalf("AllDocuments: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("trust guarantee violated: expected 3 stored documents from regression fixture, got %d", len(docs))
	}

	expectedDocs := map[string]string{
		"company-vision.md":       "vision",
		"payment-strategy.md":     "strategy",
		"template-vision-note.md": "vision",
	}
	for _, doc := range docs {
		base := filepath.Base(doc.Path)
		expectedLevel, ok := expectedDocs[base]
		if !ok {
			t.Fatalf("trust guarantee violated: unexpected document in regression fixture ingest: %s", doc.Path)
		}
		if doc.LevelID != expectedLevel {
			t.Fatalf("trust guarantee violated: document %s classified as %s, want %s", doc.Path, doc.LevelID, expectedLevel)
		}
		sections, err := store.SectionsByDocument(context.Background(), doc.ID)
		if err != nil {
			t.Fatalf("SectionsByDocument(%s): %v", doc.ID, err)
		}
		if len(sections) == 0 {
			t.Fatalf("trust guarantee violated: document %s has no materialized sections", doc.Path)
		}
	}
}
