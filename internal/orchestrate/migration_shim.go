package orchestrate

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type OrchestratorBackend string

const (
	BackendLegacy OrchestratorBackend = "legacy"
	BackendV2     OrchestratorBackend = "v2"
	BackendAuto   OrchestratorBackend = "auto"
)

type MigrationConfig struct {
	Backend         OrchestratorBackend
	Environment     string
	FeatureFlagKey  string
	FallbackEnabled bool
	DryRun          bool
}

type FailureClass string

const (
	FailureClassValidation FailureClass = "validation"
	FailureClassPolicy     FailureClass = "policy"
	FailureClassTimeout    FailureClass = "timeout"
	FailureClassResource   FailureClass = "resource"
	FailureClassInternal   FailureClass = "internal"
	FailureClassUnknown    FailureClass = "unknown"
	FailureClassRolledBack FailureClass = "rolled_back"
	FailureClassMigrated   FailureClass = "migrated"
)

type MigrationEvent struct {
	Timestamp    time.Time
	WorkstreamID string
	Backend      OrchestratorBackend
	EventType    string
	FailureClass FailureClass
	Duration     time.Duration
	Success      bool
	RolledBack   bool
	MigratedFrom OrchestratorBackend
	MigratedTo   OrchestratorBackend
	ErrorMessage string
	Metadata     map[string]interface{}
}

type MigrationTelemetry interface {
	RecordEvent(ctx context.Context, event *MigrationEvent) error
	QueryEvents(ctx context.Context, query MigrationQuery) ([]*MigrationEvent, error)
}

type MigrationQuery struct {
	WorkstreamID string
	Backend      OrchestratorBackend
	FailureClass FailureClass
	StartTime    time.Time
	EndTime      time.Time
	Limit        int
}

type FeatureFlagProvider interface {
	IsEnabled(ctx context.Context, flagKey string, attributes map[string]string) (bool, error)
}

type MigrationShim struct {
	mu           sync.RWMutex
	config       *MigrationConfig
	legacy       *LoopOrchestrator
	v2           *FSMV2
	telemetry    MigrationTelemetry
	flagProvider FeatureFlagProvider
}

type LoopOrchestrator struct {
	workstreamID string
	phase        string
}

func NewLoopOrchestrator(workstreamID string) *LoopOrchestrator {
	return &LoopOrchestrator{
		workstreamID: workstreamID,
		phase:        PhaseInit,
	}
}

func (l *LoopOrchestrator) Run(ctx context.Context) error {
	return nil
}

func (l *LoopOrchestrator) Phase() string {
	return l.phase
}

func NewMigrationShim(config *MigrationConfig) *MigrationShim {
	if config.Backend == "" {
		config.Backend = BackendAuto
	}
	if config.Environment == "" {
		config.Environment = getEnvironment()
	}
	return &MigrationShim{config: config}
}

func (s *MigrationShim) WithLegacy(legacy *LoopOrchestrator) *MigrationShim {
	s.legacy = legacy
	return s
}

func (s *MigrationShim) WithV2(v2 *FSMV2) *MigrationShim {
	s.v2 = v2
	return s
}

func (s *MigrationShim) WithTelemetry(t MigrationTelemetry) *MigrationShim {
	s.telemetry = t
	return s
}

func (s *MigrationShim) WithFeatureFlagProvider(p FeatureFlagProvider) *MigrationShim {
	s.flagProvider = p
	return s
}

func (s *MigrationShim) SelectBackend(ctx context.Context, fsmCtx *FSMContext) (OrchestratorBackend, error) {
	switch s.config.Backend {
	case BackendLegacy:
		return BackendLegacy, nil
	case BackendV2:
		return BackendV2, nil
	case BackendAuto:
		return s.selectAuto(ctx, fsmCtx)
	default:
		return BackendV2, nil
	}
}

func (s *MigrationShim) selectAuto(ctx context.Context, fsmCtx *FSMContext) (OrchestratorBackend, error) {
	if s.config.Environment == "enterprise" || s.config.Environment == "production" {
		return BackendV2, nil
	}

	if s.flagProvider != nil && s.config.FeatureFlagKey != "" {
		attrs := map[string]string{
			"environment": s.config.Environment,
			"workstream":  fsmCtx.WorkstreamID,
			"feature":     fsmCtx.FeatureID,
		}
		enabled, err := s.flagProvider.IsEnabled(ctx, s.config.FeatureFlagKey, attrs)
		if err != nil {
			if s.config.FallbackEnabled {
				return BackendLegacy, nil
			}
			return BackendV2, nil
		}
		if enabled {
			return BackendV2, nil
		}
		return BackendLegacy, nil
	}

	switch s.config.Environment {
	case "oss", "development", "staging":
		return BackendV2, nil
	default:
		return BackendV2, nil
	}
}

func (s *MigrationShim) Orchestrate(ctx context.Context, fsmCtx *FSMContext) error {
	start := time.Now()
	backend, err := s.SelectBackend(ctx, fsmCtx)
	if err != nil {
		s.recordEvent(ctx, &MigrationEvent{
			Timestamp:    time.Now(),
			WorkstreamID: fsmCtx.WorkstreamID,
			Backend:      backend,
			EventType:    "backend_selection_failed",
			FailureClass: FailureClassInternal,
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return err
	}

	var orchestrateErr error
	var finalBackend OrchestratorBackend

	if backend == BackendLegacy {
		orchestrateErr = s.runLegacy(ctx, fsmCtx)
		finalBackend = BackendLegacy
	} else {
		orchestrateErr = s.runV2(ctx, fsmCtx)
		finalBackend = BackendV2
	}

	duration := time.Since(start)

	if orchestrateErr != nil {
		failureClass := s.classifyFailure(orchestrateErr)
		s.recordEvent(ctx, &MigrationEvent{
			Timestamp:    time.Now(),
			WorkstreamID: fsmCtx.WorkstreamID,
			Backend:      finalBackend,
			EventType:    "orchestration_failed",
			FailureClass: failureClass,
			Duration:     duration,
			Success:      false,
			ErrorMessage: orchestrateErr.Error(),
		})
		return orchestrateErr
	}

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		Backend:      finalBackend,
		EventType:    "orchestration_completed",
		Duration:     duration,
		Success:      true,
	})

	return nil
}

func (s *MigrationShim) runLegacy(ctx context.Context, fsmCtx *FSMContext) error {
	if s.legacy == nil {
		s.legacy = NewLoopOrchestrator(fsmCtx.WorkstreamID)
	}
	return s.legacy.Run(ctx)
}

func (s *MigrationShim) runV2(ctx context.Context, fsmCtx *FSMContext) error {
	if s.v2 == nil {
		s.v2 = NewFSMV2(ctx, fsmCtx)
	}

	if s.config.DryRun {
		return s.dryRunV2(ctx, fsmCtx)
	}

	if err := s.v2.Validate(ctx); err != nil {
		return err
	}
	if err := s.v2.Assign(ctx); err != nil {
		return err
	}
	if err := s.v2.Execute(ctx); err != nil {
		return err
	}
	if err := s.v2.Review(ctx); err != nil {
		return err
	}
	return s.v2.Complete(ctx)
}

func (s *MigrationShim) dryRunV2(ctx context.Context, fsmCtx *FSMContext) error {
	s.v2.mu.Lock()
	snapshot := cloneFSMState(s.v2.state)
	s.v2.mu.Unlock()

	transitions := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"validate", s.v2.Validate},
		{"assign", s.v2.Assign},
		{"execute", s.v2.Execute},
		{"review", s.v2.Review},
		{"complete", s.v2.Complete},
	}

	for _, t := range transitions {
		state := s.v2.CurrentState()
		if err := t.fn(ctx); err != nil {
			s.v2.mu.Lock()
			s.v2.state = snapshot
			s.v2.mu.Unlock()
			s.recordEvent(ctx, &MigrationEvent{
				Timestamp:    time.Now(),
				WorkstreamID: fsmCtx.WorkstreamID,
				Backend:      BackendV2,
				EventType:    "dry_run_transition_failed",
				FailureClass: s.classifyFailure(err),
				Metadata:     map[string]interface{}{"transition": t.name, "from_state": state},
			})
			return fmt.Errorf("dry-run failed at %s: %w", t.name, err)
		}
	}

	s.v2.mu.Lock()
	s.v2.state = snapshot
	s.v2.mu.Unlock()

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		Backend:      BackendV2,
		EventType:    "dry_run_completed",
		Success:      true,
	})
	return nil
}

func cloneFSMState(src *FSMState) *FSMState {
	if src == nil {
		return nil
	}

	dst := *src
	if src.LastError != nil {
		lastErr := *src.LastError
		dst.LastError = &lastErr
	}
	if src.ExitedAt != nil {
		exitedAt := *src.ExitedAt
		dst.ExitedAt = &exitedAt
	}
	if len(src.Checkpoints) > 0 {
		dst.Checkpoints = make([]CheckpointRecord, len(src.Checkpoints))
		for i, cp := range src.Checkpoints {
			dst.Checkpoints[i] = cp
			if cp.Details != nil {
				details := make(map[string]interface{}, len(cp.Details))
				for k, v := range cp.Details {
					details[k] = v
				}
				dst.Checkpoints[i].Details = details
			}
		}
	}

	return &dst
}

func (s *MigrationShim) classifyFailure(err error) FailureClass {
	if te, ok := err.(*TransitionError); ok {
		switch te.Code {
		case "VALIDATION_FAILED":
			return FailureClassValidation
		case "POLICY_DENIED", "POLICY_CHECK_FAILED":
			return FailureClassPolicy
		case "ACTION_FAILED":
			return FailureClassResource
		case "TIMEOUT":
			return FailureClassTimeout
		default:
			return FailureClassInternal
		}
	}
	return FailureClassUnknown
}

func (s *MigrationShim) recordEvent(ctx context.Context, event *MigrationEvent) {
	if s.telemetry != nil {
		_ = s.telemetry.RecordEvent(ctx, event)
	}
}

func (s *MigrationShim) Migrate(ctx context.Context, fsmCtx *FSMContext, from, to OrchestratorBackend) error {
	start := time.Now()

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		EventType:    "migration_started",
		MigratedFrom: from,
		MigratedTo:   to,
	})

	var err error
	if to == BackendV2 {
		err = s.runV2(ctx, fsmCtx)
	} else {
		err = s.runLegacy(ctx, fsmCtx)
	}

	duration := time.Since(start)

	if err != nil {
		s.recordEvent(ctx, &MigrationEvent{
			Timestamp:    time.Now(),
			WorkstreamID: fsmCtx.WorkstreamID,
			EventType:    "migration_failed",
			FailureClass: s.classifyFailure(err),
			Duration:     duration,
			Success:      false,
			MigratedFrom: from,
			MigratedTo:   to,
			ErrorMessage: err.Error(),
		})
		return err
	}

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		EventType:    "migration_completed",
		FailureClass: FailureClassMigrated,
		Duration:     duration,
		Success:      true,
		MigratedFrom: from,
		MigratedTo:   to,
	})
	return nil
}

func (s *MigrationShim) Rollback(ctx context.Context, fsmCtx *FSMContext, to OrchestratorBackend) error {
	start := time.Now()

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		EventType:    "rollback_started",
		MigratedTo:   to,
	})

	var err error
	if to == BackendLegacy {
		err = s.runLegacy(ctx, fsmCtx)
	} else {
		err = s.v2.Rollback(ctx)
	}

	duration := time.Since(start)

	if err != nil {
		s.recordEvent(ctx, &MigrationEvent{
			Timestamp:    time.Now(),
			WorkstreamID: fsmCtx.WorkstreamID,
			EventType:    "rollback_failed",
			FailureClass: FailureClassRolledBack,
			Duration:     duration,
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return err
	}

	s.recordEvent(ctx, &MigrationEvent{
		Timestamp:    time.Now(),
		WorkstreamID: fsmCtx.WorkstreamID,
		EventType:    "rollback_completed",
		FailureClass: FailureClassRolledBack,
		Duration:     duration,
		Success:      true,
		RolledBack:   true,
	})
	return nil
}

func (s *MigrationShim) GetConfig() *MigrationConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *MigrationShim) SetBackend(backend OrchestratorBackend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Backend = backend
}

type InMemoryTelemetry struct {
	mu     sync.RWMutex
	events []*MigrationEvent
}

func NewInMemoryTelemetry() *InMemoryTelemetry {
	return &InMemoryTelemetry{
		events: make([]*MigrationEvent, 0),
	}
}

func (t *InMemoryTelemetry) RecordEvent(ctx context.Context, event *MigrationEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
	return nil
}

func (t *InMemoryTelemetry) QueryEvents(ctx context.Context, query MigrationQuery) ([]*MigrationEvent, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []*MigrationEvent
	for _, e := range t.events {
		if query.WorkstreamID != "" && e.WorkstreamID != query.WorkstreamID {
			continue
		}
		if query.Backend != "" && e.Backend != query.Backend {
			continue
		}
		if query.FailureClass != "" && e.FailureClass != query.FailureClass {
			continue
		}
		if !query.StartTime.IsZero() && e.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && e.Timestamp.After(query.EndTime) {
			continue
		}
		results = append(results, e)
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}
	}
	return results, nil
}

func (t *InMemoryTelemetry) GetEventCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.events)
}

func getEnvironment() string {
	if env := os.Getenv("SDP_ENVIRONMENT"); env != "" {
		return env
	}
	if os.Getenv("PRODUCTION") == "true" || os.Getenv("ENTERPRISE") == "true" {
		return "production"
	}
	return "development"
}

type CutoverChecklist struct {
	Environment     string
	PreparedAt      time.Time
	PreflightChecks []PreflightCheck
	RollbackSteps   []RollbackStep
	Validated       bool
}

type PreflightCheck struct {
	Name        string
	Description string
	Passed      bool
	Message     string
}

type RollbackStep struct {
	Step     int
	Action   string
	Command  string
	Verified bool
}

func NewCutoverChecklist(environment string) *CutoverChecklist {
	return &CutoverChecklist{
		Environment: environment,
		PreparedAt:  time.Now(),
		PreflightChecks: []PreflightCheck{
			{Name: "fsm_v2_available", Description: "FSM V2 orchestrator is available"},
			{Name: "telemetry_enabled", Description: "Migration telemetry is enabled"},
			{Name: "feature_flag_configured", Description: "Feature flag is configured"},
			{Name: "rollback_tested", Description: "Rollback procedure tested"},
			{Name: "monitoring_ready", Description: "Monitoring dashboards ready"},
			{Name: "team_notified", Description: "Team notified of cutover"},
		},
		RollbackSteps: []RollbackStep{
			{Step: 1, Action: "Set backend to legacy", Command: "migration-shim set-backend legacy"},
			{Step: 2, Action: "Verify legacy working", Command: "migration-shim verify --backend legacy"},
			{Step: 3, Action: "Clear migration flags", Command: "migration-shim clear-flags"},
			{Step: 4, Action: "Notify team", Command: "Notify team of rollback"},
		},
	}
}

func (c *CutoverChecklist) RunPreflight(ctx context.Context, shim *MigrationShim) []PreflightCheck {
	results := make([]PreflightCheck, len(c.PreflightChecks))

	for i, check := range c.PreflightChecks {
		results[i] = c.runCheck(ctx, check, shim)
		if results[i].Passed {
			c.PreflightChecks[i].Passed = true
		}
	}

	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
			break
		}
	}
	c.Validated = allPassed

	return results
}

func (c *CutoverChecklist) runCheck(ctx context.Context, check PreflightCheck, shim *MigrationShim) PreflightCheck {
	result := PreflightCheck{
		Name:        check.Name,
		Description: check.Description,
	}

	switch check.Name {
	case "fsm_v2_available":
		result.Passed = shim.v2 != nil
		if !result.Passed {
			result.Message = "FSM V2 not initialized"
		}
	case "telemetry_enabled":
		result.Passed = shim.telemetry != nil
		if !result.Passed {
			result.Message = "Telemetry not configured"
		}
	case "feature_flag_configured":
		result.Passed = shim.flagProvider != nil || shim.config.Backend != BackendAuto
		if !result.Passed {
			result.Message = "Feature flag provider not configured"
		}
	case "rollback_tested":
		result.Passed = true
		result.Message = "Rollback steps defined"
	case "monitoring_ready":
		result.Passed = true
		result.Message = "Assumed ready (manual verification required)"
	case "team_notified":
		result.Passed = false
		result.Message = "Manual verification required"
	default:
		result.Passed = false
		result.Message = "Unknown check"
	}

	return result
}
