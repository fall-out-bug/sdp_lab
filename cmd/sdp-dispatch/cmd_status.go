package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/harness"
)

// statusOutput is the JSON-serialisable representation of dispatch status.
type statusOutput struct {
	Decision      *dispatch.DispatchDecision `json:"decision,omitempty"`
	ProfileCount  int                        `json:"profile_count"`
	ColdStart     string                     `json:"cold_start_strategy"`
	LimitsSummary map[string]*harness.Limits `json:"limits,omitempty"`
	Staleness     *stalenessSummary          `json:"staleness,omitempty"`
}

// stalenessSummary counts profiles by freshness status.
type stalenessSummary struct {
	Fresh   int `json:"fresh"`
	Stale   int `json:"stale"`
	Expired int `json:"expired"`
}

// computeStalenessSummary counts freshness of profiles using default config.
func computeStalenessSummary(profiles []*dispatch.CapabilityProfile) *stalenessSummary {
	if len(profiles) == 0 {
		return nil
	}
	cfg := dispatch.DefaultStalenessConfig
	now := time.Now()
	s := &stalenessSummary{}
	for _, p := range profiles {
		switch dispatch.CheckFreshness(p, cfg, now) {
		case dispatch.FreshnessFresh:
			s.Fresh++
		case dispatch.FreshnessStale:
			s.Stale++
		case dispatch.FreshnessExpired:
			s.Expired++
		}
	}
	return s
}

func runStatus() error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	projectRoot := fs.String("project", "", "Project root (default: cwd)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	root, err := resolveProjectRoot(*projectRoot)
	if err != nil {
		return err
	}

	// Load last dispatch decision (optional — may not exist yet).
	dec, decErr := dispatch.LoadDecision(root)

	// Load profiles.
	store := dispatch.NewProfileStore(root)
	profiles, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("loading profiles: %w", err)
	}

	// Check limits via z.ai provider.
	providers := []harness.Provider{
		harness.NewZAIProvider(),
	}
	checker := &dispatch.LimitsChecker{Providers: providers}
	ctx := context.Background()
	limits := checker.CheckAll(ctx)

	coldStrategy := string(dispatch.ColdStartCapabilityHeuristic)
	staleSummary := computeStalenessSummary(profiles)

	if *jsonOut {
		out := statusOutput{
			Decision:      dec,
			ProfileCount:  len(profiles),
			ColdStart:     coldStrategy,
			LimitsSummary: limits,
			Staleness:     staleSummary,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Human-readable output.
	fmt.Println("=== Dispatch Status ===")
	fmt.Println()

	if decErr != nil {
		fmt.Println("Last decision: none (no dispatch-decision.json found)")
	} else {
		fmt.Printf("Last decision:\n")
		fmt.Printf("  harness:   %s\n", dec.Harness)
		fmt.Printf("  provider:  %s\n", dec.Provider)
		fmt.Printf("  model:     %s\n", dec.Model)
		fmt.Printf("  score:     %.4f\n", dec.Score)
		fmt.Printf("  reason:    %s\n", dec.Reason)
		fmt.Printf("  timestamp: %s\n", dec.Timestamp)
		if dec.ColdStart {
			fmt.Printf("  cold_start: true\n")
		}
	}

	fmt.Println()
	fmt.Printf("Profiles:           %d\n", len(profiles))
	if staleSummary != nil {
		fmt.Printf("  profiles: %d fresh, %d stale, %d expired\n",
			staleSummary.Fresh, staleSummary.Stale, staleSummary.Expired)
	}
	fmt.Printf("Cold-start strategy: %s\n", coldStrategy)
	fmt.Println()

	if len(limits) > 0 {
		fmt.Println("Limits:")
		fmt.Print(dispatch.FormatLimitsTable(limits))
	} else {
		fmt.Println("Limits: no provider data available")
	}

	return nil
}
