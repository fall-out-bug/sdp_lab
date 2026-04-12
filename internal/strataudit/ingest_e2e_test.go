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
	configPath := os.Getenv("STRATAUDIT_E2E_CONFIG")
	if dataDir == "" || configPath == "" {
		t.Skip("set STRATAUDIT_E2E_DIR and STRATAUDIT_E2E_CONFIG to enable e2e ingest")
	}
	if _, err := os.Stat(dataDir); err != nil {
		t.Skip("data directory not available")
	}

	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Skip("config not found")
	}

	var cfg Config
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.setDefaults()

	outDir := filepath.Join("/tmp/strataudit-v11-test", cfg.Output.Dir)
	os.MkdirAll(outDir, 0755)

	dbPath := filepath.Join(outDir, "strataudit.db")
	os.Remove(dbPath)

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
