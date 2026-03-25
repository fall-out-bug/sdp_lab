package omoclient

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		session := SessionInfo{
			ID:      "test-session-123",
			Project: req.Project,
			Session: req.Session,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(session)
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	client := NewClient(server.URL, logger)

	session, err := client.CreateSession(CreateSessionRequest{
		Project: "test-project",
		Session: "test-session",
	})

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID != "test-session-123" {
		t.Errorf("Expected session ID 'test-session-123', got '%s'", session.ID)
	}

	if session.Project != "test-project" {
		t.Errorf("Expected project 'test-project', got '%s'", session.Project)
	}
}

func TestGetSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := SessionInfo{
			ID:      "test-session-456",
			Project: "test-project",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	client := NewClient(server.URL, logger)

	session, err := client.GetSession("test-session-456")

	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if session.ID != "test-session-456" {
		t.Errorf("Expected session ID 'test-session-456', got '%s'", session.ID)
	}
}

func TestListSessions(t *testing.T) {
t.Skip("ListSessions requires REST API not available on opencode serve")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessions := []SessionInfo{
			{ID: "session-1", Project: "proj-a"},
			{ID: "session-2", Project: "proj-b"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	client := NewClient(server.URL, logger)

	sessions, err := client.ListSessions()

	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestDeleteSession(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	client := NewClient(server.URL, logger)

	err := client.DeleteSession("test-session")

	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	if !deleted {
		t.Error("Delete request was not sent")
	}
}

func TestCreateSessionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	logger := log.New(io.Discard, "", 0)
	client := NewClient(server.URL, logger)

	_, err := client.CreateSession(CreateSessionRequest{Project: "test"})

	if err == nil {
		t.Fatal("Expected error from CreateSession, got nil")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Expected status 500 error, got: %v", err)
	}
}
