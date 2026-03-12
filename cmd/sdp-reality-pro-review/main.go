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
		fmt.Fprintf(os.Stderr, "reality-pro-review: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("sdp-reality-pro-review", flag.ContinueOnError)
	fs.SetOutput(stderr)

	projectRoot := fs.String("project-root", ".", "Workspace root containing .sdp/reality/repo-memory.json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := realitypro.Review(realitypro.ReviewOptions{
		ProjectRoot: *projectRoot,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "reality-pro-review: specialists=%s\n", strings.Join(result.Specialists, ","))
	fmt.Fprintf(stdout, "reality-pro-review: wrote %s\n", result.ConflictReportPath)
	fmt.Fprintf(stdout, "reality-pro-review: wrote %s\n", result.IntentGapReportPath)
	fmt.Fprintf(stdout, "reality-pro-review: gaps=%d conflicts=%d\n", result.GapCount, result.ConflictCount)
	return nil
}
