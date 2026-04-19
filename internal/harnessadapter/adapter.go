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

// namedAdapter pairs an adapter with the harness name it was registered under.
type namedAdapter struct {
	harnessName string
	adapter     Adapter
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
		// Key by harness name (from manifest), not adapter name, so that
		// multiple harnesses sharing one adapter (e.g. codex-cli and opencode
		// both use agentsAdapter) are not collapsed into a single entry.
		r.adapters[h.Name] = a
	}

	return r
}

// Get returns the adapter registered under the given harness name.
func (r *Registry) Get(name string) (Adapter, error) {
	a, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("harnessadapter: adapter %q not found", name)
	}
	return a, nil
}

// All returns all registered adapters paired with their harness names,
// in deterministic (sorted by harness name) order.
func (r *Registry) All() []namedAdapter {
	out := make([]namedAdapter, 0, len(r.adapters))
	for name, a := range r.adapters {
		out = append(out, namedAdapter{harnessName: name, adapter: a})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].harnessName < out[j].harnessName
	})
	return out
}

// RenderAll invokes Render on every registered adapter and returns a map
// from harness name to rendered bytes.
func (r *Registry) RenderAll(card *scout.ProjectCard, rules []rules.Rule) (map[string][]byte, error) {
	out := make(map[string][]byte, len(r.adapters))
	for _, na := range r.All() {
		b, err := na.adapter.Render(card, rules)
		if err != nil {
			return nil, fmt.Errorf("harnessadapter: render %s: %w", na.harnessName, err)
		}
		out[na.harnessName] = b
	}
	return out, nil
}

// isHarnessEnabled returns true unless the harness has enabled explicitly set
// to false. Nil or missing enabled means true.
func isHarnessEnabled(h harnesscfg.Harness) bool {
	return h.Enabled == nil || *h.Enabled
}
