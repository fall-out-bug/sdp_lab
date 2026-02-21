package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// runFunc is the function used to run commands. Override in tests for mocking.
var runFunc = run

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func extractJSON(out []byte) []byte {
	for i, b := range out {
		if b == '[' || b == '{' {
			return out[i:]
		}
	}
	return out
}

func runComponent(binary string, goPkg string, args ...string) ([]byte, error) {
	out, _, err := runComponentWithFallback(binary, goPkg, args...)
	return out, err
}

func runComponentWithFallback(binary string, goPkg string, args ...string) ([]byte, bool, error) {
	if _, err := exec.LookPath(binary); err == nil {
		out, runErr := runFunc(binary, args...)
		return out, false, runErr
	}
	goArgs := append([]string{"run", goPkg}, args...)
	out, runErr := runFunc("go", goArgs...)
	return out, true, runErr
}

func discardBeadsSyncNoise() {
	namesRaw, err := run("git", "diff", "--name-only")
	if err != nil {
		return
	}
	cachedRaw, err := run("git", "diff", "--cached", "--name-only")
	if err != nil {
		return
	}
	nameSet := make(map[string]struct{})
	for _, n := range strings.Fields(strings.TrimSpace(string(namesRaw))) {
		nameSet[n] = struct{}{}
	}
	for _, n := range strings.Fields(strings.TrimSpace(string(cachedRaw))) {
		nameSet[n] = struct{}{}
	}
	if len(nameSet) == 0 {
		return
	}
	for n := range nameSet {
		if n != ".beads/metadata.json" && n != ".beads/issues.jsonl" {
			return
		}
	}
	_, _ = run("git", "reset", "HEAD", "--", ".beads/metadata.json", ".beads/issues.jsonl")
	_, _ = run("git", "checkout", "--", ".beads/metadata.json", ".beads/issues.jsonl")
}

func hasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return true, nil
	}
	return false, err
}

func parseClaim(out []byte) (claimResult, error) {
	var r claimResult
	if err := json.Unmarshal(extractJSON(out), &r); err != nil {
		return r, err
	}
	if r.IssueID == "" || r.Branch == "" {
		return r, errors.New("invalid claim payload")
	}
	return r, nil
}

func loadIssue(issueID string) (issueDetail, error) {
	out, err := runFunc("bd", "show", issueID, "--json")
	if err != nil {
		return issueDetail{}, err
	}
	var list []issueDetail
	jsonOut := extractJSON(out)
	if err := json.Unmarshal(jsonOut, &list); err == nil && len(list) > 0 {
		return list[0], nil
	}
	var it issueDetail
	if err := json.Unmarshal(jsonOut, &it); err != nil {
		return issueDetail{}, err
	}
	return it, nil
}

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}
