package omoclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// OmOServeClient is an HTTP client for OmO serve API
type OmOServeClient struct {
	baseURL string
	client  *http.Client
	logger  *log.Logger
}

// NewClient creates a new OmOServeClient instance
func NewClient(baseURL string, logger *log.Logger) *OmOServeClient {
	return &OmOServeClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// CreateSession creates a new OmO serve session
func (c *OmOServeClient) CreateSession(req CreateSessionRequest) (*SessionInfo, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create session request: %w", err)
	}

	resp, err := c.client.Post(
		c.baseURL+"/api/sessions",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create session failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var session SessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode create session response: %w", err)
	}

	return &session, nil
}

// GetSession retrieves session information by ID
func (c *OmOServeClient) GetSession(id string) (*SessionInfo, error) {
	url := fmt.Sprintf("%s/api/sessions/%s", c.baseURL, url.PathEscape(id))

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get session failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var session SessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("decode get session response: %w", err)
	}

	return &session, nil
}

// ListSessions returns all active sessions
func (c *OmOServeClient) ListSessions() ([]SessionInfo, error) {
	resp, err := c.client.Get(c.baseURL + "/api/sessions")
	if err != nil {
		return nil, fmt.Errorf("list sessions request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list sessions failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var sessions []SessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("decode list sessions response: %w", err)
	}

	return sessions, nil
}

// DeleteSession deletes a session by ID
func (c *OmOServeClient) DeleteSession(id string) error {
	url := fmt.Sprintf("%s/api/sessions/%s", c.baseURL, url.PathEscape(id))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete session failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendMessageStream sends a message and returns SSE stream response
func (c *OmOServeClient) SendMessageStream(content string) (*http.Response, error) {
	req := SendMessageRequest{Content: content}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal send message request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sessions/default/messages", c.baseURL)
	resp, err := c.client.Post(
		url,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("send message request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("send message failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}
