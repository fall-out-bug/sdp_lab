package strataudit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"sdp_dev/internal/strataudit/model"
)

// ExtractResult holds extraction statistics.
type ExtractResult struct {
	EntitiesExtracted int
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

			entities, err := extractFromDocument(ctx, cfg, llm, doc, level)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", doc.Path, err))
				continue
			}

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

func extractFromDocument(ctx context.Context, cfg *Config, llm *LLMClient, doc model.Document, level model.Level) ([]model.Entity, error) {
	content := doc.Content
	chunks := ChunkContent(content, cfg.Thresholds.ChunkTokenLimit, cfg.Thresholds.ChunkOverlapTokens)

	sanitized := SanitizeForPrompt(content)

	var allEntities []model.Entity
	seen := make(map[string]bool)

	for i, chunk := range chunks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		chunkSanitized := SanitizeForPrompt(chunk)
		prompt := buildExtractionPrompt(cfg, level, chunkSanitized, i, len(chunks))

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

		entities, err := parseExtractionResponse(resp.Content, doc.ID, level.ID, cfg.LLM.ExtractModel)
		if err != nil {
			// Try to continue with partial results
			continue
		}

		for _, e := range entities {
			key := strings.ToLower(e.Title)
			if !seen[key] {
				seen[key] = true
				allEntities = append(allEntities, e)
			}
		}
	}

	// Generate embeddings for all entities
	if len(allEntities) > 0 {
		if err := generateEmbeddings(ctx, llm, allEntities); err != nil {
			// Non-fatal: entities without embeddings still work, linking will skip them
			_ = err
		}
	}

	_ = sanitized
	return allEntities, nil
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
- "title": concise name (max 100 chars)
- "description": brief explanation (max 300 chars)
- "source_quote": exact quote from the document supporting this extraction (max 500 chars)

If the content contains no strategic entities, return {"entities": []}.
If uncertain about an entity, do not include it.`, level.Name, level.Rank, types, content, types)

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
3. Use exact entity types from the allowed list only.
4. If a passage is ambiguous, skip it rather than guess.
5. Return valid JSON only. No markdown, no explanations outside JSON.

HALLUCINATION PREVENTION:
- If the document content is empty or unreadable, return {"entities": []}.
- If you cannot find a clear source quote, do NOT include the entity.
- Never fabricate quotes. Every source_quote must be a verbatim excerpt.`
}

func parseExtractionResponse(content string, docID, levelID, extractModel string) ([]model.Entity, error) {
	raw := ParseLLMJSON(content)
	if raw == nil {
		return nil, fmt.Errorf("no valid JSON in response")
	}

	var result struct {
		Entities []struct {
			Type        string `json:"type"`
			Title       string `json:"title"`
			Description string `json:"description"`
			SourceQuote string `json:"source_quote"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse entities: %w", err)
	}

	entities := make([]model.Entity, 0, len(result.Entities))
	for _, e := range result.Entities {
		if e.Title == "" || e.Type == "" {
			continue
		}
		id := entityID(docID, e.Type, e.Title)
		entities = append(entities, model.Entity{
			ID:              id,
			DocumentID:      docID,
			LevelID:         levelID,
			Type:            model.EntityType(e.Type),
			Title:           e.Title,
			Description:     e.Description,
			SourceQuote:     e.SourceQuote,
			ExtractionModel: extractModel,
		})
	}
	return entities, nil
}

func entityID(docID, entityType, title string) string {
	h := sha256.Sum256([]byte(docID + "|" + entityType + "|" + title))
	return fmt.Sprintf("ent_%x", h[:8])
}

func generateEmbeddings(ctx context.Context, llm *LLMClient, entities []model.Entity) error {
	texts := make([]string, len(entities))
	for i, e := range entities {
		texts[i] = e.Title + ". " + e.Description
	}

	// Batch in groups of 20 (embedding API limit)
	const batchSize = 20
	var mu sync.Mutex

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		embs, err := llm.Embed(ctx, texts[i:end])
		if err != nil {
			return fmt.Errorf("embed batch %d: %w", i/batchSize, err)
		}

		mu.Lock()
		for j, emb := range embs {
			if i+j < len(entities) {
				entities[i+j].Embedding = emb
				entities[i+j].EmbeddingDims = len(emb)
			}
		}
		mu.Unlock()
	}
	return nil
}
