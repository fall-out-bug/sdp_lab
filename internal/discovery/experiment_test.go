package discovery_test

import (
	"context"
	"os"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestExperimentFormat_Constants(t *testing.T) {
	formats := []discovery.ExperimentFormat{
		discovery.ExperimentSmokeTest,
		discovery.ExperimentLandingPage,
		discovery.ExperimentCustomerInterview,
		discovery.ExperimentWizardOfOz,
	}
	for _, f := range formats {
		if string(f) == "" {
			t.Errorf("empty format constant: %v", f)
		}
	}
}

func TestGenerateExperiment_SkipWithoutAPIKey(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery",
		Jobs:             []string{"validate ideas cheaply"},
		Appetite:         "medium",
	}
	val := &discovery.ValidationResult{
		FinalVerdict:    discovery.VerdictPIVOT,
		NeedsExperiment: true,
		Claims: []discovery.ClaimValidation{
			{
				Claim:   "LLM-generated validation is trusted by founders",
				RATRank: 1,
				Verdict: discovery.VerdictInsufficientData,
				Notes:   "no strong evidence either way",
			},
		},
	}
	brief, err := discovery.GenerateExperiment(context.Background(), c, frame, val)
	if err != nil {
		t.Fatalf("GenerateExperiment: %v", err)
	}
	if brief.Format == "" {
		t.Error("empty experiment format")
	}
	if brief.Objective == "" {
		t.Error("empty objective")
	}
	if brief.SuccessMetric == "" {
		t.Error("empty success metric")
	}
	if brief.TimeBoxDays <= 0 {
		t.Error("time_box_days must be positive")
	}
	if len(brief.SetupSteps) == 0 {
		t.Error("no setup steps")
	}
	validFormats := map[discovery.ExperimentFormat]bool{
		discovery.ExperimentSmokeTest:        true,
		discovery.ExperimentLandingPage:       true,
		discovery.ExperimentCustomerInterview: true,
		discovery.ExperimentWizardOfOz:        true,
	}
	if !validFormats[brief.Format] {
		t.Errorf("invalid experiment format: %q", brief.Format)
	}
	t.Logf("format: %s, time_box: %d days, cost: $%.5f", brief.Format, brief.TimeBoxDays, brief.CostUSD)
}
