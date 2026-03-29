package gate

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// BeadsGateManager creates and monitors gates using filesystem-backed storage.
// Gate state is persisted to .sdp/gates/<id>.json under the project root.
type BeadsGateManager struct {
	ProjectRoot string
}

func (m *BeadsGateManager) gatesDir() string {
	return filepath.Join(m.ProjectRoot, ".sdp", "gates")
}

var validGateID = regexp.MustCompile(`^[a-f0-9]+$`)

func (m *BeadsGateManager) gatePath(id string) (string, error) {
	if !validGateID.MatchString(id) {
		return "", fmt.Errorf("invalid gate ID: %q", id)
	}
	return filepath.Join(m.gatesDir(), id+".json"), nil
}

// generateID produces an 8-character hex string using crypto/rand.
func generateID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate gate ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateGate creates a new gate and persists it to disk.
// Returns the gate with a generated ID.
func (m *BeadsGateManager) CreateGate(question, context string, options []string) (*Gate, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}

	g := &Gate{
		ID:        id,
		Question:  question,
		Context:   context,
		Options:   options,
		CreatedAt: time.Now(),
	}

	if err := m.saveGate(g); err != nil {
		return nil, err
	}
	return g, nil
}

// CheckGate reads the gate status from disk.
func (m *BeadsGateManager) CheckGate(gateID string) (*Gate, error) {
	return m.loadGate(gateID)
}

// ResolveGate marks a gate as resolved with the given answer.
func (m *BeadsGateManager) ResolveGate(gateID, answer, answerer string) error {
	g, err := m.loadGate(gateID)
	if err != nil {
		return err
	}

	now := time.Now()
	g.Answer = answer
	g.Answerer = answerer
	g.ResolvedAt = &now

	return m.saveGate(g)
}

// ListPending returns all unresolved gates.
func (m *BeadsGateManager) ListPending() ([]*Gate, error) {
	dir := m.gatesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read gates dir: %w", err)
	}

	var pending []*Gate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		g, err := m.loadGate(id)
		if err != nil {
			continue
		}
		if g.IsBlocking() {
			pending = append(pending, g)
		}
	}
	return pending, nil
}

func (m *BeadsGateManager) saveGate(g *Gate) error {
	dir := m.gatesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create gates dir: %w", err)
	}

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal gate: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "gate-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp gate file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp gate file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp gate file: %w", err)
	}

	dest := filepath.Join(dir, g.ID+".json")
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename gate file: %w", err)
	}
	return nil
}

func (m *BeadsGateManager) loadGate(id string) (*Gate, error) {
	path, err := m.gatePath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gate %s: %w", id, err)
	}

	var g Gate
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("unmarshal gate %s: %w", id, err)
	}
	return &g, nil
}
