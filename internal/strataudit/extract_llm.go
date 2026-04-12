package strataudit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sdp_dev/internal/strataudit/model"
	"strings"
	"unicode"
)

// xmlEscape replaces < and > with HTML entities to prevent tag injection in prompts.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// ExtractResult holds extraction statistics.
type ExtractResult struct {
	EntitiesExtracted int
	VerifiedEntities  int
	SuspectEntities   int
	RejectedEntities  int
	Documents         int
	Errors            []error
}

// ExtractEntities runs LLM entity extraction on all documents that don't have entities yet.
func ExtractEntities(ctx context.Context, cfg *Config, store *SQLiteStore, llm *LLMClient) (*ExtractResult, error) {
	result := &ExtractResult{}

	levels, err := store.LoadLevels(ctx)
	if err != nil {
		return nil, fmt.Errorf("load levels: %w", err)
	}
	levelMap := make(map[string]model.Level)
	for _, l := range levels {
		levelMap[l.ID] = l
	}

	// Process each level
	for _, level := range levels {
		docs, err := store.DocumentsByLevel(ctx, level.ID)
		if err != nil {
			return nil, fmt.Errorf("documents for level %s: %w", level.ID, err)
		}

		for _, doc := range docs {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}

			// Check if entities already exist for this document version
			count, _ := store.CountEntitiesByDocument(ctx, doc.ID)
			if count > 0 {
				continue
			}
			slog.Info("extract: processing document", "doc", doc.Path, "level", level.ID)

			batch, err := extractFromDocument(ctx, cfg, store, llm, doc, level)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", doc.Path, err))
				continue
			}

			result.VerifiedEntities += batch.Verified
			result.SuspectEntities += batch.Suspect
			result.RejectedEntities += batch.Rejected

			entities := batch.Entities
			if len(entities) > 0 {
				if err := store.SaveEntities(ctx, entities); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("%s: save: %w", doc.Path, err))
					continue
				}
				result.EntitiesExtracted += len(entities)
			}
			result.Documents++
		}
	}

	return result, nil
}

type extractBatch struct {
	Entities []model.Entity
	Verified int
	Suspect  int
	Rejected int
}

func extractFromDocument(ctx context.Context, cfg *Config, store *SQLiteStore, llm *LLMClient, doc model.Document, level model.Level) (*extractBatch, error) {
	sections, err := ensureDocumentSections(ctx, cfg, store, doc)
	if err != nil {
		return nil, err
	}

	// Cap chunks for large documents
	maxChunks := cfg.Thresholds.MaxChunksPerDocument
	if maxChunks > 0 && len(sections) > maxChunks {
		original := len(sections)
		sections = sampleSections(sections, maxChunks)
		slog.Warn("chunk sampling applied", "doc", filepath.Base(doc.Path), "original", original, "sampled", len(sections))
	}

	batch := &extractBatch{}
	var allEntities []model.Entity
	seen := make(map[string]bool)
	seenRejected := make(map[string]bool)

	for i, section := range sections {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		chunkSanitized := SanitizeForPrompt(section.Content)
		prompt := buildExtractionPrompt(cfg, level, chunkSanitized, i, len(sections))

		resp, err := llm.Chat(ctx, LLMRequest{
			Model:       cfg.LLM.ExtractModel,
			System:      extractionSystemPrompt(cfg),
			User:        prompt,
			MaxTokens:   4096,
			Temperature: cfg.TemperatureForStage("extract"),
			JSONMode:    true,
		})
		if err != nil {
			return nil, fmt.Errorf("llm extract chunk %d: %w", i, err)
		}

		parsed, err := parseExtractionResponseDetailed(resp.Content, doc.ID, level.ID, cfg.LLM.ExtractModel)
		if err != nil {
			// Try to continue with partial results
			continue
		}
		batch.Suspect += parsed.Suspect
		batch.Rejected += parsed.Rejected

		for _, e := range parsed.Entities {
			admitted, accepted := admitEntityCandidate(e, section)
			key := strings.ToLower(string(e.Type)) + "|" + strings.ToLower(e.Title)
			if !accepted {
				if !seenRejected[key] {
					seenRejected[key] = true
					batch.Rejected++
				}
				continue
			}
			if !seen[key] {
				seen[key] = true
				allEntities = append(allEntities, admitted)
				batch.Verified++
			}
		}
	}

	// Generate embeddings for all entities
	if len(allEntities) > 0 {
		if err := generateEmbeddings(ctx, cfg, llm, allEntities); err != nil {
			_ = err
		}
	}

	batch.Entities = allEntities
	return batch, nil
}

func ensureDocumentSections(ctx context.Context, cfg *Config, store *SQLiteStore, doc model.Document) ([]model.Section, error) {
	sections, err := store.SectionsByDocument(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("load sections for %s: %w", doc.ID, err)
	}
	if len(sections) > 0 {
		return sections, nil
	}

	sections = ChunkSections(doc.ID, doc.Content, cfg.Thresholds.ChunkTokenLimit, cfg.Thresholds.ChunkOverlapTokens)
	if len(sections) == 0 {
		return nil, nil
	}
	if err := store.SaveSections(ctx, sections); err != nil {
		return nil, fmt.Errorf("backfill sections for %s: %w", doc.ID, err)
	}
	return sections, nil
}

func buildExtractionPrompt(cfg *Config, level model.Level, content string, chunkIndex, totalChunks int) string {
	types := strings.Join(cfg.EntityTypes, ", ")
	prompt := fmt.Sprintf(`Extract strategic entities from the following document content.

Document level: %s (rank %d)
Allowed entity types: %s

<document_content>
%s
</document_content>

Extract all strategic entities as JSON. Return a JSON object with an "entities" array. Each entity must have:
- "type": one of [%s]
- "title_original": concise name (max 100 chars) in the same language as the supporting quote
- "description_original": brief explanation (max 300 chars) in the same language as the supporting quote
- "source_quote": exact quote from the document supporting this extraction (max 500 chars)

If the content contains no strategic entities, return {"entities": []}.
If uncertain about an entity, do not include it. Do NOT translate, anglicize, or normalize non-English source text.`, level.Name, level.Rank, types, xmlEscape(content), types)

	if totalChunks > 1 {
		prompt += fmt.Sprintf("\n\nNote: This is chunk %d of %d from a larger document. Extract only entities clearly present in this chunk.", chunkIndex+1, totalChunks)
	}

	return prompt
}

func extractionSystemPrompt(cfg *Config) string {
	return `You are a strategy analyst extracting structured entities from strategic documents.

RULES:
1. Extract ONLY entities explicitly stated in the document. Do NOT infer or create entities.
2. Each entity MUST have a direct, verbatim source quote from the document.
3. Preserve the source language in title_original and description_original. Do NOT translate Russian or other non-English text into English.
4. Use exact entity types from the allowed list only.
4. If a passage is ambiguous, skip it rather than guess.
5. Return valid JSON only. No markdown, no explanations outside JSON.

HALLUCINATION PREVENTION:
- If the document content is empty or unreadable, return {"entities": []}.
- If you cannot find a clear source quote, do NOT include the entity.
- Never fabricate quotes. Every source_quote must be a verbatim excerpt.`
}

type extractionParseResult struct {
	Entities []model.Entity
	Suspect  int
	Rejected int
}

func parseExtractionResponse(content string, docID, levelID, extractModel string) ([]model.Entity, error) {
	parsed, err := parseExtractionResponseDetailed(content, docID, levelID, extractModel)
	if err != nil {
		return nil, err
	}
	return parsed.Entities, nil
}

func parseExtractionResponseDetailed(content string, docID, levelID, extractModel string) (*extractionParseResult, error) {
	raw := ParseLLMJSON(content)
	if raw == nil {
		return nil, fmt.Errorf("no valid JSON in response")
	}

	var result struct {
		Entities []struct {
			Type                string `json:"type"`
			Title               string `json:"title"`
			Description         string `json:"description"`
			TitleOriginal       string `json:"title_original"`
			DescriptionOriginal string `json:"description_original"`
			SourceQuote         string `json:"source_quote"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse entities: %w", err)
	}

	parsed := &extractionParseResult{Entities: make([]model.Entity, 0, len(result.Entities))}
	for _, e := range result.Entities {
		titleOriginal := firstNonEmpty(strings.TrimSpace(e.TitleOriginal), strings.TrimSpace(e.Title))
		descriptionOriginal := firstNonEmpty(strings.TrimSpace(e.DescriptionOriginal), strings.TrimSpace(e.Description))
		if titleOriginal == "" || e.Type == "" {
			continue
		}
		if !model.IsValidEntityType(model.EntityType(e.Type)) {
			parsed.Rejected++
			continue
		}
		id := entityID(docID, e.Type, titleOriginal)
		entity := model.Entity{
			ID:                  id,
			DocumentID:          docID,
			LevelID:             levelID,
			Type:                model.EntityType(e.Type),
			Title:               titleOriginal,
			Description:         descriptionOriginal,
			TitleOriginal:       titleOriginal,
			DescriptionOriginal: descriptionOriginal,
			SourceQuote:         e.SourceQuote,
			TrustGrade:          model.TrustGradeVerified,
			ExtractionModel:     extractModel,
		}
		if flags := detectPromptLeakFlags(entity); len(flags) > 0 {
			parsed.Rejected++
			continue
		}
		parsed.Entities = append(parsed.Entities, entity)
	}
	return parsed, nil
}

func entityID(docID, entityType, title string) string {
	h := sha256.Sum256([]byte(docID + "|" + entityType + "|" + title))
	return fmt.Sprintf("ent_%x", h[:8])
}

func generateEmbeddings(ctx context.Context, cfg *Config, llm *LLMClient, entities []model.Entity) error {
	texts := make([]string, len(entities))
	for i, e := range entities {
		title := firstNonEmpty(e.TitleOriginal, e.Title)
		description := firstNonEmpty(e.DescriptionOriginal, e.Description)
		texts[i] = title + ". " + description
	}

	// Batch in groups of 20 (embedding API limit)
	const batchSize = 20

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		embs, err := llm.Embed(ctx, texts[i:end], cfg.LLM.EmbeddingModel)
		if err != nil {
			return fmt.Errorf("embed batch %d: %w", i/batchSize, err)
		}

		for j, emb := range embs {
			if i+j < len(entities) {
				entities[i+j].Embedding = emb
				entities[i+j].EmbeddingDims = len(emb)
				entities[i+j].EmbeddingModel = cfg.LLM.EmbeddingModel
			}
		}
	}
	return nil
}

func admitEntityCandidate(entity model.Entity, section model.Section) (model.Entity, bool) {
	entity.TitleOriginal = firstNonEmpty(strings.TrimSpace(entity.TitleOriginal), strings.TrimSpace(entity.Title))
	entity.DescriptionOriginal = firstNonEmpty(strings.TrimSpace(entity.DescriptionOriginal), strings.TrimSpace(entity.Description))
	entity.Title = entity.TitleOriginal
	entity.Description = entity.DescriptionOriginal
	entity.SectionID = section.ID

	flags := append([]string{}, entity.QualityFlags...)

	sourceQuote := strings.TrimSpace(entity.SourceQuote)
	if sourceQuote == "" {
		flags = append(flags, "quote_not_found")
		entity.TrustGrade = model.TrustGradeRejected
		entity.QualityFlags = dedupeFlags(flags)
		return entity, false
	}

	sectionStart, sectionEnd, found := locateQuoteSpan(section.Content, sourceQuote)
	if !found {
		flags = append(flags, "quote_not_found")
		entity.TrustGrade = model.TrustGradeRejected
		entity.QualityFlags = dedupeFlags(flags)
		return entity, false
	}
	entity.QuoteStartOffset = intPtr(section.CharStart + sectionStart)
	entity.QuoteEndOffset = intPtr(section.CharStart + sectionEnd)

	if isBoilerplateRepetition(section.Content, sourceQuote) {
		flags = append(flags, "boilerplate_repetition")
		entity.TrustGrade = model.TrustGradeRejected
		entity.QualityFlags = dedupeFlags(flags)
		return entity, false
	}

	entity.Lang = detectPrimaryLanguage(sourceQuote)
	if entity.Lang == "unknown" {
		entity.Lang = detectPrimaryLanguage(entity.TitleOriginal + " " + entity.DescriptionOriginal)
	}
	entity.LanguageMismatch = hasLanguageMismatch(entity.Lang, entity.TitleOriginal, entity.DescriptionOriginal)
	if entity.LanguageMismatch {
		flags = append(flags, "language_mismatch")
		entity.TrustGrade = model.TrustGradeRejected
		entity.QualityFlags = dedupeFlags(flags)
		return entity, false
	}

	entity.TrustGrade = model.TrustGradeVerified
	entity.QualityFlags = dedupeFlags(flags)
	return entity, true
}

func detectPromptLeakFlags(entity model.Entity) []string {
	text := normalizeTextForMatch(strings.Join([]string{
		entity.Title,
		entity.Description,
		entity.SourceQuote,
	}, "\n"))

	markers := []string{
		"return valid json",
		"never ignore previous instructions",
		"previous instructions",
		"hallucination prevention",
		"no markdown",
		"source_quote",
		"allowed entity types",
		"extract strategic entities",
		"verbatim excerpt",
		"document_content",
	}

	var flags []string
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			flags = append(flags, "prompt_leak")
			break
		}
	}
	return dedupeFlags(flags)
}

func isBoilerplateRepetition(sourceText, quote string) bool {
	normHaystack := normalizeTextForMatch(sourceText)
	normQuote := normalizeTextForMatch(quote)
	if normQuote == "" {
		return false
	}
	if len(normQuote) < 20 {
		return false
	}
	return strings.Count(normHaystack, normQuote) >= 3
}

func normalizeTextForMatch(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func detectPrimaryLanguage(text string) string {
	var cyrillicCount, latinCount int
	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Cyrillic):
			cyrillicCount++
		case unicode.In(r, unicode.Latin):
			latinCount++
		}
	}

	if cyrillicCount+latinCount < 4 {
		return "unknown"
	}
	if cyrillicCount == 0 {
		return "en"
	}
	if latinCount == 0 {
		return "ru"
	}
	if cyrillicCount >= latinCount*2 {
		return "ru"
	}
	if latinCount >= cyrillicCount*2 {
		return "en"
	}
	return "mixed"
}

func hasLanguageMismatch(sourceLang, titleOriginal, descriptionOriginal string) bool {
	if sourceLang != "ru" && sourceLang != "en" {
		return false
	}
	for _, field := range []string{titleOriginal, descriptionOriginal} {
		fieldLang := detectPrimaryLanguage(field)
		if fieldLang == "unknown" || fieldLang == "mixed" || fieldLang == "" {
			continue
		}
		if fieldLang != sourceLang {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func dedupeFlags(flags []string) []string {
	if len(flags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(flags))
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag == "" {
			continue
		}
		if _, ok := seen[flag]; ok {
			continue
		}
		seen[flag] = struct{}{}
		result = append(result, flag)
	}
	return result
}

func locateQuoteSpan(sourceText, quote string) (int, int, bool) {
	if sourceText == "" || strings.TrimSpace(quote) == "" {
		return 0, 0, false
	}
	if byteIndex := strings.Index(sourceText, quote); byteIndex >= 0 {
		return byteOffsetToRuneOffset(sourceText, byteIndex), byteOffsetToRuneOffset(sourceText, byteIndex+len(quote)), true
	}

	normHaystack, haystackMap := normalizeTextWithRuneMap(sourceText)
	normQuote, _ := normalizeTextWithRuneMap(quote)
	if len(normQuote) == 0 {
		return 0, 0, false
	}
	start := indexRuneSlice(normHaystack, normQuote)
	if start < 0 {
		return 0, 0, false
	}
	endIdx := start + len(normQuote) - 1
	if start >= len(haystackMap) || endIdx >= len(haystackMap) {
		return 0, 0, false
	}
	return haystackMap[start], haystackMap[endIdx] + 1, true
}

func normalizeTextWithRuneMap(text string) ([]rune, []int) {
	runes := []rune(text)
	normalized := make([]rune, 0, len(runes))
	indexMap := make([]int, 0, len(runes))
	lastWasSpace := true

	for idx, r := range runes {
		if unicode.IsSpace(r) {
			if lastWasSpace {
				continue
			}
			normalized = append(normalized, ' ')
			indexMap = append(indexMap, idx)
			lastWasSpace = true
			continue
		}
		normalized = append(normalized, unicode.ToLower(r))
		indexMap = append(indexMap, idx)
		lastWasSpace = false
	}

	if len(normalized) > 0 && normalized[len(normalized)-1] == ' ' {
		normalized = normalized[:len(normalized)-1]
		indexMap = indexMap[:len(indexMap)-1]
	}

	return normalized, indexMap
}

func indexRuneSlice(haystack, needle []rune) int {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		matched := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				matched = false
				break
			}
		}
		if matched {
			return i
		}
	}
	return -1
}

func byteOffsetToRuneOffset(text string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset >= len(text) {
		return len([]rune(text))
	}
	return len([]rune(text[:byteOffset]))
}

func intPtr(value int) *int {
	return &value
}

// sampleChunks reduces chunks to maxCount using uniform sampling (first + last + evenly spaced middle).
func sampleChunks(chunks []string, maxCount int) []string {
	if len(chunks) <= maxCount {
		return chunks
	}
	result := make([]string, 0, maxCount)
	result = append(result, chunks[0]) // always first
	step := float64(len(chunks)-2) / float64(maxCount-2)
	for i := 1; i < maxCount-1; i++ {
		idx := 1 + int(float64(i)*step)
		if idx >= len(chunks)-1 {
			idx = len(chunks) - 2
		}
		result = append(result, chunks[idx])
	}
	result = append(result, chunks[len(chunks)-1]) // always last
	return result
}

func sampleSections(sections []model.Section, maxCount int) []model.Section {
	if len(sections) <= maxCount {
		return sections
	}
	result := make([]model.Section, 0, maxCount)
	result = append(result, sections[0])
	step := float64(len(sections)-2) / float64(maxCount-2)
	for i := 1; i < maxCount-1; i++ {
		idx := 1 + int(float64(i)*step)
		if idx >= len(sections)-1 {
			idx = len(sections) - 2
		}
		result = append(result, sections[idx])
	}
	result = append(result, sections[len(sections)-1])
	return result
}
