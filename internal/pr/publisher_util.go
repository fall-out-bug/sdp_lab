package pr

import (
	"sort"
	"strings"
)

func getAtPath(payload map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = payload
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := m[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func getStringAtPath(payload map[string]any, path string) (string, bool) {
	v, ok := getAtPath(payload, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok
}

func isZeroValue(v any) bool {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value) == ""
	case []string:
		return len(value) == 0
	case []any:
		return len(value) == 0
	case nil:
		return true
	}
	return false
}

func copySignals(in []GateSignal) []map[string]string {
	out := make([]map[string]string, 0, len(in))
	for _, signal := range in {
		out = append(out, map[string]string{
			"name":   strings.TrimSpace(signal.Name),
			"status": strings.TrimSpace(signal.Status),
		})
	}
	return out
}

func cloneHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		out[k] = v
	}
	return out
}

func orderRecipients(recipients []CallbackRecipientTarget, routeMode string) []CallbackRecipientTarget {
	ordered := append([]CallbackRecipientTarget(nil), recipients...)
	if routeMode == "fanout-all" {
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
		return ordered
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Required != ordered[j].Required {
			return ordered[i].Required
		}
		return ordered[i].ID < ordered[j].ID
	})
	return ordered
}
