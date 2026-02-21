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

func TestAggregator_handleReady_EmptyPayload(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	env := bus.Envelope{Payload: []byte(`{}`), ProjectID: "p1"}
	a.handleReady(env)
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for empty payload, got %d", len(tasks))
	}
}

func TestAggregator_ReadyAcrossProjects_Limit(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	_ = store.Create(&registry.Project{ID: "p1", RepoURL: ".", RepoBranch: "main"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	payload := []byte(`{"project_id":"p1","issues":[{"id":"i1","title":"t1","priority":1},{"id":"i2","title":"t2","priority":2}],"count":2}`)
	a.handleReady(bus.Envelope{Payload: payload, ProjectID: "p1"})
	tasks := a.ReadyAcrossProjects(1)
	if len(tasks) != 1 {
		t.Errorf("ReadyAcrossProjects(1) should return 1, got %d", len(tasks))
	}
}

func TestAggregator_handleReady_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	a.handleReady(bus.Envelope{Payload: []byte(`{invalid`), ProjectID: "p1"})
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 0 {
		t.Errorf("invalid JSON should not add tasks, got %d", len(tasks))
	}
}

func TestAggregator_handleReady_projectIDFromEnv(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	_ = store.Create(&registry.Project{ID: "p2", RepoURL: ".", RepoBranch: "main"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	payload := []byte(`{"issues":[{"id":"i1","title":"t1","priority":1}],"count":1}`)
	a.handleReady(bus.Envelope{Payload: payload, ProjectID: "p2"})
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 1 || tasks[0].ProjectID != "p2" {
		t.Errorf("project ID from envelope: %+v", tasks)
	}
}

func TestAggregator_rebuildTasks_unknownProject(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	payload := []byte(`{"project_id":"unknown","issues":[{"id":"i1","title":"t1","priority":1}],"count":1}`)
	a.handleReady(bus.Envelope{Payload: payload, ProjectID: "unknown"})
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 0 {
		t.Errorf("unknown project should yield 0 tasks, got %d", len(tasks))
	}
}
