package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/cli"
	"sdp_dev/internal/control"
)

func runCard(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp card <create|show|clarify|needs-input|ready|park|execute|heartbeat|feedback|feedback-export|message-export|resume|resume-import|reply-ingest|deliver>")
		os.Exit(2)
	}
	switch args[0] {
	case "create":
		runCardCreate(args[1:])
	case "show":
		runCardShow(args[1:])
	case "clarify":
		runCardClarify(args[1:])
	case "needs-input":
		runCardNeedsInput(args[1:])
	case "ready":
		runCardReady(args[1:])
	case "park":
		runCardPark(args[1:])
	case "execute":
		runCardExecute(args[1:])
	case "heartbeat":
		runCardHeartbeat(args[1:])
	case "feedback":
		runCardFeedback(args[1:])
	case "feedback-export":
		runCardFeedbackExport(args[1:])
	case "message-export":
		runCardMessageExport(args[1:])
	case "resume":
		runCardResume(args[1:])
	case "resume-import":
		runCardResumeImport(args[1:])
	case "reply-ingest":
		runCardReplyIngest(args[1:])
	case "deliver":
		runCardDeliver(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp card <create|show|clarify|needs-input|ready|park|execute|heartbeat|feedback|feedback-export|message-export|resume|resume-import|reply-ingest|deliver>")
		os.Exit(2)
	}
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

func runCardShow(args []string) {
	fs := flag.NewFlagSet("card-show", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	asJSON := fs.Bool("json", false, "render raw JSON instead of the default human summary")
	asHTML := fs.Bool("html", false, "render HTML instead of the default human summary")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.LoadCard(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load card: %v\n", err)
		os.Exit(1)
	}
	if *asJSON {
		printJSON(card)
		return
	}
	if *asHTML {
		fmt.Println(cli.RenderCardDetailHTML(card))
		return
	}
	fmt.Println(cli.RenderCardDetail(card))
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

func runCardHeartbeat(args []string) {
	fs := flag.NewFlagSet("card-heartbeat", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	session := fs.String("session", "", "executor session id")
	state := fs.String("state", "running", "runtime state (pending|running|stale|lost|completed)")
	progress := fs.String("progress", "", "runtime progress summary")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *session == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --session are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.RecordExecutorHeartbeat(*project, *id, *session, *state, *progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: record heartbeat: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runCardDeliver(args []string) {
	fs := flag.NewFlagSet("card-deliver", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	state := fs.String("state", "", "delivery state (pending|deployed|failed|rolled_back)")
	target := fs.String("target", "", "delivery target/environment")
	summary := fs.String("summary", "", "delivery summary")
	ref := fs.String("ref", "", "delivery reference (e.g., PR URL, commit SHA)")
	followup := fs.String("followup", "", "semicolon-separated follow-up refs (e.g., hotfix issue IDs)")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *state == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --state are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.RecordDelivery(*project, *id, *state, *target, *summary, *ref, splitList(*followup))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: record delivery: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Recorded delivery for card %s: %s\n", *id, *state)
	printJSON(card)
}
