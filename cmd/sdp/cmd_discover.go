// cmd/sdp/cmd_discover.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/control"
	"sdp_dev/internal/discovery"
)

func runDiscover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	outDir := fs.String("out", "docs/discovery", "output directory for artifacts")
	model := fs.String("model", "", "override default LLM model")
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
		discovery.DefaultSynthModel = *model
	}

	client := discovery.NewLLMClient(apiKey, discovery.DefaultOpenRouterBase)
	ctx := context.Background()
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

	// ── Checkpoint C: Depth decisions ─────────────────────────────
	fmt.Println(discovery.RenderCheckpoint(scanResult))
	interactive := isTerminal()
	if !interactive {
		fmt.Printf("   (non-interactive mode — proceeding with defaults for all flagged items)\n\n")
	}
	resolutions := resolveCheckpointC(scanResult, interactive, nil)
	if interactive && len(resolutions) > 0 {
		printResolutionSummary(resolutions)
		fmt.Println()
	} else if !interactive && len(resolutions) > 0 {
		printResolutionSummary(resolutions)
	}
	if len(resolutions) > 0 {
		scanResult = discovery.ApplyResolutions(scanResult, resolutions)
		session.Scan = scanResult
	}

	// ── Phase 4a: VALIDATE (desk research) ────────────────────────
	fmt.Printf("🔬 Phase 4a: Validating top assumptions...\n")
	validation, err := discovery.Validate(ctx, client, frame, hypothesis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: validate: %v\n", err)
		os.Exit(1)
	}
	session.Validation = validation

	// ── Checkpoint D: Validation verdict ──────────────────────────
	verdictIcon := map[discovery.FinalVerdict]string{
		discovery.VerdictGO:    "✅ GO",
		discovery.VerdictPIVOT: "🔄 PIVOT",
		discovery.VerdictKILL:  "❌ KILL",
	}
	verdictLabel := verdictIcon[validation.FinalVerdict]
	if verdictLabel == "" {
		verdictLabel = string(validation.FinalVerdict)
	}
	fmt.Printf("\n── Checkpoint D — Validation Verdict ──\n\n")
	fmt.Printf("  Verdict:  %s\n", verdictLabel)
	fmt.Printf("  Reason:   %s\n", validation.VerdictReason)
	if validation.PivotSuggestion != "" {
		fmt.Printf("  Pivot to: %s\n", validation.PivotSuggestion)
	}
	if validation.KillReason != "" {
		fmt.Printf("  Kill why: %s\n", validation.KillReason)
	}
	fmt.Printf("  Claims:   %d validated (needs_experiment=%v)\n",
		len(validation.Claims), validation.NeedsExperiment)
	fmt.Printf("  Cost:     $%.5f\n\n", validation.CostUSD)

	// ── Phase 4b: EXPERIMENT (conditional on insufficient_data) ──
	if validation.NeedsExperiment {
		fmt.Printf("🧪 Phase 4b: Designing experiment for unresolved assumptions...\n")
		experiment, err := discovery.GenerateExperiment(ctx, client, frame, validation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   warning: experiment design: %v\n", err)
		} else {
			session.Experiment = experiment
			fmt.Printf("   format:  %s\n", experiment.Format)
			fmt.Printf("   metric:  %s\n", experiment.SuccessMetric)
			fmt.Printf("   timebox: %d days\n", experiment.TimeBoxDays)
			fmt.Printf("   cost:    $%.5f\n\n", experiment.CostUSD)
		}
	}

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

	// ── Create experiment beads issue (if Phase 4b ran) ───────────
	if session.Experiment != nil {
		fmt.Printf("\n📌 Creating experiment issue...\n")
		expID, err := createExperimentIssue(idea, session.Experiment, absOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "   warning: could not create experiment issue: %v\n", err)
		} else {
			fmt.Printf("   created: %s\n", expID)
		}
	}

	// ── Create beads issue ─────────────────────────────────────────
	fmt.Printf("\n📌 Creating beads issue...\n")
	issueID, err := createDiscoveryIssue(idea, frame, hypothesis, validation, absOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   warning: could not create beads issue: %v\n", err)
	} else {
		fmt.Printf("   created: %s\n", issueID)
	}

	// On GO verdict: create FeatureCard and workstream stub
	if validation != nil && validation.FinalVerdict == discovery.VerdictGO {
		fmt.Printf("\nCreating FeatureCard...\n")
		slug := strings.ToLower(strings.ReplaceAll(idea, " ", "-"))
		projectRoot := filepath.Dir(filepath.Dir(absOut))
		store, storeErr := control.Open(projectRoot)
		if storeErr != nil {
			fmt.Fprintf(os.Stderr, "   warning: could not open control store: %v\n", storeErr)
		} else {
			card, cardErr := createFeatureCard(store, "sdp", slug, frame, hypothesis, absOut)
			if cardErr != nil {
				fmt.Fprintf(os.Stderr, "   warning: could not create feature card: %v\n", cardErr)
			} else {
				fmt.Printf("   feature card: %s\n", card.ID)
			}
		}

		fmt.Printf("\nCreating workstream stub...\n")
		wsDir := filepath.Join(projectRoot, "docs/workstreams/backlog")
		wsID := "00-NEW-01"
		featureID := "FNEW"
		wsPath, wsErr := createWorkstreamStub(wsDir, wsID, featureID, frame, hypothesis, absOut)
		if wsErr != nil {
			fmt.Fprintf(os.Stderr, "   warning: could not create workstream stub: %v\n", wsErr)
		} else {
			fmt.Printf("   workstream stub: %s\n", wsPath)
		}
	}
}

// buildDiscoveryDescription constructs the beads issue description for a discovery run.
// On GO verdict, includes hypothesis requirements for the feature backlog.
func buildDiscoveryDescription(
	idea string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	val *discovery.ValidationResult,
	artifactDir string,
) string {
	verdictSection := ""
	if val != nil {
		verdictSection = fmt.Sprintf("\n\n**Verdict:** %s — %s", val.FinalVerdict, val.VerdictReason)
		if val.PivotSuggestion != "" {
			verdictSection += fmt.Sprintf("\n\n**Pivot:** %s", val.PivotSuggestion)
		}
		if val.KillReason != "" {
			verdictSection += fmt.Sprintf("\n\n**Kill reason:** %s", val.KillReason)
		}
	}

	hypoSection := ""
	if hyp != nil && hyp.WeBelieve != "" {
		riskiest := "—"
		if len(hyp.Assumptions) > 0 {
			riskiest = hyp.Assumptions[0].Statement
		}
		hypoSection = fmt.Sprintf("\n\n**Hypothesis:** %s\n\n**Riskiest assumption:** %s",
			hyp.WeBelieve, riskiest)
	}

	reqSection := ""
	if val != nil && val.FinalVerdict == discovery.VerdictGO && hyp != nil && len(hyp.Requirements) > 0 {
		var sb strings.Builder
		sb.WriteString("\n\n## Requirements\n\n")
		for _, r := range hyp.Requirements {
			fmt.Fprintf(&sb, "- %s\n", r)
		}
		reqSection = sb.String()
	}

	return fmt.Sprintf(
		"## Discovery: %s\n\n**Problem:** %s\n\n**Appetite:** %s%s%s%s\n\n**Artifacts:** %s/",
		idea, frame.ProblemStatement, frame.Appetite,
		hypoSection, verdictSection, reqSection, artifactDir,
	)
}

func createExperimentIssue(idea string, e *discovery.ExperimentBrief, artifactDir string) (string, error) {
	title := fmt.Sprintf("Experiment: %s [%s]", idea, e.Format)
	desc := fmt.Sprintf(
		"## Experiment Brief\n\n**Idea:** %s\n\n**Format:** %s\n\n**Objective:** %s\n\n**Hypothesis:** %s\n\n**Success metric:** %s\n\n**Time box:** %d days\n\n**Testing claim:** %s\n\n**Artifacts:** %s/",
		idea, e.Format, e.Objective, e.Hypothesis, e.SuccessMetric, e.TimeBoxDays, e.RawClaim, artifactDir,
	)
	cmd := exec.Command("bd", "create",
		"--title="+title,
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

func createDiscoveryIssue(
	idea string,
	frame *discovery.FrameResult,
	hypothesis *discovery.HypothesisResult,
	validation *discovery.ValidationResult,
	artifactDir string,
) (string, error) {
	issueType := "task"
	title := "Discovery: " + idea
	if validation != nil && validation.FinalVerdict == discovery.VerdictGO {
		issueType = "feature"
		title = "Feature: " + idea
	}

	desc := buildDiscoveryDescription(idea, frame, hypothesis, validation, artifactDir)

	cmd := exec.Command("bd", "create",
		"--title="+title,
		"--description="+desc,
		"--type="+issueType,
		"--priority=2",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bd create: %w: %s", err, out)
	}
	return string(out), nil
}

// createFeatureCard creates a control.FeatureCard from Discovery phase outputs.
// Called only on GO verdict. Returns the saved card with its assigned ID.
func createFeatureCard(
	store *control.Store,
	projectID, slug string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	discoveryDir string,
) (*control.FeatureCard, error) {
	card, err := store.CreateCard(projectID, slug, frame.ProblemStatement)
	if err != nil {
		return nil, fmt.Errorf("create feature card: %w", err)
	}
	card.NormalizedIntent = frame.ProblemStatement
	card.DiscoveryDir = discoveryDir
	card.Status = "shaping"
	card.ScopeIn = []string{frame.Scope}
	if hyp != nil {
		card.AcceptanceShape = hyp.Requirements
	}
	if err := store.SaveCard(card); err != nil {
		return nil, fmt.Errorf("save feature card: %w", err)
	}
	return card, nil
}

// createWorkstreamStub writes a backlog workstream file for the first delivery step.
// Returns the path of the created file.
func createWorkstreamStub(
	wsDir, wsID, featureID string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	discoveryDir string,
) (string, error) {
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return "", fmt.Errorf("create ws dir: %w", err)
	}

	var acLines strings.Builder
	if hyp != nil {
		for _, r := range hyp.Requirements {
			fmt.Fprintf(&acLines, "- [ ] %s\n", r)
		}
	}
	if acLines.Len() == 0 {
		acLines.WriteString("- [ ] (to be defined in planning phase)\n")
	}

	content := fmt.Sprintf(`---
ws_id: %s
feature_id: %s
status: backlog
priority: P2
size: M
depends_on: []
ws_kind: leaf
parent_ws_id: null
dispatch_lifecycle: active
---

# %s: Delivery — Phase 1

Feature: %s

## Goal

%s

## Beads

## Acceptance Criteria

%s
## Out of Scope

%s

## Discovery Reference

Artifacts: %s/

## Implementation Notes

Read Discovery artifacts before planning. Start with the frame and hypothesis docs. Bind a concrete primary Beads issue before moving this leaf to active execution.
`, wsID, featureID, wsID, featureID,
		frame.ProblemStatement,
		acLines.String(),
		frame.Scope,
		discoveryDir,
	)

	path := filepath.Join(wsDir, wsID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write workstream stub: %w", err)
	}
	return path, nil
}
