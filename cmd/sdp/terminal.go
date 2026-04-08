// cmd/sdp/terminal.go
package main

import "os"

// isTerminal returns true if stdout is connected to an interactive terminal.
// Copies the pattern from sdp/sdp-plugin/internal/ui/colors.go.
func isTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
