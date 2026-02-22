// swarm-role-agent is a universal role agent subscribing to sdp.dispatch.*.{role}.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/llm"
	"sdp_dev/internal/roles"
)

func main() {
	role := flag.String("role", "coder", "role: analyst, coder, reviewer")
	natsURL := flag.String("nats", os.Getenv("NATS_URL"), "NATS server URL")
	agentID := flag.String("agent-id", "", "agent ID (default: hostname-role-timestamp)")
	flag.Parse()

	if *natsURL == "" {
		log.Fatal("NATS_URL or -nats required")
	}

	if *agentID == "" {
		host, _ := os.Hostname()
		*agentID = host + "-" + *role + "-" + time.Now().Format("20060102150405")
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

	subject := "sdp.dispatch.*." + *role
	queue := "role-" + *role

	_, err = b.Subscribe(subject, queue, func(env bus.Envelope) {
		handleDispatch(ctx, b, env, *role, *agentID)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Printf("swarm-role-agent %s listening on %s", *role, subject)
	<-ctx.Done()
}

func handleDispatch(ctx context.Context, b bus.Bus, env bus.Envelope, role, agentID string) {
	var task federation.FederatedTask
	if len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &task); err != nil {
			log.Printf("parse task: %v", err)
			return
		}
	}
	if task.Workspace == "" {
		task.Workspace = "."
	}
	if task.ProjectID == "" {
		task.ProjectID = env.ProjectID
	}

	runID := env.RunID
	if runID == "" {
		runID = "run-" + time.Now().Format("20060102150405")
	}

	boundary, _ := llm.LoadBoundary(task.Workspace, "builder")
	cfg := agent.Config{
		AgentID:   agentID,
		Role:      role,
		ProjectID: task.ProjectID,
		RunID:     runID,
		IssueID:   task.Issue.ID,
		WorkDir:   task.Workspace,
		Bus:       b,
		Beads:     beads.NewAdapter(task.Workspace),
		Boundary:  boundary,
	}
	agentCtx, err := agent.NewContext(cfg)
	if err != nil {
		log.Printf("agent context: %v", err)
		return
	}

	if agentCtx.Trace != nil {
		_ = agentCtx.Trace.BeginTrace(task.Issue.ID)
	}
	if agentCtx.Hooks != nil {
		_ = agentCtx.Hooks.RunPreExecute(ctx, agent.HookData{
			IssueID: task.Issue.ID, RunID: runID, Role: role, WorkDir: task.Workspace,
		})
	}

	input := roles.TaskInput{FederatedTask: task, Ctx: agentCtx}
	res, execErr := roles.Execute(ctx, role, input)
	if execErr != nil {
		log.Printf("execute: %v", execErr)
		return
	}

	if agentCtx.Hooks != nil {
		_ = agentCtx.Hooks.RunPostExecute(ctx, agent.HookData{
			IssueID: task.Issue.ID, RunID: runID, Role: role, WorkDir: task.Workspace,
			ChangedFiles: res.ChangedFiles, ResultSummary: res.Summary,
		})
	}

	artifactSubject := "sdp.artifact." + task.ProjectID + "." + runID + "." + role
	resultPayload := map[string]any{
		"issue_id":      task.Issue.ID,
		"changed_files": res.ChangedFiles,
		"summary":       res.Summary,
		"verdict":       res.Verdict,
		"error":         "",
	}
	if res.Err != nil {
		resultPayload["error"] = res.Err.Error()
	}

	signed, err := agentCtx.Provenance.Sign(agent.SignInput{
		IssueID:       task.Issue.ID,
		ArtifactID:    "result-" + role,
		ArtifactClass: "artifact",
		Phase:         "completed",
		Payload:       resultPayload,
		ModelUsed:     agentCtx.Policy.SelectModelSimple(),
		TraceLink:     agentCtx.Trace.RunPath(),
		Sequence:     0,
		HashPrev:      "",
	})
	if err != nil {
		log.Printf("sign: %v", err)
		return
	}
	signed.RunID = runID
	signed.ProjectID = task.ProjectID
	if err := b.Publish(artifactSubject, signed); err != nil {
		log.Printf("publish artifact: %v", err)
	}

	if agentCtx.Trace != nil {
		_ = agentCtx.Trace.EmitPhase("completed", "ok", res.Summary)
	}
}
