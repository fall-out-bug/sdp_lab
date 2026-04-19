// Package harnessadapter transforms rules and scout data into
// harness-specific configuration file snippets.
package harnessadapter

import (
	"fmt"
	"sort"

	"sdp_dev/internal/harnesscfg"
	"sdp_dev/internal/rules"
	"sdp_dev/internal/scout"
)

// Adapter renders rules + scout data into a harness-specific format.
// Each implementation targets a single harness (claude-code, cursor, etc.)
// and returns a section/snippet, not a complete file.
type Adapter interface {
	Name() string
	Render(card *scout.ProjectCard, rules []rules.Rule) ([]byte, error)
}

// Registry holds all registered adapters keyed by harness name.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a Registry populated with adapters for each enabled
// harness listed in the manifest. Unknown harness names are skipped silently.
func NewRegistry(manifest *harnesscfg.Manifest) *Registry {
	r := &Registry{adapters: make(map[string]Adapter)}
	if manifest == nil {
		return r
	}

	factories := map[string]func() Adapter{
		"claude-code": newClaudeAdapter,
		"cursor":      newCursorAdapter,
		"codex-cli":   newAgentsAdapter,
		"opencode":    newAgentsAdapter,
		"copilot":     newAgentsAdapter,
		"zed":         newAgentsAdapter,
		"warp":        newAgentsAdapter,
	}

	for _, h := range manifest.Harnesses {
		if !isHarnessEnabled(h) {
			continue
		}
		factory, ok := factories[h.Name]
		if !ok {
			continue
		}
		a := factory()
		r.adapters[a.Name()] = a
	}

	return r
}

// Get returns the adapter registered under the given name.
func (r *Registry) Get(name string) (Adapter, error) {
	a, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("harnessadapter: adapter %q not found", name)
	}
	return a, nil
}

// All returns all registered adapters in deterministic (sorted) order.
func (r *Registry) All() []Adapter {
	out := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

// RenderAll invokes Render on every registered adapter and returns a map
// from adapter name to rendered bytes.
func (r *Registry) RenderAll(card *scout.ProjectCard, rules []rules.Rule) (map[string][]byte, error) {
	out := make(map[string][]byte, len(r.adapters))
	for _, a := range r.All() {
		b, err := a.Render(card, rules)
		if err != nil {
			return nil, fmt.Errorf("harnessadapter: render %s: %w", a.Name(), err)
		}
		out[a.Name()] = b
	}
	return out, nil
}

// isHarnessEnabled returns true unless the harness has enabled explicitly set
// to false. Nil or missing enabled means true.
func isHarnessEnabled(h harnesscfg.Harness) bool {
	return h.Enabled == nil || *h.Enabled
}
