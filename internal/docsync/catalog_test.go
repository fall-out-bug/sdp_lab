package docsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCatalogExists verifies the catalog file exists
func TestCatalogExists(t *testing.T) {
	catalogPath := filepath.Join("..", "..", ".agents", "skills", "index.json")
	_, err := os.Stat(catalogPath)
	require.NoError(t, err, "Catalog file should exist at %s", catalogPath)
}

// TestCatalogValidJSON verifies the catalog is valid JSON
func TestCatalogValidJSON(t *testing.T) {
	catalogPath := filepath.Join("..", "..", ".agents", "skills", "index.json")
	data, err := os.ReadFile(catalogPath)
	require.NoError(t, err, "Should be able to read catalog file")

	var catalog map[string]interface{}
	err = json.Unmarshal(data, &catalog)
	require.NoError(t, err, "Catalog should be valid JSON")
}

// TestCatalogHasRequiredFields verifies the catalog has required top-level fields
func TestCatalogHasRequiredFields(t *testing.T) {
	catalogPath := filepath.Join("..", "..", ".agents", "skills", "index.json")
	data, err := os.ReadFile(catalogPath)
	require.NoError(t, err)

	var catalog map[string]interface{}
	err = json.Unmarshal(data, &catalog)
	require.NoError(t, err)

	// Check required fields
	assert.Contains(t, catalog, "version", "Catalog should have version")
	assert.Contains(t, catalog, "generated", "Catalog should have generated date")
	assert.Contains(t, catalog, "skills", "Catalog should have skills array")
	assert.Contains(t, catalog, "deprecated", "Catalog should have deprecated array")
	assert.Contains(t, catalog, "removed", "Catalog should have removed array")
	assert.Contains(t, catalog, "summary", "Catalog should have summary")
}

// TestCatalogActiveSkills verifies active skills count and structure
func TestCatalogActiveSkills(t *testing.T) {
	catalogPath := filepath.Join("..", "..", ".agents", "skills", "index.json")
	data, err := os.ReadFile(catalogPath)
	require.NoError(t, err)

	var catalog struct {
		Skills []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Path   string `json:"path"`
		} `json:"skills"`
	}
	err = json.Unmarshal(data, &catalog)
	require.NoError(t, err)

	activeSkills := 0
	for _, skill := range catalog.Skills {
		if skill.Status == "active" {
			activeSkills++
			// Verify skill has required fields
			assert.NotEmpty(t, skill.Name, "Active skill should have name")
			assert.NotEmpty(t, skill.Path, "Active skill should have path")
		}
	}

	// Should have at least 5 active F125 intents + specialized skills
	assert.GreaterOrEqual(t, activeSkills, 10, "Should have at least 10 active skills")
}

// TestInventoryDocumentExists verifies the inventory document exists
func TestInventoryDocumentExists(t *testing.T) {
	inventoryPath := filepath.Join("..", "..", "docs", "reference", "skill-catalog-inventory.md")
	_, err := os.Stat(inventoryPath)
	require.NoError(t, err, "Inventory document should exist at %s", inventoryPath)
}

// TestCatalogLintScriptExists verifies the lint script exists
func TestCatalogLintScriptExists(t *testing.T) {
	lintPath := filepath.Join("catalog_lint.sh")
	_, err := os.Stat(lintPath)
	require.NoError(t, err, "Catalog lint script should exist at %s", lintPath)
}
