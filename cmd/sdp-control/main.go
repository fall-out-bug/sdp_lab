package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/control"
	"sdp_dev/internal/orchestrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "card-create":
		runCardCreate(os.Args[2:])
	case "card-clarify":
		runCardClarify(os.Args[2:])
	case "card-needs-input":
		runCardNeedsInput(os.Args[2:])
	case "card-ready":
		runCardReady(os.Args[2:])
	case "card-park":
		runCardPark(os.Args[2:])
	case "card-execute":
		runCardExecute(os.Args[2:])
	case "card-feedback":
		runCardFeedback(os.Args[2:])
	case "card-feedback-export":
		runCardFeedbackExport(os.Args[2:])
	case "card-message-export":
		runCardMessageExport(os.Args[2:])
	case "card-resume":
		runCardResume(os.Args[2:])
	case "card-resume-import":
		runCardResumeImport(os.Args[2:])
	case "card-reply-ingest":
		runCardReplyIngest(os.Args[2:])
	case "board-build":
		runBoardBuild(os.Args[2:])
	case "board-show":
		runBoardShow(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp-control <card-create|card-clarify|card-needs-input|card-ready|card-park|card-execute|card-feedback|card-feedback-export|card-message-export|card-resume|card-resume-import|card-reply-ingest|board-build|board-show> [flags]")
}

func openStore() *control.Store {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err)
		os.Exit(1)
	}
	root, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err)
		os.Exit(1)
	}
	store, err := control.Open(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err)
		os.Exit(1)
	}
	return store
}

func runCardCreate(args []string) {
	fs := flag.NewFlagSet("card-create", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	title := fs.String("title", "", "card title")
	raw := fs.String("raw", "", "raw request text")
	_ = fs.Parse(args)
	if *project == "" || *title == "" || *raw == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --title, and --raw are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.CreateCard(*project, *title, *raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardClarify(args []string) {
	fs := flag.NewFlagSet("card-clarify", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	intent := fs.String("intent", "", "normalized intent")
	taskType := fs.String("task-type", "", "task type")
	targetRepo := fs.String("target-repo", "", "target repo")
	risk := fs.String("risk", "", "risk level")
	next := fs.String("next", "", "recommended next step")
	scopeIn := fs.String("scope-in", "", "semicolon-separated scope_in items")
	scopeOut := fs.String("scope-out", "", "semicolon-separated scope_out items")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.ClarifyCard(*project, *id, *intent, *taskType, *targetRepo, *risk, *next, splitList(*scopeIn), splitList(*scopeOut))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: clarify card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardNeedsInput(args []string) {
	fs := flag.NewFlagSet("card-needs-input", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	needs := fs.String("needs", "", "semicolon-separated audience: author;admin;human")
	feedback := fs.String("feedback", "", "semicolon-separated feedback questions")
	decision := fs.String("decision", "", "semicolon-separated decisions required")
	update := fs.String("update", "", "semicolon-separated author updates")
	admin := fs.String("admin-action", "", "semicolon-separated admin actions required")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.MarkNeedsInput(*project, *id, splitList(*needs), splitList(*feedback), splitList(*decision), splitList(*update), splitList(*admin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: mark needs-input: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardReady(args []string) {
	fs := flag.NewFlagSet("card-ready", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.MarkReady(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: mark ready: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardPark(args []string) {
	fs := flag.NewFlagSet("card-park", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	reason := fs.String("reason", "", "park reason")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.ParkCard(*project, *id, *reason)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: park card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardExecute(args []string) {
	fs := flag.NewFlagSet("card-execute", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.ExecuteCard(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: execute card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardFeedbackExport(args []string) {
	fs := flag.NewFlagSet("card-feedback-export", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	path := fs.String("output", "", "output file path (required)")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *path == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --output are required")
		os.Exit(2)
	}
	store := openStore()
	packet, err := store.ExportFeedbackPacket(*project, *id, *path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export feedback packet: %v\n", err)
		os.Exit(1)
	}
	printJSON(packet)
}

func runBoardBuild(args []string) {
	fs := flag.NewFlagSet("board-build", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	_ = fs.Parse(args)
	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err)
			os.Exit(1)
		}
		printJSON(snap)
		return
	}
	port, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}
	printJSON(port)
}

func runBoardShow(args []string) {
	runBoardBuild(args)
}

func runCardFeedback(args []string) {
	fs := flag.NewFlagSet("card-feedback", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	packet, err := store.GenerateFeedbackPacket(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate feedback packet: %v\n", err)
		os.Exit(1)
	}
	printJSON(packet)
}

func runCardResume(args []string) {
	fs := flag.NewFlagSet("card-resume", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	answers := fs.String("answers", "", "semicolon-separated answers to feedback questions")
	decisions := fs.String("decisions", "", "semicolon-separated decision answers")
	updates := fs.String("updates", "", "semicolon-separated author updates")
	adminActions := fs.String("admin-actions", "", "semicolon-separated admin actions taken")
	unblock := fs.String("unblock", "", "semicolon-separated blocking reasons resolved")
	targetStatus := fs.String("target-status", "", "target status (clarifying or ready, default: clarifying)")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	answer := &control.FeedbackAnswer{
		FeedbackAnswers:    splitList(*answers),
		DecisionAnswers:    splitList(*decisions),
		AuthorUpdates:      splitList(*updates),
		AdminActions:       splitList(*adminActions),
		UnblockReasons:     splitList(*unblock),
		ResumeTargetStatus: *targetStatus,
	}
	store := openStore()
	card, err := store.ApplyFeedback(*project, *id, answer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: apply feedback: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardResumeImport(args []string) {
	fs := flag.NewFlagSet("card-resume-import", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	path := fs.String("input", "", "input file path (required)")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *path == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --input are required")
		os.Exit(2)
	}
	answer, err := control.ImportFeedbackAnswer(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: import feedback answer: %v\n", err)
		os.Exit(1)
	}
	store := openStore()
	card, err := store.ApplyFeedback(*project, *id, answer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: apply feedback: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func runCardMessageExport(args []string) {
	fs := flag.NewFlagSet("card-message-export", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	targetRole := fs.String("target-role", "human", "target role (default: human)")
	path := fs.String("output", "", "output file path (required)")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *path == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --output are required")
		os.Exit(2)
	}
	store := openStore()
	envelope, err := store.ExportOutboundMessage(*project, *id, *targetRole)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: export outbound message: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal envelope: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write envelope to %s: %v\n", *path, err)
		os.Exit(1)
	}
	fmt.Printf("Exported outbound message envelope to %s (correlation_id: %s)\n", *path, envelope.CorrelationID)
	printJSON(envelope)
}

func runCardReplyIngest(args []string) {
	fs := flag.NewFlagSet("card-reply-ingest", flag.ExitOnError)
	path := fs.String("input", "", "input file path (required)")
	_ = fs.Parse(args)
	if *path == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(2)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read reply envelope from %s: %v\n", *path, err)
		os.Exit(1)
	}

	var envelope control.InboundReplyEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse reply envelope: %v\n", err)
		os.Exit(1)
	}

	store := openStore()
	card, err := store.IngestReply(&envelope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: ingest reply: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ingested reply for card %s (correlation_id: %s)\n", envelope.CardID, envelope.CorrelationID)
	printJSON(card)
}
