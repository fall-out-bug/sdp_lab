package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

func patchRiskK8sHigh(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "regexp.MustCompile(`k8s`)") {
		return nil
	}
	needle := "\tregexp.MustCompile(`git`),\n"
	insert := "\tregexp.MustCompile(`git`),\n\tregexp.MustCompile(`k8s`),\n"
	if !strings.Contains(content, needle) {
		return errors.New("decision high risk pattern block not found")
	}
	content = strings.Replace(content, needle, insert, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func addRiskK8sRegressionTest(repo string) error {
	path := filepath.Join(repo, "internal", "policy", "decision_test.go")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	if strings.Contains(content, "TestDecideK8sPathIsHighRisk") {
		return nil
	}
	needle := "func TestDecideCriticalEscalates(t *testing.T) {"
	insert := "func TestDecideK8sPathIsHighRisk(t *testing.T) {\n\tres := Decide(DecisionRequest{IssueID: \"id-k8s\", Title: \"Update worker manifests\", PreferredModel: \"glm-5\", ChangedPaths: []string{\"deploy/k8s/workers/opencode-agent.yaml\"}})\n\tif res.RiskClass != \"high\" {\n\t\tt.Fatalf(\"expected high risk, got %s\", res.RiskClass)\n\t}\n\tif res.PolicyVerdict != \"allow\" {\n\t\tt.Fatalf(\"expected allow, got %s\", res.PolicyVerdict)\n\t}\n}\n\n"
	if !strings.Contains(content, needle) {
		return errors.New("decision test insertion point not found")
	}
	content = strings.Replace(content, needle, insert+needle, 1)
	return os.WriteFile(path, []byte(content), 0o644)
}

func ensurePlannerEnvelopeFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "planner", "envelope.go")
	testPath := filepath.Join(repo, "internal", "planner", "envelope_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package planner

import "fmt"

type PlanningInput struct {
	FeatureText string   ` + "`json:\"feature_text\"`" + `
	Repo        string   ` + "`json:\"repo\"`" + `
	RiskClass   string   ` + "`json:\"risk_class\"`" + `
	Lane        string   ` + "`json:\"lane\"`" + `
	Model       string   ` + "`json:\"model\"`" + `
	Boundaries  []string ` + "`json:\"boundaries\"`" + `
}

type ConstraintEnvelope struct {
	FeatureText string   ` + "`json:\"feature_text\"`" + `
	Repo        string   ` + "`json:\"repo\"`" + `
	RiskClass   string   ` + "`json:\"risk_class\"`" + `
	Lane        string   ` + "`json:\"lane\"`" + `
	Model       string   ` + "`json:\"model\"`" + `
	Boundaries  []string ` + "`json:\"boundaries\"`" + `
}

func BuildConstraintEnvelope(in PlanningInput) (ConstraintEnvelope, error) {
	if in.FeatureText == "" {
		return ConstraintEnvelope{}, fmt.Errorf("feature_text is required")
	}
	if in.Repo == "" {
		return ConstraintEnvelope{}, fmt.Errorf("repo is required")
	}
	if in.RiskClass == "" {
		in.RiskClass = "medium"
	}
	if in.Lane == "" {
		in.Lane = "commit"
	}
	if in.Model == "" {
		in.Model = "glm-5"
	}
	return ConstraintEnvelope{
		FeatureText: in.FeatureText,
		Repo:        in.Repo,
		RiskClass:   in.RiskClass,
		Lane:        in.Lane,
		Model:       in.Model,
		Boundaries:  in.Boundaries,
	}, nil
}
`
	test := `package planner

import "testing"

func TestBuildConstraintEnvelopeDefaults(t *testing.T) {
	out, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "Add parallel swarm", Repo: "fall-out-bug/sdp_private", Boundaries: []string{"internal/", "cmd/"}})
	if err != nil {
		t.Fatalf("build envelope: %v", err)
	}
	if out.RiskClass != "medium" || out.Lane != "commit" || out.Model != "glm-5" {
		t.Fatalf("unexpected defaults: %#v", out)
	}
	if len(out.Boundaries) != 2 {
		t.Fatalf("unexpected boundaries: %#v", out.Boundaries)
	}
}

func TestBuildConstraintEnvelopeRequiresInputs(t *testing.T) {
	if _, err := BuildConstraintEnvelope(PlanningInput{Repo: "fall-out-bug/sdp_private"}); err == nil {
		t.Fatal("expected feature_text validation error")
	}
	if _, err := BuildConstraintEnvelope(PlanningInput{FeatureText: "x"}); err == nil {
		t.Fatal("expected repo validation error")
	}
}
`
	if err := os.WriteFile(corePath, []byte(core), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(testPath, []byte(test), 0o644); err != nil {
		return err
	}
	return nil
}

func ensureTelegramIntakeFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "intake", "telegram.go")
	testPath := filepath.Join(repo, "internal", "intake", "telegram_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package intake

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Attachment struct {
	Kind string ` + "`json:\"kind\"`" + `
	ID   string ` + "`json:\"id\"`" + `
}

type IntakeInput struct {
	Command     string       ` + "`json:\"command\"`" + `
	FeatureText string       ` + "`json:\"feature_text\"`" + `
	MessageID   int64        ` + "`json:\"message_id\"`" + `
	ChatID      int64        ` + "`json:\"chat_id\"`" + `
	UserID      int64        ` + "`json:\"user_id\"`" + `
	Username    string       ` + "`json:\"username\"`" + `
	Language    string       ` + "`json:\"language\"`" + `
	Attachments []Attachment ` + "`json:\"attachments\"`" + `
	RawText     string       ` + "`json:\"raw_text\"`" + `
}

type telegramUpdate struct {
	Message       *telegramMessage ` + "`json:\"message\"`" + `
	EditedMessage *telegramMessage ` + "`json:\"edited_message\"`" + `
}

type telegramMessage struct {
	MessageID int64            ` + "`json:\"message_id\"`" + `
	Chat      telegramChat     ` + "`json:\"chat\"`" + `
	From      telegramUser     ` + "`json:\"from\"`" + `
	Text      string           ` + "`json:\"text\"`" + `
	Photo     []telegramPhoto  ` + "`json:\"photo\"`" + `
	Document  *telegramFileRef ` + "`json:\"document\"`" + `
	Voice     *telegramFileRef ` + "`json:\"voice\"`" + `
}

type telegramChat struct {
	ID int64 ` + "`json:\"id\"`" + `
}

type telegramUser struct {
	ID           int64  ` + "`json:\"id\"`" + `
	Username     string ` + "`json:\"username\"`" + `
	LanguageCode string ` + "`json:\"language_code\"`" + `
}

type telegramPhoto struct {
	FileID string ` + "`json:\"file_id\"`" + `
}

type telegramFileRef struct {
	FileID string ` + "`json:\"file_id\"`" + `
}

func NormalizeTelegramUpdate(raw []byte) (IntakeInput, error) {
	var upd telegramUpdate
	if err := json.Unmarshal(raw, &upd); err != nil {
		return IntakeInput{}, err
	}
	msg := upd.Message
	if msg == nil {
		msg = upd.EditedMessage
	}
	if msg == nil {
		return IntakeInput{}, fmt.Errorf("telegram update missing message")
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return IntakeInput{}, fmt.Errorf("telegram message text is empty")
	}

	cmd, payload := parseCommand(text)
	input := IntakeInput{
		Command:     cmd,
		FeatureText: payload,
		MessageID:   msg.MessageID,
		ChatID:      msg.Chat.ID,
		UserID:      msg.From.ID,
		Username:    msg.From.Username,
		Language:    msg.From.LanguageCode,
		RawText:     text,
	}
	input.Attachments = extractAttachments(*msg)
	return input, nil
}

func parseCommand(text string) (string, string) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return "message", trimmed
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "message", trimmed
	}
	cmd := strings.TrimPrefix(parts[0], "/")
	payload := strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0]))
	if cmd == "feature" && payload != "" {
		return "feature", payload
	}
	return cmd, payload
}

func extractAttachments(msg telegramMessage) []Attachment {
	result := make([]Attachment, 0)
	if len(msg.Photo) > 0 {
		result = append(result, Attachment{Kind: "photo", ID: msg.Photo[len(msg.Photo)-1].FileID})
	}
	if msg.Document != nil {
		result = append(result, Attachment{Kind: "document", ID: msg.Document.FileID})
	}
	if msg.Voice != nil {
		result = append(result, Attachment{Kind: "voice", ID: msg.Voice.FileID})
	}
	return result
}
`
	test := `package intake

import "testing"

func TestNormalizeTelegramFeatureCommand(t *testing.T) {
	raw := []byte(` + "`" + `{"message":{"message_id":42,"chat":{"id":1001},"from":{"id":5001,"username":"alice","language_code":"ru"},"text":"/feature add telegram intake flow","photo":[{"file_id":"p1"},{"file_id":"p2"}]}}` + "`" + `)
	out, err := NormalizeTelegramUpdate(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Command != "feature" {
		t.Fatalf("expected feature command, got %s", out.Command)
	}
	if out.FeatureText != "add telegram intake flow" {
		t.Fatalf("unexpected payload: %q", out.FeatureText)
	}
	if out.ChatID != 1001 || out.UserID != 5001 || out.MessageID != 42 {
		t.Fatalf("unexpected ids: %#v", out)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].Kind != "photo" || out.Attachments[0].ID != "p2" {
		t.Fatalf("unexpected attachments: %#v", out.Attachments)
	}
}

func TestNormalizeTelegramEditedMessageFallback(t *testing.T) {
	raw := []byte(` + "`" + `{"edited_message":{"message_id":7,"chat":{"id":2001},"from":{"id":6001,"username":"bob","language_code":"en"},"text":"hello world","document":{"file_id":"doc1"}}}` + "`" + `)
	out, err := NormalizeTelegramUpdate(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Command != "message" {
		t.Fatalf("expected message command, got %s", out.Command)
	}
	if out.FeatureText != "hello world" {
		t.Fatalf("unexpected payload: %q", out.FeatureText)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].Kind != "document" {
		t.Fatalf("unexpected attachments: %#v", out.Attachments)
	}
}

func TestNormalizeTelegramMissingText(t *testing.T) {
	raw := []byte(` + "`" + `{"message":{"message_id":1,"chat":{"id":1},"from":{"id":1},"text":"   "}}` + "`" + `)
	_, err := NormalizeTelegramUpdate(raw)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}
`
	if err := os.WriteFile(corePath, []byte(core), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(testPath, []byte(test), 0o644); err != nil {
		return err
	}
	return nil
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

	boundary, _ := payload["boundary"].(map[string]any)
	if boundary == nil {
		boundary = map[string]any{}
		payload["boundary"] = boundary
	}
	declared, _ := boundary["declared"].(map[string]any)
	if declared == nil {
		declared = map[string]any{}
		boundary["declared"] = declared
	}
	allowed := toStringSlice(declared["allowed_path_prefixes"])
	control := toStringSlice(declared["control_path_prefixes"])
	forbidden := toStringSlice(declared["forbidden_path_prefixes"])

	outOfBoundary := make([]string, 0)
	for _, f := range changedFiles {
		if hasPrefixAny(f, control) {
			continue
		}
		if hasPrefixAny(f, forbidden) {
			outOfBoundary = append(outOfBoundary, f)
			continue
		}
		if len(allowed) > 0 && !hasPrefixAny(f, allowed) {
			outOfBoundary = append(outOfBoundary, f)
		}
	}
	sort.Strings(outOfBoundary)

	observed, _ := boundary["observed"].(map[string]any)
	if observed == nil {
		observed = map[string]any{}
		boundary["observed"] = observed
	}
	observed["touched_paths"] = changedFiles
	observed["out_of_boundary_paths"] = outOfBoundary

	compliance, _ := boundary["compliance"].(map[string]any)
	if compliance == nil {
		compliance = map[string]any{}
		boundary["compliance"] = compliance
	}
	compliance["ok"] = len(outOfBoundary) == 0
	if len(outOfBoundary) == 0 {
		compliance["reason"] = "changed paths within declared boundary"
	} else {
		compliance["reason"] = "changed paths exceed declared boundary"
	}

	provenance, _ := payload["provenance"].(map[string]any)
	if provenance == nil {
		provenance = map[string]any{}
		payload["provenance"] = provenance
	}
	provenance["orchestrator"] = "swarm-worker"
	provenance["runtime"] = os.Getenv("SDP_RUNTIME")
	if model := os.Getenv("SDP_MODEL"); model != "" {
		provenance["model"] = model
	}
	gates := toStringSlice(provenance["gate_results"])
	gates = append(gates, "verification:go test ./...", fmt.Sprintf("boundary:ok=%t", len(outOfBoundary) == 0))
	provenance["gate_results"] = uniqueStrings(gates)

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if s, ok := v.([]string); ok {
			return s
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func hasPrefixAny(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, s := range items {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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
	if hasLabel(issue.Labels, "workstream:policy-k8s-risk-high") {
		workstream = "policy-k8s-risk-high"
	}
	if hasLabel(issue.Labels, "workstream:telegram-ingress-intake") {
		workstream = "telegram-ingress-intake"
	}
	if hasLabel(issue.Labels, "workstream:planner-boundary-decomposition") {
		workstream = "planner-boundary-decomposition"
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
	case "policy-k8s-risk-high":
		if err := patchRiskK8sHigh("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := addRiskK8sRegressionTest("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/policy/decision.go", "internal/policy/decision_test.go"}
	case "telegram-ingress-intake":
		if err := ensureTelegramIntakeFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/intake/telegram.go", "internal/intake/telegram_test.go"}
	case "planner-boundary-decomposition":
		if err := ensurePlannerEnvelopeFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/planner/envelope.go", "internal/planner/envelope_test.go"}
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
	args = append(args, ".beads/metadata.json")
	if _, err := run("git", args...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	staged, err := hasStagedChanges()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !staged {
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", "worker: no code diff produced; likely already implemented")
		_, _ = runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "blocked", "--apply")
		out, _ := json.MarshalIndent(map[string]any{
			"issue":  claim.IssueID,
			"branch": claim.Branch,
			"status": "blocked",
		}, "", "  ")
		fmt.Println(string(out))
		return
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
