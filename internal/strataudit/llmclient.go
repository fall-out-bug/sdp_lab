package strataudit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"sdp_dev/internal/llmclient"

	"golang.org/x/time/rate"
)

type LLMRequest struct {
	Model       string
	System      string
	User        string
	MaxTokens   int
	Temperature float64
	JSONMode    bool
}

type LLMResponse struct {
	Content    string
	TokensIn   int
	TokensOut  int
	Cached     bool
	Model      string
	DurationMs int64
}

type LLMClient struct {
	inner      *llmclient.Client
	apiKey     string
	baseURL    string
	http       *http.Client // used only for Embed()
	limiter    *rate.Limiter
	maxRetries int
	retryDelay time.Duration
}

var _ ModelRuntime = (*LLMClient)(nil)

func NewLLMClient(apiKey, baseURL string) *LLMClient {
	return &LLMClient{
		inner:      llmclient.New(apiKey, strings.TrimRight(baseURL, "/")),
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		http:       &http.Client{Timeout: 120 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(0.5), 1),
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

func (c *LLMClient) SetRetryConfig(maxRetries int, baseDelay time.Duration) {
	c.maxRetries = maxRetries
	c.retryDelay = baseDelay
}

func (c *LLMClient) SetRateLimit(requestsPerMinute int) {
	c.limiter = rate.NewLimiter(rate.Limit(requestsPerMinute)/60, 1)
}

func (c *LLMClient) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	cacheKey := c.cacheKey(req)
	if cached := c.checkCache(cacheKey); cached != "" {
		return &LLMResponse{Content: cached, Cached: true}, nil
	}

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	start := time.Now()

	msgs := []llmclient.Message{}
	if req.System != "" {
		msgs = append(msgs, llmclient.Message{Role: "system", Content: req.System})
	}
	msgs = append(msgs, llmclient.Message{Role: "user", Content: req.User})

	llmReq := llmclient.ChatRequest{
		Model:       req.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	var resp *llmclient.ChatResponse
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(c.retryDelay * time.Duration(1<<(attempt-1))):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		var err error
		resp, err = c.inner.Chat(ctx, llmReq)
		if err == nil {
			break
		}
		lastErr = err
		// 4xx errors are not retriable
		if strings.Contains(err.Error(), "status 4") {
			return nil, err
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("llm request after %d retries: %w", c.maxRetries, lastErr)
	}

	content := resp.Content
	duration := time.Since(start).Milliseconds()

	c.storeCache(cacheKey, content)

	return &LLMResponse{
		Content:    content,
		TokensIn:   resp.InputTokens,
		TokensOut:  resp.OutputTokens,
		DurationMs: duration,
	}, nil
}

// extractFinalAnswer extracts the final answer from reasoning model output.
// Looks for <answer>...</answer> tags first, then falls back to the last
// non-empty paragraph.
func extractFinalAnswer(reasoning string) string {
	// Try <answer>...</answer> tag
	re := regexp.MustCompile(`(?s)<answer>(.*?)</answer>`)
	if matches := re.FindStringSubmatch(reasoning); len(matches) > 1 {
		trimmed := strings.TrimSpace(matches[1])
		if trimmed != "" {
			return trimmed
		}
	}

	// Fallback: last non-empty paragraph (split by double newline)
	paragraphs := strings.Split(reasoning, "\n\n")
	for i := len(paragraphs) - 1; i >= 0; i-- {
		p := strings.TrimSpace(paragraphs[i])
		if p != "" {
			return p
		}
	}

	return reasoning
}

func (c *LLMClient) Embed(ctx context.Context, texts []string, model string) ([][]float32, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	if model == "" {
		model = "openai/text-embedding-3-small"
	}
	body := map[string]interface{}{
		"model": model,
		"input": texts,
	}
	bodyJSON, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}

	embs := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embs[i] = d.Embedding
	}
	return embs, nil
}

const maxCacheEntries = 10000

type cacheEntry struct {
	value     string
	createdAt time.Time
}

var (
	llmCache   = make(map[string]cacheEntry)
	llmCacheMu sync.RWMutex
)

func (c *LLMClient) cacheKey(req LLMRequest) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%f", req.Model, req.System, req.User, req.Temperature)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *LLMClient) checkCache(key string) string {
	llmCacheMu.RLock()
	defer llmCacheMu.RUnlock()
	if e, ok := llmCache[key]; ok {
		return e.value
	}
	return ""
}

func (c *LLMClient) storeCache(key, value string) {
	llmCacheMu.Lock()
	defer llmCacheMu.Unlock()
	// Evict 20% of entries when cache is full
	if len(llmCache) >= maxCacheEntries {
		count := 0
		for k := range llmCache {
			delete(llmCache, k)
			count++
			if count >= maxCacheEntries/5 {
				break
			}
		}
	}
	llmCache[key] = cacheEntry{value: value, createdAt: time.Now()}
}

// ParseLLMJSON extracts JSON from LLM response (handles markdown wrapping, prefixes)
func ParseLLMJSON(input string) json.RawMessage {
	input = strings.TrimSpace(input)

	if json.Valid([]byte(input)) {
		return json.RawMessage(input)
	}

	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?```")
	if matches := re.FindStringSubmatch(input); len(matches) > 1 {
		candidate := strings.TrimSpace(matches[1])
		if json.Valid([]byte(candidate)) {
			return json.RawMessage(candidate)
		}
	}

	for _, delim := range []byte{'{', '['} {
		start := strings.IndexByte(input, delim)
		if start == -1 {
			continue
		}
		remainder := input[start:]
		if json.Valid([]byte(remainder)) {
			return json.RawMessage(remainder)
		}
		level := 0
		for i, ch := range remainder {
			if byte(ch) == delim {
				level++
			}
			closeDelim := byte('}')
			if delim == '[' {
				closeDelim = ']'
			}
			if byte(ch) == closeDelim {
				level--
				if level == 0 {
					candidate := remainder[:i+1]
					if json.Valid([]byte(candidate)) {
						return json.RawMessage(candidate)
					}
					break
				}
			}
		}
	}

	return nil
}
