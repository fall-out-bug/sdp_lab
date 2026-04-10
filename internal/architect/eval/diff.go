package eval

import (
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// DiffAction classifies the type of difference found.
type DiffAction string

const (
	DiffAdded    DiffAction = "added"
	DiffRemoved  DiffAction = "removed"
	DiffModified DiffAction = "modified"
)

// DiffEntry represents a single difference between two ProfileFragments.
type DiffEntry struct {
	// Field is the top-level field name (e.g. "languages", "dependencies").
	Field string `json:"field"`

	// Action is the type of change.
	Action DiffAction `json:"action"`

	// Key is the specific item that differs (e.g. a language name, container name).
	Key string `json:"key"`

	// Expected is the expected value (empty for "added" action).
	Expected string `json:"expected,omitempty"`

	// Actual is the actual value (empty for "removed" action).
	Actual string `json:"actual,omitempty"`
}

// FragmentDiff holds all differences between two ProfileFragments.
type FragmentDiff struct {
	Entries []DiffEntry `json:"entries"`
}

// HasDiffs returns true if any differences were found.
func (d *FragmentDiff) HasDiffs() bool {
	return len(d.Entries) > 0
}

// Summary returns a short summary of diff counts by action.
func (d *FragmentDiff) Summary() string {
	added, removed, modified := 0, 0, 0
	for _, e := range d.Entries {
		switch e.Action {
		case DiffAdded:
			added++
		case DiffRemoved:
			removed++
		case DiffModified:
			modified++
		}
	}
	return fmt.Sprintf("%d added, %d removed, %d modified", added, removed, modified)
}

// DiffFragments compares two ProfileFragments and returns a structural diff.
// It inspects each field and records added/removed/modified entries.
func DiffFragments(expected, actual *architect.ProfileFragment) *FragmentDiff {
	diff := &FragmentDiff{}

	diffLanguages(diff, expected.Languages, actual.Languages)
	diffDependencies(diff, expected.Dependencies, actual.Dependencies)
	diffImportGraph(diff, expected.ImportGraph, actual.ImportGraph)
	diffInfra(diff, expected.Infra, actual.Infra)
	diffFileTree(diff, expected.FileTree, actual.FileTree)
	diffSpecs(diff, expected.Specs, actual.Specs)
	diffSQL(diff, expected.SQLAnalysis, actual.SQLAnalysis)
	diffGenerated(diff, expected.Generated, actual.Generated)

	return diff
}

// --- Per-field diff functions ---

func diffLanguages(diff *FragmentDiff, expected, actual []architect.LanguageInfo) {
	expectedMap := make(map[string]architect.LanguageInfo)
	for _, l := range expected {
		expectedMap[l.Primary] = l
	}
	actualMap := make(map[string]architect.LanguageInfo)
	for _, l := range actual {
		actualMap[l.Primary] = l
	}

	// Check for removed and modified
	for name, exp := range expectedMap {
		act, ok := actualMap[name]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "languages",
				Action:   DiffRemoved,
				Key:      name,
				Expected: fmt.Sprintf("primary=%s", name),
			})
			continue
		}
		// Check for modification (distribution differs)
		if !distributionEqual(exp.Distribution, act.Distribution) {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "languages",
				Action:   DiffModified,
				Key:      name,
				Expected: fmt.Sprintf("all=%v dist=%v", exp.All, exp.Distribution),
				Actual:   fmt.Sprintf("all=%v dist=%v", act.All, act.Distribution),
			})
		}
	}

	// Check for added
	for name := range actualMap {
		if _, ok := expectedMap[name]; !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "languages",
				Action: DiffAdded,
				Key:    name,
				Actual: fmt.Sprintf("primary=%s", name),
			})
		}
	}
}

func diffDependencies(diff *FragmentDiff, expected, actual []architect.DependencyInfo) {
	expectedMap := make(map[string]architect.DependencyInfo)
	for _, d := range expected {
		key := depKey(d)
		expectedMap[key] = d
	}
	actualMap := make(map[string]architect.DependencyInfo)
	for _, d := range actual {
		key := depKey(d)
		actualMap[key] = d
	}

	for key, exp := range expectedMap {
		act, ok := actualMap[key]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "dependencies",
				Action:   DiffRemoved,
				Key:      key,
				Expected: fmt.Sprintf("file=%s lang=%s deps=%d", exp.File, exp.Language, exp.DepCount),
			})
			continue
		}
		if exp.DepCount != act.DepCount || !stringSlicesEqual(exp.Signals, act.Signals) {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "dependencies",
				Action:   DiffModified,
				Key:      key,
				Expected: fmt.Sprintf("deps=%d signals=%v", exp.DepCount, exp.Signals),
				Actual:   fmt.Sprintf("deps=%d signals=%v", act.DepCount, act.Signals),
			})
		}
	}

	for key := range actualMap {
		if _, ok := expectedMap[key]; !ok {
			act := actualMap[key]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "dependencies",
				Action: DiffAdded,
				Key:    key,
				Actual: fmt.Sprintf("file=%s lang=%s deps=%d", act.File, act.Language, act.DepCount),
			})
		}
	}
}

func diffImportGraph(diff *FragmentDiff, expected, actual *architect.ImportGraph) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:  "import_graph",
			Action: DiffAdded,
			Key:    "graph",
			Actual: fmt.Sprintf("method=%s nodes=%d edges=%d", actual.ExtractionMethod, actual.Nodes, actual.Edges),
		})
		return
	}
	if actual == nil {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "import_graph",
			Action:   DiffRemoved,
			Key:      "graph",
			Expected: fmt.Sprintf("method=%s nodes=%d edges=%d", expected.ExtractionMethod, expected.Nodes, expected.Edges),
		})
		return
	}

	// Compare node/edge counts
	if expected.Nodes != actual.Nodes {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "import_graph",
			Action:   DiffModified,
			Key:      "nodes",
			Expected: fmt.Sprintf("%d", expected.Nodes),
			Actual:   fmt.Sprintf("%d", actual.Nodes),
		})
	}
	if expected.Edges != actual.Edges {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "import_graph",
			Action:   DiffModified,
			Key:      "edges",
			Expected: fmt.Sprintf("%d", expected.Edges),
			Actual:   fmt.Sprintf("%d", actual.Edges),
		})
	}

	// Compare clusters
	diffClusters(diff, expected.Clusters, actual.Clusters)

	// Compare circular dependencies
	diffCircularDeps(diff, expected.CircularDependencies, actual.CircularDependencies)
}

func diffClusters(diff *FragmentDiff, expected, actual []architect.ImportCluster) {
	expectedMap := make(map[string]architect.ImportCluster)
	for _, c := range expected {
		expectedMap[c.ID] = c
	}
	actualMap := make(map[string]architect.ImportCluster)
	for _, c := range actual {
		actualMap[c.ID] = c
	}

	for id, exp := range expectedMap {
		act, ok := actualMap[id]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "import_graph.clusters",
				Action:   DiffRemoved,
				Key:      id,
				Expected: fmt.Sprintf("packages=%v internal=%d external=%d", exp.Packages, exp.InternalEdges, exp.ExternalEdges),
			})
			continue
		}
		if exp.InternalEdges != act.InternalEdges || exp.ExternalEdges != act.ExternalEdges ||
			!stringSlicesEqual(exp.Packages, act.Packages) {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "import_graph.clusters",
				Action:   DiffModified,
				Key:      id,
				Expected: fmt.Sprintf("packages=%v internal=%d external=%d", exp.Packages, exp.InternalEdges, exp.ExternalEdges),
				Actual:   fmt.Sprintf("packages=%v internal=%d external=%d", act.Packages, act.InternalEdges, act.ExternalEdges),
			})
		}
	}

	for id := range actualMap {
		if _, ok := expectedMap[id]; !ok {
			act := actualMap[id]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "import_graph.clusters",
				Action: DiffAdded,
				Key:    id,
				Actual: fmt.Sprintf("packages=%v internal=%d external=%d", act.Packages, act.InternalEdges, act.ExternalEdges),
			})
		}
	}
}

func diffCircularDeps(diff *FragmentDiff, expected, actual []architect.CircularDep) {
	expectedSet := make(map[string]bool)
	for _, cd := range expected {
		key := fmt.Sprintf("%s<->%s", cd.A, cd.B)
		expectedSet[key] = true
	}
	actualSet := make(map[string]bool)
	for _, cd := range actual {
		key := fmt.Sprintf("%s<->%s", cd.A, cd.B)
		actualSet[key] = true
	}

	for key := range expectedSet {
		if !actualSet[key] {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "import_graph.circular_deps",
				Action:   DiffRemoved,
				Key:      key,
				Expected: key,
			})
		}
	}
	for key := range actualSet {
		if !expectedSet[key] {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "import_graph.circular_deps",
				Action: DiffAdded,
				Key:    key,
				Actual: key,
			})
		}
	}
}

func diffInfra(diff *FragmentDiff, expected, actual *architect.InfraInfo) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		for _, c := range actual.Containers {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "infra.containers",
				Action: DiffAdded,
				Key:    c.Name,
				Actual: fmt.Sprintf("type=%s image=%s source=%s", c.Type, c.Image, c.Source),
			})
		}
		return
	}
	if actual == nil {
		for _, c := range expected.Containers {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "infra.containers",
				Action:   DiffRemoved,
				Key:      c.Name,
				Expected: fmt.Sprintf("type=%s image=%s source=%s", c.Type, c.Image, c.Source),
			})
		}
		return
	}

	// Compare containers
	expectedContainers := make(map[string]architect.ContainerInfo)
	for _, c := range expected.Containers {
		expectedContainers[c.Name] = c
	}
	actualContainers := make(map[string]architect.ContainerInfo)
	for _, c := range actual.Containers {
		actualContainers[c.Name] = c
	}

	for name, exp := range expectedContainers {
		act, ok := actualContainers[name]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "infra.containers",
				Action:   DiffRemoved,
				Key:      name,
				Expected: fmt.Sprintf("type=%s image=%s", exp.Type, exp.Image),
			})
			continue
		}
		if exp.Type != act.Type || exp.Image != act.Image || exp.Source != act.Source {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "infra.containers",
				Action:   DiffModified,
				Key:      name,
				Expected: fmt.Sprintf("type=%s image=%s source=%s", exp.Type, exp.Image, exp.Source),
				Actual:   fmt.Sprintf("type=%s image=%s source=%s", act.Type, act.Image, act.Source),
			})
		}
	}

	for name := range actualContainers {
		if _, ok := expectedContainers[name]; !ok {
			act := actualContainers[name]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "infra.containers",
				Action: DiffAdded,
				Key:    name,
				Actual: fmt.Sprintf("type=%s image=%s source=%s", act.Type, act.Image, act.Source),
			})
		}
	}

	// Compare deployment type
	if expected.DeploymentType != actual.DeploymentType {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "infra.deployment_type",
			Action:   DiffModified,
			Key:      "deployment_type",
			Expected: expected.DeploymentType,
			Actual:   actual.DeploymentType,
		})
	}
}

func diffFileTree(diff *FragmentDiff, expected, actual *architect.FileTreeInfo) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:  "file_tree",
			Action: DiffAdded,
			Key:    "file_tree",
			Actual: fmt.Sprintf("files=%d dirs=%d", actual.TotalFiles, actual.TotalDirs),
		})
		return
	}
	if actual == nil {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "file_tree",
			Action:   DiffRemoved,
			Key:      "file_tree",
			Expected: fmt.Sprintf("files=%d dirs=%d", expected.TotalFiles, expected.TotalDirs),
		})
		return
	}

	// Compare top-level directories
	expectedTop := make(map[string]bool)
	for _, d := range expected.TopLevel {
		expectedTop[d] = true
	}
	actualTop := make(map[string]bool)
	for _, d := range actual.TopLevel {
		actualTop[d] = true
	}

	for d := range expectedTop {
		if !actualTop[d] {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "file_tree.top_level",
				Action:   DiffRemoved,
				Key:      d,
				Expected: d,
			})
		}
	}
	for d := range actualTop {
		if !expectedTop[d] {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "file_tree.top_level",
				Action: DiffAdded,
				Key:    d,
				Actual: d,
			})
		}
	}

	// Compare file counts
	if expected.TotalFiles != actual.TotalFiles {
		diff.Entries = append(diff.Entries, DiffEntry{
			Field:    "file_tree.total_files",
			Action:   DiffModified,
			Key:      "total_files",
			Expected: fmt.Sprintf("%d", expected.TotalFiles),
			Actual:   fmt.Sprintf("%d", actual.TotalFiles),
		})
	}

	// Compare extension counts
	diffExtCounts(diff, expected.ExtCounts, actual.ExtCounts)
}

func diffExtCounts(diff *FragmentDiff, expected, actual map[string]int) {
	for ext, expCount := range expected {
		actCount, ok := actual[ext]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "file_tree.ext_counts",
				Action:   DiffRemoved,
				Key:      ext,
				Expected: fmt.Sprintf("%d", expCount),
			})
			continue
		}
		if expCount != actCount {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "file_tree.ext_counts",
				Action:   DiffModified,
				Key:      ext,
				Expected: fmt.Sprintf("%d", expCount),
				Actual:   fmt.Sprintf("%d", actCount),
			})
		}
	}
	for ext, actCount := range actual {
		if _, ok := expected[ext]; !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "file_tree.ext_counts",
				Action: DiffAdded,
				Key:    ext,
				Actual: fmt.Sprintf("%d", actCount),
			})
		}
	}
}

func diffSpecs(diff *FragmentDiff, expected, actual []architect.SpecArtifact) {
	expectedMap := make(map[string]architect.SpecArtifact)
	for _, s := range expected {
		expectedMap[s.Path] = s
	}
	actualMap := make(map[string]architect.SpecArtifact)
	for _, s := range actual {
		actualMap[s.Path] = s
	}

	for path, exp := range expectedMap {
		act, ok := actualMap[path]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "specs",
				Action:   DiffRemoved,
				Key:      path,
				Expected: fmt.Sprintf("kind=%s", exp.Kind),
			})
			continue
		}
		if exp.Kind != act.Kind || exp.Version != act.Version {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "specs",
				Action:   DiffModified,
				Key:      path,
				Expected: fmt.Sprintf("kind=%s version=%s", exp.Kind, exp.Version),
				Actual:   fmt.Sprintf("kind=%s version=%s", act.Kind, act.Version),
			})
		}
	}

	for path := range actualMap {
		if _, ok := expectedMap[path]; !ok {
			act := actualMap[path]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "specs",
				Action: DiffAdded,
				Key:    path,
				Actual: fmt.Sprintf("kind=%s", act.Kind),
			})
		}
	}
}

func diffSQL(diff *FragmentDiff, expected, actual *architect.SQLAnalysis) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		for _, t := range actual.Tables {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "sql.tables",
				Action: DiffAdded,
				Key:    t.Name,
				Actual: fmt.Sprintf("schema=%s columns=%d", t.Schema, len(t.Columns)),
			})
		}
		return
	}
	if actual == nil {
		for _, t := range expected.Tables {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "sql.tables",
				Action:   DiffRemoved,
				Key:      t.Name,
				Expected: fmt.Sprintf("schema=%s columns=%d", t.Schema, len(t.Columns)),
			})
		}
		return
	}

	expectedTables := make(map[string]architect.Table)
	for _, t := range expected.Tables {
		expectedTables[t.Name] = t
	}
	actualTables := make(map[string]architect.Table)
	for _, t := range actual.Tables {
		actualTables[t.Name] = t
	}

	for name, exp := range expectedTables {
		act, ok := actualTables[name]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "sql.tables",
				Action:   DiffRemoved,
				Key:      name,
				Expected: fmt.Sprintf("schema=%s columns=%d", exp.Schema, len(exp.Columns)),
			})
			continue
		}
		if exp.Schema != act.Schema || len(exp.Columns) != len(act.Columns) {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "sql.tables",
				Action:   DiffModified,
				Key:      name,
				Expected: fmt.Sprintf("schema=%s columns=%d", exp.Schema, len(exp.Columns)),
				Actual:   fmt.Sprintf("schema=%s columns=%d", act.Schema, len(act.Columns)),
			})
		}
	}

	for name := range actualTables {
		if _, ok := expectedTables[name]; !ok {
			act := actualTables[name]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "sql.tables",
				Action: DiffAdded,
				Key:    name,
				Actual: fmt.Sprintf("schema=%s columns=%d", act.Schema, len(act.Columns)),
			})
		}
	}
}

func diffGenerated(diff *FragmentDiff, expected, actual []architect.GeneratedFile) {
	expectedMap := make(map[string]architect.GeneratedFile)
	for _, g := range expected {
		expectedMap[g.Path] = g
	}
	actualMap := make(map[string]architect.GeneratedFile)
	for _, g := range actual {
		actualMap[g.Path] = g
	}

	for path, exp := range expectedMap {
		act, ok := actualMap[path]
		if !ok {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "generated",
				Action:   DiffRemoved,
				Key:      path,
				Expected: fmt.Sprintf("reason=%s", exp.Reason),
			})
			continue
		}
		if exp.Reason != act.Reason {
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:    "generated",
				Action:   DiffModified,
				Key:      path,
				Expected: exp.Reason,
				Actual:   act.Reason,
			})
		}
	}

	for path := range actualMap {
		if _, ok := expectedMap[path]; !ok {
			act := actualMap[path]
			diff.Entries = append(diff.Entries, DiffEntry{
				Field:  "generated",
				Action: DiffAdded,
				Key:    path,
				Actual: fmt.Sprintf("reason=%s", act.Reason),
			})
		}
	}
}

// --- Helpers ---

func depKey(d architect.DependencyInfo) string {
	if d.File != "" {
		return d.File
	}
	return d.Language
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Compare as sets for unordered comparison
	aSet := make(map[string]bool, len(a))
	for _, s := range a {
		aSet[s] = true
	}
	for _, s := range b {
		if !aSet[s] {
			return false
		}
	}
	return true
}

func distributionEqual(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// FormatDiff produces a human-readable diff output.
func FormatDiff(diff *FragmentDiff) string {
	if len(diff.Entries) == 0 {
		return "No differences found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Structural Diff (%s):\n", diff.Summary())

	// Group entries by field for readability
	byField := make(map[string][]DiffEntry)
	for _, e := range diff.Entries {
		byField[e.Field] = append(byField[e.Field], e)
	}

	// Sort field names for deterministic output
	fields := make([]string, 0, len(byField))
	for f := range byField {
		fields = append(fields, f)
	}
	sort.Strings(fields)

	for _, field := range fields {
		entries := byField[field]
		fmt.Fprintf(&b, "\n  [%s]\n", field)
		for _, e := range entries {
			switch e.Action {
			case DiffAdded:
				fmt.Fprintf(&b, "    + %s: %s\n", e.Key, e.Actual)
			case DiffRemoved:
				fmt.Fprintf(&b, "    - %s: %s\n", e.Key, e.Expected)
			case DiffModified:
				fmt.Fprintf(&b, "    ~ %s: %s -> %s\n", e.Key, e.Expected, e.Actual)
			}
		}
	}

	return b.String()
}
