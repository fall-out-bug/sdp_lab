package strataudit

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/strataudit/model"
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
	if entities[0].TitleOriginal != "Market Leadership" {
		t.Errorf("title_original = %q", entities[0].TitleOriginal)
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

func TestBuildExtractionPrompt_RequiresSourceLanguagePreservation(t *testing.T) {
	cfg := &Config{EntityTypes: []string{"goal"}}
	level := model.Level{Name: "strategy", Rank: 0}

	prompt := buildExtractionPrompt(cfg, level, "Наша цель — лидерство на рынке", 0, 1)
	if !strings.Contains(prompt, "title_original") {
		t.Error("prompt should request title_original")
	}
	if !strings.Contains(prompt, "Do NOT translate") {
		t.Error("prompt should explicitly forbid translation")
	}
}

func TestAdmitEntityCandidate_PreservesRussianSourceLanguage(t *testing.T) {
	section := model.Section{
		ID:         "sec-1",
		DocumentID: "doc-1",
		CharStart:  0,
		CharEnd:    len([]rune("Наша цель — лидерство на рынке цифровых платежей.")),
		Content:    "Наша цель — лидерство на рынке цифровых платежей.",
	}
	entity := model.Entity{
		Type:                model.EntityGoal,
		TitleOriginal:       "Лидерство на рынке",
		DescriptionOriginal: "Стать лидером в цифровых платежах",
		SourceQuote:         "Наша цель — лидерство на рынке цифровых платежей.",
	}

	admitted, accepted := admitEntityCandidate(entity, section)
	if !accepted {
		t.Fatal("expected russian source entity to be accepted")
	}
	if admitted.Lang != "ru" {
		t.Fatalf("Lang = %q, want ru", admitted.Lang)
	}
	if admitted.LanguageMismatch {
		t.Fatal("LanguageMismatch = true, want false")
	}
	if admitted.Title != "Лидерство на рынке" {
		t.Fatalf("Title = %q", admitted.Title)
	}
	if admitted.TitleOriginal != "Лидерство на рынке" {
		t.Fatalf("TitleOriginal = %q", admitted.TitleOriginal)
	}
	if admitted.SectionID != "sec-1" {
		t.Fatalf("SectionID = %q, want sec-1", admitted.SectionID)
	}
	if admitted.QuoteStartOffset == nil || admitted.QuoteEndOffset == nil {
		t.Fatal("expected quote offsets to be populated")
	}
	if *admitted.QuoteStartOffset != 0 {
		t.Fatalf("QuoteStartOffset = %d, want 0", *admitted.QuoteStartOffset)
	}
}

func TestAdmitEntityCandidate_RejectsEnglishRewriteForRussianSource(t *testing.T) {
	section := model.Section{
		ID:      "sec-1",
		Content: "Наша цель — лидерство на рынке цифровых платежей.",
	}
	entity := model.Entity{
		Type:                model.EntityGoal,
		TitleOriginal:       "Market leadership",
		DescriptionOriginal: "Become the leader in digital payments",
		SourceQuote:         "Наша цель — лидерство на рынке цифровых платежей.",
	}

	admitted, accepted := admitEntityCandidate(entity, section)
	if accepted {
		t.Fatal("expected english rewrite on russian source to be rejected")
	}
	if !admitted.LanguageMismatch {
		t.Fatal("LanguageMismatch = false, want true")
	}
	if admitted.TrustGrade != model.TrustGradeRejected {
		t.Fatalf("TrustGrade = %q, want rejected", admitted.TrustGrade)
	}
	if len(admitted.QualityFlags) == 0 || admitted.QualityFlags[0] != "language_mismatch" {
		t.Fatalf("QualityFlags = %+v, want language_mismatch", admitted.QualityFlags)
	}
}

func TestAdmitEntityCandidate_AllowsEnglishSourceLanguage(t *testing.T) {
	section := model.Section{
		ID:      "sec-1",
		Content: "Our goal is market leadership in digital payments.",
	}
	entity := model.Entity{
		Type:                model.EntityGoal,
		TitleOriginal:       "Market leadership",
		DescriptionOriginal: "Become the leader in digital payments",
		SourceQuote:         "Our goal is market leadership in digital payments.",
	}

	admitted, accepted := admitEntityCandidate(entity, section)
	if !accepted {
		t.Fatal("expected english source entity to be accepted")
	}
	if admitted.Lang != "en" {
		t.Fatalf("Lang = %q, want en", admitted.Lang)
	}
	if admitted.LanguageMismatch {
		t.Fatal("LanguageMismatch = true, want false")
	}
}

func TestAdmitEntityCandidate_BindsAbsoluteQuoteOffsetsAcrossSectionWindow(t *testing.T) {
	content := "Введение.\n\nНаша цель — лидерство на рынке цифровых платежей.\n\nДальше идёт контекст."
	start := strings.Index(content, "Наша цель")
	if start < 0 {
		t.Fatal("test fixture broken: quote not found")
	}
	startRune := len([]rune(content[:start]))

	section := model.Section{
		ID:         "sec-2",
		DocumentID: "doc-1",
		CharStart:  startRune,
		CharEnd:    len([]rune(content)),
		Content:    content[start:],
	}
	entity := model.Entity{
		Type:                model.EntityGoal,
		TitleOriginal:       "Лидерство на рынке",
		DescriptionOriginal: "Стать лидером в цифровых платежах",
		SourceQuote:         "Наша цель — лидерство на рынке цифровых платежей.",
	}

	admitted, accepted := admitEntityCandidate(entity, section)
	if !accepted {
		t.Fatal("expected candidate to be accepted")
	}
	if admitted.QuoteStartOffset == nil || admitted.QuoteEndOffset == nil {
		t.Fatal("expected absolute quote offsets to be populated")
	}
	if *admitted.QuoteStartOffset != startRune {
		t.Fatalf("QuoteStartOffset = %d, want %d", *admitted.QuoteStartOffset, startRune)
	}
	if *admitted.QuoteEndOffset <= *admitted.QuoteStartOffset {
		t.Fatalf("invalid offsets: start=%d end=%d", *admitted.QuoteStartOffset, *admitted.QuoteEndOffset)
	}
}

func TestLocateQuoteSpan_NormalizedWhitespaceFallback(t *testing.T) {
	source := "Цель:\nлидерство   на рынке цифровых платежей."
	quote := "Цель: лидерство на рынке цифровых платежей."

	start, end, ok := locateQuoteSpan(source, quote)
	if !ok {
		t.Fatal("expected whitespace-normalized quote match")
	}
	if start != 0 {
		t.Fatalf("start = %d, want 0", start)
	}
	if end <= start {
		t.Fatalf("invalid span: start=%d end=%d", start, end)
	}
}
