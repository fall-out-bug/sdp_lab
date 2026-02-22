package federation

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"sync"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/registry"
)

// Aggregator subscribes to sdp.beads.*.ready and maintains a cross-project priority queue.
type Aggregator struct {
	bus       bus.Bus
	store     *registry.Store
	workspace *WorkspaceManager
	mu        sync.RWMutex
	tasks     []FederatedTask
	byProject map[string][]beads.Issue
}

// NewAggregator creates an Aggregator.
func NewAggregator(b bus.Bus, store *registry.Store, workspace *WorkspaceManager) *Aggregator {
	return &Aggregator{
		bus:       b,
		store:     store,
		workspace: workspace,
		byProject: make(map[string][]beads.Issue),
	}
}

// Run subscribes to sdp.beads.*.ready and updates the queue.
func (a *Aggregator) Run(ctx context.Context) error {
	if a.bus == nil {
		return nil
	}
	_, err := a.bus.Subscribe("sdp.beads.*.ready", "aggregator", a.handleReady)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

// handleReady processes a ready snapshot from a project.
func (a *Aggregator) handleReady(env bus.Envelope) {
	var snap struct {
		ProjectID string        `json:"project_id"`
		Issues    []beads.Issue `json:"issues"`
		Count     int           `json:"count"`
	}
	if len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &snap); err != nil {
			log.Printf("aggregator: parse ready: %v", err)
			return
		}
	}
	if snap.ProjectID == "" {
		snap.ProjectID = env.ProjectID
	}

	a.mu.Lock()
	a.byProject[snap.ProjectID] = snap.Issues
	a.rebuildTasks()
	a.mu.Unlock()
}

func (a *Aggregator) rebuildTasks() {
	var tasks []FederatedTask
	for projectID, issues := range a.byProject {
		proj, ok := a.store.Get(projectID)
		if !ok {
			continue
		}
		workspace, err := a.workspace.EnsureWorkspaceFromProject(proj)
		if err != nil {
			log.Printf("aggregator: workspace %s: %v", projectID, err)
			continue
		}
		for _, iss := range issues {
			tasks = append(tasks, FederatedTask{
				ProjectID: projectID,
				Issue:     iss,
				Workspace: workspace,
			})
		}
	}
	// Sort by priority (P0=0 highest first: lower number first), then by age (older first), then by project
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Issue.Priority != tasks[j].Issue.Priority {
			return tasks[i].Issue.Priority < tasks[j].Issue.Priority
		}
		if tasks[i].Issue.CreatedAt != tasks[j].Issue.CreatedAt {
			return tasks[i].Issue.CreatedAt < tasks[j].Issue.CreatedAt
		}
		return tasks[i].ProjectID < tasks[j].ProjectID
	})
	a.tasks = tasks
}

// ReadyAcrossProjects returns up to limit tasks from the cross-project queue.
func (a *Aggregator) ReadyAcrossProjects(limit int) []FederatedTask {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if limit <= 0 || limit > len(a.tasks) {
		limit = len(a.tasks)
	}
	out := make([]FederatedTask, limit)
	copy(out, a.tasks[:limit])
	return out
}
