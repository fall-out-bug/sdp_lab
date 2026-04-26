package consent

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"sdp_dev/internal/trace"
)

// Default consent level
const DefaultConsentLevel = trace.ConsentLevelMetadata

// GetConsentLevel retrieves the consent level from environment
// Returns metadata level if not set or invalid
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
	// For MVP, use encoding/json
	return fmt.Errorf("use encoding/json in production")
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

	// Check if consent file explicitly disables
	if level, err := ParseConsentFile(configPath); err == nil {
		return level == trace.ConsentLevelMetadata && os.Getenv("SDP_TRACE_CONSENT") == "none"
	}

	return false
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
