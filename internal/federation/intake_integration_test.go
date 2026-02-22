package federation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/registry"
)

// TestIntakePublishToBridge verifies that publishing to sdp.intake.{pid}
// is received by a subscriber. Uses embedded NATS.
func TestIntakePublishToBridge(t *testing.T) {
	opts := &server.Options{Port: -1, JetStream: true}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("start NATS: %v", err)
	}
	go ns.Start()
	defer ns.Shutdown()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS not ready")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, err := bus.ConnectAndProvision(ctx, ns.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	workDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(workDir, ".beads"), 0o755)
	_ = os.WriteFile(filepath.Join(workDir, ".beads", "issues.jsonl"), []byte("[]\n"), 0o644)
	_ = os.WriteFile(filepath.Join(workDir, ".beads", "metadata.json"), []byte("{}"), 0o644)

	regDir := t.TempDir()
	regPath := filepath.Join(regDir, "registry.yaml")
	_ = os.WriteFile(regPath, []byte("projects:\n  - id: test-proj\n    repo_url: .\n    repo_branch: main\n"), 0o644)
	regStore := registry.NewStore(registry.StoreConfig{RegistryPath: regPath})
	_ = regStore.Load()

	received := make(chan bus.Envelope, 1)
	_, err = b.Subscribe("sdp.intake.test-proj", "bridge-test", func(env bus.Envelope) {
		received <- env
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"project_id":  "test-proj",
		"title":       "Test task",
		"description": "Integration test",
		"priority":    1,
	})
	env := bus.Envelope{
		IssueID:       "intake-1",
		ArtifactID:    "intake",
		ArtifactClass: "intake",
		Phase:         "created",
		Role:          "gateway",
		Payload:       payload,
		ProjectID:     "test-proj",
	}
	if err := b.Publish("sdp.intake.test-proj", env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-received:
		if got.ProjectID != "test-proj" {
			t.Errorf("got ProjectID %q", got.ProjectID)
		}
		var m map[string]any
		_ = json.Unmarshal(got.Payload, &m)
		if m["title"] != "Test task" {
			t.Errorf("payload title = %v", m["title"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for intake message")
	}
}

// TestIntakeTwoProjects verifies intake routing for 2 different projects.
func TestIntakeTwoProjects(t *testing.T) {
	opts := &server.Options{Port: -1, JetStream: true}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("start NATS: %v", err)
	}
	go ns.Start()
	defer ns.Shutdown()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS not ready")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, err := bus.ConnectAndProvision(ctx, ns.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	projA := make(chan bus.Envelope, 1)
	projB := make(chan bus.Envelope, 1)
	_, _ = b.Subscribe("sdp.intake.proj-a", "test", func(env bus.Envelope) { projA <- env })
	_, _ = b.Subscribe("sdp.intake.proj-b", "test", func(env bus.Envelope) { projB <- env })

	envA := bus.Envelope{IssueID: "a1", ProjectID: "proj-a", Payload: mustMarshal(map[string]string{"title": "Task A"})}
	envB := bus.Envelope{IssueID: "b1", ProjectID: "proj-b", Payload: mustMarshal(map[string]string{"title": "Task B"})}
	if err := b.Publish("sdp.intake.proj-a", envA); err != nil {
		t.Fatalf("publish proj-a: %v", err)
	}
	if err := b.Publish("sdp.intake.proj-b", envB); err != nil {
		t.Fatalf("publish proj-b: %v", err)
	}

	select {
	case got := <-projA:
		if got.ProjectID != "proj-a" {
			t.Errorf("proj-a got ProjectID %q", got.ProjectID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout proj-a")
	}
	select {
	case got := <-projB:
		if got.ProjectID != "proj-b" {
			t.Errorf("proj-b got ProjectID %q", got.ProjectID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout proj-b")
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
