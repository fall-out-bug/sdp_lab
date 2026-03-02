package planner

import (
	"testing"
)

func TestNewPlan(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	if plan.ID() == "" {
		t.Error("plan ID should not be empty")
	}
	if plan.Title() == "" {
		t.Error("plan title should not be empty")
	}
	if plan.Goal() == "" {
		t.Error("plan goal should not be empty")
	}
	if plan.Status() != PlanStatusDraft {
		t.Errorf("new plan should have draft status, got %s", plan.Status())
	}
}

func TestPlanAddPhase(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	phase := &Phase{
		ID:          "phase-001",
		Name:        "Phase 1",
		Description: "First phase",
	}

	err := plan.AddPhase(phase)
	if err != nil {
		t.Fatalf("failed to add phase: %v", err)
	}

	phases := plan.GetPhases()
	if len(phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(phases))
	}
}

func TestPlanAddTask(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	task := &Task{
		ID:          "task-001",
		Title:       "Task 1",
		Description: "First task",
		Status:      TaskStatusPending,
	}

	err := plan.AddTask(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	tasks := plan.GetReadyTasks()
	if len(tasks) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(tasks))
	}
}

func TestPlanAddTaskWithDependency(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})

	task2 := &Task{
		ID:           "task-002",
		Title:        "Task 2",
		Status:       TaskStatusPending,
		Dependencies: []TaskID{"task-001"},
	}

	err := plan.AddTask(task2)
	if err != nil {
		t.Fatalf("failed to add task with dependency: %v", err)
	}
}

func TestPlanAddTaskWithMissingDependency(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	task := &Task{
		ID:           "task-001",
		Title:        "Task 1",
		Status:       TaskStatusPending,
		Dependencies: []TaskID{"nonexistent"},
	}

	err := plan.AddTask(task)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestPlanValidate(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})

	err := plan.Validate()
	if err != nil {
		t.Fatalf("plan validation failed: %v", err)
	}

	if plan.Status() != PlanStatusValidated {
		t.Errorf("plan status should be validated, got %s", plan.Status())
	}
}

func TestPlanValidateEmptyPhases(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")

	err := plan.Validate()
	if err == nil {
		t.Fatal("expected error for empty phases")
	}
}

func TestPlanStart(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})

	_ = plan.Validate()

	err := plan.Start()
	if err != nil {
		t.Fatalf("failed to start plan: %v", err)
	}

	if plan.Status() != PlanStatusExecuting {
		t.Errorf("expected status executing, got %s", plan.Status())
	}
}

func TestPlanCompleteTask(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.Validate()
	_ = plan.Start()

	err := plan.CompleteTask("task-001")
	if err != nil {
		t.Fatalf("failed to complete task: %v", err)
	}

	progress := plan.GetProgress()
	if progress.CompletedTasks != 1 {
		t.Errorf("expected 1 completed task, got %d", progress.CompletedTasks)
	}
}

func TestPlanBlockTask(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.Validate()
	_ = plan.Start()

	err := plan.BlockTask("task-001", "blocked for testing")
	if err != nil {
		t.Fatalf("failed to block task: %v", err)
	}

	progress := plan.GetProgress()
	if progress.BlockedTasks != 1 {
		t.Errorf("expected 1 blocked task, got %d", progress.BlockedTasks)
	}
}

func TestPlanGetProgress(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusCompleted})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusInProgress})
	_ = plan.AddTask(&Task{ID: "task-003", Title: "Task 3", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-004", Title: "Task 4", Status: TaskStatusBlocked})

	progress := plan.GetProgress()
	if progress.TotalTasks != 4 {
		t.Errorf("expected 4 total tasks, got %d", progress.TotalTasks)
	}
	if progress.CompletedTasks != 1 {
		t.Errorf("expected 1 completed task, got %d", progress.CompletedTasks)
	}
	if progress.InProgressTasks != 1 {
		t.Errorf("expected 1 in progress task, got %d", progress.InProgressTasks)
	}
	if progress.BlockedTasks != 1 {
		t.Errorf("expected 1 blocked task, got %d", progress.BlockedTasks)
	}
}

func TestPlanToGraph(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending, Dependencies: []TaskID{"task-001"}})

	graph := plan.ToGraph()
	if len(graph.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) < 1 {
		t.Errorf("expected at least 1 edge, got %d", len(graph.Edges))
	}
}

func TestGraphTopologicalSort(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending, Dependencies: []TaskID{"task-001"}})
	_ = plan.AddTask(&Task{ID: "task-003", Title: "Task 3", Status: TaskStatusPending, Dependencies: []TaskID{"task-002"}})

	graph := plan.ToGraph()
	sort, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	if len(sort) != 3 {
		t.Errorf("expected 3 sorted items, got %d", len(sort))
	}
}

func TestGraphFindCycles(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending})

	// Create a cycle manually in the graph
	graph := plan.ToGraph()
	graph.AddEdge(&PlanEdge{From: "task-001", To: "task-002", Type: "depends_on"})
	graph.AddEdge(&PlanEdge{From: "task-002", To: "task-001", Type: "depends_on"})

	cycles := graph.FindCycles()
	if len(cycles) == 0 {
		t.Error("expected cycles in cyclic dependency graph")
	}
}

func TestGraphCriticalPath(t *testing.T) {
	plan := NewPlan("plan-001", "Test Plan", "Implement features")
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending, Dependencies: []TaskID{"task-001"}})
	_ = plan.AddTask(&Task{ID: "task-003", Title: "Task 3", Status: TaskStatusPending, Dependencies: []TaskID{"task-002"}})

	graph := plan.ToGraph()
	critical, err := graph.CriticalPath()
	if err != nil {
		t.Fatalf("critical path failed: %v", err)
	}
	if len(critical) == 0 {
		t.Error("expected at least one critical path item")
	}
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status       TaskStatus
		isReady      bool
		isComplete   bool
		isBlocked    bool
		isInProgress bool
	}{
		{TaskStatusPending, true, false, false, false},
		{TaskStatusReady, true, false, false, false},
		{TaskStatusInProgress, false, false, false, true},
		{TaskStatusCompleted, false, true, false, false},
		{TaskStatusBlocked, false, false, true, false},
	}

	for _, tt := range tests {
		task := &Task{ID: "task-001", Title: "Test", Status: tt.status}
		if task.IsReady() != tt.isReady {
			t.Errorf("status %s: IsReady() = %v, want %v", tt.status, task.IsReady(), tt.isReady)
		}
		if task.IsComplete() != tt.isComplete {
			t.Errorf("status %s: IsComplete() = %v, want %v", tt.status, task.IsComplete(), tt.isComplete)
		}
		if task.IsBlocked() != tt.isBlocked {
			t.Errorf("status %s: IsBlocked() = %v, want %v", tt.status, task.IsBlocked(), tt.isBlocked)
		}
		if task.IsInProgress() != tt.isInProgress {
			t.Errorf("status %s: IsInProgress() = %v, want %v", tt.status, task.IsInProgress(), tt.isInProgress)
		}
	}
}

func TestTaskToMap(t *testing.T) {
	task := &Task{
		ID:             "task-001",
		Title:          "Test Task",
		Description:    "Test Description",
		Status:         TaskStatusPending,
		Priority:       1,
		EstimatedHours: 1.5,
	}

	m := task.ToMap()
	if m["id"] != "task-001" {
		t.Errorf("expected id task-001, got %v", m["id"])
	}
	if m["title"] != "Test Task" {
		t.Errorf("expected title 'Test Task', got %v", m["title"])
	}
	if m["status"] != "pending" {
		t.Errorf("expected status pending, got %v", m["status"])
	}
}

func TestTaskValidation(t *testing.T) {
	task := &Task{ID: "task-001", Title: "Task 1"}
	if err := task.Validate(); err != nil {
		t.Errorf("task validation failed: %v", err)
	}

	taskEmptyID := &Task{ID: "", Title: "Task 1"}
	if err := taskEmptyID.Validate(); err == nil {
		t.Error("expected error for empty task ID")
	}

	taskEmptyTitle := &Task{ID: "task-001", Title: ""}
	if err := taskEmptyTitle.Validate(); err == nil {
		t.Error("expected error for empty task title")
	}
}

func TestSchedulerCreatePlan(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	plan, err := scheduler.CreatePlan(nil, "plan-001", "Test Plan", "Test Goal")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	if plan.ID() != "plan-001" {
		t.Errorf("expected plan ID plan-001, got %s", plan.ID())
	}
}

func TestSchedulerGetPlan(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	_, _ = scheduler.CreatePlan(nil, "plan-001", "Test Plan", "Test Goal")

	plan, err := scheduler.GetPlan("plan-001")
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}

	if plan.Title() != "Test Plan" {
		t.Errorf("expected title 'Test Plan', got %s", plan.Title())
	}
}

func TestSchedulerGetPlanNotFound(t *testing.T) {
	scheduler := NewScheduler(nil, nil)

	_, err := scheduler.GetPlan("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plan")
	}
}
