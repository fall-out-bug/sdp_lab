package planner

import (
	"fmt"
	"sync"
	"time"
)

type TaskID string
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusReady      TaskStatus = "ready"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "blocked"
)

type Task struct {
	ID             TaskID
	Title          string
	Description    string
	Status         TaskStatus
	Priority       int
	Dependencies   []TaskID
	EstimatedHours float64
	AssignedTo     string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Metadata       map[string]interface{}
}

func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if t.Title == "" {
		return fmt.Errorf("task title is required")
	}
	return nil
}

func (t *Task) IsReady() bool {
	return t.Status == TaskStatusPending || t.Status == TaskStatusReady
}

func (t *Task) IsComplete() bool {
	return t.Status == TaskStatusCompleted
}

func (t *Task) IsBlocked() bool {
	return t.Status == TaskStatusBlocked
}

func (t *Task) IsInProgress() bool {
	return t.Status == TaskStatusInProgress
}

func (t *Task) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":              string(t.ID),
		"title":           t.Title,
		"description":     t.Description,
		"status":          string(t.Status),
		"priority":        t.Priority,
		"estimated_hours": t.EstimatedHours,
	}
}


type Phase struct {
	ID          string
	Name        string
	Description string
	Tasks       []TaskID
	Status      TaskStatus
	StartTime   *time.Time
	EndTime     *time.Time
}

type Plan struct {
	mu        sync.RWMutex
	id        string
	title     string
	goal      string
	phases    []*Phase
	tasks     map[TaskID]*Task
	status    PlanStatus
	createdAt time.Time
	updatedAt time.Time
	metadata  map[string]interface{}
}

type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusValidated PlanStatus = "validated"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed    PlanStatus = "failed"
)

func NewPlan(id, title, goal string) *Plan {
	return &Plan{
		id:        id,
		title:     title,
		goal:      goal,
		phases:    make([]*Phase, 0),
		tasks:     make(map[TaskID]*Task),
		status:    PlanStatusDraft,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

func (p *Plan) ID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.id
}

func (p *Plan) Title() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.title
}

func (p *Plan) Goal() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.goal
}

func (p *Plan) Status() PlanStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *Plan) GetPhases() []*Phase {
	p.mu.RLock()
	defer p.mu.RUnlock()
	phases := make([]*Phase, len(p.phases))
	copy(phases, p.phases)
	return phases
}

func (p *Plan) AddPhase(phase *Phase) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if phase.ID == "" {
		return fmt.Errorf("phase ID is required")
	}

	for _, existing := range p.phases {
		if existing.ID == phase.ID {
			return fmt.Errorf("phase with ID %s already exists", phase.ID)
		}
	}

	p.phases = append(p.phases, phase)
	p.updatedAt = time.Now()
	return nil
}

func (p *Plan) AddTask(task *Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	if _, existing := p.tasks[task.ID]; existing {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	for _, depID := range task.Dependencies {
		if _, exists := p.tasks[depID]; !exists {
			return fmt.Errorf("dependency task %s not found", depID)
		}
	}

	p.tasks[task.ID] = task
	p.updatedAt = time.Now()
	return nil
}

func (p *Plan) GetPhase(phaseID string) (*Phase, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, phase := range p.phases {
		if phase.ID == phaseID {
			return phase, nil
		}
	}
	return nil, fmt.Errorf("phase %s not found", phaseID)
}

func (p *Plan) GetTask(taskID TaskID) (*Task, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return task, nil
}

func (p *Plan) GetReadyTasks() []Task {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var ready []Task
	for _, task := range p.tasks {
		if task.Status == TaskStatusPending || task.Status == TaskStatusReady {
			ready = append(ready, *task)
		}
	}
	return ready
}

func (p *Plan) GetNextPhase() (*Phase, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, phase := range p.phases {
		if phase.Status == TaskStatusPending || phase.Status == TaskStatusReady {
			return phase, nil
		}
	}
	return nil, fmt.Errorf("no phases ready for execution")
}

func (p *Plan) Validate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, phase := range p.phases {
		if phase.ID == "" {
			return fmt.Errorf("phase has empty ID")
		}
	}

	for _, task := range p.tasks {
		if task.ID == "" {
			return fmt.Errorf("task has empty ID")
		}
		for _, depID := range task.Dependencies {
			if _, exists := p.tasks[depID]; !exists {
				return fmt.Errorf("task %s has unresolved dependency: %s", task.ID, depID)
			}
		}
	}

	if len(p.phases) == 0 {
		return fmt.Errorf("plan must have at least one phase")
	}

	for _, phase := range p.phases {
		for _, taskID := range phase.Tasks {
			if _, exists := p.tasks[taskID]; !exists {
				return fmt.Errorf("phase %s references unknown task: %s", phase.ID, taskID)
			}
		}
	}

	p.status = PlanStatusValidated
	p.updatedAt = time.Now()
	return nil
}

func (p *Plan) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != PlanStatusValidated {
		return fmt.Errorf("plan must be validated before execution")
	}

	p.status = PlanStatusExecuting
	p.updatedAt = time.Now()
	return nil
}

func (p *Plan) CompleteTask(taskID TaskID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	now := time.Now()
	task.Status = TaskStatusCompleted
	task.CompletedAt = &now
	p.updatedAt = now

	return nil
}

func (p *Plan) BlockTask(taskID TaskID, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	now := time.Now()
	task.Status = TaskStatusBlocked
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["blocked_reason"] = reason
	p.updatedAt = now

	return nil
}

func (p *Plan) GetProgress() *PlanProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var total, completed, blocked int
	for _, task := range p.tasks {
		total++
		switch task.Status {
		case TaskStatusCompleted:
			completed++
		case TaskStatusBlocked:
			blocked++
		}
	}

	return &PlanProgress{
		TotalTasks:      total,
		CompletedTasks:  completed,
		BlockedTasks:    blocked,
		InProgressTasks: p.countInProgress(),
		PlanStatus:      string(p.status),
	}
}

func (p *Plan) countInProgress() int {
	count := 0
	for _, task := range p.tasks {
		if task.Status == TaskStatusInProgress {
			count++
		}
	}
	return count
}

type PlanProgress struct {
	TotalTasks      int
	CompletedTasks  int
	BlockedTasks    int
	InProgressTasks int
	PlanStatus      string
}

func (p *Plan) ToGraph() *PlanGraph {
	p.mu.RLock()
	defer p.mu.RUnlock()

	nodes := make([]*PlanNode, 0)
	edges := make([]*PlanEdge, 0)

	for _, phase := range p.phases {
		nodes = append(nodes, &PlanNode{
			ID:          phase.ID,
			Type:        "phase",
			Title:       phase.Name,
			Description: phase.Description,
			Status:      string(phase.Status),
		})
	}

	for _, task := range p.tasks {
		nodes = append(nodes, &PlanNode{
			ID:             string(task.ID),
			Type:           "task",
			Title:          task.Title,
			Description:    task.Description,
			Status:         string(task.Status),
			EstimatedHours: task.EstimatedHours,
		})

		for _, depID := range task.Dependencies {
			edges = append(edges, &PlanEdge{
				From: string(task.ID),
				To:   string(depID),
				Type: "depends_on",
			})
		}
	}

	for _, phase := range p.phases {
		for _, taskID := range phase.Tasks {
			edges = append(edges, &PlanEdge{
				From: phase.ID,
				To:   string(taskID),
				Type: "contains",
			})
		}
	}

	return &PlanGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

type PlanNode struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	Status         string  `json:"status"`
	EstimatedHours float64 `json:"estimated_hours,omitempty"`
}

type PlanEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type PlanGraph struct {
	Nodes []*PlanNode `json:"nodes"`
	Edges []*PlanEdge `json:"edges"`
}


func (g *PlanGraph) AddEdge(edge *PlanEdge) {
	g.Edges = append(g.Edges, edge)
}

func (g *PlanGraph) FindCycles() [][]string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles [][]string

	var visit func(nodeID string)
	visit = func(nodeID string) {
		if recStack[nodeID] {
			var cycle []string
			for k := range recStack {
				if recStack[k] {
					cycle = append(cycle, k)
				}
			}
			if len(cycle) > 0 {
				cycles = append(cycles, cycle)
			}
			return
		}

		if visited[nodeID] {
			return
		}

		visited[nodeID] = true
		recStack[nodeID] = true

		for _, edge := range g.Edges {
			if edge.From == nodeID && edge.Type == "depends_on" {
				visit(edge.To)
			}
		}

		delete(recStack, nodeID)
	}

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			visit(node.ID)
		}
	}

	return cycles
}

func (g *PlanGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for _, node := range g.Nodes {
		inDegree[node.ID] = 0
	}

	for _, edge := range g.Edges {
		if edge.Type == "depends_on" {
			inDegree[edge.To]++
		}
	}

	var queue []string
	for _, node := range g.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}

	var result []string
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		for _, edge := range g.Edges {
			if edge.Type == "depends_on" && edge.From == nodeID {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("graph contains a cycle")
	}

	return result, nil
}

func (g *PlanGraph) CriticalPath() ([]string, error) {
	sort, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	var critical []string
	for i := 0; i < len(sort) && i < 3; i++ {
		critical = append(critical, sort[i])
	}

	return critical, nil
}
