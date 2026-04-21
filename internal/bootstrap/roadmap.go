package bootstrap

import (
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/scout"
)

// Roadmap holds the generated project roadmap derived from scout data.
type Roadmap struct {
	Vision      string        `json:"vision"`
	CurrentState ProjectState `json:"current_state"`
	TargetState  ProjectState `json:"target_state"`
	Gaps        []Gap         `json:"gaps"`
	Milestones  []Milestone   `json:"milestones"`
}

// ProjectState describes the current or target state of a project.
type ProjectState struct {
	Description string   `json:"description"`
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
}

// Gap describes a missing capability or infrastructure deficit.
type Gap struct {
	Area       string `json:"area"`       // "testing", "ci", "docs", "linting", "docker"
	Severity   string `json:"severity"`   // "critical", "important", "nice-to-have"
	Suggestion string `json:"suggestion"`
}

// Milestone describes a single actionable step on the roadmap.
type Milestone struct {
	Name     string `json:"name"`
	GapRef   string `json:"gap_ref"`  // references Gap.Area
	Priority int    `json:"priority"` // 1=first, 2=second, etc.
}

// severityOrder defines the sorting precedence for gap severities.
// Lower value = higher priority.
var severityOrder = map[string]int{
	"critical":    1,
	"important":   2,
	"nice-to-have": 3,
}

// GenerateRoadmap produces a roadmap from scout data.
// The output is deterministic: the same ProjectCard always produces the same Roadmap.
// Returns a minimal Roadmap if card is nil.
func GenerateRoadmap(card *scout.ProjectCard) *Roadmap {
	if card == nil {
		return &Roadmap{Vision: "unknown project"}
	}
	gaps := detectGaps(card)
	strengths := detectStrengths(card)
	weaknesses := detectWeaknesses(card)
	milestones := buildMilestones(gaps)

	return &Roadmap{
		Vision:       buildVision(card),
		CurrentState: buildCurrentState(card, strengths, weaknesses),
		TargetState:  buildTargetState(card),
		Gaps:        gaps,
		Milestones:  milestones,
	}
}

// detectGaps examines Maturity signals and returns detected gaps sorted by severity.
func detectGaps(card *scout.ProjectCard) []Gap {
	var gaps []Gap
	m := card.Maturity

	if !m.HasCI {
		gaps = append(gaps, Gap{
			Area:       "ci",
			Severity:   "critical",
			Suggestion: "Add continuous integration (e.g. GitHub Actions, GitLab CI)",
		})
	}
	if !m.HasTests {
		gaps = append(gaps, Gap{
			Area:       "testing",
			Severity:   "critical",
			Suggestion: "Add a test suite with meaningful coverage",
		})
	}
	if !m.HasLinter {
		gaps = append(gaps, Gap{
			Area:       "linting",
			Severity:   "important",
			Suggestion: "Add a linter to enforce code style and catch errors early",
		})
	}
	if !m.HasDocker {
		gaps = append(gaps, Gap{
			Area:       "docker",
			Severity:   "nice-to-have",
			Suggestion: "Add Docker support for reproducible builds and deployments",
		})
	}
	if !m.HasReadme {
		gaps = append(gaps, Gap{
			Area:       "docs",
			Severity:   "nice-to-have",
			Suggestion: "Add a README with project description and usage instructions",
		})
	}

	sort.Slice(gaps, func(i, j int) bool {
		return severityOrder[gaps[i].Severity] < severityOrder[gaps[j].Severity]
	})
	return gaps
}

// detectStrengths identifies project strengths from Maturity signals and Health.
func detectStrengths(card *scout.ProjectCard) []string {
	var s []string
	m := card.Maturity
	h := card.Health

	if m.HasCI {
		s = append(s, "CI pipeline configured")
	}
	if m.HasTests && h.TestCoverageHint != scout.CovNone && h.TestCoverageHint != scout.Unknown {
		s = append(s, fmt.Sprintf("Test coverage: %s", h.TestCoverageHint))
	}
	if m.HasLinter {
		s = append(s, "Linter configured")
	}
	if m.HasDocker {
		s = append(s, "Docker support available")
	}
	if m.HasReadme {
		s = append(s, "README present")
	}
	if card.Activity.Contributors > 5 {
		s = append(s, fmt.Sprintf("%d contributors", card.Activity.Contributors))
	}
	if card.Scale.TestRatio > 0.3 {
		s = append(s, "Healthy test ratio")
	}
	return s
}

// detectWeaknesses identifies project weaknesses from missing Maturity signals.
func detectWeaknesses(card *scout.ProjectCard) []string {
	var w []string
	m := card.Maturity

	if !m.HasCI {
		w = append(w, "No CI pipeline")
	}
	if !m.HasTests {
		w = append(w, "No test suite")
	}
	if !m.HasLinter {
		w = append(w, "No linter configured")
	}
	if !m.HasDocker {
		w = append(w, "No Docker support")
	}
	if !m.HasReadme {
		w = append(w, "No README")
	}
	if card.Health.BusFactorEstimate <= 1 {
		w = append(w, "Low bus factor")
	}
	if card.Activity.AgeMonths <= 2 && card.Activity.TotalCommits < 20 {
		w = append(w, "Early-stage project")
	}
	return w
}

// buildMilestones creates ordered milestones from detected gaps.
// Critical gaps become priority 1, important become priority 2, etc.
func buildMilestones(gaps []Gap) []Milestone {
	if len(gaps) == 0 {
		return nil
	}
	ms := make([]Milestone, len(gaps))
	priority := 1
	for i, g := range gaps {
		ms[i] = Milestone{
			Name:     fmt.Sprintf("Address %s gap", g.Area),
			GapRef:   g.Area,
			Priority: priority,
		}
		priority++
	}
	return ms
}

// buildVision generates a vision statement for the project.
func buildVision(card *scout.ProjectCard) string {
	name := card.Identity.Name
	lang := card.Identity.PrimaryLanguage
	if lang == "" {
		lang = "unknown"
	}
	return fmt.Sprintf("Establish %s as a well-engineered %s project with robust CI, "+
		"comprehensive tests, and clear documentation.", name, lang)
}

// buildCurrentState describes the project's current state.
func buildCurrentState(card *scout.ProjectCard, strengths, weaknesses []string) ProjectState {
	desc := fmt.Sprintf("%s is a %s project", card.Identity.Name, card.Identity.PrimaryLanguage)
	if card.Activity.AgeMonths <= 2 && card.Activity.TotalCommits < 20 {
		desc += " in an early stage of development"
	} else if card.Activity.Contributors > 10 && card.Activity.TotalCommits > 1000 {
		desc += fmt.Sprintf(" with %d contributors and %d commits", card.Activity.Contributors, card.Activity.TotalCommits)
	}
	return ProjectState{
		Description: desc,
		Strengths:   strengths,
		Weaknesses:  weaknesses,
	}
}

// buildTargetState describes the desired end state for the project.
func buildTargetState(card *scout.ProjectCard) ProjectState {
	var strengths []string
	strengths = append(strengths,
		"CI pipeline running on every push",
		"Test coverage at good or better level",
		"Linter enforcing code standards",
		"Clear README with usage instructions",
	)
	if card.Activity.Contributors > 1 {
		strengths = append(strengths, "Contributing guide for new contributors")
	}
	return ProjectState{
		Description: fmt.Sprintf("%s with full engineering infrastructure in place", card.Identity.Name),
		Strengths:   strengths,
		Weaknesses:  nil,
	}
}

// RenderRoadmapMarkdown renders the roadmap as DRAFT-ROADMAP.md content.
// The output always starts with "# DRAFT-ROADMAP".
func RenderRoadmapMarkdown(r *Roadmap) string {
	var b strings.Builder

	b.WriteString("# DRAFT-ROADMAP\n\n")

	b.WriteString("## Vision\n\n")
	b.WriteString(r.Vision)
	b.WriteString("\n\n")

	b.WriteString("## Current State\n\n")
	b.WriteString(r.CurrentState.Description)
	b.WriteString("\n\n")
	writeStringsSection(&b, "Strengths", r.CurrentState.Strengths)
	writeStringsSection(&b, "Weaknesses", r.CurrentState.Weaknesses)

	b.WriteString("## Target State\n\n")
	b.WriteString(r.TargetState.Description)
	b.WriteString("\n\n")
	writeStringsSection(&b, "Strengths", r.TargetState.Strengths)

	b.WriteString("## Gap Analysis\n\n")
	for _, g := range r.Gaps {
		b.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", g.Area, g.Severity, g.Suggestion))
	}
	if len(r.Gaps) == 0 {
		b.WriteString("No significant gaps detected.\n")
	}
	b.WriteString("\n")

	b.WriteString("## Milestones\n\n")
	for _, m := range r.Milestones {
		b.WriteString(fmt.Sprintf("%d. **%s** (addresses: %s)\n", m.Priority, m.Name, m.GapRef))
	}
	if len(r.Milestones) == 0 {
		b.WriteString("No milestones — project infrastructure is complete.\n")
	}
	b.WriteString("\n")

	return b.String()
}

// writeStringsSection writes a titled bullet list if items is non-empty.
func writeStringsSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("**%s:**\n", title))
	for _, s := range items {
		b.WriteString(fmt.Sprintf("- %s\n", s))
	}
	b.WriteString("\n")
}
