package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/policy"
)

type dep struct {
	IssueID        string `json:"issue_id"`
	DependsOnID    string `json:"depends_on_id"`
	ID             string `json:"id"`
	Type           string `json:"type"`
	DependencyType string `json:"dependency_type"`
	IssueType      string `json:"issue_type"`
	Status         string `json:"status"`
}

func (d dep) refID() string {
	if d.DependsOnID != "" {
		return d.DependsOnID
	}
	return d.ID
}

func (d dep) kind() string {
	if d.Type != "" {
		return d.Type
	}
	if d.DependencyType != "" {
		return d.DependencyType
	}
	if d.IssueType == "epic" || d.IssueType == "feature" {
		return "parent-child"
	}
	return d.DependencyType
}

type issue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	SpecID       string   `json:"spec_id"`
	IssueType    string   `json:"issue_type"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels"`
	Dependencies []dep    `json:"dependencies"`
	CreatedAt    string   `json:"created_at"`
}

func runBD(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, string(out))
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

func listIssues() (map[string]issue, error) {
	out, err := runBD("list", "--json")
	if err != nil {
		return nil, err
	}
	var items []issue
	if err := json.Unmarshal(extractJSON(out), &items); err != nil {
		return nil, err
	}
	byID := make(map[string]issue, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	return byID, nil
}

func loadIssueDetail(issueID string) (issue, error) {
	out, err := runBD("show", issueID, "--json")
	if err != nil {
		return issue{}, err
	}
	var list []issue
	jsonOut := extractJSON(out)
	if err := json.Unmarshal(jsonOut, &list); err == nil && len(list) > 0 {
		return list[0], nil
	}
	var it issue
	if err := json.Unmarshal(jsonOut, &it); err != nil {
		return issue{}, err
	}
	return it, nil
}

func hasLabel(labels []string, name string) bool {
	for _, v := range labels {
		if v == name {
			return true
		}
	}
	return false
}

func modelFromLabels(labels []string) (string, error) {
	for _, label := range labels {
		if strings.HasPrefix(label, "model:") {
			m := strings.TrimPrefix(label, "model:")
			if !policy.AllowedModel(m) {
				return "", fmt.Errorf("model '%s' is not allowed", m)
			}
			return m, nil
		}
	}
	return policy.DefaultModel(), nil
}

func laneFromLabels(labels []string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, "lane:") {
			v := strings.TrimPrefix(label, "lane:")
			if v == "commit" || v == "explore" {
				return v
			}
		}
	}
	return "commit"
}

func allowedPrefixesFromLabels(labels []string) []string {
	for _, label := range labels {
		switch label {
		case "workstream:policy-slugify-trim", "workstream:model-chain-default-fallback", "workstream:policy-k8s-risk-high":
			return []string{"internal/policy/", "internal/evidence/", "cmd/", "docs/", "specs/", "scripts/"}
		}
	}
	return []string{"internal/", "cmd/", "docs/", "specs/", "scripts/", "deploy/"}
}

func depsSatisfied(it issue, byID map[string]issue) bool {
	for _, d := range it.Dependencies {
		if d.IssueID != "" && d.IssueID != it.ID {
			continue
		}
		if d.kind() == "parent-child" {
			continue
		}
		if d.Status != "" {
			if d.Status == "closed" || d.Status == "done" {
				continue
			}
			return false
		}
		depIssue, ok := byID[d.refID()]
		if !ok {
			return false
		}
		if depIssue.Status != "closed" && depIssue.Status != "done" {
			return false
		}
	}
	return true
}

func pickCandidate(byID map[string]issue, debug bool) (*issue, error) {
	items := make([]issue, 0)
	for _, it := range byID {
		if it.IssueType != "task" {
			if debug {
				fmt.Printf("skip %s: issue_type=%s\n", it.ID, it.IssueType)
			}
			continue
		}
		if it.Status != "open" {
			if debug {
				fmt.Printf("skip %s: status=%s\n", it.ID, it.Status)
			}
			continue
		}
		if !hasLabel(it.Labels, "autonomy") {
			if debug {
				fmt.Printf("skip %s: missing label autonomy\n", it.ID)
			}
			continue
		}
		if !hasLabel(it.Labels, "strict-evidence") {
			if debug {
				fmt.Printf("skip %s: missing label strict-evidence\n", it.ID)
			}
			continue
		}
		detail, err := loadIssueDetail(it.ID)
		if err != nil {
			if debug {
				fmt.Printf("skip %s: load issue detail failed: %v\n", it.ID, err)
			}
			continue
		}
		if depsSatisfied(detail, byID) {
			items = append(items, detail)
		} else if debug {
			fmt.Printf("skip %s: dependencies not satisfied\n", it.ID)
		}
	}
	if len(items) == 0 {
		return nil, nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return &items[0], nil
}

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func loadEvidenceTemplate(root string) (map[string]any, error) {
	path := filepath.Join(root, "specs", "strict-evidence-template.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func appendNote(issueID string, note string) error {
	_, err := runBD("update", issueID, "--append-notes", note)
	return err
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Print selected task without changes")
	debug := flag.Bool("debug", false, "Print candidate selection diagnostics")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	byID, err := listIssues()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	picked, err := pickCandidate(byID, *debug)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if picked == nil {
		fmt.Println("No eligible autonomy tasks found")
		return
	}

	model, err := modelFromLabels(picked.Labels)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	decision := policy.Decide(policy.DecisionRequest{
		IssueID:        picked.ID,
		Title:          picked.Title,
		Lane:           laneFromLabels(picked.Labels),
		PreferredModel: model,
		ChangedPaths:   []string{picked.SpecID},
	})

	branch := decision.BranchName

	output := map[string]string{"issue_id": picked.ID, "title": picked.Title, "model": model, "branch": branch}
	if *dryRun {
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(b))
		return
	}

	if _, err := runBD("update", picked.ID, "--status", "in_progress"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runPacket := map[string]any{
		"issue_id":       picked.ID,
		"title":          picked.Title,
		"model":          decision.SelectedModel,
		"fallback":       decision.FallbackChain,
		"branch":         branch,
		"started_at":     time.Now().UTC().Format(time.RFC3339),
		"status":         "in_progress",
		"flow":           "in_progress",
		"risk_class":     decision.RiskClass,
		"lane":           decision.Lane,
		"policy_verdict": decision.PolicyVerdict,
		"policy_stack":   []string{"go-first", "strict-evidence", "model-allowlist"},
	}
	runPath := filepath.Join(root, ".sdp", "runs", picked.ID+".json")
	if err := writeJSON(runPath, runPacket); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tmpl, err := loadEvidenceTemplate(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	intent, ok := tmpl["intent"].(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, errors.New("invalid evidence template: intent"))
		os.Exit(1)
	}
	intent["issue_id"] = picked.ID
	intent["trigger"] = "agent"
	intent["risk_class"] = decision.RiskClass

	execSection, ok := tmpl["execution"].(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, errors.New("invalid evidence template: execution"))
		os.Exit(1)
	}
	execSection["branch"] = branch
	execSection["claimed_issue_ids"] = []string{picked.ID}

	boundary, ok := tmpl["boundary"].(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, errors.New("invalid evidence template: boundary"))
		os.Exit(1)
	}
	declared, _ := boundary["declared"].(map[string]any)
	if declared == nil {
		declared = map[string]any{}
		boundary["declared"] = declared
	}
	declared["allowed_path_prefixes"] = allowedPrefixesFromLabels(picked.Labels)
	declared["control_path_prefixes"] = []string{".beads/", ".sdp/"}
	declared["forbidden_path_prefixes"] = []string{".git/"}
	declared["role"] = "builder"
	declared["lane"] = decision.Lane

	observed, _ := boundary["observed"].(map[string]any)
	if observed == nil {
		observed = map[string]any{}
		boundary["observed"] = observed
	}
	observed["touched_paths"] = []string{}
	observed["out_of_boundary_paths"] = []string{}

	compliance, _ := boundary["compliance"].(map[string]any)
	if compliance == nil {
		compliance = map[string]any{}
		boundary["compliance"] = compliance
	}
	compliance["ok"] = true
	compliance["reason"] = "declared boundary initialized"

	provenance, ok := tmpl["provenance"].(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, errors.New("invalid evidence template: provenance"))
		os.Exit(1)
	}
	provenance["run_id"] = picked.ID
	provenance["orchestrator"] = "autonomy-worker"
	provenance["runtime"] = os.Getenv("SDP_RUNTIME")
	provenance["model"] = decision.SelectedModel
	provenance["gate_results"] = []string{"policy:allow"}

	trace, ok := tmpl["trace"].(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, errors.New("invalid evidence template: trace"))
		os.Exit(1)
	}
	trace["branch"] = branch
	trace["beads_ids"] = []string{picked.ID}

	evidencePath := filepath.Join(root, ".sdp", "evidence", picked.ID+".json")
	if err := writeJSON(evidencePath, tmpl); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if decision.EscalationRequired {
		runPacket["flow"] = "escalated"
		_, _ = runBD("update", picked.ID, "--append-notes", "protocol flow -> escalated")
	}

	note := fmt.Sprintf("autonomy-worker(go): claimed; verdict=%s; model=%s; branch=%s; packet=%s; evidence=%s", decision.PolicyVerdict, decision.SelectedModel, branch, runPath, evidencePath)
	if err := appendNote(picked.ID, note); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(b))
}
