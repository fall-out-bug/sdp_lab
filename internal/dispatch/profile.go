package dispatch

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CapabilityScore holds performance metrics for a harness on a given task type + language.
type CapabilityScore struct {
	TestPassRate float64 `json:"test_pass_rate"` // 0.0-1.0
	AvgDuration  float64 `json:"avg_duration"`   // minutes
	SampleCount  int     `json:"sample_count"`
}

// TierClass groups profiles by cost/capability tier for cascade routing.
// Used by SelectTiers to filter Router-ranked profiles into ordered
// tier-chains (fast → strong escalation).
type TierClass string

const (
	TierFast     TierClass = "fast"     // cheap, low-latency: composer-2-fast, gpt-5.3-codex-low, qwen2.5-coder, glm-4.7
	TierBalanced TierClass = "balanced" // medium: composer-2, gpt-5.3-codex, sonnet, gpt-5.2
	TierStrong   TierClass = "strong"   // top-tier: gpt-5.3-codex-xhigh, opus-4
	TierLocal    TierClass = "local"    // Ollama tier — no API cost
)

// IsValidTier reports whether s is a recognised TierClass value.
// Empty string is also valid (untiered profile, back-compat).
func IsValidTier(s string) bool {
	switch TierClass(s) {
	case TierFast, TierBalanced, TierStrong, TierLocal, "":
		return true
	}
	return false
}

// CapabilityProfile stores scored capabilities for a specific harness/provider/model triple.
type CapabilityProfile struct {
	Harness      string                     `json:"harness"`
	Provider     string                     `json:"provider"`
	Model        string                     `json:"model"`
	Capabilities map[string]CapabilityScore `json:"capabilities"` // key: "taskType:language"
	UpdatedAt    string                     `json:"updated_at,omitempty"`
	TierClass    TierClass                  `json:"tier_class,omitempty"` // F145: cascade tier label
}

// ScoreFor returns the TestPassRate for the given taskType and language combination.
// Returns 0 if no data exists for that combination.
func (p *CapabilityProfile) ScoreFor(taskType, language string) float64 {
	if p.Capabilities == nil {
		return 0.0
	}
	key := fmt.Sprintf("%s:%s", taskType, language)
	score, ok := p.Capabilities[key]
	if !ok {
		return 0.0
	}
	return score.TestPassRate
}

// ProfileStore reads and writes CapabilityProfile JSON files from a directory.
type ProfileStore struct {
	Dir string // .sdp/dispatch/profiles/
}

// NewProfileStore returns a ProfileStore rooted under projectRoot.
func NewProfileStore(projectRoot string) *ProfileStore {
	return &ProfileStore{
		Dir: filepath.Join(projectRoot, ".sdp", "dispatch", "profiles"),
	}
}

// profileFileName returns the canonical file name for a harness/provider/model triple.
func profileFileName(harness, provider, model string) string {
	return fmt.Sprintf("%s-%s-%s.json", harness, provider, model)
}

// Load reads the profile for the given harness/provider/model from disk.
func (s *ProfileStore) Load(harness, provider, model string) (*CapabilityProfile, error) {
	path := filepath.Join(s.Dir, profileFileName(harness, provider, model))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("profile load %s: %w", path, err)
	}
	var p CapabilityProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("profile parse %s: %w", path, err)
	}
	return &p, nil
}

// Save writes the profile to disk atomically (tmp file + rename).
func (s *ProfileStore) Save(p *CapabilityProfile) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("profile save mkdir: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("profile marshal: %w", err)
	}

	dest := filepath.Join(s.Dir, profileFileName(p.Harness, p.Provider, p.Model))

	tmp, err := os.CreateTemp(s.Dir, "profile-*.tmp")
	if err != nil {
		return fmt.Errorf("profile save tmp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("profile write tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("profile close tmp: %w", err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("profile rename: %w", err)
	}

	slog.Debug("profile saved", "path", dest)
	return nil
}

// LoadAll reads every .json file in Dir and returns the parsed profiles.
// If Dir does not exist, it returns nil, nil.
// Non-JSON files and unparseable files are silently skipped.
func (s *ProfileStore) LoadAll() ([]*CapabilityProfile, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("profile loadall readdir: %w", err)
	}

	var profiles []*CapabilityProfile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(s.Dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("profile loadall read error", "path", path, "err", err)
			continue
		}
		var p CapabilityProfile
		if err := json.Unmarshal(data, &p); err != nil {
			slog.Warn("profile loadall parse error", "path", path, "err", err)
			continue
		}
		profiles = append(profiles, &p)
	}
	return profiles, nil
}
