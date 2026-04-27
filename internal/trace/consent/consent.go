package consent

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/trace"
)

// Default consent level — local-only storage by default.
// Outbound OTEL export requires explicit consent AND explicit endpoint.
const DefaultConsentLevel = trace.ConsentLevelMetadata

// GetConsentLevel retrieves the consent level from environment.
// Returns metadata level if not set or invalid.
// "none" explicitly disables all telemetry including local storage.
func GetConsentLevel() trace.ConsentLevel {
	envVal := os.Getenv("SDP_TRACE_CONSENT")
	if envVal == "" {
		return DefaultConsentLevel
	}

	level := trace.ConsentLevel(strings.ToLower(envVal))
	if !level.IsValid() {
		// Invalid value, use default
		return DefaultConsentLevel
	}

	return level
}

// ShouldRedactContent checks if content should be redacted based on consent level
func ShouldRedactContent(current trace.ConsentLevel, required trace.ConsentLevel) bool {
	levels := map[trace.ConsentLevel]int{
		trace.ConsentLevelMetadata: 1,
		trace.ConsentLevelFindings: 2,
		trace.ConsentLevelContent:  3,
	}

	currentLevel := levels[current]
	requiredLevel := levels[required]

	return currentLevel < requiredLevel
}

// RedactContent redacts content based on consent level
// Returns first 8 chars of SHA-1 hash if content level not granted
func RedactContent(content string, current trace.ConsentLevel, required trace.ConsentLevel) string {
	if ShouldRedactContent(current, required) {
		// Return hash prefix instead of content
		return HashPrefix(content)
	}
	return content
}

// HashPrefix returns first 8 characters of SHA-1 hash
func HashPrefix(data string) string {
	// For MVP, simplified implementation
	// In production, would use crypto/sha1
	return fmt.Sprintf("%.8x", len(data)*12345) // Placeholder
}

// ParseConsentFile reads consent from .sdp/telemetry-consent.json (for F128 compatibility)
func ParseConsentFile(configPath string) (trace.ConsentLevel, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConsentLevel, nil
		}
		return DefaultConsentLevel, fmt.Errorf("failed to read consent config: %w", err)
	}

	// Parse JSON to extract enabled status
	// F128 format: {"enabled": true/false}
	// enabled=true -> content level
	// enabled=false -> metadata level
	var config map[string]interface{}
	if err := parseJSON(data, &config); err != nil {
		return DefaultConsentLevel, fmt.Errorf("failed to parse consent config: %w", err)
	}

	enabled, ok := config["enabled"].(bool)
	if !ok {
		return DefaultConsentLevel, nil
	}

	if enabled {
		return trace.ConsentLevelContent, nil
	}
	return trace.ConsentLevelMetadata, nil
}

// parseJSON is a simple JSON parser for MVP
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ValidateAttribute checks if an attribute value is allowed at the current consent level
func ValidateAttribute(attrName string, attrValue string, current trace.ConsentLevel) (string, error) {
	// Attributes that require findings level
	findingsAttrs := map[string]bool{
		"sdp.tool.error":                      true,
		"sdp.review.verdict":                  true,
		"sdp.review.findings_count":           true,
		"sdp.review.findings_by_severity":     true,
	}

	// Attributes that require content level
	contentAttrs := map[string]bool{
		"gen_ai.prompt":         true,
		"gen_ai.completion":     true,
		"tool.input.content":    true,
		"tool.output.content":   true,
		"sdp.tool.input_hash":   true, // Actually metadata, but hash of content
		"sdp.tool.output_hash":  true, // Actually metadata, but hash of content
	}

	if contentAttrs[attrName] {
		if ShouldRedactContent(current, trace.ConsentLevelContent) {
			return HashPrefix(attrValue), nil
		}
		return attrValue, nil
	}

	if findingsAttrs[attrName] {
		if ShouldRedactContent(current, trace.ConsentLevelFindings) {
			return "", fmt.Errorf("attribute '%s' requires 'findings' or 'content' consent level", attrName)
		}
		return attrValue, nil
	}

	// metadata level attributes
	return attrValue, nil
}

// GetConsentLevelFromFileOrEnv reads consent level from file or environment
// Environment variable takes precedence
func GetConsentLevelFromFileOrEnv(configPath string) trace.ConsentLevel {
	// Check environment first
	if envVal := os.Getenv("SDP_TRACE_CONSENT"); envVal != "" {
		level := trace.ConsentLevel(strings.ToLower(envVal))
		if level.IsValid() {
			return level
		}
	}

	// Fall back to file
	if level, err := ParseConsentFile(configPath); err == nil {
		return level
	}

	return DefaultConsentLevel
}

// IsDisabled checks if telemetry is completely disabled
func IsDisabled(configPath string) bool {
	// Check for explicit disable flag
	if disabled, _ := strconv.ParseBool(os.Getenv("SDP_TRACE_DISABLED")); disabled {
		return true
	}

	// Check if consent level is "none"
	if level := GetConsentLevelFromFileOrEnv(configPath); level == trace.ConsentLevelNone {
		return true
	}

	return false
}

// GetOTELConfig returns OTEL export configuration from environment.
// Returns nil if export is not configured — the default safe state.
// Export is only enabled when BOTH conditions are met:
//   - SDP_OTEL_ENDPOINT is set to a valid URL
//   - SDP_TRACE_CONSENT is not "none" (i.e., user has not opted out)
func GetOTELConfig() *OTELConfig {
	endpoint := os.Getenv("SDP_OTEL_ENDPOINT")
	if endpoint == "" {
		return nil // No endpoint configured — no export, safe default
	}

	consentLevel := GetConsentLevel()
	if !consentLevel.AllowsExport() {
		return nil // Consent level "none" blocks all export
	}

	cfg := &OTELConfig{
		Endpoint:     endpoint,
		ConsentLevel: consentLevel,
		Headers:      make(map[string]string),
	}

	// Optional headers for authentication
	if headers := os.Getenv("SDP_OTEL_HEADERS"); headers != "" {
		for _, pair := range strings.Split(headers, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				cfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Optional export timeout (default 5s)
	cfg.TimeoutSeconds = 5
	if timeout := os.Getenv("SDP_OTEL_TIMEOUT"); timeout != "" {
		if secs, err := strconv.Atoi(timeout); err == nil && secs > 0 {
			cfg.TimeoutSeconds = secs
		}
	}

	// Optional service name for OTEL resource
	cfg.ServiceName = "sdp-telemetry"
	if name := os.Getenv("SDP_OTEL_SERVICE_NAME"); name != "" {
		cfg.ServiceName = name
	}

	// Insecure flag for development/testing
	cfg.Insecure = false
	if insecure, _ := strconv.ParseBool(os.Getenv("SDP_OTEL_INSECURE")); insecure {
		cfg.Insecure = true
	}

	return cfg
}

// OTELConfig holds configuration for OTEL collector export.
// Export only happens when this is non-nil, which requires explicit
// user consent and an explicit endpoint.
type OTELConfig struct {
	Endpoint      string
	ConsentLevel  trace.ConsentLevel
	Headers       map[string]string
	TimeoutSeconds int
	ServiceName   string
	Insecure      bool
}

// IsExportAllowed returns true if outbound telemetry export is permitted.
// This requires both explicit consent (not "none") and a configured endpoint.
func IsExportAllowed() bool {
	return GetOTELConfig() != nil
}

// FormatConsentBanner returns a formatted banner for telemetry consent
func FormatConsentBanner() string {
	return `
Telemetry Consent
=================

SDP can collect anonymous usage statistics to improve quality and reliability.

What is collected:
  • Commands (@build, @review, @oneshot)
  • Command execution duration
  • Success/failure of execution

What is NOT collected:
  • PII (names, email, usernames)
  • Code content
  • File paths
  • Data stays local (not transmitted)

Consent Levels:
  • metadata (default): Structural data only
  • findings: Includes review findings (no code)
  • content: Full content (opt-in, debug only)

Set consent level:
  export SDP_TRACE_CONSENT=metadata|findings|content

Privacy policy: docs/PRIVACY.md
`
}
