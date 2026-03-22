package control

import (
	"fmt"
	"os"
)

// DoctorCheck represents a single hygiene check result
type DoctorCheck struct {
	CheckID   string `json:"check_id"`
	Severity  string `json:"severity"` // error, warning
	Message   string `json:"message"`
	ProjectID string `json:"project_id,omitempty"`
	CardID    string `json:"card_id,omitempty"`
}

// DoctorReport is the complete report from doctor checks
type DoctorReport struct {
	TotalChecks int           `json:"total_checks"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	Checks      []DoctorCheck `json:"checks"`
}

// DoctorControl runs hygiene checks across all control-store cards
func (s *Store) DoctorControl() (*DoctorReport, error) {
	report := &DoctorReport{
		Checks: []DoctorCheck{},
	}

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
			}

			if cardPassed {
				report.Passed++
			}
		}
	}

	return report, nil
}
