package healthcheck

import (
	"context"
	"fmt"
)

// Runner executes a set of Checkers and collects their results.
type Runner struct {
	cfg      Config
	checkers []Checker
}

// NewRunner constructs a Runner for the given config.
func NewRunner(cfg Config) (*Runner, error) {
	if cfg.ProjectRoot == "" {
		return nil, fmt.Errorf("healthcheck: ProjectRoot must not be empty")
	}
	all := []Checker{
		&goBuildChecker{projectRoot: cfg.ProjectRoot},
		&beadsReadyChecker{projectRoot: cfg.ProjectRoot},
		&gitCleanChecker{projectRoot: cfg.ProjectRoot},
	}
	selected := make([]Checker, 0, len(all))
	for _, ch := range all {
		if cfg.Only == "" || ch.Name() == cfg.Only {
			selected = append(selected, ch)
		}
	}
	if cfg.Only != "" && len(selected) == 0 {
		return nil, fmt.Errorf("healthcheck: unknown check %q", cfg.Only)
	}
	return &Runner{cfg: cfg, checkers: selected}, nil
}

// Run executes all selected checkers and returns their results.
func (r *Runner) Run(ctx context.Context) []CheckResult {
	results := make([]CheckResult, 0, len(r.checkers))
	for _, ch := range r.checkers {
		results = append(results, ch.Run(ctx))
	}
	return results
}
