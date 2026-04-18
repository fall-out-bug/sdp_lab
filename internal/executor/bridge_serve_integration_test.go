//go:build integration

package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/agentloop/livegw"
	"sdp_dev/internal/control"
)

// testPhaseMap is a custom phase map for integration tests. It uses "glm-5"
// which is in LiveGateway's allowedModels allowlist, avoiding "no available model"
// errors when the test uses a LiveGateway pointed at httptest.Server.
var testPhaseMap = map[agentloop.Role]agentloop.PhaseConfig{
	agentloop.RoleDiscover: {
		Models:          []string{"glm-5"},
		Tools:           []string{"read_file"},
		AllowedNext:     []agentloop.Role{agentloop.RolePlan},
		RecoveryNext:    []agentloop.Role{agentloop.RoleDiscover},
		GateRequired:    true,
		MinOutputTokens: 50,
	},
	agentloop.RolePlan: {
		Models:       []string{"glm-5"},
		Tools:        []string{"read_file"},
		AllowedNext:  []agentloop.Role{agentloop.RoleBuild},
		RecoveryNext: []agentloop.Role{agentloop.RoleDiscover},
		GateRequired: true,
	},
	agentloop.RoleBuild: {
		Models:       []string{"glm-5"},
		Tools:        []string{"read_file", "edit_file", "bash"},
		AllowedNext:  []agentloop.Role{agentloop.RoleReview},
		RecoveryNext: []agentloop.Role{agentloop.RolePlan},
		GateRequired: true,
	},
	agentloop.RoleReview: {
		Models:       []string{"glm-5"},
		Tools:        []string{"read_file"},
		AllowedNext:  []agentloop.Role{agentloop.RoleEval},
		RecoveryNext: []agentloop.Role{agentloop.RoleBuild},
		GateRequired: true,
	},
	agentloop.RoleEval: {
		Models:       []string{"glm-5"},
		Tools:        []string{"bash", "read_file"},
		AllowedNext:  []agentloop.Role{},
		RecoveryNext: []agentloop.Role{agentloop.RoleBuild},
		GateRequired: true,
	},
}

// TestServeBridgeHarnessE2E proves the full chain:
//
//	ServeBridge.DispatchAndRun -> Harness -> LiveGateway -> httptest.Server (LLM)
//	-> SSE parsing -> event draining -> gate evaluation -> evidence accumulated.
//
// No real API key or network access required. The httptest.Server returns minimal
// valid OpenRouter-compatible SSE responses. Deterministic, CI-safe.
func TestServeBridgeHarnessE2E(t *testing.T) {
	// --- 1. Stand up fake LLM server returning SSE ---
	content := "Hello world. Discovery complete."
	sseResp := buildSSETextResponse(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		fmt.Fprint(w, sseResp)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	// --- 2. Create LiveGateway pointed at httptest.Server ---
	gw, err := livegw.New("test-api-key", srv.URL)
	if err != nil {
		t.Fatalf("livegw.New: %v", err)
	}

	// --- 3. Create control.Store with test card ---
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "E2E integration test", "discover the codebase")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	card.NormalizedIntent = "discover the codebase"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	// --- 4. Build ServeBridge with harness components wired to fake LLM ---
	// Uses testPhaseMap with "glm-5" (allowed by LiveGateway) instead of
	// DefaultPhaseMap which lists models not in the allowlist.
	projectRoot := store.ProjectRoot
	tools := agentloop.BuildLiveTools(projectRoot, store)
	registry := agentloop.NewToolRegistry(tools)
	router := agentloop.NewPhaseRouter(
		testPhaseMap, registry, gw, nil,
	)
	gate := agentloop.NewGateEngine(nil, 0)
	harnessData := filepath.Join(t.TempDir(), "sessions")

	sb := &ServeBridge{
		Store:         store,
		ProjectRoot:   projectRoot,
		harnessRouter: router,
		harnessGate:   gate,
		harnessData:   harnessData,
	}

	// --- 5. Execute DispatchAndRun with timeout ---
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sb.DispatchAndRun(ctx, card.ProjectID, card.ID)

	// --- 6. Assert result status ---
	if err != nil {
		t.Fatalf("DispatchAndRun error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Status == control.ResultStatusFailed {
		t.Fatalf("result status = %s (failed); summary: %s", result.Status, result.Summary)
	}
	if result.ParentFeatureID != card.ID {
		t.Errorf("ParentFeatureID = %s, want %s", result.ParentFeatureID, card.ID)
	}

	// --- 7. Assert session DB file created ---
	dbPath := filepath.Join(harnessData, card.ID+".db")
	if _, statErr := os.Stat(dbPath); statErr != nil {
		t.Fatalf("expected session DB at %s: %v", dbPath, statErr)
	}

	// --- 8. Assert session contains at least one TurnRecord ---
	sqliteStore, openErr := agentloop.NewSQLiteStore(dbPath)
	if openErr != nil {
		t.Fatalf("open sqlite store: %v", openErr)
	}
	defer sqliteStore.Close()

	turns, loadErr := sqliteStore.LoadTurnRecords(card.ID)
	if loadErr != nil {
		t.Fatalf("load turn records: %v", loadErr)
	}
	if len(turns) == 0 {
		t.Fatal("expected at least one TurnRecord in session, got 0")
	}

	// Verify the turn record contains the text from our fake LLM response.
	first := turns[0]
	if first.AssistantText == "" {
		t.Error("TurnRecord.AssistantText is empty — expected text from fake LLM response")
	}
	t.Logf("TurnRecord: phase=%s assistant_text_len=%d user_msg=%q",
		first.Phase, len(first.AssistantText), first.UserMsg.Content)
}

// TestServeBridgeHarnessRealLLM is an optional integration test that uses a real LLM.
// It is skipped unless OPENROUTER_API_KEY is set. Run with:
//
//	OPENROUTER_API_KEY=sk-xxx go test -tags integration -run TestServeBridgeHarnessRealLLM -v ./internal/executor/
func TestServeBridgeHarnessRealLLM(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set — skipping real LLM integration test")
	}

	baseURL := os.Getenv("OPENROUTER_BASE_URL")

	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Real LLM integration test", "list files in the project")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	card.NormalizedIntent = "list files in the project"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	gw, err := livegw.New(apiKey, baseURL)
	if err != nil {
		t.Fatalf("livegw.New: %v", err)
	}

	projectRoot := store.ProjectRoot
	tools := agentloop.BuildLiveTools(projectRoot, store)
	registry := agentloop.NewToolRegistry(tools)
	router := agentloop.NewPhaseRouter(
		agentloop.DefaultPhaseMap, registry, gw, nil,
	)
	gate := agentloop.NewGateEngine(nil, 0)
	harnessData := filepath.Join(t.TempDir(), "sessions")

	sb := &ServeBridge{
		Store:         store,
		ProjectRoot:   projectRoot,
		harnessRouter: router,
		harnessGate:   gate,
		harnessData:   harnessData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := sb.DispatchAndRun(ctx, card.ProjectID, card.ID)
	if err != nil {
		t.Fatalf("DispatchAndRun error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	t.Logf("status=%s summary=%s", result.Status, result.Summary)

	dbPath := filepath.Join(harnessData, card.ID+".db")
	if _, statErr := os.Stat(dbPath); statErr != nil {
		t.Fatalf("expected session DB at %s: %v", dbPath, statErr)
	}
}

// buildSSETextResponse builds a valid OpenRouter-compatible SSE stream that
// sends text content and finishes. The format matches what llmclient.parseSSE
// expects: lines prefixed with "data: " followed by JSON chunks, ending with
// "data: [DONE]".
func buildSSETextResponse(text string) string {
	// First chunk: role announcement (empty content, sets role).
	roleChunk := map[string]any{
		"choices": []map[string]any{{
			"delta": map[string]any{
				"role": "assistant",
			},
		}},
	}

	// Content chunk: the actual text.
	contentChunk := map[string]any{
		"choices": []map[string]any{{
			"delta": map[string]any{
				"content": text,
			},
		}},
	}

	// Finish chunk: signals the stream is done.
	finishChunk := map[string]any{
		"choices": []map[string]any{{
			"delta":       map[string]any{},
			"finish_reason": "stop",
		}},
	}

	roleJSON, _ := json.Marshal(roleChunk)
	contentJSON, _ := json.Marshal(contentChunk)
	finishJSON, _ := json.Marshal(finishChunk)

	return fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: [DONE]\n\n",
		string(roleJSON), string(contentJSON), string(finishJSON))
}
