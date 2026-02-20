package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/evidence"
)

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func main() {
	runID := flag.String("run-id", "", "Run identifier")
	analystLog := flag.String("analyst-log", "", "Path to analyst log file")
	coderLog := flag.String("coder-log", "", "Path to coder log file")
	reviewerLog := flag.String("reviewer-log", "", "Path to reviewer log file")
	flag.Parse()

	if *runID == "" || *analystLog == "" || *coderLog == "" || *reviewerLog == "" {
		fmt.Fprintln(os.Stderr, "--run-id, --analyst-log, --coder-log, --reviewer-log are required")
		os.Exit(2)
	}

	aLog, err := readFile(*analystLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cLog, err := readFile(*coderLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rLog, err := readFile(*reviewerLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	results := []evidence.RoleGateResult{
		evidence.ValidateRoleLog("analyst", *runID, aLog),
		evidence.ValidateRoleLog("coder", *runID, cLog),
		evidence.ValidateRoleLog("reviewer", *runID, rLog),
	}

	ok := true
	for _, r := range results {
		if !r.OK {
			ok = false
		}
	}
	out := map[string]any{"run_id": *runID, "ok": ok, "roles": results}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
	if !ok {
		os.Exit(2)
	}
}
