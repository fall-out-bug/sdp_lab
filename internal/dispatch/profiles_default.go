package dispatch

import (
	_ "embed"
	"encoding/json"
)

//go:embed profiles_default.json
var defaultProfilesData []byte

// DefaultProfiles returns the embedded default profiles.
// Panics if the JSON is invalid (should never happen in production).
func DefaultProfiles() []*CapabilityProfile {
	var profiles []*CapabilityProfile
	if err := json.Unmarshal(defaultProfilesData, &profiles); err != nil {
		// This should never happen at runtime; indicates a corrupt build.
		panic("profiles_default.json unmarshal failed: " + err.Error())
	}
	return profiles
}
