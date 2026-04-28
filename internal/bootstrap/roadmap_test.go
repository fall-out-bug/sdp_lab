package bootstrap

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/scout"
)

// healthyCard returns a ProjectCard with all maturity signals present.
func healthyCard() *scout.ProjectCard {
	desc := "A healthy test project"
	repo := "https://github.com/example/healthy"
	bs := "go_modules"
	return &scout.ProjectCard{
		Identity: scout.Identity{
			Name:            "healthy-project",
			Description:     &desc,
			RepoURL:         &repo,
			PrimaryLanguage: "Go",
			BuildSystem:     &bs,
		},
		Scale: scout.Scale{
			TotalFiles: 150,
			TotalLoc:   25000,
			TestFiles:  40,
			TestRatio:  0.35,
		},
		Activity: scout.Activity{
			AgeMonths:    24,
			TotalCommits: 500,
			Contributors: 8,
		},
		Maturity: scout.Maturity{
			HasReadme:  true,
			HasCI:      true,
			HasTests:   true,
			HasLinter:  true,
			HasDocker:  true,
			HasLicense: true,
		},
		Health: scout.HealthSignals{
			TestCoverageHint: scout.CovGood,
			ComplexityHint:   scout.ComplexityLow,
			Staleness:        scout.StalenessActive,
		},
	}
}

// newProjectCard returns a ProjectCard with low maturity and few commits.
func newProjectCard() *scout.ProjectCard {
	return &scout.ProjectCard{
		Identity: scout.Identity{
			Name:            "new-project",
			PrimaryLanguage: "TypeScript",
		},
		Scale: scout.Scale{
			TotalFiles: 10,
			TotalLoc:   500,
			TestRatio:  0,
		},
		Activity: scout.Activity{
			AgeMonths:    1,
			TotalCommits: 5,
			Contributors: 1,
		},
		Maturity: scout.Maturity{
			HasReadme: true,
		},
		Health: scout.HealthSignals{
			TestCoverageHint: scout.CovNone,
			ComplexityHint:   scout.ComplexityLow,
			Staleness:        scout.StalenessActive,
		},
	}
}

// matureProjectCard returns a ProjectCard with high maturity and many contributors.
func matureProjectCard() *scout.ProjectCard {
	desc := "A mature production project"
	repo := "https://github.com/example/mature"
	bs := "cargo"
	return &scout.ProjectCard{
		Identity: scout.Identity{
			Name:            "mature-project",
			Description:     &desc,
			RepoURL:         &repo,
			PrimaryLanguage: "Rust",
			BuildSystem:     &bs,
		},
		Scale: scout.Scale{
			TotalFiles: 500,
			TotalLoc:   120000,
			TestFiles:  200,
			TestRatio:  0.45,
		},
		Activity: scout.Activity{
			AgeMonths:    60,
			TotalCommits: 5000,
			Contributors: 50,
		},
		Maturity: scout.Maturity{
			HasReadme:      true,
			HasCI:          true,
			HasTests:       true,
			HasLinter:      true,
			HasDocker:      true,
			HasLicense:     true,
			HasReleases:    true,
			HasCodeowners:  true,
			HasContributing: true,
			HasChangelog:   true,
		},
		Health: scout.HealthSignals{
			TestCoverageHint: scout.CovGood,
			ComplexityHint:   scout.ComplexityMedium,
			Staleness:        scout.StalenessActive,
		},
	}
}

func TestGenerateRoadmap_HealthyProject(t *testing.T) {
	card := healthyCard()
	rm := GenerateRoadmap(card)

	if rm == nil {
		t.Fatal("GenerateRoadmap returned nil")
	}
	if rm.Vision == "" {
		t.Error("Vision is empty")
	}
	if rm.CurrentState.Description == "" {
		t.Error("CurrentState.Description is empty")
	}
	if len(rm.CurrentState.Strengths) == 0 {
		t.Error("Healthy project should have strengths")
	}
	// A healthy project should have no critical gaps.
	for _, g := range rm.Gaps {
		if g.Severity == "critical" {
			t.Errorf("Healthy project should not have critical gaps, got: %s - %s", g.Area, g.Suggestion)
		}
	}
}

func TestGenerateRoadmap_NewProject(t *testing.T) {
	card := newProjectCard()
	rm := GenerateRoadmap(card)

	if rm == nil {
		t.Fatal("GenerateRoadmap returned nil")
	}
	// New project should have several gaps (no CI, no tests, no linter, no docker).
	if len(rm.Gaps) == 0 {
		t.Error("New project should have gaps detected")
	}
	if len(rm.CurrentState.Weaknesses) == 0 {
		t.Error("New project should have weaknesses")
	}
	// Should mention the project is new.
	if !strings.Contains(rm.CurrentState.Description, "new") && !strings.Contains(rm.CurrentState.Description, "early") {
		t.Errorf("CurrentState.Description should mention new/early stage, got: %s", rm.CurrentState.Description)
	}
}

func TestGenerateRoadmap_MatureProject(t *testing.T) {
	card := matureProjectCard()
	rm := GenerateRoadmap(card)

	if rm == nil {
		t.Fatal("GenerateRoadmap returned nil")
	}
	if len(rm.CurrentState.Strengths) == 0 {
		t.Error("Mature project should have strengths")
	}
	// Mature project should have a target state describing maintenance.
	if rm.TargetState.Description == "" {
		t.Error("Mature project should have a target state description")
	}
	// With all maturity flags set, there should be very few gaps.
	criticalCount := 0
	for _, g := range rm.Gaps {
		if g.Severity == "critical" {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		t.Errorf("Fully mature project should have no critical gaps, got %d", criticalCount)
	}
}

func TestGenerateRoadmap_NoCI(t *testing.T) {
	card := healthyCard()
	card.Maturity.HasCI = false
	rm := GenerateRoadmap(card)

	found := false
	for _, g := range rm.Gaps {
		if g.Area == "ci" {
			found = true
			if g.Severity != "critical" {
				t.Errorf("Missing CI should be critical gap, got: %s", g.Severity)
			}
		}
	}
	if !found {
		t.Error("Missing CI should produce a gap with area 'ci'")
	}
}

func TestGenerateRoadmap_NoTests(t *testing.T) {
	card := healthyCard()
	card.Maturity.HasTests = false
	card.Health.TestCoverageHint = scout.CovNone
	rm := GenerateRoadmap(card)

	found := false
	for _, g := range rm.Gaps {
		if g.Area == "testing" {
			found = true
			if g.Severity != "critical" {
				t.Errorf("Missing tests should be critical gap, got: %s", g.Severity)
			}
		}
	}
	if !found {
		t.Error("Missing tests should produce a gap with area 'testing'")
	}
}

func TestRenderRoadmapMarkdown_ContainsSections(t *testing.T) {
	card := healthyCard()
	rm := GenerateRoadmap(card)
	md := RenderRoadmapMarkdown(rm)

	sections := []string{"Vision", "Current State", "Target State", "Gap Analysis", "Milestones"}
	for _, s := range sections {
		if !strings.Contains(md, s) {
			t.Errorf("Markdown should contain section %q", s)
		}
	}
}

func TestRenderRoadmapMarkdown_DRAFTPrefix(t *testing.T) {
	card := healthyCard()
	rm := GenerateRoadmap(card)
	md := RenderRoadmapMarkdown(rm)

	if !strings.HasPrefix(md, "# DRAFT-ROADMAP") {
		t.Errorf("Markdown should start with '# DRAFT-ROADMAP', got: %q", firstLine(md))
	}
}

func TestMilestoneOrdering(t *testing.T) {
	card := newProjectCard()
	rm := GenerateRoadmap(card)

	if len(rm.Milestones) < 2 {
		t.Skip("Not enough milestones to test ordering")
	}
	for i := 1; i < len(rm.Milestones); i++ {
		if rm.Milestones[i].Priority < rm.Milestones[i-1].Priority {
			t.Errorf("Milestones not ordered by priority: milestone[%d].Priority=%d > milestone[%d].Priority=%d",
				i-1, rm.Milestones[i-1].Priority, i, rm.Milestones[i].Priority)
		}
	}
}

func TestGapSeverityClassification(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *scout.ProjectCard
		area     string
		severity string
	}{
		{
			name:     "no CI is critical",
			setup:    noCICard,
			area:     "ci",
			severity: "critical",
		},
		{
			name:     "no tests is critical",
			setup:    noTestsCard,
			area:     "testing",
			severity: "critical",
		},
		{
			name:     "no linter is important",
			setup:    noLinterCard,
			area:     "linting",
			severity: "important",
		},
		{
			name:     "no docker is nice-to-have",
			setup:    noDockerCard,
			area:     "docker",
			severity: "nice-to-have",
		},
		{
			name:     "no docs is nice-to-have",
			setup:    noDocsCard,
			area:     "docs",
			severity: "nice-to-have",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := tt.setup()
			rm := GenerateRoadmap(card)
			found := false
			for _, g := range rm.Gaps {
				if g.Area == tt.area {
					found = true
					if g.Severity != tt.severity {
						t.Errorf("gap %q severity = %q, want %q", g.Area, g.Severity, tt.severity)
					}
				}
			}
			if !found {
				t.Errorf("expected gap with area %q not found", tt.area)
			}
		})
	}
}

// Card variants for severity classification tests.

func noCICard() *scout.ProjectCard {
	card := healthyCard()
	card.Maturity.HasCI = false
	return card
}

func noTestsCard() *scout.ProjectCard {
	card := healthyCard()
	card.Maturity.HasTests = false
	card.Health.TestCoverageHint = scout.CovNone
	return card
}

func noLinterCard() *scout.ProjectCard {
	card := healthyCard()
	card.Maturity.HasLinter = false
	return card
}

func noDockerCard() *scout.ProjectCard {
	card := healthyCard()
	card.Maturity.HasDocker = false
	return card
}

func noDocsCard() *scout.ProjectCard {
	card := healthyCard()
	card.Maturity.HasReadme = false
	return card
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
