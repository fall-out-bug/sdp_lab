package consent

import (
	"os"
	"testing"

	"sdp_dev/internal/trace"
)

func TestGetConsentLevel(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected trace.ConsentLevel
	}{
		{"default", "", trace.ConsentLevelMetadata},
		{"metadata", "metadata", trace.ConsentLevelMetadata},
		{"findings", "findings", trace.ConsentLevelFindings},
		{"content", "content", trace.ConsentLevelContent},
		{"invalid", "invalid", trace.ConsentLevelMetadata},
		{"uppercase", "METADATA", trace.ConsentLevelMetadata},
		{"mixed case", "FinDinGs", trace.ConsentLevelFindings},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVal != "" {
				os.Setenv("SDP_TRACE_CONSENT", tt.envVal)
				defer os.Unsetenv("SDP_TRACE_CONSENT")
			}

			got := GetConsentLevel()
			if got != tt.expected {
				t.Errorf("GetConsentLevel() = %v, want %v", got, tt.expected)
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
		name     string
		envVal   string
		expected bool
	}{
		{"not disabled", "", false},
		{"explicitly disabled", "true", true},
		{"explicitly enabled", "false", false},
		{"capitalized", "TRUE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv("SDP_TRACE_DISABLED", tt.envVal)
				defer os.Unsetenv("SDP_TRACE_DISABLED")
			}

			got := IsDisabled("")
			if got != tt.expected {
				t.Errorf("IsDisabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}
