package llmguard

import (
	"fmt"
	"strings"
)

// Chunk carries metadata for a classified text slice.
type Chunk struct {
	ChunkID       string `json:"chunk_id"`
	ByteStart     int    `json:"byte_start"`
	ByteEnd       int    `json:"byte_end"`
	OverlapBefore int    `json:"overlap_before"`
	OverlapAfter  int    `json:"overlap_after"`
	Source        string `json:"source"`
	Text          string `json:"text"`
	IsBoundary    bool   `json:"is_boundary"`
}

// Chunker splits prompt text into classifier-sized chunks.
type Chunker struct {
	maxChunkBytes int
	overlap       int
	maxChunks     int
}

// NewChunker creates a chunker from policy. Validates overlap < maxChunkBytes/4.
func NewChunker(cfg ClassifierConfig) (*Chunker, error) {
	if cfg.MaxChunkBytes <= 0 {
		return nil, fmt.Errorf("chunker: max_chunk_bytes must be > 0")
	}
	if cfg.OverlapBytes < 0 {
		return nil, fmt.Errorf("chunker: overlap_bytes must be >= 0")
	}
	if cfg.OverlapBytes > cfg.MaxChunkBytes/4 {
		return nil, fmt.Errorf("chunker: overlap_bytes %d exceeds max_chunk_bytes/4 %d", cfg.OverlapBytes, cfg.MaxChunkBytes/4)
	}
	return &Chunker{
		maxChunkBytes: cfg.MaxChunkBytes,
		overlap:       cfg.OverlapBytes,
		maxChunks:     cfg.MaxClassifierChunks,
	}, nil
}

// Split breaks text into normal chunks plus synthetic boundary chunks.
// Returns error if total chunks (normal + boundary) would exceed maxChunks.
func (c *Chunker) Split(text string) ([]Chunk, error) {
	normal := c.splitNormal(text)
	if len(normal) == 0 {
		return nil, nil
	}

	// Build boundary chunks for adjacent pairs.
	var chunks []Chunk
	for i, ch := range normal {
		chunks = append(chunks, ch)
		if i < len(normal)-1 {
			next := normal[i+1]
			boundaryText := boundaryWindow(text, ch.ByteEnd-c.overlap, next.ByteStart+c.overlap)
			if boundaryText != "" {
				chunks = append(chunks, Chunk{
					ChunkID:       fmt.Sprintf("boundary-%s-%s", ch.ChunkID, next.ChunkID),
					ByteStart:     max(ch.ByteEnd-c.overlap, 0),
					ByteEnd:       min(next.ByteStart+c.overlap, len(text)),
					OverlapBefore: c.overlap,
					OverlapAfter:  c.overlap,
					Source:        fmt.Sprintf("boundary:%s:%s", ch.ChunkID, next.ChunkID),
					Text:          boundaryText,
					IsBoundary:    true,
				})
			}
		}
	}

	if len(chunks) > c.maxChunks {
		return nil, fmt.Errorf("chunker: total chunks %d exceed max %d", len(chunks), c.maxChunks)
	}
	return chunks, nil
}

// splitNormal produces non-overlapping normal chunks.
func (c *Chunker) splitNormal(text string) []Chunk {
	if len(text) == 0 {
		return nil
	}
	if len(text) <= c.maxChunkBytes {
		return []Chunk{{
			ChunkID:   "chunk-0000",
			ByteStart: 0,
			ByteEnd:   len(text),
			Source:    "message:user:0",
			Text:      text,
		}}
	}

	var chunks []Chunk
	start := 0
	idx := 0
	for start < len(text) {
		end := min(start+c.maxChunkBytes, len(text))
		// Try to find a structural split point near the end.
		if end < len(text) {
			if split := findStructuralSplit(text, start, end); split > start {
				end = split
			}
		}
		chunkText := text[start:end]
		chunks = append(chunks, Chunk{
			ChunkID:       fmt.Sprintf("chunk-%04d", idx),
			ByteStart:     start,
			ByteEnd:       end,
			OverlapBefore: c.overlap,
			OverlapAfter:  c.overlap,
			Source:        fmt.Sprintf("message:user:%d", idx),
			Text:          chunkText,
		})
		start = end
		idx++
	}
	return chunks
}

// findStructuralSplit looks for a newline, markdown heading, or code fence
// within the trailing 25% of the chunk to produce a cleaner split.
func findStructuralSplit(text string, start, end int) int {
	searchStart := start + (end-start)*3/4
	if searchStart < start {
		searchStart = start
	}
	candidates := []string{"\n```", "\n#", "\n##", "\n###", "\n\n", "\n"}
	best := end
	for _, sep := range candidates {
		if idx := strings.LastIndex(text[searchStart:end], sep); idx >= 0 {
			candidate := searchStart + idx + len(sep)
			if candidate > start && candidate < best {
				best = candidate
			}
		}
	}
	if best == end {
		return 0 // no good split found; caller uses hard byte split
	}
	return best
}

func boundaryWindow(text string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	if start >= end {
		return ""
	}
	return text[start:end]
}
