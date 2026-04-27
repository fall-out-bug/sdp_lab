package dispatch

import (
	"encoding/json"
	"testing"
)

// TestDefaultProfiles_LoadAndValidate checks that profiles_default.json is valid,
// has the required number of entries, and each entry is properly formed.
func TestDefaultProfiles_LoadAndValidate(t *testing.T) {
	// Load the embedded JSON
	data := defaultProfilesJSON()
	if len(data) == 0 {
		t.Fatal("profiles_default.json not found or empty")
	}

	var profiles []*CapabilityProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		t.Fatalf("failed to unmarshal profiles_default.json: %v", err)
	}

	// AC: ≥40 entries
	if len(profiles) < 40 {
		t.Errorf("profiles count = %d, want ≥40", len(profiles))
	}

	// Count by tier
	tierCounts := map[TierClass]int{
		TierFast:     0,
		TierBalanced: 0,
		TierStrong:   0,
		TierLocal:    0,
	}

	// Validate each profile
	for i, p := range profiles {
		if p.Harness == "" {
			t.Errorf("profiles[%d]: harness is empty", i)
		}
		if p.Provider == "" {
			t.Errorf("profiles[%d]: provider is empty", i)
		}
		if p.Model == "" {
			t.Errorf("profiles[%d]: model is empty", i)
		}

		// Check tier_class is valid
		if !IsValidTier(string(p.TierClass)) {
			t.Errorf("profiles[%d]: invalid tier_class %q (harness=%s, provider=%s, model=%s)",
				i, p.TierClass, p.Harness, p.Provider, p.Model)
		}

		// Count tiers
		if p.TierClass != "" {
			tierCounts[p.TierClass]++
		}

		// Check capabilities are non-empty
		if len(p.Capabilities) == 0 {
			t.Errorf("profiles[%d]: capabilities map is empty (harness=%s, provider=%s, model=%s)",
				i, p.Harness, p.Provider, p.Model)
		}

		// Check at least one capability has non-zero test_pass_rate
		hasNonZeroScore := false
		for _, score := range p.Capabilities {
			if score.TestPassRate > 0 {
				hasNonZeroScore = true
				break
			}
		}
		if !hasNonZeroScore {
			t.Errorf("profiles[%d]: all capabilities have zero test_pass_rate (harness=%s, provider=%s, model=%s)",
				i, p.Harness, p.Provider, p.Model)
		}
	}

	// AC: Tier distribution
	if tierCounts[TierFast] < 10 {
		t.Errorf("TierFast count = %d, want ≥10", tierCounts[TierFast])
	}
	if tierCounts[TierBalanced] < 10 {
		t.Errorf("TierBalanced count = %d, want ≥10", tierCounts[TierBalanced])
	}
	if tierCounts[TierStrong] < 6 {
		t.Errorf("TierStrong count = %d, want ≥6", tierCounts[TierStrong])
	}
	if tierCounts[TierLocal] < 3 {
		t.Errorf("TierLocal count = %d, want ≥3", tierCounts[TierLocal])
	}
}

// TestDefaultProfiles_ProviderModelValidation verifies that every (provider, model)
// pair in profiles_default.json corresponds to a known model in the respective Provider.
func TestDefaultProfiles_ProviderModelValidation(t *testing.T) {
	data := defaultProfilesJSON()
	var profiles []*CapabilityProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		t.Fatalf("failed to unmarshal profiles_default.json: %v", err)
	}

	// Build provider catalogs
	openaiCatalog := map[string]bool{
		"gpt-5":          true,
		"gpt-5-codex":    true,
		"o1":             true,
		"o1-pro":         true,
		"o3":             true,
		"o3-mini":        true,
		"gpt-4o":         true,
		"gpt-4o-mini":    true,
	}

	anthropicCatalog := map[string]bool{
		"claude-opus-4-7":    true,
		"claude-sonnet-4-6":  true,
		"claude-haiku-4-5":   true,
		"claude-opus-4-1":    true,
		"claude-sonnet-4-5":  true,
		"claude-haiku-4-1":   true,
	}

	kimiCatalog := map[string]bool{
		"kimi-k1.5":              true,
		"kimi-k2":                true,
		"moonshot-v1-8k":         true,
		"moonshot-v1-32k":        true,
		"moonshot-v1-128k":       true,
	}

	// Ollama and Cursor are dynamic, so we allow any models.
	// The test is validating against the static catalogs we know about.

	for i, p := range profiles {
		switch p.Provider {
		case "openai":
			if !openaiCatalog[p.Model] {
				t.Errorf("profiles[%d]: model %q not in openai catalog", i, p.Model)
			}
		case "anthropic":
			if !anthropicCatalog[p.Model] {
				t.Errorf("profiles[%d]: model %q not in anthropic catalog", i, p.Model)
			}
		case "kimi":
			if !kimiCatalog[p.Model] {
				t.Errorf("profiles[%d]: model %q not in kimi catalog", i, p.Model)
			}
		case "cursor":
			// Cursor models are dynamic; just check not empty
			if p.Model == "" {
				t.Errorf("profiles[%d]: cursor model is empty", i)
			}
		case "ollama":
			// Ollama models are dynamic; just check not empty
			if p.Model == "" {
				t.Errorf("profiles[%d]: ollama model is empty", i)
			}
		default:
			t.Errorf("profiles[%d]: unknown provider %q", i, p.Provider)
		}
	}
}

// TestDefaultProfiles_Function verifies that DefaultProfiles() returns all profiles.
func TestDefaultProfiles_Function(t *testing.T) {
	profiles := DefaultProfiles()
	if len(profiles) < 40 {
		t.Errorf("DefaultProfiles() returned %d profiles, want ≥40", len(profiles))
	}

	// Spot check: verify at least one profile from each tier
	tiers := map[TierClass]bool{}
	for _, p := range profiles {
		if p.TierClass != "" {
			tiers[p.TierClass] = true
		}
	}

	for _, requiredTier := range []TierClass{TierFast, TierBalanced, TierStrong, TierLocal} {
		if !tiers[requiredTier] {
			t.Errorf("DefaultProfiles() missing tier %s", requiredTier)
		}
	}
}

// defaultProfilesJSON returns the embedded JSON.
func defaultProfilesJSON() []byte {
	return defaultProfilesData
}
