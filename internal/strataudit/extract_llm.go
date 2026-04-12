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
	Documents         int
	Errors            []error
}

// ExtractEntities runs LLM entity extraction on all documents that don't have entities yet.
func ExtractEntities(ctx context.Context, cfg *Config, store *SQLiteStore, runtime ModelRuntime) (*ExtractResult, error) {
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

			entities, err := extractFromDocument(ctx, cfg, runtime, doc, level)
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

func extractFromDocument(ctx context.Context, cfg *Config, runtime ModelRuntime, doc model.Document, level model.Level) ([]model.Entity, error) {
	content := doc.Content
	chunks := ChunkContent(content, cfg.Thresholds.ChunkTokenLimit, cfg.Thresholds.ChunkOverlapTokens)

	// Cap chunks for large documents
	maxChunks := cfg.Thresholds.MaxChunksPerDocument
	if maxChunks > 0 && len(chunks) > maxChunks {
		original := len(chunks)
		chunks = sampleChunks(chunks, maxChunks)
		slog.Warn("chunk sampling applied", "doc", filepath.Base(doc.Path), "original", original, "sampled", len(chunks))
	}

	var allEntities []model.Entity
	seen := make(map[string]bool)

	for i, chunk := range chunks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		chunkSanitized := SanitizeForPrompt(chunk)
		prompt := buildExtractionPrompt(cfg, level, chunkSanitized, i, len(chunks))

		resp, err := runtime.Chat(ctx, LLMRequest{
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
			key := strings.ToLower(string(e.Type)) + "|" + strings.ToLower(e.Title)
			if !seen[key] {
				seen[key] = true
				allEntities = append(allEntities, e)
			}
		}
	}

	// Generate embeddings for all entities
	if len(allEntities) > 0 {
		if err := generateEmbeddings(ctx, cfg, runtime, allEntities); err != nil {
			_ = err
		}
	}

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
If uncertain about an entity, do not include it.`, level.Name, level.Rank, types, xmlEscape(content), types)

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

func generateEmbeddings(ctx context.Context, cfg *Config, runtime ModelRuntime, entities []model.Entity) error {
	texts := make([]string, len(entities))
	for i, e := range entities {
		texts[i] = e.Title + ". " + e.Description
	}

	// Batch in groups of 20 (embedding API limit)
	const batchSize = 20

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		embs, err := runtime.Embed(ctx, texts[i:end], cfg.LLM.EmbeddingModel)
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
