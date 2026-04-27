package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/profile"
)

var (
	profileFlag = flag.String("profile", "", "Profile to provision (oss-combine)")
	dryRun      = flag.Bool("dry-run", false, "Show planned operations without executing")
	rollback    = flag.Bool("rollback", false, "Remove provisioned state")
	list        = flag.Bool("list", false, "List available profiles")
	help        = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), `sdp-up - Provision SDP integration environment

Usage:
  sdp up --profile <name>     Provision environment
  sdp up --profile <name> --rollback   Remove provisioned state
  sdp up --list               List available profiles

Flags:
`)
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), `
Examples:
  sdp up --profile oss-combine
  sdp up --profile oss-combine --dry-run
  sdp up --profile oss-combine --rollback
`)
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *list {
		fmt.Println("Available profiles:")
		for _, name := range profile.AvailableProfiles() {
			fmt.Printf("  - %s\n", name)
		}
		os.Exit(0)
	}

	if *profileFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --profile is required")
		fmt.Fprintln(os.Stderr, "Run 'sdp up --help' for usage")
		os.Exit(1)
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			break
		}
		projectRoot = parent
	}

	p, err := profile.GetProfile(*profileFlag, projectRoot, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Profile: %s\n", p.Name())
	fmt.Printf("Description: %s\n\n", p.Description())

	if *dryRun {
		fmt.Println("=== DRY RUN MODE ===")
		fmt.Println("The following operations would be performed:")
	}

	if *rollback {
		if err := rollbackProfile(p); err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := provisionProfile(p); err != nil {
			fmt.Fprintf(os.Stderr, "Provision failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func provisionProfile(p profile.Profile) error {
	return p.Provision()
}

func rollbackProfile(p profile.Profile) error {
	return p.Rollback()
}
