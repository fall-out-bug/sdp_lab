// evaluator-orchestrator runs the evaluator swarm cycle: load plan, build execution packet,
// spawn persona tasks (via kubeopencode when available), assemble score report, inject Beads tasks.
//
// Usage: evaluator-orchestrator [--issue <id>] [--work-dir .] [--max-inject 3]
package main

import (
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/evaluator"
)

func main() {
	issueID := flag.String("issue", "", "Source issue ID for evaluation")
	workDir := flag.String("work-dir", "", "Working directory (default: cwd)")
	maxInject := flag.Int("max-inject", 3, "Max Beads tasks to create from recommendations")
	flag.Parse()

	wd := *workDir
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	plan := evaluator.LoadSwarmPlanFromRegistry(wd)
	if *issueID == "" {
		fmt.Println("No issue specified; use --issue <id>")
		return
	}

	packet, err := evaluator.BuildPersonaExecutionPacket(*issueID, plan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build packet: %v\n", err)
		os.Exit(1)
	}

	// Stub: in full implementation, spawn kubeopencode Tasks per persona and collect scores
	// For now, create synthetic scores and inject
	scores := []evaluator.PersonaScore{
		{PersonaID: "systems-architect", Score: 85, Recommendation: "Consider adding boundary documentation"},
		{PersonaID: "sre", Score: 80, Recommendation: "Add runbook for rollback"},
	}
	report := evaluator.AssembleSwarmScoreReport(packet, scores)

	if len(report.PriorityRecommendations) == 0 {
		fmt.Println("No recommendations to inject")
		return
	}

	created, err := evaluator.RecommendationsToBeadsTasks(wd, report.PriorityRecommendations, *maxInject)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inject: %v\n", err)
		os.Exit(1)
	}
	for _, id := range created {
		fmt.Printf("Injected: %s\n", id)
	}
}
