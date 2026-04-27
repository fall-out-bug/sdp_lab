package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/bridge"
	"github.com/fall-out-bug/sdp_lab/internal/sdputil"
)

func EmitReviewFailureFinding(ctx context.Context, projectRoot string, cp *Checkpoint, reviewOutput string, reviewErr error) (string, error) {
	sink := bridge.NewBeadsSink("", false, []string{"autonomy"})
	_ = sink.LoadExistingFindings(ctx)
	return sink.CreateReviewFinding(ctx, buildReviewFailureFindingInput(cp, reviewOutput, reviewErr))
}

func EmitQAFailureFinding(ctx context.Context, projectRoot string, cp *Checkpoint, input bridge.QAFindingInput) (string, error) {
	sink := bridge.NewBeadsSink("", false, []string{"autonomy"})
	_ = sink.LoadExistingFindings(ctx)
	if strings.TrimSpace(input.FeatureID) == "" && cp != nil {
		input.FeatureID = cp.FeatureID
	}
	if strings.TrimSpace(input.WSID) == "" && cp != nil {
		input.WSID = checkpointFindingWSID(cp)
	}
	if strings.TrimSpace(input.PRURL) == "" && cp != nil {
		input.PRURL = cp.PRURL
	}
	return sink.CreateQAFinding(ctx, input)
}

type ReviewVerdict struct {
	Feature             string                    `json:"feature"`
	Verdict             string                    `json:"verdict"`
	Timestamp           string                    `json:"timestamp"`
	Round               int                       `json:"round"`
	Reviewers           map[string]ReviewerResult `json:"reviewers"`
	FindingIDs          []string                  `json:"finding_ids,omitempty"`
	BlockingIDs         []string                  `json:"blocking_ids,omitempty"`
	Summary             string                    `json:"summary,omitempty"`
	OverrideReason      string                    `json:"override_reason,omitempty"`
	PartialFailingRoles []string                  `json:"partial_failing_roles,omitempty"`
	EscalationIssue     string                    `json:"escalation_issue,omitempty"`
	P0Count             int                       `json:"p0_count,omitempty"`
	P1Count             int                       `json:"p1_count,omitempty"`
}

type ReviewerResult struct {
	Verdict  string   `json:"verdict"`
	Findings []string `json:"findings"`
	Notes    string   `json:"notes,omitempty"`
}

type QAVerdict struct {
	Feature     string   `json:"feature"`
	Verdict     string   `json:"verdict"`
	Timestamp   string   `json:"timestamp"`
	Iteration   int      `json:"iteration"`
	FindingIDs  []string `json:"finding_ids,omitempty"`
	BlockingIDs []string `json:"blocking_ids,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	EvidenceRef string   `json:"evidence_ref,omitempty"`
	PRURL       string   `json:"pr_url,omitempty"`
}

func WriteReviewVerdict(projectRoot string, cp *Checkpoint, verdict ReviewVerdict) (string, error) {
	path := reviewVerdictArtifactPath(projectRoot)
	if err := writeJSONArtifact(path, verdict); err != nil {
		return "", err
	}
	if cp != nil {
		if cp.Review == nil {
			cp.Review = &ReviewStatus{}
		}
		cp.Review.VerdictFile = path
	}
	return path, nil
}

func reviewVerdictArtifactPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".sdp", "review_verdict.json")
}

func adoptExistingReviewVerdict(projectRoot string, cp *Checkpoint, expectedVerdict string) (bool, error) {
	return adoptExistingReviewVerdictSince(projectRoot, cp, expectedVerdict, time.Time{})
}

func adoptExistingReviewVerdictSince(projectRoot string, cp *Checkpoint, expectedVerdict string, notBefore time.Time) (bool, error) {
	path := reviewVerdictArtifactPath(projectRoot)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat review verdict: %w", err)
	}
	if !notBefore.IsZero() && info.ModTime().Before(notBefore) {
		return false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read review verdict: %w", err)
	}
	var verdict ReviewVerdict
	if err := json.Unmarshal(data, &verdict); err != nil {
		return false, fmt.Errorf("parse review verdict: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(verdict.Verdict), strings.TrimSpace(expectedVerdict)) {
		return false, fmt.Errorf("existing review verdict %q does not match expected %q", verdict.Verdict, expectedVerdict)
	}
	if err := validateReviewVerdictEscapeFields(verdict); err != nil {
		return false, err
	}
	if cp != nil {
		if cp.Review == nil {
			cp.Review = &ReviewStatus{}
		}
		cp.Review.VerdictFile = path
	}
	return true, nil
}

func validateReviewVerdictEscapeFields(verdict ReviewVerdict) error {
	switch strings.ToUpper(strings.TrimSpace(verdict.Verdict)) {
	case "APPROVED":
		if verdict.OverrideReason != "" {
			return ValidateOverrideReason(verdict.OverrideReason)
		}
	case "PARTIALLY_APPROVED":
		if len(nonEmptyStrings(verdict.PartialFailingRoles...)) == 0 {
			return errors.New("partial review verdict requires partial_failing_roles")
		}
	case "ESCALATED":
		if strings.TrimSpace(verdict.EscalationIssue) == "" {
			return errors.New("escalated review verdict requires escalation_issue")
		}
	}
	return nil
}

func WriteQAVerdict(projectRoot string, cp *Checkpoint, verdict QAVerdict) (string, error) {
	path := filepath.Join(projectRoot, ".sdp", "qa_verdict.json")
	if err := writeJSONArtifact(path, verdict); err != nil {
		return "", err
	}
	if cp != nil {
		if cp.QA == nil {
			cp.QA = &QAStatus{}
		}
		cp.QA.VerdictFile = path
		cp.QA.EvidenceRef = verdict.EvidenceRef
	}
	return path, nil
}

func buildApprovedReviewVerdict(cp *Checkpoint, summary string) ReviewVerdict {
	return ReviewVerdict{
		Feature:   checkpointFeatureID(cp),
		Verdict:   "APPROVED",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Round:     checkpointReviewRound(cp),
		Reviewers: reviewerResults("PASS", nil, summary),
		Summary:   strings.TrimSpace(summary),
	}
}

func buildChangesRequestedReviewVerdict(cp *Checkpoint, summary, findingID string) ReviewVerdict {
	return buildReviewVerdictWithFindings(cp, "CHANGES_REQUESTED", "FAIL", summary, nonEmptyStrings(findingID))
}

func buildBlockedReviewVerdict(cp *Checkpoint, summary string, findingIDs []string) ReviewVerdict {
	return buildReviewVerdictWithFindings(cp, "CHANGES_REQUESTED", "BLOCKED", summary, findingIDs)
}

// ErrEmptyOverrideReason is returned when an override verdict is attempted
// without a justification string.
var ErrEmptyOverrideReason = errors.New("override requires non-empty justification")

// ValidateOverrideReason rejects empty justification as required by
// the governed override contract (prompts/skills/review/SKILL.md).
func ValidateOverrideReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrEmptyOverrideReason
	}
	return nil
}

func buildOverrideReviewVerdict(cp *Checkpoint, summary, overrideReason string) (ReviewVerdict, error) {
	if err := ValidateOverrideReason(overrideReason); err != nil {
		return ReviewVerdict{}, err
	}
	return ReviewVerdict{
		Feature:        checkpointFeatureID(cp),
		Verdict:        "APPROVED",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Round:          checkpointReviewRound(cp),
		Reviewers:      reviewerResults("PASS", nil, summary),
		Summary:        strings.TrimSpace(summary),
		OverrideReason: strings.TrimSpace(overrideReason),
	}, nil
}

func buildPartialReviewVerdict(cp *Checkpoint, summary string, failingRoles []string, roleFindings map[string][]string) ReviewVerdict {
	reviewers := reviewerPartialResults(failingRoles, roleFindings, summary)
	var allIDs []string
	seen := make(map[string]bool)
	for _, ids := range roleFindings {
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				allIDs = append(allIDs, id)
			}
		}
	}
	sort.Strings(allIDs)
	return ReviewVerdict{
		Feature:             checkpointFeatureID(cp),
		Verdict:             "PARTIALLY_APPROVED",
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
		Round:               checkpointReviewRound(cp),
		Reviewers:           reviewers,
		FindingIDs:          allIDs,
		BlockingIDs:         allIDs,
		Summary:             strings.TrimSpace(summary),
		PartialFailingRoles: nonEmptyStrings(failingRoles...),
	}
}

func buildEscalatedReviewVerdict(cp *Checkpoint, summary, escalationIssue string) ReviewVerdict {
	return ReviewVerdict{
		Feature:         checkpointFeatureID(cp),
		Verdict:         "ESCALATED",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Round:           checkpointReviewRound(cp),
		Reviewers:       reviewerResults("FAIL", nil, summary),
		Summary:         strings.TrimSpace(summary),
		EscalationIssue: strings.TrimSpace(escalationIssue),
	}
}

func buildReviewVerdictWithFindings(cp *Checkpoint, verdict, reviewerVerdict, summary string, findingIDs []string) ReviewVerdict {
	ids := nonEmptyStrings(findingIDs...)
	return ReviewVerdict{
		Feature:     checkpointFeatureID(cp),
		Verdict:     verdict,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Round:       checkpointReviewRound(cp),
		Reviewers:   reviewerResults(reviewerVerdict, ids, summary),
		FindingIDs:  ids,
		BlockingIDs: ids,
		Summary:     strings.TrimSpace(summary),
	}
}

func buildPassedQAVerdict(cp *Checkpoint, summary, evidenceRef string) QAVerdict {
	return QAVerdict{
		Feature:     checkpointFeatureID(cp),
		Verdict:     "qa:pass",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Iteration:   checkpointQAIteration(cp),
		Summary:     strings.TrimSpace(summary),
		EvidenceRef: strings.TrimSpace(evidenceRef),
		PRURL:       checkpointPRURL(cp),
	}
}

func buildFailedQAVerdict(cp *Checkpoint, summary, evidenceRef, findingID string) QAVerdict {
	return buildQAVerdictWithFindings(cp, "qa:fail", summary, evidenceRef, nonEmptyStrings(findingID))
}

func buildBlockedQAVerdict(cp *Checkpoint, summary, evidenceRef string, findingIDs []string) QAVerdict {
	return buildQAVerdictWithFindings(cp, "qa:fail", summary, evidenceRef, findingIDs)
}

func buildQAVerdictWithFindings(cp *Checkpoint, verdict, summary, evidenceRef string, findingIDs []string) QAVerdict {
	ids := nonEmptyStrings(findingIDs...)
	return QAVerdict{
		Feature:     checkpointFeatureID(cp),
		Verdict:     verdict,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Iteration:   checkpointQAIteration(cp),
		FindingIDs:  ids,
		BlockingIDs: ids,
		Summary:     strings.TrimSpace(summary),
		EvidenceRef: strings.TrimSpace(evidenceRef),
		PRURL:       checkpointPRURL(cp),
	}
}

func reviewerResults(verdict string, findings []string, notes string) map[string]ReviewerResult {
	roles := []string{"qa", "security", "devops", "sre", "techlead", "docs", "promptops"}
	results := make(map[string]ReviewerResult, len(roles))
	for _, role := range roles {
		results[role] = ReviewerResult{Verdict: verdict, Findings: append([]string(nil), findings...), Notes: strings.TrimSpace(notes)}
	}
	return results
}

func reviewerPartialResults(failingRoles []string, roleFindings map[string][]string, notes string) map[string]ReviewerResult {
	roles := []string{"qa", "security", "devops", "sre", "techlead", "docs", "promptops"}
	failSet := make(map[string]bool, len(failingRoles))
	for _, r := range failingRoles {
		failSet[strings.ToLower(strings.TrimSpace(r))] = true
	}
	results := make(map[string]ReviewerResult, len(roles))
	for _, role := range roles {
		if failSet[role] {
			findings := roleFindings[role]
			results[role] = ReviewerResult{Verdict: "FAIL", Findings: append([]string(nil), findings...), Notes: strings.TrimSpace(notes)}
		} else {
			results[role] = ReviewerResult{Verdict: "PASS", Findings: nil, Notes: strings.TrimSpace(notes)}
		}
	}
	return results
}

func writeJSONArtifact(path string, payload interface{}) error {
	return sdputil.AtomicWriteJSON(path, payload)
}

func buildReviewFailureFindingInput(cp *Checkpoint, reviewOutput string, reviewErr error) bridge.ReviewFindingInput {
	summary := "review did not produce APPROVED verdict"
	if reviewErr != nil {
		summary = fmt.Sprintf("review execution failed: %v", reviewErr)
	}
	description := strings.TrimSpace(reviewOutput)
	if description == "" && reviewErr != nil {
		description = reviewErr.Error()
	}
	return bridge.ReviewFindingInput{
		FeatureID:   checkpointFeatureID(cp),
		WSID:        checkpointFindingWSID(cp),
		Blocking:    true,
		Role:        "reviewer",
		Title:       "review not approved",
		Summary:     summary,
		Description: description,
		Severity:    "P1",
		Priority:    1,
		PRURL:       checkpointPRURL(cp),
		DedupKey:    checkpointFeatureID(cp) + ":review-not-approved",
	}
}

func buildQAFailureFindingInput(cp *Checkpoint, qaOutput string, qaErr error) bridge.QAFindingInput {
	summary := "QA/UAT did not produce QA_PASS verdict"
	if qaErr != nil {
		summary = fmt.Sprintf("QA/UAT execution failed: %v", qaErr)
	}
	description := strings.TrimSpace(qaOutput)
	if description == "" && qaErr != nil {
		description = qaErr.Error()
	}
	return bridge.QAFindingInput{
		FeatureID:   checkpointFeatureID(cp),
		WSID:        checkpointFindingWSID(cp),
		Blocking:    true,
		Title:       "qa not passed",
		Summary:     summary,
		Description: description,
		Severity:    "P1",
		Priority:    1,
		PRURL:       checkpointPRURL(cp),
		DedupKey:    checkpointFeatureID(cp) + ":qa-not-passed",
	}
}

func checkpointFeatureID(cp *Checkpoint) string {
	if cp == nil {
		return ""
	}
	return cp.FeatureID
}

func checkpointPRURL(cp *Checkpoint) string {
	if cp == nil {
		return ""
	}
	return cp.PRURL
}

func checkpointFindingWSID(cp *Checkpoint) string {
	if cp == nil {
		return ""
	}
	for i := len(cp.Workstreams) - 1; i >= 0; i-- {
		if strings.TrimSpace(cp.Workstreams[i].ID) != "" {
			return cp.Workstreams[i].ID
		}
	}
	return ""
}

func checkpointReviewRound(cp *Checkpoint) int {
	if cp == nil || cp.Review == nil || cp.Review.Iteration <= 0 {
		return 1
	}
	return cp.Review.Iteration
}

func checkpointQAIteration(cp *Checkpoint) int {
	if cp == nil || cp.QA == nil || cp.QA.Iteration <= 0 {
		return 1
	}
	return cp.QA.Iteration
}

// nonEmptyStrings calls sdputil.NonEmptyStrings
func nonEmptyStrings(values ...string) []string {
	return sdputil.NonEmptyStrings(values...)
}

// firstNonEmpty calls sdputil.FirstNonEmpty
func firstNonEmpty(values ...string) string {
	return sdputil.FirstNonEmpty(values...)
}
