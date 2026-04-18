package healthcheck

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type goBuildChecker struct{ projectRoot string }

func (c *goBuildChecker) Name() string { return "go-build" }

func (c *goBuildChecker) Run(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = c.projectRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return CheckResult{Name: c.Name(), Status: StatusFail, Detail: strings.TrimSpace(string(out))}
	}
	return CheckResult{Name: c.Name(), Status: StatusPass, Detail: "ok"}
}

type beadsReadyChecker struct{ projectRoot string }

func (c *beadsReadyChecker) Name() string { return "beads-ready" }

func (c *beadsReadyChecker) Run(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "bd", "ready", "--json")
	cmd.Dir = c.projectRoot
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Name:   c.Name(),
			Status: StatusFail,
			Detail: fmt.Sprintf("bd ready failed: %s", strings.TrimSpace(string(out))),
		}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" && t != "[]" && t != "null" {
			count++
		}
	}
	return CheckResult{Name: c.Name(), Status: StatusPass, Detail: fmt.Sprintf("%d ready issues", count)}
}

type gitCleanChecker struct{ projectRoot string }

func (c *gitCleanChecker) Name() string { return "git-clean" }

func (c *gitCleanChecker) Run(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = c.projectRoot
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Name: c.Name(), Status: StatusFail, Detail: fmt.Sprintf("git status failed: %v", err)}
	}
	dirty := 0
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(l) != "" {
			dirty++
		}
	}
	if dirty == 0 {
		return CheckResult{Name: c.Name(), Status: StatusPass, Detail: "clean"}
	}
	return CheckResult{Name: c.Name(), Status: StatusWarn, Detail: fmt.Sprintf("working tree dirty (%d files)", dirty)}
}
