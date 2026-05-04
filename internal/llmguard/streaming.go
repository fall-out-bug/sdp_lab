package llmguard

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// StreamChunk is a generic chunk that HTTP handlers format into SSE.
type StreamChunk struct {
	// Content is the assistant text delta for this chunk.
	Content string
	// FinishReason is set on the final chunk.
	FinishReason string
	// Usage is set on the final chunk when include_usage is true.
	Usage *TokenUsageAudit
	// Blocked is true when the guard blocked the request/response.
	Blocked bool
	// BlockedReason is the human-readable block reason.
	BlockedReason string
}

// StreamingGateway wraps a non-streaming Gateway to produce streaming-shaped
// output while preserving the guard-before-upstream and output-scan invariants.
type StreamingGateway struct {
	gw *Gateway
}

// NewStreamingGateway creates a streaming gateway from an existing Gateway.
func NewStreamingGateway(gw *Gateway) *StreamingGateway {
	return &StreamingGateway{gw: gw}
}

// ChatStream performs guarded chat and returns a stream iterator.
// The upstream is called once (non-streaming), output is buffered and scanned,
// then emitted as chunks. A supplementary audit record is written with
// StreamReturned=true when the stream is successfully created.
func (sg *StreamingGateway) ChatStream(ctx context.Context, req *ChatRequest, prov *Provenance, harness, endpoint string) (*guardedStreamIterator, *Verdict, error) {
	if prov == nil {
		prov = &Provenance{}
	}
	prov.Harness = harness
	prov.EndpointSurface = endpoint
	prov.StreamRequested = true

	resp, verdict, err := sg.gw.Chat(ctx, req, prov)
	if err != nil && verdict != nil && verdict.State == VerdictAuditFailed {
		return nil, verdict, err
	}

	// Write supplementary streaming meta audit record.
	if verdict != nil && verdict.State != VerdictAuditFailed {
		meta := GuardEvent{
			EventID:         uuid.New().String(),
			Timestamp:       time.Now(),
			Harness:         harness,
			EndpointSurface: endpoint,
			StreamRequested: true,
			StreamReturned:  true,
			VerdictState:    verdict.State,
			Model:           string(req.Model),
		}
		if verdict.State != VerdictInputBlocked && verdict.State != VerdictScanBudgetExceeded {
			meta.UpstreamCalled = true
		}
		_ = sg.gw.audit.WriteGuardEvent(ctx, meta) // best-effort supplementary record
	}

	it := &guardedStreamIterator{}

	switch verdict.State {
	case VerdictInputBlocked, VerdictScanBudgetExceeded:
		it.chunks = append(it.chunks, StreamChunk{
			Blocked:       true,
			BlockedReason: "input blocked by guard policy",
			FinishReason:  "stop",
		})

	case VerdictOutputBlocked:
		it.chunks = append(it.chunks, StreamChunk{
			Blocked:       true,
			BlockedReason: "output blocked by guard policy",
			FinishReason:  "stop",
		})

	case VerdictProviderErrorAfterInputPass:
		it.chunks = append(it.chunks, StreamChunk{
			Blocked:       true,
			BlockedReason: verdict.ProviderErrorText,
			FinishReason:  "stop",
		})

	default:
		// Allowed variants: emit content then finish chunk.
		if resp != nil && resp.Message.Content != "" {
			it.chunks = append(it.chunks, StreamChunk{
				Content: resp.Message.Content,
			})
		}
		var usage *TokenUsageAudit
		if resp != nil && resp.Usage != nil {
			usage = &TokenUsageAudit{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
		}
		it.chunks = append(it.chunks, StreamChunk{
			FinishReason: "stop",
			Usage:        usage,
		})
	}

	return it, verdict, nil
}

// guardedStreamIterator yields chunks until EOF.
type guardedStreamIterator struct {
	chunks []StreamChunk
	index  int
}

// Next returns the next chunk or io.EOF when exhausted.
func (it *guardedStreamIterator) Next() (*StreamChunk, error) {
	if it.index >= len(it.chunks) {
		return nil, io.EOF
	}
	chunk := it.chunks[it.index]
	it.index++
	return &chunk, nil
}
