package a2a

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/control"
)

func setupStore(t *testing.T) *control.Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := []byte("projects:\n  - id: openclaw\n    repo_url: https://github.com/openclaw/openclaw\n    beads_prefix: openclaw\n")
	if err := os.WriteFile(filepath.Join(root, "docs", "specs", "project-registry.yaml"), registry, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := control.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestAgentCardEndpoint(t *testing.T) {
	store := setupStore(t)
	ts := httptest.NewServer(&Server{Store: store, Addr: ":8080"})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/.well-known/agent.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var card map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatal(err)
	}
	if card["name"] != "sdp-control-tower" {
		t.Fatalf("name = %v", card["name"])
	}
	if card["url"] != "http://localhost:8080/a2a" {
		t.Fatalf("url = %v", card["url"])
	}
}

func TestTasksSendCreatesCardAndTasksGetReturnsStatus(t *testing.T) {
	store := setupStore(t)
	oldCreate := control.MockCreateBeadsIssue("BEADS-123")
	control.SetCreateBeadsIssueFn(oldCreate)
	defer control.SetCreateBeadsIssueFn(control.MockCreateBeadsIssue(""))

	ts := httptest.NewServer(&Server{Store: store, Addr: ":8080"})
	defer ts.Close()

	sendReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tasks/send",
		"params": map[string]any{
			"title":       "Ship thin slice",
			"raw_request": "implement thin vertical slice",
			"project_id":  "openclaw",
		},
	}
	var sendResp rpcResponse
	postRPC(t, ts.URL+"/a2a", sendReq, "", &sendResp)
	if sendResp.Error != nil {
		t.Fatalf("tasks/send error = %+v", sendResp.Error)
	}
	result := sendResp.Result.(map[string]any)
	if result["task_id"] == "" {
		t.Fatalf("task_id missing: %+v", result)
	}
	if result["status"] != "executing" {
		t.Fatalf("status = %v", result["status"])
	}

	getReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tasks/get",
		"params": map[string]any{
			"task_id":    result["task_id"],
			"project_id": "openclaw",
		},
	}
	var getResp rpcResponse
	postRPC(t, ts.URL+"/a2a", getReq, "", &getResp)
	if getResp.Error != nil {
		t.Fatalf("tasks/get error = %+v", getResp.Error)
	}
	getResult := getResp.Result.(map[string]any)
	if getResult["status"] != "executing" {
		t.Fatalf("get status = %v", getResult["status"])
	}
}

func TestInvalidMethodReturnsJSONRPCError(t *testing.T) {
	store := setupStore(t)
	ts := httptest.NewServer(&Server{Store: store})
	defer ts.Close()

	req := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tasks/boom", "params": map[string]any{}}
	var resp rpcResponse
	postRPC(t, ts.URL+"/a2a", req, "", &resp)
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32600 {
		t.Fatalf("error code = %d", resp.Error.Code)
	}
}

func TestAuthRequiredReturns401(t *testing.T) {
	store := setupStore(t)
	ts := httptest.NewServer(&Server{Store: store, APIKey: "secret"})
	defer ts.Close()

	reqBody := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tasks/list", "params": map[string]any{"project_id": "openclaw"}}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(ts.URL+"/a2a", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func postRPC(t *testing.T, url string, payload map[string]any, auth string, out *rpcResponse) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}
