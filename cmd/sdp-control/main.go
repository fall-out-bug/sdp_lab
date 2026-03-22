package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/control"
	"sdp_dev/internal/orchestrate"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	switch cmd {
	case "card-create":
		runCardCreate(os.Args[2:])
	case "board-build":
		runBoardBuild(os.Args[2:])
	case "board-show":
		runBoardShow(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sdp-control <card-create|board-build|board-show> [flags]")
}

func openStore() *control.Store {
	wd, err := os.Getwd()
	if err != nil { fmt.Fprintf(os.Stderr, "error: get cwd: %v\n", err); os.Exit(1) }
	root, err := orchestrate.FindProjectRoot(wd)
	if err != nil { fmt.Fprintf(os.Stderr, "error: find project root: %v\n", err); os.Exit(1) }
	store, err := control.Open(root)
	if err != nil { fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err); os.Exit(1) }
	return store
}

func runCardCreate(args []string) {
	fs := flag.NewFlagSet("card-create", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	title := fs.String("title", "", "card title")
	raw := fs.String("raw", "", "raw request text")
	_ = fs.Parse(args)
	if *project == "" || *title == "" || *raw == "" { fmt.Fprintln(os.Stderr, "error: --project, --title, and --raw are required"); os.Exit(2) }
	store := openStore()
	card, err := store.CreateCard(*project, *title, *raw)
	if err != nil { fmt.Fprintf(os.Stderr, "error: create card: %v\n", err); os.Exit(1) }
	enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(card)
}

func runBoardBuild(args []string) {
	fs := flag.NewFlagSet("board-build", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	_ = fs.Parse(args)
	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil { fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err); os.Exit(1) }
		enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(snap); return
	}
	port, err := store.BuildPortfolioSnapshot()
	if err != nil { fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err); os.Exit(1) }
	enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(port)
}

func runBoardShow(args []string) {
	fs := flag.NewFlagSet("board-show", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	_ = fs.Parse(args)
	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil { fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err); os.Exit(1) }
		enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(snap); return
	}
	port, err := store.BuildPortfolioSnapshot()
	if err != nil { fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err); os.Exit(1) }
	enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(port)
}
