// telemetry-analyzer subscribes to sdp.beads.*.closed, analyzes evidence, generates backlog proposals.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/telemetry"
)

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	workDir := flag.String("dir", os.Getenv("SDP_WORK_DIR"), "Workspace base (default /workspaces)")
	interval := flag.Duration("interval", 30*time.Second, "Poll interval for rate limit reset")
	model := flag.String("model", os.Getenv("TELEMETRY_MODEL"), "LLM model for analysis (default glm-4.7)")
	maxPerCycle := flag.Int("max-per-cycle", 5, "Max proposals per 30min cycle")
	flag.Parse()

	if *workDir == "" {
		*workDir = "/workspaces"
	}
	if *model == "" {
		*model = "glm-4.7"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required")
	}

	b, err := bus.ConnectAndProvision(ctx, *natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer b.Close()

	analyzer := telemetry.NewAnalyzer(*workDir, *model, *maxPerCycle)

	_, err = b.Subscribe("sdp.beads.*.closed", "telemetry-analyzer", func(env bus.Envelope) {
		issueID := env.IssueID
		projectID := env.ProjectID
		if issueID == "" {
			return
		}
		created, err := analyzer.HandleClosed(ctx, issueID, projectID)
		if err != nil {
			log.Printf("analyze %s: %v", issueID, err)
			return
		}
		if created {
			log.Printf("created proposal from %s", issueID)
		}
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Printf("telemetry-analyzer listening for sdp.beads.*.closed (interval=%v, max=%d/cycle)", *interval, *maxPerCycle)
	<-ctx.Done()
}
