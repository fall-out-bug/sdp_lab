package oneshot

import (
	"fmt"
	"sort"
	"strings"
)

type PlannerNode struct {
	ID         string   `json:"id"`
	Owner      string   `json:"owner"`
	DependsOn  []string `json:"depends_on"`
	Artifacts  []string `json:"artifacts"`
	ContractID string   `json:"contract_id"`
}

type PlannerGraph struct {
	Nodes []PlannerNode `json:"nodes"`
}

type ExecutionTask struct {
	ID        string   `json:"id"`
	Role      string   `json:"role"`
	DependsOn []string `json:"depends_on"`
	Artifacts []string `json:"artifacts"`
	Contract  string   `json:"contract"`
}

type ExecutionManifest struct {
	RoleLanes map[string][]string `json:"role_lanes"`
	Tasks     []ExecutionTask     `json:"tasks"`
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
