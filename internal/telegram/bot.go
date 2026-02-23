package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const apiBase = "https://api.telegram.org/bot"

// Client is a Telegram Bot API client.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

// NewClient returns a client for the given bot token.
func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// SendMessage sends a text message to a chat.
func (c *Client) SendMessage(chatID int64, text string) error {
	url := apiBase + c.Token + "/sendMessage"
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api status %d", resp.StatusCode)
	}
	return nil
}
