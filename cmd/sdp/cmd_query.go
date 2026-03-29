package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/cli"
	"sdp_dev/internal/control"
)

func runAttention(args []string) {
	fs := flag.NewFlagSet("attention", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "render raw JSON instead of the default human summary")
	_ = fs.Parse(args)

	store := openStore()
	snap, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(snap)
		return
	}
	fmt.Println(cli.RenderAttention(snap))
}

func runWhy(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp why <card-id>")
		os.Exit(2)
	}
	store := openStore()
	blockers, err := store.WhyBlocked(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	control.PrintWhyBlocked(blockers, nil)
	if *flagSetFrom(args, "--json") {
		printJSON(blockers)
	}
}

func runNext(args []string) {
	fs := flag.NewFlagSet("next", flag.ExitOnError)
	limit := fs.Int("limit", 10, "max items to show")
	asJSON := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)

	store := openStore()
	items, err := store.WhatNext(*limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(items)
		return
	}
	if len(items) == 0 {
		fmt.Println("📭  No actionable items.")
		return
	}
	for _, item := range items {
		fmt.Printf("  ▶  %s: %s [%s]\n", item.ID, item.Title, item.Status)
	}
}

func runMissing(args []string) {
	store := openStore()
	projectID := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectID = args[0]
	}
	asJSON := flagSetFrom(args, "--json")

	missing, err := store.WhatMissing(projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		printJSON(missing)
		return
	}
	control.PrintMissing(missing, nil)
}

func runApprove(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sdp approve <card-id>")
		os.Exit(2)
	}
	store := openStore()
	beadsRepo := store.BeadsRepo()
	if beadsRepo == nil {
		fmt.Fprintln(os.Stderr, "error: approve requires beads or dual mode (set SDP_REPO_MODE)")
		os.Exit(1)
	}
	if err := beadsRepo.ResolveGate(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve gate: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅  Gate %s resolved.\n", args[0])
}
