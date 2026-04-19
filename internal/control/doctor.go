package control

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	doctorReadyStaleAfter              = 72 * time.Hour
	doctorNeedsInputStaleAfter         = 48 * time.Hour
	doctorBlockedStaleAfter            = 72 * time.Hour
	doctorMissingInitialHeartbeatAfter = 10 * time.Minute
	doctorExecutorHeartbeatStaleAfter  = 20 * time.Minute
	doctorExecutorHeartbeatLostAfter   = 60 * time.Minute
)

// DoctorCheck represents a single hygiene check result
type DoctorCheck struct {
	CheckID   string `json:"check_id"`
	Severity  string `json:"severity"` // error, warning, info
	Message   string `json:"message"`
	ProjectID string `json:"project_id,omitempty"`
	CardID    string `json:"card_id,omitempty"`
}

// DoctorReport is the complete report from doctor checks
type DoctorReport struct {
	TotalChecks int           `json:"total_checks"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	Infos       int           `json:"infos"`
	Checks      []DoctorCheck `json:"checks"`
}

// DoctorControl runs hygiene checks across all control-store cards
func (s *Store) DoctorControl() (*DoctorReport, error) {
	report := &DoctorReport{
		Checks: []DoctorCheck{},
	}
	now := time.Now().UTC()

	for _, project := range s.Registry.Projects {
		cards, err := s.LoadCards(project.ID)
		if err != nil {
			return nil, fmt.Errorf("load cards for project %s: %w", project.ID, err)
		}

		for _, card := range cards {
			report.TotalChecks++
			cardPassed := true

			if len(card.IntakeArtifact) == 0 {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "missing-intake-artifact",
					Severity:  "error",
					Message:   "card has no intake artifacts",
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			} else {
				intakePath := card.IntakeArtifact[0]
				if _, err := os.Stat(intakePath); os.IsNotExist(err) {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "intake-artifact-not-found",
						Severity:  "error",
						Message:   fmt.Sprintf("intake artifact file not found: %s", intakePath),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
			}

			if card.Status == "ready" {
				if err := validateReady(&card); err != nil {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "ready-gate-missing",
						Severity:  "error",
						Message:   fmt.Sprintf("ready card fails ready gate: %v", err),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
				if isCardStale(card, now, doctorReadyStaleAfter) {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "stale-ready-card",
						Severity:  "warning",
						Message:   fmt.Sprintf("ready card has been idle for more than %s", doctorReadyStaleAfter),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
			}

			if card.Status == "executing" && len(card.LinkedBeadsIDs) == 0 {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "executing-without-beads",
					Severity:  "error",
					Message:   "executing card has no linked beads IDs",
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			}

			if card.Status == "executing" && strings.TrimSpace(card.ExecutorSessionID) == "" {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "executing-without-session",
					Severity:  "warning",
					Message:   "executing card has no executor_session_id yet",
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			}

			if card.Status == "executing" {
				if missing, age := missingInitialHeartbeat(card, now); missing {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "executing-without-heartbeat",
						Severity:  "warning",
						Message:   fmt.Sprintf("executing card has no executor heartbeat %s after dispatch", age),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
				if stale, severity, age := staleHeartbeat(card, now); stale {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "stale-executor-heartbeat",
						Severity:  severity,
						Message:   fmt.Sprintf("executor heartbeat is stale (%s old)", age),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
				if strings.TrimSpace(card.ExecutorRuntimeState) == ExecutorRuntimeLost {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "executing-runtime-lost",
						Severity:  "error",
						Message:   "executing card runtime is marked lost",
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
			}

			if card.Status == "executing" && missingDispatchMetadata(card) {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "executing-without-dispatch-metadata",
					Severity:  "warning",
					Message:   "executing card is missing dispatched_at, dispatched_to, or dispatched_packet_path",
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			}

			if card.Status == "needs_input" {
				hasFeedback := len(card.FeedbackRequest) > 0
				hasDecision := len(card.DecisionRequired) > 0
				if !hasFeedback && !hasDecision {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "needs-input-without-questions",
						Severity:  "error",
						Message:   "needs_input card has no feedback_request or decision_required",
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
				if isCardStale(card, now, doctorNeedsInputStaleAfter) {
					report.Failed++
					report.Checks = append(report.Checks, DoctorCheck{
						CheckID:   "stale-needs-input-card",
						Severity:  "warning",
						Message:   fmt.Sprintf("needs_input card has been waiting for more than %s", doctorNeedsInputStaleAfter),
						ProjectID: project.ID,
						CardID:    card.ID,
					})
					cardPassed = false
				}
			}

			if card.Status == "blocked" && isCardStale(card, now, doctorBlockedStaleAfter) {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "stale-blocked-card",
					Severity:  "warning",
					Message:   fmt.Sprintf("blocked card has been stuck for more than %s", doctorBlockedStaleAfter),
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			}

			if card.Status == "done" && card.ExecutorResult == nil {
				report.Failed++
				report.Checks = append(report.Checks, DoctorCheck{
					CheckID:   "done-without-result-summary",
					Severity:  "warning",
					Message:   "done card has no executor_result summary",
					ProjectID: project.ID,
					CardID:    card.ID,
				})
				cardPassed = false
			}

			if cardPassed {
				report.Passed++
			}
		}
	}

	// DRAFT file hygiene (informational)
	draftChecks := checkDraftFiles(s.ProjectRoot)
	for _, dc := range draftChecks {
		report.TotalChecks++
		report.Infos++
		report.Checks = append(report.Checks, dc)
	}

	return report, nil
}

func isCardStale(card FeatureCard, now time.Time, threshold time.Duration) bool {
	updatedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(card.UpdatedAt))
	if err != nil {
		return false
	}
	return now.Sub(updatedAt.UTC()) > threshold
}

func missingDispatchMetadata(card FeatureCard) bool {
	return strings.TrimSpace(card.DispatchedAt) == "" ||
		strings.TrimSpace(card.DispatchedTo) == "" ||
		strings.TrimSpace(card.DispatchedPacketPath) == ""
}

func missingInitialHeartbeat(card FeatureCard, now time.Time) (bool, string) {
	if strings.TrimSpace(card.LastExecutorHeartbeatAt) != "" {
		return false, ""
	}
	dispatchedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(card.DispatchedAt))
	if err != nil {
		return false, ""
	}
	age := now.Sub(dispatchedAt.UTC())
	if age <= doctorMissingInitialHeartbeatAfter {
		return false, ""
	}
	return true, age.Round(time.Minute).String()
}

func staleHeartbeat(card FeatureCard, now time.Time) (bool, string, string) {
	heartbeatAt, err := time.Parse(time.RFC3339, strings.TrimSpace(card.LastExecutorHeartbeatAt))
	if err != nil {
		return false, "", ""
	}
	age := now.Sub(heartbeatAt.UTC())
	if age <= doctorExecutorHeartbeatStaleAfter {
		return false, "", ""
	}
	severity := "warning"
	if age > doctorExecutorHeartbeatLostAfter || strings.TrimSpace(card.ExecutorRuntimeState) == ExecutorRuntimeLost {
		severity = "error"
	}
	return true, severity, age.Round(time.Minute).String()
}

// checkDraftFiles scans the project root for DRAFT- prefixed files and returns
// an informational check when any are found.
func checkDraftFiles(projectRoot string) []DoctorCheck {
	pattern := filepath.Join(projectRoot, "DRAFT-*")
	matches, _ := filepath.Glob(pattern) // Glob only errors on malformed patterns; DRAFT-* is valid.
	if len(matches) == 0 {
		return nil
	}

	var paths []string
	for _, m := range matches {
		paths = append(paths, filepath.Base(m))
	}

	return []DoctorCheck{{
		CheckID:  "draft-files",
		Severity: "info",
		Message:  fmt.Sprintf("bootstrap incomplete — %d DRAFT file(s) require curation: %s", len(matches), strings.Join(paths, ", ")),
	}}
}
