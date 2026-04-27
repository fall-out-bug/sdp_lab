package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
)

func runLimits() error {
	fs := flag.NewFlagSet("limits", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	providers := []harness.Provider{
		harness.NewZAIProvider(),
	}

	checker := &dispatch.LimitsChecker{Providers: providers}
	ctx := context.Background()
	limits := checker.CheckAll(ctx)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(limits)
	}

	fmt.Print(dispatch.FormatLimitsTable(limits))
	return nil
}
