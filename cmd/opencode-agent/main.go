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
	if _, err := exec.LookPath(binary); err == nil {
		return run(binary)
	}
	return run("go", "run", goPkg)
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
		names := strings.Fields(strings.TrimSpace(string(namesRaw)))
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
	if _, err := run("git", "pull", "--rebase", "origin", branch); err != nil {
		return err
	}
	return nil
}

func runCycle() error {
	if err := syncWorkspace(); err != nil {
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
