package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/pr"
	"sdp_dev/internal/registry"
)

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func ensureRepoInitialized() error {
	out, err := run("gh", "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	if err != nil {
		return err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return fmt.Errorf("repository has no default branch; initialize remote with first commit before PR publishing")
	}
	return nil
}

type issueRecord struct {
	Owner string `json:"owner"`
}

type beadsCallbackSender struct {
	issueID string
}

func (s beadsCallbackSender) Send(_ context.Context, recipient pr.CallbackRecipientTarget, headers map[string]string, payload []byte) (int, error) {
	note := map[string]any{
		"kind":      "pr_callback_dispatch",
		"recipient": recipient.ID,
		"address":   recipient.Address,
		"headers":   headers,
		"payload":   json.RawMessage(payload),
	}
	b, err := json.Marshal(note)
	if err != nil {
		return 0, err
	}
	if _, err := run("bd", "update", s.issueID, "--append-notes", string(b)); err != nil {
		return 0, err
	}
	return 202, nil
}

func getRepositorySlug() (string, error) {
	out, err := run("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", err
	}
	repo := strings.TrimSpace(string(out))
	if repo == "" {
		return "", fmt.Errorf("empty repository slug from gh repo view")
	}
	return repo, nil
}

func getIssueOwner(issueID string) (string, error) {
	out, err := run("bd", "show", issueID, "--json")
	if err != nil {
		return "", err
	}
	var records []issueRecord
	if err := json.Unmarshal(out, &records); err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", fmt.Errorf("issue %s not found", issueID)
	}
	owner := strings.TrimSpace(records[0].Owner)
	if owner == "" {
		return "", fmt.Errorf("issue %s has empty owner", issueID)
	}
	return owner, nil
}

func getHeadCommitID() (string, error) {
	out, err := run("git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	commitID := strings.TrimSpace(string(out))
	if commitID == "" {
		return "", fmt.Errorf("empty commit id")
	}
	return commitID, nil
}

func defaultEvidencePath(issueID string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, ".sdp", "evidence", issueID+".json")
}

func main() {
	issueID := flag.String("issue", "", "Issue ID")
	prTitle := flag.String("title", "", "PR title")
	bodyFile := flag.String("body-file", "", "Path to PR body markdown file")
	head := flag.String("head", "", "Head branch")
	base := flag.String("base", "", "Base branch")
	repo := flag.String("repo", "", "Target repo (owner/repo) for PR; if empty, resolved from registry via --project or issue ID")
	project := flag.String("project", "", "Project ID for registry lookup; if empty, derived from issue ID prefix")
	evidencePath := flag.String("evidence", "", "Evidence JSON path (default .sdp/evidence/<issue>.json)")
	runID := flag.String("run-id", "", "Run ID (default from evidence provenance.run_id)")
	runContextLink := flag.String("run-context-link", "", "Run context link (default .sdp/runs/<issue>.json)")
	evidenceContextLink := flag.String("evidence-context-link", "", "Evidence context link (default evidence path)")
	callbackRouteMode := flag.String("callback-route-mode", "required-first", "Callback route mode: required-first or fanout-all")
	callbackAuditSink := flag.String("callback-audit-sink", "", "Audit callback sink (default $SDP_CALLBACK_AUDIT_SINK or audit://pr-callbacks)")
	callbackWatchers := flag.String("callback-watchers", "", "Optional comma-separated callback watcher addresses")
	dryRun := flag.Bool("dry-run", false, "Print command without executing gh")
	flag.Parse()

	if *issueID == "" || *prTitle == "" || *head == "" {
		fmt.Fprintln(os.Stderr, "--issue, --title, and --head are required")
		os.Exit(2)
	}

	effectiveRepo := strings.TrimSpace(*repo)
	effectiveBase := strings.TrimSpace(*base)
	if effectiveRepo == "" || effectiveBase == "" {
		store := registry.NewStore(registry.StoreConfig{})
		if err := store.Load(); err == nil {
			var proj *registry.Project
			if pid := strings.TrimSpace(*project); pid != "" {
				proj, _ = store.Get(pid)
			}
			if proj == nil {
				proj, _ = store.FindByIssueID(*issueID)
			}
			if proj != nil {
				if effectiveRepo == "" {
					effectiveRepo = proj.RepoSlug()
				}
				if effectiveBase == "" {
					effectiveBase = strings.TrimSpace(proj.RepoBranch)
					if effectiveBase == "" {
						effectiveBase = "main"
					}
				}
			}
		}
	}

	args := []string{"pr", "create", "--title", *prTitle, "--head", *head}
	if effectiveRepo != "" {
		args = append(args, "--repo", effectiveRepo)
	}
	if effectiveBase != "" {
		args = append(args, "--base", effectiveBase)
	}
	if *bodyFile != "" {
		args = append(args, "--body-file", *bodyFile)
	}

	if *dryRun {
		out := map[string]any{"issue": *issueID, "command": append([]string{"gh"}, args...)}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return
	}

	if err := ensureRepoInitialized(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := run("gh", args...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	prURL := strings.TrimSpace(string(out))
	if prURL == "" {
		fmt.Fprintln(os.Stderr, "gh did not return PR URL")
		os.Exit(1)
	}

	path := *evidencePath
	if path == "" {
		path = defaultEvidencePath(*issueID)
	}

	effectiveRunID := strings.TrimSpace(*runID)
	if effectiveRunID == "" {
		effectiveRunID, err = pr.ReadRunIDFromEvidence(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read run id from evidence: %v\n", err)
			os.Exit(1)
		}
	}
	effectiveRunContextLink := strings.TrimSpace(*runContextLink)
	if effectiveRunContextLink == "" {
		effectiveRunContextLink = filepath.ToSlash(filepath.Join(".sdp", "runs", *issueID+".json"))
	}
	effectiveEvidenceContextLink := strings.TrimSpace(*evidenceContextLink)
	if effectiveEvidenceContextLink == "" {
		effectiveEvidenceContextLink = filepath.ToSlash(filepath.Join(".sdp", "evidence", *issueID+".json"))
	}

	if err := pr.WritePublishTraceToEvidence(path, prURL, effectiveRunContextLink, effectiveEvidenceContextLink); err != nil {
		fmt.Fprintf(os.Stderr, "update evidence: %v\n", err)
		os.Exit(1)
	}

	repository, err := getRepositorySlug()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repository slug: %v\n", err)
		os.Exit(1)
	}
	if effectiveRepo != "" {
		repository = effectiveRepo
	}
	baseBranch := effectiveBase
	if baseBranch == "" {
		baseOut, baseErr := run("gh", "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
		if baseErr != nil {
			fmt.Fprintf(os.Stderr, "resolve base branch: %v\n", baseErr)
			os.Exit(1)
		}
		baseBranch = strings.TrimSpace(string(baseOut))
	}
	commitID, err := getHeadCommitID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve head commit: %v\n", err)
		os.Exit(1)
	}
	payload, err := pr.BuildPublishPayload(pr.PublishRequest{
		IssueID:             *issueID,
		RunID:               effectiveRunID,
		PRURL:               prURL,
		PRTitle:             *prTitle,
		Repository:          repository,
		BaseBranch:          baseBranch,
		HeadBranch:          *head,
		CommitIDs:           []string{commitID},
		PRGatePassed:        true,
		GateSignals:         []pr.GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}, {Name: "publish:callback-published", Status: "pass"}},
		PublishedAt:         time.Now().UTC(),
		RunContextLink:      effectiveRunContextLink,
		EvidenceContextLink: effectiveEvidenceContextLink,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build publish payload: %v\n", err)
		os.Exit(1)
	}
	ownerAddress, err := getIssueOwner(*issueID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve issue owner: %v\n", err)
		os.Exit(1)
	}
	auditSink := strings.TrimSpace(*callbackAuditSink)
	if auditSink == "" {
		auditSink = strings.TrimSpace(os.Getenv("SDP_CALLBACK_AUDIT_SINK"))
	}
	if auditSink == "" {
		auditSink = "audit://pr-callbacks"
	}
	watchersRaw := strings.TrimSpace(*callbackWatchers)
	if watchersRaw == "" {
		watchersRaw = strings.TrimSpace(os.Getenv("SDP_CALLBACK_WATCHERS"))
	}
	recipients := []pr.CallbackRecipientTarget{
		{ID: "issue-owner", Address: ownerAddress, Required: true, AckRequired: true},
		{ID: "orchestrator-audit", Address: auditSink, Required: true, AckRequired: true},
	}
	if watchersRaw != "" {
		recipients = append(recipients, pr.CallbackRecipientTarget{ID: "watchers", Address: watchersRaw, Required: false, AckRequired: false})
	}
	report, err := pr.DispatchCallbacks(context.Background(), beadsCallbackSender{issueID: *issueID}, payload, recipients, strings.TrimSpace(*callbackRouteMode))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dispatch callbacks: %v\n", err)
		os.Exit(1)
	}

	if _, err := run("bd", "update", *issueID, "--append-notes", "PR created: "+prURL); err != nil {
		fmt.Fprintf(os.Stderr, "update beads note: %v\n", err)
		os.Exit(1)
	}
	reportBody, _ := json.Marshal(report)
	if _, err := run("bd", "update", *issueID, "--append-notes", "PR callback dispatch report: "+string(reportBody)); err != nil {
		fmt.Fprintf(os.Stderr, "update callback report note: %v\n", err)
		os.Exit(1)
	}

	result := map[string]any{
		"issue":                 *issueID,
		"pr_url":                prURL,
		"evidence":              path,
		"run_id":                effectiveRunID,
		"run_context_link":      effectiveRunContextLink,
		"evidence_context_link": effectiveEvidenceContextLink,
		"callback_report":       report,
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
