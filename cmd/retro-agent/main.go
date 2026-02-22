// retro-agent performs multi-lens LLM analysis on epic closure.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/retrospective"
)

func main() {
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	epicID := flag.String("epic", "", "one-shot: analyze this epic ID")
	workDir := flag.String("dir", ".", "workspace directory")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	if *epicID != "" {
		agg := retrospective.NewAggregator(*workDir)
		runs, evidence, intakePath := agg.CollectPaths(*epicID)
		log.Printf("retro epic %s: runs=%d evidence=%d intake=%s", *epicID, len(runs), len(evidence), intakePath)
		return
	}

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required for daemon mode")
	}

	b, err := bus.ConnectAndProvision(ctx, *natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer b.Close()

	_, err = b.Subscribe("sdp.beads.*.closed", "retro-agent", func(env bus.Envelope) {
		log.Printf("retro: closed event %s", env.IssueID)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Printf("retro-agent listening for sdp.beads.*.closed")
	<-ctx.Done()
}
