package llmguard

import (
	"context"
	"sync"
	"time"
)

// ClassifierOrchestrator runs chunking, parallel classification, and reduction.
type ClassifierOrchestrator struct {
	client  *ClassifierClient
	chunker *Chunker
	reducer *Reducer
	cfg     ClassifierConfig
}

// NewClassifierOrchestrator creates an orchestrator from config.
func NewClassifierOrchestrator(cfg ClassifierConfig) (*ClassifierOrchestrator, error) {
	client, err := NewClassifierClient(cfg)
	if err != nil {
		return nil, err
	}
	chunker, err := NewChunker(cfg)
	if err != nil {
		return nil, err
	}
	return &ClassifierOrchestrator{
		client:  client,
		chunker: chunker,
		reducer: NewReducer(cfg),
		cfg:     cfg,
	}, nil
}

// ClassifyPrompt chunks text, classifies chunks in parallel, and reduces.
func (co *ClassifierOrchestrator) ClassifyPrompt(ctx context.Context, text string, deterministicBlocked bool) (VerdictState, []SuggestedSpan, string, []string, error) {
	chunks, err := co.chunker.Split(text)
	if err != nil {
		return VerdictClassifierIncomplete, nil, "", nil, err
	}
	if len(chunks) == 0 {
		return VerdictCleanAllowed, nil, "", nil, nil
	}

	totalTimeout := time.Duration(co.cfg.TotalTimeoutMs) * time.Millisecond
	if totalTimeout <= 0 {
		totalTimeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()

	results := make(map[string]*ClassifierResult, len(chunks))
	var failed []string
	var mu sync.Mutex

	sem := make(chan struct{}, co.cfg.MaxParallelChunks)
	if cap(sem) <= 0 {
		sem = make(chan struct{}, 4)
	}
	var wg sync.WaitGroup

	for _, chunk := range chunks {
		wg.Add(1)
		go func(ch Chunk) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := co.client.ClassifyChunk(ctx, ch)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = append(failed, ch.ChunkID)
				return
			}
			results[ch.ChunkID] = res
		}(chunk)
	}
	wg.Wait()

	state, spans, reason := co.reducer.ReduceVerdict(results, failed, deterministicBlocked)
	return state, spans, reason, failed, nil
}
