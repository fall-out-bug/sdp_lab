package ciloop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefFixer describes a deterministic fixer: command + regex to match failure log.
type DefFixer struct {
	Name      string
	Command   string
	AppliesTo string
	Timeout   int // seconds
}

// builtinFixers are the default deterministic fixers (goimports, go mod tidy, go fmt).
var builtinFixers = []DefFixer{
	{
		Name:      "goimports",
		Command:   "goimports -w .",
		AppliesTo: `could not import|imported and not used|undefined:`,
		Timeout:   30,
	},
	{
		Name:      "go-mod-tidy",
		Command:   "go mod tidy",
		AppliesTo: `missing go\.sum entry|go\.mod file not found|cannot find package`,
		Timeout:   30,
	},
	{
		Name:      "go-fmt",
		Command:   "go fmt ./...",
		AppliesTo: `gofmt|formatting`,
		Timeout:   30,
	},
}

// AutofixerRegistry holds built-in and config-loaded fixers.
type AutofixerRegistry struct {
	Fixers []DefFixer
}

// NewAutofixerRegistry returns a registry with built-ins; optionally loads .sdp/auto-fixers.yaml.
func NewAutofixerRegistry(projectRoot string) *AutofixerRegistry {
	r := &AutofixerRegistry{Fixers: append([]DefFixer{}, builtinFixers...)}
	cfgPath := filepath.Join(projectRoot, ".sdp", "auto-fixers.yaml")
	if data, err := os.ReadFile(cfgPath); err == nil {
		extra, err := ParseAutoFixersYAML(data)
		if err == nil {
			r.Fixers = append(r.Fixers, extra...)
		}
	}
	return r
}

type autoFixersYAML struct {
	Fixers []struct {
		Name      string `yaml:"name"`
		Command   string `yaml:"command"`
		AppliesTo string `yaml:"applies_to"`
		Timeout   int    `yaml:"timeout"`
	} `yaml:"fixers"`
}

// ParseAutoFixersYAML parses .sdp/auto-fixers.yaml format. Exported for testing.
func ParseAutoFixersYAML(data []byte) ([]DefFixer, error) {
	var cfg autoFixersYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	out := make([]DefFixer, 0, len(cfg.Fixers))
	for _, f := range cfg.Fixers {
		if f.Name != "" && f.Command != "" && f.AppliesTo != "" {
			t := f.Timeout
			if t <= 0 {
				t = 30
			}
			out = append(out, DefFixer{Name: f.Name, Command: f.Command, AppliesTo: f.AppliesTo, Timeout: t})
		}
	}
	return out, nil
}

// MatchingFixers returns fixers whose AppliesTo regex matches the failure log.
func (r *AutofixerRegistry) MatchingFixers(failureLog string) []DefFixer {
	var out []DefFixer
	for _, f := range r.Fixers {
		re, err := regexp.Compile(f.AppliesTo)
		if err != nil {
			continue
		}
		if re.MatchString(failureLog) {
			out = append(out, f)
		}
	}
	return out
}

// RunDeterministicFixers runs matching fixers in order. If any produces changes,
// commits and pushes, returns true. Otherwise returns false (fall through to LLM).
// Uses exec directly for fixer commands (need Dir, Stdout, Stderr).
func RunDeterministicFixers(ctx context.Context, projectRoot string, failureLog string, registry *AutofixerRegistry, committer Committer, decisionLogger func(decision, rationale string) error, runFileLogger func(fixerNames []string, duration time.Duration)) (changed bool, err error) {
	matching := registry.MatchingFixers(failureLog)
	if len(matching) == 0 {
		return false, nil
	}

	start := time.Now()
	for _, f := range matching {
		timeout := time.Duration(f.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		parts := SplitCommand(f.Command)
		if len(parts) == 0 {
			cancel()
			continue
		}
		cmd := exec.CommandContext(runCtx, parts[0], parts[1:]...)
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if runErr := cmd.Run(); runErr != nil {
			cancel()
			continue // fixer failed, try next
		}
		cancel()
	}

	// Check if anything changed
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--quiet")
	diffCmd.Dir = projectRoot
	if diffErr := diffCmd.Run(); diffErr == nil {
		return false, nil // no changes
	}

	// Changes produced: commit and push
	msg := fmt.Sprintf("fix(ci): auto-fix %s [deterministic]", strings.Join(func() []string {
		var names []string
		for _, f := range matching {
			names = append(names, f.Name)
		}
		return names
	}(), ", "))
	if err := committer.Commit(msg); err != nil {
		return false, fmt.Errorf("commit after deterministic fix: %w", err)
	}
	if err := committer.Push(); err != nil {
		return false, fmt.Errorf("push after deterministic fix: %w", err)
	}
	if decisionLogger != nil {
		names := make([]string, len(matching))
		for i, f := range matching {
			names[i] = f.Name
		}
		_ = decisionLogger("AUTO-FIX", fmt.Sprintf("Deterministic fixers applied: %s", strings.Join(names, ", ")))
	}
	if runFileLogger != nil {
		names := make([]string, len(matching))
		for i, f := range matching {
			names[i] = f.Name
		}
		runFileLogger(names, time.Since(start))
	}
	return true, nil
}

// SplitCommand splits a command string into executable and args (handles quoted args).
func SplitCommand(s string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"' || r == '\'':
			inQuote = !inQuote
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// DeterministicFirstFixer wraps an inner Fixer: tries deterministic fixers first,
// only invokes inner Fixer if they don't produce changes.
type DeterministicFirstFixer struct {
	ProjectRoot   string
	Registry      *AutofixerRegistry
	Runner        CommandRunner
	Committer     Committer
	LogFetcher    LogFetcher
	DecisionLog   func(decision, rationale string) error
	RunFileLogger func(fixerNames []string, duration time.Duration)
	Inner         Fixer
	PRNumber      int
}

// Fix implements Fixer: tries deterministic fixers first, then inner Fixer.
func (d *DeterministicFirstFixer) Fix(checks []CheckResult) error {
	log, err := d.LogFetcher.FailedLogs(d.PRNumber)
	if err != nil {
		return fmt.Errorf("fetch CI logs: %w", err)
	}

	changed, err := RunDeterministicFixers(context.Background(), d.ProjectRoot, log, d.Registry, d.Committer, d.DecisionLog, d.RunFileLogger)
	if err != nil {
		return err
	}
	if changed {
		return nil
	}

	return d.Inner.Fix(checks)
}
