package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAPIContracts_FromTestdata(t *testing.T) {
	contracts, err := ExtractAPIContracts("testdata")
	assert.NoError(t, err)
	assert.NotNil(t, contracts)

	// Should find routes from at least some testdata files
	assert.Greater(t, contracts.Total, 0, "should find at least one endpoint")
}

func TestExtractAPIContracts_NonexistentDir(t *testing.T) {
	contracts, err := ExtractAPIContracts("testdata/nonexistent_dir")
	assert.NoError(t, err)
	assert.Equal(t, 0, contracts.Total)
}

func TestExtractAPIContracts_EndpointHasFields(t *testing.T) {
	contracts, err := ExtractAPIContracts("testdata")
	assert.NoError(t, err)
	if contracts.Total > 0 {
		ep := contracts.HTTPEndpoints[0]
		assert.NotEmpty(t, ep.Method, "endpoint should have a method")
		assert.NotEmpty(t, ep.Path, "endpoint should have a path")
		assert.NotEmpty(t, ep.SourceFile, "endpoint should have source file")
	}
}
