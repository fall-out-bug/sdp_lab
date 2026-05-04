// Package main implements a thin HTTP demo proxy over the llmguard core.
//
// This is an acceptance/demo surface, not the production SDP model gateway.
// It exposes OpenAI-compatible endpoints:
//   - POST /v1/responses    (Codex SSE)
//   - POST /v1/chat/completions (Pi streaming)
//   - POST /               (legacy demo JSON)
//
// It uses httptest and fake providers in CI; no live provider calls are made.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/llmguard"
	"github.com/fall-out-bug/sdp_lab/internal/modelgateway"
)

// writeJSON is a helper that writes a JSON response. Errors on write are logged
// but not propagated — the response header is already sent.
func writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// --- Rate limiter ---

type rateLimiter struct {
	mu     sync.Mutex
	limits map[string]*bucket
	rate   int // requests per minute
	window time.Duration
}

type bucket struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(ratePerMinute int) *rateLimiter {
	return &rateLimiter{
		limits: make(map[string]*bucket),
		rate:   ratePerMinute,
		window: time.Minute,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.limits[ip]
	if !ok || now.After(b.resetAt) {
		rl.limits[ip] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if b.count >= rl.rate {
		return false
	}

	b.count++
	return true
}

func clientIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

// --- Request/Response schemas ---

type demoRequest struct {
	Model    string        `json:"model"`
	Messages []demoMessage `json:"messages"`
	Metadata *demoMetadata `json:"metadata,omitempty"`
}

type demoMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type demoMetadata struct {
	CorrelationID string `json:"correlation_id,omitempty"`
	FeatureID     string `json:"feature_id,omitempty"`
	WsID          string `json:"ws_id,omitempty"`
	BeadsID       string `json:"beads_id,omitempty"`
}

type demoBlockedResponse struct {
	VerdictState string        `json:"verdict_state"`
	Warning      string        `json:"warning"`
	Findings     []demoFinding `json:"findings"`
	EventID      string        `json:"event_id"`
}

type demoFinding struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
}

type demoAllowedResponse struct {
	VerdictState string       `json:"verdict_state"`
	Message      *demoMessage `json:"message,omitempty"`
	EventID      string       `json:"event_id"`
	Usage        *demoUsage   `json:"usage,omitempty"`
	Warning      string       `json:"warning,omitempty"`
}

type demoUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type demoErrorResponse struct {
	Error   string `json:"error"`
	EventID string `json:"event_id,omitempty"`
}

// --- OpenAI Chat Completions streaming schemas ---

type chatCompletionRequest struct {
	Model         string        `json:"model"`
	Messages      []demoMessage `json:"messages"`
	Stream        bool          `json:"stream,omitempty"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
	Temperature   float64       `json:"temperature,omitempty"`
	MaxTokens     int           `json:"max_tokens,omitempty"`
	Metadata      *demoMetadata `json:"metadata,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type chatCompletionStreamChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   *demoUsage             `json:"usage,omitempty"`
}

type chatCompletionChoice struct {
	Index        int                  `json:"index"`
	Delta        chatCompletionDelta  `json:"delta"`
	FinishReason string               `json:"finish_reason,omitempty"`
}

type chatCompletionDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// --- Codex Responses API schemas ---

type responsesRequest struct {
	Model    string        `json:"model"`
	Input    string        `json:"input"`
	Stream   bool          `json:"stream,omitempty"`
	Metadata *demoMetadata `json:"metadata,omitempty"`
}

type responsesEvent struct {
	Type     string            `json:"type"`
	Response *responsesResponse `json:"response,omitempty"`
}

type responsesResponse struct {
	ID     string                `json:"id"`
	Status string                `json:"status"`
	Output []responsesOutputItem `json:"output,omitempty"`
	Usage  *demoUsage            `json:"usage,omitempty"`
}

type responsesOutputItem struct {
	Type    string               `json:"type"`
	ID      string               `json:"id,omitempty"`
	Status  string               `json:"status,omitempty"`
	Role    string               `json:"role,omitempty"`
	Content []responsesContentPart `json:"content,omitempty"`
}

type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// --- Demo handler (legacy JSON) ---

type demoHandler struct {
	gateway *llmguard.Gateway
	stream  *llmguard.StreamingGateway
	limiter *rateLimiter
}

func (h *demoHandler) checkRateLimit(w http.ResponseWriter, r *http.Request) bool {
	if !h.limiter.allow(clientIP(r)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, map[string]string{"error": "rate limit exceeded"})
		return false
	}
	return true
}

func (h *demoHandler) buildProvenance(meta *demoMetadata) *llmguard.Provenance {
	prov := &llmguard.Provenance{}
	if meta != nil {
		prov.CorrelationID = meta.CorrelationID
		prov.FeatureID = meta.FeatureID
		prov.WsID = meta.WsID
		prov.BeadsID = meta.BeadsID
	}
	return prov
}

func (h *demoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkRateLimit(w, r) {
		return
	}

	var req demoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	messages := make([]llmguard.ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = llmguard.ChatMessage{Role: modelgateway.MessageRole(m.Role), Content: m.Content}
	}

	resp, verdict, err := h.gateway.Chat(r.Context(), &llmguard.ChatRequest{
		Model:    modelgateway.ModelID(req.Model),
		Messages: messages,
	}, h.buildProvenance(req.Metadata))

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, demoErrorResponse{Error: err.Error(), EventID: verdict.EventID})
		return
	}

	switch verdict.State {
	case llmguard.VerdictInputBlocked:
		w.WriteHeader(http.StatusOK)
		findings := make([]demoFinding, len(verdict.InputFindings))
		for i, f := range verdict.InputFindings {
			findings[i] = demoFinding{Type: string(f.Type), Severity: string(f.Severity)}
		}
		writeJSON(w, demoBlockedResponse{
			VerdictState: string(verdict.State),
			Warning:      "request blocked by input guard",
			Findings:     findings,
			EventID:      verdict.EventID,
		})
	case llmguard.VerdictOutputBlocked:
		w.WriteHeader(http.StatusOK)
		findings := make([]demoFinding, len(verdict.OutputFindings))
		for i, f := range verdict.OutputFindings {
			findings[i] = demoFinding{Type: string(f.Type), Severity: string(f.Severity)}
		}
		writeJSON(w, demoBlockedResponse{
			VerdictState: string(verdict.State),
			Warning:      "response blocked by output guard",
			Findings:     findings,
			EventID:      verdict.EventID,
		})
	case llmguard.VerdictAuditFailed:
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, demoErrorResponse{Error: "audit write failed", EventID: verdict.EventID})
	default:
		w.WriteHeader(http.StatusOK)
		var msg *demoMessage
		if resp != nil {
			msg = &demoMessage{Role: string(resp.Message.Role), Content: resp.Message.Content}
		}
		var usage *demoUsage
		if resp != nil && resp.Usage != nil {
			usage = &demoUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
		}
		var warning string
		if verdict.State == llmguard.VerdictAllowedWithOutputFindings {
			warning = "response contains advisory output findings"
		}
		if verdict.State == llmguard.VerdictRedactedAllowed {
			warning = "input was redacted before provider call"
		}
		writeJSON(w, demoAllowedResponse{
			VerdictState: string(verdict.State),
			Message:      msg,
			EventID:      verdict.EventID,
			Usage:        usage,
			Warning:      warning,
		})
	}
}

// --- Chat Completions streaming handler ---

func (h *demoHandler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkRateLimit(w, r) {
		return
	}

	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	messages := make([]llmguard.ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = llmguard.ChatMessage{Role: modelgateway.MessageRole(m.Role), Content: m.Content}
	}

	includeUsage := req.StreamOptions != nil && req.StreamOptions.IncludeUsage
	now := time.Now().Unix()

	it, verdict, err := h.stream.ChatStream(r.Context(), &llmguard.ChatRequest{
		Model:    modelgateway.ModelID(req.Model),
		Messages: messages,
	}, h.buildProvenance(req.Metadata), "pi", "/v1/chat/completions")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, demoErrorResponse{Error: err.Error(), EventID: verdict.EventID})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for {
		chunk, err := it.Next()
		if err != nil {
			break
		}

		if chunk.Blocked {
			// Emit a single blocked delta and finish.
			data, _ := json.Marshal(chatCompletionStreamChunk{
				ID:     verdict.EventID,
				Object: "chat.completion.chunk",
				Created: now,
				Model:  req.Model,
				Choices: []chatCompletionChoice{{
					Index: 0,
					Delta: chatCompletionDelta{Content: chunk.BlockedReason},
					FinishReason: chunk.FinishReason,
				}},
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			continue
		}

		choices := []chatCompletionChoice{}
		if chunk.Content != "" {
			choices = append(choices, chatCompletionChoice{
				Index: 0,
				Delta: chatCompletionDelta{Content: chunk.Content},
			})
		}
		if chunk.FinishReason != "" {
			choices = append(choices, chatCompletionChoice{
				Index:        0,
				Delta:        chatCompletionDelta{},
				FinishReason: chunk.FinishReason,
			})
		}

		var usage *demoUsage
		if includeUsage && chunk.Usage != nil {
			usage = &demoUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		if len(choices) > 0 || usage != nil {
			data, _ := json.Marshal(chatCompletionStreamChunk{
				ID:      verdict.EventID,
				Object:  "chat.completion.chunk",
				Created: now,
				Model:   req.Model,
				Choices: choices,
				Usage:   usage,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// --- Codex Responses SSE handler ---

func (h *demoHandler) handleResponses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.checkRateLimit(w, r) {
		return
	}

	var req responsesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid request body"})
		return
	}

	// Normalize input into messages
	messages := []llmguard.ChatMessage{
		{Role: modelgateway.RoleUser, Content: req.Input},
	}

	it, verdict, err := h.stream.ChatStream(r.Context(), &llmguard.ChatRequest{
		Model:    modelgateway.ModelID(req.Model),
		Messages: messages,
	}, h.buildProvenance(req.Metadata), "codex", "/v1/responses")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, demoErrorResponse{Error: err.Error(), EventID: verdict.EventID})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for {
		chunk, err := it.Next()
		if err != nil {
			break
		}

		if chunk.Blocked {
			event := responsesEvent{
				Type: "response.output_text.delta",
				Response: &responsesResponse{
					ID:     verdict.EventID,
					Status: "completed",
					Output: []responsesOutputItem{{
						Type: "message",
						Role: "assistant",
						Content: []responsesContentPart{{
							Type: "output_text",
							Text: chunk.BlockedReason,
						}},
					}},
				},
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			// Terminal event for blocked responses.
			completed := responsesEvent{
				Type: "response.completed",
				Response: &responsesResponse{
					ID:     verdict.EventID,
					Status: "completed",
				},
			}
			data, _ = json.Marshal(completed)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			continue
		}

		if chunk.Content != "" {
			event := responsesEvent{
				Type: "response.output_text.delta",
				Response: &responsesResponse{
					ID:     verdict.EventID,
					Status: "in_progress",
					Output: []responsesOutputItem{{
						Type: "message",
						Role: "assistant",
						Content: []responsesContentPart{{
							Type: "output_text",
							Text: chunk.Content,
						}},
					}},
				},
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

		if chunk.FinishReason != "" {
			var usage *demoUsage
			if chunk.Usage != nil {
				usage = &demoUsage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}
			completed := responsesEvent{
				Type: "response.completed",
				Response: &responsesResponse{
					ID:     verdict.EventID,
					Status: "completed",
					Usage:  usage,
				},
			}
			data, _ := json.Marshal(completed)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// --- Fake echo provider for demo ---

type echoProvider struct{}

func (e *echoProvider) Chat(ctx context.Context, req *llmguard.ChatRequest) (*llmguard.ChatResponse, error) {
	var content string
	for _, m := range req.Messages {
		if m.Role == "user" {
			content = m.Content
		}
	}
	return &llmguard.ChatResponse{
		ID:    "echo-response",
		Model: req.Model,
		Message: llmguard.ChatMessage{
			Role:    "assistant",
			Content: "Echo: " + content,
		},
		Usage: &llmguard.TokenUsageAudit{
			PromptTokens:     len(content),
			CompletionTokens: len(content) + 6,
			TotalTokens:      2*len(content) + 6,
		},
	}, nil
}

// --- Main ---

func main() {
	addr := flag.String("addr", ":8090", "listen address")
	rateLimit := flag.Int("rate-limit", 60, "requests per minute per IP")
	flag.Parse()

	audit := llmguard.NewJSONLAuditSink(&bytes.Buffer{})
	gw := llmguard.NewGateway(&echoProvider{}, llmguard.DefaultPolicy(), audit)
	stream := llmguard.NewStreamingGateway(gw)

	handler := &demoHandler{
		gateway: gw,
		stream:  stream,
		limiter: newRateLimiter(*rateLimit),
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/v1/chat/completions", handler.handleChatCompletions)
	mux.HandleFunc("/v1/responses", handler.handleResponses)

	fmt.Printf("sdp-llm-gateway demo proxy listening on %s (rate limit: %d/min/IP)\n", *addr, *rateLimit)
	fmt.Println("NOTE: This is a demo/acceptance surface, not the production SDP model gateway.")
	log.Fatal(http.ListenAndServe(*addr, mux))
}
