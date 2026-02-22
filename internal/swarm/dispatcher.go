package swarm

import (
	"context"
	"encoding/json"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
)

// Dispatcher publishes tasks to sdp.dispatch.{project}.{role}.
func Dispatcher(b bus.Bus) *DispatchService {
	return &DispatchService{bus: b}
}

// DispatchService dispatches tasks to role agents.
type DispatchService struct {
	bus bus.Bus
}

// Dispatch sends a FederatedTask to the given role (no trace propagation).
func (d *DispatchService) Dispatch(task federation.FederatedTask, role string) error {
	return d.DispatchWithContext(context.Background(), task, role)
}

// DispatchWithContext sends with W3C trace context in NATS headers.
func (d *DispatchService) DispatchWithContext(ctx context.Context, task federation.FederatedTask, role string) error {
	if d.bus == nil {
		return nil
	}
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	subject := "sdp.dispatch." + task.ProjectID + "." + role
	env := bus.Envelope{
		IssueID:       task.Issue.ID,
		ArtifactID:    "dispatch",
		ArtifactClass: "dispatch",
		Phase:         role,
		Role:          "orchestrator",
		Payload:       payload,
		ProjectID:     task.ProjectID,
		RunID:         task.ProjectID + "-" + task.Issue.ID,
	}
	return d.bus.PublishWithContext(ctx, subject, env)
}
