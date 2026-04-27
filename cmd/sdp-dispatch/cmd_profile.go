//go:build sdp_experimental

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

func runProfile() error {
	fs := flag.NewFlagSet("profile", flag.ExitOnError)
	projectRoot := fs.String("project", "", "Project root (default: cwd)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	root, err := resolveProjectRoot(*projectRoot)
	if err != nil {
		return err
	}

	store := dispatch.NewProfileStore(root)
	profiles, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("loading profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found. Run 'sdp dispatch bench' to generate profiles.")
		return nil
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profiles)
	}

	for _, p := range profiles {
		fmt.Printf("harness: %s  provider: %s  model: %s\n", p.Harness, p.Provider, p.Model)
		if len(p.Capabilities) == 0 {
			fmt.Println("  (no capability data)")
			continue
		}

		// Sort capability keys for deterministic output.
		keys := make([]string, 0, len(p.Capabilities))
		for k := range p.Capabilities {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			score := p.Capabilities[k]
			fmt.Printf("  %-30s  pass_rate=%.2f  avg_duration=%.1fm  samples=%d\n",
				k, score.TestPassRate, score.AvgDuration, score.SampleCount)
		}
	}
	return nil
}

// resolveProjectRoot returns root if non-empty, otherwise the current working directory.
func resolveProjectRoot(root string) (string, error) {
	if root != "" {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return wd, nil
}
