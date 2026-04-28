package consent

import (
	"os"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/trace"
)

// Helper to clear all telemetry-related env vars before each test.
func clearTelemetryEnv(t *testing.T) {
	t.Helper()
	os.Unsetenv("SDP_TRACE_CONSENT")
	os.Unsetenv("SDP_TRACE_DISABLED")
	os.Unsetenv("SDP_OTEL_ENDPOINT")
	os.Unsetenv("SDP_OTEL_HEADERS")
	os.Unsetenv("SDP_OTEL_TIMEOUT")
	os.Unsetenv("SDP_OTEL_SERVICE_NAME")
	os.Unsetenv("SDP_OTEL_INSECURE")
}

func TestGetConsentLevel(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected trace.ConsentLevel
	}{
		{"default", "", trace.ConsentLevelMetadata},
		{"none", "none", trace.ConsentLevelNone},
		{"metadata", "metadata", trace.ConsentLevelMetadata},
		{"findings", "findings", trace.ConsentLevelFindings},
		{"content", "content", trace.ConsentLevelContent},
		{"invalid", "invalid", trace.ConsentLevelMetadata},
		{"uppercase", "METADATA", trace.ConsentLevelMetadata},
		{"uppercase none", "NONE", trace.ConsentLevelNone},
		{"mixed case", "FinDinGs", trace.ConsentLevelFindings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTelemetryEnv(t)
			// Set environment variable
			if tt.envVal != "" {
				os.Setenv("SDP_TRACE_CONSENT", tt.envVal)
			}

			got := GetConsentLevel()
			if got != tt.expected {
				t.Errorf("GetConsentLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConsentLevelIsValid(t *testing.T) {
	tests := []struct {
		level    trace.ConsentLevel
		expected bool
	}{
		{trace.ConsentLevelNone, true},
		{trace.ConsentLevelMetadata, true},
		{trace.ConsentLevelFindings, true},
		{trace.ConsentLevelContent, true},
		{trace.ConsentLevel("invalid"), false},
		{trace.ConsentLevel(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.expected {
				t.Errorf("ConsentLevel(%q).IsValid() = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestConsentLevelAllowsExport(t *testing.T) {
	tests := []struct {
		level    trace.ConsentLevel
		expected bool
	}{
		{trace.ConsentLevelNone, false},
		{trace.ConsentLevelMetadata, true},
		{trace.ConsentLevelFindings, true},
		{trace.ConsentLevelContent, true},
		{trace.ConsentLevel("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if got := tt.level.AllowsExport(); got != tt.expected {
				t.Errorf("ConsentLevel(%q).AllowsExport() = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestShouldRedactContent(t *testing.T) {
	tests := []struct {
		name     string
		current  trace.ConsentLevel
		required trace.ConsentLevel
		expected bool
	}{
		{"metadata to metadata", trace.ConsentLevelMetadata, trace.ConsentLevelMetadata, false},
		{"metadata to findings", trace.ConsentLevelMetadata, trace.ConsentLevelFindings, true},
		{"metadata to content", trace.ConsentLevelMetadata, trace.ConsentLevelContent, true},
		{"findings to metadata", trace.ConsentLevelFindings, trace.ConsentLevelMetadata, false},
		{"findings to findings", trace.ConsentLevelFindings, trace.ConsentLevelFindings, false},
		{"findings to content", trace.ConsentLevelFindings, trace.ConsentLevelContent, true},
		{"content to all", trace.ConsentLevelContent, trace.ConsentLevelMetadata, false},
		{"none to metadata", trace.ConsentLevelNone, trace.ConsentLevelMetadata, true},
		{"none to content", trace.ConsentLevelNone, trace.ConsentLevelContent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRedactContent(tt.current, tt.required)
			if got != tt.expected {
				t.Errorf("ShouldRedactContent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateAttribute(t *testing.T) {
	tests := []struct {
		name          string
		attrName      string
		attrValue     string
		consentLevel  trace.ConsentLevel
		expectError   bool
		expectedValue string
	}{
		{
			name:         "metadata attribute at metadata level",
			attrName:     "gen_ai.tool.name",
			attrValue:    "Bash",
			consentLevel: trace.ConsentLevelMetadata,
			expectError:  false,
		},
		{
			name:         "findings attribute at metadata level",
			attrName:     "sdp.review.verdict",
			attrValue:    "APPROVED",
			consentLevel: trace.ConsentLevelMetadata,
			expectError:  true,
		},
		{
			name:         "findings attribute at findings level",
			attrName:     "sdp.review.verdict",
			attrValue:    "APPROVED",
			consentLevel: trace.ConsentLevelFindings,
			expectError:  false,
		},
		{
			name:         "findings attribute at none level",
			attrName:     "sdp.review.verdict",
			attrValue:    "APPROVED",
			consentLevel: trace.ConsentLevelNone,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAttribute(tt.attrName, tt.attrValue, tt.consentLevel)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && tt.expectedValue != "" && got != tt.expectedValue {
				t.Errorf("ValidateAttribute() = %v, want %v", got, tt.expectedValue)
			}
		})
	}
}

func TestIsDisabled(t *testing.T) {
	tests := []struct {
		name        string
		disabledEnv string
		consentEnv  string
		expected    bool
	}{
		{"not disabled", "", "", false},
		{"explicitly disabled flag", "true", "", true},
		{"explicitly enabled flag", "false", "", false},
		{"capitalized flag", "TRUE", "", true},
		{"consent none", "", "none", true},
		{"consent none with disabled false", "false", "none", true},
		{"consent metadata", "", "metadata", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTelemetryEnv(t)
			if tt.disabledEnv != "" {
				os.Setenv("SDP_TRACE_DISABLED", tt.disabledEnv)
			}
			if tt.consentEnv != "" {
				os.Setenv("SDP_TRACE_CONSENT", tt.consentEnv)
			}

			got := IsDisabled("")
			if got != tt.expected {
				t.Errorf("IsDisabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetOTELConfig_NoEndpoint(t *testing.T) {
	clearTelemetryEnv(t)
	// No endpoint configured — config must be nil
	cfg := GetOTELConfig()
	if cfg != nil {
		t.Error("GetOTELConfig() should return nil when no endpoint is set")
	}
}

func TestGetOTELConfig_ConsentNone(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "http://localhost:4318/v1/traces")
	os.Setenv("SDP_TRACE_CONSENT", "none")

	cfg := GetOTELConfig()
	if cfg != nil {
		t.Error("GetOTELConfig() should return nil when consent is none")
	}
}

func TestGetOTELConfig_ValidConfig(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "http://localhost:4318/v1/traces")
	os.Setenv("SDP_TRACE_CONSENT", "metadata")

	cfg := GetOTELConfig()
	if cfg == nil {
		t.Fatal("GetOTELConfig() should return non-nil config with endpoint and valid consent")
	}
	if cfg.Endpoint != "http://localhost:4318/v1/traces" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "http://localhost:4318/v1/traces")
	}
	if cfg.ConsentLevel != trace.ConsentLevelMetadata {
		t.Errorf("ConsentLevel = %v, want %v", cfg.ConsentLevel, trace.ConsentLevelMetadata)
	}
	if cfg.TimeoutSeconds != 5 {
		t.Errorf("TimeoutSeconds = %d, want 5 (default)", cfg.TimeoutSeconds)
	}
	if cfg.ServiceName != "sdp-telemetry" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "sdp-telemetry")
	}
	if cfg.Insecure {
		t.Error("Insecure should be false by default")
	}
}

func TestGetOTELConfig_AllOptions(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "https://otel.example.com:4317")
	os.Setenv("SDP_TRACE_CONSENT", "findings")
	os.Setenv("SDP_OTEL_HEADERS", "api-key=secret123,tenant=acme")
	os.Setenv("SDP_OTEL_TIMEOUT", "10")
	os.Setenv("SDP_OTEL_SERVICE_NAME", "my-project")
	os.Setenv("SDP_OTEL_INSECURE", "true")

	cfg := GetOTELConfig()
	if cfg == nil {
		t.Fatal("GetOTELConfig() should return non-nil config")
	}
	if cfg.ConsentLevel != trace.ConsentLevelFindings {
		t.Errorf("ConsentLevel = %v, want %v", cfg.ConsentLevel, trace.ConsentLevelFindings)
	}
	if cfg.TimeoutSeconds != 10 {
		t.Errorf("TimeoutSeconds = %d, want 10", cfg.TimeoutSeconds)
	}
	if cfg.ServiceName != "my-project" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "my-project")
	}
	if !cfg.Insecure {
		t.Error("Insecure should be true")
	}
	if cfg.Headers["api-key"] != "secret123" {
		t.Errorf("Headers[api-key] = %q, want %q", cfg.Headers["api-key"], "secret123")
	}
	if cfg.Headers["tenant"] != "acme" {
		t.Errorf("Headers[tenant] = %q, want %q", cfg.Headers["tenant"], "acme")
	}
}

func TestGetOTELConfig_InvalidTimeout(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "http://localhost:4318")
	os.Setenv("SDP_TRACE_CONSENT", "content")
	os.Setenv("SDP_OTEL_TIMEOUT", "not-a-number")

	cfg := GetOTELConfig()
	if cfg == nil {
		t.Fatal("GetOTELConfig() should return non-nil config")
	}
	if cfg.TimeoutSeconds != 5 {
		t.Errorf("TimeoutSeconds = %d, want 5 (default on invalid input)", cfg.TimeoutSeconds)
	}
}

func TestGetOTELConfig_NegativeTimeout(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "http://localhost:4318")
	os.Setenv("SDP_TRACE_CONSENT", "content")
	os.Setenv("SDP_OTEL_TIMEOUT", "-3")

	cfg := GetOTELConfig()
	if cfg == nil {
		t.Fatal("GetOTELConfig() should return non-nil config")
	}
	if cfg.TimeoutSeconds != 5 {
		t.Errorf("TimeoutSeconds = %d, want 5 (default on negative input)", cfg.TimeoutSeconds)
	}
}

func TestGetOTELConfig_MalformedHeaders(t *testing.T) {
	clearTelemetryEnv(t)
	os.Setenv("SDP_OTEL_ENDPOINT", "http://localhost:4318")
	os.Setenv("SDP_TRACE_CONSENT", "metadata")
	os.Setenv("SDP_OTEL_HEADERS", "no-equals-sign,valid-key=valid-value")

	cfg := GetOTELConfig()
	if cfg == nil {
		t.Fatal("GetOTELConfig() should return non-nil config")
	}
	if len(cfg.Headers) != 1 {
		t.Errorf("Headers count = %d, want 1 (malformed entry skipped)", len(cfg.Headers))
	}
	if cfg.Headers["valid-key"] != "valid-value" {
		t.Errorf("Headers[valid-key] = %q, want %q", cfg.Headers["valid-key"], "valid-value")
	}
}

func TestIsExportAllowed(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		consent     string
		expected    bool
	}{
		{"no endpoint", "", "metadata", false},
		{"no endpoint no consent", "", "", false},
		{"endpoint with consent none", "http://localhost:4318", "none", false},
		{"endpoint with consent metadata", "http://localhost:4318", "metadata", true},
		{"endpoint with consent findings", "http://localhost:4318", "findings", true},
		{"endpoint with consent content", "http://localhost:4318", "content", true},
		{"endpoint with default consent", "http://localhost:4318", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTelemetryEnv(t)
			if tt.endpoint != "" {
				os.Setenv("SDP_OTEL_ENDPOINT", tt.endpoint)
			}
			if tt.consent != "" {
				os.Setenv("SDP_TRACE_CONSENT", tt.consent)
			}

			got := IsExportAllowed()
			if got != tt.expected {
				t.Errorf("IsExportAllowed() = %v, want %v", got, tt.expected)
			}
		})
	}
}
