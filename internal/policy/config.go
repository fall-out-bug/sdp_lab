package policy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ModelPolicyConfig holds the model policy loaded from YAML.
type ModelPolicyConfig struct {
	Providers        map[string]PolicyProviderConfig `yaml:"providers"`
	Allowlist        []string                        `yaml:"allowlist"`
	Budget           BudgetConfig                    `yaml:"budget"`
	CostOptimization CostOptimizationConfig          `yaml:"cost_optimization"`
	Roles            map[string]RoleModelConfig      `yaml:"roles"`
}

// PolicyProviderConfig holds provider API config from YAML.
type PolicyProviderConfig struct {
	APIKeySecret string `yaml:"api_key_secret"`
	BaseURL      string `yaml:"base_url"`
}

// BudgetConfig holds daily and per-run limits.
type BudgetConfig struct {
	DailyLimitUSD   float64 `yaml:"daily_limit_usd"`
	PerRunLimitUSD  float64 `yaml:"per_run_limit_usd"`
}

// CostOptimizationConfig holds auto-downgrade settings.
type CostOptimizationConfig struct {
	AutoDowngradeAtPct float64  `yaml:"auto_downgrade_at_pct"`
	ExemptRoles        []string `yaml:"exempt_roles"`
}

// RoleModelConfig holds per-role model assignment.
type RoleModelConfig struct {
	Primary   string `yaml:"primary"`
	Fallback  string `yaml:"fallback"`
	Economy   string `yaml:"economy"`
}

// BudgetTracking holds runtime budget state.
type BudgetTracking struct {
	DailySpent   float64
	PerRunSpent  float64
	mu           sync.RWMutex
}

var (
	configMu     sync.RWMutex
	loadedConfig *ModelPolicyConfig
	allowlistSet map[string]struct{}
	budget       BudgetTracking
)

// LoadFromPath loads policy from a YAML file. Returns nil if path empty or file unreadable.
func LoadFromPath(path string) (*ModelPolicyConfig, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ModelPolicyConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	ApplyConfig(&cfg)
	return &cfg, nil
}

// ApplyConfig applies config to package-level allowlist.
func ApplyConfig(cfg *ModelPolicyConfig) {
	configMu.Lock()
	defer configMu.Unlock()
	loadedConfig = cfg
	if cfg == nil || len(cfg.Allowlist) == 0 {
		allowlistSet = nil
		return
	}
	allowlistSet = make(map[string]struct{})
	if len(cfg.Allowlist) > 0 {
		for _, m := range cfg.Allowlist {
			m = strings.TrimSpace(m)
			if m != "" {
				allowlistSet[m] = struct{}{}
				// Also allow provider/model form for OpenRouter
				if !strings.Contains(m, "/") {
					allowlistSet["openrouter/"+m] = struct{}{}
				}
			}
		}
	}
}

// AllowedModelFromConfig returns true if model is in the loaded config allowlist.
// Falls back to built-in allowlist when no config loaded.
func AllowedModelFromConfig(model string) bool {
	configMu.RLock()
	defer configMu.RUnlock()
	if allowlistSet != nil {
		if _, ok := allowlistSet[model]; ok {
			return true
		}
		_, modelID := ParseProviderModel(model)
		if modelID != "" {
			if _, ok := allowlistSet[modelID]; ok {
				return true
			}
		}
		return false
	}
	// Fallback to built-in
	return AllowedModelBuiltin(model)
}

// RoleModel returns primary, fallback, economy for role from config. Empty when no config or role not found.
func RoleModel(role string) (primary, fallback, economy string) {
	configMu.RLock()
	defer configMu.RUnlock()
	if loadedConfig == nil || loadedConfig.Roles == nil {
		return "", "", ""
	}
	r, ok := loadedConfig.Roles[role]
	if !ok {
		return "", "", ""
	}
	return r.Primary, r.Fallback, r.Economy
}

// IsExemptFromAutoDowngrade returns true if role is in cost_optimization.exempt_roles.
func IsExemptFromAutoDowngrade(role string) bool {
	configMu.RLock()
	defer configMu.RUnlock()
	if loadedConfig == nil || loadedConfig.CostOptimization.ExemptRoles == nil {
		return false
	}
	for _, r := range loadedConfig.CostOptimization.ExemptRoles {
		if r == role {
			return true
		}
	}
	return false
}

// AutoDowngradeThreshold returns cost_optimization.auto_downgrade_at_pct (0-100).
func AutoDowngradeThreshold() float64 {
	configMu.RLock()
	defer configMu.RUnlock()
	if loadedConfig == nil {
		return 0
	}
	return loadedConfig.CostOptimization.AutoDowngradeAtPct
}

// RoleDefaultModel returns primary model for role from config, or built-in RoleDefaultModels, or DefaultModel.
func RoleDefaultModel(role string) string {
	primary, _, _ := RoleModel(role)
	if primary != "" && AllowedModel(primary) {
		return primary
	}
	if m, ok := roleDefaultModelsBuiltin[role]; ok {
		return m
	}
	return DefaultModel()
}

// ResolveFallbackSequenceFromRole returns 3-tier sequence for role: primary → fallback → economy → escalated.
// Uses config when available; otherwise falls back to ResolveFallbackSequence with DefaultModel.
func ResolveFallbackSequenceFromRole(role, preferred string) []string {
	primary, fallback, economy := RoleModel(role)
	if primary == "" {
		start := preferred
		if start == "" || !AllowedModel(start) {
			start = RoleDefaultModel(role)
		}
		return ResolveFallbackSequence(start)
	}
	start := preferred
	if start == "" || !AllowedModel(start) {
		start = primary
	}
	sequence := []string{start}
	seen := map[string]bool{start: true}
	addIfAllowed := func(m string) {
		if m != "" && !seen[m] && AllowedModel(m) {
			sequence = append(sequence, m)
			seen[m] = true
		}
	}
	// Build 3-tier: primary, fallback, economy
	if start != primary {
		addIfAllowed(primary)
	}
	addIfAllowed(fallback)
	addIfAllowed(economy)
	sequence = append(sequence, "escalated")
	return sequence
}

// roleDefaultModelsBuiltin is the built-in map when no config (used by RoleDefaultModel).
var roleDefaultModelsBuiltin = map[string]string{
	"analyst": "glm-5", "coder": "glm-4.7", "reviewer": "glm-5", "retro": "glm-5",
	"orchestrator": "glm-5",
}

// AllowedModelBuiltin checks against built-in allowlist (used when no config).
func AllowedModelBuiltin(model string) bool {
	if _, ok := allowedProviderModels[model]; ok {
		return true
	}
	_, modelID := ParseProviderModel(model)
	if modelID != "" {
		model = modelID
	}
	_, ok := allowedModels[model]
	return ok
}

// StartReloadWatcher polls path for changes and reloads. Stops when ctx done.
// Run in a goroutine: go policy.StartReloadWatcher(ctx, path, 30*time.Second)
func StartReloadWatcher(ctx context.Context, path string, interval time.Duration) {
	if path == "" || interval <= 0 {
		return
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	lastMod := time.Time{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(abs)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				if cfg, err := LoadFromPath(abs); err == nil && cfg != nil {
					ApplyConfig(cfg)
				}
			}
		}
	}
}
