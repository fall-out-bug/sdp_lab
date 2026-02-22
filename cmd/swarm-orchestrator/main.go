// swarm-orchestrator coordinates multi-project swarm with NATS lifecycle.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/observability"
	"sdp_dev/internal/registry"
	"sdp_dev/internal/swarm"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	_, _ = observability.SetupTracing("swarm-orchestrator")

	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	workspaceBase := flag.String("workspace", "/workspaces", "base dir for project workspaces")
	flag.Parse()

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	b, err := bus.ConnectAndProvision(ctx, *natsURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer b.Close()

	store := registry.NewStore(registry.StoreConfig{})
	_ = store.Load()

	ws := federation.NewWorkspaceManager(*workspaceBase)
	agg := federation.NewAggregator(b, store, ws)

	// Start Bridge per registered project (intake -> beads, beads ready -> NATS)
	for _, proj := range store.List() {
		proj := proj
		workspace, err := ws.EnsureWorkspaceFromProject(&proj)
		if err != nil {
			log.Printf("bridge %s: workspace: %v (skipping)", proj.ID, err)
			continue
		}
		labels := []string{}
		if proj.BeadsPrefix != "" {
			labels = append(labels, proj.BeadsPrefix)
		}
		br := federation.NewBridge(federation.BridgeConfig{
			ProjectID: proj.ID,
			WorkDir:   workspace,
			Bus:       b,
			Store:     store,
			Labels:    labels,
			Limit:     10,
			Workspace: ws,
		})
		pid := proj.ID
		go func() {
			if err := br.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("bridge %s: %v", pid, err)
			}
		}()
		log.Printf("bridge started for %s (workdir=%s)", proj.ID, workspace)
	}
	go func() {
		_ = agg.Run(ctx)
	}()

	coord := swarm.NewCoordinator()
	disp := swarm.Dispatcher(b)

	_, err = b.SubscribeWithContext("sdp.beads.*.ready", "orchestrator", func(ctx context.Context, env bus.Envelope) {
		handleReady(ctx, env, agg, coord, disp)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Printf("swarm-orchestrator running")
	<-ctx.Done()
}

func handleReady(ctx context.Context, env bus.Envelope, agg *federation.Aggregator, coord *swarm.Coordinator, disp *swarm.DispatchService) {
	ctx, span := otel.Tracer("swarm-orchestrator").Start(ctx, "dispatch")
	defer span.End()

	tasks := agg.ReadyAcrossProjects(3)
	span.SetAttributes(attribute.Int("ready_count", len(tasks)))
	for _, task := range tasks {
		key := task.ProjectID + ":" + task.Issue.ID
		if coord.Get(task.ProjectID, task.Issue.ID) != nil {
			continue
		}
		coord.Claim(task.ProjectID, task.Issue.ID, key)
		_, dispSpan := otel.Tracer("swarm-orchestrator").Start(ctx, "Dispatch")
		dispSpan.SetAttributes(attribute.String("project", task.ProjectID), attribute.String("issue", task.Issue.ID))
		_ = disp.DispatchWithContext(ctx, task, "coder")
		dispSpan.End()
	}
}
