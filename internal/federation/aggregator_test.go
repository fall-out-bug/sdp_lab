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

func TestAggregator_rebuildTasks_prioritySort(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	_ = store.Create(&registry.Project{ID: "p1", RepoURL: ".", RepoBranch: "main"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	// P0=0 highest, P1=1, P2=2, P3=3. Lower number first.
	payload := []byte(`{"project_id":"p1","issues":[{"id":"i2","title":"t2","priority":2},{"id":"i0","title":"t0","priority":0},{"id":"i1","title":"t1","priority":1}],"count":3}`)
	a.handleReady(bus.Envelope{Payload: payload, ProjectID: "p1"})
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].Issue.ID != "i0" || tasks[1].Issue.ID != "i1" || tasks[2].Issue.ID != "i2" {
		t.Errorf("expected order i0,i1,i2 (P0,P1,P2), got %s,%s,%s",
			tasks[0].Issue.ID, tasks[1].Issue.ID, tasks[2].Issue.ID)
	}
}

// TestAggregator_rebuildTasks_sortByAgeThenProjectID documents AC: P0>P1>P2>P3 then by age (older first), then project.
func TestAggregator_rebuildTasks_sortByAgeThenProjectID(t *testing.T) {
	dir := t.TempDir()
	store := registry.NewStore(registry.StoreConfig{RegistryPath: dir + "/reg.yaml"})
	_ = store.Create(&registry.Project{ID: "p1", RepoURL: ".", RepoBranch: "main"})
	_ = store.Create(&registry.Project{ID: "p2", RepoURL: ".", RepoBranch: "main"})
	ws := NewWorkspaceManager(dir)
	a := NewAggregator(nil, store, ws)
	// Same priority (1); different CreatedAt (older first), then ProjectID as tiebreaker.
	payload := []byte(`{"project_id":"p1","issues":[{"id":"i1b","title":"t1b","priority":1,"created_at":"2026-02-02T10:00:00Z"}],"count":1}`)
	a.handleReady(bus.Envelope{Payload: payload, ProjectID: "p1"})
	payload2 := []byte(`{"project_id":"p2","issues":[{"id":"i2a","title":"t2a","priority":1,"created_at":"2026-02-01T10:00:00Z"}],"count":1}`)
	a.handleReady(bus.Envelope{Payload: payload2, ProjectID: "p2"})
	tasks := a.ReadyAcrossProjects(10)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	// Older (2026-02-01) should be first, then 2026-02-02.
	if tasks[0].Issue.CreatedAt > tasks[1].Issue.CreatedAt {
		t.Errorf("expected order by age (older first): got %s then %s",
			tasks[0].Issue.CreatedAt, tasks[1].Issue.CreatedAt)
	}
}
