package convoy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// WorkstreamGenerator converts convoys to SDP workstream files
type WorkstreamGenerator struct {
	backlogDir string
}

// NewWorkstreamGenerator creates a new workstream generator
func NewWorkstreamGenerator(backlogDir string) *WorkstreamGenerator {
	return &WorkstreamGenerator{backlogDir: backlogDir}
}

// GenerateAll converts all active convoys to workstream files
func (g *WorkstreamGenerator) GenerateAll(convoys []Convoy) ([]string, error) {
	var generated []string

	for _, convoy := range convoys {
		if convoy.Status == "complete" {
			continue // Skip completed convoys
		}

		path, err := g.Generate(convoy)
		if err != nil {
			return nil, fmt.Errorf("failed to generate WS for convoy %s: %w", convoy.ID, err)
		}
		generated = append(generated, path)
	}

	return generated, nil
}

// Generate creates a workstream file from a convoy
func (g *WorkstreamGenerator) Generate(convoy Convoy) (string, error) {
	// Extract feature ID from convoy metadata or use default
	featureID := "F060"
	if fid, ok := convoy.Metadata["feature_id"]; ok {
		featureID = fid
	}

	// Generate workstream ID from convoy ID
	wsID := g.generateWSID(convoy.ID)

	// Determine status
	status := "backlog"
	if convoy.Status == "active" {
		status = "in_progress"
	}

	// Create workstream content
	content, err := g.renderWS(convoy, wsID, featureID, status)
	if err != nil {
		return "", fmt.Errorf("failed to render WS template: %w", err)
	}

	// Write to file
	path := filepath.Join(g.backlogDir, fmt.Sprintf("%s.md", wsID))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write WS file: %w", err)
	}

	return path, nil
}

// generateWSID creates a workstream ID from a convoy ID
func (g *WorkstreamGenerator) generateWSID(convoyID string) string {
	// Extract numeric part or generate hash-based ID
	// For now, use a simple format: 00-060-XX
	// In production, this would be more sophisticated
	return strings.ToLower(strings.ReplaceAll(convoyID, "_", "-"))
}

// renderWS renders the workstream file template
func (g *WorkstreamGenerator) renderWS(convoy Convoy, wsID, featureID, status string) (string, error) {
	tmpl := `---
ws_id: {{.WSID}}
feature_id: {{.FeatureID}}
status: {{.Status}}
priority: {{.Priority}}
size: M
depends_on: []
convoy_id: {{.ConvoyID}}
convoy_synced_at: {{.SyncedAt}}
---

# {{.WSID}}: {{.Title}}

Feature: {{.FeatureID}} (Gas Town Adapter)

## Goal

{{.Description}}

## Beads

- sdplab-6: {{.WSID}} {{.Title}}

## Scope Files

- Generated from Gas Town convoy {{.ConvoyID}}
- TODO: Add specific scope files

## Acceptance Criteria

- [ ] Imported from Gas Town convoy
- [ ] Allocated to SDP workstream
- [ ] Status sync: convoy complete → WS done
- [ ] Evidence collected and validated

## Reference

- [ECOSYSTEM_SYNERGIES.md](../../integrations/ECOSYSTEM_SYNERGIES.md)
- Gas Town MEOW workflow (Mayor → Convoy → Agent)
- Convoy ID: {{.ConvoyID}}

---

*Auto-generated from Gas Town convoy {{.ConvoyID}} at {{.SyncedAt}}*
`

	data := struct {
		WSID        string
		FeatureID   string
		Status      string
		Priority    string
		ConvoyID    string
		Title       string
		Description string
		SyncedAt    string
	}{
		WSID:        wsID,
		FeatureID:   featureID,
		Status:      status,
		Priority:    strings.ToUpper(convoy.Priority),
		ConvoyID:    convoy.ID,
		Title:       convoy.Title,
		Description: convoy.Description,
		SyncedAt:    time.Now().Format(time.RFC3339),
	}

	t, err := template.New("ws").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
