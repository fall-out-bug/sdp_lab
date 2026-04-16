package bootstrap

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultVerifyTimeout = 30 * time.Second

// VerifyResult holds the outcome of running a single verification command.
type VerifyResult struct {
	// Command is the full command string that was run.
	Command string `json:"command"`
	// ExitCode is the process exit code. -1 means timeout or execution failure.
	ExitCode int `json:"exit_code"`
	// TimedOut reports whether the command exceeded the timeout.
	TimedOut bool `json:"timed_out"`
	// Output captures combined stdout + stderr (truncated to 2KB).
	Output string `json:"output"`
	// Recovery contains actionable steps to fix the failure.
	Recovery []string `json:"recovery,omitempty"`
}

// VerifyCommands runs each non-empty command from cmds with a timeout and
// returns the results. Commands that are empty strings are skipped.
func VerifyCommands(ctx context.Context, cmds BuildCommands) []VerifyResult {
	return VerifyCommandsWithTimeout(ctx, cmds, defaultVerifyTimeout)
}

// VerifyCommandsWithTimeout runs each command with the specified per-command timeout.
func VerifyCommandsWithTimeout(ctx context.Context, cmds BuildCommands, timeout time.Duration) []VerifyResult {
	var results []VerifyResult

	for _, cmd := range []string{cmds.Build, cmds.Test, cmds.Lint} {
		if cmd == "" {
			continue
		}
		results = append(results, runVerifyCommand(ctx, cmd, timeout))
	}

	return results
}

// runVerifyCommand executes a single command string with a timeout.
// It splits the command into name + args, runs it via os/exec, and captures
// the combined output.
func runVerifyCommand(ctx context.Context, command string, timeout time.Duration) VerifyResult {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return VerifyResult{
			Command:  command,
			ExitCode: -1,
			Output:   "empty command",
		}
	}

	name := parts[0]
	args := parts[1:]

	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()

	result := VerifyResult{
		Command: command,
		Output:  truncateString(string(out), 2048),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
		result.Recovery = buildRecoveryHints(command, "timeout")
		return result
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Recovery = buildRecoveryHints(command, "failure")
		return result
	}

	result.ExitCode = 0
	return result
}

// AllPassed reports whether every verification result succeeded.
func AllPassed(results []VerifyResult) bool {
	for _, r := range results {
		if r.ExitCode != 0 {
			return false
		}
	}
	return true
}

// UnverifiedCommands returns the command strings that did not pass verification.
func UnverifiedCommands(results []VerifyResult) []string {
	var failed []string
	for _, r := range results {
		if r.ExitCode != 0 {
			failed = append(failed, r.Command)
		}
	}
	return failed
}

// FormatVerifyResults renders verification results as human-readable text.
func FormatVerifyResults(results []VerifyResult) string {
	if len(results) == 0 {
		return "No commands to verify.\n"
	}

	var sb strings.Builder
	sb.WriteString("Verification Results:\n")
	for _, r := range results {
		mark := "[ok]"
		if r.ExitCode != 0 {
			if r.TimedOut {
				mark = "[timeout]"
			} else {
				mark = "[fail]"
			}
		}
		sb.WriteString(fmt.Sprintf("  %s %s (exit=%d)\n", mark, r.Command, r.ExitCode))

		if r.ExitCode != 0 && r.Output != "" {
			for _, line := range strings.Split(r.Output, "\n") {
				if line != "" {
					sb.WriteString(fmt.Sprintf("       %s\n", line))
				}
			}
		}

		if len(r.Recovery) > 0 {
			sb.WriteString("    Recovery:\n")
			for _, hint := range r.Recovery {
				sb.WriteString(fmt.Sprintf("      - %s\n", hint))
			}
		}
	}
	return sb.String()
}

// buildRecoveryHints generates actionable recovery steps for a failed command.
func buildRecoveryHints(command, reason string) []string {
	var hints []string

	switch reason {
	case "timeout":
		hints = append(hints,
			fmt.Sprintf("Command '%s' timed out after 30s", command),
			"Possible causes: infinite loop, waiting for user input, or network hang",
			"Fix: run the command manually to diagnose, or increase timeout",
			"Skip: re-run bootstrap with --no-verify to skip verification",
		)
	case "failure":
		hints = append(hints,
			fmt.Sprintf("Command '%s' exited with a non-zero status", command),
		)
		// Language-specific hints. Use HasPrefix for "go" to avoid matching
		// "cargo" which contains the substring "go ".
		if strings.HasPrefix(command, "go ") {
			hints = append(hints,
				"Fix: run 'go build ./...' and 'go test ./...' to see errors",
				"Common issues: missing dependencies (run 'go mod tidy'), compilation errors",
			)
		} else if strings.Contains(command, "npm ") || strings.Contains(command, "yarn ") {
			hints = append(hints,
				"Fix: run 'npm install' then 'npm run build' to see errors",
				"Common issues: missing node_modules, type errors",
			)
		} else if strings.Contains(command, "cargo ") {
			hints = append(hints,
				"Fix: run 'cargo check' and 'cargo test' to see errors",
				"Common issues: missing dependencies, compilation errors",
			)
		} else if strings.Contains(command, "make ") {
			hints = append(hints,
				"Fix: run 'make build && make test' to see errors",
				"Common issues: missing build tools, outdated Makefile targets",
			)
		}

		hints = append(hints,
			"Rollback: 'git checkout -- CLAUDE.md AGENTS.md' to restore previous files",
			"Skip: re-run bootstrap with --no-verify to skip verification",
		)
	}

	return hints
}

// truncateString truncates s to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// contentHash returns a quick fingerprint of content for idempotency comparison.
// Samples prefix, middle, and suffix to detect changes anywhere in large content.
func contentHash(content string) uint64 {
	const (
		prime  = 1099511628211
		offset = 14695981039346656037
	)

	h := uint64(offset)
	data := []byte(content)

	// Hash length.
	h ^= uint64(len(data))
	h *= prime

	// Hash first 256 bytes.
	for i := 0; i < len(data) && i < 256; i++ {
		h ^= uint64(data[i])
		h *= prime
	}

	// Hash middle 256 bytes (catches changes outside prefix/suffix).
	if len(data) > 512 {
		mid := len(data) / 2
		for i := mid; i < len(data) && i < mid+256; i++ {
			h ^= uint64(data[i])
			h *= prime
		}
	}

	// Hash last 256 bytes.
	start := len(data) - 256
	if start < 256 {
		start = 256
	}
	for i := start; i < len(data); i++ {
		h ^= uint64(data[i])
		h *= prime
	}

	return h
}

// ContentChanged reports whether newContent differs from oldContent.
// Uses content hashing for fast comparison.
func ContentChanged(oldContent, newContent string) bool {
	if oldContent == newContent {
		return false
	}
	return contentHash(oldContent) != contentHash(newContent)
}
