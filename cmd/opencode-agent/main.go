package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "SDP_RUNTIME=opencode", "SDP_MODEL=glm-5")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func runComponent(binary string, goPkg string) ([]byte, error) {
	if os.Getenv("SDP_RUNTIME") == "opencode" && (binary == "swarm-worker" || binary == "swarm-reviewer") {
		if _, err := exec.LookPath("go"); err == nil {
			return run("go", "run", goPkg)
		}
	}

	if _, err := exec.LookPath(binary); err == nil {
		return run(binary)
	}
	return run("go", "run", goPkg)
}

func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"could not resolve host",
		"failed to connect",
		"i/o timeout",
		"no such host",
		"connection reset",
		"connection refused",
	}
	for _, needle := range needles {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func preflightGitHubHealth() error {
	if _, err := run("gh", "auth", "status", "--hostname", "github.com"); err != nil {
		return fmt.Errorf("preflight gh auth status: %w", err)
	}

	repoURL := os.Getenv("SDP_REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/fall-out-bug/sdp_private.git"
	}
	if strings.HasPrefix(repoURL, "https://github.com/") {
		token := strings.TrimSpace(os.Getenv("GH_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
		}
		if token != "" {
			repoURL = "https://x-access-token:" + token + "@" + strings.TrimPrefix(repoURL, "https://")
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := run("git", "ls-remote", "--exit-code", repoURL, "HEAD"); err == nil {
			return nil
		} else {
			lastErr = err
			if !isTransientNetworkError(err) || attempt == 3 {
				break
			}
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	return fmt.Errorf("preflight git ls-remote %s: %w", repoURL, lastErr)
}

func syncWorkspace() error {
	if _, err := os.Stat(filepath.Join(".", ".git")); err != nil {
		return nil
	}
	branch := os.Getenv("SDP_REPO_BRANCH")
	if branch == "" {
		branch = "master"
	}

	currentRaw, err := run("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	current := strings.TrimSpace(string(currentRaw))

	dirtyRaw, err := run("git", "status", "--porcelain")
	if err != nil {
		return err
	}
	dirty := strings.TrimSpace(string(dirtyRaw)) != ""

	if dirty {
		namesRaw, err := run("git", "diff", "--name-only")
		if err != nil {
			return err
		}
		cachedRaw, err := run("git", "diff", "--cached", "--name-only")
		if err != nil {
			return err
		}
		nameSet := make(map[string]struct{})
		for _, n := range strings.Fields(strings.TrimSpace(string(namesRaw))) {
			nameSet[n] = struct{}{}
		}
		for _, n := range strings.Fields(strings.TrimSpace(string(cachedRaw))) {
			nameSet[n] = struct{}{}
		}
		names := make([]string, 0, len(nameSet))
		for n := range nameSet {
			names = append(names, n)
		}
		if len(names) > 0 {
			onlyBeadsSyncNoise := true
			for _, n := range names {
				if n != ".beads/metadata.json" && n != ".beads/issues.jsonl" {
					onlyBeadsSyncNoise = false
					break
				}
			}
			if onlyBeadsSyncNoise {
				_, _ = run("git", "reset", "HEAD", "--", ".beads/metadata.json", ".beads/issues.jsonl")
				if _, err := run("git", "checkout", "--", ".beads/metadata.json", ".beads/issues.jsonl"); err != nil {
					return err
				}
				dirty = false
			}
		}
		if len(names) == 1 && names[0] == ".beads/metadata.json" {
			if _, err := run("git", "checkout", "--", ".beads/metadata.json"); err != nil {
				return err
			}
			dirty = false
		}
	}

	if current != branch {
		if dirty {
			return fmt.Errorf("workspace dirty on branch %s; cannot switch to %s", current, branch)
		}
		if _, err := run("git", "checkout", branch); err != nil {
			return err
		}
	}

	if dirty {
		return nil
	}
	if _, err := run("git", "fetch", "origin", branch); err != nil {
		return err
	}
	if _, err := run("git", "rebase", "FETCH_HEAD"); err != nil {
		return err
	}
	return nil
}

func runCycle() error {
	if err := syncWorkspace(); err != nil {
		return err
	}

	if err := preflightGitHubHealth(); err != nil {
		return err
	}

	if _, err := run("bd", "sync", "--import-only"); err != nil {
		return err
	}

	if out, err := runComponent("swarm-worker", "./cmd/swarm-worker"); err != nil {
		return err
	} else {
		fmt.Print(string(out))
	}

	if out, err := runComponent("swarm-reviewer", "./cmd/swarm-reviewer"); err != nil {
		return err
	} else {
		fmt.Print(string(out))
	}

	return nil
}

func main() {
	loop := flag.Bool("loop", false, "Run continuously")
	interval := flag.Duration("interval", 30*time.Second, "Loop interval")
	flag.Parse()

	if !*loop {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	for {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(*interval)
	}
}
