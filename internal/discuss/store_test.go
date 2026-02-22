package discuss

import (
	"testing"
)

func TestNewOpenRouterAnalyzer(t *testing.T) {
	a := NewOpenRouterAnalyzer("")
	if a == nil || a.Model == "" {
		t.Errorf("NewOpenRouterAnalyzer: got %+v", a)
	}
	a2 := NewOpenRouterAnalyzer("custom/model")
	if a2.Model != "custom/model" {
		t.Errorf("NewOpenRouterAnalyzer(custom): model = %q", a2.Model)
	}
}

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
}

func TestStore_Create_Get(t *testing.T) {
	s := NewStore()
	req := DiscussRequest{Title: "Feature X", Description: "Do X"}
	sess, err := s.Create(req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.ID == "" {
		t.Error("Create: session ID empty")
	}
	if sess.Phase != PhaseCreated {
		t.Errorf("Create: phase = %q, want created", sess.Phase)
	}
	if sess.ProjectID != "default" {
		t.Errorf("Create: project_id = %q, want default", sess.ProjectID)
	}

	got, ok := s.Get(sess.ID)
	if !ok {
		t.Fatal("Get: session not found")
	}
	if got.ID != sess.ID || got.Title != req.Title {
		t.Errorf("Get: got %+v", got)
	}
}

func TestStore_Create_withProjectID(t *testing.T) {
	s := NewStore()
	req := DiscussRequest{ProjectID: "proj1", Title: "T", Description: "D"}
	sess, _ := s.Create(req)
	if sess.ProjectID != "proj1" {
		t.Errorf("Create with ProjectID: got %q", sess.ProjectID)
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore()
	sess, _ := s.Create(DiscussRequest{Title: "T", Description: "D"})
	sess.Phase = PhaseReady
	if err := s.Update(sess); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(sess.ID)
	if got.Phase != PhaseReady {
		t.Errorf("Update: phase = %q, want ready", got.Phase)
	}
}

func TestStore_Update_notFound(t *testing.T) {
	s := NewStore()
	sess, _ := s.Create(DiscussRequest{Title: "T", Description: "D"})
	sess.ID = "nonexistent"
	if err := s.Update(sess); err == nil {
		t.Error("Update(nonexistent): expected error")
	}
}

func TestStore_Get_missing(t *testing.T) {
	s := NewStore()
	_, ok := s.Get("no-such-id")
	if ok {
		t.Error("Get(missing): expected !ok")
	}
}
