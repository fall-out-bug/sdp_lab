package realitypro

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	specVersion          = "v1.0"
	hotspotLineThreshold = 800
)

type Options struct {
	ProjectRoot string
	Repos       []string
	WithDocs    bool
	DocRoots    []string
	Now         func() time.Time
}

type Result struct {
	RepoMemoryPath   string
	MultiRepoMapPath string
	RepoCount        int
	SourceCount      int
}

type RepoMemory struct {
	SpecVersion               string           `json:"spec_version"`
	GeneratedAt               string           `json:"generated_at"`
	Repos                     []RepoRecord     `json:"repos"`
	ModuleSummaries           []ModuleSummary  `json:"module_summaries"`
	FeatureMappings           []FeatureMapping `json:"feature_mappings,omitempty"`
	OwnershipZones            []OwnershipZone  `json:"ownership_zones,omitempty"`
	Teams                     []TeamMetadata   `json:"teams,omitempty"`
	PreviousValidatedClaimIDs []string         `json:"previous_validated_claim_ids,omitempty"`
	UnresolvedQuestions       []string         `json:"unresolved_questions"`
	Hotspots                  []HotspotRecord  `json:"hotspots,omitempty"`
	Sources                   []ReviewSource   `json:"sources,omitempty"`
}

type RepoRecord struct {
	RepoID        string `json:"repo_id"`
	Name          string `json:"name"`
	RootPath      string `json:"root_path"`
	Role          string `json:"role,omitempty"`
	Summary       string `json:"summary,omitempty"`
	LastIndexedAt string `json:"last_indexed_at,omitempty"`
}

type ModuleSummary struct {
	ModuleID   string   `json:"module_id"`
	RepoID     string   `json:"repo_id"`
	Summary    string   `json:"summary"`
	Paths      []string `json:"paths,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
	RiskLevel  string   `json:"risk_level,omitempty"`
}

type FeatureMapping struct {
	FeatureID    string   `json:"feature_id"`
	Title        string   `json:"title"`
	RepoIDs      []string `json:"repo_ids,omitempty"`
	ComponentIDs []string `json:"component_ids,omitempty"`
	Confidence   float64  `json:"confidence"`
}

type HotspotRecord struct {
	HotspotID string `json:"hotspot_id"`
	RepoID    string `json:"repo_id"`
	Path      string `json:"path"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity,omitempty"`
}

type OwnershipZone struct {
	ZoneID           string   `json:"zone_id"`
	RepoID           string   `json:"repo_id"`
	Pattern          string   `json:"pattern"`
	Owners           []string `json:"owners"`
	TeamIDs          []string `json:"team_ids,omitempty"`
	EscalationTarget string   `json:"escalation_target,omitempty"`
	Responsibility   string   `json:"responsibility,omitempty"`
	SourceID         string   `json:"source_id,omitempty"`
}

type TeamMetadata struct {
	TeamID           string   `json:"team_id"`
	Name             string   `json:"name"`
	RepoID           string   `json:"repo_id,omitempty"`
	SourceID         string   `json:"source_id,omitempty"`
	Aliases          []string `json:"aliases,omitempty"`
	Contact          string   `json:"contact,omitempty"`
	Slack            string   `json:"slack,omitempty"`
	Email            string   `json:"email,omitempty"`
	EscalationTarget string   `json:"escalation_target,omitempty"`
	Owns             []string `json:"owns,omitempty"`
}

type repoScan struct {
	Record              RepoRecord
	Modules             []ModuleSummary
	FeatureMappings     []FeatureMapping
	OwnershipZones      []OwnershipZone
	Teams               []TeamMetadata
	Hotspots            []HotspotRecord
	UnresolvedQuestions []string
	ModuleNames         []string
	Sources             []ReviewSource
}

type repoLink struct {
	From     string
	To       string
	Relation string
}

type evidenceCoverageItem struct {
	Kind  string
	Repo  string
	Count int
}

// Ingest builds or refreshes reality-pro persistent memory for one repo or an explicit reposet.
func Ingest(opts Options) (Result, error) {
	projectRoot, repoRoots, now, err := normalizeOptions(opts)
	if err != nil {
		return Result{}, err
	}
	withDocs := opts.WithDocs || len(opts.DocRoots) > 0

	existing, err := loadExistingMemory(projectRoot)
	if err != nil {
		return Result{}, err
	}

	scans := make([]repoScan, 0, len(repoRoots))
	for _, repoRoot := range repoRoots {
		scan, err := scanRepo(projectRoot, repoRoot, repoRoots, now)
		if err != nil {
			return Result{}, err
		}
		scans = append(scans, scan)
	}

	sources := make([]ReviewSource, 0)
	if withDocs {
		sources, err = scanEvidenceSources(projectRoot, repoRoots, opts.DocRoots)
		if err != nil {
			return Result{}, err
		}
	}

	generatedAt := now.UTC().Format(time.RFC3339)
	updated := mergeMemory(existing, scans, generatedAt, sources, withDocs)
	if err := writeJSON(filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"), updated); err != nil {
		return Result{}, err
	}

	repoIndex := make(map[string]RepoRecord, len(updated.Repos))
	moduleIndex := make(map[string][]ModuleSummary, len(updated.Repos))
	for _, repo := range updated.Repos {
		repoIndex[repo.RepoID] = repo
	}
	for _, module := range updated.ModuleSummaries {
		moduleIndex[module.RepoID] = append(moduleIndex[module.RepoID], module)
	}
	if err := writeText(
		filepath.Join(projectRoot, "docs", "reality", "multi-repo-map.md"),
		renderMultiRepoMap(updated, buildRepoLinks(updated.Repos), moduleIndex),
	); err != nil {
		return Result{}, err
	}

	return Result{
		RepoMemoryPath:   filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json"),
		MultiRepoMapPath: filepath.Join(projectRoot, "docs", "reality", "multi-repo-map.md"),
		RepoCount:        len(scans),
		SourceCount:      len(updated.Sources),
	}, nil
}

func normalizeOptions(opts Options) (string, []string, time.Time, error) {
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("resolve project root: %w", err)
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()

	repoRoots := opts.Repos
	if len(repoRoots) == 0 {
		repoRoots = []string{absProjectRoot}
	}

	seen := map[string]bool{}
	resolved := make([]string, 0, len(repoRoots))
	for _, repo := range repoRoots {
		if strings.TrimSpace(repo) == "" {
			continue
		}
		absRepo, err := filepath.Abs(repo)
		if err != nil {
			return "", nil, time.Time{}, fmt.Errorf("resolve repo %q: %w", repo, err)
		}
		if seen[absRepo] {
			continue
		}
		info, err := os.Stat(absRepo)
		if err != nil {
			return "", nil, time.Time{}, fmt.Errorf("stat repo %q: %w", absRepo, err)
		}
		if !info.IsDir() {
			return "", nil, time.Time{}, fmt.Errorf("repo path %q is not a directory", absRepo)
		}
		seen[absRepo] = true
		resolved = append(resolved, absRepo)
	}
	sort.Strings(resolved)
	return absProjectRoot, resolved, now, nil
}

func scanRepo(projectRoot, repoRoot string, allRepoRoots []string, now time.Time) (repoScan, error) {
	repoID := repoID(projectRoot, repoRoot)
	record := RepoRecord{
		RepoID:        repoID,
		Name:          filepath.Base(repoRoot),
		RootPath:      filepath.ToSlash(repoRoot),
		LastIndexedAt: now.UTC().Format(time.RFC3339),
	}

	sourceExt := map[string]bool{
		".go":   true,
		".py":   true,
		".js":   true,
		".ts":   true,
		".java": true,
		".rs":   true,
		".sh":   true,
	}
	modules := map[string]int{}
	modulePaths := map[string][]string{}
	hotspots := make([]HotspotRecord, 0)
	unresolved := make([]string, 0)
	sourceCount := 0
	testCount := 0
	hasGoMod := false
	hasDeploy := false
	hasSchema := false
	hasPrompts := false

	nestedRepoRoots := nestedRepoSet(repoRoot, allRepoRoots)
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != repoRoot && nestedRepoRoots[path] {
				return filepath.SkipDir
			}
			base := d.Name()
			if base == ".git" || base == ".sdp" || base == ".beads" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		switch {
		case rel == "go.mod":
			hasGoMod = true
		case strings.HasPrefix(rel, "deploy/"):
			hasDeploy = true
		case strings.HasPrefix(rel, "schema/"):
			hasSchema = true
		case strings.HasPrefix(rel, "prompts/"):
			hasPrompts = true
		}

		ext := strings.ToLower(filepath.Ext(rel))
		if !sourceExt[ext] {
			return nil
		}

		sourceCount++
		if isTestFile(rel) {
			testCount++
		}
		module := topModule(rel)
		modules[module]++
		modulePaths[module] = append(modulePaths[module], rel)

		lineCount := countLines(path)
		if lineCount >= hotspotLineThreshold {
			hotspots = append(hotspots, HotspotRecord{
				HotspotID: hotspotID(repoID, rel),
				RepoID:    repoID,
				Path:      rel,
				Reason:    fmt.Sprintf("line_count=%d", lineCount),
				Severity:  "high",
			})
		}
		return nil
	})
	if err != nil {
		return repoScan{}, fmt.Errorf("scan repo %s: %w", repoRoot, err)
	}

	record.Role = classifyRole(hasGoMod, hasDeploy, hasSchema, hasPrompts, modules)
	record.Summary = fmt.Sprintf("%s repo with %d source files across %d top-level modules.", record.Role, sourceCount, len(modules))

	moduleSummaries := make([]ModuleSummary, 0, len(modules))
	moduleNames := make([]string, 0, len(modules))
	for module, count := range modules {
		moduleNames = append(moduleNames, module)
		paths := dedupeStrings(modulePaths[module])
		sort.Strings(paths)
		riskLevel := "low"
		if count >= 10 {
			riskLevel = "medium"
		}
		moduleSummaries = append(moduleSummaries, ModuleSummary{
			ModuleID:   fmt.Sprintf("module:%s:%s", repoID, module),
			RepoID:     repoID,
			Summary:    fmt.Sprintf("%s contains %d source files.", module, count),
			Paths:      paths,
			Interfaces: detectInterfaces(paths),
			RiskLevel:  riskLevel,
		})
	}
	sort.Strings(moduleNames)
	sort.Slice(moduleSummaries, func(i, j int) bool {
		return moduleSummaries[i].ModuleID < moduleSummaries[j].ModuleID
	})
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].HotspotID < hotspots[j].HotspotID
	})

	if sourceCount == 0 {
		unresolved = append(unresolved, fmt.Sprintf("%s: no source files detected", record.Name))
	}
	if testCount == 0 {
		unresolved = append(unresolved, fmt.Sprintf("%s: where should baseline verification be added first?", record.Name))
	}
	if len(moduleNames) == 0 {
		unresolved = append(unresolved, fmt.Sprintf("%s: module boundaries are still unclear", record.Name))
	}
	if len(allRepoRoots) > 1 {
		unresolved = append(unresolved, fmt.Sprintf("%s: how does this repo coordinate versioning with the rest of the reposet?", record.Name))
	}

	featureMappings := []FeatureMapping{
		{
			FeatureID:    fmt.Sprintf("feature:%s:baseline", repoID),
			Title:        fmt.Sprintf("%s baseline", record.Name),
			RepoIDs:      []string{repoID},
			ComponentIDs: moduleIDs(moduleSummaries),
			Confidence:   confidenceForRepo(sourceCount, testCount),
		},
	}

	ownershipZones, teams, ownershipSources, ownershipQuestions, err := scanOwnershipMetadata(projectRoot, repoRoot, repoID)
	if err != nil {
		return repoScan{}, fmt.Errorf("scan ownership metadata for %s: %w", repoRoot, err)
	}
	unresolved = append(unresolved, ownershipQuestions...)

	return repoScan{
		Record:              record,
		Modules:             moduleSummaries,
		FeatureMappings:     featureMappings,
		OwnershipZones:      ownershipZones,
		Teams:               teams,
		Hotspots:            hotspots,
		UnresolvedQuestions: dedupeStrings(unresolved),
		ModuleNames:         moduleNames,
		Sources:             ownershipSources,
	}, nil
}

func scanOwnershipMetadata(projectRoot, repoRoot, repoID string) ([]OwnershipZone, []TeamMetadata, []ReviewSource, []string, error) {
	zoneMap := map[string]OwnershipZone{}
	teamMap := map[string]TeamMetadata{}
	sourceMap := map[string]ReviewSource{}

	for _, path := range ownershipCandidatePaths(repoRoot) {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		source := ownershipSource(projectRoot, repoRoot, path, info.ModTime())
		sourceMap[source.SourceID] = source

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("read %s: %w", path, err)
		}

		switch {
		case strings.EqualFold(filepath.Base(path), "CODEOWNERS"):
			for _, zone := range parseCODEOWNERS(repoID, source.SourceID, string(data)) {
				zoneMap[zone.ZoneID] = zone
			}
		case strings.HasPrefix(strings.ToUpper(filepath.Base(path)), "OWNERS"):
			for _, zone := range parseOwnersFile(repoID, source.SourceID, string(data)) {
				zoneMap[zone.ZoneID] = zone
			}
		default:
			teams, err := parseTeamMetadata(repoID, source.SourceID, path, data)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("parse %s: %w", path, err)
			}
			for _, team := range teams {
				teamMap[team.TeamID] = team
			}
		}
	}

	teams := sortedTeams(teamMap)
	zones := attachTeamMetadataToZones(sortedOwnershipZones(zoneMap), teams)
	questions := ownershipQuestions(repoID, zones, teams)
	return zones, teams, sortedSources(sourceMap), questions, nil
}

func ownershipCandidatePaths(repoRoot string) []string {
	candidates := []string{
		filepath.Join(repoRoot, "CODEOWNERS"),
		filepath.Join(repoRoot, ".github", "CODEOWNERS"),
		filepath.Join(repoRoot, "OWNERS"),
		filepath.Join(repoRoot, "OWNERS.md"),
		filepath.Join(repoRoot, "OWNERS.yaml"),
		filepath.Join(repoRoot, "OWNERS.yml"),
		filepath.Join(repoRoot, "OWNERS.json"),
		filepath.Join(repoRoot, ".github", "OWNERS"),
		filepath.Join(repoRoot, ".github", "OWNERS.md"),
		filepath.Join(repoRoot, ".github", "OWNERS.yaml"),
		filepath.Join(repoRoot, ".github", "OWNERS.yml"),
		filepath.Join(repoRoot, ".github", "OWNERS.json"),
		filepath.Join(repoRoot, "team.json"),
		filepath.Join(repoRoot, "teams.json"),
		filepath.Join(repoRoot, "team.yaml"),
		filepath.Join(repoRoot, "team.yml"),
		filepath.Join(repoRoot, "teams.yaml"),
		filepath.Join(repoRoot, "teams.yml"),
		filepath.Join(repoRoot, "team.md"),
		filepath.Join(repoRoot, "teams.md"),
		filepath.Join(repoRoot, ".github", "team.json"),
		filepath.Join(repoRoot, ".github", "teams.json"),
		filepath.Join(repoRoot, ".github", "team.yaml"),
		filepath.Join(repoRoot, ".github", "team.yml"),
		filepath.Join(repoRoot, ".github", "teams.yaml"),
		filepath.Join(repoRoot, ".github", "teams.yml"),
		filepath.Join(repoRoot, ".github", "team.md"),
		filepath.Join(repoRoot, ".github", "teams.md"),
		filepath.Join(repoRoot, "docs", "team.md"),
		filepath.Join(repoRoot, "docs", "teams.md"),
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		result = append(result, candidate)
	}
	return result
}

func ownershipSource(projectRoot, repoRoot, path string, modTime time.Time) ReviewSource {
	locator := filepath.ToSlash(path)
	if rel, err := filepath.Rel(projectRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
		locator = filepath.ToSlash(rel)
	}
	repoRel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		repoRel = filepath.Base(path)
	}
	kind := "config"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".rst", ".adoc":
		kind = "doc"
	}
	return ReviewSource{
		SourceID: "source:ownership:" + sanitizeID(locator),
		Kind:     kind,
		Locator:  locator,
		Revision: modTime.UTC().Format(time.RFC3339),
		Repo:     repoID(projectRoot, repoRoot),
		Path:     filepath.ToSlash(repoRel),
	}
}

func parseCODEOWNERS(repoID, sourceID, body string) []OwnershipZone {
	zones := make([]OwnershipZone, 0)
	for _, raw := range strings.Split(body, "\n") {
		line := trimInlineComment(raw)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		owners := normalizeOwnerTokens(fields[1:])
		if len(owners) == 0 {
			continue
		}
		zones = append(zones, OwnershipZone{
			ZoneID:         ownershipZoneID(repoID, fields[0]),
			RepoID:         repoID,
			Pattern:        fields[0],
			Owners:         owners,
			Responsibility: "Review and approve changes in the matched boundary.",
			SourceID:       sourceID,
		})
	}
	return zones
}

func parseOwnersFile(repoID, sourceID, body string) []OwnershipZone {
	zones := make([]OwnershipZone, 0)
	repoOwners := make([]string, 0)
	repoEscalation := ""
	responsibility := ""

	for _, raw := range strings.Split(body, "\n") {
		line := trimInlineComment(raw)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && looksLikeOwnershipPattern(fields[0]) {
			owners := normalizeOwnerTokens(fields[1:])
			if len(owners) == 0 {
				continue
			}
			zones = append(zones, OwnershipZone{
				ZoneID:           ownershipZoneID(repoID, fields[0]),
				RepoID:           repoID,
				Pattern:          fields[0],
				Owners:           owners,
				EscalationTarget: repoEscalation,
				Responsibility:   firstNonEmpty(responsibility, "Own and review the matched repo boundary."),
				SourceID:         sourceID,
			})
			continue
		}

		key, value, ok := splitOwnershipKV(line)
		if !ok {
			continue
		}
		switch key {
		case "owners", "owner", "reviewers", "approvers", "team", "teams":
			repoOwners = append(repoOwners, normalizeOwnerTokens(splitOwnershipList(value))...)
		case "escalation", "escalationtarget", "escalate":
			repoEscalation = firstToken(splitOwnershipList(value))
		case "responsibility", "responsibilities", "scope":
			responsibility = strings.TrimSpace(value)
		}
	}

	if len(zones) == 0 && len(repoOwners) > 0 {
		zones = append(zones, OwnershipZone{
			ZoneID:           ownershipZoneID(repoID, "/"),
			RepoID:           repoID,
			Pattern:          "/",
			Owners:           dedupeStrings(repoOwners),
			EscalationTarget: repoEscalation,
			Responsibility:   firstNonEmpty(responsibility, "Own and review repository-wide changes."),
			SourceID:         sourceID,
		})
	}
	return zones
}

func parseTeamMetadata(repoID, sourceID, path string, data []byte) ([]TeamMetadata, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return parseTeamMetadataJSON(repoID, sourceID, data)
	default:
		team := parseTeamMetadataText(repoID, sourceID, string(data))
		if team.TeamID == "" {
			return nil, nil
		}
		return []TeamMetadata{team}, nil
	}
}

func parseTeamMetadataJSON(repoID, sourceID string, data []byte) ([]TeamMetadata, error) {
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0)
	switch typed := payload.(type) {
	case []any:
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				items = append(items, object)
			}
		}
	case map[string]any:
		if nested, ok := typed["teams"].([]any); ok {
			for _, item := range nested {
				if object, ok := item.(map[string]any); ok {
					items = append(items, object)
				}
			}
		} else {
			items = append(items, typed)
		}
	}

	result := make([]TeamMetadata, 0, len(items))
	for _, item := range items {
		name := firstNonEmpty(stringValue(item["name"]), stringValue(item["team"]), stringValue(item["id"]))
		if name == "" {
			continue
		}
		owns := append(stringListValue(item["owns"]), stringListValue(item["paths"])...)
		aliases := append(stringListValue(item["aliases"]), stringListValue(item["owners"])...)
		team := TeamMetadata{
			TeamID:           firstNonEmpty(stringValue(item["team_id"]), "team:"+sanitizeID(name)),
			Name:             name,
			RepoID:           repoID,
			SourceID:         sourceID,
			Aliases:          dedupeStrings(aliases),
			Contact:          firstNonEmpty(stringValue(item["contact"]), stringValue(item["slack"]), stringValue(item["email"])),
			Slack:            stringValue(item["slack"]),
			Email:            stringValue(item["email"]),
			EscalationTarget: firstNonEmpty(stringValue(item["escalation_target"]), stringValue(item["escalation"])),
			Owns:             dedupeStrings(owns),
		}
		result = append(result, team)
	}
	return result, nil
}

func parseTeamMetadataText(repoID, sourceID, body string) TeamMetadata {
	team := TeamMetadata{
		RepoID:   repoID,
		SourceID: sourceID,
	}
	listKey := ""
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			listKey = ""
			continue
		}
		if strings.HasPrefix(line, "- ") && listKey != "" {
			value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			switch listKey {
			case "aliases":
				team.Aliases = append(team.Aliases, normalizeOwnerTokens([]string{value})...)
			case "owns":
				team.Owns = append(team.Owns, normalizePathToken(value))
			}
			continue
		}
		key, value, ok := splitOwnershipKV(line)
		if !ok {
			continue
		}
		switch key {
		case "team", "name", "id":
			if team.Name == "" {
				team.Name = strings.TrimSpace(value)
			}
		case "teamid":
			team.TeamID = strings.TrimSpace(value)
		case "aliases", "owners":
			listKey = "aliases"
			team.Aliases = append(team.Aliases, normalizeOwnerTokens(splitOwnershipList(value))...)
		case "owns", "paths":
			listKey = "owns"
			for _, item := range splitOwnershipList(value) {
				team.Owns = append(team.Owns, normalizePathToken(item))
			}
		case "contact":
			team.Contact = strings.TrimSpace(value)
		case "slack":
			team.Slack = strings.TrimSpace(value)
			if team.Contact == "" {
				team.Contact = team.Slack
			}
		case "email":
			team.Email = strings.TrimSpace(value)
			if team.Contact == "" {
				team.Contact = team.Email
			}
		case "escalation", "escalationtarget":
			team.EscalationTarget = strings.TrimSpace(value)
		}
	}
	if team.Name == "" {
		return TeamMetadata{}
	}
	if team.TeamID == "" {
		team.TeamID = "team:" + sanitizeID(team.Name)
	}
	team.Aliases = dedupeStrings(team.Aliases)
	team.Owns = dedupeStrings(team.Owns)
	return team
}

func attachTeamMetadataToZones(zones []OwnershipZone, teams []TeamMetadata) []OwnershipZone {
	teamByAlias := map[string]TeamMetadata{}
	for _, team := range teams {
		aliases := append([]string{team.TeamID, team.Name}, team.Aliases...)
		for _, alias := range aliases {
			key := normalizeOwnerAlias(alias)
			if key == "" {
				continue
			}
			teamByAlias[key] = team
		}
	}

	for i := range zones {
		teamIDs := append([]string{}, zones[i].TeamIDs...)
		for _, owner := range zones[i].Owners {
			if team, ok := teamByAlias[normalizeOwnerAlias(owner)]; ok {
				teamIDs = append(teamIDs, team.TeamID)
				if zones[i].EscalationTarget == "" && team.EscalationTarget != "" {
					zones[i].EscalationTarget = team.EscalationTarget
				}
			}
		}
		if len(teamIDs) == 0 {
			for _, team := range teams {
				if teamOwnsPattern(team.Owns, zones[i].Pattern) {
					teamIDs = append(teamIDs, team.TeamID)
					if zones[i].EscalationTarget == "" && team.EscalationTarget != "" {
						zones[i].EscalationTarget = team.EscalationTarget
					}
				}
			}
		}
		zones[i].TeamIDs = dedupeStrings(teamIDs)
	}
	return zones
}

func ownershipQuestions(repoID string, zones []OwnershipZone, teams []TeamMetadata) []string {
	name := strings.TrimPrefix(repoID, "repo:")
	questions := make([]string, 0)
	if len(zones) == 0 {
		questions = append(questions, fmt.Sprintf("%s: ownership zones are not explicit; add CODEOWNERS or OWNERS metadata", name))
		return questions
	}
	missingEscalation := 0
	for _, zone := range zones {
		if strings.TrimSpace(zone.EscalationTarget) == "" {
			missingEscalation++
		}
	}
	if missingEscalation > 0 {
		questions = append(questions, fmt.Sprintf("%s: %d ownership zone(s) still lack escalation targets", name, missingEscalation))
	}
	if len(teams) == 0 {
		questions = append(questions, fmt.Sprintf("%s: ownership zones exist, but structured team metadata is still missing", name))
	}
	return dedupeStrings(questions)
}

func ownershipZoneID(repoID, pattern string) string {
	return fmt.Sprintf("zone:%s:%s", repoID, sanitizeID(pattern))
}

func trimInlineComment(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	if idx := strings.Index(line, " #"); idx >= 0 {
		line = line[:idx]
	}
	return strings.TrimSpace(line)
}

func splitOwnershipKV(line string) (string, string, bool) {
	sep := ":"
	if strings.Contains(line, "=") && (!strings.Contains(line, ":") || strings.Index(line, "=") < strings.Index(line, ":")) {
		sep = "="
	}
	parts := strings.SplitN(line, sep, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.ToLower(strings.TrimSpace(parts[0]))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}

func splitOwnershipList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '[', ']', '"', '\'':
			return true
		default:
			return false
		}
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		result = append(result, field)
	}
	return result
}

func normalizeOwnerTokens(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, ",[](){}\"'")
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return dedupeStrings(result)
}

func normalizeOwnerAlias(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "@")
	value = strings.TrimPrefix(value, "team:")
	value = strings.TrimSpace(value)
	return strings.ToLower(value)
}

func normalizePathToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, ",[](){}\"'")
	if value == "" {
		return ""
	}
	return filepath.ToSlash(value)
}

func looksLikeOwnershipPattern(value string) bool {
	return value == "/" || strings.ContainsAny(value, "/*.")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func stringListValue(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				result = append(result, text)
			}
		}
		return dedupeStrings(result)
	case []string:
		return dedupeStrings(typed)
	case string:
		return dedupeStrings(splitOwnershipList(typed))
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstToken(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func teamOwnsPattern(patterns []string, zonePattern string) bool {
	for _, pattern := range patterns {
		pattern = normalizePathToken(pattern)
		if pattern == "" {
			continue
		}
		if pattern == zonePattern || zonePattern == "/" {
			return true
		}
		if strings.HasSuffix(pattern, "**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(zonePattern, prefix) {
				return true
			}
		}
	}
	return false
}

func scanEvidenceSources(projectRoot string, repoRoots, extraDocRoots []string) ([]ReviewSource, error) {
	candidates := make([]string, 0)
	seenRoots := map[string]bool{}
	defaultDocDirs := []string{"docs", "adr", "adrs", "runbook", "runbooks", "playbook", "playbooks"}

	addCandidate := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil || seenRoots[abs] {
			return
		}
		if _, err := os.Stat(abs); err != nil {
			return
		}
		seenRoots[abs] = true
		candidates = append(candidates, abs)
	}

	for _, repoRoot := range repoRoots {
		for _, rel := range defaultDocDirs {
			addCandidate(filepath.Join(repoRoot, rel))
		}
	}
	for _, root := range extraDocRoots {
		for _, item := range splitCSV(root) {
			addCandidate(item)
		}
	}

	sources := make([]ReviewSource, 0)
	seenFiles := map[string]bool{}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			source, ok := buildEvidenceSource(projectRoot, repoRoots, candidate)
			if ok && !seenFiles[candidate] {
				seenFiles[candidate] = true
				sources = append(sources, source)
			}
			continue
		}

		err = filepath.WalkDir(candidate, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if shouldSkipEvidenceDir(path) {
					return filepath.SkipDir
				}
				return nil
			}
			if seenFiles[path] {
				return nil
			}
			source, ok := buildEvidenceSource(projectRoot, repoRoots, path)
			if !ok {
				return nil
			}
			seenFiles[path] = true
			sources = append(sources, source)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan evidence root %s: %w", candidate, err)
		}
	}

	sourceMap := make(map[string]ReviewSource, len(sources))
	for _, source := range sources {
		sourceMap[source.SourceID] = source
	}
	return sortedSources(sourceMap), nil
}

func buildEvidenceSource(projectRoot string, repoRoots []string, path string) (ReviewSource, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".mdx", ".txt", ".rst", ".adoc":
	default:
		return ReviewSource{}, false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ReviewSource{}, false
	}

	locator := filepath.ToSlash(path)
	if rel, err := filepath.Rel(projectRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
		locator = filepath.ToSlash(rel)
	}
	repoID, repoRel := evidenceRepoContext(projectRoot, repoRoots, path)
	return ReviewSource{
		SourceID: "source:evidence:" + sanitizeID(locator),
		Kind:     classifyEvidenceKind(path),
		Locator:  locator,
		Revision: info.ModTime().UTC().Format(time.RFC3339),
		Repo:     repoID,
		Path:     repoRel,
	}, true
}

func evidenceRepoContext(projectRoot string, repoRoots []string, path string) (string, string) {
	for _, repoRoot := range repoRoots {
		if !isNestedPath(repoRoot, path) && filepath.Clean(repoRoot) != filepath.Clean(path) {
			continue
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			continue
		}
		return repoID(projectRoot, repoRoot), filepath.ToSlash(rel)
	}
	return "", ""
}

func classifyEvidenceKind(path string) string {
	lowered := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lowered, "/adr/"), strings.Contains(lowered, "/adrs/"), strings.Contains(lowered, "adr-"):
		return "adr"
	case strings.Contains(lowered, "runbook"), strings.Contains(lowered, "playbook"), strings.Contains(lowered, "incident"):
		return "runbook"
	default:
		return "doc"
	}
}

func shouldSkipEvidenceDir(path string) bool {
	base := filepath.Base(path)
	if base == ".git" || base == ".sdp" || base == ".beads" || base == "node_modules" || base == "vendor" {
		return true
	}
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/docs/reality")
}

func mergeMemory(existing RepoMemory, scans []repoScan, generatedAt string, sources []ReviewSource, withDocs bool) RepoMemory {
	repoMap := make(map[string]RepoRecord, len(existing.Repos))
	for _, repo := range existing.Repos {
		repoMap[repo.RepoID] = repo
	}
	moduleMap := make(map[string]ModuleSummary, len(existing.ModuleSummaries))
	for _, module := range existing.ModuleSummaries {
		moduleMap[module.ModuleID] = module
	}
	featureMap := make(map[string]FeatureMapping, len(existing.FeatureMappings))
	for _, feature := range existing.FeatureMappings {
		featureMap[feature.FeatureID] = feature
	}
	hotspotMap := make(map[string]HotspotRecord, len(existing.Hotspots))
	for _, hotspot := range existing.Hotspots {
		hotspotMap[hotspot.HotspotID] = hotspot
	}
	ownershipMap := make(map[string]OwnershipZone, len(existing.OwnershipZones))
	for _, zone := range existing.OwnershipZones {
		ownershipMap[zone.ZoneID] = zone
	}
	teamMap := make(map[string]TeamMetadata, len(existing.Teams))
	for _, team := range existing.Teams {
		teamMap[team.TeamID] = team
	}
	sourceMap := make(map[string]ReviewSource, len(existing.Sources))
	for _, source := range existing.Sources {
		sourceMap[source.SourceID] = source
	}

	unresolved := append([]string{}, existing.UnresolvedQuestions...)
	for _, scan := range scans {
		repoMap[scan.Record.RepoID] = scan.Record
		for _, module := range scan.Modules {
			moduleMap[module.ModuleID] = module
		}
		for _, feature := range scan.FeatureMappings {
			featureMap[feature.FeatureID] = feature
		}
		for _, zone := range scan.OwnershipZones {
			ownershipMap[zone.ZoneID] = zone
		}
		for _, team := range scan.Teams {
			teamMap[team.TeamID] = team
		}
		for _, hotspot := range scan.Hotspots {
			hotspotMap[hotspot.HotspotID] = hotspot
		}
		for _, source := range scan.Sources {
			sourceMap[source.SourceID] = source
		}
		unresolved = append(unresolved, scan.UnresolvedQuestions...)
	}
	for _, source := range sources {
		sourceMap[source.SourceID] = source
	}
	if withDocs && len(sources) == 0 {
		unresolved = append(unresolved, "reality-pro: docs mode was requested but no docs/ADR/runbook evidence was ingested")
	}

	memory := RepoMemory{
		SpecVersion:               specVersion,
		GeneratedAt:               generatedAt,
		Repos:                     sortedRepos(repoMap),
		ModuleSummaries:           sortedModules(moduleMap),
		FeatureMappings:           sortedFeatures(featureMap),
		OwnershipZones:            sortedOwnershipZones(ownershipMap),
		Teams:                     sortedTeams(teamMap),
		PreviousValidatedClaimIDs: dedupeStrings(existing.PreviousValidatedClaimIDs),
		UnresolvedQuestions:       dedupeStrings(unresolved),
		Hotspots:                  sortedHotspots(hotspotMap),
		Sources:                   sortedSources(sourceMap),
	}
	return memory
}

func loadExistingMemory(projectRoot string) (RepoMemory, error) {
	path := filepath.Join(projectRoot, ".sdp", "reality", "repo-memory.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RepoMemory{}, nil
		}
		return RepoMemory{}, err
	}
	var memory RepoMemory
	if err := json.Unmarshal(data, &memory); err != nil {
		return RepoMemory{}, fmt.Errorf("parse existing repo memory: %w", err)
	}
	return memory, nil
}

func renderMultiRepoMap(memory RepoMemory, links []repoLink, moduleIndex map[string][]ModuleSummary) string {
	var b strings.Builder
	b.WriteString("# Reality Multi-Repo Map\n\n")
	b.WriteString(fmt.Sprintf("- Generated At: `%s`\n", memory.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Repositories Indexed: `%d`\n", len(memory.Repos)))
	b.WriteString("\n## Repo Roles\n\n")
	b.WriteString("| Repo | Role | Root | Modules |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, repo := range memory.Repos {
		modules := moduleIndex[repo.RepoID]
		moduleNames := make([]string, 0, len(modules))
		for _, module := range modules {
			moduleNames = append(moduleNames, strings.TrimPrefix(module.ModuleID, fmt.Sprintf("module:%s:", repo.RepoID)))
		}
		sort.Strings(moduleNames)
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` |\n", repo.Name, repo.Role, repo.RootPath, strings.Join(moduleNames, ", ")))
	}

	b.WriteString("\n## Ownership Zones\n\n")
	if len(memory.OwnershipZones) == 0 {
		b.WriteString("- none reconstructed yet\n")
	} else {
		b.WriteString("| Repo | Pattern | Owners | Teams | Escalation |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, zone := range memory.OwnershipZones {
			escalation := zone.EscalationTarget
			if escalation == "" {
				escalation = "unassigned"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` |\n", zone.RepoID, zone.Pattern, strings.Join(zone.Owners, ", "), strings.Join(zone.TeamIDs, ", "), escalation))
		}
	}

	b.WriteString("\n## Team Metadata\n\n")
	if len(memory.Teams) == 0 {
		b.WriteString("- none ingested yet\n")
	} else {
		b.WriteString("| Team | Repo | Contact | Escalation | Owns |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, team := range memory.Teams {
			contact := firstNonEmpty(team.Contact, team.Slack, team.Email)
			if contact == "" {
				contact = "n/a"
			}
			escalation := team.EscalationTarget
			if escalation == "" {
				escalation = "unassigned"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` |\n", team.Name, team.RepoID, contact, escalation, strings.Join(team.Owns, ", ")))
		}
	}

	b.WriteString("\n## Boundary Edges\n\n")
	if len(links) == 0 {
		b.WriteString("- single-repo scope; no cross-repo boundaries reconstructed yet\n")
	} else {
		for _, link := range links {
			b.WriteString(fmt.Sprintf("- `%s` %s `%s`\n", link.From, link.Relation, link.To))
		}
	}

	b.WriteString("\n## Evidence Sources\n\n")
	if len(memory.Sources) == 0 {
		b.WriteString("- none\n")
	} else {
		b.WriteString(fmt.Sprintf("- Total Sources: `%d`\n\n", len(memory.Sources)))
		b.WriteString("| Kind | Repo | Count |\n")
		b.WriteString("|---|---|---|\n")
		for _, item := range evidenceCoverage(memory.Sources) {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%d` |\n", item.Kind, item.Repo, item.Count))
		}
		b.WriteString("\n### Sample Sources\n\n")
		b.WriteString("| Kind | Repo | Locator |\n")
		b.WriteString("|---|---|---|\n")
		for _, source := range sampleSources(memory.Sources, 20) {
			repo := source.Repo
			if repo == "" {
				repo = "shared"
			}
			locator := source.Locator
			if source.Path != "" {
				locator = source.Path
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` |\n", source.Kind, repo, locator))
		}
	}

	b.WriteString("\n## Persistent Questions\n\n")
	if len(memory.UnresolvedQuestions) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, question := range memory.UnresolvedQuestions {
			b.WriteString(fmt.Sprintf("- %s\n", question))
		}
	}
	return b.String()
}

func buildRepoLinks(repos []RepoRecord) []repoLink {
	links := make([]repoLink, 0)
	for i := 0; i < len(repos); i++ {
		for j := 0; j < len(repos); j++ {
			if i == j {
				continue
			}
			from := repos[i]
			to := repos[j]
			if isNestedPath(from.RootPath, to.RootPath) {
				links = append(links, repoLink{
					From:     from.RepoID,
					To:       to.RepoID,
					Relation: "contains",
				})
			}
			if from.Role == "service" && to.Role == "protocol" {
				links = append(links, repoLink{
					From:     from.RepoID,
					To:       to.RepoID,
					Relation: "consumes contracts from",
				})
			}
		}
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].From == links[j].From {
			if links[i].To == links[j].To {
				return links[i].Relation < links[j].Relation
			}
			return links[i].To < links[j].To
		}
		return links[i].From < links[j].From
	})
	return dedupeLinks(links)
}

func nestedRepoSet(repoRoot string, allRepoRoots []string) map[string]bool {
	nested := map[string]bool{}
	for _, other := range allRepoRoots {
		if other == repoRoot {
			continue
		}
		if isNestedPath(repoRoot, other) {
			nested[other] = true
		}
	}
	return nested
}

func isNestedPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

func repoID(projectRoot, repoRoot string) string {
	rel, err := filepath.Rel(projectRoot, repoRoot)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "repo:" + sanitizeID(filepath.Base(repoRoot))
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "repo:" + sanitizeID(filepath.Base(projectRoot))
	}
	return "repo:" + sanitizeID(rel)
}

func sanitizeID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/.")
	value = strings.ReplaceAll(value, "/", ":")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "root"
	}
	return value
}

func classifyRole(hasGoMod, hasDeploy, hasSchema, hasPrompts bool, modules map[string]int) string {
	hasServiceFootprint := modules["cmd"] > 0 || modules["internal"] > 0
	switch {
	case hasPrompts:
		return "protocol"
	case hasSchema && !hasServiceFootprint:
		return "protocol"
	case hasDeploy && !hasGoMod:
		return "infra"
	case hasGoMod && hasServiceFootprint:
		return "service"
	default:
		return "mixed"
	}
}

func moduleIDs(modules []ModuleSummary) []string {
	result := make([]string, 0, len(modules))
	for _, module := range modules {
		result = append(result, module.ModuleID)
	}
	sort.Strings(result)
	return result
}

func detectInterfaces(paths []string) []string {
	interfaces := make([]string, 0)
	for _, path := range paths {
		switch {
		case strings.Contains(path, "/api/"):
			interfaces = append(interfaces, "api")
		case strings.Contains(path, "/cmd/"):
			interfaces = append(interfaces, "cli")
		case strings.Contains(path, "handler"):
			interfaces = append(interfaces, "handler")
		}
	}
	return dedupeStrings(interfaces)
}

func confidenceForRepo(sourceCount, testCount int) float64 {
	if sourceCount == 0 {
		return 0.3
	}
	if testCount == 0 {
		return 0.7
	}
	return 0.9
}

func hotspotID(repoID, relPath string) string {
	return fmt.Sprintf("hotspot:%s:%s", repoID, sanitizeID(relPath))
}

func topModule(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) <= 1 {
		return "root"
	}
	return parts[0]
}

func isTestFile(rel string) bool {
	return strings.HasSuffix(rel, "_test.go") ||
		strings.HasSuffix(rel, ".test.js") ||
		strings.HasSuffix(rel, ".test.ts") ||
		strings.HasSuffix(rel, "_spec.py")
}

func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimRight(string(data), "\n"), "\n"))
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func dedupeLinks(values []repoLink) []repoLink {
	seen := map[string]bool{}
	result := make([]repoLink, 0, len(values))
	for _, value := range values {
		key := value.From + "|" + value.Relation + "|" + value.To
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func sortedRepos(items map[string]RepoRecord) []RepoRecord {
	result := make([]RepoRecord, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].RepoID < result[j].RepoID
	})
	return result
}

func sortedModules(items map[string]ModuleSummary) []ModuleSummary {
	result := make([]ModuleSummary, 0, len(items))
	for _, item := range items {
		sort.Strings(item.Paths)
		sort.Strings(item.Interfaces)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ModuleID < result[j].ModuleID
	})
	return result
}

func sortedFeatures(items map[string]FeatureMapping) []FeatureMapping {
	result := make([]FeatureMapping, 0, len(items))
	for _, item := range items {
		sort.Strings(item.RepoIDs)
		sort.Strings(item.ComponentIDs)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].FeatureID < result[j].FeatureID
	})
	return result
}

func sortedOwnershipZones(items map[string]OwnershipZone) []OwnershipZone {
	result := make([]OwnershipZone, 0, len(items))
	for _, item := range items {
		item.Owners = dedupeStrings(item.Owners)
		item.TeamIDs = dedupeStrings(item.TeamIDs)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ZoneID < result[j].ZoneID
	})
	return result
}

func sortedTeams(items map[string]TeamMetadata) []TeamMetadata {
	result := make([]TeamMetadata, 0, len(items))
	for _, item := range items {
		item.Aliases = dedupeStrings(item.Aliases)
		item.Owns = dedupeStrings(item.Owns)
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TeamID < result[j].TeamID
	})
	return result
}

func sortedHotspots(items map[string]HotspotRecord) []HotspotRecord {
	result := make([]HotspotRecord, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].HotspotID < result[j].HotspotID
	})
	return result
}

func sortedSources(items map[string]ReviewSource) []ReviewSource {
	result := make([]ReviewSource, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].SourceID < result[j].SourceID
	})
	return result
}

func evidenceCoverage(sources []ReviewSource) []evidenceCoverageItem {
	counts := map[string]int{}
	for _, source := range sources {
		repo := source.Repo
		if repo == "" {
			repo = "shared"
		}
		key := source.Kind + "|" + repo
		counts[key]++
	}
	result := make([]evidenceCoverageItem, 0, len(counts))
	for key, count := range counts {
		parts := strings.SplitN(key, "|", 2)
		result = append(result, evidenceCoverageItem{
			Kind:  parts[0],
			Repo:  parts[1],
			Count: count,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind == result[j].Kind {
			return result[i].Repo < result[j].Repo
		}
		return result[i].Kind < result[j].Kind
	})
	return result
}

func sampleSources(sources []ReviewSource, limit int) []ReviewSource {
	if limit <= 0 || len(sources) <= limit {
		return append([]ReviewSource{}, sources...)
	}
	return append([]ReviewSource{}, sources[:limit]...)
}

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeText(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body = strings.TrimRight(body, "\n") + "\n"
	return os.WriteFile(path, []byte(body), 0o644)
}
