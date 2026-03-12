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
	Now         func() time.Time
}

type Result struct {
	RepoMemoryPath   string
	MultiRepoMapPath string
	RepoCount        int
}

type RepoMemory struct {
	SpecVersion               string           `json:"spec_version"`
	GeneratedAt               string           `json:"generated_at"`
	Repos                     []RepoRecord     `json:"repos"`
	ModuleSummaries           []ModuleSummary  `json:"module_summaries"`
	FeatureMappings           []FeatureMapping `json:"feature_mappings,omitempty"`
	PreviousValidatedClaimIDs []string         `json:"previous_validated_claim_ids,omitempty"`
	UnresolvedQuestions       []string         `json:"unresolved_questions"`
	Hotspots                  []HotspotRecord  `json:"hotspots,omitempty"`
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

type repoScan struct {
	Record              RepoRecord
	Modules             []ModuleSummary
	FeatureMappings     []FeatureMapping
	Hotspots            []HotspotRecord
	UnresolvedQuestions []string
	ModuleNames         []string
}

type repoLink struct {
	From     string
	To       string
	Relation string
}

// Ingest builds or refreshes reality-pro persistent memory for one repo or an explicit reposet.
func Ingest(opts Options) (Result, error) {
	projectRoot, repoRoots, now, err := normalizeOptions(opts)
	if err != nil {
		return Result{}, err
	}

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

	generatedAt := now.UTC().Format(time.RFC3339)
	updated := mergeMemory(existing, scans, generatedAt)
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

	return repoScan{
		Record:              record,
		Modules:             moduleSummaries,
		FeatureMappings:     featureMappings,
		Hotspots:            hotspots,
		UnresolvedQuestions: dedupeStrings(unresolved),
		ModuleNames:         moduleNames,
	}, nil
}

func mergeMemory(existing RepoMemory, scans []repoScan, generatedAt string) RepoMemory {
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

	unresolved := append([]string{}, existing.UnresolvedQuestions...)
	for _, scan := range scans {
		repoMap[scan.Record.RepoID] = scan.Record
		for _, module := range scan.Modules {
			moduleMap[module.ModuleID] = module
		}
		for _, feature := range scan.FeatureMappings {
			featureMap[feature.FeatureID] = feature
		}
		for _, hotspot := range scan.Hotspots {
			hotspotMap[hotspot.HotspotID] = hotspot
		}
		unresolved = append(unresolved, scan.UnresolvedQuestions...)
	}

	memory := RepoMemory{
		SpecVersion:               specVersion,
		GeneratedAt:               generatedAt,
		Repos:                     sortedRepos(repoMap),
		ModuleSummaries:           sortedModules(moduleMap),
		FeatureMappings:           sortedFeatures(featureMap),
		PreviousValidatedClaimIDs: dedupeStrings(existing.PreviousValidatedClaimIDs),
		UnresolvedQuestions:       dedupeStrings(unresolved),
		Hotspots:                  sortedHotspots(hotspotMap),
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

	b.WriteString("\n## Boundary Edges\n\n")
	if len(links) == 0 {
		b.WriteString("- single-repo scope; no cross-repo boundaries reconstructed yet\n")
	} else {
		for _, link := range links {
			b.WriteString(fmt.Sprintf("- `%s` %s `%s`\n", link.From, link.Relation, link.To))
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
