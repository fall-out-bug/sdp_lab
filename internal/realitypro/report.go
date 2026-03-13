package realitypro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReportOptions struct {
	ProjectRoot string
	Now         func() time.Time
}

type ReportResult struct {
	WrittenPaths    []string
	BacklogCount    int
	PhaseCount      int
	CurrentVerdict  string
	TargetVerdict   string
	RelationshipCnt int
}

type C4SystemContext struct {
	SpecVersion   string           `json:"spec_version"`
	GeneratedAt   string           `json:"generated_at"`
	Scope         C4Scope          `json:"scope"`
	People        []C4Person       `json:"people,omitempty"`
	Systems       []C4System       `json:"systems"`
	Relationships []C4Relationship `json:"relationships"`
	Claims        []ReviewClaim    `json:"claims,omitempty"`
	Sources       []ReviewSource   `json:"sources,omitempty"`
}

type C4Scope struct {
	SystemName string   `json:"system_name"`
	Repos      []string `json:"repos"`
}

type C4Person struct {
	PersonID    string `json:"person_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type C4System struct {
	SystemID    string   `json:"system_id"`
	Name        string   `json:"name"`
	Boundary    string   `json:"boundary"`
	Description string   `json:"description,omitempty"`
	RepoIDs     []string `json:"repo_ids,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type C4Relationship struct {
	RelationshipID string  `json:"relationship_id"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	Description    string  `json:"description"`
	Technology     string  `json:"technology,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
}

type C4ContainerView struct {
	SpecVersion   string           `json:"spec_version"`
	GeneratedAt   string           `json:"generated_at"`
	SystemName    string           `json:"system_name"`
	Containers    []C4Container    `json:"containers"`
	Relationships []C4Relationship `json:"relationships"`
	Claims        []ReviewClaim    `json:"claims,omitempty"`
	Sources       []ReviewSource   `json:"sources,omitempty"`
}

type C4Container struct {
	ContainerID      string   `json:"container_id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Technology       string   `json:"technology"`
	Boundary         string   `json:"boundary,omitempty"`
	RepoID           string   `json:"repo_id,omitempty"`
	Responsibilities []string `json:"responsibilities,omitempty"`
}

type C4ComponentView struct {
	SpecVersion   string           `json:"spec_version"`
	GeneratedAt   string           `json:"generated_at"`
	ContainerID   string           `json:"container_id"`
	Components    []C4Component    `json:"components"`
	Relationships []C4Relationship `json:"relationships"`
	Claims        []ReviewClaim    `json:"claims,omitempty"`
	Sources       []ReviewSource   `json:"sources,omitempty"`
}

type C4Component struct {
	ComponentID      string   `json:"component_id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Technology       string   `json:"technology,omitempty"`
	Paths            []string `json:"paths"`
	Interfaces       []string `json:"interfaces,omitempty"`
	Responsibilities []string `json:"responsibilities,omitempty"`
}

type BootstrapBacklog struct {
	SpecVersion string                `json:"spec_version"`
	GeneratedAt string                `json:"generated_at"`
	Workstreams []BootstrapWorkstream `json:"workstreams"`
	Claims      []ReviewClaim         `json:"claims,omitempty"`
	Sources     []ReviewSource        `json:"sources,omitempty"`
}

type BootstrapWorkstream struct {
	BacklogID        string   `json:"backlog_id"`
	Title            string   `json:"title"`
	Goal             string   `json:"goal"`
	Priority         string   `json:"priority"`
	Status           string   `json:"status"`
	Scope            []string `json:"scope,omitempty"`
	Repositories     []string `json:"repositories,omitempty"`
	EvidenceClaimIDs []string `json:"evidence_claim_ids,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
	RecommendedAgent string   `json:"recommended_agent,omitempty"`
	Rationale        string   `json:"rationale,omitempty"`
	RiskLevel        string   `json:"risk_level,omitempty"`
	ExitCriteria     []string `json:"exit_criteria,omitempty"`
}

type AgentReadinessPlan struct {
	SpecVersion     string                `json:"spec_version"`
	GeneratedAt     string                `json:"generated_at"`
	CurrentVerdict  string                `json:"current_verdict"`
	TargetVerdict   string                `json:"target_verdict"`
	Phases          []AgentReadinessPhase `json:"phases"`
	KeyRisks        []string              `json:"key_risks,omitempty"`
	SequencingNotes []string              `json:"sequencing_notes,omitempty"`
	Claims          []ReviewClaim         `json:"claims,omitempty"`
	Sources         []ReviewSource        `json:"sources,omitempty"`
}

type AgentReadinessPhase struct {
	PhaseID                  string   `json:"phase_id"`
	Title                    string   `json:"title"`
	Objective                string   `json:"objective"`
	AllowedScope             []string `json:"allowed_scope,omitempty"`
	BlockedZones             []string `json:"blocked_zones,omitempty"`
	RequiredEvidence         []string `json:"required_evidence,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
	ExitCriteria             []string `json:"exit_criteria,omitempty"`
	JustificationClaimIDs    []string `json:"justification_claim_ids,omitempty"`
}

// EmitReports synthesizes the full reality-pro report surface from repo memory and reviewed findings.
func EmitReports(opts ReportOptions) (ReportResult, error) {
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return ReportResult{}, fmt.Errorf("resolve project root: %w", err)
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	generatedAt := nowFn().UTC().Format(time.RFC3339)

	memory, err := loadExistingMemory(absProjectRoot)
	if err != nil {
		return ReportResult{}, err
	}
	if len(memory.Repos) == 0 {
		return ReportResult{}, fmt.Errorf("repo memory is empty; run sdp-reality-pro-ingest first")
	}
	conflicts, err := loadConflictReport(absProjectRoot)
	if err != nil {
		return ReportResult{}, err
	}
	intent, err := loadIntentGapReport(absProjectRoot)
	if err != nil {
		return ReportResult{}, err
	}

	repoByID := make(map[string]RepoRecord, len(memory.Repos))
	moduleIndex := make(map[string][]ModuleSummary, len(memory.Repos))
	for _, repo := range memory.Repos {
		repoByID[repo.RepoID] = repo
	}
	for _, module := range memory.ModuleSummaries {
		moduleIndex[module.RepoID] = append(moduleIndex[module.RepoID], module)
	}

	claims := mergeClaims(conflicts.Claims, intent.Claims)
	sources := mergeSources(
		conflicts.Sources,
		intent.Sources,
		[]ReviewSource{
			{
				SourceID: "source:conflicts-report",
				Kind:     "doc",
				Locator:  ".sdp/reality/conflicts-report.json",
				Revision: conflicts.GeneratedAt,
				Path:     ".sdp/reality/conflicts-report.json",
			},
			{
				SourceID: "source:intent-gap-report",
				Kind:     "doc",
				Locator:  ".sdp/reality/intent-gap-report.json",
				Revision: intent.GeneratedAt,
				Path:     ".sdp/reality/intent-gap-report.json",
			},
		},
	)

	systemContext := buildSystemContext(absProjectRoot, generatedAt, memory, repoByID, claims, sources)
	containerView := buildContainerView(generatedAt, memory, repoByID, moduleIndex, claims, sources)
	componentView := buildComponentView(generatedAt, memory, repoByID, moduleIndex, claims, sources)
	backlog := buildBootstrapBacklog(generatedAt, memory, intent, claims, sources)
	readiness := buildAgentReadinessPlan(generatedAt, memory, intent, conflicts, backlog, claims, sources)

	memoryForReport := memory
	memoryForReport.GeneratedAt = generatedAt
	multiRepoMap := renderMultiRepoMap(memoryForReport, buildRepoLinks(memory.Repos), moduleIndex)

	writes := []struct {
		path    string
		payload any
		text    string
	}{
		{path: filepath.Join(absProjectRoot, ".sdp", "reality", "c4-system-context.json"), payload: systemContext},
		{path: filepath.Join(absProjectRoot, ".sdp", "reality", "c4-container.json"), payload: containerView},
		{path: filepath.Join(absProjectRoot, ".sdp", "reality", "c4-component.json"), payload: componentView},
		{path: filepath.Join(absProjectRoot, ".sdp", "reality", "bootstrap-backlog.json"), payload: backlog},
		{path: filepath.Join(absProjectRoot, ".sdp", "reality", "agent-readiness-plan.json"), payload: readiness},
		{path: filepath.Join(absProjectRoot, "docs", "reality", "c4-system-context.md"), text: renderSystemContextMarkdown(systemContext, repoByID)},
		{path: filepath.Join(absProjectRoot, "docs", "reality", "c4-containers.md"), text: renderContainerMarkdown(containerView, repoByID)},
		{path: filepath.Join(absProjectRoot, "docs", "reality", "c4-components.md"), text: renderComponentMarkdown(componentView)},
		{path: filepath.Join(absProjectRoot, "docs", "reality", "intent-gap.md"), text: renderIntentGapMarkdown(intent, conflicts, backlog, readiness, repoByID)},
		{path: filepath.Join(absProjectRoot, "docs", "reality", "multi-repo-map.md"), text: multiRepoMap},
	}

	written := make([]string, 0, len(writes))
	for _, item := range writes {
		if item.payload != nil {
			if err := writeJSON(item.path, item.payload); err != nil {
				return ReportResult{}, err
			}
		} else {
			if err := writeText(item.path, item.text); err != nil {
				return ReportResult{}, err
			}
		}
		written = append(written, item.path)
	}
	sort.Strings(written)

	return ReportResult{
		WrittenPaths:    written,
		BacklogCount:    len(backlog.Workstreams),
		PhaseCount:      len(readiness.Phases),
		CurrentVerdict:  readiness.CurrentVerdict,
		TargetVerdict:   readiness.TargetVerdict,
		RelationshipCnt: len(systemContext.Relationships) + len(containerView.Relationships) + len(componentView.Relationships),
	}, nil
}

func loadConflictReport(projectRoot string) (ConflictReport, error) {
	path := filepath.Join(projectRoot, ".sdp", "reality", "conflicts-report.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConflictReport{}, fmt.Errorf("conflicts report is missing; run sdp-reality-pro-review first")
		}
		return ConflictReport{}, err
	}
	var report ConflictReport
	if err := json.Unmarshal(data, &report); err != nil {
		return ConflictReport{}, fmt.Errorf("parse conflicts report: %w", err)
	}
	return report, nil
}

func loadIntentGapReport(projectRoot string) (IntentGapReport, error) {
	path := filepath.Join(projectRoot, ".sdp", "reality", "intent-gap-report.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return IntentGapReport{}, fmt.Errorf("intent-gap report is missing; run sdp-reality-pro-review first")
		}
		return IntentGapReport{}, err
	}
	var report IntentGapReport
	if err := json.Unmarshal(data, &report); err != nil {
		return IntentGapReport{}, fmt.Errorf("parse intent-gap report: %w", err)
	}
	return report, nil
}

func buildSystemContext(projectRoot, generatedAt string, memory RepoMemory, repoByID map[string]RepoRecord, claims []ReviewClaim, sources []ReviewSource) C4SystemContext {
	repoIDs := repoIDsFromMemory(memory)
	systemName := reportSystemName(projectRoot, memory)
	systems := make([]C4System, 0, len(memory.Repos))
	relationships := make([]C4Relationship, 0)

	for _, repo := range memory.Repos {
		systems = append(systems, C4System{
			SystemID:    systemID(repo.RepoID),
			Name:        repo.Name,
			Boundary:    "internal",
			Description: repoDescription(repo),
			RepoIDs:     []string{repo.RepoID},
			Tags:        dedupeStrings([]string{repo.Role}),
		})
	}
	sort.Slice(systems, func(i, j int) bool {
		return systems[i].SystemID < systems[j].SystemID
	})

	for _, link := range buildRepoLinks(memory.Repos) {
		relationships = append(relationships, C4Relationship{
			RelationshipID: relationshipID("system-rel", link.From, link.Relation, link.To),
			From:           systemID(link.From),
			To:             systemID(link.To),
			Description:    link.Relation,
			Technology:     repoLinkTechnology(link),
			Confidence:     repoLinkConfidence(link),
		})
	}

	people, ownershipRelationships := ownershipPeopleAndRelationships(memory)
	relationships = append(relationships, ownershipRelationships...)
	if len(relationships) == 0 || len(memory.UnresolvedQuestions) > 0 {
		people = append(people, C4Person{
			PersonID:    "person:operator",
			Name:        "Operator",
			Description: "Human reviewer who resolves intent gaps and arbitrates uncertain claims.",
		})
		if len(memory.Repos) > 0 {
			target := systemID(memory.Repos[0].RepoID)
			relationships = append(relationships, C4Relationship{
				RelationshipID: relationshipID("system-rel", "person:operator", "reviews", target),
				From:           "person:operator",
				To:             target,
				Description:    "reviews and arbitrates open questions",
				Technology:     "human review",
				Confidence:     0.66,
			})
		}
	}

	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].RelationshipID < relationships[j].RelationshipID
	})

	return C4SystemContext{
		SpecVersion: specVersion,
		GeneratedAt: generatedAt,
		Scope: C4Scope{
			SystemName: systemName,
			Repos:      repoIDs,
		},
		People:        people,
		Systems:       systems,
		Relationships: relationships,
		Claims:        claims,
		Sources:       sources,
	}
}

func buildContainerView(generatedAt string, memory RepoMemory, repoByID map[string]RepoRecord, moduleIndex map[string][]ModuleSummary, claims []ReviewClaim, sources []ReviewSource) C4ContainerView {
	containers := make([]C4Container, 0, len(memory.Repos))
	for _, repo := range memory.Repos {
		containers = append(containers, C4Container{
			ContainerID:      containerID(repo.RepoID),
			Name:             repo.Name,
			Description:      repoDescription(repo),
			Technology:       repoTechnology(repo),
			Boundary:         "internal",
			RepoID:           repo.RepoID,
			Responsibilities: repoResponsibilities(repo, moduleIndex[repo.RepoID]),
		})
	}

	relationships := make([]C4Relationship, 0)
	for _, link := range buildRepoLinks(memory.Repos) {
		relationships = append(relationships, C4Relationship{
			RelationshipID: relationshipID("container-rel", link.From, link.Relation, link.To),
			From:           containerID(link.From),
			To:             containerID(link.To),
			Description:    link.Relation,
			Technology:     repoLinkTechnology(link),
			Confidence:     repoLinkConfidence(link),
		})
	}

	if len(relationships) == 0 {
		containers = append(containers, C4Container{
			ContainerID:      "container:human-review",
			Name:             "Human review loop",
			Description:      "Fallback boundary for arbitration when repo scope is not yet fully trusted.",
			Technology:       "Human review",
			Boundary:         "external",
			Responsibilities: []string{"Resolves open questions", "Qualifies uncertain findings"},
		})
		if len(memory.Repos) > 0 {
			relationships = append(relationships, C4Relationship{
				RelationshipID: "container-rel:human-review",
				From:           containerID(memory.Repos[0].RepoID),
				To:             "container:human-review",
				Description:    "requires human arbitration for unresolved intent",
				Technology:     "review workflow",
				Confidence:     0.66,
			})
		}
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].ContainerID < containers[j].ContainerID
	})
	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].RelationshipID < relationships[j].RelationshipID
	})

	return C4ContainerView{
		SpecVersion:   specVersion,
		GeneratedAt:   generatedAt,
		SystemName:    reportSystemName("", memory),
		Containers:    containers,
		Relationships: relationships,
		Claims:        claims,
		Sources:       sources,
	}
}

func buildComponentView(generatedAt string, memory RepoMemory, repoByID map[string]RepoRecord, moduleIndex map[string][]ModuleSummary, claims []ReviewClaim, sources []ReviewSource) C4ComponentView {
	selected := selectComponentModules(memory, moduleIndex)
	components := make([]C4Component, 0, len(selected))
	componentByKey := map[string]C4Component{}
	for _, module := range selected {
		repo := repoByID[module.RepoID]
		name := componentDisplayName(repo, module)
		component := C4Component{
			ComponentID:      componentID(module.ModuleID),
			Name:             name,
			Description:      module.Summary,
			Technology:       moduleTechnology(repo, module),
			Paths:            modulePathsForComponent(module),
			Interfaces:       dedupeStrings(module.Interfaces),
			Responsibilities: moduleResponsibilities(repo, module),
		}
		components = append(components, component)
		componentByKey[module.ModuleID] = component
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].ComponentID < components[j].ComponentID
	})

	relationships := buildComponentRelationships(memory, repoByID, selected, componentByKey)
	if len(relationships) == 0 && len(components) > 0 {
		target := components[0].ComponentID
		relationships = append(relationships, C4Relationship{
			RelationshipID: relationshipID("component-rel", target, "self", target),
			From:           target,
			To:             target,
			Description:    "anchors the current component boundary",
			Technology:     components[0].Technology,
			Confidence:     0.5,
		})
	}

	return C4ComponentView{
		SpecVersion:   specVersion,
		GeneratedAt:   generatedAt,
		ContainerID:   componentContainerID(memory),
		Components:    components,
		Relationships: relationships,
		Claims:        claims,
		Sources:       sources,
	}
}

func buildBootstrapBacklog(generatedAt string, memory RepoMemory, intent IntentGapReport, claims []ReviewClaim, sources []ReviewSource) BootstrapBacklog {
	finalClaimIndex := finalClaimIndex(intent.Claims)
	workstreams := make([]BootstrapWorkstream, 0, max(1, len(intent.Gaps)))
	contractBacklogID := ""

	gaps := append([]IntentGapItem{}, intent.Gaps...)
	sort.Slice(gaps, func(i, j int) bool {
		if severityWeight(gaps[i].Severity) == severityWeight(gaps[j].Severity) {
			return gaps[i].GapID < gaps[j].GapID
		}
		return severityWeight(gaps[i].Severity) > severityWeight(gaps[j].Severity)
	})

	for _, gap := range gaps {
		backlogID := strings.Replace(gap.GapID, "gap:", "backlog:", 1)
		scope := backlogScope(gap, memory)
		status := "proposed"
		deps := []string{}
		if contractBacklogID == "" {
			status = "sequenced"
			if strings.Contains(gap.GapID, "contract-boundary") {
				contractBacklogID = backlogID
			}
		}
		if strings.Contains(gap.GapID, "contract-boundary") {
			contractBacklogID = backlogID
		}
		if contractBacklogID != "" && backlogID != contractBacklogID {
			deps = append(deps, contractBacklogID)
			if strings.Contains(gap.GapID, "hotspot") {
				status = "blocked"
			}
		}
		workstreams = append(workstreams, BootstrapWorkstream{
			BacklogID:        backlogID,
			Title:            gap.Title,
			Goal:             backlogGoal(gap),
			Priority:         severityToPriority(gap.Severity),
			Status:           status,
			Scope:            scope,
			Repositories:     dedupeStrings(gap.AffectedRepos),
			EvidenceClaimIDs: evidenceClaimIDs(gap.SupportingClaimIDs, finalClaimIndex),
			Dependencies:     deps,
			RecommendedAgent: recommendedAgent(gap),
			Rationale:        gap.ObservedState,
			RiskLevel:        gap.Severity,
			ExitCriteria:     backlogExitCriteria(gap),
		})
	}

	if len(workstreams) == 0 {
		workstreams = append(workstreams, BootstrapWorkstream{
			BacklogID:        "backlog:memory-refresh",
			Title:            "Refresh reposet memory before autonomous work expands",
			Goal:             "Keep repo memory, reviewed findings, and repo landscape current before new workstreams are staged.",
			Priority:         "P2",
			Status:           "sequenced",
			Scope:            []string{".sdp/reality/repo-memory.json", "docs/reality/multi-repo-map.md"},
			Repositories:     repoIDsFromMemory(memory),
			RecommendedAgent: "architecture-analyst",
			Rationale:        "No explicit intent gaps were emitted, so the next safe move is to refresh evidence and keep memory current.",
			RiskLevel:        "medium",
			ExitCriteria: []string{
				"Repo memory refresh produces stable repo roles and module inventory.",
				"Reviewed artifacts remain schema-valid after refresh.",
			},
		})
	}

	sort.Slice(workstreams, func(i, j int) bool {
		if workstreams[i].Priority == workstreams[j].Priority {
			return workstreams[i].BacklogID < workstreams[j].BacklogID
		}
		return workstreamPriorityWeight(workstreams[i].Priority) < workstreamPriorityWeight(workstreams[j].Priority)
	})

	return BootstrapBacklog{
		SpecVersion: specVersion,
		GeneratedAt: generatedAt,
		Workstreams: workstreams,
		Claims:      claims,
		Sources:     sources,
	}
}

func buildAgentReadinessPlan(generatedAt string, memory RepoMemory, intent IntentGapReport, conflicts ConflictReport, backlog BootstrapBacklog, claims []ReviewClaim, sources []ReviewSource) AgentReadinessPlan {
	currentVerdict := readinessVerdict(memory, intent, conflicts)
	targetVerdict := "ready"
	finalClaimIndex := finalClaimIndex(intent.Claims)
	topHotspots := hotspotPaths(memory.Hotspots, 5)
	phases := []AgentReadinessPhase{
		{
			PhaseID:   "phase:boundary-clarity",
			Title:     "Stabilize boundaries and intent",
			Objective: "Make repo ownership, contract rollout, and key open questions explicit before broad agent execution.",
			AllowedScope: []string{
				"docs/reality/",
				"docs/specs/",
				"docs/workstreams/",
				"schema/reality/",
			},
			BlockedZones:             topHotspots,
			RequiredEvidence:         []string{".sdp/reality/repo-memory.json", ".sdp/reality/intent-gap-report.json", "docs/reality/multi-repo-map.md"},
			VerificationRequirements: []string{"go run ./cmd/sdp-protocol-check", "refresh repo-memory and reviewed findings"},
			ExitCriteria: []string{
				"Cross-repo ownership and rollout expectations are documented.",
				"Ownership zones and escalation paths are explicit for active repos.",
				"Open intent questions are either answered or assigned to owners.",
			},
			JustificationClaimIDs: phaseJustification(finalClaimIndex, "finding:contract-boundary:final", "finding:unresolved-questions:final"),
		},
		{
			PhaseID:   "phase:hotspot-containment",
			Title:     "Reduce hotspot blast radius",
			Objective: "Fence high-risk zones with narrow workstreams and verification before agents touch them directly.",
			AllowedScope: []string{
				"targeted tests around hotspot files",
				"small modules adjacent to top hotspots",
				"quality gate scripts and evidence outputs",
			},
			BlockedZones:             topHotspots,
			RequiredEvidence:         []string{".sdp/reality/bootstrap-backlog.json", "targeted test evidence for hotspot files"},
			VerificationRequirements: []string{"run targeted go test slices", "./scripts/run_go_quality_gates.sh"},
			ExitCriteria: []string{
				"Top hotspot paths are either isolated behind tests or explicitly deferred.",
				"Bootstrap backlog clearly marks blocked zones and safe-first slices.",
			},
			JustificationClaimIDs: phaseJustification(finalClaimIndex, "finding:hotspot-risk:final"),
		},
		{
			PhaseID:                  "phase:bootstrap-delivery",
			Title:                    "Execute bootstrap workstreams",
			Objective:                "Land the first safe SDP workstreams with reviewed evidence and rerun reality-pro after each slice.",
			AllowedScope:             readinessAllowedScope(backlog.Workstreams),
			BlockedZones:             topHotspots,
			RequiredEvidence:         []string{".sdp/reality/bootstrap-backlog.json", ".sdp/reality/agent-readiness-plan.json"},
			VerificationRequirements: []string{"rerun sdp-reality-pro-ingest", "rerun sdp-reality-pro-review", "rerun sdp-reality-pro-report"},
			ExitCriteria:             []string{"Sequenced bootstrap workstreams are complete.", "A follow-up reality-pro run improves readiness verdict or shrinks blocked zones."},
			JustificationClaimIDs:    phaseJustification(finalClaimIndex, "finding:contract-boundary:final", "finding:hotspot-risk:final", "finding:unresolved-questions:final"),
		},
	}

	if currentVerdict == "ready" {
		phases = phases[:1]
		phases[0].Title = "Maintain readiness"
		phases[0].Objective = "Keep evidence, boundaries, and verification current while the system stays ready."
		phases[0].BlockedZones = nil
		phases[0].ExitCriteria = []string{"Reality-pro reruns keep the verdict at ready.", "New workstreams preserve evidence quality and schema-valid artifacts."}
	}

	return AgentReadinessPlan{
		SpecVersion:    specVersion,
		GeneratedAt:    generatedAt,
		CurrentVerdict: currentVerdict,
		TargetVerdict:  targetVerdict,
		Phases:         phases,
		KeyRisks:       readinessKeyRisks(memory, intent, conflicts),
		SequencingNotes: []string{
			"Do boundary clarity before hotspot work widens scope.",
			"Keep blocked zones explicit until tests or narrower seams reduce risk.",
			"Rerun reality-pro after each bootstrap slice to refresh memory and reviewed synthesis.",
		},
		Claims:  claims,
		Sources: sources,
	}
}

func reportSystemName(projectRoot string, memory RepoMemory) string {
	if len(memory.Repos) == 1 {
		return memory.Repos[0].Name
	}
	if projectRoot != "" {
		base := filepath.Base(projectRoot)
		if base != "" && base != "." && base != string(filepath.Separator) {
			return base + " reposet"
		}
	}
	if len(memory.Repos) > 0 {
		return memory.Repos[0].Name + " reposet"
	}
	return "reality-pro reposet"
}

func repoDescription(repo RepoRecord) string {
	if strings.TrimSpace(repo.Summary) != "" {
		return repo.Summary
	}
	switch repo.Role {
	case "service":
		return "Primary service and runtime workspace."
	case "protocol":
		return "Protocol, schema, and prompt surface consumed by other repos."
	case "infra":
		return "Infrastructure and deployment configuration."
	default:
		return "Mixed repository role reconstructed from code shape."
	}
}

func repoTechnology(repo RepoRecord) string {
	switch repo.Role {
	case "service":
		return "Go workspace"
	case "protocol":
		return "Go, Markdown, JSON Schema"
	case "infra":
		return "Infrastructure configuration"
	default:
		return "Repository workspace"
	}
}

func repoResponsibilities(repo RepoRecord, modules []ModuleSummary) []string {
	names := moduleNames(modules)
	result := []string{}
	switch repo.Role {
	case "service":
		result = append(result, "Owns executable runtime and delivery automation")
	case "protocol":
		result = append(result, "Publishes prompts, contracts, and protocol runtime surfaces")
	case "infra":
		result = append(result, "Carries deployment and environment boundaries")
	default:
		result = append(result, "Carries mixed runtime, docs, and automation concerns")
	}
	if stringInSlice(names, "cmd") {
		result = append(result, "Exposes operator-facing commands")
	}
	if stringInSlice(names, "internal") || stringInSlice(names, "sdp-plugin") {
		result = append(result, "Contains core implementation logic")
	}
	if stringInSlice(names, "docs") || stringInSlice(names, "src") {
		result = append(result, "Documents expected behavior and usage")
	}
	if stringInSlice(names, "schema") {
		result = append(result, "Defines machine-readable contracts")
	}
	return dedupeStrings(result)
}

func componentContainerID(memory RepoMemory) string {
	if len(memory.Repos) == 1 {
		return containerID(memory.Repos[0].RepoID)
	}
	for _, repo := range memory.Repos {
		if repo.Role == "service" {
			return containerID(repo.RepoID)
		}
	}
	return "container:workspace"
}

func selectComponentModules(memory RepoMemory, moduleIndex map[string][]ModuleSummary) []ModuleSummary {
	selected := make([]ModuleSummary, 0)
	for _, repo := range memory.Repos {
		modules := append([]ModuleSummary{}, moduleIndex[repo.RepoID]...)
		sort.Slice(modules, func(i, j int) bool {
			leftName := moduleName(modules[i])
			rightName := moduleName(modules[j])
			leftWeight := moduleSelectionWeight(leftName, modules[i].RiskLevel)
			rightWeight := moduleSelectionWeight(rightName, modules[j].RiskLevel)
			if leftWeight == rightWeight {
				return leftName < rightName
			}
			return leftWeight > rightWeight
		})
		limit := 5
		if len(modules) < limit {
			limit = len(modules)
		}
		if limit == 0 {
			continue
		}
		selected = append(selected, modules[:limit]...)
	}
	return selected
}

func buildComponentRelationships(memory RepoMemory, repoByID map[string]RepoRecord, selected []ModuleSummary, componentByKey map[string]C4Component) []C4Relationship {
	moduleMap := map[string]ModuleSummary{}
	modulesByRepo := map[string]map[string]ModuleSummary{}
	for _, module := range selected {
		moduleMap[module.ModuleID] = module
		if modulesByRepo[module.RepoID] == nil {
			modulesByRepo[module.RepoID] = map[string]ModuleSummary{}
		}
		modulesByRepo[module.RepoID][moduleName(module)] = module
	}

	relationships := make([]C4Relationship, 0)
	add := func(prefix, fromModuleID, desc, toModuleID, tech string, confidence float64) {
		fromComponent, okFrom := componentByKey[fromModuleID]
		toComponent, okTo := componentByKey[toModuleID]
		if !okFrom || !okTo {
			return
		}
		relationships = append(relationships, C4Relationship{
			RelationshipID: relationshipID(prefix, fromComponent.ComponentID, desc, toComponent.ComponentID),
			From:           fromComponent.ComponentID,
			To:             toComponent.ComponentID,
			Description:    desc,
			Technology:     tech,
			Confidence:     confidence,
		})
	}

	for _, repo := range memory.Repos {
		perRepo := modulesByRepo[repo.RepoID]
		if perRepo == nil {
			continue
		}
		if cmd, ok := perRepo["cmd"]; ok {
			if internal, ok := perRepo["internal"]; ok {
				add("component-rel", cmd.ModuleID, "invokes runtime packages", internal.ModuleID, "Go packages", 0.9)
			}
			if plugin, ok := perRepo["sdp-plugin"]; ok {
				add("component-rel", cmd.ModuleID, "surfaces plugin runtime", plugin.ModuleID, "CLI", 0.88)
			}
		}
		if docs, ok := perRepo["docs"]; ok {
			if internal, ok := perRepo["internal"]; ok {
				add("component-rel", docs.ModuleID, "documents runtime behavior", internal.ModuleID, "Markdown", 0.74)
			}
		}
		if scripts, ok := perRepo["scripts"]; ok {
			if cmd, ok := perRepo["cmd"]; ok {
				add("component-rel", scripts.ModuleID, "automates command workflows", cmd.ModuleID, "shell automation", 0.78)
			} else if internal, ok := perRepo["internal"]; ok {
				add("component-rel", scripts.ModuleID, "automates internal workflows", internal.ModuleID, "shell automation", 0.72)
			}
		}
		if tests, ok := perRepo["tests"]; ok {
			if internal, ok := perRepo["internal"]; ok {
				add("component-rel", tests.ModuleID, "verifies runtime paths", internal.ModuleID, "tests", 0.8)
			}
		}
		if src, ok := perRepo["src"]; ok {
			if plugin, ok := perRepo["sdp-plugin"]; ok {
				add("component-rel", src.ModuleID, "provides assets consumed by plugin runtime", plugin.ModuleID, "protocol assets", 0.81)
			}
		}
	}

	serviceModule := preferredModuleForRole(memory, modulesByRepo, "service")
	protocolModule := preferredModuleForRole(memory, modulesByRepo, "protocol")
	if serviceModule.ModuleID != "" && protocolModule.ModuleID != "" {
		add("component-rel", serviceModule.ModuleID, "consumes protocol contracts and skills", protocolModule.ModuleID, "schemas and prompts", 0.84)
	}

	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].RelationshipID < relationships[j].RelationshipID
	})
	return dedupeRelationships(relationships)
}

func preferredModuleForRole(memory RepoMemory, modulesByRepo map[string]map[string]ModuleSummary, role string) ModuleSummary {
	preferred := []string{"internal", "cmd", "sdp-plugin", "src", "docs", "scripts"}
	for _, repo := range memory.Repos {
		if repo.Role != role {
			continue
		}
		perRepo := modulesByRepo[repo.RepoID]
		for _, name := range preferred {
			if module, ok := perRepo[name]; ok {
				return module
			}
		}
		for _, module := range perRepo {
			return module
		}
	}
	return ModuleSummary{}
}

func moduleSelectionWeight(name, risk string) int {
	base := 0
	switch name {
	case "internal":
		base = 120
	case "cmd":
		base = 115
	case "sdp-plugin":
		base = 110
	case "src":
		base = 100
	case "docs":
		base = 95
	case "schema", "prompts":
		base = 90
	case "scripts":
		base = 85
	case "tests":
		base = 80
	case "hooks", "ci":
		base = 70
	case ".tmp":
		base = -100
	default:
		if strings.HasPrefix(name, ".") {
			base = 20
		} else if name == "root" {
			base = 10
		} else {
			base = 60
		}
	}
	return base + riskWeight(risk)
}

func componentDisplayName(repo RepoRecord, module ModuleSummary) string {
	name := moduleName(module)
	if repo.Name == "" {
		return name
	}
	return repo.Name + "/" + name
}

func moduleName(module ModuleSummary) string {
	return strings.TrimPrefix(module.ModuleID, fmt.Sprintf("module:%s:", module.RepoID))
}

func moduleTechnology(repo RepoRecord, module ModuleSummary) string {
	extSeen := map[string]bool{}
	for _, path := range module.Paths {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			extSeen[ext] = true
		}
	}
	switch {
	case extSeen[".go"] && (moduleName(module) == "internal" || moduleName(module) == "cmd" || moduleName(module) == "sdp-plugin"):
		return "Go"
	case extSeen[".sh"]:
		return "Shell"
	case extSeen[".md"]:
		return "Markdown"
	case extSeen[".json"]:
		return "JSON"
	case repo.Role == "protocol":
		return "Markdown and JSON Schema"
	default:
		return repoTechnology(repo)
	}
}

func modulePathsForComponent(module ModuleSummary) []string {
	paths := append([]string{}, module.Paths...)
	sort.Strings(paths)
	if len(paths) > 20 {
		paths = paths[:20]
	}
	return paths
}

func moduleResponsibilities(repo RepoRecord, module ModuleSummary) []string {
	name := moduleName(module)
	result := []string{}
	switch name {
	case "internal":
		result = append(result, "Implements core runtime behavior")
	case "cmd":
		result = append(result, "Exposes executable entrypoints")
	case "docs":
		result = append(result, "Captures operator-facing intent and guidance")
	case "scripts":
		result = append(result, "Automates repeatable maintenance and quality tasks")
	case "tests":
		result = append(result, "Verifies behavior at repository level")
	case "schema":
		result = append(result, "Defines open machine-readable contracts")
	case "prompts":
		result = append(result, "Stores protocol prompt surfaces")
	case "sdp-plugin":
		result = append(result, "Carries protocol runtime and CLI integration logic")
	default:
		result = append(result, module.Summary)
	}
	if len(module.Interfaces) > 0 {
		result = append(result, "Interfaces: "+strings.Join(dedupeStrings(module.Interfaces), ", "))
	}
	if repo.Role == "protocol" && (name == "src" || name == "schema") {
		result = append(result, "Feeds downstream service repos with protocol assets")
	}
	return dedupeStrings(result)
}

func finalClaimIndex(claims []ReviewClaim) map[string]ReviewClaim {
	index := map[string]ReviewClaim{}
	for _, claim := range claims {
		if claim.ReviewState != "arbitrated" {
			continue
		}
		index[claim.ClaimID] = claim
	}
	return index
}

func evidenceClaimIDs(candidateIDs []string, finalClaimIndex map[string]ReviewClaim) []string {
	result := make([]string, 0, len(candidateIDs))
	for _, claimID := range candidateIDs {
		if _, ok := finalClaimIndex[claimID]; ok {
			result = append(result, claimID)
		}
	}
	if len(result) == 0 {
		for claimID := range finalClaimIndex {
			result = append(result, claimID)
			break
		}
	}
	sort.Strings(result)
	return result
}

func backlogScope(gap IntentGapItem, memory RepoMemory) []string {
	scope := make([]string, 0)
	switch {
	case strings.Contains(gap.GapID, "contract-boundary"):
		scope = append(scope, "docs/reality/", "docs/specs/", "docs/workstreams/")
	case strings.Contains(gap.GapID, "ownership"):
		scope = append(scope, ".github/CODEOWNERS", "CODEOWNERS", "OWNERS", ".github/OWNERS", "docs/reality/multi-repo-map.md")
	case strings.Contains(gap.GapID, "hotspot"):
		scope = append(scope, hotspotPathsByRepo(memory.Hotspots, gap.AffectedRepos, 4)...)
	case strings.Contains(gap.GapID, "unresolved"):
		scope = append(scope, "docs/reality/intent-gap.md", ".sdp/reality/bootstrap-backlog.json")
	default:
		scope = append(scope, gap.AffectedRepos...)
	}
	if len(scope) == 0 {
		scope = append(scope, gap.AffectedRepos...)
	}
	return dedupeStrings(scope)
}

func backlogGoal(gap IntentGapItem) string {
	if len(gap.RecommendedActions) > 0 {
		return gap.RecommendedActions[0]
	}
	return gap.ExpectedState
}

func backlogExitCriteria(gap IntentGapItem) []string {
	result := []string{
		"Evidence-backed follow-up reduces or resolves the gap.",
		"Reviewed artifacts can cite concrete proof instead of operator memory.",
	}
	if strings.Contains(gap.GapID, "ownership") {
		result = append(result, "Ownership zones and escalation targets are explicit in repo memory.")
	}
	if gap.ExpectedState != "" {
		result = append(result, gap.ExpectedState)
	}
	return dedupeStrings(result)
}

func recommendedAgent(gap IntentGapItem) string {
	switch {
	case strings.Contains(gap.GapID, "ownership"):
		return "ownership-analyst"
	case strings.Contains(gap.GapID, "hotspot"):
		return "test-quality-analyst"
	case strings.Contains(gap.GapID, "unresolved"):
		return "documentation-analyst"
	default:
		return "architecture-analyst"
	}
}

func severityToPriority(severity string) string {
	switch severity {
	case "high":
		return "P1"
	case "medium":
		return "P2"
	default:
		return "P3"
	}
}

func workstreamPriorityWeight(priority string) int {
	switch priority {
	case "P0":
		return 0
	case "P1":
		return 1
	case "P2":
		return 2
	default:
		return 3
	}
}

func readinessVerdict(memory RepoMemory, intent IntentGapReport, conflicts ConflictReport) string {
	highGaps := 0
	for _, gap := range intent.Gaps {
		if gap.Severity == "high" {
			highGaps++
		}
	}
	switch {
	case highGaps > 0 && (len(conflicts.Conflicts) > 0 || len(memory.Hotspots) > 10):
		return "not_ready"
	case len(intent.Gaps) > 0 || len(conflicts.Conflicts) > 0 || len(memory.UnresolvedQuestions) > 0 || len(memory.Hotspots) > 0:
		return "ready_with_constraints"
	default:
		return "ready"
	}
}

func phaseJustification(finalClaimIndex map[string]ReviewClaim, ids ...string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := finalClaimIndex[id]; ok {
			result = append(result, id)
		}
	}
	sort.Strings(result)
	return result
}

func readinessAllowedScope(workstreams []BootstrapWorkstream) []string {
	result := make([]string, 0)
	for _, item := range workstreams {
		if item.Status == "blocked" {
			continue
		}
		result = append(result, item.Scope...)
	}
	if len(result) == 0 {
		return []string{"docs/reality/", ".sdp/reality/"}
	}
	return dedupeStrings(result)
}

func readinessKeyRisks(memory RepoMemory, intent IntentGapReport, conflicts ConflictReport) []string {
	result := make([]string, 0)
	for _, gap := range intent.Gaps {
		result = append(result, gap.Title)
	}
	for _, item := range conflicts.Conflicts {
		result = append(result, item.Summary)
	}
	for _, path := range hotspotPaths(memory.Hotspots, 3) {
		result = append(result, "Hotspot: "+path)
	}
	return dedupeStrings(result)
}

func hotspotPaths(hotspots []HotspotRecord, limit int) []string {
	if limit <= 0 {
		return nil
	}
	filtered := append([]HotspotRecord{}, hotspots...)
	sort.Slice(filtered, func(i, j int) bool {
		leftWeight := hotspotSortWeight(filtered[i])
		rightWeight := hotspotSortWeight(filtered[j])
		if leftWeight == rightWeight {
			if filtered[i].RepoID == filtered[j].RepoID {
				return filtered[i].Path < filtered[j].Path
			}
			return filtered[i].RepoID < filtered[j].RepoID
		}
		return leftWeight > rightWeight
	})

	result := make([]string, 0, limit)
	for _, hotspot := range filtered {
		if strings.Contains(hotspot.Path, ".tmp/") && len(filtered) > limit {
			continue
		}
		result = append(result, hotspot.RepoID+":"+hotspot.Path)
		if len(result) == limit {
			break
		}
	}
	if len(result) == 0 {
		for _, hotspot := range filtered {
			result = append(result, hotspot.RepoID+":"+hotspot.Path)
			if len(result) == limit {
				break
			}
		}
	}
	return dedupeStrings(result)
}

func hotspotPathsByRepo(hotspots []HotspotRecord, repoIDs []string, limit int) []string {
	repoSet := map[string]bool{}
	for _, repoID := range repoIDs {
		repoSet[repoID] = true
	}
	filtered := make([]HotspotRecord, 0)
	for _, hotspot := range hotspots {
		if len(repoSet) == 0 || repoSet[hotspot.RepoID] {
			filtered = append(filtered, hotspot)
		}
	}
	return hotspotPaths(filtered, limit)
}

func hotspotSortWeight(hotspot HotspotRecord) int {
	weight := 0
	if hotspot.Severity == "high" {
		weight += 10
	}
	if !strings.Contains(hotspot.Path, ".tmp/") {
		weight += 5
	}
	return weight
}

func repoLinkTechnology(link repoLink) string {
	switch link.Relation {
	case "consumes contracts from":
		return "schemas and prompts"
	case "contains":
		return "workspace nesting"
	default:
		return "repository relationship"
	}
}

func repoLinkConfidence(link repoLink) float64 {
	switch link.Relation {
	case "contains":
		return 0.94
	case "consumes contracts from":
		return 0.84
	default:
		return 0.7
	}
}

func mergeClaims(groups ...[]ReviewClaim) []ReviewClaim {
	index := map[string]ReviewClaim{}
	for _, group := range groups {
		for _, claim := range group {
			if existing, ok := index[claim.ClaimID]; ok {
				if claim.ReviewState == "arbitrated" && existing.ReviewState != "arbitrated" {
					index[claim.ClaimID] = claim
				}
				continue
			}
			index[claim.ClaimID] = claim
		}
	}
	result := make([]ReviewClaim, 0, len(index))
	for _, claim := range index {
		result = append(result, claim)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClaimID < result[j].ClaimID
	})
	return result
}

func mergeSources(groups ...[]ReviewSource) []ReviewSource {
	index := map[string]ReviewSource{}
	for _, group := range groups {
		for _, source := range group {
			if source.SourceID == "" {
				continue
			}
			index[source.SourceID] = source
		}
	}
	result := make([]ReviewSource, 0, len(index))
	for _, source := range index {
		result = append(result, source)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].SourceID < result[j].SourceID
	})
	return result
}

func ownershipPeopleAndRelationships(memory RepoMemory) ([]C4Person, []C4Relationship) {
	peopleMap := map[string]C4Person{}
	relationships := make([]C4Relationship, 0)
	teamIndex := map[string]TeamMetadata{}
	for _, team := range memory.Teams {
		teamIndex[team.TeamID] = team
	}

	addPerson := func(personID, name, description string) {
		if personID == "" || name == "" {
			return
		}
		peopleMap[personID] = C4Person{
			PersonID:    personID,
			Name:        name,
			Description: description,
		}
	}

	for _, team := range memory.Teams {
		addPerson(ownerPersonID(team.TeamID), team.Name, ownershipDescription(team.Name, team.Contact, team.EscalationTarget))
	}

	for _, zone := range memory.OwnershipZones {
		personIDs := make([]string, 0)
		for _, teamID := range zone.TeamIDs {
			if team, ok := teamIndex[teamID]; ok {
				personID := ownerPersonID(team.TeamID)
				addPerson(personID, team.Name, ownershipDescription(team.Name, team.Contact, team.EscalationTarget))
				personIDs = append(personIDs, personID)
			}
		}
		if len(personIDs) == 0 {
			for _, owner := range zone.Owners {
				personID := ownerPersonID(owner)
				addPerson(personID, ownerDisplayName(owner), "Listed as owner for a reconstructed repo boundary.")
				personIDs = append(personIDs, personID)
			}
		}

		for _, personID := range dedupeStrings(personIDs) {
			description := "owns repo boundary"
			if zone.Pattern != "" && zone.Pattern != "/" {
				description = "owns " + zone.Pattern
			}
			relationships = append(relationships, C4Relationship{
				RelationshipID: relationshipID("system-rel", personID, description, systemID(zone.RepoID)),
				From:           personID,
				To:             systemID(zone.RepoID),
				Description:    description,
				Technology:     "ownership metadata",
				Confidence:     0.78,
			})
		}
	}

	people := make([]C4Person, 0, len(peopleMap))
	for _, person := range peopleMap {
		people = append(people, person)
	}
	sort.Slice(people, func(i, j int) bool {
		return people[i].PersonID < people[j].PersonID
	})
	return people, dedupeRelationships(relationships)
}

func ownerPersonID(value string) string {
	return "person:" + sanitizeID(strings.TrimPrefix(value, "@"))
}

func ownerDisplayName(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "@")
}

func ownershipDescription(name, contact, escalation string) string {
	parts := []string{fmt.Sprintf("%s owns a reconstructed repo boundary.", name)}
	if contact != "" {
		parts = append(parts, "Contact: "+contact+".")
	}
	if escalation != "" {
		parts = append(parts, "Escalation: "+escalation+".")
	}
	return strings.Join(parts, " ")
}

func systemID(repoID string) string {
	return strings.Replace(repoID, "repo:", "system:", 1)
}

func containerID(repoID string) string {
	return strings.Replace(repoID, "repo:", "container:", 1)
}

func componentID(moduleID string) string {
	return strings.Replace(moduleID, "module:", "component:", 1)
}

func relationshipID(prefix, from, relation, to string) string {
	return fmt.Sprintf("%s:%s:%s:%s", prefix, sanitizeID(from), sanitizeID(relation), sanitizeID(to))
}

func dedupeRelationships(items []C4Relationship) []C4Relationship {
	seen := map[string]bool{}
	result := make([]C4Relationship, 0, len(items))
	for _, item := range items {
		if seen[item.RelationshipID] {
			continue
		}
		seen[item.RelationshipID] = true
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].RelationshipID < result[j].RelationshipID
	})
	return result
}

func moduleNames(modules []ModuleSummary) []string {
	names := make([]string, 0, len(modules))
	for _, module := range modules {
		names = append(names, moduleName(module))
	}
	sort.Strings(names)
	return names
}

func stringInSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func severityWeight(value string) int {
	switch value {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func riskWeight(value string) int {
	switch value {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func renderSystemContextMarkdown(view C4SystemContext, repoByID map[string]RepoRecord) string {
	var b strings.Builder
	b.WriteString("# Reality C4 System Context\n\n")
	b.WriteString(fmt.Sprintf("- Generated At: `%s`\n", view.GeneratedAt))
	b.WriteString(fmt.Sprintf("- System Scope: `%s`\n", view.Scope.SystemName))
	b.WriteString(fmt.Sprintf("- Repositories: `%s`\n", strings.Join(view.Scope.Repos, "`, `")))
	b.WriteString("\n## Systems\n\n")
	b.WriteString("| System | Boundary | Repos | Notes |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, system := range view.Systems {
		repoNames := repoNames(system.RepoIDs, repoByID)
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s |\n", system.Name, system.Boundary, strings.Join(repoNames, ", "), system.Description))
	}
	b.WriteString("\n## Relationships\n\n")
	for _, rel := range view.Relationships {
		b.WriteString(fmt.Sprintf("- `%s` -> `%s`: %s\n", rel.From, rel.To, rel.Description))
	}
	if len(view.People) > 0 {
		b.WriteString("\n## Review Roles\n\n")
		for _, person := range view.People {
			b.WriteString(fmt.Sprintf("- `%s`: %s\n", person.Name, person.Description))
		}
	}
	return b.String()
}

func renderContainerMarkdown(view C4ContainerView, repoByID map[string]RepoRecord) string {
	var b strings.Builder
	b.WriteString("# Reality C4 Containers\n\n")
	b.WriteString(fmt.Sprintf("- Generated At: `%s`\n", view.GeneratedAt))
	b.WriteString(fmt.Sprintf("- System Name: `%s`\n", view.SystemName))
	b.WriteString("\n## Containers\n\n")
	b.WriteString("| Container | Repo | Technology | Responsibilities |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, container := range view.Containers {
		repoName := container.RepoID
		if repo, ok := repoByID[container.RepoID]; ok {
			repoName = repo.Name
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s |\n", container.Name, repoName, container.Technology, strings.Join(container.Responsibilities, "; ")))
	}
	b.WriteString("\n## Relationships\n\n")
	for _, rel := range view.Relationships {
		b.WriteString(fmt.Sprintf("- `%s` -> `%s`: %s\n", rel.From, rel.To, rel.Description))
	}
	return b.String()
}

func renderComponentMarkdown(view C4ComponentView) string {
	var b strings.Builder
	b.WriteString("# Reality C4 Components\n\n")
	b.WriteString(fmt.Sprintf("- Generated At: `%s`\n", view.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Container Scope: `%s`\n", view.ContainerID))
	b.WriteString("\n## Components\n\n")
	for _, component := range view.Components {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", component.Name))
		b.WriteString(fmt.Sprintf("- ID: `%s`\n", component.ComponentID))
		b.WriteString(fmt.Sprintf("- Technology: `%s`\n", component.Technology))
		if component.Description != "" {
			b.WriteString(fmt.Sprintf("- Summary: %s\n", component.Description))
		}
		if len(component.Responsibilities) > 0 {
			b.WriteString(fmt.Sprintf("- Responsibilities: %s\n", strings.Join(component.Responsibilities, "; ")))
		}
		if len(component.Paths) > 0 {
			preview := component.Paths
			if len(preview) > 6 {
				preview = preview[:6]
			}
			b.WriteString(fmt.Sprintf("- Paths: `%s`\n", strings.Join(preview, "`, `")))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Relationships\n\n")
	for _, rel := range view.Relationships {
		b.WriteString(fmt.Sprintf("- `%s` -> `%s`: %s\n", rel.From, rel.To, rel.Description))
	}
	return b.String()
}

func renderIntentGapMarkdown(intent IntentGapReport, conflicts ConflictReport, backlog BootstrapBacklog, readiness AgentReadinessPlan, repoByID map[string]RepoRecord) string {
	var b strings.Builder
	b.WriteString("# Reality Intent Gap Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated At: `%s`\n", intent.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Gaps: `%d`\n", len(intent.Gaps)))
	b.WriteString(fmt.Sprintf("- Conflicts: `%d`\n", len(conflicts.Conflicts)))
	b.WriteString(fmt.Sprintf("- Current Readiness: `%s`\n", readiness.CurrentVerdict))
	b.WriteString(fmt.Sprintf("- Target Readiness: `%s`\n", readiness.TargetVerdict))
	b.WriteString("\n## Intent Gaps\n\n")
	b.WriteString("| Gap | Severity | Status | Repos | Next Step |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, gap := range intent.Gaps {
		repoNames := repoNames(gap.AffectedRepos, repoByID)
		nextStep := ""
		if len(gap.RecommendedActions) > 0 {
			nextStep = gap.RecommendedActions[0]
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %s |\n", gap.Title, gap.Severity, gap.Status, strings.Join(repoNames, ", "), nextStep))
	}
	b.WriteString("\n## Conflicts\n\n")
	if len(conflicts.Conflicts) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, item := range conflicts.Conflicts {
			b.WriteString(fmt.Sprintf("- `%s`: %s\n", item.ConflictID, item.ResolutionNotes))
		}
	}
	b.WriteString("\n## Bootstrap Workstreams\n\n")
	for _, item := range backlog.Workstreams {
		b.WriteString(fmt.Sprintf("- `%s` [%s/%s]: %s\n", item.Title, item.Priority, item.Status, item.Goal))
	}
	return b.String()
}

func repoNames(ids []string, repoByID map[string]RepoRecord) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if repo, ok := repoByID[id]; ok {
			result = append(result, repo.Name)
			continue
		}
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
