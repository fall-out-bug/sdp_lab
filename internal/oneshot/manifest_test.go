package oneshot

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
