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

	// Map to WS output
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
		case issue.Priority >= 1:
			priority = "P1"
		case issue.Priority == 2:
			priority = "P2"
		case issue.Priority >= 3:
			priority = "P3"
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
