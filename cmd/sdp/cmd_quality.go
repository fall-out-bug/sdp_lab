package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func runQuality(args []string) {
	os.Exit(runQualityWithWriters(args, os.Stdout, os.Stderr))
}

func runQualityWithWriters(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("quality", flag.ContinueOnError)
	fs.SetOutput(stderr)
	full := fs.Bool("full", false, "Run full coverage and test/code ratio checks")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "usage: sdp quality [--full]\n")
		return 2
	}

	root, err := findRepoRootForQuality()
	if err != nil {
		fmt.Fprintf(stderr, "sdp quality: %v\n", err)
		return 1
	}
	script := filepath.Join(root, "scripts", "quality-metrics.sh")
	if _, err := os.Stat(script); err != nil {
		fmt.Fprintf(stderr, "sdp quality: quality metrics script unavailable: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, script)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()
	if !*full {
		cmd.Env = append(cmd.Env, "SDP_QUALITY_MATRIX_ONLY=1")
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			fmt.Fprintf(stderr, "sdp quality: timed out: %v\n", ctx.Err())
			return 1
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "sdp quality: %v\n", err)
		return 1
	}
	return 0
}

func findRepoRootForQuality() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "scripts", "quality-metrics.sh")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find scripts/quality-metrics.sh from %s", wd)
		}
		dir = parent
	}
}
