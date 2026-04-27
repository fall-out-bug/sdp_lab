package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/cli"
)

func runBoard(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp board <build|show>")
		os.Exit(2)
	}
	switch args[0] {
	case "build":
		runBoardBuild(args[1:])
	case "show":
		runBoardShow(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp board <build|show>")
		os.Exit(2)
	}
}

func runBoardBuild(args []string) {
	fs := flag.NewFlagSet("board-build", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	_ = fs.Parse(args)
	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err)
			os.Exit(1)
		}
		printJSON(snap)
		return
	}
	port, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}
	printJSON(port)
}

func runBoardShow(args []string) {
	fs := flag.NewFlagSet("board-show", flag.ExitOnError)
	project := fs.String("project", "", "optional project id")
	asJSON := fs.Bool("json", false, "render raw JSON instead of the default human summary")
	_ = fs.Parse(args)

	store := openStore()
	if *project != "" {
		snap, err := store.BuildProjectSnapshot(*project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: build project snapshot: %v\n", err)
			os.Exit(1)
		}
		if *asJSON {
			printJSON(snap)
			return
		}
		fmt.Println(cli.RenderProjectBoard(snap))
		return
	}

	port, err := store.BuildPortfolioSnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: build portfolio snapshot: %v\n", err)
		os.Exit(1)
	}
	if *asJSON {
		printJSON(port)
		return
	}
	fmt.Println(cli.RenderPortfolioBoard(port))
}
