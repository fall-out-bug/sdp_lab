package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/executor"
)

func runStuck(args []string) {
	store := openStore()
	detector := executor.NewStuckDetector(store, executor.DefaultRankingPolicy())
	stuck, err := detector.DetectStuck()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(stuck) == 0 {
		fmt.Println("✅  No stuck cards detected.")
		return
	}
	for _, s := range stuck {
		fmt.Printf("  ⚠️  %s: %s [%s] — %s\n", s.ID, s.Title, s.Status, s.Reason)
	}
}

func runIntent(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp intent \"description\"")
		os.Exit(2)
	}
	raw := strings.TrimSpace(strings.Join(args, " "))
	store := openStore()
	card, err := store.CreateCard("sdp", raw, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create intent card: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Created intake card %s\n", card.ID)
	printJSON(card)
}

func runEval(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp eval <card-id>")
		os.Exit(2)
	}
	store := openStore()
	bridge := executor.NewServeBridge(store, store.ProjectRoot)
	result, err := bridge.Evaluate(context.Background(), args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: evaluate card: %v\n", err)
		os.Exit(1)
	}
	printJSON(result)
}

func runClarify(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp clarify <card-id>")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.LoadCardByID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load card: %v\n", err)
		os.Exit(1)
	}
	bridge := executor.NewServeBridge(store, store.ProjectRoot)
	result, err := bridge.Clarify(context.Background(), card)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: clarify card: %v\n", err)
		os.Exit(1)
	}
	if result.Status == "needs_clarification" {
		fmt.Printf("❓ Card %s needs clarification\n", card.ID)
		for _, q := range result.Questions {
			fmt.Printf(" - %s\n", q)
		}
	} else if result.Card != nil {
		printJSON(result.Card)
		return
	}
	printJSON(result)
}

func runStatus(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp status <card-id>")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.LoadCardByID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load card: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📌 %s: %s\n", card.ID, card.Title)
	fmt.Printf("   Status: %s\n", card.Status)
	if card.TaskType != "" {
		fmt.Printf("   Phase: %s\n", card.TaskType)
	}
	if card.ExecutorRuntimeState != "" {
		fmt.Printf("   Executor: %s\n", card.ExecutorRuntimeState)
	}
	if card.ExecutorResult != nil {
		fmt.Printf("   Result: %s\n", card.ExecutorResult.Status)
	}
	printJSON(card)
}

func runPlan(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp plan <card-id>")
		os.Exit(2)
	}
	store := openStore()
	plan, err := executor.LoadPlan(store.ProjectRoot, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load plan: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📋 Plan for card %s\n", args[0])
	fmt.Printf("   Status: %s\n", plan.Status)
	if plan.Approach != "" {
		fmt.Printf("   Approach: %s\n", plan.Approach)
	}
	if len(plan.FilesToChange) > 0 {
		fmt.Println("   Files to change:")
		for _, f := range plan.FilesToChange {
			fmt.Printf("     - %s\n", f)
		}
	}
	if len(plan.TestsToWrite) > 0 {
		fmt.Println("   Tests to write:")
		for _, t := range plan.TestsToWrite {
			fmt.Printf("     - %s\n", t)
		}
	}
	if plan.RiskAssessment != "" {
		fmt.Printf("   Risk: %s\n", plan.RiskAssessment)
	}
	if plan.EstimatedSteps > 0 {
		fmt.Printf("   Estimated steps: %d\n", plan.EstimatedSteps)
	}
	if plan.ApprovalPending {
		fmt.Println("   ⏳ Awaiting approval")
	}
	printJSON(plan)
}

func runApprovePlan(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp approve-plan <card-id>")
		os.Exit(2)
	}
	store := openStore()
	if err := executor.ApprovePlan(store, store.ProjectRoot, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "error: approve plan: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Plan approved for card %s\n", args[0])
}
