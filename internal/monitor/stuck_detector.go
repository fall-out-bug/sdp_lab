// Package monitor provides agent health monitoring capabilities.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// stuckDetector monitors session evidence for stuck agents.
type stuckDetector struct {
	mu            sync.Mutex
	sessionPath   string
	timeout       time.Duration
	checkTicker   *time.Ticker
	onStuck       func(sessionID string, lastEvent time.Time)
	onRecovered   func(sessionID string)
	stuckSessions map[string]time.Time
}

// stuckDetectorConfig configures the stuck detector.
type stuckDetectorConfig struct {
	// SessionPath is the path to the session evidence directory.
	// If empty, uses XDG_DATA_HOME/sdp/log
	SessionPath string

	// Timeout is the duration after which an agent is considered stuck.
	// Default: 5 minutes
	Timeout time.Duration

	// CheckInterval is how often to check for stuck agents.
	// Default: 30 seconds
	CheckInterval time.Duration

	// OnStuck is called when an agent is detected as stuck.
	OnStuck func(sessionID string, lastEvent time.Time)

	// OnRecovered is called when a stuck agent recovers.
	OnRecovered func(sessionID string)
}

// DefaultStuckTimeout is the default timeout for stuck detection.
const defaultStuckTimeout = 5 * time.Minute

// DefaultCheckInterval is the default interval for checking stuck agents.
const defaultCheckInterval = 30 * time.Second

// newStuckDetector creates a new stuck detector.
func newStuckDetector(cfg stuckDetectorConfig) (*stuckDetector, error) {
	sessionPath := cfg.SessionPath
	if sessionPath == "" {
		// Try XDG_DATA_HOME first, then fallback to .sdp/log
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData != "" {
			sessionPath = filepath.Join(xdgData, "sdp/log")
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				sessionPath = ".sdp/log"
			} else {
				sessionPath = filepath.Join(homeDir, ".local/share/sdp/log")
			}
		}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultStuckTimeout
	}

	checkInterval := cfg.CheckInterval
	if checkInterval == 0 {
		checkInterval = defaultCheckInterval
	}

	sd := &stuckDetector{
		sessionPath:   sessionPath,
		timeout:       timeout,
		onStuck:       cfg.OnStuck,
		onRecovered:   cfg.OnRecovered,
		stuckSessions: make(map[string]time.Time),
		checkTicker:   time.NewTicker(checkInterval),
	}

	return sd, nil
}

// start begins monitoring for stuck agents.
func (sd *stuckDetector) start(ctx context.Context) {
	go sd.run(ctx)
}

func (sd *stuckDetector) run(ctx context.Context) {
	defer func() {
		if sd.checkTicker != nil {
			sd.checkTicker.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sd.checkTicker.C:
			sd.check(ctx)
		}
	}
}

// check looks for stuck sessions.
func (sd *stuckDetector) check(ctx context.Context) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Find all session files
	files, err := filepath.Glob(filepath.Join(sd.sessionPath, "session-*.jsonl"))
	if err != nil {
		return
	}

	// Track which sessions are still active
	activeSessions := make(map[string]bool)

	for _, file := range files {
		sessionID, lastEvent, err := sd.getLastEventTime(file)
		if err != nil {
			continue
		}

		activeSessions[sessionID] = true

		timeSinceLastEvent := time.Since(lastEvent)

		if timeSinceLastEvent > sd.timeout {
			// Agent is stuck
			if _, wasStuck := sd.stuckSessions[sessionID]; !wasStuck {
				sd.stuckSessions[sessionID] = lastEvent
				if sd.onStuck != nil {
					sd.onStuck(sessionID, lastEvent)
				}
			}
		} else {
			// Agent is active
			if _, wasStuck := sd.stuckSessions[sessionID]; wasStuck {
				delete(sd.stuckSessions, sessionID)
				if sd.onRecovered != nil {
					sd.onRecovered(sessionID)
				}
			}
		}
	}

	// Clean up sessions that no longer exist
	for sessionID := range sd.stuckSessions {
		if !activeSessions[sessionID] {
			delete(sd.stuckSessions, sessionID)
		}
	}
}

// getLastEventTime reads the last event timestamp from a session file.
func (sd *stuckDetector) getLastEventTime(file string) (sessionID string, lastEvent time.Time, err error) {
	// Extract session ID from filename
	sessionID = filepath.Base(file)
	sessionID = sessionID[len("session-"):]
	sessionID = sessionID[:len(sessionID)-len(".jsonl")]

	// Get file modification time as fallback
	info, err := os.Stat(file)
	if err != nil {
		return sessionID, time.Time{}, err
	}
	lastEvent = info.ModTime()

	// Try to read the last line for actual timestamp
	f, err := os.Open(file)
	if err != nil {
		return sessionID, lastEvent, nil
	}
	defer func() { _ = f.Close() }()

	// Read backwards from end to find last complete line
	buf := make([]byte, 4096)
	offset, err := f.Seek(0, 2)
	if err != nil {
		return sessionID, lastEvent, nil
	}

	for offset > 0 {
		readSize := int64(len(buf))
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		_, err := f.Seek(offset, 0)
		if err != nil {
			break
		}

		n, err := f.Read(buf[:readSize])
		if err != nil {
			break
		}

		// Find last newline
		data := buf[:n]
		lastNewline := -1
		for i := len(data) - 1; i >= 0; i-- {
			if data[i] == '\n' {
				lastNewline = i
				break
			}
		}

		if lastNewline >= 0 {
			// Parse the last complete line
			line := data[lastNewline+1:]
			if len(line) > 0 {
				var event struct {
					Timestamp time.Time `json:"timestamp"`
				}
				if err := json.Unmarshal(line, &event); err == nil && !event.Timestamp.IsZero() {
					return sessionID, event.Timestamp, nil
				}
			}
			break
		}
	}

	return sessionID, lastEvent, nil
}

// isStuck checks if a specific session is stuck.
func (sd *stuckDetector) isStuck(sessionID string) bool {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	_, stuck := sd.stuckSessions[sessionID]
	return stuck
}

// stats returns monitoring statistics.
func (sd *stuckDetector) stats() stats {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	// Copy the map to prevent external mutation
	stuckCopy := make(map[string]time.Time, len(sd.stuckSessions))
	for k, v := range sd.stuckSessions {
		stuckCopy[k] = v
	}

	return stats{
		StuckCount:    len(sd.stuckSessions),
		StuckSessions: stuckCopy,
		Timeout:       sd.timeout,
	}
}

// stats contains monitoring statistics.
type stats struct {
	StuckCount    int                  `json:"stuck_count"`
	StuckSessions map[string]time.Time `json:"stuck_sessions"`
	Timeout       time.Duration        `json:"timeout"`
}

// String returns a human-readable representation of stats.
func (s stats) String() string {
	return fmt.Sprintf("StuckAgents: %d, Timeout: %v", s.StuckCount, s.Timeout)
}
