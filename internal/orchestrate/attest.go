package orchestrate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
	"github.com/fall-out-bug/sdp_lab/internal/executil"
	"github.com/fall-out-bug/sdp_lab/internal/gitutil"
	"github.com/fall-out-bug/sdp_lab/internal/sdputil"
)

// GenerateOrchestratorAttestation creates an in-toto attestation from a checkpoint.
// Called by sdp-orchestrate --advance after each phase transition.
// The attestation captures what the orchestrator knows: intent, plan, execution boundary.
// CI auto-attestation later adds verification (test results, lint, coverage).
func GenerateOrchestratorAttestation(projectRoot string, cp *Checkpoint) (evidence.CodingWorkflowStatement, error) {
	branch := cp.Branch
	headSHA, err := gitHeadSHA(projectRoot)
	if err != nil {
		headSHA = "unknown"
	}

	// Extract beads IDs from the workstream mapping for this feature
	beadsIDs := lookupBeadsIDsForFeature(projectRoot, cp.FeatureID)
	issueID := firstBeadsID(beadsIDs)
	if issueID == "" {
		issueID = cp.FeatureID
	}

	// Collect workstream IDs in order
	wsIDs := make([]string, 0, len(cp.Workstreams))
	for _, ws := range cp.Workstreams {
		wsIDs = append(wsIDs, ws.ID)
	}

	// Get changed files since the repository default branch.
	changedFiles := getChangedFilesSinceBase(projectRoot, gitutil.ComparisonBase(context.Background(), projectRoot, ""))

	// Determine scope from workstream files
	scopePrefixes := collectWorkstreamScopePrefixes(projectRoot, wsIDs)
	outOfBoundary := checkOutOfBoundary(changedFiles, scopePrefixes)
	scopeOK := len(outOfBoundary) == 0

	scopeReason := fmt.Sprintf("all %d changed files within declared scope", len(changedFiles))
	if !scopeOK {
		scopeReason = fmt.Sprintf("%d files outside declared scope: %s", len(outOfBoundary), strings.Join(outOfBoundary, ", "))
	}

	subjects := []intoto.Subject{{ //nolint:staticcheck // intoto v0 types for compatibility
		Name:   fmt.Sprintf("branch:%s", branch),
		Digest: common.DigestSet{"sha256": headSHA},
	}}

	predicate := evidence.CodingWorkflowPredicate{
		Intent: evidence.Intent{
			IssueID: issueID,
			Trigger: "sdp-orchestrate",
		},
		Plan: evidence.Plan{
			Workstreams:       wsIDs,
			OrderingRationale: "sequential execution via sdp-orchestrate state machine",
		},
		Execution: evidence.Execution{
			ClaimedIssueIDs: beadsIDs,
			Branch:          branch,
			ChangedFiles:    changedFiles,
		},
		Verification: evidence.Verification{
			// Tests filled by CI auto-attestation; leave empty with a note
			Tests: []evidence.GateResult{{
				Name:   "orchestrator-phase",
				Status: fmt.Sprintf("phase=%s", cp.Phase),
			}},
		},
		Boundary: evidence.Boundary{
			Declared: evidence.DeclaredBoundary{
				AllowedPathPrefixes: scopePrefixes,
			},
			Observed: evidence.ObservedBoundary{
				TouchedPaths:       changedFiles,
				OutOfBoundaryPaths: outOfBoundary,
			},
			Compliance: evidence.BoundaryCompliance{
				OK:     scopeOK,
				Reason: scopeReason,
			},
		},
		Provenance: evidence.Provenance{
			RunID:         fmt.Sprintf("orch-%s-%s", cp.FeatureID, headSHA[:minLen(len(headSHA), 8)]),
			Orchestrator:  "sdp-orchestrate",
			Runtime:       "local",
			Phase:         cp.Phase,
			SourceIssueID: issueID,
			CapturedAt:    time.Now().UTC().Format(time.RFC3339),
		},
		Trace: evidence.Trace{
			BeadsIDs: beadsIDs,
			Branch:   branch,
			Commits:  []string{headSHA},
			PRURL:    cp.PRURL,
		},
	}

	// Add dispatch evidence from first workstream with dispatch info
	for _, ws := range cp.Workstreams {
		if ws.Dispatch != nil {
			predicate.Dispatch = &evidence.DispatchEvidence{
				Harness:   ws.Dispatch.Harness,
				Provider:  ws.Dispatch.Provider,
				Model:     ws.Dispatch.Model,
				Score:     ws.Dispatch.Score,
				Reason:    ws.Dispatch.Reason,
				ColdStart: ws.Dispatch.ColdStart,
			}
			// Set provenance model from dispatch if not already set
			if predicate.Provenance.Model == "" {
				predicate.Provenance.Model = ws.Dispatch.Model
			}
			break
		}
	}

	if cp.Review != nil && cp.Review.Status == "approved" {
		predicate.Review.SelfReview = []evidence.ReviewItem{{
			Reviewer: "sdp-orchestrate",
			Verdict:  "APPROVED",
			Notes:    fmt.Sprintf("iteration %d", cp.Review.Iteration),
		}}
	}
	if cp.QA != nil && cp.QA.Status == "passed" {
		predicate.Review.SelfReview = append(predicate.Review.SelfReview, evidence.ReviewItem{
			Reviewer: "sdp-qa",
			Verdict:  "QA_PASS",
			Notes:    fmt.Sprintf("iteration %d", cp.QA.Iteration),
		})
	}

	return evidence.NewStatement(subjects, predicate), nil
}

// WriteOrchestratorAttestation saves the attestation to .sdp/evidence/FXXX.json.
func WriteOrchestratorAttestation(projectRoot string, cp *Checkpoint) error {
	stmt, err := GenerateOrchestratorAttestation(projectRoot, cp)
	if err != nil {
		return fmt.Errorf("generate attestation: %w", err)
	}

	evidenceDir := filepath.Join(projectRoot, ".sdp", "evidence")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return fmt.Errorf("mkdir evidence: %w", err)
	}

	outPath := filepath.Join(evidenceDir, cp.FeatureID+".json")
	return evidence.WriteAttestation(outPath, stmt)
}

// lookupBeadsIDsForFeature reads the beads mapping file to find issues for a feature.
func lookupBeadsIDsForFeature(projectRoot, featureID string) []string {
	mappingPath := filepath.Join(projectRoot, ".beads-sdp-mapping.jsonl")
	f, err := os.Open(mappingPath)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	// Feature ID "F028" → workstream prefix "00-028"
	featureNum := extractFeatureNum(featureID)
	if featureNum == "" {
		return nil
	}

	prefix := fmt.Sprintf("00-%s-", featureNum)
	var ids []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry struct {
			SDPID   string `json:"sdp_id"`
			BeadsID string `json:"beads_id"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) == nil {
			if strings.HasPrefix(entry.SDPID, prefix) {
				ids = append(ids, entry.BeadsID)
			}
		}
	}
	return ids
}

var featureNumRe = regexp.MustCompile(`[Ff](\d+)`)

func extractFeatureNum(featureID string) string {
	m := featureNumRe.FindStringSubmatch(featureID)
	if m == nil {
		return ""
	}
	n := m[1]
	// Pad to 3 digits
	for len(n) < 3 {
		n = "0" + n
	}
	return n
}

// collectWorkstreamScopePrefixes reads workstream files and extracts declared scope.
func collectWorkstreamScopePrefixes(projectRoot string, wsIDs []string) []string {
	backlogDir := filepath.Join(projectRoot, "docs", "workstreams", "backlog")
	var prefixes []string
	seen := map[string]bool{}

	for _, wsID := range wsIDs {
		wsPath := filepath.Join(backlogDir, wsID+".md")
		if err := func() error {
			f, err := os.Open(wsPath)
			if err != nil {
				return fmt.Errorf("open workstream file: %w", err)
			}
			defer func() { _ = f.Close() }()

			inScope := false
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "## Scope Files") {
					inScope = true
					continue
				}
				if inScope && strings.HasPrefix(line, "##") {
					break
				}
				if inScope && strings.HasPrefix(line, "- ") {
					path := strings.TrimPrefix(line, "- ")
					path = strings.TrimSpace(strings.Trim(path, "`"))
					if path != "" && !seen[path] {
						seen[path] = true
						prefixes = append(prefixes, path)
					}
				}
			}

			return scanner.Err()
		}(); err != nil {
			slog.Warn("error reading workstream scope file", "workstream", wsID, "error", err)
			continue
		}
	}
	return prefixes
}

func checkOutOfBoundary(files, prefixes []string) []string {
	if len(prefixes) == 0 {
		return nil
	}
	var out []string
	for _, f := range files {
		if !matchesPrefix(f, prefixes) {
			out = append(out, f)
		}
	}
	return out
}

func matchesPrefix(file string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(file, p) || file == p {
			return true
		}
	}
	return false
}

// GetChangedFiles returns changed files vs the repository default branch.
func GetChangedFiles(projectRoot string) []string {
	return getChangedFilesSinceBase(projectRoot, gitutil.ComparisonBase(context.Background(), projectRoot, ""))
}

func getChangedFilesSinceBase(projectRoot, baseRef string) []string {
	ctx := context.Background()
	out, err := executil.GetDefaultRunner().Output(ctx, projectRoot, "git", "diff", "--name-only", baseRef+"...HEAD")
	if err != nil {
		// Fallback: uncommitted changes
		out2, _ := executil.GetDefaultRunner().Output(ctx, projectRoot, "git", "diff", "--name-only", "HEAD")
		return splitLines(string(out2))
	}
	return splitLines(string(out))
}

func gitHeadSHA(projectRoot string) (string, error) {
	out, err := executil.GetDefaultRunner().Output(context.Background(), projectRoot, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func splitLines(s string) []string {
	return sdputil.SplitLines(s)
}

func firstBeadsID(ids []string) string {
	return sdputil.FirstOrEmpty(ids)
}

func minLen(a, b int) int {
	return sdputil.MinLen(a, b)
}
