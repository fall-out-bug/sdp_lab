// Package orchestrator provides run monitoring for feature-orchestrator.
package orchestrator

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/bus"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultTimeoutSec = 3600

// MonitorAgentRunTimeouts polls AgentRuns and publishes escalated for timed-out runs.
func MonitorAgentRunTimeouts(ctx context.Context, k8s client.Client, b bus.Bus, namespace string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			escalateTimedOut(ctx, k8s, b, namespace)
		}
	}
}

func escalateTimedOut(ctx context.Context, k8s client.Client, b bus.Bus, namespace string) {
	list := &v1alpha1.AgentRunList{}
	if err := k8s.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.Printf("monitor list AgentRuns: %v", err)
		return
	}
	now := time.Now()
	for i := range list.Items {
		run := &list.Items[i]
		// Only Running/Pending/Reviewer* can timeout (not Escalated/Succeeded/Failed)
		phase := run.Status.Phase
		if phase == "Escalated" || phase == "Succeeded" || phase == "Failed" {
			continue
		}
		if !ActivePhases[phase] {
			continue
		}
		timeoutSec := run.Spec.TimeoutSec
		if timeoutSec <= 0 {
			timeoutSec = defaultTimeoutSec
		}
		created := run.ObjectMeta.CreationTimestamp
		if created.IsZero() {
			continue
		}
		deadline := created.Add(time.Duration(timeoutSec) * time.Second)
		if now.After(deadline) {
			run.Status.Phase = "Escalated"
			run.Status.LastError = "timeout exceeded"
			if err := k8s.Status().Update(ctx, run); err != nil {
				log.Printf("update AgentRun %s to Escalated: %v", run.Name, err)
				continue
			}
			if b != nil {
				pl, _ := json.Marshal(map[string]string{
					"agent_run": run.Name, "issue_id": run.Spec.IssueID, "reason": "timeout",
				})
				proj := ""
				if run.Labels != nil {
					proj = run.Labels["project"]
				}
				_ = b.Publish("sdp.lifecycle.agentrun.escalated", bus.Envelope{
					IssueID: run.Spec.IssueID, ProjectID: proj, Phase: "escalated",
					Payload: pl,
				})
			}
			IncEscalated()
			log.Printf("escalated AgentRun %s (timeout)", run.Name)
		}
	}
}
