package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/deploy"
)

func runDeploy(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sdp deploy <staging|prod|rollback> [args...] [--staged] [--skip-gates]")
		os.Exit(2)
	}

	ctx := context.Background()

	// Parse flags.
	staged := false
	skipGates := false
	var cleanArgs []string
	for _, a := range args[1:] {
		switch a {
		case "--staged":
			staged = true
		case "--skip-gates":
			skipGates = true
		default:
			cleanArgs = append(cleanArgs, a)
		}
	}

	target := args[0]
	projectRoot := "."
	if len(cleanArgs) > 1 {
		projectRoot = cleanArgs[1]
	}

	cfg := deploy.DefaultConfig(projectRoot)
	gates := deploy.DefaultGatesConfig()
	gates.Staged = staged

	// Run gates before any deploy action (unless explicitly skipped or rollback).
	// Rollback is an emergency recovery path — gates must not block it.
	if !skipGates && target != "rollback" {
		fmt.Println("Checking deploy gates...")
		results, err := deploy.CheckGates(ctx, cfg, gates)
		for _, gr := range results {
			status := "PASS"
			if !gr.Passed {
				status = "FAIL"
			}
			fmt.Printf("  %s %s: %s\n", status, gr.Gate, gr.Message)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nDeploy blocked: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All gates passed.")
	}

	var result *deploy.Result
	var err error

	switch target {
	case "staging":
		commitHash := "latest"
		if len(cleanArgs) > 0 {
			commitHash = cleanArgs[0]
		}
		fmt.Printf("Deploying to staging (commit: %s)...\n", commitHash)
		result, err = deploy.Staging(ctx, cfg, commitHash)
	case "prod":
		if len(cleanArgs) < 1 || cleanArgs[0] == "" {
			fmt.Fprintln(os.Stderr, "usage: sdp deploy prod <image_tag> [--staged] [--skip-gates]")
			os.Exit(2)
		}
		imageTag := cleanArgs[0]
		if staged {
			fmt.Printf("Staged rollout to production (image: %s)...\n", imageTag)
			result, err = deploy.StagedRollout(ctx, cfg, gates, imageTag)
		} else {
			fmt.Printf("Deploying to production (image: %s)...\n", imageTag)
			result, err = deploy.Production(ctx, cfg, imageTag)
		}
	case "rollback":
		if len(cleanArgs) < 1 {
			fmt.Fprintln(os.Stderr, "usage: sdp deploy rollback <tag>")
			os.Exit(2)
		}
		previousTag := cleanArgs[0]
		fmt.Printf("Rolling back to %s...\n", previousTag)
		result, err = deploy.Rollback(ctx, cfg, previousTag)
	default:
		fmt.Fprintf(os.Stderr, "unknown deploy target: %s (use staging|prod|rollback)\n", target)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Deploy failed: %v\n", err)
		if result != nil && result.Error != "" {
			fmt.Fprintf(os.Stderr, "   %s\n", result.Error)
		}
		os.Exit(1)
	}

	fmt.Printf("Deploy %s complete\n", target)
	fmt.Printf("   Image: %s\n", result.ImageTag)
	fmt.Printf("   Duration: %s\n", result.Duration)
	if result.SmokeTest != nil {
		emoji := "PASS"
		if !result.SmokeTest.Passed {
			emoji = "FAIL"
		}
		fmt.Printf("   Smoke test: %s (exit %d)\n", emoji, result.SmokeTest.ExitCode)
	}
	if result.Health != nil {
		emoji := "PASS"
		if !result.Health.Passed {
			emoji = "FAIL"
		}
		fmt.Printf("   Health: %s (%.1f min)\n", emoji, result.Health.Minutes)
	}
	for _, c := range result.Containers {
		fmt.Printf("   %s [%s] %s\n", c.Name, c.ID[:min(len(c.ID), 12)], c.Status)
	}

	printJSON(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
