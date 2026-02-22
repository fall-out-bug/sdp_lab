// Package orchestrator provides capacity management and run monitoring for feature-orchestrator.
package orchestrator

import (
	"context"

	"sdp_dev/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ActivePhases are AgentRun phases that count toward capacity.
var ActivePhases = map[string]bool{
	"":                true,
	"Running":         true,
	"Pending":         true,
	"ReviewerPending": true,
	"ReviewerRunning": true,
	"Escalated":       true, // still holds capacity until cleaned up
}

// CountActiveAgentRuns returns the number of AgentRuns in active phases in the namespace.
func CountActiveAgentRuns(ctx context.Context, k8s client.Client, namespace string) (int, error) {
	list := &v1alpha1.AgentRunList{}
	if err := k8s.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return 0, err
	}
	n := 0
	for i := range list.Items {
		if ActivePhases[list.Items[i].Status.Phase] {
			n++
		}
	}
	return n, nil
}
