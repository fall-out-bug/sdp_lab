package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// secretKeys are keys whose values should be redacted.
var secretKeys = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private_key", "credential", "auth_token", "access_token",
	"refresh_token", "database_url",
}

// ExtractConfigParameters scans YAML/JSON config files for SLA parameters.
func ExtractConfigParameters(root string) ([]SLAParam, error) {
	var params []SLAParam
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("spec: resolve path: %w", err)
	}
	filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yaml", ".yml":
			p, _ := parseYAMLFile(path)
			params = append(params, p...)
		case ".json":
			p, _ := parseJSONFile(path)
			params = append(params, p...)
		}
		return nil
	})
	return params, nil
}

func parseYAMLFile(path string) ([]SLAParam, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return flattenConfig(raw, filepath.Base(path), ""), nil
}

func parseJSONFile(path string) ([]SLAParam, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return flattenConfig(raw, filepath.Base(path), ""), nil
}

// flattenConfig recursively walks a config map and extracts SLA parameters.
func flattenConfig(m map[string]interface{}, rel, prefix string) []SLAParam {
	var params []SLAParam
	for key, val := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		if isSecretKey(key) {
			params = append(params, SLAParam{
				Category:     "secret",
				Component:    fullKey,
				Value:        "[REDACTED]",
				Location:     rel,
				Configurable: true,
			})
			continue
		}
		if sub, ok := val.(map[string]interface{}); ok {
			params = append(params, flattenConfig(sub, rel, fullKey)...)
			continue
		}
		p := classifyConfigParam(fullKey, val, rel)
		if p != nil {
			params = append(params, *p)
		}
	}
	return params
}

func isSecretKey(key string) bool {
	low := strings.ToLower(key)
	for _, sk := range secretKeys {
		if low == sk || strings.Contains(low, sk) {
			return true
		}
	}
	return false
}

func classifyConfigParam(key string, val interface{}, rel string) *SLAParam {
	low := strings.ToLower(key)
	strVal := fmt.Sprintf("%v", val)
	p := &SLAParam{Component: key, Value: strVal, Location: rel, Configurable: true}
	switch {
	case strings.Contains(low, "timeout") || strings.Contains(low, "ttl") ||
		strings.Contains(low, "lifetime") || strings.Contains(low, "interval"):
		p.Category = "timeout"
	case strings.Contains(low, "retry") || strings.Contains(low, "backoff"):
		p.Category = "retry"
	case strings.Contains(low, "rate_limit") || strings.Contains(low, "requests_per_second") ||
		strings.Contains(low, "burst") || strings.Contains(low, "rate"):
		p.Category = "rate_limit"
	case strings.Contains(low, "pool") || strings.Contains(low, "max_open") ||
		strings.Contains(low, "max_idle") || strings.Contains(low, "max_connections") ||
		strings.Contains(low, "max_size"):
		p.Category = "resource_pool"
	case strings.Contains(low, "feature") || strings.Contains(low, "flag"):
		p.Category = "feature_flag"
	default:
		return nil
	}
	return p
}
