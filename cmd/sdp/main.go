package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"sdp_dev/internal/cli"
	"sdp_dev/internal/control"
	"sdp_dev/internal/deploy"
	"sdp_dev/internal/executor"
	"sdp_dev/internal/orchestrate"
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
<<<<<<< HEAD
	case "roles":
		runRoles(os.Args[2:])
=======
	case "summary":
		runSummary(os.Args[2:])
>>>>>>> feature/sdp-summarizer
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp <card|board|doctor|dispatch|result|orchestrate|attention|roles> <subcommand> [flags]")
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
	fmt.Fprintln(os.Stderr, "Pipeline commands:")
	fmt.Fprintln(os.Stderr, "  sdp intent \"description\"   Create intake card from raw intent")
	fmt.Fprintln(os.Stderr, "  sdp status <card-id>        Show card status and phase")
	fmt.Fprintln(os.Stderr, "  sdp stuck                  Show stuck/long-running cards")
	fmt.Fprintln(os.Stderr, "  sdp eval <card-id>         Run build evaluation manually")
	fmt.Fprintln(os.Stderr, "  sdp clarify <card-id>      Run clarification manually")
	fmt.Fprintln(os.Stderr, "  sdp summary <card-id>      Print human-readable evidence summary")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Other:")
	fmt.Fprintln(os.Stderr, "  sdp attention")
	fmt.Fprintln(os.Stderr, "  sdp roles")
}

func runRoles(args []string) {
	fs := flag.NewFlagSet("roles", flag.ExitOnError)
	_ = fs.Parse(args)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PHASE\tAGENT\tDESCRIPTION")
	for _, role := range executor.PhaseRoleMap {
		fmt.Fprintf(w, "%s\t%s\t%s\n", role.Phase, role.Agent, role.Description)
	}
	if override := strings.TrimSpace(os.Getenv("SDP_DEFAULT_AGENT")); override != "" {
		fmt.Fprintf(w, "*\t%s\tSDP_DEFAULT_AGENT override\n", override)
	}
	_ = w.Flush()
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
	store, err := control.OpenFromEnv(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err)
		os.Exit(1)
	}
	return store
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
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

func runBoard(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp board <build|show>")
		os.Exit(2)
	}
	switch args[0] {
	case "build":
		runBoardBuild(args[1:])
	case "show":
		runBoardShow(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp board <build|show>")
		os.Exit(2)
	}
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
	fs := flag.NewFlagSet("board-show", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	asJSON := fs.Bool("json", false, "render raw JSON instead of the default human summary")
	_ = fs.Parse(args)

	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err)
			os.Exit(1)
		}
		if *asJSON {
			printJSON(snap)
			return
		}
		fmt.Println(cli.RenderProjectBoard(snap))
		return
	}

	port, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}
	if *asJSON {
		printJSON(port)
		return
	}
	fmt.Println(cli.RenderPortfolioBoard(port))
}

func runDoctor(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp doctor control")
		os.Exit(2)
	}
	switch args[0] {
	case "control":
		runDoctorControl()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown doctor subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runDoctorControl() {
	store := openStore()
	report, err := store.DoctorControl()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: doctor control: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cli.RenderDoctorControl(report))
	if len(report.Checks) > 0 {
		os.Exit(1)
	}
}

func runDispatch(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp dispatch card")
		os.Exit(2)
	}
	switch args[0] {
	case "card":
		runDispatchCard(args[1:])
	case "next":
		runDispatchNext(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown dispatch subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func runDispatchCard(args []string) {
	fs := flag.NewFlagSet("dispatch-card", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.DispatchCard(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: dispatch card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runDispatchNext(args []string) {
	fs := flag.NewFlagSet("dispatch-next", flag.ExitOnError)
	execute := fs.Bool("execute", false, "Dispatch, execute through the bridge, then auto-ingest the result")
	_ = fs.Parse(args)

	store := openStore()
	result, err := store.DispatchNext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: dispatch next: %v\n", err)
		os.Exit(1)
	}

	if result.Success {
		fmt.Printf("✅ %s\n", result.Message)
		if result.ProjectID != "" && result.CardID != "" {
			fmt.Printf("   Project: %s | Card: %s\n", result.ProjectID, result.CardID)
		}
		if result.ExecutorRole != "" {
			fmt.Printf("   Executor: %s\n", result.ExecutorRole)
		}
		if result.PacketPath != "" {
			fmt.Printf("   Packet: %s\n", result.PacketPath)
		}
		if *execute {
			bridge := &executor.ExecutorBridge{Store: store, Invoker: orchestrate.DefaultLLMInvoker, ProjectRoot: store.ProjectRoot}
			execResult, err := bridge.DispatchAndRun(context.Background(), result.ProjectID, result.CardID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: execute dispatched card: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("   Execution result: %s\n", execResult.Status)
			if _, err := store.OrchestrateOnce(); err != nil {
				fmt.Fprintf(os.Stderr, "error: auto-ingest executor result: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Printf("⏸️  %s\n", result.Message)
		if result.NoDispatchableReason != "" {
			fmt.Printf("   Reason: %s\n", result.NoDispatchableReason)
		}
	}

	printJSON(result)
}

func runResult(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp result ingest")
		os.Exit(2)
	}
	switch args[0] {
	case "ingest":
		runResultIngest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown result subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func runResultIngest(args []string) {
	fs := flag.NewFlagSet("result-ingest", flag.ExitOnError)
	path := fs.String("input", "", "input file path (required)")
	_ = fs.Parse(args)
	if *path == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(2)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read result packet from %s: %v\n", *path, err)
		os.Exit(1)
	}

	var packet control.ExecutorResultPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse result packet: %v\n", err)
		os.Exit(1)
	}

	store := openStore()
	card, err := store.IngestExecutorResult(&packet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: ingest executor result: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Ingested executor result for card %s\n", packet.ParentFeatureID)
	printJSON(card)
}

func runOrchestrate(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp orchestrate once")
		os.Exit(2)
	}
	switch args[0] {
	case "once":
		runOrchestrateOnce(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp orchestrate once")
		os.Exit(2)
	}
}

func runOrchestrateOnce(args []string) {
	fs := flag.NewFlagSet("orchestrate-once", flag.ExitOnError)
	_ = fs.Parse(args)

	store := openStore()
	result, err := store.OrchestrateOnce()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: orchestrate once: %v\n", err)
		os.Exit(1)
	}

	if result.Action == "ingested" {
		fmt.Printf("✅ %s\n", result.Message)
		if result.IngestedCard != nil {
			fmt.Printf("   Card: %s/%s\n", result.IngestedCard.ProjectID, result.IngestedCard.ID)
		}
	} else if result.Action == "dispatched" {
		fmt.Printf("✅ %s\n", result.Message)
		if result.DispatchedCard != nil {
			fmt.Printf("   Project: %s | Card: %s\n", result.DispatchedCard.ProjectID, result.DispatchedCard.ID)
		}
		if result.ExecutorRole != "" {
			fmt.Printf("   Executor: %s\n", result.ExecutorRole)
		}
		if result.PacketPath != "" {
			fmt.Printf("   Packet: %s\n", result.PacketPath)
		}
	} else {
		fmt.Printf("⏸️  %s\n", result.Message)
		if result.NoActionReason != "" {
			fmt.Printf("   Reason: %s\n", result.NoActionReason)
		}
	}

	printJSON(result)
}

func runAttention(args []string) {
	fs := flag.NewFlagSet("attention", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "render raw JSON instead of the default human summary")
	_ = fs.Parse(args)

	store := openStore()
	snap, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(snap)
		return
	}
	fmt.Println(cli.RenderAttention(snap))
}

func runWhy(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp why <card-id>")
		os.Exit(2)
	}
	store := openStore()
	blockers, err := store.WhyBlocked(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	control.PrintWhyBlocked(blockers, nil)
	if *flagSetFrom(args, "--json") {
		printJSON(blockers)
	}
}

func runNext(args []string) {
	fs := flag.NewFlagSet("next", flag.ExitOnError)
	limit := fs.Int("limit", 10, "max items to show")
	asJSON := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)

	store := openStore()
	items, err := store.WhatNext(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(items)
		return
	}
	if len(items) == 0 {
		fmt.Println("📭  No actionable items.")
		return
	}
	for _, item := range items {
		fmt.Printf("  ▶  %s: %s [%s]\n", item.ID, item.Title, item.Status)
	}
}

func runMissing(args []string) {
	store := openStore()
	projectID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectID = args[0]
	}
	asJSON := flagSetFrom(args, "--json")

	missing, err := store.WhatMissing(projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(missing)
		return
	}
	control.PrintMissing(missing, nil)
}

func runApprove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp approve <card-id>")
		os.Exit(2)
	}
	store := openStore()
	beadsRepo := store.BeadsRepo()
	if beadsRepo == nil {
		fmt.Fprintln(os.Stderr, "error: approve requires beads or dual mode (set SDP_REPO_MODE)")
		os.Exit(1)
	}
	if err := beadsRepo.ResolveGate(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve gate: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅  Gate %s resolved.\n", args[0])
}

func runTrace(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp trace <card-id>")
		os.Exit(2)
	}
	store := openStore()
	trace, err := store.TraceFeature(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📌 %s\n", trace.Root)
	if len(trace.Children) == 0 {
		fmt.Println("   (no children)")
		return
	}
	for _, child := range trace.Children {
		fmt.Printf("   ├─ %s: %s [%s]\n", child.ID, child.Title, child.Status)
	}
}

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

func flagSetFrom(args []string, flag string) *bool {
	v := false
	for _, a := range args {
		if a == flag || a == flag+"=true" {
			v = true
		}
	}
	return &v
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

func runSummary(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp summary <card-id>")
		os.Exit(2)
	}
	store := openStore()
	bridge := executor.NewServeBridge(store, store.ProjectRoot)
	result, err := bridge.Summarize(context.Background(), args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: summarize card: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.Text)
}
