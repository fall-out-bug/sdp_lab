package federation

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/registry"
)

// Bridge runs a per-project Beads-to-NATS bridge.
type Bridge struct {
	projectID string
	workDir   string
	bus       bus.Bus
	store     *registry.Store
	adapter   *beads.Adapter
	labels      []string
	limit       int
	mu          sync.Mutex
	lastReady   []string
	lastClosed  map[string]struct{} // IDs we already published as closed (capped)
	maxClosed   int                 // cap for lastClosed size
}

// BridgeConfig holds options for a Bridge.
type BridgeConfig struct {
	ProjectID string
	WorkDir   string
	Bus       bus.Bus
	Store     *registry.Store
	Labels   []string
	Limit    int
}

// NewBridge creates a Bridge for a project.
func NewBridge(cfg BridgeConfig) *Bridge {
	if cfg.Limit <= 0 {
		cfg.Limit = 10
	}
	return &Bridge{
		projectID:  cfg.ProjectID,
		workDir:    cfg.WorkDir,
		bus:        cfg.Bus,
		store:      cfg.Store,
		adapter:    beads.NewAdapter(cfg.WorkDir),
		labels:     cfg.Labels,
		limit:      cfg.Limit,
		lastClosed: make(map[string]struct{}),
		maxClosed:  1000,
	}
}

// Run starts the bridge.
func (b *Bridge) Run(ctx context.Context) error {
	if b.bus != nil {
		subj := "sdp.intake." + b.projectID
		_, err := b.bus.Subscribe(subj, "bridge-"+b.projectID, b.handleIntake)
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := b.pollReady(); err != nil {
				log.Printf("bridge %s poll: %v", b.projectID, err)
			}
			if err := b.pollClosed(); err != nil {
				log.Printf("bridge %s pollClosed: %v", b.projectID, err)
			}
		}
	}
}

func (b *Bridge) pollReady() error {
	issues, err := b.adapter.Ready(b.labels, b.limit)
	if err != nil {
		return err
	}

	ids := make([]string, len(issues))
	for i := range issues {
		ids[i] = issues[i].ID
	}

	b.mu.Lock()
	changed := !sliceEqual(b.lastReady, ids)
	b.lastReady = ids
	b.mu.Unlock()

	if !changed || b.bus == nil {
		return nil
	}

	subject := "sdp.beads." + b.projectID + ".ready"
	payload, _ := json.Marshal(map[string]any{
		"project_id": b.projectID,
		"issues":     issues,
		"count":     len(issues),
	})
	env := bus.Envelope{
		IssueID:       b.projectID,
		ArtifactID:    "ready-snapshot",
		ArtifactClass: "beads",
		Phase:         "ready",
		Role:          "bridge",
		Payload:       payload,
		ProjectID:     b.projectID,
	}
	return b.bus.Publish(subject, env)
}

func (b *Bridge) pollClosed() error {
	issues, err := b.adapter.Closed(b.labels, b.limit)
	if err != nil {
		return err
	}
	if b.bus == nil {
		return nil
	}

	b.mu.Lock()
	newlyClosed := make([]string, 0, len(issues))
	for i := range issues {
		id := issues[i].ID
		if _, ok := b.lastClosed[id]; !ok {
			newlyClosed = append(newlyClosed, id)
			b.lastClosed[id] = struct{}{}
		}
	}
	// Cap lastClosed to avoid unbounded growth
	for len(b.lastClosed) > b.maxClosed {
		for k := range b.lastClosed {
			delete(b.lastClosed, k)
			break
		}
	}
	b.mu.Unlock()

	subject := "sdp.beads." + b.projectID + ".closed"
	for _, id := range newlyClosed {
		env := bus.Envelope{
			IssueID:       id,
			ProjectID:     b.projectID,
			ArtifactClass: "beads",
			Phase:         "closed",
			Role:          "bridge",
		}
		if err := b.bus.Publish(subject, env); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bridge) handleIntake(env bus.Envelope) {
	var req struct {
		ProjectID   string   `json:"project_id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Source      string   `json:"source"`
		Priority    int      `json:"priority"`
		Labels      []string `json:"labels"`
	}
	if len(env.Payload) > 0 {
		_ = json.Unmarshal(env.Payload, &req)
	}
	if req.Title == "" {
		var m map[string]any
		_ = json.Unmarshal(env.Payload, &m)
		if t, ok := m["title"].(string); ok {
			req.Title = t
		}
		if d, ok := m["description"].(string); ok {
			req.Description = d
		}
	}
	if req.Title == "" {
		log.Printf("bridge %s: intake missing title", b.projectID)
		return
	}
	if req.Priority <= 0 {
		req.Priority = 1
	}

	id, err := b.adapter.Create(beads.CreateOpts{
		Title:       req.Title,
		Type:        "task",
		Priority:    req.Priority,
		Description: req.Description,
		Labels:      req.Labels,
	})
	if err != nil {
		log.Printf("bridge %s create: %v", b.projectID, err)
		return
	}
	log.Printf("bridge %s: created issue %s", b.projectID, id)
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
