package main

import (
	"os"
	"path/filepath"
)

func ensureOneShotManifestFiles(repo string) error {
	corePath := filepath.Join(repo, "internal", "oneshot", "manifest.go")
	testPath := filepath.Join(repo, "internal", "oneshot", "manifest_test.go")
	if err := os.MkdirAll(filepath.Dir(corePath), 0o755); err != nil {
		return err
	}
	core := `package oneshot

import (
	"fmt"
	"sort"
	"strings"
)

type PlannerNode struct {
	ID         string   ` + "`json:\"id\"`" + `
	Owner      string   ` + "`json:\"owner\"`" + `
	DependsOn  []string ` + "`json:\"depends_on\"`" + `
	Artifacts  []string ` + "`json:\"artifacts\"`" + `
	ContractID string   ` + "`json:\"contract_id\"`" + `
}

type PlannerGraph struct {
	Nodes []PlannerNode ` + "`json:\"nodes\"`" + `
}

type ExecutionTask struct {
	ID        string   ` + "`json:\"id\"`" + `
	Role      string   ` + "`json:\"role\"`" + `
	DependsOn []string ` + "`json:\"depends_on\"`" + `
	Artifacts []string ` + "`json:\"artifacts\"`" + `
	Contract  string   ` + "`json:\"contract\"`" + `
}

type ExecutionManifest struct {
	RoleLanes map[string][]string ` + "`json:\"role_lanes\"`" + `
	Tasks     []ExecutionTask     ` + "`json:\"tasks\"`" + `
}

func BuildExecutionManifest(graph PlannerGraph) (ExecutionManifest, error) {
	if len(graph.Nodes) == 0 {
		return ExecutionManifest{}, fmt.Errorf("planner graph has no nodes")
	}

	tasks := make([]ExecutionTask, 0, len(graph.Nodes))
	lanes := make(map[string][]string)

	for _, n := range graph.Nodes {
		id := strings.TrimSpace(n.ID)
		owner := strings.TrimSpace(n.Owner)
		if id == "" {
			return ExecutionManifest{}, fmt.Errorf("node id is required")
		}
		if owner == "" {
			return ExecutionManifest{}, fmt.Errorf("node %s owner is required", id)
		}

		deps := append([]string(nil), n.DependsOn...)
		sort.Strings(deps)
		arts := append([]string(nil), n.Artifacts...)
		sort.Strings(arts)

		tasks = append(tasks, ExecutionTask{ID: id, Role: owner, DependsOn: deps, Artifacts: arts, Contract: strings.TrimSpace(n.ContractID)})
		lanes[owner] = append(lanes[owner], id)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	for role := range lanes {
		sort.Strings(lanes[role])
	}

	return ExecutionManifest{RoleLanes: lanes, Tasks: tasks}, nil
}
`
	test := `package oneshot

import (
	"reflect"
	"testing"
)

func TestBuildExecutionManifestDeterministic(t *testing.T) {
	graph := PlannerGraph{Nodes: []PlannerNode{{ID: "review", Owner: "reviewer", DependsOn: []string{"build"}, Artifacts: []string{"evidence", "pr"}, ContractID: "handoff-review"}, {ID: "build", Owner: "coder", DependsOn: []string{"plan"}, Artifacts: []string{"diff", "tests"}, ContractID: "handoff-build"}, {ID: "plan", Owner: "analyst", Artifacts: []string{"manifest"}, ContractID: "handoff-plan"}}}
	out, err := BuildExecutionManifest(graph)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if got := out.Tasks[0].ID; got != "build" {
		t.Fatalf("unexpected first task ordering: %s", got)
	}
	if !reflect.DeepEqual(out.RoleLanes["analyst"], []string{"plan"}) {
		t.Fatalf("unexpected analyst lane: %#v", out.RoleLanes["analyst"])
	}
	if !reflect.DeepEqual(out.Tasks[0].Artifacts, []string{"diff", "tests"}) {
		t.Fatalf("unexpected sorted artifacts: %#v", out.Tasks[0].Artifacts)
	}
}

func TestBuildExecutionManifestValidation(t *testing.T) {
	if _, err := BuildExecutionManifest(PlannerGraph{}); err == nil {
		t.Fatal("expected empty graph validation error")
	}
	if _, err := BuildExecutionManifest(PlannerGraph{Nodes: []PlannerNode{{ID: "x"}}}); err == nil {
		t.Fatal("expected missing owner validation error")
	}
}
`
	if err := os.WriteFile(corePath, []byte(core), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(testPath, []byte(test), 0o644); err != nil {
		return err
	}
	return nil
}
