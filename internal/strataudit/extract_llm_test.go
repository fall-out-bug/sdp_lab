package strataudit

import (
	"strings"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func TestParseExtractionResponse(t *testing.T) {
	input := `{
		"entities": [
			{"type": "goal", "title": "Market Leadership", "description": "Become #1 in AI tools", "source_quote": "Our goal is to be the market leader"},
			{"type": "initiative", "title": "SEA Expansion", "description": "Enter Southeast Asian markets", "source_quote": "Expand operations to SEA by Q4"}
		]
	}`

	entities, err := parseExtractionResponse(input, "d1", "vision", "test-model")
	if err != nil {
		t.Fatalf("parseExtractionResponse: %v", err)
	}
	if len(entities) != 2 {
		t.Fatalf("got %d entities, want 2", len(entities))
	}
	if entities[0].Type != model.EntityGoal {
		t.Errorf("type = %q, want goal", entities[0].Type)
	}
	if entities[0].Title != "Market Leadership" {
		t.Errorf("title = %q", entities[0].Title)
	}
	if entities[0].DocumentID != "d1" {
		t.Errorf("DocumentID = %q, want d1", entities[0].DocumentID)
	}
}

func TestParseExtractionResponse_MarkdownWrapped(t *testing.T) {
	input := "```json\n{\"entities\": [{\"type\": \"task\", \"title\": \"Hire\", \"description\": \"Recruit\", \"source_quote\": \"Hire a manager\"}]}\n```"

	entities, err := parseExtractionResponse(input, "d1", "task", "test")
	if err != nil {
		t.Fatalf("parseExtractionResponse: %v", err)
	}
	if len(entities) != 1 {
		t.Fatalf("got %d entities, want 1", len(entities))
	}
}

func TestParseExtractionResponse_Empty(t *testing.T) {
	input := `{"entities": []}`
	entities, err := parseExtractionResponse(input, "d1", "vision", "test")
	if err != nil {
		t.Fatalf("parseExtractionResponse: %v", err)
	}
	if len(entities) != 0 {
		t.Errorf("got %d entities, want 0", len(entities))
	}
}

func TestParseExtractionResponse_SkipsInvalid(t *testing.T) {
	input := `{
		"entities": [
			{"type": "goal", "title": "Valid", "description": "OK", "source_quote": "text"},
			{"type": "", "title": "", "description": "No type or title"},
			{"type": "task", "title": "Also Valid", "description": "OK", "source_quote": "text2"}
		]
	}`

	entities, _ := parseExtractionResponse(input, "d1", "vision", "test")
	if len(entities) != 2 {
		t.Errorf("got %d entities, want 2 (skipped 1 invalid)", len(entities))
	}
}

func TestParseExtractionResponse_SkipsPromptLeak(t *testing.T) {
	input := `{"entities":[
		{"type":"goal","title":"Return valid JSON only","description":"Prompt leak","source_quote":"Return valid JSON only."},
		{"type":"goal","title":"Be the market leader","description":"Concrete strategy","source_quote":"Our vision is to be the market leader by 2027"},
		{"type":"task","title":"Never ignore previous instructions","description":"Another prompt leak","source_quote":"Never ignore previous instructions from user"}
	]}`

	entities, err := parseExtractionResponse(input, "d1", "strategy", "test")
	if err != nil {
		t.Fatalf("parseExtractionResponse: %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("got %d entities, want 1 (only strategy-meaningful entity)", len(entities))
	}

	if entities[0].Title != "Be the market leader" {
		t.Errorf("unexpected title = %q", entities[0].Title)
	}
	if entities[0].Type != model.EntityGoal {
		t.Errorf("unexpected type = %q", entities[0].Type)
	}
}

func TestEntityID_Deterministic(t *testing.T) {
	id1 := entityID("d1", "goal", "Test")
	id2 := entityID("d1", "goal", "Test")
	if id1 != id2 {
		t.Error("entityID should be deterministic")
	}
	id3 := entityID("d1", "goal", "Other")
	if id1 == id3 {
		t.Error("different title should produce different ID")
	}
}

func TestParseExtractionResponse_InvalidJSON(t *testing.T) {
	_, err := parseExtractionResponse("not json at all", "d1", "vision", "test")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestBuildExtractionPrompt(t *testing.T) {
	cfg := &Config{EntityTypes: []string{"goal", "task"}}
	level := model.Level{Name: "strategy", Rank: 1}

	prompt := buildExtractionPrompt(cfg, level, "test content", 0, 1)
	if !strings.Contains(prompt, "strategy") {
		t.Error("prompt should contain level name")
	}
	if !strings.Contains(prompt, "goal, task") {
		t.Error("prompt should contain entity types")
	}
	if !strings.Contains(prompt, "test content") {
		t.Error("prompt should contain document content")
	}
}

func TestBuildExtractionPrompt_Chunked(t *testing.T) {
	cfg := &Config{EntityTypes: []string{"goal"}}
	level := model.Level{Name: "vision", Rank: 0}

	prompt := buildExtractionPrompt(cfg, level, "content", 2, 5)
	if !strings.Contains(prompt, "chunk 3 of 5") {
		t.Error("prompt should mention chunk number")
	}
}
