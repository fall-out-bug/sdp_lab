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
	fmt.Fprintln(os.Stderr, "⚠️  DEPRECATED: sdp-control is deprecated. Use 'sdp' instead.")
	fmt.Fprintln(os.Stderr, "   Example: sdp-control card-create → sdp card create")
	fmt.Fprintln(os.Stderr, "            sdp-control board-show → sdp board show")
	fmt.Fprintln(os.Stderr, "            sdp-control doctor control → sdp doctor control")
	fmt.Fprintln(os.Stderr, "            sdp-control dispatch-card → sdp dispatch card")
	fmt.Fprintln(os.Stderr, "            sdp-control result-ingest → sdp result ingest")
	fmt.Fprintln(os.Stderr, "            sdp-control attention → sdp attention")
	fmt.Fprintln(os.Stderr)
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
	case "card-deliver":
		runCardDeliver(os.Args[2:])
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
	case "attention":
		runAttention(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
	case "packet-emit":
		runPacketEmit(os.Args[2:])
	case "dispatch-card":
		runDispatchCard(os.Args[2:])
	case "result-ingest":
		runResultIngest(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp-control <card-create|card-clarify|card-needs-input|card-ready|card-park|card-execute|card-deliver|card-feedback|card-feedback-export|card-message-export|card-resume|card-resume-import|card-reply-ingest|dispatch-card|packet-emit|result-ingest|board-build|board-show|attention|doctor> [flags]")
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

func runCardDeliver(args []string) {
	fs := flag.NewFlagSet("card-deliver", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	state := fs.String("state", "", "delivery state: pending|deployed|failed|rolled_back")
	target := fs.String("target", "", "delivery target or environment")
	summary := fs.String("summary", "", "delivery summary")
	ref := fs.String("ref", "", "delivery or rollback reference")
	followups := fs.String("followups", "", "semicolon-separated follow-up refs")
	_ = fs.Parse(args)
	if *project == "" || *id == "" || *state == "" {
		fmt.Fprintln(os.Stderr, "error: --project, --id, and --state are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.RecordDelivery(*project, *id, *state, *target, *summary, *ref, splitList(*followups))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: record delivery: %v\n", err)
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

type cardDisplay struct {
	ProjectID string
	control.CardSummary
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

func runDoctor(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: subcommand is required (use 'control')")
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

func runPacketEmit(args []string) {
	fs := flag.NewFlagSet("packet-emit", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	packet, err := store.BuildExecutionPacket(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build execution packet: %v\n", err)
		os.Exit(1)
	}
	printJSON(packet)
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
