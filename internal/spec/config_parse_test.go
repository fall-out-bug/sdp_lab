package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractConfig_YAML(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	assert.NotEmpty(t, params)
}

func TestExtractConfig_Timeouts(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	var found int
	for _, p := range params {
		if p.Category == "timeout" {
			found++
			assert.True(t, p.Configurable)
		}
	}
	assert.Greater(t, found, 0, "should find timeout params in config")
}

func TestExtractConfig_Retries(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	var found bool
	for _, p := range params {
		if p.Category == "retry" {
			found = true
		}
	}
	assert.True(t, found, "should find retry params in config")
}

func TestExtractConfig_ResourcePools(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	var found bool
	for _, p := range params {
		if p.Category == "resource_pool" {
			found = true
		}
	}
	assert.True(t, found, "should find pool/conn params in config")
}

func TestExtractConfig_FeatureFlags(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	var found bool
	for _, p := range params {
		if p.Category == "feature_flag" {
			found = true
		}
	}
	assert.True(t, found, "should find feature flags in config")
}

func TestExtractConfig_SecretRedaction(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	for _, p := range params {
		if p.Category == "secret" {
			assert.Equal(t, "[REDACTED]", p.Value, "secret values must be redacted")
		}
	}
	// Verify known secrets are found
	var secrets []string
	for _, p := range params {
		if p.Category == "secret" {
			secrets = append(secrets, p.Component)
		}
	}
	assert.Contains(t, secrets, "auth.api_key")
	assert.Contains(t, secrets, "auth.token")
	assert.Contains(t, secrets, "auth.secret")
	assert.Contains(t, secrets, "database.database_url")
}

func TestExtractConfig_JSON(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/configs")
	require.NoError(t, err)
	// JSON file should contribute params too
	var jsonFound bool
	for _, p := range params {
		if p.Location == "app.json" {
			jsonFound = true
			break
		}
	}
	assert.True(t, jsonFound, "should parse JSON config files")
}

func TestExtractConfig_BadDir(t *testing.T) {
	params, err := ExtractConfigParameters("testdata/nonexistent_xyz")
	require.NoError(t, err)
	assert.Empty(t, params)
}

func TestIsSecretKey(t *testing.T) {
	tests := []struct{ key string; secret bool }{
		{"password", true},
		{"api_key", true},
		{"token", true},
		{"secret", true},
		{"database_url", true},
		{"auth_token", true},
		{"timeout", false},
		{"max_connections", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.secret, isSecretKey(tt.key), "isSecretKey(%q)", tt.key)
	}
}
