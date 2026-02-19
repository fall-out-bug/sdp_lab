package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type claimResult struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
	Branch  string `json:"branch"`
}

type issueDetail struct {
	ID     string   `json:"id"`
	Labels []string `json:"labels"`
}

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
	if _, err := exec.LookPath(binary); err == nil {
		return run(binary, args...)
	}
	goArgs := append([]string{"run", goPkg}, args...)
	return run("go", goArgs...)
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
	out, err := run("bd", "show", issueID, "--json")
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

func patchSlugifyForTrim(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	old := "\tif len(t) > 48 {\n\t\treturn t[:48]\n\t}\n\treturn t\n"
	new := "\tif len(t) > 48 {\n\t\tt = t[:48]\n\t\tt = strings.Trim(t, \"-\")\n\t}\n\tif t == \"\" {\n\t\treturn \"task\"\n\t}\n\treturn t\n"
	if !strings.Contains(content, old) {
		return errors.New("slugify block not found")
	}
	content = strings.Replace(content, old, new, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addSlugifyRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	needle := "func TestDecideCriticalEscalates(t *testing.T) {"
	insert := "func TestBuildBranchNameTrimsTrailingDashAfterTruncation(t *testing.T) {\n\ttitle := strings.Repeat(\"word-\", 20)\n\tbranch := BuildBranchName(\"id-4\", title)\n\tif strings.HasSuffix(branch, \"-\") {\n\t\tt.Fatalf(\"expected no trailing dash, got %s\", branch)\n\t}\n}\n\n"
	if strings.Contains(content, "TestBuildBranchNameTrimsTrailingDashAfterTruncation") {
		return nil
	}
	if !strings.Contains(content, needle) {
		return errors.New("test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	if !strings.Contains(content, "\"strings\"") {
		content = strings.Replace(content, "import \"testing\"", "import (\n\t\"strings\"\n\t\"testing\"\n)", 1)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func patchModelChainUnknownFallback(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "model_chain.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	old := "func ResolveFallbackSequence(start string) []string {\n\tsequence := []string{start}\n\tcurrent := start\n"
	new := "func ResolveFallbackSequence(start string) []string {\n\tif start == \"\" || !AllowedModel(start) {\n\t\tstart = DefaultModel()\n\t}\n\tsequence := []string{start}\n\tcurrent := start\n"
	if strings.Contains(content, "!AllowedModel(start)") {
		return nil
	}
	if !strings.Contains(content, old) {
		return errors.New("model_chain sequence block not found")
	}
	content = strings.Replace(content, old, new, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addModelChainRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "model_chain_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "TestResolveFallbackSequenceUnknownStartsFromDefault") {
		return nil
	}
	needle := "func TestResolveFallbackSequence(t *testing.T) {"
	insert := "func TestResolveFallbackSequenceUnknownStartsFromDefault(t *testing.T) {\n\tseq := ResolveFallbackSequence(\"unknown-model\")\n\tif len(seq) != 3 {\n\t\tt.Fatalf(\"expected 3 steps, got %d\", len(seq))\n\t}\n\tif seq[0] != \"glm-5\" || seq[1] != \"glm-4.7\" || seq[2] != \"escalated\" {\n\t\tt.Fatalf(\"unexpected sequence: %#v\", seq)\n\t}\n}\n\n"
	if !strings.Contains(content, needle) {
		return errors.New("model_chain test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func updateEvidence(issueID, branch string, changedFiles []string) error {
	path := filepath.Join(".sdp", "evidence", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}
	execSection, _ := payload["execution"].(map[string]any)
	if execSection == nil {
		execSection = map[string]any{}
		payload["execution"] = execSection
	}
	execSection["branch"] = branch
	execSection["changed_files"] = changedFiles
	execSection["claimed_issue_ids"] = []string{issueID}

	trace, _ := payload["trace"].(map[string]any)
	if trace == nil {
		trace = map[string]any{}
		payload["trace"] = trace
	}
	trace["branch"] = branch
	trace["beads_ids"] = []string{issueID}

	verification, _ := payload["verification"].(map[string]any)
	if verification == nil {
		verification = map[string]any{}
		payload["verification"] = verification
	}
	verification["tests"] = []string{"go test ./..."}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func main() {
	claimOut, err := runComponent("autonomy-worker", "./cmd/autonomy-worker")
	if err != nil {
		if strings.Contains(err.Error(), "No eligible autonomy tasks found") {
			fmt.Println("No eligible autonomy tasks found")
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.Contains(string(claimOut), "No eligible autonomy tasks found") {
		fmt.Println("No eligible autonomy tasks found")
		return
	}
	claim, err := parseClaim(claimOut)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	issue, err := loadIssue(claim.IssueID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	workstream := ""
	if hasLabel(issue.Labels, "workstream:policy-slugify-trim") {
		workstream = "policy-slugify-trim"
	}
	if hasLabel(issue.Labels, "workstream:model-chain-default-fallback") {
		workstream = "model-chain-default-fallback"
	}
	if workstream == "" {
		fmt.Fprintf(os.Stderr, "unsupported workstream labels for issue %s\n", claim.IssueID)
		os.Exit(1)
	}

	if _, err := run("git", "checkout", "-b", claim.Branch); err != nil {
		_, err = run("git", "checkout", claim.Branch)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	changedFiles := []string{}
	switch workstream {
	case "policy-slugify-trim":
		if err := patchSlugifyForTrim("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addSlugifyRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/decision.go", "internal/policy/decision_test.go"}
	case "model-chain-default-fallback":
		if err := patchModelChainUnknownFallback("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addModelChainRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/model_chain.go", "internal/policy/model_chain_test.go"}
	}

	if _, err := run("go", "test", "./..."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := updateEvidence(claim.IssueID, claim.Branch, changedFiles); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "review", "--apply"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runComponent("pr-gate", "./cmd/pr-gate", "--issue", claim.IssueID, "--prepublish"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := []string{"add"}
	args = append(args, changedFiles...)
	args = append(args, ".beads/issues.jsonl")
	if _, err := run("git", args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	commitBody := "Implement workstream changes with regression coverage."
	if workstream == "policy-slugify-trim" {
		commitBody = "Fix slugify truncation and add regression coverage."
	}
	if workstream == "model-chain-default-fallback" {
		commitBody = "Make unknown model fallback deterministic and add regression coverage."
	}
	if _, err := run("git", "commit", "-m", "worker: implement "+claim.IssueID, "-m", commitBody); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := run("git", "push", "-u", "origin", claim.Branch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	bodyPath := filepath.Join(".sdp", "pr-body-"+claim.IssueID+".md")
	body := "## Summary\n\n- worker workflow execution for " + claim.IssueID + "\n- implemented workstream: " + workstream + "\n"
	if err := os.WriteFile(bodyPath, []byte(body), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runComponent("pr-publish", "./cmd/pr-publish", "--issue", claim.IssueID, "--title", "Worker: "+claim.Title, "--head", claim.Branch, "--base", "master", "--body-file", bodyPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(map[string]any{
		"issue":  claim.IssueID,
		"branch": claim.Branch,
		"status": "review",
	}, "", "  ")
	fmt.Println(string(out))
}
