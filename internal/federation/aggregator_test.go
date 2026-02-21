package federation

import (
	"testing"

	"sdp_dev/internal/bus"
	"sdp_dev/internal/registry"
)

func TestNewAggregator(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	if a == nil {
		t.Fatal("NewAggregator returned nil")
	}
}

func TestAggregator_ReadyAcrossProjects_empty(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	tasks := a.ReadyAcrossProjects(5)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestAggregator_handleReady(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	_ = store.Create(&registry.Project{
		ID: "p1", RepoURL: ".", RepoBranch: "main",
	})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)

	payload := []byte(`{"project_id":"p1","issues":[{"id":"i1","title":"t1","priority":1}],"count":1}`)
	env := bus.Envelope{Payload: payload, ProjectID: "p1"}
	a.handleReady(env)

	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 1 || tasks[0].Issue.ID != "i1" {
		t.Errorf("ReadyAcrossProjects: %+v", tasks)
	}
}
