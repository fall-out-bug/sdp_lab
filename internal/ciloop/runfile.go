package ciloop

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sdp_dev/internal/sdputil"
)

// RunEvent is a single event appended to a run file.
type RunEvent struct {
	At    string `json:"at"`
	Phase string `json:"phase"`
	State string `json:"state"`
	Notes string `json:"notes,omitempty"`
}

// RunFile mirrors the .sdp/runs/{run-id}.json schema.
type RunFile struct {
	RunID        string     `json:"run_id"`
	FeatureID    string     `json:"feature_id"`
	Orchestrator string     `json:"orchestrator"`
	Branch       string     `json:"branch"`
	StartedAt    string     `json:"started_at"`
	Events       []RunEvent `json:"events"`
	LastPhase    string     `json:"last_phase"`
	LastState    string     `json:"last_state"`
}

// maxRunEventFieldBytes caps phase/state/notes length to avoid disk DoS.
const maxRunEventFieldBytes = 1024

func truncateField(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// AppendRunEvent finds the latest run file for featureID in dir and appends an event.
// Uses flock for inter-process safety (concurrent access from multiple processes).
func AppendRunEvent(dir, featureID, phase, state, notes string) error {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return err
	}
	phase = truncateField(phase, maxRunEventFieldBytes)
	state = truncateField(state, maxRunEventFieldBytes)
	notes = truncateField(notes, maxRunEventFieldBytes)
	path, err := findRunFile(dir, featureID)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open run file: %w", err)
	}
	defer f.Close()
	if err := flock(f.Fd(), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock run file: %w", err)
	}
	defer func() { _ = flock(f.Fd(), syscall.LOCK_UN) }()
	data, err := io.ReadAll(io.LimitReader(f, sdputil.MaxJSONDecodeBytes))
	if err != nil {
		return fmt.Errorf("read run file: %w", err)
	}
	var rf RunFile
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&rf); err != nil {
		return fmt.Errorf("parse run file: %w", err)
	}
	rf.Events = append(rf.Events, RunEvent{
		At:    time.Now().UTC().Format(time.RFC3339),
		Phase: phase,
		State: state,
		Notes: notes,
	})
	rf.LastPhase = phase
	rf.LastState = state
	out, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run file: %w", err)
	}
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("truncate run file: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("seek run file: %w", err)
	}
	if _, err := f.Write(out); err != nil {
		return fmt.Errorf("write run file: %w", err)
	}
	return nil
}

func flock(fd uintptr, op int) error {
	for {
		err := syscall.Flock(int(fd), op)
		if err != syscall.EINTR {
			return err
		}
	}
}

func findRunFile(dir, featureID string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read runs dir %s: %w", dir, err)
	}
	prefix := "oneshot-" + featureID + "-"
	var matches []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".json") {
			matches = append(matches, e.Name())
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no run file found for feature %s in %s", featureID, dir)
	}
	slices.SortFunc(matches, func(a, b string) int {
		sa := strings.TrimSuffix(a, ".json")
		sb := strings.TrimSuffix(b, ".json")
		na := strings.TrimPrefix(sa, prefix)
		nb := strings.TrimPrefix(sb, prefix)
		va, ea := strconv.Atoi(na)
		vb, eb := strconv.Atoi(nb)
		if ea == nil && eb == nil {
			return cmp.Compare(va, vb) // ascending: last in slice = latest
		}
		return cmp.Compare(sa, sb) // fallback: string sort (e.g. timestamps)
	})
	return filepath.Join(dir, matches[len(matches)-1]), nil
}
