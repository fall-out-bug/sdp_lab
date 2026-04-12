package workstream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type FeatureMode string

const (
	FeatureModeLegacy       FeatureMode = "legacy"
	FeatureModeNormalized   FeatureMode = "normalized"
	FeatureModeMixedInvalid FeatureMode = "mixed_invalid"
)

type PolicyVersions struct {
	Normalization      string `json:"normalization"`
	DispatchResolution string `json:"dispatch_resolution"`
	AggregateStatus    string `json:"aggregate_status"`
}

type CompileOptions struct {
	PolicyVersions PolicyVersions
}

func DefaultCompileOptions() CompileOptions {
	return CompileOptions{
		PolicyVersions: PolicyVersions{
			Normalization:      "v1",
			DispatchResolution: "v1",
			AggregateStatus:    "shadow-v1",
		},
	}
}

type WorkgraphLock struct {
	SchemaVersion    int            `json:"schema_version"`
	SourceInputsHash string         `json:"source_inputs_hash"`
	PolicyVersions   PolicyVersions `json:"policy_versions"`
	Features         []FeatureLock  `json:"features"`
}

type FeatureLock struct {
	FeatureID   string           `json:"feature_id"`
	Mode        FeatureMode      `json:"mode"`
	Workstreams []WorkstreamLock `json:"workstreams"`
}

type WorkstreamLock struct {
	WSID                     string   `json:"ws_id"`
	WSKind                   string   `json:"ws_kind"`
	ParentWSID               *string  `json:"parent_ws_id"`
	Children                 []string `json:"children"`
	LifecycleState           string   `json:"lifecycle_state"`
	DeclaredStatus           string   `json:"declared_status"`
	BoundPrimaryIssueID      string   `json:"bound_primary_issue_id,omitempty"`
	FindingIssueIDs          []string `json:"finding_issue_ids,omitempty"`
	HistoricalIssueIDs       []string `json:"historical_issue_ids,omitempty"`
	DerivedStatus            string   `json:"derived_status,omitempty"`
	AggregateFindingIssueIDs []string `json:"aggregate_finding_issue_ids,omitempty"`
}

type ExecutionTarget struct {
	Lock       WorkgraphLock
	Feature    FeatureLock
	Workstream WorkstreamLock
}

type beadsRoles struct {
	Primary    []string
	Finding    []string
	Historical []string
}

type atomicDoc struct {
	Path                      string
	File                      string
	Frontmatter               map[string]string
	WSID                      string
	FeatureID                 string
	Status                    string
	WSKind                    string
	ParentWSID                string
	DispatchLifecycle         string
	HasWSKindField            bool
	HasParentWSIDField        bool
	HasDispatchLifecycleField bool
	HasBeadsSection           bool
	StrictBeadsValid          bool
	StrictBeadsUsed           bool
	Beads                     beadsRoles
}

type compiledWorkgraph struct {
	Lock            WorkgraphLock
	NormalizedPaths []string
}

var strictRolePattern = regexp.MustCompile(`^(primary|finding|historical):\s*(sdplab-[a-z0-9]+)$`)

func CompileWorkgraphLock(projectRoot string, opts CompileOptions) (WorkgraphLock, ValidationReport, error) {
	compiled, report, err := compileWorkgraph(projectRoot, opts)
	return compiled.Lock, report, err
}

func WriteWorkgraphLock(projectRoot string, lock WorkgraphLock) error {
	lockPath := filepath.Join(projectRoot, ".sdp", "workgraph.lock.json")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}
	payload, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workgraph lock: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(lockPath, payload, 0o644); err != nil {
		return fmt.Errorf("write workgraph lock: %w", err)
	}
	return nil
}

func LoadWorkgraphLock(projectRoot string) (WorkgraphLock, error) {
	lockPath := filepath.Join(projectRoot, ".sdp", "workgraph.lock.json")
	payload, err := os.ReadFile(lockPath)
	if err != nil {
		return WorkgraphLock{}, fmt.Errorf("read workgraph lock: %w", err)
	}
	var lock WorkgraphLock
	if err := json.Unmarshal(payload, &lock); err != nil {
		return WorkgraphLock{}, fmt.Errorf("parse workgraph lock: %w", err)
	}
	return lock, nil
}

func ReadFreshWorkgraphLock(projectRoot string, opts CompileOptions) (WorkgraphLock, error) {
	stored, err := LoadWorkgraphLock(projectRoot)
	if err != nil {
		return WorkgraphLock{}, err
	}

	compiled, report, err := compileWorkgraph(projectRoot, opts)
	if err != nil {
		return WorkgraphLock{}, err
	}
	if report.HasErrors() {
		return WorkgraphLock{}, fmt.Errorf("compile workgraph lock: %s", summarizeValidationIssues(report.Issues))
	}
	if stored.SourceInputsHash != compiled.Lock.SourceInputsHash {
		return WorkgraphLock{}, fmt.Errorf("workgraph lock is stale: stored %s, current %s", stored.SourceInputsHash, compiled.Lock.SourceInputsHash)
	}
	if err := ensureNormalizedFilesClean(projectRoot, compiled.NormalizedPaths); err != nil {
		return WorkgraphLock{}, err
	}
	return compiled.Lock, nil
}

func ResolveExecutableLeaf(projectRoot, featureID, wsID string, opts CompileOptions) (ExecutionTarget, error) {
	lock, err := ReadFreshWorkgraphLock(projectRoot, opts)
	if err != nil {
		return ExecutionTarget{}, err
	}

	for _, feature := range lock.Features {
		if feature.FeatureID != featureID {
			continue
		}
		for _, ws := range feature.Workstreams {
			if ws.WSID != wsID {
				continue
			}
			if ws.WSKind != "leaf" {
				return ExecutionTarget{}, fmt.Errorf("workstream %s is %s, not executable leaf", wsID, ws.WSKind)
			}
			if ws.LifecycleState != "active" {
				return ExecutionTarget{}, fmt.Errorf("workstream %s lifecycle is %s, want active", wsID, ws.LifecycleState)
			}
			if ws.BoundPrimaryIssueID == "" {
				return ExecutionTarget{}, fmt.Errorf("workstream %s has no bound primary issue", wsID)
			}
			return ExecutionTarget{Lock: lock, Feature: feature, Workstream: ws}, nil
		}
		return ExecutionTarget{}, fmt.Errorf("workstream %s not found under feature %s", wsID, featureID)
	}

	return ExecutionTarget{}, fmt.Errorf("feature %s is not normalized in the current workgraph lock", featureID)
}

func compileWorkgraph(projectRoot string, opts CompileOptions) (compiledWorkgraph, ValidationReport, error) {
	if opts.PolicyVersions.Normalization == "" {
		opts = DefaultCompileOptions()
	}

	docs, err := loadAtomicDocs(projectRoot)
	if err != nil {
		return compiledWorkgraph{}, ValidationReport{}, err
	}

	features := make(map[string][]*atomicDoc)
	for _, doc := range docs {
		if doc.FeatureID == "" {
			continue
		}
		features[doc.FeatureID] = append(features[doc.FeatureID], doc)
	}

	report := ValidationReport{Issues: []ValidationIssue{}}
	lock := WorkgraphLock{
		SchemaVersion:  1,
		PolicyVersions: opts.PolicyVersions,
	}
	var normalizedPaths []string

	featureIDs := make([]string, 0, len(features))
	for featureID := range features {
		featureIDs = append(featureIDs, featureID)
	}
	sort.Strings(featureIDs)

	for _, featureID := range featureIDs {
		docsForFeature := features[featureID]
		slices.SortFunc(docsForFeature, func(a, b *atomicDoc) int {
			return strings.Compare(a.WSID, b.WSID)
		})

		mode, featureIssues, featureLock, featurePaths := compileFeature(featureID, docsForFeature, opts.PolicyVersions)
		report.Issues = append(report.Issues, featureIssues...)
		if mode == FeatureModeNormalized {
			lock.Features = append(lock.Features, featureLock)
			normalizedPaths = append(normalizedPaths, featurePaths...)
		}
	}

	lock.SourceInputsHash = computeSourceInputsHash(lock)
	return compiledWorkgraph{Lock: lock, NormalizedPaths: normalizedPaths}, report, nil
}

func compileFeature(featureID string, docs []*atomicDoc, policies PolicyVersions) (FeatureMode, []ValidationIssue, FeatureLock, []string) {
	issues := make([]ValidationIssue, 0)
	active := make([]*atomicDoc, 0)
	for _, doc := range docs {
		if doc.Status != "archived" {
			active = append(active, doc)
		}
	}

	hasNormalizedSignals := false
	for _, doc := range active {
		if usesAtomicitySchema(doc) {
			hasNormalizedSignals = true
			break
		}
	}
	if !hasNormalizedSignals {
		return FeatureModeLegacy, issues, FeatureLock{}, nil
	}

	mode := FeatureModeNormalized
	normalizedDocs := make([]*atomicDoc, 0, len(docs))
	for _, doc := range docs {
		include := usesAtomicitySchema(doc) || doc.Status != "archived"
		if !include {
			continue
		}
		normalizedDocs = append(normalizedDocs, doc)
		if doc.Status == "archived" && !usesAtomicitySchema(doc) {
			continue
		}
		if !doc.HasWSKindField {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: active workstream %s lacks ws_kind", featureID, doc.WSID)})
		}
		if !doc.HasDispatchLifecycleField {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: active workstream %s lacks dispatch_lifecycle", featureID, doc.WSID)})
		}
		if !doc.HasBeadsSection {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: active workstream %s lacks strict ## Beads section", featureID, doc.WSID)})
		} else if !doc.StrictBeadsValid {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: strict ## Beads parsing failed for %s", featureID, doc.WSID)})
		}
	}
	if mode == FeatureModeMixedInvalid {
		return mode, issues, FeatureLock{}, nil
	}

	docByID := make(map[string]*atomicDoc, len(normalizedDocs))
	childrenByParent := make(map[string][]string)
	normalizedPaths := make([]string, 0, len(normalizedDocs))
	for _, doc := range normalizedDocs {
		docByID[doc.WSID] = doc
		normalizedPaths = append(normalizedPaths, doc.Path)
		validateAtomicityDoc(featureID, doc, &mode, &issues)
	}

	for _, doc := range normalizedDocs {
		if doc.ParentWSID == "" {
			continue
		}
		parent, ok := docByID[doc.ParentWSID]
		if !ok {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: %s references missing parent %s", featureID, doc.WSID, doc.ParentWSID)})
			continue
		}
		if parent.FeatureID != doc.FeatureID {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: %s parent %s belongs to another feature", featureID, doc.WSID, doc.ParentWSID)})
		}
		if parent.WSKind != "aggregate" {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: leaf %s parent %s is not aggregate", featureID, doc.WSID, doc.ParentWSID)})
		}
		if parent.ParentWSID != "" {
			mode = FeatureModeMixedInvalid
			issues = append(issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: aggregate depth exceeds 1 at %s", featureID, doc.WSID)})
		}
		childrenByParent[doc.ParentWSID] = append(childrenByParent[doc.ParentWSID], doc.WSID)
	}

	for parent := range childrenByParent {
		sort.Strings(childrenByParent[parent])
	}

	if mode == FeatureModeMixedInvalid {
		return mode, issues, FeatureLock{}, nil
	}

	featureLock := FeatureLock{
		FeatureID: featureID,
		Mode:      FeatureModeNormalized,
	}
	for _, doc := range normalizedDocs {
		lockWS := WorkstreamLock{
			WSID:               doc.WSID,
			WSKind:             doc.WSKind,
			Children:           append([]string(nil), childrenByParent[doc.WSID]...),
			LifecycleState:     doc.DispatchLifecycle,
			DeclaredStatus:     doc.Status,
			HistoricalIssueIDs: append([]string(nil), doc.Beads.Historical...),
		}
		if doc.ParentWSID != "" {
			parent := doc.ParentWSID
			lockWS.ParentWSID = &parent
		}
		switch doc.WSKind {
		case "leaf":
			if len(doc.Beads.Primary) == 1 {
				lockWS.BoundPrimaryIssueID = doc.Beads.Primary[0]
			}
			lockWS.FindingIssueIDs = append([]string(nil), doc.Beads.Finding...)
		case "aggregate":
			lockWS.AggregateFindingIssueIDs = append([]string(nil), doc.Beads.Finding...)
			lockWS.DerivedStatus = deriveAggregateStatus(doc, normalizedDocs, childrenByParent)
			if lockWS.DerivedStatus != doc.Status {
				severity := "warning"
				if policies.AggregateStatus == "enforced-v1" {
					severity = "error"
					mode = FeatureModeMixedInvalid
				}
				issues = append(issues, ValidationIssue{Severity: severity, File: doc.File, Message: fmt.Sprintf("aggregate %s declared_status=%s derived_status=%s under policy %s", doc.WSID, doc.Status, lockWS.DerivedStatus, policies.AggregateStatus)})
			}
		}
		featureLock.Workstreams = append(featureLock.Workstreams, lockWS)
	}

	if mode == FeatureModeMixedInvalid {
		return mode, issues, FeatureLock{}, nil
	}
	return mode, issues, featureLock, normalizedPaths
}

func validateAtomicityDoc(featureID string, doc *atomicDoc, mode *FeatureMode, issues *[]ValidationIssue) {
	validStatuses := map[string]bool{"backlog": true, "open": true, "blocked": true, "done": true, "archived": true}
	if !validStatuses[doc.Status] {
		*mode = FeatureModeMixedInvalid
		*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: %s has invalid status %q", featureID, doc.WSID, doc.Status)})
	}

	if doc.WSKind != "leaf" && doc.WSKind != "aggregate" {
		*mode = FeatureModeMixedInvalid
		*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: %s has invalid ws_kind %q", featureID, doc.WSID, doc.WSKind)})
	}
	validLifecycle := map[string]bool{"active": true, "frozen": true, "archived": true}
	if !validLifecycle[doc.DispatchLifecycle] {
		*mode = FeatureModeMixedInvalid
		*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: %s has invalid dispatch_lifecycle %q", featureID, doc.WSID, doc.DispatchLifecycle)})
	}
	if doc.Status == "archived" && doc.DispatchLifecycle != "archived" {
		*mode = FeatureModeMixedInvalid
		*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: archived workstream %s must set dispatch_lifecycle=archived", featureID, doc.WSID)})
	}
	if doc.Status != "archived" && doc.DispatchLifecycle == "archived" {
		*mode = FeatureModeMixedInvalid
		*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: non-archived workstream %s cannot set dispatch_lifecycle=archived", featureID, doc.WSID)})
	}
	if doc.WSKind == "aggregate" {
		if doc.ParentWSID != "" {
			*mode = FeatureModeMixedInvalid
			*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: aggregate %s must have parent_ws_id null", featureID, doc.WSID)})
		}
		if len(doc.Beads.Primary) > 0 {
			*mode = FeatureModeMixedInvalid
			*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: aggregate %s cannot bind primary issue", featureID, doc.WSID)})
		}
	}
	if doc.WSKind == "leaf" {
		if len(doc.Beads.Primary) > 1 {
			*mode = FeatureModeMixedInvalid
			*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: leaf %s has more than one primary issue", featureID, doc.WSID)})
		}
		if doc.Status == "open" && len(doc.Beads.Primary) != 1 {
			*mode = FeatureModeMixedInvalid
			*issues = append(*issues, ValidationIssue{Severity: "error", File: doc.File, Message: fmt.Sprintf("feature %s is mixed_invalid: open leaf %s must bind exactly one primary issue", featureID, doc.WSID)})
		}
	}
}

func deriveAggregateStatus(doc *atomicDoc, docs []*atomicDoc, childrenByParent map[string][]string) string {
	if doc.Status == "archived" {
		return "archived"
	}
	childrenIDs := childrenByParent[doc.WSID]
	if len(childrenIDs) == 0 {
		return doc.Status
	}
	docByID := make(map[string]*atomicDoc, len(docs))
	for _, item := range docs {
		docByID[item.WSID] = item
	}

	allDoneOrArchived := true
	allBacklog := true
	allNonTerminalBlocked := true
	anyProgress := false
	hasNonTerminal := false

	for _, childID := range childrenIDs {
		child := docByID[childID]
		if child == nil {
			continue
		}
		switch child.Status {
		case "done", "archived":
		default:
			allDoneOrArchived = false
		}
		if child.Status != "backlog" {
			allBacklog = false
		}
		if child.Status != "done" && child.Status != "archived" {
			hasNonTerminal = true
			if child.Status != "blocked" {
				allNonTerminalBlocked = false
			}
		}
		if child.Status == "open" || child.Status == "blocked" || child.Status == "done" {
			anyProgress = true
		}
	}

	if allDoneOrArchived && len(doc.Beads.Finding) == 0 {
		return "done"
	}
	if len(doc.Beads.Finding) > 0 {
		return "blocked"
	}
	if hasNonTerminal && allNonTerminalBlocked {
		return "blocked"
	}
	if anyProgress {
		return "open"
	}
	if allBacklog {
		return "backlog"
	}
	return doc.Status
}

func usesAtomicitySchema(doc *atomicDoc) bool {
	return doc.HasWSKindField || doc.HasParentWSIDField || doc.HasDispatchLifecycleField || doc.StrictBeadsUsed
}

func loadAtomicDocs(projectRoot string) ([]*atomicDoc, error) {
	backlogDir := filepath.Join(projectRoot, "docs", "workstreams", "backlog")
	entries, err := os.ReadDir(backlogDir)
	if err != nil {
		return nil, fmt.Errorf("read backlog directory: %w", err)
	}

	docs := make([]*atomicDoc, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(backlogDir, entry.Name())
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read workstream file %s: %w", path, err)
		}
		content := string(contentBytes)
		fm := parseFrontmatter(content)
		section, hasBeads := extractSection(content, "Beads")
		beads, strictValid, strictUsed := parseStrictBeads(section, hasBeads)
		parentWSID := normalizeNullableString(fm["parent_ws_id"])
		doc := &atomicDoc{
			Path:                      path,
			File:                      rel(projectRoot, path),
			Frontmatter:               fm,
			WSID:                      fm["ws_id"],
			FeatureID:                 fm["feature_id"],
			Status:                    fm["status"],
			WSKind:                    fm["ws_kind"],
			ParentWSID:                parentWSID,
			DispatchLifecycle:         fm["dispatch_lifecycle"],
			HasWSKindField:            hasFrontmatterKey(fm, "ws_kind"),
			HasParentWSIDField:        hasFrontmatterKey(fm, "parent_ws_id"),
			HasDispatchLifecycleField: hasFrontmatterKey(fm, "dispatch_lifecycle"),
			HasBeadsSection:           hasBeads,
			StrictBeadsValid:          strictValid,
			StrictBeadsUsed:           strictUsed,
			Beads:                     beads,
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func parseStrictBeads(section string, hasSection bool) (beadsRoles, bool, bool) {
	if !hasSection {
		return beadsRoles{}, false, false
	}
	trimmed := strings.TrimSpace(section)
	if trimmed == "" {
		return beadsRoles{}, true, false
	}

	var out beadsRoles
	used := false
	for _, raw := range strings.Split(section, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "- ") {
			return beadsRoles{}, false, used
		}
		item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		m := strictRolePattern.FindStringSubmatch(item)
		if len(m) != 3 {
			return beadsRoles{}, false, used
		}
		used = true
		role, id := m[1], m[2]
		switch role {
		case "primary":
			out.Primary = append(out.Primary, id)
		case "finding":
			out.Finding = append(out.Finding, id)
		case "historical":
			out.Historical = append(out.Historical, id)
		}
	}

	sort.Strings(out.Primary)
	sort.Strings(out.Finding)
	sort.Strings(out.Historical)
	return out, true, used
}

func computeSourceInputsHash(lock WorkgraphLock) string {
	payload := canonicalHashEnvelope{
		SchemaVersion: lock.SchemaVersion,
		PolicyVersions: canonicalPolicyVersions{
			AggregateStatus:    lock.PolicyVersions.AggregateStatus,
			DispatchResolution: lock.PolicyVersions.DispatchResolution,
			Normalization:      lock.PolicyVersions.Normalization,
		},
		Features: make([]canonicalHashFeature, 0, len(lock.Features)),
	}

	for _, feature := range lock.Features {
		item := canonicalHashFeature{
			FeatureID:   feature.FeatureID,
			Workstreams: make([]canonicalHashWorkstream, 0, len(feature.Workstreams)),
		}
		for _, ws := range feature.Workstreams {
			parent := ""
			if ws.ParentWSID != nil {
				parent = *ws.ParentWSID
			}
			beads := canonicalHashBeads{
				Finding:    append([]string(nil), ws.FindingIssueIDs...),
				Historical: append([]string(nil), ws.HistoricalIssueIDs...),
			}
			if ws.BoundPrimaryIssueID != "" {
				beads.Primary = []string{ws.BoundPrimaryIssueID}
			}
			item.Workstreams = append(item.Workstreams, canonicalHashWorkstream{
				WSID: ws.WSID,
				Frontmatter: canonicalHashFrontmatter{
					DispatchLifecycle: ws.LifecycleState,
					FeatureID:         feature.FeatureID,
					ParentWSID:        parent,
					Status:            ws.DeclaredStatus,
					WSID:              ws.WSID,
					WSKind:            ws.WSKind,
				},
				Beads: beads,
			})
		}
		payload.Features = append(payload.Features, item)
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshal canonical workgraph hash payload: %v", err))
	}
	sum := sha256.Sum256(buf)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type canonicalHashEnvelope struct {
	SchemaVersion  int                     `json:"schema_version"`
	PolicyVersions canonicalPolicyVersions `json:"policy_versions"`
	Features       []canonicalHashFeature  `json:"features"`
}

type canonicalPolicyVersions struct {
	AggregateStatus    string `json:"aggregate_status"`
	DispatchResolution string `json:"dispatch_resolution"`
	Normalization      string `json:"normalization"`
}

type canonicalHashFeature struct {
	FeatureID   string                    `json:"feature_id"`
	Workstreams []canonicalHashWorkstream `json:"workstreams"`
}

type canonicalHashWorkstream struct {
	WSID        string                   `json:"ws_id"`
	Frontmatter canonicalHashFrontmatter `json:"frontmatter"`
	Beads       canonicalHashBeads       `json:"beads"`
}

type canonicalHashFrontmatter struct {
	DispatchLifecycle string `json:"dispatch_lifecycle"`
	FeatureID         string `json:"feature_id"`
	ParentWSID        string `json:"parent_ws_id"`
	Status            string `json:"status"`
	WSID              string `json:"ws_id"`
	WSKind            string `json:"ws_kind"`
}

type canonicalHashBeads struct {
	Finding    []string `json:"finding"`
	Historical []string `json:"historical"`
	Primary    []string `json:"primary"`
}

func ensureNormalizedFilesClean(projectRoot string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := []string{"-C", projectRoot, "status", "--porcelain", "--"}
	args = append(args, paths...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status for normalized workstreams failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) > 0 {
		return fmt.Errorf("normalized workstream sources are dirty:\n%s", strings.TrimSpace(string(trimmed)))
	}
	return nil
}

func summarizeValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return "no validation issues"
	}
	limit := 3
	if len(issues) < limit {
		limit = len(issues)
	}
	parts := make([]string, 0, limit)
	for i, issue := range issues {
		if i == limit {
			break
		}
		parts = append(parts, issue.Message)
	}
	if len(issues) > len(parts) {
		parts = append(parts, fmt.Sprintf("%d more", len(issues)-len(parts)))
	}
	return strings.Join(parts, "; ")
}

func hasFrontmatterKey(fm map[string]string, key string) bool {
	_, ok := fm[key]
	return ok
}

func normalizeNullableString(s string) string {
	s = strings.TrimSpace(strings.Trim(s, `"`))
	if s == "" || strings.EqualFold(s, "null") {
		return ""
	}
	return s
}
