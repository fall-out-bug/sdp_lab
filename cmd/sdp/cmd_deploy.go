package main

import (
	"context"
	"fmt"
	"os"

	"sdp_dev/internal/deploy"
)

func runDeploy(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sdp deploy <staging|prod|rollback> [args...]")
		os.Exit(2)
	}

	ctx := context.Background()
	target := args[0]
	projectRoot := "."
	if len(args) > 2 {
		projectRoot = args[2]
	}

	cfg := deploy.DefaultConfig(projectRoot)

	var result *deploy.Result
	var err error

	switch target {
	case "staging":
		commitHash := "latest"
		if len(args) > 1 {
			commitHash = args[1]
		}
		fmt.Printf("🚀 Deploying to staging (commit: %s)...\n", commitHash)
		result, err = deploy.Staging(ctx, cfg, commitHash)
	case "prod":
		imageTag := args[1]
		fmt.Printf("🔥 Deploying to production (image: %s)...\n", imageTag)
		result, err = deploy.Production(ctx, cfg, imageTag)
	case "rollback":
		previousTag := args[1]
		fmt.Printf("⏪ Rolling back to %s...\n", previousTag)
		result, err = deploy.Rollback(ctx, cfg, previousTag)
	default:
		fmt.Fprintf(os.Stderr, "unknown deploy target: %s (use staging|prod|rollback)\n", target)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Deploy failed: %v\n", err)
		if result != nil && result.Error != "" {
			fmt.Fprintf(os.Stderr, "   %s\n", result.Error)
		}
		os.Exit(1)
	}

	fmt.Printf("✅ Deploy %s complete\n", target)
	fmt.Printf("   Image: %s\n", result.ImageTag)
	fmt.Printf("   Duration: %s\n", result.Duration)
	if result.SmokeTest != nil {
		emoji := "✅"
		if !result.SmokeTest.Passed {
			emoji = "❌"
		}
		fmt.Printf("   Smoke test: %s (exit %d)\n", emoji, result.SmokeTest.ExitCode)
	}
	if result.Health != nil {
		emoji := "✅"
		if !result.Health.Passed {
			emoji = "❌"
		}
		fmt.Printf("   Health: %s (%.1f min)\n", emoji, result.Health.Minutes)
	}
	for _, c := range result.Containers {
		fmt.Printf("   📦 %s [%s] %s\n", c.Name, c.ID[:12], c.Status)
	}

	printJSON(result)
}
