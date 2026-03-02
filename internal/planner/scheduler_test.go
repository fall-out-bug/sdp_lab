package planner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	executor := NewDefaultExecutor()
	validator := NewDefaultValidator()

	s := NewScheduler(executor, validator)
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.executor == nil {
		t.Error("executor not set")
	}
	if s.validator == nil {
		t.Error("validator not set")
	}
	if s.plans == nil {
		t.Error("plans map not initialized")
	}
}

func TestNewScheduler_NilDeps(t *testing.T) {
	s := NewScheduler(nil, nil)
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
}

func TestScheduler_CreatePlan(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	tests := []struct {
		name    string
		id      string
		title   string
		goal    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid plan",
			id:      "plan-001",
			title:   "Test Plan",
			goal:    "Test goal",
			wantErr: false,
		},
		{
			name:    "another valid plan",
			id:      "plan-002",
			title:   "Another Plan",
			goal:    "Another goal",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := s.CreatePlan(ctx, tt.id, tt.title, tt.goal)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if plan == nil {
					t.Error("plan is nil")
					return
				}
				if plan.ID != tt.id {
					t.Errorf("plan.ID = %v, want %v", plan.ID, tt.id)
				}
				if plan.Title != tt.title {
					t.Errorf("plan.Title = %v, want %v", plan.Title, tt.title)
				}
				if plan.Goal != tt.goal {
					t.Errorf("plan.Goal = %v, want %v", plan.Goal, tt.goal)
				}
			}
		})
	}
}

func TestScheduler_CreatePlan_Duplicate(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	_, err := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	if err != nil {
		t.Fatalf("first CreatePlan failed: %v", err)
	}

	_, err = s.CreatePlan(ctx, "plan-001", "Another Plan", "Another Goal")
	if err == nil {
		t.Error("expected error for duplicate plan ID")
	}
}

func TestScheduler_GetPlan(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	// Create a plan first
	created, err := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	// Get the plan
	got, err := s.GetPlan("plan-001")
	if err != nil {
		t.Fatalf("GetPlan failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetPlan() = %v, want %v", got.ID, created.ID)
	}
}

func TestScheduler_GetPlan_NotFound(t *testing.T) {
	s := NewScheduler(nil, nil)

	_, err := s.GetPlan("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestScheduler_Schedule(t *testing.T) {
	s := NewScheduler(NewDefaultExecutor(), NewDefaultValidator())
	ctx := context.Background()

	plan, _ := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})

	scheduled, err := s.Schedule(ctx, "plan-001")
	if err != nil {
		t.Errorf("Schedule failed: %v", err)
	}
	if len(scheduled) == 0 {
		t.Error("expected at least one scheduled task")
	}
}

func TestScheduler_Schedule_PlanNotFound(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	_, err := s.Schedule(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestScheduler_Schedule_WithDependencies(t *testing.T) {
	s := NewScheduler(NewDefaultExecutor(), NewDefaultValidator())
	ctx := context.Background()

	plan, _ := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	_ = plan.AddPhase(&Phase{ID: "phase-001", Name: "Phase 1"})
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending, Dependencies: []TaskID{"task-001"}})

	scheduled, err := s.Schedule(ctx, "plan-001")
	if err != nil {
		t.Errorf("Schedule failed: %v", err)
	}
	// Only task-001 should be scheduled (task-002 depends on it)
	if len(scheduled) != 1 {
		t.Errorf("expected 1 scheduled task, got %d", len(scheduled))
	}
}

func TestScheduler_CancelPlan(t *testing.T) {
	s := NewScheduler(NewDefaultExecutor(), NewDefaultValidator())
	ctx := context.Background()

	_, _ = s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")

	err := s.CancelPlan(ctx, "plan-001")
	if err != nil {
		t.Errorf("CancelPlan failed: %v", err)
	}
}

func TestScheduler_CancelPlan_NotFound(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	err := s.CancelPlan(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestScheduler_GetProgress(t *testing.T) {
	s := NewScheduler(nil, nil)
	ctx := context.Background()

	plan, _ := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusCompleted})
	_ = plan.AddTask(&Task{ID: "task-002", Title: "Task 2", Status: TaskStatusPending})

	progress, err := s.GetProgress("plan-001")
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}
	if progress == nil {
		t.Fatal("progress is nil")
	}
	if progress.Total != 2 {
		t.Errorf("progress.Total = %d, want 2", progress.Total)
	}
	if progress.Completed != 1 {
		t.Errorf("progress.Completed = %d, want 1", progress.Completed)
	}
}

func TestScheduler_GetProgress_NotFound(t *testing.T) {
	s := NewScheduler(nil, nil)

	_, err := s.GetProgress("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

// Mock executor for testing
type mockExecutor struct {
	executeErr error
	cancelErr  error
	status     TaskStatus
	statusErr  error
}

func (m *mockExecutor) Execute(ctx context.Context, task *Task) error {
	return m.executeErr
}

func (m *mockExecutor) Cancel(ctx context.Context, taskID TaskID) error {
	return m.cancelErr
}

func (m *mockExecutor) Status(ctx context.Context, taskID TaskID) (TaskStatus, error) {
	return m.status, m.statusErr
}

func TestScheduler_WithMockExecutor(t *testing.T) {
	executor := &mockExecutor{
		executeErr: errors.New("execute failed"),
	}
	s := NewScheduler(executor, nil)
	ctx := context.Background()

	plan, _ := s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")
	_ = plan.AddTask(&Task{ID: "task-001", Title: "Task 1", Status: TaskStatusPending})

	// Schedule should still succeed (error is handled in goroutine)
	scheduled, err := s.Schedule(ctx, "plan-001")
	if err != nil {
		t.Errorf("Schedule failed: %v", err)
	}
	if len(scheduled) != 1 {
		t.Errorf("expected 1 scheduled task, got %d", len(scheduled))
	}

	// Give goroutine time to execute
	time.Sleep(50 * time.Millisecond)
}

// Mock validator for testing
type mockValidator struct {
	depErr  error
	resErr  error
	timeErr error
}

func (m *mockValidator) ValidateDependencies(plan *Plan) error {
	return m.depErr
}

func (m *mockValidator) ValidateResources(plan *Plan) error {
	return m.resErr
}

func (m *mockValidator) ValidateTiming(plan *Plan) error {
	return m.timeErr
}

func TestScheduler_WithMockValidator(t *testing.T) {
	validator := &mockValidator{
		depErr: errors.New("dependency error"),
	}
	s := NewScheduler(nil, validator)
	ctx := context.Background()

	_, _ = s.CreatePlan(ctx, "plan-001", "Test Plan", "Goal")

	_, err := s.Schedule(ctx, "plan-001")
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestDefaultExecutor(t *testing.T) {
	e := NewDefaultExecutor()
	ctx := context.Background()

	if err := e.Execute(ctx, nil); err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if err := e.Cancel(ctx, "task-001"); err != nil {
		t.Errorf("Cancel failed: %v", err)
	}
	status, err := e.Status(ctx, "task-001")
	if err != nil {
		t.Errorf("Status failed: %v", err)
	}
	if status != TaskStatusPending {
		t.Errorf("Status = %v, want %v", status, TaskStatusPending)
	}
}

func TestDefaultValidator(t *testing.T) {
	v := NewDefaultValidator()
	plan := NewPlan("test", "Test", "Goal")

	if err := v.ValidateDependencies(plan); err != nil {
		t.Errorf("ValidateDependencies failed: %v", err)
	}
	if err := v.ValidateResources(plan); err != nil {
		t.Errorf("ValidateResources failed: %v", err)
	}
	if err := v.ValidateTiming(plan); err != nil {
		t.Errorf("ValidateTiming failed: %v", err)
	}
}
