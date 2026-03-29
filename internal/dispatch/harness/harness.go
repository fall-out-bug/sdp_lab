// Package harness defines interfaces and types for agent harness implementations
// and their provider integrations. It also provides a Registry for managing
// multiple harness instances.
package harness

import (
	"context"
	"log/slog"
	"time"
)

// Harness represents an execution environment capable of spawning agent processes.
type Harness interface {
	Name() string
	Spawn(ctx context.Context, opts SpawnOpts) (*Process, error)
	Available() bool
	SupportedProviders() []string
}

// Provider represents an AI model provider with rate-limit awareness.
type Provider interface {
	Name() string
	CheckLimits(ctx context.Context) (*Limits, error)
	Models() []string
}

// SpawnOpts holds options for spawning a new agent process.
type SpawnOpts struct {
	Worktree  string
	TaskFile  string
	Prompt    string
	Timeout   time.Duration
	Model     string
	Agent     string
	ExtraArgs []string
}

// Process represents a running agent process.
type Process struct {
	HarnessName string
	PID         int
	Worktree    string
	StartedAt   time.Time
	Done        <-chan Result
}

// Result holds the outcome of a completed agent process.
type Result struct {
	ExitCode int
	Duration time.Duration
	Output   string
}

// Limits holds rate-limit information returned by a provider.
type Limits struct {
	Total     int       `json:"total"`
	Used      int       `json:"used"`
	Window    string    `json:"window"`
	Source    string    `json:"source"`
	CheckedAt time.Time `json:"checked_at"`
}

// UsagePercent returns the fraction of the limit used (0.0–1.0).
// Returns 0 if Total <= 0 to avoid division by zero.
func (l *Limits) UsagePercent() float64 {
	if l.Total <= 0 {
		return 0
	}
	return float64(l.Used) / float64(l.Total)
}

// Registry manages a collection of named Harness implementations.
type Registry struct {
	harnesses map[string]Harness
}

// NewRegistry returns an initialised, empty Registry.
func NewRegistry() *Registry {
	return &Registry{harnesses: make(map[string]Harness)}
}

// Register adds h to the registry under its Name(). A second registration
// with the same name overwrites the first.
func (r *Registry) Register(h Harness) {
	slog.Debug("registering harness", "name", h.Name())
	r.harnesses[h.Name()] = h
}

// Get returns the Harness registered under name, or nil if not found.
func (r *Registry) Get(name string) Harness {
	return r.harnesses[name]
}

// Available returns all harnesses for which Available() == true.
func (r *Registry) Available() []Harness {
	out := make([]Harness, 0, len(r.harnesses))
	for _, h := range r.harnesses {
		if h.Available() {
			out = append(out, h)
		}
	}
	return out
}

// ForProvider returns all harnesses that list provider in SupportedProviders().
func (r *Registry) ForProvider(provider string) []Harness {
	out := make([]Harness, 0)
	for _, h := range r.harnesses {
		for _, p := range h.SupportedProviders() {
			if p == provider {
				out = append(out, h)
				break
			}
		}
	}
	return out
}

// All returns the names of every registered harness.
func (r *Registry) All() []string {
	names := make([]string, 0, len(r.harnesses))
	for n := range r.harnesses {
		names = append(names, n)
	}
	return names
}
