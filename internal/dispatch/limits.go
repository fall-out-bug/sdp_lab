package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

// AvailabilityFactor returns a multiplier in [0.0, 1.0] reflecting how much
// headroom the provider has left under its limits.
//
// Bucketing rules (based on UsagePercent ratio):
//   - nil or Total<=0  → 1.0  (unknown = no constraint)
//   - 0–70% used       → 1.0
//   - 71–90% used      → 0.5
//   - 91–99% used      → 0.1
//   - 100%+ used       → 0.0
func AvailabilityFactor(limits *harness.Limits) float64 {
	if limits == nil || limits.Total <= 0 {
		return 1.0
	}

	usage := limits.UsagePercent() // ratio in [0.0, 1.0+]

	switch {
	case usage <= 0.70:
		return 1.0
	case usage <= 0.90:
		return 0.5
	case usage < 1.0:
		return 0.1
	default:
		return 0.0
	}
}

// LimitsChecker queries a set of providers and aggregates their limits.
type LimitsChecker struct {
	Providers []harness.Provider
}

// CheckAll queries every provider and returns a map of provider name to limits.
// Providers that return an error are logged at Warn level and skipped.
func (lc *LimitsChecker) CheckAll(ctx context.Context) map[string]*harness.Limits {
	result := make(map[string]*harness.Limits, len(lc.Providers))

	for _, p := range lc.Providers {
		limits, err := p.CheckLimits(ctx)
		if err != nil {
			slog.Warn("limits check failed", "provider", p.Name(), "error", err)
			continue
		}
		result[p.Name()] = limits
	}

	return result
}

// usageStatus maps a usage ratio to a human-readable status label.
func usageStatus(usage float64) string {
	switch {
	case usage >= 1.0:
		return "exhausted"
	case usage > 0.90:
		return "critical"
	case usage > 0.70:
		return "warning"
	default:
		return "ok"
	}
}

// FormatLimitsTable returns a human-readable table of provider limits.
// Columns: Provider, Used, Total, Window, Status.
func FormatLimitsTable(limits map[string]*harness.Limits) string {
	var sb strings.Builder

	header := fmt.Sprintf("%-20s  %8s  %8s  %8s  %s\n", "Provider", "Used", "Total", "Window", "Status")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(strings.TrimRight(header, "\n"))) + "\n")

	// Sort provider names for deterministic output.
	names := make([]string, 0, len(limits))
	for n := range limits {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		l := limits[name]
		status := usageStatus(l.UsagePercent())
		fmt.Fprintf(&sb, "%-20s  %8d  %8d  %8s  %s\n",
				name, l.Used, l.Total, l.Window, status)
	}

	return sb.String()
}
