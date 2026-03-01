// Package main implements sdp-ready, a CLI that wraps Beads ready queue with SDP workstream mapping.
//
// Usage:
//
//	sdp ready [--format json|text] [--cache]
//
// Output includes Beads issues mapped to SDP workstream IDs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/orchestrate"
)

// WSOutput is the output format for sdp ready.
type WSOutput struct {
	WSID      string   `json:"ws_id"`
	BeadsID   string   `json:"beads_id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	Labels    []string `json:"labels,omitempty"`
	BlockedBy []string `json:"blocked_by,omitempty"`
	Ready     bool     `json:"ready"`
	CachedAt  string   `json:"cached_at,omitempty"`
}

// CacheEntry represents a cached ready result.
type CacheEntry struct {
	Issues   []WSOutput `json:"issues"`
	CachedAt time.Time  `json:"cached_at"`
}

func main() {
	format := flag.String("format", "text", "Output format: json or text")
	useCache := flag.Bool("cache", true, "Use cached results (5 min TTL)")
	cacheTTL := flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL")
	noCache := flag.Bool("no-cache", false, "Disable cache")
	phaseFilter := flag.Int("phase", 0, "Filter by roadmap phase (0=all)")
	flag.Parse()
	flag.Parse()

	if *noCache {
		*useCache = false
	}

	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err)
		os.Exit(1)
	}

	// Load mapping
	mapping, err := beads.LoadMapping(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: load mapping: %v\n", err)
		mapping = beads.EmptyMapping()
	}

	// Get ready issues (try cache first)
	var issues []beads.ReadyIssue
	cachePath := ".sdp/cache/ready.json"

	if *useCache {
		cached, err := loadCache(projectRoot, cachePath, *cacheTTL)
		if err == nil && cached != nil {
			issues = cached
		}
	}

	if len(issues) == 0 {
		// Use bd ready command
		issues, err = beads.ReadyCommand()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: bd ready: %v\n", err)
			os.Exit(1)
		}

		// Save to cache
		if *useCache {
			_ = saveCache(projectRoot, cachePath, issues)
		}
	}

	// Map to WS output (with optional phase filter)
	output := make([]WSOutput, 0, len(issues))
	for _, issue := range issues {
		ws := WSOutput{
			BeadsID:   issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			Labels:    issue.Labels,
			BlockedBy: issue.BlockedBy,
			Ready:     len(issue.BlockedBy) == 0,
		}
		ws.WSID = mapping.GetSDPID(issue.ID)

		// Filter by phase if specified
		if *phaseFilter > 0 {
			phase := getPhaseFromWSID(ws.WSID)
			if phase != *phaseFilter {
				continue
			}
		}

		output = append(output, ws)
	}

	// Output
	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
			os.Exit(1)
		}
	default:
		printText(output)
	}
}

func printText(issues []WSOutput) {
	ready := 0
	blocked := 0

	for _, issue := range issues {
		if issue.Ready {
			ready++
		} else {
			blocked++
		}
	}

	fmt.Printf("📋 Ready work (%d ready, %d blocked):\n\n", ready, blocked)

	for _, issue := range issues {
		status := "●"
		if !issue.Ready {
			status = "○"
		}

		priority := ""
		switch {
		case issue.Priority == 1:
			priority = "P1"
		case issue.Priority == 2:
			priority = "P2"
		case issue.Priority >= 3:
			priority = "P3"
		default:
			priority = fmt.Sprintf("P%d", issue.Priority)
		}

		wsRef := ""
		if issue.WSID != "" {
			wsRef = fmt.Sprintf(" [%s]", issue.WSID)
		}

		fmt.Printf("  %s [%s] %s%s: %s\n", status, priority, issue.BeadsID, wsRef, issue.Title)

		if !issue.Ready && len(issue.BlockedBy) > 0 {
			fmt.Printf("      blocked by: %s\n", strings.Join(issue.BlockedBy, ", "))
		}
	}
}

func loadCache(projectRoot, path string, ttl time.Duration) ([]beads.ReadyIssue, error) {
	fullPath := projectRoot + "/" + path
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	if time.Since(entry.CachedAt) > ttl {
		return nil, fmt.Errorf("cache expired")
	}

	// Convert WSOutput back to ReadyIssue
	issues := make([]beads.ReadyIssue, 0, len(entry.Issues))
	for _, ws := range entry.Issues {
		issues = append(issues, beads.ReadyIssue{
			Issue: beads.Issue{
				ID:        ws.BeadsID,
				Title:     ws.Title,
				Priority:  ws.Priority,
				Labels:    ws.Labels,
				BlockedBy: ws.BlockedBy,
			},
			WSID: ws.WSID,
		})
	}

	return issues, nil
}

func saveCache(projectRoot, path string, issues []beads.ReadyIssue) error {
	// Ensure cache dir
	cacheDir := projectRoot + "/.sdp/cache"
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	// Convert to WSOutput for caching
	output := make([]WSOutput, 0, len(issues))
	for _, issue := range issues {
		output = append(output, WSOutput{
			BeadsID:   issue.ID,
			Title:     issue.Title,
			Priority:  issue.Priority,
			Labels:    issue.Labels,
			BlockedBy: issue.BlockedBy,
			Ready:     len(issue.BlockedBy) == 0,
		})
	}

	entry := CacheEntry{
		Issues:   output,
		CachedAt: time.Now(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheDir+"/ready.json", data, 0o644)
}

// getPhaseFromWSID extracts the phase number from a workstream ID.
// WSID format: 00-XXX-YY where XXX is the feature number.
// Feature → Phase mapping based on ROADMAP.md.
func getPhaseFromWSID(wsid string) int {
	if wsid == "" {
		return 0
	}

	// Parse feature number from WSID (00-XXX-YY)
	parts := strings.Split(wsid, "-")
	if len(parts) != 3 {
		return 0
	}

	featureStr := parts[1]
	var feature int
	if _, err := fmt.Sscanf(featureStr, "%d", &feature); err != nil {
		return 0
	}

	// Feature → Phase mapping (from ROADMAP.md)
	// Phase 0: F001-F027 (Agent Loop Reliability) - done
	// Phase 1-2: F028-F052 (Enforcement Foundation, Archive) - done
	// Phase 3: in-toto Migration (F053 partial)
	// Phase 4: Auto-Attestation
	// Phase 5: Policy-as-Code (F059, F061, F063)
	// Phase 6: Runtime Governance
	// Phase 7: Ecosystem & Launch
	// Phase 8-9: K8s (F060, F062)
	switch {
	case feature >= 1 && feature <= 27:
		return 0 // Phase 0 - Agent Loop Reliability (done)
	case feature >= 28 && feature <= 52:
		return 2 // Phase 1-2 - Enforcement Foundation, Archive (done)
	case feature == 53:
		return 3 // Phase 3 - in-toto Migration
	case feature == 54:
		return 3 // Phase 3 - Continuous Protocol Improvement
	case feature >= 55 && feature <= 58:
		return 4 // Phase 4 - Auto-Attestation
	case feature == 59:
		return 5 // Phase 5 - OhMyOpenCode Integration
	case feature == 60:
		return 8 // Phase 8-9 - Gas Town (K8s)
	case feature == 61:
		return 5 // Phase 5 - Beads Integration
	case feature == 62:
		return 8 // Phase 8-9 - vibe-kanban (K8s)
	case feature == 63:
		return 5 // Phase 5 - opencode-mem Integration
	default:
		return 0
	}
}
