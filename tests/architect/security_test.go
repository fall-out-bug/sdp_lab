package architect_test

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ScanForSecrets
// ---------------------------------------------------------------------------

func TestScanForSecrets_AWSKey(t *testing.T) {
	sf := architect.NewSecurityFilter()
	content := "config: AKIAIOSFODNN7EXAMPLE rest"
	matches := sf.ScanForSecrets(content)

	require.Len(t, matches, 1)
	assert.Equal(t, "aws_key", matches[0].Type)
	assert.Equal(t, strings.Index(content, "AKIA"), matches[0].Position)
	assert.Equal(t, 20, matches[0].Length) // AKIA + 16 chars
}

func TestScanForSecrets_GitHubToken(t *testing.T) {
	sf := architect.NewSecurityFilter()
	token := "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	content := "token=" + token
	matches := sf.ScanForSecrets(content)

	require.Len(t, matches, 1)
	assert.Equal(t, "github_token", matches[0].Type)
	assert.Equal(t, strings.Index(content, "ghp_"), matches[0].Position)
	assert.Equal(t, 40, matches[0].Length) // ghp_ + 36 chars
}

func TestScanForSecrets_PrivateKey(t *testing.T) {
	sf := architect.NewSecurityFilter()
	content := "-----BEGIN RSA PRIVATE KEY-----\nMIIEow..."
	matches := sf.ScanForSecrets(content)

	require.Len(t, matches, 1)
	assert.Equal(t, "private_key", matches[0].Type)
	assert.Equal(t, 0, matches[0].Position)

	// Also test EC variant.
	contentEC := "-----BEGIN EC PRIVATE KEY-----\ndata"
	matchesEC := sf.ScanForSecrets(contentEC)
	require.Len(t, matchesEC, 1)
	assert.Equal(t, "private_key", matchesEC[0].Type)

	// Also test generic (no prefix).
	contentGeneric := "-----BEGIN PRIVATE KEY-----\ndata"
	matchesGeneric := sf.ScanForSecrets(contentGeneric)
	require.Len(t, matchesGeneric, 1)
	assert.Equal(t, "private_key", matchesGeneric[0].Type)
}

func TestScanForSecrets_OpenAIKey(t *testing.T) {
	sf := architect.NewSecurityFilter()
	key := "sk-" + strings.Repeat("a", 48)
	content := "openai_key=" + key
	matches := sf.ScanForSecrets(content)

	found := false
	for _, m := range matches {
		if m.Type == "openai_key" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find openai_key match")
}

func TestScanForSecrets_PasswordAssignment(t *testing.T) {
	sf := architect.NewSecurityFilter()
	content := `password="SuperSecret123!"`
	matches := sf.ScanForSecrets(content)

	found := false
	for _, m := range matches {
		if m.Type == "password_assignment" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find password_assignment match")
}

func TestScanForSecrets_NoFalsePositives(t *testing.T) {
	sf := architect.NewSecurityFilter()

	benign := []string{
		"func main() { fmt.Println(\"hello\") }",
		"import \"net/http\"",
		"// This is a comment about security",
		"var x = 42",
		"the password field is required",
		"use a private key for authentication",
		"skeleton key pattern",
	}

	for _, s := range benign {
		matches := sf.ScanForSecrets(s)
		assert.Empty(t, matches, "false positive in: %q", s)
	}
}

// ---------------------------------------------------------------------------
// Sanitize
// ---------------------------------------------------------------------------

func TestSanitize_RedactsSecrets(t *testing.T) {
	sf := architect.NewSecurityFilter()

	profile := &architect.CodebaseProfile{
		Name: "test-project",
		Files: map[string]string{
			"config.yaml": "aws_key: AKIAIOSFODNN7EXAMPLE\nother: value",
			"main.go":     "// normal code\nfunc main() {}",
		},
		Metadata: map[string]string{
			"token": "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
		},
		Summary: "Project with -----BEGIN RSA PRIVATE KEY----- embedded",
	}

	sanitized, secrets := sf.Sanitize(profile)

	// Secrets must be replaced.
	assert.Contains(t, sanitized.Files["config.yaml"], "[REDACTED_aws_key]")
	assert.NotContains(t, sanitized.Files["config.yaml"], "AKIAIOSFODNN7EXAMPLE")

	assert.Contains(t, sanitized.Metadata["token"], "[REDACTED_github_token]")
	assert.NotContains(t, sanitized.Metadata["token"], "ghp_")

	assert.Contains(t, sanitized.Summary, "[REDACTED_private_key]")
	assert.NotContains(t, sanitized.Summary, "-----BEGIN RSA PRIVATE KEY-----")

	// SecretsFound must report redactions.
	assert.True(t, secrets.Redacted)
	assert.Greater(t, secrets.Count, 0)

	// Non-secret content preserved.
	assert.Contains(t, sanitized.Files["config.yaml"], "other: value")
}

func TestSanitize_PreservesStructure(t *testing.T) {
	sf := architect.NewSecurityFilter()

	profile := &architect.CodebaseProfile{
		Name: "my-project",
		Files: map[string]string{
			"a.go": "package main",
			"b.go": "package main",
		},
		Metadata: map[string]string{
			"language": "go",
			"version":  "1.22",
		},
		Summary: "A simple project",
	}

	sanitized, _ := sf.Sanitize(profile)

	// Structure must be identical.
	assert.Equal(t, profile.Name, sanitized.Name)
	assert.Len(t, sanitized.Files, 2)
	assert.Len(t, sanitized.Metadata, 2)
	assert.Equal(t, "go", sanitized.Metadata["language"])
	assert.Equal(t, "1.22", sanitized.Metadata["version"])

	// Original must be unmodified.
	assert.Equal(t, "package main", profile.Files["a.go"])
}

func TestSanitize_ScrubsUserPaths(t *testing.T) {
	sf := architect.NewSecurityFilter()

	profile := &architect.CodebaseProfile{
		Name: "path-test",
		Files: map[string]string{
			"/Users/johndoe/projects/app/main.go": "source at /Users/johndoe/projects/app",
		},
		Metadata: map[string]string{
			"root": "/Users/janedoe/work/project",
		},
		Summary: "Built from /Users/admin/src/project",
	}

	sanitized, _ := sf.Sanitize(profile)

	// Check file content.
	for _, content := range sanitized.Files {
		assert.NotContains(t, content, "/Users/johndoe/")
		assert.Contains(t, content, "/Users/[REDACTED]/")
	}

	// Check file keys.
	for path := range sanitized.Files {
		assert.NotContains(t, path, "/Users/johndoe/")
		assert.Contains(t, path, "/Users/[REDACTED]/")
	}

	// Check metadata.
	assert.Contains(t, sanitized.Metadata["root"], "/Users/[REDACTED]/")
	assert.NotContains(t, sanitized.Metadata["root"], "/Users/janedoe/")

	// Check summary.
	assert.Contains(t, sanitized.Summary, "/Users/[REDACTED]/")
	assert.NotContains(t, sanitized.Summary, "/Users/admin/")
}

func TestSanitize_HashesInternalPackages(t *testing.T) {
	sf := architect.NewSecurityFilter()

	profile := &architect.CodebaseProfile{
		Name: "pkg-test",
		Files: map[string]string{
			"App.java": "import com.acme.internal.service.UserService;",
		},
		Summary: "Uses com.company.core.Engine",
	}

	sanitized, _ := sf.Sanitize(profile)

	// Internal packages must be hashed.
	assert.NotContains(t, sanitized.Files["App.java"], "com.acme")
	assert.Contains(t, sanitized.Files["App.java"], "pkg.")

	assert.NotContains(t, sanitized.Summary, "com.company")
	assert.Contains(t, sanitized.Summary, "pkg.")
}

// ---------------------------------------------------------------------------
// Default SecurityFilter
// ---------------------------------------------------------------------------

func TestDefaultSecurityFilter_BlocksExternalLLM(t *testing.T) {
	sf := architect.NewSecurityFilter()
	assert.False(t, sf.ExternalLLMAllowed(), "default SecurityFilter must block external LLM")
}

func TestSecurityFilter_AllowExternalLLM(t *testing.T) {
	sf := architect.NewSecurityFilter()
	sf.AllowExternalLLM = true
	assert.True(t, sf.ExternalLLMAllowed())
}
