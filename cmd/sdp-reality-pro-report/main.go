package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"sdp_dev/internal/realitypro"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "reality-pro-report: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("sdp-reality-pro-report", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectRoot := fs.String("project-root", ".", "Workspace root containing repo-memory and review artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := realitypro.EmitReports(realitypro.ReportOptions{
		ProjectRoot: *projectRoot,
	})
	if err != nil {
		return err
	}

	for _, path := range result.WrittenPaths {
		fmt.Fprintf(stdout, "reality-pro-report: wrote %s\n", path)
	}
	fmt.Fprintf(stdout, "reality-pro-report: backlog=%d phases=%d verdict=%s->%s relationships=%d\n", result.BacklogCount, result.PhaseCount, result.CurrentVerdict, result.TargetVerdict, result.RelationshipCnt)
	return nil
}
