package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"sdp_dev/internal/harness"
)

func runContractCheck(contractPath, snapshotPath string) {
	if contractPath == "" || snapshotPath == "" {
		fmt.Fprintln(os.Stderr, "error: --check-contract requires --contract and --snapshot")
		os.Exit(2)
	}

	contract, err := harness.LoadTaskContract(contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	snapshot, err := harness.LoadTaskSnapshot(snapshotPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	report := harness.EvaluateCompliance(contract, snapshot)
	printJSON(report)
	if report.Blocked {
		os.Exit(1)
	}
	os.Exit(0)
}

func runClarificationFlow(contractPath, clarificationPath, clarificationText string, apply bool, approvedBy string) {
	if clarificationText != "" {
		if apply {
			fmt.Fprintln(os.Stderr, "error: --apply-clarification requires structured --clarification JSON, not --clarification-text")
			os.Exit(2)
		}
		decision := harness.ClassifyClarificationText(clarificationText)
		printJSON(decision)
		if decision.RequiresApproval {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if contractPath == "" || clarificationPath == "" {
		fmt.Fprintln(os.Stderr, "error: clarification flow requires --contract and --clarification (or use --clarification-text)")
		os.Exit(2)
	}

	contract, err := harness.LoadTaskContract(contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	change, err := harness.LoadClarificationChange(clarificationPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	decision := harness.ClassifyClarification(change)
	if !apply {
		printJSON(decision)
		if decision.RequiresApproval {
			os.Exit(1)
		}
		os.Exit(0)
	}

	decision, err = harness.ApplyClarification(contract, change, approvedBy, time.Now().UTC())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if err := harness.SaveTaskContract(contractPath, contract); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	printJSON(map[string]any{
		"decision": decision,
		"version":  contract.Version,
	})
	os.Exit(0)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "error: encode output: %v\n", err)
		os.Exit(2)
	}
}
