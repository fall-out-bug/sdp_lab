package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "card":
		runCard(os.Args[2:])
	case "board":
		runBoard(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
	case "dispatch":
		runDispatch(os.Args[2:])
	case "result":
		runResult(os.Args[2:])
	case "orchestrate":
		runOrchestrate(os.Args[2:])
	case "attention":
		runAttention(os.Args[2:])
	case "why":
		runWhy(os.Args[2:])
	case "next":
		runNext(os.Args[2:])
	case "missing":
		runMissing(os.Args[2:])
	case "approve":
		runApprove(os.Args[2:])
	case "tower":
		runTower(os.Args[2:])
	case "trace":
		runTrace(os.Args[2:])
	case "deploy":
		runDeploy(os.Args[2:])
	case "intent":
		runIntent(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "stuck":
		runStuck(os.Args[2:])
	case "eval":
		runEval(os.Args[2:])
	case "clarify":
		runClarify(os.Args[2:])
	case "plan":
		runPlan(os.Args[2:])
	case "approve-plan":
		runApprovePlan(os.Args[2:])
	case "discover":
		runDiscover(os.Args[2:])
	case "architect":
		runArchitect(os.Args[2:])
	case "scout":
		runScout(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp <card|board|doctor|dispatch|result|orchestrate|attention> <subcommand> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Card commands:")
	fmt.Fprintln(os.Stderr, "  sdp card <create|show|clarify|needs-input|ready|park|execute|heartbeat|feedback|feedback-export|message-export|resume|resume-import|reply-ingest|deliver>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Board commands:")
	fmt.Fprintln(os.Stderr, "  sdp board <build|show>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Doctor commands:")
	fmt.Fprintln(os.Stderr, "  sdp doctor control")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Dispatch commands:")
	fmt.Fprintln(os.Stderr, "  sdp dispatch card")
	fmt.Fprintln(os.Stderr, "  sdp dispatch next")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Result commands:")
	fmt.Fprintln(os.Stderr, "  sdp result ingest")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Orchestrate commands:")
	fmt.Fprintln(os.Stderr, "  sdp orchestrate once")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Query commands (require beads/dual mode):")
	fmt.Fprintln(os.Stderr, "  sdp why <card-id>           Show why a card is blocked")
	fmt.Fprintln(os.Stderr, "  sdp next [--limit N]        Show next actionable items (default 10)")
	fmt.Fprintln(os.Stderr, "  sdp missing [project-id]    Show items lacking evidence")
	fmt.Fprintln(os.Stderr, "  sdp approve <card-id>       Resolve a human gate")
	fmt.Fprintln(os.Stderr, "  sdp trace <card-id>         Show full feature trace")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Deploy commands:")
	fmt.Fprintln(os.Stderr, "  sdp deploy staging [project-root]")
	fmt.Fprintln(os.Stderr, "  sdp deploy prod <staging-image-tag> [project-root]")
	fmt.Fprintln(os.Stderr, "  sdp deploy rollback <previous-tag> [project-root]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Discovery commands (Stage 0):")
	fmt.Fprintln(os.Stderr, "  sdp discover \"raw idea\"    Run discovery pipeline (FRAME + SCAN + checkpoint)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Pipeline commands:")
	fmt.Fprintln(os.Stderr, "  sdp intent \"description\"   Create intake card from raw intent")
	fmt.Fprintln(os.Stderr, "  sdp status <card-id>        Show card status and phase")
	fmt.Fprintln(os.Stderr, "  sdp stuck                  Show stuck/long-running cards")
	fmt.Fprintln(os.Stderr, "  sdp eval <card-id>         Run build evaluation manually")
	fmt.Fprintln(os.Stderr, "  sdp clarify <card-id>      Run clarification manually")
	fmt.Fprintln(os.Stderr, "  sdp plan <card-id>         Show plan for a card")
	fmt.Fprintln(os.Stderr, "  sdp approve-plan <card-id> Approve a pending plan")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Other:")
	fmt.Fprintln(os.Stderr, "  sdp attention")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Scout commands:")
	fmt.Fprintln(os.Stderr, "  sdp scout [--format json|text|card] [--output DIR] <repo-path>")
}
