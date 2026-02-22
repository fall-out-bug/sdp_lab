package swarm

import (
	"testing"

	"sdp_dev/internal/federation"
)

func TestDispatcher_nilBus(t *testing.T) {
	svc := Dispatcher(nil)
	if svc == nil {
		t.Fatal("Dispatcher(nil) returned nil")
	}
	task := federation.FederatedTask{ProjectID: "p1"}
	if err := svc.Dispatch(task, "coder"); err != nil {
		t.Errorf("Dispatch with nil bus: %v", err)
	}
}
