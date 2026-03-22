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
	case "attention":
		runAttention(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp <card|board|doctor|dispatch|result|attention> <subcommand> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Card commands:")
	fmt.Fprintln(os.Stderr, "  sdp card <create|clarify|needs-input|ready|park|execute|feedback|feedback-export|message-export|resume|resume-import|reply-ingest>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Board commands:")
	fmt.Fprintln(os.Stderr, "  sdp board <build|show>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Doctor commands:")
	fmt.Fprintln(os.Stderr, "  sdp doctor control")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Dispatch commands:")
	fmt.Fprintln(os.Stderr, "  sdp dispatch card")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Result commands:")
	fmt.Fprintln(os.Stderr, "  sdp result ingest")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Other:")
	fmt.Fprintln(os.Stderr, "  sdp attention")
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
		fmt.Fprintln(os.Stderr, "usage: sdp card <create|clarify|needs-input|ready|park|execute|feedback|feedback-export|message-export|resume|resume-import|reply-ingest>")
		os.Exit(2)
	}
	switch args[0] {
	case "create":
		runCardCreate(args[1:])
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
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp card <create|clarify|needs-input|ready|park|execute|feedback|feedback-export|message-export|resume|resume-import|reply-ingest>")
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
	runBoardBuild(args)
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

	fmt.Println("🩺 DOCTOR CONTROL REPORT")
	fmt.Println("========================")
	fmt.Printf("Total checks: %d\n", report.TotalChecks)
	fmt.Printf("Passed: %d\n", report.Passed)
	fmt.Printf("Failed: %d\n", report.Failed)
	fmt.Println()

	if len(report.Checks) > 0 {
		fmt.Println("❌ ISSUES FOUND")
		fmt.Println("---------------")
		for _, check := range report.Checks {
			fmt.Printf("[%s] %s", check.Severity, check.CheckID)
			if check.ProjectID != "" {
				fmt.Printf(" | project: %s", check.ProjectID)
			}
			if check.CardID != "" {
				fmt.Printf(" | card: %s", check.CardID)
			}
			fmt.Println()
			fmt.Printf("  %s\n", check.Message)
		}
		fmt.Println()
		os.Exit(1)
	}

	fmt.Println("✅ ALL CHECKS PASSED")
}

func runDispatch(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp dispatch card")
		os.Exit(2)
	}
	switch args[0] {
	case "card":
		runDispatchCard(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown dispatch subcommand: %s\n", args[0])
		os.Exit(2)
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

func runAttention(args []string) {
	store := openStore()
	snap, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔍 ATTENTION SURFACE")
	fmt.Println("=================")
	fmt.Println()

	fmt.Println("📌 NEXT RECOMMENDED ACTION")
	fmt.Println("--------------------------")
	if snap.NextAction != nil {
		fmt.Printf("Action: %s\n", snap.NextAction["recommended"])
		if reason, ok := snap.NextAction["reason"]; ok {
			fmt.Printf("Reason: %s\n", reason)
		}
		if targetProj, ok := snap.NextAction["target_project_id"]; ok && targetProj != "" {
			fmt.Printf("Target Project: %s\n", targetProj)
		}
		if targetCard, ok := snap.NextAction["target_card_id"]; ok && targetCard != "" {
			fmt.Printf("Target Card: %s\n", targetCard)
		}
	} else {
		fmt.Println("No immediate action needed")
	}
	fmt.Println()

	fmt.Println("📊 QUEUES")
	fmt.Println("----------")
	printQueue("Waiting on Human", snap.Queues["waiting_on_human"])
	printQueue("Ready to Execute", snap.Queues["ready_to_execute"])
	printQueue("Blocked", snap.Queues["blocked"])
	fmt.Println()

	printExecutingCards(snap)
	fmt.Println()

	fmt.Println("📈 TOTALS")
	fmt.Println("---------")
	fmt.Printf("Inbox: %d | Clarifying: %d | Ready: %d | Executing: %d | Blocked: %d | Done: %d\n",
		snap.Totals["inbox"], snap.Totals["clarifying"], snap.Totals["ready"],
		snap.Totals["executing"], snap.Totals["blocked"], snap.Totals["done"])
}

type cardDisplay struct {
	ProjectID string
	control.CardSummary
}

func printQueue(name string, items []control.QueueItem) {
	fmt.Printf("%s: %d\n", name, len(items))
	for _, item := range items {
		fmt.Printf("  [%s/%s] %s\n", item.ProjectID, item.CardID, item.Title)
		if item.RecommendedNextStep != "" {
			fmt.Printf("    → %s\n", item.RecommendedNextStep)
		}
		if len(item.ActiveAgents) > 0 {
			fmt.Printf("    👤 Agents: %v\n", item.ActiveAgents)
		}
		if len(item.NeedsFeedbackFrom) > 0 {
			fmt.Printf("    📝 Waiting from: %v\n", item.NeedsFeedbackFrom)
		}
		if len(item.AuthorUpdate) > 0 {
			fmt.Printf("    📎 Updates: %v\n", item.AuthorUpdate)
		}
		if len(item.AdminActionRequired) > 0 {
			fmt.Printf("    ⚙️  Admin actions: %v\n", item.AdminActionRequired)
		}
	}
	fmt.Println()
}

func printExecutingCards(snap *control.PortfolioBoardSnapshot) {
	executing := []cardDisplay{}

	for _, proj := range snap.Projects {
		projID, _ := proj["project_id"].(string)
		counts, ok := proj["counts"].(map[string]any)
		if !ok {
			continue
		}
		execCount, _ := counts["executing"].(int)
		if execCount == 0 {
			continue
		}

		projSnap, err := openStore().BuildProjectSnapshot(projID)
		if err != nil {
			continue
		}
		for _, card := range projSnap.Columns["executing"] {
			executing = append(executing, cardDisplay{ProjectID: projID, CardSummary: card})
		}
	}

	if len(executing) == 0 {
		fmt.Println("🔄 EXECUTING")
		fmt.Println("------------")
		fmt.Println("No cards currently executing")
		fmt.Println()
		return
	}

	fmt.Println("🔄 EXECUTING")
	fmt.Println("------------")
	for _, e := range executing {
		fmt.Printf("  [%s/%s] %s\n", e.ProjectID, e.ID, e.Title)
		if e.RecommendedNextStep != "" {
			fmt.Printf("    → %s\n", e.RecommendedNextStep)
		}
		if len(e.ActiveAgents) > 0 {
			fmt.Printf("    👤 Agents: %v\n", e.ActiveAgents)
		}
		if len(e.LinkedBeadsIDs) > 0 {
			fmt.Printf("    🔗 Beads: %v\n", e.LinkedBeadsIDs)
		}
	}
	fmt.Println()
}
