// cmd/sdp/cmd_discover.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sdp_dev/internal/discovery"
)

func runDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	outDir := fs.String("out", "docs/discovery", "output directory for artifacts")
	model  := fs.String("model", "", "override default LLM model")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp discover [--out DIR] [--model MODEL] \"raw idea\"")
		os.Exit(2)
	}
	idea := fs.Arg(0)

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: OPENROUTER_API_KEY not set")
		os.Exit(1)
	}

	if *model != "" {
		discovery.DefaultPlannerModel = *model
		discovery.DefaultSynthModel   = *model
	}

	client := discovery.NewLLMClient(apiKey, discovery.DefaultOpenRouterBase)
	ctx    := context.Background()
	session := discovery.NewSession(idea)

	// ── Phase 1: FRAME ─────────────────────────────────────────────
	fmt.Printf("🔍 Phase 1: Framing idea...\n")
	frame, err := discovery.Frame(ctx, client, idea)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: frame: %v\n", err)
		os.Exit(1)
	}
	session.Frame = frame
	fmt.Printf("   problem:  %s\n", frame.ProblemStatement)
	fmt.Printf("   appetite: %s\n\n", frame.Appetite)

	// ── Phase 3: SCAN ──────────────────────────────────────────────
	fmt.Printf("🔍 Phase 3: Scanning market...\n")
	scanResult, err := discovery.Scan(ctx, client, frame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: scan: %v\n", err)
		os.Exit(1)
	}
	session.Scan = scanResult
	fmt.Printf("   found %d items (settled=%d flagged=%d)\n",
		len(scanResult.Items),
		len(scanResult.Settled()),
		len(scanResult.Flagged()))
	fmt.Printf("   cost: $%.5f\n\n", scanResult.CostUSD)

	// ── Checkpoint ─────────────────────────────────────────────────
	fmt.Println(discovery.RenderCheckpoint(scanResult))

	// ── Write artifacts ────────────────────────────────────────────
	absOut, err := filepath.Abs(*outDir)
	if err != nil {
		absOut = *outDir
	}
	if err := discovery.WriteArtifacts(absOut, session); err != nil {
		fmt.Fprintf(os.Stderr, "error: write artifacts: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📁 Artifacts written to %s/\n", absOut)

	// ── Create beads issue ─────────────────────────────────────────
	fmt.Printf("\n📌 Creating beads issue...\n")
	issueID, err := createDiscoveryIssue(idea, frame, absOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   warning: could not create beads issue: %v\n", err)
	} else {
		fmt.Printf("   created: %s\n", issueID)
	}
}

func createDiscoveryIssue(idea string, frame *discovery.FrameResult, artifactDir string) (string, error) {
	desc := fmt.Sprintf(
		"## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s\n\n**Artifacts:** %s/",
		idea, frame.ProblemStatement, frame.Appetite, artifactDir,
	)
	cmd := exec.Command("bd", "create",
		"--title=Discovery: "+idea,
		"--description="+desc,
		"--type=task",
		"--priority=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create: %w: %s", err, out)
	}
	return string(out), nil
}
