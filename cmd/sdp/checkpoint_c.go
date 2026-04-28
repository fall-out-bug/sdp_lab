// cmd/sdp/checkpoint_c.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

const (
	resolutionProceedProvisional = "proceed_provisional"
	resolutionDeepDive           = "deep_dive"
	resolutionDowngrade          = "downgrade"
)

// resolveCheckpointC decides how to handle each flagged scan item.
//
// In interactive mode (isInteractive=true), it prompts the user at stdin for
// each flagged item with options [D]eep dive / [P]roceed provisional / [I]gnore.
// In non-interactive mode (isInteractive=false), it silently applies
// "proceed_provisional" for all flagged items.
//
// The reader parameter allows test injection of stdin. Pass nil to use os.Stdin.
//
// Returns a map of item.Name -> resolution string.
func resolveCheckpointC(scan *discovery.ScanResult, isInteractive bool, reader io.Reader) map[string]string {
	resolutions := make(map[string]string)

	flagged := scan.Flagged()
	if len(flagged) == 0 {
		return resolutions
	}

	if !isInteractive {
		for _, item := range flagged {
			resolutions[item.Name] = resolutionProceedProvisional
		}
		return resolutions
	}

	// Interactive: prompt for each flagged item.
	if reader == nil {
		reader = os.Stdin
	}
	scanner := bufio.NewScanner(reader)

	for _, item := range flagged {
		blocking := ""
		if item.DepthFlag != nil && item.DepthFlag.Blocking {
			blocking = " BLOCKING"
		}
		fmt.Printf("\n  %s%s\n", item.Name, blocking)
		if item.DepthFlag != nil {
			fmt.Printf("  reason: %s\n", item.DepthFlag.Reason)
		}
		fmt.Printf("  [D] Deep dive now  [P] Proceed provisional  [I] Downgrade to MONITOR\n")
		fmt.Printf("  Choice (D/P/I, default=P): ")

		choice := "P"
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				choice = strings.ToUpper(line[:1])
			}
		}

		switch choice {
		case "D":
			resolutions[item.Name] = resolutionDeepDive
		case "I":
			resolutions[item.Name] = resolutionDowngrade
		default:
			resolutions[item.Name] = resolutionProceedProvisional
		}
	}
	return resolutions
}

// printResolutionSummary prints what was decided for each flagged item.
func printResolutionSummary(resolutions map[string]string) {
	if len(resolutions) == 0 {
		return
	}
	fmt.Printf("\n  Depth resolutions applied:\n")
	for name, res := range resolutions {
		icon := "->"
		switch res {
		case resolutionDeepDive:
			icon = "search"
		case resolutionDowngrade:
			icon = "down"
		case resolutionProceedProvisional:
			icon = "ok"
		}
		fmt.Printf("    [%s] %s: %s\n", icon, name, res)
	}
}
