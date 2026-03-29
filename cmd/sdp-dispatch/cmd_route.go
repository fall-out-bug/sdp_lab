package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/dispatch"
)

func runRoute() error {
	fs := flag.NewFlagSet("route", flag.ExitOnError)
	task := fs.String("task", "", "Workstream ID or path")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	projectRoot := fs.String("project", "", "Project root (default: cwd)")
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
		return fmt.Errorf("no capability profiles found; run 'sdp dispatch bench' to generate profiles")
	}

	router := &dispatch.Router{Profiles: profiles}

	pkt := dispatch.ContextPacketSummary{
		Phase:      "build",
		Workstream: *task,
	}
	classification := dispatch.Classify(pkt)

	ctx := context.Background()
	dec, err := router.Route(ctx, classification, nil)
	if err != nil {
		return fmt.Errorf("routing: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(dec)
	}

	fmt.Printf("harness:  %s\n", dec.Harness)
	fmt.Printf("provider: %s\n", dec.Provider)
	fmt.Printf("model:    %s\n", dec.Model)
	fmt.Printf("score:    %.4f\n", dec.Score)
	fmt.Printf("reason:   %s\n", dec.Reason)
	if len(dec.Alternatives) > 0 {
		fmt.Println("alternatives:")
		for _, alt := range dec.Alternatives {
			fmt.Printf("  - %s (score: %.4f)\n", alt.Harness, alt.Score)
		}
	}
	return nil
}
