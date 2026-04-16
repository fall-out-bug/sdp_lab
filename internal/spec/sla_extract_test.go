package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSLA_Timeouts(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata")
	require.NoError(t, err)
	assert.NotEmpty(t, sla.Timeouts, "should find timeout patterns")
	for _, p := range sla.Timeouts {
		assert.Equal(t, "timeout", p.Category)
		assert.NotEmpty(t, p.Location)
	}
}

func TestExtractSLA_RateLimits(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata")
	require.NoError(t, err)
	assert.NotEmpty(t, sla.RateLimits, "should find rate limit patterns")
	for _, p := range sla.RateLimits {
		assert.Equal(t, "rate_limit", p.Category)
		assert.Contains(t, p.Value, "rate=")
	}
}

func TestExtractSLA_Retries(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata")
	require.NoError(t, err)
	assert.NotEmpty(t, sla.Retries, "should find retry patterns")
	for _, p := range sla.Retries {
		assert.Equal(t, "retry", p.Category)
	}
}

func TestExtractSLA_StructTimeout(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata")
	require.NoError(t, err)
	var found bool
	for _, p := range sla.Timeouts {
		if p.Component == "Timeout" {
			found = true
		}
	}
	assert.True(t, found, "should find struct literal timeout (http.Client.Timeout)")
}

func TestExtractSLA_Total(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata")
	require.NoError(t, err)
	expected := len(sla.Timeouts) + len(sla.Retries) + len(sla.RateLimits) +
		len(sla.CircuitBreakers) + len(sla.ResourcePools) + len(sla.HealthChecks)
	assert.Equal(t, expected, sla.Total)
}

func TestExtractSLA_BadDir(t *testing.T) {
	sla, err := ExtractSLAParameters("testdata/nonexistent_xyz")
	require.NoError(t, err)
	assert.Equal(t, 0, sla.Total)
}
