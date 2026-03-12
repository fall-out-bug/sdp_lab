package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sdp_dev/internal/realitypro"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "reality-pro-ingest: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("sdp-reality-pro-ingest", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectRoot := fs.String("project-root", ".", "Workspace root where .sdp/reality and docs/reality outputs will be written")
	repo := fs.String("repo", "", "Single repository path to ingest (default: project root)")
	reposet := fs.String("reposet", "", "Comma-separated repository paths for coordinated ingestion")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo != "" && *reposet != "" {
		return fmt.Errorf("choose either --repo or --reposet")
	}

	var repos []string
	switch {
	case *reposet != "":
		repos = splitReposet(*reposet)
	case *repo != "":
		repos = []string{*repo}
	}

	result, err := realitypro.Ingest(realitypro.Options{
		ProjectRoot: *projectRoot,
		Repos:       repos,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "reality-pro-ingest: wrote %s\n", result.RepoMemoryPath)
	fmt.Fprintf(stdout, "reality-pro-ingest: wrote %s\n", result.MultiRepoMapPath)
	fmt.Fprintf(stdout, "reality-pro-ingest: indexed %d repo(s)\n", result.RepoCount)
	return nil
}

func splitReposet(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}
