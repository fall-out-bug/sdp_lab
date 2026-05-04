// Package main implements a thin HTTP demo proxy over the llmguard core.
//
// This is an acceptance/demo surface, not the production SDP model gateway.
// It exposes a minimal HTTP endpoint that accepts a simplified chat-like JSON
// request and returns guarded JSON responses. It uses httptest and fake providers
// in CI; no live provider calls are made.
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
func writeJSON(w http.ResponseWriter, v interface{}) {
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

// --- Demo handler ---

type demoHandler struct {
	gateway *llmguard.Gateway
	limiter *rateLimiter
}

func (h *demoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limit check
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	if !h.limiter.allow(ip) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, map[string]string{
			"error": "rate limit exceeded",
		})
		return
	}

	// Parse request
	var req demoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	// Build llmguard request
	messages := make([]llmguard.ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = llmguard.ChatMessage{Role: modelgateway.MessageRole(m.Role), Content: m.Content}
	}

	prov := &llmguard.Provenance{}
	if req.Metadata != nil {
		prov.CorrelationID = req.Metadata.CorrelationID
		prov.FeatureID = req.Metadata.FeatureID
		prov.WsID = req.Metadata.WsID
		prov.BeadsID = req.Metadata.BeadsID
	}

	resp, verdict, err := h.gateway.Chat(r.Context(), &llmguard.ChatRequest{
		Model:    modelgateway.ModelID(req.Model),
		Messages: messages,
	}, prov)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, demoErrorResponse{
			Error:   err.Error(),
			EventID: verdict.EventID,
		})
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
		writeJSON(w, demoErrorResponse{
			Error:   "audit write failed",
			EventID: verdict.EventID,
		})

	default:
		// Allowed variants
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

// --- Fake echo provider for demo ---

type echoProvider struct{}

func (e *echoProvider) Chat(ctx context.Context, req *llmguard.ChatRequest) (*llmguard.ChatResponse, error) {
	// Echo back the last user message
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

	handler := &demoHandler{
		gateway: gw,
		limiter: newRateLimiter(*rateLimit),
	}

	fmt.Printf("sdp-llm-gateway demo proxy listening on %s (rate limit: %d/min/IP)\n", *addr, *rateLimit)
	fmt.Println("NOTE: This is a demo/acceptance surface, not the production SDP model gateway.")
	log.Fatal(http.ListenAndServe(*addr, handler))
}
