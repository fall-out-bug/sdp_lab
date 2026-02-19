package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/evidence"
)

func main() {
	issueID := flag.String("issue", "", "Issue ID")
	evidencePath := flag.String("evidence", "", "Path to evidence file (optional)")
	prePublish := flag.Bool("prepublish", false, "Allow missing trace.pr_url before PR creation")
	flag.Parse()

	if *issueID == "" {
		fmt.Fprintln(os.Stderr, "--issue is required")
		os.Exit(2)
	}

	path := *evidencePath
	if path == "" {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, ".sdp", "evidence", *issueID+".json")
	}

	res, err := evidence.ValidateStrictFile(path, !*prePublish)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate strict evidence: %v\n", err)
		os.Exit(1)
	}

	out := map[string]any{
		"issue":   *issueID,
		"ok":      res.OK,
		"reason":  res.Reason,
		"missing": res.Missing,
		"mode":    map[bool]string{true: "prepublish", false: "publish"}[*prePublish],
		"path":    path,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
	if !res.OK {
		os.Exit(2)
	}
}
