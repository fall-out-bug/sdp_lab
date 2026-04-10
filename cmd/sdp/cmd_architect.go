package main

import (
	"flag"
	"fmt"
	"os"
)

func runArchitect(args []string) {
	if len(args) < 1 {
		architectUsage()
		os.Exit(2)
	}
	switch args[0] {
	case "analyze":
		runArchitectAnalyze(args[1:])
	case "c4":
		fmt.Println("sdp architect c4: not implemented yet")
	case "contracts":
		fmt.Println("sdp architect contracts: not implemented yet")
	case "conform":
		fmt.Println("sdp architect conform: not implemented yet")
	case "greenfield":
		fmt.Println("sdp architect greenfield: not implemented yet")
	default:
		architectUsage()
		os.Exit(2)
	}
}

func runArchitectAnalyze(args []string) {
	fs := flag.NewFlagSet("architect analyze", flag.ExitOnError)
	allowExtLLM := fs.Bool("allow-external-llm", false, "allow sending sanitized data to cloud LLMs")
	skipGit := fs.Bool("skip-git", false, "skip git history analysis")
	lang := fs.String("language", "", "comma-separated language filter (e.g. go,python)")
	_ = fs.Parse(args)
	repoPath := fs.Arg(0)
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sdp architect analyze [flags] <repo-path>")
		os.Exit(2)
	}
	fmt.Printf("AI Architect: analyzing %s\n", repoPath)
	fmt.Printf("  allow-external-llm: %v\n", *allowExtLLM)
	fmt.Printf("  skip-git: %v\n", *skipGit)
	if *lang != "" {
		fmt.Printf("  languages: %s\n", *lang)
	}
	fmt.Println("  status: not implemented yet")
}

func architectUsage() {
	fmt.Fprintln(os.Stderr, "usage: sdp architect <analyze|c4|contracts|conform|greenfield> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  analyze <repo-path>   Full architecture analysis (probe mode)")
	fmt.Fprintln(os.Stderr, "  c4 <repo-path>        Generate C4 diagrams only")
	fmt.Fprintln(os.Stderr, "  contracts <repo-path>  Discover integration contracts")
	fmt.Fprintln(os.Stderr, "  conform <repo-path>    Run conformance check")
	fmt.Fprintln(os.Stderr, "  greenfield             Guided architecture conversation")
}
