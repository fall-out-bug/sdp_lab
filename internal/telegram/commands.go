package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GatewayClient calls the intake-gateway API.
type GatewayClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewGatewayClient returns a client for the intake gateway.
func NewGatewayClient(baseURL string) *GatewayClient {
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	return &GatewayClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{},
	}
}

// Intake creates a task via the intake API.
func (g *GatewayClient) Intake(projectID, title string) (map[string]any, error) {
	body := map[string]any{
		"project_id": projectID,
		"title":      title,
		"source":     "telegram",
	}
	payload, _ := json.Marshal(body)
	resp, err := g.HTTPClient.Post(g.BaseURL+"/api/v1/intake", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// DiscussSubmit starts a discussion session.
func (g *GatewayClient) DiscussSubmit(projectID, title, description string) (map[string]any, error) {
	body := map[string]any{
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"source":      "telegram",
	}
	payload, _ := json.Marshal(body)
	resp, err := g.HTTPClient.Post(g.BaseURL+"/api/v1/discuss", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// DiscussStatus gets discussion session status.
func (g *GatewayClient) DiscussStatus(sessionID string) (map[string]any, error) {
	resp, err := g.HTTPClient.Get(g.BaseURL + "/api/v1/discuss/" + sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// DiscussApprove approves a discussion and creates Beads issues.
func (g *GatewayClient) DiscussApprove(sessionID string) (map[string]any, error) {
	resp, err := g.HTTPClient.Post(g.BaseURL+"/api/v1/discuss/"+sessionID+"/approve", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// Projects lists projects from the registry.
func (g *GatewayClient) Projects() ([]map[string]any, error) {
	resp, err := g.HTTPClient.Get(g.BaseURL + "/api/v1/projects")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// HandleCommand processes a command and returns the reply text.
func HandleCommand(cmd, args string, chatID int64, state *StateStore, gw *GatewayClient) string {
	switch cmd {
	case "task":
		project := "default"
		title := strings.TrimSpace(args)
		for i, c := range args {
			if c == ' ' {
				project = strings.TrimSpace(args[:i])
				title = strings.TrimSpace(args[i+1:])
				break
			}
		}
		if title == "" {
			return "Usage: /task [project] <title>"
		}
		res, err := gw.Intake(project, title)
		if err != nil {
			return "Error: " + err.Error()
		}
		id, _ := res["id"].(string)
		return fmt.Sprintf("Task queued: %s", id)
	case "feature":
		if args == "" {
			return "Usage: /feature <description>"
		}
		res, err := gw.DiscussSubmit("default", args, args)
		if err != nil {
			return "Error: " + err.Error()
		}
		id, _ := res["id"].(string)
		phase, _ := res["phase"].(string)
		st := state.Get(chatID)
		st.ActiveDiscussID = id
		st.PendingApprove = id
		state.Set(chatID, st)
		return fmt.Sprintf("Discussion started: %s (phase: %s). Use /approve to create Beads issues.", id, phase)
	case "status":
		sessionID := strings.TrimSpace(args)
		if sessionID == "" {
			st := state.Get(chatID)
			sessionID = st.ActiveDiscussID
		}
		if sessionID == "" {
			return "Usage: /status [session_id]"
		}
		res, err := gw.DiscussStatus(sessionID)
		if err != nil {
			return "Error: " + err.Error()
		}
		phase, _ := res["phase"].(string)
		return fmt.Sprintf("Session %s: %s", sessionID, phase)
	case "approve":
		sessionID := strings.TrimSpace(args)
		if sessionID == "" {
			st := state.Get(chatID)
			sessionID = st.PendingApprove
		}
		if sessionID == "" {
			return "Usage: /approve [session_id]"
		}
		res, err := gw.DiscussApprove(sessionID)
		if err != nil {
			return "Error: " + err.Error()
		}
		st := state.Get(chatID)
		st.PendingApprove = ""
		state.Set(chatID, st)
		ids, _ := res["created_issues"].([]interface{})
		return fmt.Sprintf("Approved. Created %d issues.", len(ids))
	case "models":
		return "Available: glm-5, glm-4.7, openai/gpt-5.2-codex, anthropic/claude-sonnet-4.6"
	case "swarm":
		projs, err := gw.Projects()
		if err != nil {
			return "Error: " + err.Error()
		}
		return fmt.Sprintf("Projects: %d registered", len(projs))
	default:
		return "Commands: /task, /feature, /status, /approve, /models, /swarm"
	}
}
