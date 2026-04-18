package security

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ScrubSecrets function
func TestScrubSecrets_AWSKey(t *testing.T) {
	text := "aws_access_key_id=AKIAIOSFODNN7EXAMPLE"
	scrubbed, counts, err := ScrubSecrets(text)
	require.NoError(t, err)
	assert.Contains(t, scrubbed, "[REDACTED_aws_key]")
	assert.NotContains(t, scrubbed, "AKIAIOSFODNN7EXAMPLE")
	assert.Greater(t, counts["aws_key"], 0)
}

func TestScrubSecrets_GitHubToken(t *testing.T) {
	text := "GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	scrubbed, counts, err := ScrubSecrets(text)
	require.NoError(t, err)
	assert.Contains(t, scrubbed, "[REDACTED_github_token]")
	assert.NotContains(t, scrubbed, "ghp_")
	assert.Greater(t, counts["github_token"], 0)
}

func TestScrubSecrets_PrivateKey(t *testing.T) {
	text := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7X\n-----END RSA PRIVATE KEY-----"
	scrubbed, counts, err := ScrubSecrets(text)
	require.NoError(t, err)
	assert.Contains(t, scrubbed, "[REDACTED_private_key]")
	assert.NotContains(t, scrubbed, "-----BEGIN RSA PRIVATE KEY-----")
	assert.Greater(t, counts["private_key"], 0)
}

func TestScrubSecrets_MultipleSecrets(t *testing.T) {
	text := `
		aws_key: AKIAIOSFODNN7EXAMPLE
		github: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
		private: -----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQ\n-----END PRIVATE KEY-----
	`
	scrubbed, counts, err := ScrubSecrets(text)
	require.NoError(t, err)
	assert.Contains(t, scrubbed, "[REDACTED_aws_key]")
	assert.Contains(t, scrubbed, "[REDACTED_github_token]")
	assert.Contains(t, scrubbed, "[REDACTED_private_key]")
	assert.Greater(t, counts["aws_key"], 0)
	assert.Greater(t, counts["github_token"], 0)
	assert.Greater(t, counts["private_key"], 0)
}

func TestScrubSecrets_NoFalsePositives(t *testing.T) {
	text := `
		function main() {
			fmt.Println("hello")
			const sk = "not a secret"
			// This is a comment about security
			var password string // password field
		}
	`
	scrubbed, counts, err := ScrubSecrets(text)
	require.NoError(t, err)
	// Should not redact anything
	assert.Equal(t, 0, len(counts))
	assert.Equal(t, text, scrubbed)
}

// Test ScrubSecretsJSON function
func TestScrubSecretsJSON_ScrubsValuesNotKeys(t *testing.T) {
	input := map[string]interface{}{
		"aws_key": "AKIAIOSFODNN7EXAMPLE",
		"token":   "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
		"config": map[string]interface{}{
			"api_key": "sk-abcdefghijklmnopqrstuvwyxz123456789",
		},
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	scrubbed, counts, err := ScrubSecretsJSON(jsonBytes)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(scrubbed, &result)
	require.NoError(t, err)

	// Keys should be preserved
	assert.Contains(t, result, "aws_key")
	assert.Contains(t, result, "token")
	assert.Contains(t, result, "config")

	// Values should be scrubbed
	assert.Contains(t, result["aws_key"], "[REDACTED_aws_key]")
	assert.Contains(t, result["token"], "[REDACTED_github_token]")

	config := result["config"].(map[string]interface{})
	assert.Contains(t, config["api_key"], "[REDACTED_openai_key]")

	// Check counts
	assert.Greater(t, counts["aws_key"], 0)
	assert.Greater(t, counts["github_token"], 0)
	assert.Greater(t, counts["openai_key"], 0)
}

func TestScrubSecretsJSON_PreservesNonStringValues(t *testing.T) {
	input := map[string]interface{}{
		"count": 42,
		"flag":  true,
		"list":  []interface{}{"item1", "item2"},
		"secret": "AKIAIOSFODNN7EXAMPLE",
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	scrubbed, _, err := ScrubSecretsJSON(jsonBytes)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(scrubbed, &result)
	require.NoError(t, err)

	assert.Equal(t, 42, result["count"])
	assert.Equal(t, true, result["flag"])
	assert.Contains(t, result["secret"], "[REDACTED_aws_key]")
}

// Test SecurityFilter methods
func TestSecurityFilter_ScanForSecrets(t *testing.T) {
	sf := NewSecurityFilter()

	text := `
		aws: AKIAIOSFODNN7EXAMPLE
		github: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij
		private: -----BEGIN RSA PRIVATE KEY-----\ndata\n-----END RSA PRIVATE KEY-----
	`

	matches := sf.ScanForSecrets(text)

	assert.GreaterOrEqual(t, len(matches), 3)

	// Check for AWS key
	foundAWS := false
	for _, m := range matches {
		if m.Type == SecretTypeAWSKey {
			foundAWS = true
			break
		}
	}
	assert.True(t, foundAWS, "should find AWS key")
}

func TestSecurityFilter_ContainsSecrets(t *testing.T) {
	sf := NewSecurityFilter()

	assert.True(t, sf.ContainsSecrets("AKIAIOSFODNN7EXAMPLE"))
	assert.False(t, sf.ContainsSecrets("normal code"))
}

func TestSecurityFilter_GetSecretTypes(t *testing.T) {
	sf := NewSecurityFilter()

	text := "AKIAIOSFODNN7EXAMPLE ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	types := sf.GetSecretTypes(text)

	assert.GreaterOrEqual(t, len(types), 2)
}

func TestSecurityFilter_AddPattern(t *testing.T) {
	sf := NewSecurityFilter()

	err := sf.AddPattern(`CUSTOM_[A-Z0-9]+`, SecretType("custom"))
	require.NoError(t, err)

	assert.True(t, sf.ContainsSecrets("CUSTOM_ABC123"))
}

func TestRedactionFormat(t *testing.T) {
	format := RedactionFormat(SecretTypeAWSKey)
	assert.Equal(t, "[REDACTED_aws_key]", format)
}

func TestIsRedaction(t *testing.T) {
	assert.True(t, IsRedaction("[REDACTED_aws_key]"))
	assert.False(t, IsRedaction("AKIAIOSFODNN7EXAMPLE"))
}

func TestExtractRedactionType(t *testing.T) {
	typ, ok := ExtractRedactionType("[REDACTED_aws_key]")
	assert.True(t, ok)
	assert.Equal(t, SecretTypeAWSKey, typ)

	typ, ok = ExtractRedactionType("not a redaction")
	assert.False(t, ok)
	assert.Equal(t, SecretType(""), typ)
}

// Test HighEntropyCheck
func TestHighEntropyCheck_FlagsHighEntropy(t *testing.T) {
	// A string that should be flagged as high entropy
	highEntropy := "aB3$xY7!mN9@qR2#kL5%pW8&jH4^fG6*dS0!zX1"

	assert.True(t, HighEntropyCheck(highEntropy, "test"))
}

func TestHighEntropyCheck_AllowsUUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	assert.False(t, HighEntropyCheck(uuid, "test"))
}

func TestHighEntropyCheck_AllowsSHA256(t *testing.T) {
	sha256 := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	assert.False(t, HighEntropyCheck(sha256, "test"))
}

func TestHighEntropyCheck_AllowsSHA512(t *testing.T) {
	sha512 := "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff"
	assert.False(t, HighEntropyCheck(sha512, "test"))
}

func TestHighEntropyCheck_RejectsShortStrings(t *testing.T) {
	short := "short"
	assert.False(t, HighEntropyCheck(short, "test"))
}

func TestShannonEntropy(t *testing.T) {
	// Test with known entropy values
	// All same character -> entropy 0
	entropy := shannonEntropy("aaaa")
	assert.Equal(t, 0.0, entropy)

	// Two characters equally likely -> entropy 1
	entropy = shannonEntropy("ab")
	assert.InDelta(t, 1.0, entropy, 0.01)

	// Higher entropy for more diverse strings
	entropy1 := shannonEntropy("abc")
	entropy2 := shannonEntropy("abcdefghijklmnopqrstuvwxyz")
	assert.Greater(t, entropy2, entropy1)
}

func TestEstimateCharsetSize(t *testing.T) {
	size := EstimateCharsetSize("abcabc")
	assert.Equal(t, 3, size)

	size = EstimateCharsetSize("a1! a1!")
	assert.Equal(t, 3, size)
}

func TestNormalizeEntropy(t *testing.T) {
	// Test normalization
	s := "aB3$xY7!mN9@qR2#kL5%pW8&jH4^fG6*dS0!zX1"
	entropy := shannonEntropy(s)
	normalized := NormalizeEntropy(s, entropy)

	assert.GreaterOrEqual(t, normalized, 0.0)
	assert.LessOrEqual(t, normalized, 1.0)
}
