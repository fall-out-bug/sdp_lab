package bridge

import (
	"encoding/json"
	"fmt"
	"time"
)

// severityToPriority maps finding severity to Beads priority.
func severityToPriority(severity string) int {
	switch severity {
	case "error":
		return 1 // P1
	case "warning":
		return 2 // P2
	case "info":
		return 3 // P3
	default:
		return 4 // P4
	}
}

// truncate truncates a string to maxLen runes, handling multi-byte UTF-8 correctly.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return string(runes[:maxLen-3]) + "..."
	}
	return string(runes[:maxLen])
}

// PrintSummary prints a summary of the sync.
func (s *BeadsSink) PrintSummary() {
	stats := s.GetStats()
	fmt.Printf("\nSync Summary:\n")
	fmt.Printf("  Processed: %d\n", stats.Processed)
	fmt.Printf("  Created:   %d\n", stats.Created)
	fmt.Printf("  Updated:   %d\n", stats.Updated)
	fmt.Printf("  Skipped:   %d\n", stats.Skipped)
	fmt.Printf("  Failed:    %d\n", stats.Failed)
}

// GenerateReport generates a JSON report of the sync.
func (s *BeadsSink) GenerateReport() ([]byte, error) {
	report := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.GetStats(),
	}
	return json.MarshalIndent(report, "", "  ")
}
