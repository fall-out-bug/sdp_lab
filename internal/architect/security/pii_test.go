package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIIScrubber_ScanEmail(t *testing.T) {
	ps := NewPIIScrubber()

	text := "Contact us at support@example.com for help"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeEmail, matches[0].Type)
	assert.Equal(t, "support@example.com", matches[0].Original)
}

func TestPIIScrubber_ScanIPv4(t *testing.T) {
	ps := NewPIIScrubber()

	text := "Server is at 192.168.1.1"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeIPv4, matches[0].Type)
	assert.Equal(t, "192.168.1.1", matches[0].Original)
}

func TestPIIScrubber_ScanSSN(t *testing.T) {
	ps := NewPIIScrubber()

	text := "SSN: 123-45-6789"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeSSN, matches[0].Type)
	assert.Equal(t, "123-45-6789", matches[0].Original)
}

func TestPIIScrubber_ScanPhone(t *testing.T) {
	ps := NewPIIScrubber()

	text := "Call me at (555) 123-4567"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypePhone, matches[0].Type)
}

func TestPIIScrubber_ScanUserPath(t *testing.T) {
	ps := NewPIIScrubber()

	text := "File at /Users/johndoe/projects/app/main.go"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeUserPath, matches[0].Type)
	assert.Contains(t, matches[0].Original, "/Users/johndoe/")
}

func TestPIIScrubber_ScanWindowsUserPath(t *testing.T) {
	ps := NewPIIScrubber()

	text := `C:\Users\janedoe\Documents\file.txt`
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeUserPath, matches[0].Type)
	assert.Contains(t, matches[0].Original, `C:\Users\janedoe\`)
}

func TestPIIScrubber_ScanInternalPackage(t *testing.T) {
	ps := NewPIIScrubber()

	text := "import com.acme.internal.service.UserService"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeInternalPkg, matches[0].Type)
	assert.Equal(t, "com.acme.internal.service.UserService", matches[0].Original)
}

func TestPIIScrubber_ScrubEmail(t *testing.T) {
	ps := NewPIIScrubber()

	text := "Contact support@example.com for help"
	result, matches := ps.Scrub(text)

	assert.NotContains(t, result, "support@example.com")
	assert.Contains(t, result, "[REDACTED_email]")
	assert.Len(t, matches, 1)
}

func TestPIIScrubber_ScrubMultiple(t *testing.T) {
	ps := NewPIIScrubber()

	text := `
		Email: john@example.com
		IP: 192.168.1.1
		SSN: 123-45-6789
		Phone: (555) 123-4567
		Path: /Users/johndoe/file.txt
	`
	result, matches := ps.Scrub(text)

	assert.NotContains(t, result, "john@example.com")
	assert.NotContains(t, result, "192.168.1.1")
	assert.NotContains(t, result, "123-45-6789")
	assert.NotContains(t, result, "(555) 123-4567")
	assert.NotContains(t, result, "/Users/johndoe/")
	assert.NotContains(t, result, "[REDACTED_johndoe]")

	assert.GreaterOrEqual(t, len(matches), 5)
}

func TestPIIScrubber_EnableDisableType(t *testing.T) {
	ps := NewPIIScrubber()

	// Disable email detection
	ps.DisableType(PIITypeEmail)

	text := "Contact support@example.com"
	matches := ps.Scan(text)

	assert.Empty(t, matches)

	// Re-enable email detection
	ps.EnableType(PIITypeEmail)
	matches = ps.Scan(text)

	assert.Len(t, matches, 1)
}

func TestPIIScrubber_AddCustomPattern(t *testing.T) {
	ps := NewPIIScrubber()

	err := ps.AddCustomPattern(PIIType("employee_id"), `EMP-\d{4}`)
	require.NoError(t, err)

	// Enable the custom type
	ps.EnableType(PIIType("employee_id"))

	text := "Employee ID: EMP-1234"
	matches := ps.Scan(text)

	assert.Len(t, matches, 1)
	assert.Equal(t, PIIType("employee_id"), matches[0].Type)
}

func TestScrubPII(t *testing.T) {
	text := `
		Email: admin@example.com
		IP: 10.0.0.1
		Path: /Users/admin/config.yaml
	`

	result, counts := ScrubPII(text)

	assert.NotContains(t, result, "admin@example.com")
	assert.NotContains(t, result, "10.0.0.1")
	assert.NotContains(t, result, "/Users/admin/")

	assert.Greater(t, counts[PIITypeEmail], 0)
	assert.Greater(t, counts[PIITypeIPv4], 0)
	assert.Greater(t, counts[PIITypeUserPath], 0)
}

func TestPIIScrubber_APIKeyDetection(t *testing.T) {
	ps := NewPIIScrubber()
	ps.EnableType(PIITypeAPIKey)

	// High entropy alphanumeric string should be detected
	// The regex matches alphanumeric strings 20+ chars, so we need
	// a string with high entropy that matches this pattern
	// Note: Using lowercase+uppercase+digits to avoid base64 allowlist
	highEntropy := "aB3xY7mN9qR2kL5pW8jH4fG6dS0zX1vN2"
	text := "API key: " + highEntropy

	matches := ps.Scan(text)

	// Should detect as API key due to high entropy
	found := false
	for _, m := range matches {
		if m.Type == PIITypeAPIKey {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect high entropy string as API key")
}

func TestPIIScrubber_NoFalsePositives(t *testing.T) {
	ps := NewPIIScrubber()

	benign := []string{
		"function main() { return true; }",
		"const x = 42",
		"package main",
		"import fmt",
		"// This is a comment",
		"TODO: implement this",
		"NOTE: check this later",
	}

	for _, text := range benign {
		matches := ps.Scan(text)
		assert.Empty(t, matches, "false positive in: %q", text)
	}
}

func TestPIIScrubber_CreditCard(t *testing.T) {
	ps := NewPIIScrubber()
	ps.EnableType(PIITypeCreditCard)

	text := "Card number: 4532015112830366"
	matches := ps.Scan(text)

	// Should detect credit card pattern
	found := false
	for _, m := range matches {
		if m.Type == PIITypeCreditCard {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect credit card pattern")
}

func TestPIIScrubber_MixedPII(t *testing.T) {
	ps := NewPIIScrubber()

	text := `
		User: John Doe <john.doe@example.com>
		Last login from 192.168.1.100
		Phone: +1-555-123-4567
		SSN: 987-65-4321
		Home: /Users/johndoe/Documents
		Windows: C:\Users\janedoe\Downloads
		Package: com.company.internal.auth.AuthService
	`

	result, matches := ps.Scrub(text)

	// Check that all PII was scrubbed
	assert.NotContains(t, result, "john.doe@example.com")
	assert.NotContains(t, result, "192.168.1.100")
	assert.NotContains(t, result, "+1-555-123-4567")
	assert.NotContains(t, result, "987-65-4321")
	assert.NotContains(t, result, "/Users/johndoe/")
	assert.NotContains(t, result, `C:\Users\janedoe\`)
	assert.NotContains(t, result, "com.company.internal")

	// Check that we found multiple types
	typeCount := make(map[PIIType]bool)
	for _, m := range matches {
		typeCount[m.Type] = true
	}

	assert.GreaterOrEqual(t, len(typeCount), 6, "should detect at least 6 types of PII")
}

func TestPIIScrubber_JWTToken(t *testing.T) {
	ps := NewPIIScrubber()
	// JWT detection is in the filter, not PII, but let's make sure we don't conflict
	text := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0"

	matches := ps.Scan(text)
	// JWT won't be detected by PII scrubber (that's in security filter)
	// This just ensures no false positives
	for _, m := range matches {
		if strings.Contains(m.Original, "eyJ") {
			t.Errorf("PII scrubber should not detect JWT tokens")
		}
	}
}

func TestPIIScrubber_URLDetection(t *testing.T) {
	ps := NewPIIScrubber()
	ps.EnableType(PIITypeURL)

	text := "Visit https://example.com/page?id=123 for more info"
	matches := ps.Scan(text)

	require.Len(t, matches, 1)
	assert.Equal(t, PIITypeURL, matches[0].Type)
	assert.Equal(t, "https://example.com/page?id=123", matches[0].Original)
}

func TestPIIScrubber_IPv6(t *testing.T) {
	ps := NewPIIScrubber()
	ps.EnableType(PIITypeIPv6)

	text := "IPv6 address: 2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	matches := ps.Scan(text)

	found := false
	for _, m := range matches {
		if m.Type == PIITypeIPv6 {
			found = true
			break
		}
	}
	assert.True(t, found, "should detect IPv6 address")
}
