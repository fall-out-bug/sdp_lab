package llmguard

import (
	"testing"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()

	if p.InputAction != InputActionBlock {
		t.Errorf("default input action should be block, got %s", p.InputAction)
	}
	if p.OutputAction != OutputActionBlock {
		t.Errorf("default output action should be block, got %s", p.OutputAction)
	}
	if p.MaxInputBytes <= 0 {
		t.Error("max input bytes should be positive")
	}
	if p.MaxDecodedBytes <= 0 {
		t.Error("max decoded bytes should be positive")
	}
	if !p.StrictBudgetMode {
		t.Error("default should be strict budget mode")
	}
}

func TestVerdictStates(t *testing.T) {
	states := map[string]VerdictState{
		"clean_allowed":                       VerdictCleanAllowed,
		"redacted_allowed":                    VerdictRedactedAllowed,
		"input_blocked":                       VerdictInputBlocked,
		"output_blocked":                      VerdictOutputBlocked,
		"allowed_with_output_findings":        VerdictAllowedWithOutputFindings,
		"provider_error_after_input_pass":     VerdictProviderErrorAfterInputPass,
		"audit_failed":                        VerdictAuditFailed,
		"scan_budget_exceeded":                VerdictScanBudgetExceeded,
	}

	for expected, actual := range states {
		if string(actual) != expected {
			t.Errorf("expected %s, got %s", expected, actual)
		}
	}
}

func TestRedactedPlaceholder(t *testing.T) {
	tests := []struct {
		ft         FindingType
		placeholder string
	}{
		{FindingOpenAIKey, "[REDACTED_API_KEY]"},
		{FindingBase64Key, "[REDACTED_API_KEY]"},
		{FindingSplitKey, "[REDACTED_API_KEY]"},
		{FindingGitHubToken, "[REDACTED_GITHUB_TOKEN]"},
		{FindingAWSKey, "[REDACTED_AWS_KEY]"},
		{FindingBearerToken, "[REDACTED_BEARER_TOKEN]"},
		{FindingEmail, "[REDACTED_EMAIL]"},
		{FindingCard, "[REDACTED_CARD]"},
		{FindingPhone, "[REDACTED_PHONE]"},
		{FindingGeneratedSecret, "[REDACTED_GENERATED_SECRET]"},
		{FindingPromptDisclosure, "[REDACTED_PROMPT_DISCLOSURE]"},
		{FindingSuspiciousURL, "[REDACTED_URL]"},
		{FindingShellCommand, "[REDACTED_SHELL_COMMAND]"},
	}

	for _, tc := range tests {
		got := RedactedPlaceholder(tc.ft)
		if got != tc.placeholder {
			t.Errorf("RedactedPlaceholder(%s) = %q, want %q", tc.ft, got, tc.placeholder)
		}
	}
}

func TestRedactedPlaceholder_Unknown(t *testing.T) {
	got := RedactedPlaceholder(FindingType("unknown"))
	if got != "[REDACTED]" {
		t.Errorf("unknown finding type should return generic placeholder, got %q", got)
	}
}
