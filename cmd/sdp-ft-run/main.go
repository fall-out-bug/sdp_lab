//go:build sdp_experimental

// Command sdp-ft-run drives a fine-tune backend: upload → create job → poll
// status. The same flags target either OpenAI or local MLX.
//
// Usage:
//
//	# Dry-run (no upload, no job): print what would be sent.
//	sdp-ft-run --backend openai --train internal/dispatch/training/train.jsonl --dry-run
//
//	# Live OpenAI run (requires OPENAI_API_KEY).
//	sdp-ft-run --backend openai --train ... --suffix sdp-complexity --poll-secs 30
//
//	# Local MLX run (requires mlx_lm CLI on PATH).
//	sdp-ft-run --backend mlx --train ... --base-model mlx-community/Qwen2.5-3B-Instruct-4bit
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/finetune/runner"
)

func main() {
	var (
		backend   = flag.String("backend", "openai", "openai | mlx")
		train     = flag.String("train", "internal/dispatch/training/train.jsonl", "path to JSONL")
		baseModel = flag.String("base-model", "", "base model id (backend-specific default if empty)")
		suffix    = flag.String("suffix", "sdp-complexity", "fine-tune suffix")
		epochs    = flag.Int("epochs", 0, "epochs (0 = backend default)")
		pollEvery = flag.Int("poll-secs", 0, "poll interval in seconds (0 = no polling, exit after CreateJob)")
		maxMins   = flag.Int("max-poll-mins", 60, "max wall-clock minutes to keep polling before giving up")
		dryRun    = flag.Bool("dry-run", false, "print plan and exit")
	)
	flag.Parse()

	r, err := pickRunner(*backend, *dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *dryRun {
		printDryRun(r, *train, *baseModel, *suffix, *epochs, *pollEvery)
		return
	}

	ctx := context.Background()

	fmt.Printf("[%s] uploading %s...\n", r.Name(), *train)
	file, err := r.Upload(ctx, *train)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upload: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[%s] file ref: %+v\n", r.Name(), file)

	fmt.Printf("[%s] creating job...\n", r.Name())
	job, err := r.CreateJob(ctx, file, runner.CreateJobOpts{
		BaseModel: *baseModel,
		Suffix:    *suffix,
		Epochs:    *epochs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create job: %v\n", err)
		os.Exit(1)
	}
	printJob(job)

	if *pollEvery <= 0 {
		fmt.Println("(no polling — exiting)")
		return
	}

	deadline := time.Now().Add(time.Duration(*maxMins) * time.Minute)
	for {
		if time.Now().After(deadline) {
			fmt.Fprintf(os.Stderr, "poll: exceeded --max-poll-mins=%d, last status was %s\n", *maxMins, job.Status)
			os.Exit(2)
		}
		time.Sleep(time.Duration(*pollEvery) * time.Second)
		info, err := r.Poll(ctx, job.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "poll: %v\n", err)
			os.Exit(1)
		}
		printJob(info)
		job = info
		if info.Status == runner.StatusSucceeded ||
			info.Status == runner.StatusFailed ||
			info.Status == runner.StatusCancelled {
			return
		}
	}
}

func pickRunner(name string, dryRun bool) (runner.Runner, error) {
	switch name {
	case "openai":
		r := runner.NewOpenAIRunner("")
		if r.APIKey == "" && !dryRun {
			return nil, fmt.Errorf("OPENAI_API_KEY not set (use --dry-run to test plumbing)")
		}
		return r, nil
	case "mlx":
		return runner.NewMLXRunner(""), nil
	default:
		return nil, fmt.Errorf("unknown backend: %s (want: openai | mlx)", name)
	}
}

func printJob(j runner.JobInfo) {
	b, _ := json.MarshalIndent(j, "", "  ")
	fmt.Println(string(b))
}

func printDryRun(r runner.Runner, train, baseModel, suffix string, epochs, pollEvery int) {
	plan := map[string]any{
		"backend":    r.Name(),
		"train":      train,
		"base_model": baseModel,
		"suffix":     suffix,
		"epochs":     epochs,
		"poll_secs":  pollEvery,
	}
	b, _ := json.MarshalIndent(plan, "", "  ")
	fmt.Println(string(b))
}
