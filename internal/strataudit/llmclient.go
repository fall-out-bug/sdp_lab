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
	"time"

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
	apiKey  string
	baseURL string
	http    *http.Client
	limiter *rate.Limiter
}

func NewLLMClient(apiKey, baseURL string) *LLMClient {
	return &LLMClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 120 * time.Second},
		limiter: rate.NewLimiter(rate.Limit(0.5), 1),
	}
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

	messages := []map[string]string{}
	if req.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.User})

	body := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	if req.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}

	bodyJSON, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := result.Choices[0].Message.Content
	duration := time.Since(start).Milliseconds()

	c.storeCache(cacheKey, content)

	return &LLMResponse{
		Content:    content,
		TokensIn:   result.Usage.PromptTokens,
		TokensOut:  result.Usage.CompletionTokens,
		Model:      result.Model,
		DurationMs: duration,
	}, nil
}

func (c *LLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	body := map[string]interface{}{
		"model": "openai/text-embedding-3-small",
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
	defer resp.Body.Close()

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

var llmCache = make(map[string]string)

func (c *LLMClient) cacheKey(req LLMRequest) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%f", req.Model, req.System, req.User, req.Temperature)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *LLMClient) checkCache(key string) string {
	return llmCache[key]
}

func (c *LLMClient) storeCache(key, value string) {
	llmCache[key] = value
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
