package agent

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// HookKind identifies when a hook runs.
type HookKind string

const (
	HookPreExecute  HookKind = "pre-execute"
	HookPostExecute HookKind = "post-execute"
	HookPrePublish  HookKind = "pre-publish"
	HookPostReview  HookKind = "post-review"
)

// HookFunc is called at lifecycle points. Return error to abort.
type HookFunc func(ctx context.Context, data HookData) error

// HookData provides context to hooks.
type HookData struct {
	IssueID       string
	RunID         string
	Role          string
	WorkDir       string
	ChangedFiles  []string
	ResultSummary string
}

// HookRegistry manages extensible hooks per role.
type HookRegistry struct {
	role  string
	hooks map[HookKind][]HookFunc
}

// NewHookRegistry creates a HookRegistry for the role.
func NewHookRegistry(role string) *HookRegistry {
	r := &HookRegistry{
		role:  role,
		hooks: make(map[HookKind][]HookFunc),
	}
	r.registerDefaults()
	return r
}

func (r *HookRegistry) registerDefaults() {
	r.Register(HookPreExecute, r.boundaryCheck)
	r.Register(HookPreExecute, r.workspaceCleanCheck)
	r.Register(HookPostExecute, r.boundaryRevalidate)
	r.Register(HookPostExecute, r.goTestCheck)
}

// Register adds a hook for the given kind.
func (r *HookRegistry) Register(kind HookKind, fn HookFunc) {
	r.hooks[kind] = append(r.hooks[kind], fn)
}

// RunPreExecute runs all pre-execute hooks.
func (r *HookRegistry) RunPreExecute(ctx context.Context, data HookData) error {
	return r.run(HookPreExecute, ctx, data)
}

// RunPostExecute runs all post-execute hooks.
func (r *HookRegistry) RunPostExecute(ctx context.Context, data HookData) error {
	return r.run(HookPostExecute, ctx, data)
}

// RunPrePublish runs all pre-publish hooks.
func (r *HookRegistry) RunPrePublish(ctx context.Context, data HookData) error {
	return r.run(HookPrePublish, ctx, data)
}

// RunPostReview runs all post-review hooks.
func (r *HookRegistry) RunPostReview(ctx context.Context, data HookData) error {
	return r.run(HookPostReview, ctx, data)
}

func (r *HookRegistry) run(kind HookKind, ctx context.Context, data HookData) error {
	for _, fn := range r.hooks[kind] {
		if err := fn(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

func (r *HookRegistry) boundaryCheck(ctx context.Context, data HookData) error {
	// Placeholder: actual boundary validation happens in executor
	return nil
}

func (r *HookRegistry) workspaceCleanCheck(ctx context.Context, data HookData) error {
	if data.WorkDir == "" {
		return nil
	}
	gitDir := filepath.Join(data.WorkDir, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		return nil
	}
	// No .git is ok for some flows
	return nil
}

func (r *HookRegistry) boundaryRevalidate(ctx context.Context, data HookData) error {
	// Placeholder: re-validation of changed paths
	return nil
}

func (r *HookRegistry) goTestCheck(ctx context.Context, data HookData) error {
	// Placeholder: run go test ./... - actual execution in executor
	return nil
}

// LoadFromFile loads hooks from specs/agent-hooks.yaml (optional).
func (r *HookRegistry) LoadFromFile(workDir string) error {
	path := filepath.Join(workDir, "specs", "agent-hooks.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var cfg struct {
		Roles map[string]struct {
			PreExecute  []string `yaml:"pre_execute"`
			PostExecute []string `yaml:"post_execute"`
			PrePublish  []string `yaml:"pre_publish"`
			PostReview  []string `yaml:"post_review"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	roleCfg, ok := cfg.Roles[r.role]
	if !ok {
		return nil
	}
	// Named hooks are registered by name; we'd need a hook factory.
	// For now, we only use built-in hooks. Extensible via Register().
	_ = roleCfg
	return nil
}
