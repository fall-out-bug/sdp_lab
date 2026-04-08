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

	// ── Checkpoint A: Typed clarifications (non-blocking) ──────────
	fmt.Printf("💬 Checkpoint A: Generating clarifications...\n")
	clarifications, err := discovery.GenerateClarifications(ctx, client, frame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   warning: clarifications: %v\n", err)
	} else if len(clarifications) > 0 {
		fmt.Printf("\n── Clarifications (refine idea before continuing) ──\n\n")
		for i, clr := range clarifications {
			fmt.Printf("  %d. [%s] %s\n", i+1, clr.Type, clr.Question)
			if clr.Context != "" {
				fmt.Printf("     context: %s\n", clr.Context)
			}
			if len(clr.Options) > 0 {
				for j, opt := range clr.Options {
					fmt.Printf("     [%c] %s\n", 'A'+j, opt)
				}
			}
		}
		fmt.Printf("\n   (proceeding with defaults — re-run with refined idea to update)\n\n")
	}

	// ── Phase 2: HYPOTHESIZE ────────────────────────────────────────
	fmt.Printf("💡 Phase 2: Generating hypothesis...\n")
	hypothesis, err := discovery.Hypothesize(ctx, client, frame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: hypothesize: %v\n", err)
		os.Exit(1)
	}
	session.Hypothesis = hypothesis
	fmt.Printf("   we believe: %s\n", hypothesis.WeBelieve)
	if len(hypothesis.Assumptions) > 0 {
		fmt.Printf("   riskiest:   %s (RAT=%.0f)\n",
			hypothesis.Assumptions[0].Statement,
			hypothesis.Assumptions[0].RATScore)
	}
	fmt.Printf("\n")

	// ── Checkpoint B: Hypothesis summary ───────────────────────────
	fmt.Printf("── Checkpoint B — Hypothesis ──\n\n")
	fmt.Printf("  Test Card:\n")
	fmt.Printf("    We believe: %s\n", hypothesis.WeBelieve)
	fmt.Printf("    To verify:  %s\n", hypothesis.ToVerify)
	fmt.Printf("    Measure:    %s\n", hypothesis.WeMeasure)
	fmt.Printf("    Right if:   %s\n\n", hypothesis.WeAreRightIf)
	if len(hypothesis.Assumptions) > 0 {
		fmt.Printf("  RAT-ranked assumptions:\n")
		for _, a := range hypothesis.Assumptions {
			fmt.Printf("    %d. [RAT=%.0f] %s (%s risk, %s uncertainty)\n",
				a.RATRank, a.RATScore, a.Statement, a.RiskLevel, a.Uncertainty)
		}
		fmt.Printf("\n")
	}

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

	// ── Checkpoint C: Depth decisions ──────────────────────────────
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
	issueID, err := createDiscoveryIssue(idea, frame, hypothesis, absOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   warning: could not create beads issue: %v\n", err)
	} else {
		fmt.Printf("   created: %s\n", issueID)
	}
}

func createDiscoveryIssue(idea string, frame *discovery.FrameResult,
	hypothesis *discovery.HypothesisResult, artifactDir string) (string, error) {

	hypoSection := ""
	if hypothesis != nil && hypothesis.WeBelieve != "" {
		riskiest := "—"
		if len(hypothesis.Assumptions) > 0 {
			riskiest = hypothesis.Assumptions[0].Statement
		}
		hypoSection = fmt.Sprintf("\n\n**Hypothesis:** %s\n\n**Riskiest assumption:** %s",
			hypothesis.WeBelieve, riskiest)
	}

	desc := fmt.Sprintf(
		"## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s%s\n\n**Artifacts:** %s/",
		idea, frame.ProblemStatement, frame.Appetite, hypoSection, artifactDir,
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
