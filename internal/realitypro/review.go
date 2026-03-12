package realitypro

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReviewOptions struct {
	ProjectRoot string
	Now         func() time.Time
}

type ReviewResult struct {
	ConflictReportPath  string
	IntentGapReportPath string
	ConflictCount       int
	GapCount            int
	Specialists         []string
}

type ReviewClaim struct {
	ClaimID         string   `json:"claim_id"`
	Title           string   `json:"title"`
	Statement       string   `json:"statement"`
	Status          string   `json:"status"`
	Confidence      float64  `json:"confidence"`
	SourceIDs       []string `json:"source_ids"`
	ReviewState     string   `json:"review_state"`
	AffectedRepos   []string `json:"affected_repos,omitempty"`
	AffectedPaths   []string `json:"affected_paths,omitempty"`
	OpenQuestions   []string `json:"open_questions,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	CounterEvidence []string `json:"counter_evidence,omitempty"`
}

type ReviewSource struct {
	SourceID string `json:"source_id"`
	Kind     string `json:"kind"`
	Locator  string `json:"locator"`
	Revision string `json:"revision"`
	Repo     string `json:"repo,omitempty"`
	Path     string `json:"path,omitempty"`
}

type ConflictReport struct {
	SpecVersion string         `json:"spec_version"`
	GeneratedAt string         `json:"generated_at"`
	Conflicts   []ConflictItem `json:"conflicts"`
	Claims      []ReviewClaim  `json:"claims,omitempty"`
	Sources     []ReviewSource `json:"sources,omitempty"`
}

type ConflictItem struct {
	ConflictID        string   `json:"conflict_id"`
	Summary           string   `json:"summary"`
	CompetingClaimIDs []string `json:"competing_claim_ids"`
	Severity          string   `json:"severity"`
	Status            string   `json:"status"`
	ArbitratedClaimID string   `json:"arbitrated_claim_id,omitempty"`
	ResolutionNotes   string   `json:"resolution_notes,omitempty"`
	SourceIDs         []string `json:"source_ids,omitempty"`
}

type IntentGapReport struct {
	SpecVersion string          `json:"spec_version"`
	GeneratedAt string          `json:"generated_at"`
	Gaps        []IntentGapItem `json:"gaps"`
	Claims      []ReviewClaim   `json:"claims,omitempty"`
	Sources     []ReviewSource  `json:"sources,omitempty"`
}

type IntentGapItem struct {
	GapID              string   `json:"gap_id"`
	Title              string   `json:"title"`
	ExpectedState      string   `json:"expected_state"`
	ObservedState      string   `json:"observed_state"`
	GapType            string   `json:"gap_type"`
	Severity           string   `json:"severity"`
	Status             string   `json:"status"`
	SupportingClaimIDs []string `json:"supporting_claim_ids,omitempty"`
	AffectedRepos      []string `json:"affected_repos,omitempty"`
	RecommendedActions []string `json:"recommended_actions,omitempty"`
}

type reviewFinding struct {
	ID                 string
	Title              string
	Specialist         string
	Severity           string
	GapType            string
	ExpectedState      string
	ObservedState      string
	RecommendedActions []string
	AffectedRepos      []string
	OpenQuestions      []string
	SourceIDs          []string
}

// Review executes the reality-pro specialist/skeptic/arbitration flow against repo memory.
func Review(opts ReviewOptions) (ReviewResult, error) {
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("resolve project root: %w", err)
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	generatedAt := nowFn().UTC().Format(time.RFC3339)

	memory, err := loadExistingMemory(absProjectRoot)
	if err != nil {
		return ReviewResult{}, err
	}
	if len(memory.Repos) == 0 {
		return ReviewResult{}, fmt.Errorf("repo memory is empty; run sdp-reality-pro-ingest first")
	}

	sources := reviewSources(memory, generatedAt)
	specialists := selectSpecialists(memory)
	findings := primaryFindings(memory)

	conflicts := make([]ConflictItem, 0)
	gaps := make([]IntentGapItem, 0, len(findings))
	claims := make([]ReviewClaim, 0, len(findings)*3)
	validatedClaimIDs := append([]string{}, memory.PreviousValidatedClaimIDs...)

	for _, finding := range findings {
		primary := primaryClaim(finding)
		skeptic := skepticClaim(finding, memory)
		finalClaim, conflict, gap := arbitrateFinding(finding, primary, skeptic)
		finalClaim = synthesisReview(finalClaim, finding)

		claims = append(claims, primary, skeptic, finalClaim)
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
		gaps = append(gaps, gap)
		if finalClaim.ReviewState == "arbitrated" {
			validatedClaimIDs = append(validatedClaimIDs, finalClaim.ClaimID)
		}
	}

	sort.Slice(claims, func(i, j int) bool {
		return claims[i].ClaimID < claims[j].ClaimID
	})
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].ConflictID < conflicts[j].ConflictID
	})
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].GapID < gaps[j].GapID
	})

	conflictReport := ConflictReport{
		SpecVersion: specVersion,
		GeneratedAt: generatedAt,
		Conflicts:   conflicts,
		Claims:      claims,
		Sources:     sources,
	}
	intentReport := IntentGapReport{
		SpecVersion: specVersion,
		GeneratedAt: generatedAt,
		Gaps:        gaps,
		Claims:      claims,
		Sources:     sources,
	}

	conflictPath := filepath.Join(absProjectRoot, ".sdp", "reality", "conflicts-report.json")
	if err := writeJSON(conflictPath, conflictReport); err != nil {
		return ReviewResult{}, err
	}
	intentPath := filepath.Join(absProjectRoot, ".sdp", "reality", "intent-gap-report.json")
	if err := writeJSON(intentPath, intentReport); err != nil {
		return ReviewResult{}, err
	}

	memory.GeneratedAt = generatedAt
	memory.PreviousValidatedClaimIDs = dedupeStrings(validatedClaimIDs)
	if err := writeJSON(filepath.Join(absProjectRoot, ".sdp", "reality", "repo-memory.json"), memory); err != nil {
		return ReviewResult{}, err
	}

	return ReviewResult{
		ConflictReportPath:  conflictPath,
		IntentGapReportPath: intentPath,
		ConflictCount:       len(conflicts),
		GapCount:            len(gaps),
		Specialists:         specialists,
	}, nil
}

func reviewSources(memory RepoMemory, generatedAt string) []ReviewSource {
	sources := []ReviewSource{
		{
			SourceID: "source:repo-memory",
			Kind:     "doc",
			Locator:  ".sdp/reality/repo-memory.json",
			Revision: generatedAt,
			Path:     ".sdp/reality/repo-memory.json",
		},
	}
	if len(memory.Repos) > 1 {
		sources = append(sources, ReviewSource{
			SourceID: "source:multi-repo-map",
			Kind:     "doc",
			Locator:  "docs/reality/multi-repo-map.md",
			Revision: generatedAt,
			Path:     "docs/reality/multi-repo-map.md",
		})
	}
	return sources
}

func selectSpecialists(memory RepoMemory) []string {
	specialists := []string{"architecture-analyst", "intent-analyst"}
	if len(memory.Repos) > 1 {
		specialists = append(specialists, "api-analyst")
	}
	if len(memory.Hotspots) > 0 {
		specialists = append(specialists, "test-quality-analyst")
	}
	if len(memory.UnresolvedQuestions) > 0 {
		specialists = append(specialists, "documentation-analyst")
	}
	hasProtocol := false
	for _, repo := range memory.Repos {
		if repo.Role == "protocol" {
			hasProtocol = true
			break
		}
	}
	if hasProtocol {
		specialists = append(specialists, "runtime-analyst")
	}
	return dedupeStrings(specialists)
}

func primaryFindings(memory RepoMemory) []reviewFinding {
	findings := make([]reviewFinding, 0)

	if hasRolePair(memory.Repos, "service", "protocol") {
		findings = append(findings, reviewFinding{
			ID:            "finding:contract-boundary",
			Title:         "Service-to-protocol coordination is only partially evidenced",
			Specialist:    "architecture-analyst",
			Severity:      "high",
			GapType:       "partial",
			ExpectedState: "Versioning, ownership, and contract rollout between service and protocol repos are explicit.",
			ObservedState: "The reposet shows a service repo consuming a protocol repo, but coordination still depends on unresolved operator knowledge.",
			RecommendedActions: []string{
				"Record protocol ownership and versioning rules in the reposet bootstrap plan.",
				"Add contract rollout checks before cross-repo changes land.",
			},
			AffectedRepos: repoIDsFromMemory(memory),
			OpenQuestions: filterQuestions(memory.UnresolvedQuestions, "version"),
			SourceIDs:     []string{"source:repo-memory", "source:multi-repo-map"},
		})
	}

	if len(memory.Hotspots) > 0 {
		findings = append(findings, reviewFinding{
			ID:            "finding:hotspot-risk",
			Title:         "Hotspot concentration still limits autonomous change scope",
			Specialist:    "test-quality-analyst",
			Severity:      "medium",
			GapType:       "partial",
			ExpectedState: "High-risk files are isolated behind tests or reduced into narrow modules.",
			ObservedState: fmt.Sprintf("Repo memory still tracks %d hotspot zones, so broad autonomous changes would overreach.", len(memory.Hotspots)),
			RecommendedActions: []string{
				"Fence the largest hotspots with narrow workstreams before orchestration expands scope.",
			},
			AffectedRepos: hotspotRepoIDs(memory.Hotspots),
			SourceIDs:     []string{"source:repo-memory"},
		})
	}

	if len(memory.UnresolvedQuestions) > 0 {
		findings = append(findings, reviewFinding{
			ID:            "finding:unresolved-questions",
			Title:         "Open questions still block confident synthesis",
			Specialist:    "documentation-analyst",
			Severity:      "medium",
			GapType:       "ambiguous",
			ExpectedState: "Important cross-repo questions are answered or explicitly assigned to owners.",
			ObservedState: fmt.Sprintf("Reality memory still carries %d unresolved question(s), so some claims would overstate certainty.", len(memory.UnresolvedQuestions)),
			RecommendedActions: []string{
				"Promote unresolved questions into explicit bootstrap backlog items with owners.",
			},
			AffectedRepos: repoIDsFromMemory(memory),
			OpenQuestions: memory.UnresolvedQuestions,
			SourceIDs:     []string{"source:repo-memory"},
		})
	}

	return findings
}

func primaryClaim(finding reviewFinding) ReviewClaim {
	return ReviewClaim{
		ClaimID:       finding.ID + ":primary",
		Title:         finding.Title,
		Statement:     finding.ObservedState,
		Status:        "observed",
		Confidence:    0.88,
		SourceIDs:     finding.SourceIDs,
		ReviewState:   "cross_checked",
		AffectedRepos: finding.AffectedRepos,
		OpenQuestions: finding.OpenQuestions,
		Tags:          []string{"primary", finding.Specialist},
	}
}

func skepticClaim(finding reviewFinding, memory RepoMemory) ReviewClaim {
	status := "challenged"
	claimStatus := "inferred"
	confidence := 0.62
	statement := "The primary finding needs stronger qualifiers before it can be treated as final."
	counterEvidence := []string{
		"Repo memory captures repository shape, but not full operator intent or historical rollout decisions.",
	}

	switch finding.ID {
	case "finding:hotspot-risk", "finding:thin-lineage":
		status = "cross_checked"
		claimStatus = "observed"
		confidence = 0.82
		statement = "The evidence is sufficient to keep this finding without further downgrade."
		counterEvidence = nil
	case "finding:unresolved-questions":
		statement = "Open questions clearly exist, but severity should stay moderate until tied to concrete failures."
	case "finding:contract-boundary":
		statement = "Topology proves dependency direction, but governance and rollout discipline are still inferred rather than observed."
	}

	return ReviewClaim{
		ClaimID:         finding.ID + ":skeptic",
		Title:           finding.Title + " skeptic review",
		Statement:       statement,
		Status:          claimStatus,
		Confidence:      confidence,
		SourceIDs:       finding.SourceIDs,
		ReviewState:     status,
		AffectedRepos:   finding.AffectedRepos,
		OpenQuestions:   finding.OpenQuestions,
		CounterEvidence: counterEvidence,
		Tags:            []string{"skeptic", finding.Specialist},
	}
}

func arbitrateFinding(finding reviewFinding, primary, skeptic ReviewClaim) (ReviewClaim, *ConflictItem, IntentGapItem) {
	finalClaim := ReviewClaim{
		ClaimID:       finding.ID + ":final",
		Title:         finding.Title,
		Statement:     primary.Statement,
		Status:        primary.Status,
		Confidence:    primary.Confidence,
		SourceIDs:     primary.SourceIDs,
		ReviewState:   "arbitrated",
		AffectedRepos: finding.AffectedRepos,
		OpenQuestions: finding.OpenQuestions,
		Tags:          []string{"final", finding.Specialist},
	}

	var conflict *ConflictItem
	if skeptic.ReviewState == "challenged" || skeptic.Status != primary.Status {
		finalClaim.Status = "inferred"
		finalClaim.Confidence = 0.67
		finalClaim.Statement = qualifyStatement(primary.Statement)
		finalClaim.CounterEvidence = skeptic.CounterEvidence
		conflict = &ConflictItem{
			ConflictID:        strings.ReplaceAll(finding.ID, "finding:", "conflict:"),
			Summary:           fmt.Sprintf("%s: primary observation required qualifier after skeptic review.", finding.Title),
			CompetingClaimIDs: []string{primary.ClaimID, skeptic.ClaimID},
			Severity:          finding.Severity,
			Status:            "arbitrated",
			ArbitratedClaimID: finalClaim.ClaimID,
			ResolutionNotes:   fmt.Sprintf("Synthesis reviewer kept the finding but downgraded certainty: %s", skeptic.Statement),
			SourceIDs:         finding.SourceIDs,
		}
	}

	gapStatus := "accepted"
	if conflict != nil || finding.GapType == "ambiguous" {
		gapStatus = "triaged"
	}

	gap := IntentGapItem{
		GapID:              strings.ReplaceAll(finding.ID, "finding:", "gap:"),
		Title:              finding.Title,
		ExpectedState:      finding.ExpectedState,
		ObservedState:      finalClaim.Statement,
		GapType:            finding.GapType,
		Severity:           finding.Severity,
		Status:             gapStatus,
		SupportingClaimIDs: []string{finalClaim.ClaimID},
		AffectedRepos:      finding.AffectedRepos,
		RecommendedActions: dedupeStrings(finding.RecommendedActions),
	}
	return finalClaim, conflict, gap
}

func synthesisReview(claim ReviewClaim, finding reviewFinding) ReviewClaim {
	if claim.Status == "observed" && strings.Contains(strings.ToLower(finding.ExpectedState), "explicit") {
		claim.Status = "inferred"
		claim.Confidence = 0.74
		claim.Statement = qualifyStatement(claim.Statement)
		claim.Tags = append(claim.Tags, "qualified-by-synthesis")
	}
	return claim
}

func qualifyStatement(statement string) string {
	lowered := strings.ToLower(statement)
	if strings.Contains(lowered, "likely") || strings.Contains(lowered, "suggests") {
		return statement
	}
	return "Available evidence suggests: " + strings.TrimSpace(statement)
}

func hasRolePair(repos []RepoRecord, left, right string) bool {
	hasLeft := false
	hasRight := false
	for _, repo := range repos {
		if repo.Role == left {
			hasLeft = true
		}
		if repo.Role == right {
			hasRight = true
		}
	}
	return hasLeft && hasRight
}

func repoIDsFromMemory(memory RepoMemory) []string {
	result := make([]string, 0, len(memory.Repos))
	for _, repo := range memory.Repos {
		result = append(result, repo.RepoID)
	}
	sort.Strings(result)
	return result
}

func hotspotRepoIDs(hotspots []HotspotRecord) []string {
	result := make([]string, 0, len(hotspots))
	for _, hotspot := range hotspots {
		result = append(result, hotspot.RepoID)
	}
	return dedupeStrings(result)
}

func filterQuestions(questions []string, pattern string) []string {
	result := make([]string, 0)
	for _, question := range questions {
		if strings.Contains(strings.ToLower(question), strings.ToLower(pattern)) {
			result = append(result, question)
		}
	}
	return dedupeStrings(result)
}
