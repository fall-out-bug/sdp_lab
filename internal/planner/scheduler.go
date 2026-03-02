package planner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	mu        sync.RWMutex
	plans     map[string]*Plan
	executor  TaskExecutor
	validator ScheduleValidator
}

type TaskExecutor interface {
	Execute(ctx context.Context, task *Task) error
	Cancel(ctx context.Context, taskID TaskID) error
	Status(ctx context.Context, taskID TaskID) (TaskStatus, error)
}

type ScheduleValidator interface {
	ValidateDependencies(plan *Plan) error
	ValidateResources(plan *Plan) error
	ValidateTiming(plan *Plan) error
}

func NewScheduler(executor TaskExecutor, validator ScheduleValidator) *Scheduler {
	return &Scheduler{
		plans:     make(map[string]*Plan),
		executor:  executor,
		validator: validator,
	}
}

func (s *Scheduler) CreatePlan(ctx context.Context, id, title, goal string) (*Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[id]; exists {
		return nil, fmt.Errorf("plan %s already exists", id)
	}

	plan := NewPlan(id, title, goal)
	s.plans[id] = plan
	return plan, nil
}

func (s *Scheduler) GetPlan(id string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, exists := s.plans[id]
	if !exists {
		return nil, fmt.Errorf("plan %s not found", id)
	}
	return plan, nil
}

func (s *Scheduler) Schedule(ctx context.Context, planID string) ([]TaskID, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return nil, err
	}

	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	if s.validator != nil {
		if err := s.validator.ValidateDependencies(plan); err != nil {
			return nil, fmt.Errorf("dependency validation failed: %w", err)
		}
		if err := s.validator.ValidateResources(plan); err != nil {
			return nil, fmt.Errorf("resource validation failed: %w", err)
		}
		if err := s.validator.ValidateTiming(plan); err != nil {
			return nil, fmt.Errorf("timing validation failed: %w", err)
		}
	}

	graph := plan.ToGraph()

	sort, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("schedule contains cycles: %w", err)
	}

	if err := plan.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plan: %w", err)
	}

	var scheduled []TaskID
	for _, nodeID := range sort {
		task, err := plan.GetTask(TaskID(nodeID))
		if err != nil {
			continue
		}

		if task.Status != TaskStatusPending {
			continue
		}

		if !s.canStartTask(plan, task) {
			continue
		}

		if err := s.startTask(ctx, plan, task); err != nil {
			plan.BlockTask(task.ID, err.Error())
			continue
		}

		scheduled = append(scheduled, task.ID)
	}

	return scheduled, nil
}

func (s *Scheduler) canStartTask(plan *Plan, task *Task) bool {
	for _, depID := range task.Dependencies {
		depTask, err := plan.GetTask(depID)
		if err != nil {
			return false
		}
		if depTask.Status != TaskStatusCompleted {
			return false
		}
	}
	return true
}

func (s *Scheduler) startTask(ctx context.Context, plan *Plan, task *Task) error {
	now := time.Now()
	task.Status = TaskStatusInProgress
	task.StartedAt = &now

	if s.executor != nil {
		go func() {
			err := s.executor.Execute(context.Background(), task)
			if err != nil {
				plan.BlockTask(task.ID, err.Error())
			} else {
				plan.CompleteTask(task.ID)
			}
		}()
	}

	return nil
}

func (s *Scheduler) CancelPlan(ctx context.Context, planID string) error {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return err
	}

	for _, task := range plan.tasks {
		if task.Status == TaskStatusInProgress {
			if s.executor != nil {
				s.executor.Cancel(ctx, task.ID)
			}
			task.Status = TaskStatusPending
			task.StartedAt = nil
		}
	}

	return nil
}

func (s *Scheduler) GetProgress(planID string) (*PlanProgress, error) {
	plan, err := s.GetPlan(planID)
	if err != nil {
		return nil, err
	}
	return plan.GetProgress(), nil
}

type DefaultExecutor struct{}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e *DefaultExecutor) Execute(ctx context.Context, task *Task) error {
	return nil
}

func (e *DefaultExecutor) Cancel(ctx context.Context, taskID TaskID) error {
	return nil
}

func (e *DefaultExecutor) Status(ctx context.Context, taskID TaskID) (TaskStatus, error) {
	return TaskStatusPending, nil
}

type DefaultValidator struct{}

func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{}
}

func (v *DefaultValidator) ValidateDependencies(plan *Plan) error {
	return nil
}

func (v *DefaultValidator) ValidateResources(plan *Plan) error {
	return nil
}

func (v *DefaultValidator) ValidateTiming(plan *Plan) error {
	return nil
}
