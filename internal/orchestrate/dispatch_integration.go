package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/kernel"
)

// NewDispatchingInvoker creates a DispatchingInvoker if profiles exist.
// Returns nil if no profiles found (caller should use DefaultLLMInvoker).
func NewDispatchingInvoker(projectRoot string) *dispatch.DispatchingInvoker {
	store := dispatch.NewProfileStore(projectRoot)
	profiles, err := store.LoadAll()
	if err != nil || len(profiles) == 0 {
		slog.Info("dispatch: no profiles, using default invoker")
		return nil
	}

	// Check for local model configuration from environment variables
	var localCfg *dispatch.LocalConfig
	if os.Getenv("SDP_LOCAL_ENABLED") == "true" {
		baseURL := os.Getenv("OLLAMA_HOST")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		model := os.Getenv("SDP_LOCAL_MODEL")
		if model == "" {
			model = "qwen2.5-coder:7b"
		}
		localCfg = &dispatch.LocalConfig{
			BaseURL: baseURL,
			Model:   model,
			Score:   0.9,
		}
		slog.Info("dispatch: local model routing enabled",
			"base_url", baseURL, "model", model)

		// Perform health check
		client := dispatch.NewOllamaClient(baseURL)
		if err := client.HealthCheck(context.Background()); err != nil {
			slog.Warn("dispatch: ollama health check failed, falling back to cloud only",
				"error", err)
			localCfg = nil
		}
	}

	// Build harness registry
	reg := harness.NewRegistry()
	reg.Register(harness.NewClaudeHarness())
	reg.Register(harness.NewCodexHarness())
	reg.Register(harness.NewCursorHarness())
	reg.Register(harness.NewOpenCodeHarness())

	// Build invoker map — adapt harness spawning to LLMInvoker interface
	// For each available harness, create a function-based invoker
	// that spawns the harness CLI
	invokerMap := map[string]dispatch.LLMInvoker{}
	for _, h := range reg.Available() {
		name := h.Name()
		switch name {
		case "opencode":
			// Reuse existing opencode invocation (it's the current default)
			invokerMap[name] = GetDefaultInvoker()
		default:
			// For other harnesses, create spawn-based invokers
			// that launch their CLI and capture output
			hRef := h // capture loop variable
			invokerMap[name] = &harnessInvoker{harness: hRef}
		}
	}

	router := &dispatch.Router{
		Profiles:    profiles,
		LocalConfig: localCfg,
	}
	verifyRouter := &dispatch.VerificationRouter{Profiles: profiles}

	return &dispatch.DispatchingInvoker{
		Router:   router,
		Fallback: GetDefaultInvoker(),
		InvokerFor: func(name string) dispatch.LLMInvoker {
			return invokerMap[name]
		},
		PacketLoader: func(root string) (dispatch.ContextPacketSummary, error) {
			return loadPacketSummary(root)
		},
		ContextEnricher: buildPromptWithContext,
		VerifyHarness: func(buildDec *dispatch.DispatchDecision, task dispatch.TaskClassification) (*dispatch.DispatchDecision, error) {
			return verifyRouter.RouteVerification(context.Background(), task, buildDec.Harness, nil)
		},
	}
}

// toWSDispatchInfo converts a dispatch.DispatchDecision to the local WSDispatchInfo struct.
func toWSDispatchInfo(dec *dispatch.DispatchDecision) *WSDispatchInfo {
	if dec == nil {
		return nil
	}
	return &WSDispatchInfo{
		Harness:   dec.Harness,
		Provider:  dec.Provider,
		Model:     dec.Model,
		Score:     dec.Score,
		Reason:    dec.Reason,
		Timestamp: dec.Timestamp,
	}
}

// RecordDispatch finds the WSStatus with the given wsID in the checkpoint
// and sets its Dispatch field from the given DispatchDecision.
// Returns true if the workstream was found and updated, false otherwise.
func RecordDispatch(cp *Checkpoint, wsID string, dec *dispatch.DispatchDecision) bool {
	for i := range cp.Workstreams {
		if cp.Workstreams[i].ID == wsID {
			cp.Workstreams[i].Dispatch = toWSDispatchInfo(dec)
			return true
		}
	}
	return false
}

// harnessInvoker adapts a Harness to the LLMInvoker interface.
type harnessInvoker struct {
	harness harness.Harness
}

func (h *harnessInvoker) Invoke(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	proc, err := h.harness.Spawn(ctx, harness.SpawnOpts{
		Worktree: req.WorkDir,
		Prompt:   req.Prompt,
		Agent:    req.Agent,
	})
	if err != nil {
		return kernel.RuntimeResult{ExitCode: -1}, fmt.Errorf("spawn %s: %w", h.harness.Name(), err)
	}

	result := <-proc.Done
	return kernel.RuntimeResult{Output: result.Output, ExitCode: result.ExitCode}, nil
}

// loadPacketSummary reads .sdp/context-packet.json and extracts fields needed for classification.
func loadPacketSummary(projectRoot string) (dispatch.ContextPacketSummary, error) {
	path := filepath.Join(projectRoot, ".sdp", "context-packet.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return dispatch.ContextPacketSummary{}, fmt.Errorf("read context packet: %w", err)
	}

	var raw struct {
		Workstream string   `json:"workstream"`
		ScopeFiles []string `json:"scope_files"`
		Checkpoint *struct {
			Phase string `json:"phase"`
		} `json:"checkpoint"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return dispatch.ContextPacketSummary{}, fmt.Errorf("parse context packet: %w", err)
	}

	phase := "build"
	if raw.Checkpoint != nil && raw.Checkpoint.Phase != "" {
		phase = raw.Checkpoint.Phase
	}

	return dispatch.ContextPacketSummary{
		Phase:      phase,
		Workstream: raw.Workstream,
		ScopeFiles: raw.ScopeFiles,
	}, nil
}
