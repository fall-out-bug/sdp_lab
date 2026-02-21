package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/oneshot"
)

type claimResult struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
	Branch  string `json:"branch"`
}

type issueDetail struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Labels             []string `json:"labels"`
	SpecID             string   `json:"spec_id"`
	Description        string   `json:"description"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
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

func ensureOneShotManifestFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "oneshot", "manifest.go")
	testPath := filepath.Join(repo, "internal", "oneshot", "manifest_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package oneshot

import (
	"fmt"
	"sort"
	"strings"
)

type PlannerNode struct {
	ID         string   ` + "`json:\"id\"`" + `
	Owner      string   ` + "`json:\"owner\"`" + `
	DependsOn  []string ` + "`json:\"depends_on\"`" + `
	Artifacts  []string ` + "`json:\"artifacts\"`" + `
	ContractID string   ` + "`json:\"contract_id\"`" + `
}

type PlannerGraph struct {
	Nodes []PlannerNode ` + "`json:\"nodes\"`" + `
}

type ExecutionTask struct {
	ID        string   ` + "`json:\"id\"`" + `
	Role      string   ` + "`json:\"role\"`" + `
	DependsOn []string ` + "`json:\"depends_on\"`" + `
	Artifacts []string ` + "`json:\"artifacts\"`" + `
	Contract  string   ` + "`json:\"contract\"`" + `
}

type ExecutionManifest struct {
	RoleLanes map[string][]string ` + "`json:\"role_lanes\"`" + `
	Tasks     []ExecutionTask     ` + "`json:\"tasks\"`" + `
}

func BuildExecutionManifest(graph PlannerGraph) (ExecutionManifest, error) {
	if len(graph.Nodes) == 0 {
		return ExecutionManifest{}, fmt.Errorf("planner graph has no nodes")
	}

	tasks := make([]ExecutionTask, 0, len(graph.Nodes))
	lanes := make(map[string][]string)

	for _, n := range graph.Nodes {
		id := strings.TrimSpace(n.ID)
		owner := strings.TrimSpace(n.Owner)
		if id == "" {
			return ExecutionManifest{}, fmt.Errorf("node id is required")
		}
		if owner == "" {
			return ExecutionManifest{}, fmt.Errorf("node %s owner is required", id)
		}

		deps := append([]string(nil), n.DependsOn...)
		sort.Strings(deps)
		arts := append([]string(nil), n.Artifacts...)
		sort.Strings(arts)

		tasks = append(tasks, ExecutionTask{ID: id, Role: owner, DependsOn: deps, Artifacts: arts, Contract: strings.TrimSpace(n.ContractID)})
		lanes[owner] = append(lanes[owner], id)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	for role := range lanes {
		sort.Strings(lanes[role])
	}

	return ExecutionManifest{RoleLanes: lanes, Tasks: tasks}, nil
}
`
	test := `package oneshot

import (
	"reflect"
	"testing"
)

func TestBuildExecutionManifestDeterministic(t *testing.T) {
	graph := PlannerGraph{Nodes: []PlannerNode{{ID: "review", Owner: "reviewer", DependsOn: []string{"build"}, Artifacts: []string{"evidence", "pr"}, ContractID: "handoff-review"}, {ID: "build", Owner: "coder", DependsOn: []string{"plan"}, Artifacts: []string{"diff", "tests"}, ContractID: "handoff-build"}, {ID: "plan", Owner: "analyst", Artifacts: []string{"manifest"}, ContractID: "handoff-plan"}}}
	out, err := BuildExecutionManifest(graph)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if got := out.Tasks[0].ID; got != "build" {
		t.Fatalf("unexpected first task ordering: %s", got)
	}
	if !reflect.DeepEqual(out.RoleLanes["analyst"], []string{"plan"}) {
		t.Fatalf("unexpected analyst lane: %#v", out.RoleLanes["analyst"])
	}
	if !reflect.DeepEqual(out.Tasks[0].Artifacts, []string{"diff", "tests"}) {
		t.Fatalf("unexpected sorted artifacts: %#v", out.Tasks[0].Artifacts)
	}
}

func TestBuildExecutionManifestValidation(t *testing.T) {
	if _, err := BuildExecutionManifest(PlannerGraph{}); err == nil {
		t.Fatal("expected empty graph validation error")
	}
	if _, err := BuildExecutionManifest(PlannerGraph{Nodes: []PlannerNode{{ID: "x"}}}); err == nil {
		t.Fatal("expected missing owner validation error")
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
	input := IntakeInput{Command: cmd, FeatureText: payload, MessageID: msg.MessageID, ChatID: msg.Chat.ID, UserID: msg.From.ID, Username: msg.From.Username, Language: msg.From.LanguageCode, RawText: text}
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

type oneShotVerificationResult struct {
	Report        oneshot.VerificationReport
	RecoveryPlan  *oneshot.RecoveryPlan
	FailedTaskIDs []string
	RoleEvidence  []oneshot.RoleEvidence
}

func evaluateOneShotVerification(changedFiles []string, testsPassed bool) (oneShotVerificationResult, error) {
	manifest, err := oneshot.BuildExecutionManifest(oneshot.PlannerGraph{Nodes: []oneshot.PlannerNode{
		{ID: "plan", Owner: "analyst", Artifacts: []string{"manifest:plan"}, ContractID: "handoff-plan"},
		{ID: "build", Owner: "coder", DependsOn: []string{"plan"}, Artifacts: []string{"diff:worker", "tests:go-test"}, ContractID: "handoff-build"},
		{ID: "review", Owner: "reviewer", DependsOn: []string{"build"}, Artifacts: []string{"verdict:review"}, ContractID: "handoff-review"},
	}})
	if err != nil {
		return oneShotVerificationResult{}, err
	}

	hasTestChange := false
	for _, path := range changedFiles {
		if strings.HasSuffix(path, "_test.go") {
			hasTestChange = true
			break
		}
	}

	buildStatus := "ok"
	reviewStatus := "ok"
	reviewerConsumed := []string{"diff:worker"}
	if !testsPassed || !hasTestChange {
		buildStatus = "needs_changes"
		reviewStatus = "needs_changes"
		reviewerConsumed = nil
	}

	evidence := []oneshot.RoleEvidence{
		{TaskID: "plan", Role: "analyst", Status: "ok", ArtifactIDs: []string{"manifest:plan"}},
		{TaskID: "build", Role: "coder", Status: buildStatus, ArtifactIDs: []string{"diff:worker", "tests:go-test"}},
		{TaskID: "review", Role: "reviewer", Status: reviewStatus, ArtifactIDs: []string{"verdict:review"}, ConsumedArtifactIDs: reviewerConsumed},
	}

	report := oneshot.VerifyRoleEvidence(manifest, evidence)
	failedTaskIDs := make([]string, 0)
	for _, item := range evidence {
		if item.Status != "ok" {
			failedTaskIDs = append(failedTaskIDs, item.TaskID)
		}
	}
	failedTaskIDs = uniqueStrings(failedTaskIDs)

	var recoveryPlan *oneshot.RecoveryPlan
	if len(failedTaskIDs) > 0 || !report.OK {
		seed := failedTaskIDs
		if len(seed) == 0 {
			if len(report.MissingTaskEvidence) > 0 {
				seed = append(seed, report.MissingTaskEvidence...)
			}
			for taskID := range report.ReviewerDependencyGaps {
				seed = append(seed, taskID)
			}
			seed = uniqueStrings(seed)
		}
		if len(seed) > 0 {
			plan, err := oneshot.PlanFailureRecovery(manifest, seed)
			if err != nil {
				return oneShotVerificationResult{}, err
			}
			recoveryPlan = &plan
		}
	}

	return oneShotVerificationResult{
		Report:        report,
		RecoveryPlan:  recoveryPlan,
		FailedTaskIDs: failedTaskIDs,
		RoleEvidence:  evidence,
	}, nil
}

func applyOneShotVerification(payload map[string]any, runPacket map[string]any, changedFiles []string, testsPassed bool) (string, error) {
	result, err := evaluateOneShotVerification(changedFiles, testsPassed)
	if err != nil {
		return "", err
	}

	verification, _ := payload["verification"].(map[string]any)
	if verification == nil {
		verification = map[string]any{}
		payload["verification"] = verification
	}
	verification["oneshot"] = map[string]any{
		"evidence_ok":     result.Report.OK,
		"failed_task_ids": result.FailedTaskIDs,
		"report":          result.Report,
		"role_evidence":   result.RoleEvidence,
	}
	if result.RecoveryPlan != nil {
		ones, _ := verification["oneshot"].(map[string]any)
		ones["recovery_plan"] = result.RecoveryPlan
	}

	if runPacket != nil {
		runPacket["oneshot_verification"] = map[string]any{
			"evidence_ok":       result.Report.OK,
			"failed_task_count": len(result.FailedTaskIDs),
		}
		if result.RecoveryPlan != nil {
			runPacket["oneshot_recovery"] = result.RecoveryPlan
		}
	}

	note := map[string]any{
		"kind":             "oneshot_verify",
		"evidence_ok":      result.Report.OK,
		"failed_task_ids":  result.FailedTaskIDs,
		"missing_evidence": result.Report.MissingTaskEvidence,
		"dependency_gaps":  result.Report.ReviewerDependencyGaps,
		"invalid_statuses": result.Report.InvalidStatuses,
	}
	if result.RecoveryPlan != nil {
		note["requeue_task_ids"] = result.RecoveryPlan.RequeueTaskIDs
	}
	b, err := json.Marshal(note)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func updateEvidence(issueID, branch, workstream string, changedFiles []string, testsPassed bool) (string, error) {
	path := filepath.Join(".sdp", "evidence", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
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
	verification["go_test_passed"] = testsPassed

	runPath := filepath.Join(".sdp", "runs", issueID+".json")
	var runPacket map[string]any
	if runBytes, runErr := os.ReadFile(runPath); runErr == nil {
		if err := json.Unmarshal(runBytes, &runPacket); err != nil {
			return "", err
		}
	}

	note := ""
	if workstream == "oneshot-swarm-orchestrator" {
		note, err = applyOneShotVerification(payload, runPacket, changedFiles, testsPassed)
		if err != nil {
			return "", err
		}
	}

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
	provenance["phase"] = "verify"
	provenance["role"] = workstream
	provenance["captured_at"] = time.Now().UTC().Format(time.RFC3339)
	provenance["source_issue_id"] = issueID
	if _, ok := provenance["artifact_id"].(string); !ok {
		provenance["artifact_id"] = issueID + ":strict-evidence"
	}
	if _, ok := provenance["contract_version"].(string); !ok {
		provenance["contract_version"] = "artifact-provenance/v1"
	}
	if _, ok := provenance["hash_algorithm"].(string); !ok {
		provenance["hash_algorithm"] = "sha256"
	}
	if _, ok := provenance["sequence"].(float64); !ok {
		if _, intOK := provenance["sequence"].(int); !intOK {
			provenance["sequence"] = 0
		}
	}
	if _, ok := provenance["payload_digest"].(string); !ok {
		provenance["payload_digest"] = ""
	}
	if _, ok := provenance["hash"].(string); !ok {
		provenance["hash"] = ""
	}
	if _, ok := provenance["hash_prev"].(string); !ok {
		provenance["hash_prev"] = ""
	}
	gates := toStringSlice(provenance["gate_results"])
	gates = append(gates, "verification:go test ./...", fmt.Sprintf("boundary:ok=%t", len(outOfBoundary) == 0))
	provenance["gate_results"] = uniqueStrings(gates)

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", err
	}

	if runPacket != nil {
		runOut, err := json.MarshalIndent(runPacket, "", "  ")
		if err != nil {
			return "", err
		}
		runOut = append(runOut, '\n')
		if err := os.WriteFile(runPath, runOut, 0o644); err != nil {
			return "", err
		}
	}

	return note, nil
}

func main() {
	flowStartedAt := time.Now()
	claimOut, claimFallback, err := runComponentWithFallback("autonomy-worker", "./cmd/autonomy-worker")
	if err != nil {
		if strings.Contains(err.Error(), "No eligible autonomy tasks found") {
			fmt.Println("No eligible autonomy tasks found")
			emitWorkerObservability("", "plan", "blocked", "unknown", flowStartedAt, 0, claimFallback, false, "", "")
			return
		}
		emitWorkerObservability("", "plan", "failed", "unknown", flowStartedAt, 0, claimFallback, true, "", "")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.Contains(string(claimOut), "No eligible autonomy tasks found") {
		fmt.Println("No eligible autonomy tasks found")
		emitWorkerObservability("", "plan", "blocked", "unknown", flowStartedAt, 0, claimFallback, false, "", "")
		return
	}
	claim, err := parseClaim(claimOut)
	if err != nil {
		emitWorkerObservability("", "plan", "failed", "unknown", flowStartedAt, 0, claimFallback, true, "", "")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	evidenceContextLink, prURL := extractLinkage(claim.IssueID)
	emitWorkerObservability(claim.IssueID, "plan", "running", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)

	issue, err := loadIssue(claim.IssueID)
	if err != nil {
		emitWorkerObservability(claim.IssueID, "intake", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
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
	if hasLabel(issue.Labels, "workstream:oneshot-swarm-orchestrator") {
		workstream = "oneshot-swarm-orchestrator"
	}
	if hasLabel(issue.Labels, "workstream:handoff-validation") {
		workstream = "handoff-validation"
	}
	if hasLabel(issue.Labels, "workstream:generic") {
		workstream = "generic"
	}
	if hasLabel(issue.Labels, "workstream:self-improvement") {
		workstream = "self-improvement"
	}
	if hasLabel(issue.Labels, "workstream:evaluator-recommendation") {
		workstream = "evaluator-recommendation"
	}
	if workstream == "" {
		emitWorkerObservability(claim.IssueID, "plan", "escalated", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintf(os.Stderr, "unsupported workstream labels for issue %s\n", claim.IssueID)
		os.Exit(1)
	}

	discardBeadsSyncNoise()

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
	case "oneshot-swarm-orchestrator":
		if err := ensureOneShotManifestFiles("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}
	case "handoff-validation":
		if err := appendHandoffValidationTimestamp("."); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		changedFiles = []string{"docs/AGENT_HANDOFF.md"}
	case "generic":
		changedFiles = applyGenericWorkstream(".", claim.IssueID, issue)
	case "self-improvement":
		changedFiles = applySelfImprovementWorkstream(".", claim.IssueID, issue)
	case "evaluator-recommendation":
		changedFiles = applyEvaluatorRecommendationWorkstream(".", claim.IssueID, issue)
	}

	testsPassed := true
	if _, err := run("go", "test", "./..."); err != nil {
		testsPassed = false
		emitWorkerObservability(claim.IssueID, "verify", "failed", claim.Model, flowStartedAt, 0, claimFallback, workstream == "oneshot-swarm-orchestrator", evidenceContextLink, prURL)
		if workstream != "oneshot-swarm-orchestrator" {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if testsPassed {
		emitWorkerObservability(claim.IssueID, "verify", "success", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
	}

	onesNote, err := updateEvidence(claim.IssueID, claim.Branch, workstream, changedFiles, testsPassed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(onesNote) != "" {
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", onesNote)
	}
	if !testsPassed {
		emitWorkerObservability(claim.IssueID, "verify", "escalated", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", "worker: go test failed; oneshot verification emitted recovery plan")
		fmt.Fprintln(os.Stderr, "go test ./... failed")
		os.Exit(1)
	}

	if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "review", "--apply"); err != nil {
		emitWorkerObservability(claim.IssueID, "review", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runComponent("pr-gate", "./cmd/pr-gate", "--issue", claim.IssueID, "--prepublish"); err != nil {
		emitWorkerObservability(claim.IssueID, "review", "blocked", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
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
		emitWorkerObservability(claim.IssueID, "execute", "blocked", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
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
	if workstream == "handoff-validation" {
		commitBody = "Add handoff validation timestamp for adapter checklist run."
	}
	if workstream == "generic" {
		commitBody = "Generic workstream placeholder; full LLM delegation pending opencode-implement."
	}
	if workstream == "self-improvement" {
		commitBody = "Self-improvement cycle: log improvement task."
	}
	if workstream == "evaluator-recommendation" {
		commitBody = "Evaluator recommendation: log from persona consensus."
	}
	if workstream == "model-chain-default-fallback" {
		commitBody = "Make unknown model fallback deterministic and add regression coverage."
	}
	if _, err := run("git", "commit", "-m", "worker: implement "+claim.IssueID, "-m", commitBody); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
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
		emitWorkerObservability(claim.IssueID, "publish", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, prURL = extractLinkage(claim.IssueID)
	emitWorkerObservability(claim.IssueID, "publish", "success", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)

	out, _ := json.MarshalIndent(map[string]any{
		"issue":  claim.IssueID,
		"branch": claim.Branch,
		"status": "review",
	}, "", "  ")
	fmt.Println(string(out))
}
