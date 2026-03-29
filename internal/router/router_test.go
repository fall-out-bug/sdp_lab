package router

import (
	"testing"
)

var testProjects = []ProjectConfig{
	{
		Name:       "sdp_lab",
		Root:       "/home/user/projects/sdp_lab",
		Languages:  []string{"go"},
		DefaultRig: "sdp-full",
		Patterns:   []string{"sdp", "orchestrate", "dispatch", "tower"},
	},
	{
		Name:       "webapp",
		Root:       "/home/user/projects/webapp",
		Languages:  []string{"typescript", "react"},
		DefaultRig: "sdp-lite",
		Patterns:   []string{"webapp", "frontend", "ui", "dashboard"},
	},
}

func newTestRouter() *Router {
	return &Router{Projects: testProjects}
}

func TestRouter_Route_Feature(t *testing.T) {
	r := newTestRouter()
	intent := Intent{
		ID:       "feat-001",
		Source:   "beads",
		Title:    "Add tower orchestration support",
		TaskType: "feature",
		Labels:   []string{"tower", "sdp"},
	}

	decision, err := r.Route(intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.ProjectRoot != "/home/user/projects/sdp_lab" {
		t.Errorf("expected project root /home/user/projects/sdp_lab, got %s", decision.ProjectRoot)
	}
	if decision.Rig != "sdp-full" {
		t.Errorf("expected rig sdp-full, got %s", decision.Rig)
	}
	if decision.EntryPhase != "discovery" {
		t.Errorf("expected entry phase discovery, got %s", decision.EntryPhase)
	}
}

func TestRouter_Route_Hotfix(t *testing.T) {
	r := newTestRouter()
	intent := Intent{
		ID:       "hot-001",
		Source:   "manual",
		Title:    "Fix crash in dispatch loop",
		TaskType: "hotfix",
		Labels:   []string{"dispatch"},
	}

	decision, err := r.Route(intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.EntryPhase != "build" {
		t.Errorf("expected entry phase build, got %s", decision.EntryPhase)
	}
	if decision.Rig != "manual" {
		t.Errorf("expected rig manual, got %s", decision.Rig)
	}
}

func TestRouter_Route_Bugfix(t *testing.T) {
	r := newTestRouter()
	intent := Intent{
		ID:       "bug-001",
		Source:   "beads",
		Title:    "Fix webapp login redirect",
		TaskType: "bugfix",
		Labels:   []string{"webapp", "auth"},
	}

	decision, err := r.Route(intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.ProjectRoot != "/home/user/projects/webapp" {
		t.Errorf("expected project root /home/user/projects/webapp, got %s", decision.ProjectRoot)
	}
	if decision.EntryPhase != "build" {
		t.Errorf("expected entry phase build, got %s", decision.EntryPhase)
	}
	if decision.Rig != "sdp-lite" {
		t.Errorf("expected rig sdp-lite, got %s", decision.Rig)
	}
}

func TestRouter_Route_Refactor(t *testing.T) {
	r := newTestRouter()
	intent := Intent{
		ID:       "ref-001",
		Source:   "kanban",
		Title:    "Refactor orchestrate module",
		TaskType: "refactor",
		Labels:   []string{"orchestrate"},
	}

	decision, err := r.Route(intent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.EntryPhase != "design" {
		t.Errorf("expected entry phase design, got %s", decision.EntryPhase)
	}
}

func TestRouter_Route_UnknownProject(t *testing.T) {
	r := newTestRouter()
	intent := Intent{
		ID:       "unk-001",
		Source:   "manual",
		Title:    "Do something for unknown-project",
		TaskType: "feature",
		Labels:   []string{"unknown-project"},
	}

	_, err := r.Route(intent)
	if err == nil {
		t.Fatal("expected error for unknown project, got nil")
	}
}

func TestInferEntryPhase(t *testing.T) {
	tests := []struct {
		name            string
		taskType        string
		hasRequirements bool
		hasDesign       bool
		want            string
	}{
		{"hotfix goes to build", "hotfix", false, false, "build"},
		{"bugfix goes to build", "bugfix", false, false, "build"},
		{"feature with design goes to build", "feature", true, true, "build"},
		{"feature with requirements goes to design", "feature", true, false, "design"},
		{"feature without anything goes to discovery", "feature", false, false, "discovery"},
		{"refactor goes to design", "refactor", false, false, "design"},
		{"unknown goes to discovery", "", false, false, "discovery"},
		{"unknown type goes to discovery", "something-else", false, false, "discovery"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferEntryPhase(tt.taskType, tt.hasRequirements, tt.hasDesign)
			if got != tt.want {
				t.Errorf("inferEntryPhase(%q, %v, %v) = %q, want %q",
					tt.taskType, tt.hasRequirements, tt.hasDesign, got, tt.want)
			}
		})
	}
}

func TestSelectRig(t *testing.T) {
	tests := []struct {
		name           string
		taskType       string
		projectDefault string
		want           string
	}{
		{"hotfix gets manual", "hotfix", "sdp-full", "manual"},
		{"bugfix gets sdp-lite", "bugfix", "sdp-full", "sdp-lite"},
		{"feature uses project default", "feature", "sdp-full", "sdp-full"},
		{"feature uses project default lite", "feature", "sdp-lite", "sdp-lite"},
		{"refactor uses project default", "refactor", "sdp-full", "sdp-full"},
		{"unknown type uses project default", "", "sdp-lite", "sdp-lite"},
		{"empty project default falls back to sdp-full", "feature", "", "sdp-full"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectRig(tt.taskType, tt.projectDefault)
			if got != tt.want {
				t.Errorf("selectRig(%q, %q) = %q, want %q",
					tt.taskType, tt.projectDefault, got, tt.want)
			}
		})
	}
}
