package discovery

import (
	"fmt"
	"strings"
)

// RenderCheckpoint produces the two-section scan checkpoint output.
func RenderCheckpoint(result *ScanResult) string {
	var b strings.Builder
	settled := result.Settled()
	flagged := result.Flagged()

	fmt.Fprintf(&b, "\n╔══════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(&b, "  SCAN CHECKPOINT\n")
	fmt.Fprintf(&b, "  %d settled · %d flagged · whitespace: %s\n",
		len(settled), len(flagged), result.Whitespace)
	fmt.Fprintf(&b, "╚══════════════════════════════════════════════════════════╝\n")

	if len(settled) > 0 {
		fmt.Fprintf(&b, "\n── Section A — Settled (coverage ≥ 0.4, no flags) ──\n\n")
		for _, item := range settled {
			icon := dispositionIcon(item.Disposition)
			fmt.Fprintf(&b, "  %s %-30s coverage=%.2f  [%s]\n",
				icon, item.Name, item.CoverageScore, item.Disposition)
			if item.KeyStrength != "" {
				fmt.Fprintf(&b, "     strength: %s\n", item.KeyStrength)
			}
			if item.KeyGap != "" {
				fmt.Fprintf(&b, "     gap:      %s\n", item.KeyGap)
			}
		}
	}

	if len(flagged) > 0 {
		fmt.Fprintf(&b, "\n── Section B — Flagged (require depth decision) ──\n")
		for i, item := range flagged {
			fmt.Fprintf(&b, "\n  %d. %s", i+1, item.Name)
			if item.Stars > 0 {
				fmt.Fprintf(&b, " (%d★)", item.Stars)
			}
			fmt.Fprintf(&b, "\n")
			fmt.Fprintf(&b, "     coverage: %.2f/1.0 · disposition: %s\n",
				item.CoverageScore, item.Disposition)
			fmt.Fprintf(&b, "     reason:   %s\n", item.DepthFlag.Reason)
			if item.DepthFlag.Blocking {
				fmt.Fprintf(&b, "     ⚠️  BLOCKING — pipeline paused until resolved\n")
			}
			fmt.Fprintf(&b, "     options:\n")
			fmt.Fprintf(&b, "       [D] Deep dive now\n")
			fmt.Fprintf(&b, "       [P] Proceed provisional (tagged sdp:scan:unverified)\n")
			fmt.Fprintf(&b, "       [I] Downgrade to MONITOR\n")
		}
	}

	if len(result.RecommendedStack) > 0 {
		fmt.Fprintf(&b, "\n── Recommended stack ──\n")
		for _, s := range result.RecommendedStack {
			fmt.Fprintf(&b, "  • %s\n", s)
		}
	}

	return b.String()
}

func dispositionIcon(d Disposition) string {
	switch d {
	case DispositionAdopt:
		return "🟢"
	case DispositionExtract:
		return "🔵"
	case DispositionInspire:
		return "💡"
	case DispositionMonitor:
		return "👁️ "
	case DispositionIgnore:
		return "⬛"
	default:
		return "  "
	}
}
