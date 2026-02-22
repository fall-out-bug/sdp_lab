package review

import (
	"testing"
)

func TestNewPanel(t *testing.T) {
	p := NewPanel(PanelConfig{})
	if p == nil {
		t.Fatal("NewPanel returned nil")
	}
	if len(p.Personas) != 3 {
		t.Errorf("default personas: want 3, got %d", len(p.Personas))
	}
}

func TestNewPanel_customPersonas(t *testing.T) {
	p := NewPanel(PanelConfig{Personas: []string{"security", "perf"}})
	if p == nil || len(p.Personas) != 2 {
		t.Errorf("custom personas: %+v", p)
	}
}
