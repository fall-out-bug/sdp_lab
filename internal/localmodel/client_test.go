package localmodel_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sdp_dev/internal/localmodel"
)

func TestNewClient_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     localmodel.Config
		wantErr bool
	}{
		{"empty BaseURL", localmodel.Config{Model: "qwen2.5-coder:7b"}, true},
		{"empty Model", localmodel.Config{BaseURL: "http://localhost:11434"}, true},
		{"valid", localmodel.Config{BaseURL: "http://localhost:11434", Model: "qwen2.5-coder:7b"}, false},
		{"BaseURL trailing slash trimmed", localmodel.Config{BaseURL: "http://localhost:11434/", Model: "m"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := localmodel.NewClient(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tc.wantErr, err)
			}
		})
	}
}

func TestClient_Prompt_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["stream"] != false {
			t.Errorf("stream must be false, got %v", req["stream"])
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "hello world"})
	}))
	defer srv.Close()

	c, err := localmodel.NewClient(localmodel.Config{BaseURL: srv.URL, Model: "test-model"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := c.Prompt(context.Background(), "write a hello world function")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestClient_Prompt_OllamaError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "model not found"})
	}))
	defer srv.Close()

	c, _ := localmodel.NewClient(localmodel.Config{BaseURL: srv.URL, Model: "missing"})
	_, err := c.Prompt(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for ollama error response, got nil")
	}
}

func TestClient_Prompt_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// block forever to simulate slow model
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, _ := localmodel.NewClient(localmodel.Config{BaseURL: srv.URL, Model: "slow"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := c.Prompt(ctx, "test")
	if err == nil {
		t.Fatal("expected error after context cancel, got nil")
	}
}

func TestClient_Model(t *testing.T) {
	c, _ := localmodel.NewClient(localmodel.Config{
		BaseURL: "http://localhost:11434", Model: "qwen2.5-coder:7b",
	})
	if c.Model() != "qwen2.5-coder:7b" {
		t.Errorf("Model() = %q, want %q", c.Model(), "qwen2.5-coder:7b")
	}
}
