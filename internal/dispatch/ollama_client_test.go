package dispatch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

func TestNewOllamaClient_Defaults(t *testing.T) {
	client := dispatch.NewOllamaClient("")
	if client.BaseURL != "http://localhost:11434" {
		t.Errorf("expected default base URL http://localhost:11434, got %q", client.BaseURL)
	}
	if client.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", client.HTTPClient.Timeout)
	}
}

func TestNewOllamaClient_CustomBaseURL(t *testing.T) {
	client := dispatch.NewOllamaClient("http://example.com:8080")
	if client.BaseURL != "http://example.com:8080" {
		t.Errorf("expected custom base URL, got %q", client.BaseURL)
	}
}

func TestOllamaClient_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestOllamaClient_HealthCheck_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Error("HealthCheck should return error for non-200 status")
	}
}

func TestOllamaClient_HealthCheck_ConnectionError(t *testing.T) {
	client := dispatch.NewOllamaClient("http://localhost:9999") // assume nothing is listening
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Error("HealthCheck should return error for connection failure")
	}
}

func TestOllamaClient_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}

		if payload["model"] != "test-model" {
			t.Errorf("expected model test-model, got %v", payload["model"])
		}
		if payload["prompt"] != "test prompt" {
			t.Errorf("expected prompt 'test prompt', got %v", payload["prompt"])
		}
		if payload["stream"] != false {
			t.Errorf("expected stream false, got %v", payload["stream"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "generated response",
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	response, err := client.Generate(context.Background(), "test-model", "test prompt")
	if err != nil {
		t.Errorf("Generate failed: %v", err)
	}
	if response != "generated response" {
		t.Errorf("expected 'generated response', got %q", response)
	}
}

func TestOllamaClient_Generate_EmptyModel(t *testing.T) {
	client := dispatch.NewOllamaClient("http://localhost:11434")
	_, err := client.Generate(context.Background(), "", "test prompt")
	if err == nil {
		t.Error("Generate should return error for empty model")
	}
}

func TestOllamaClient_Generate_EmptyPrompt(t *testing.T) {
	client := dispatch.NewOllamaClient("http://localhost:11434")
	_, err := client.Generate(context.Background(), "test-model", "")
	if err == nil {
		t.Error("Generate should return error for empty prompt")
	}
}

func TestOllamaClient_Generate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "model not found",
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	_, err := client.Generate(context.Background(), "test-model", "test prompt")
	if err == nil {
		t.Error("Generate should return error for API error")
	}
}

func TestOllamaClient_Generate_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "response",
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	client.HTTPClient.Timeout = 10 * time.Millisecond
	_, err := client.Generate(context.Background(), "test-model", "test prompt")
	if err == nil {
		t.Error("Generate should return error for timeout")
	}
}

func TestOllamaClient_ListModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "qwen2.5-coder:7b"},
				{"name": "codegemma:7b"},
			},
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Errorf("ListModels failed: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
	if models[0] != "qwen2.5-coder:7b" {
		t.Errorf("expected first model qwen2.5-coder:7b, got %q", models[0])
	}
	if models[1] != "codegemma:7b" {
		t.Errorf("expected second model codegemma:7b, got %q", models[1])
	}
}

func TestOllamaClient_ListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
		})
	}))
	defer server.Close()

	client := dispatch.NewOllamaClient(server.URL)
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Errorf("ListModels failed: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestOllamaClient_ListModels_ConnectionError(t *testing.T) {
	client := dispatch.NewOllamaClient("http://localhost:9999")
	_, err := client.ListModels(context.Background())
	if err == nil {
		t.Error("ListModels should return error for connection failure")
	}
}
