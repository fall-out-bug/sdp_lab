package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"sdp_dev/internal/control"
	"sdp_dev/internal/executor"
)

// Server exposes a minimal A2A-compatible JSON-RPC interface over the SDP control tower.
type Server struct {
	Store       *control.Store
	Bridge      *executor.ExecutorBridge
	ProjectRoot string
	Addr        string
	APIKey      string
}

type agentCard struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	URL          string       `json:"url"`
	Capabilities []string     `json:"capabilities"`
	Skills       []agentSkill `json:"skills"`
}

type agentSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type sendParams struct {
	Title      string `json:"title"`
	RawRequest string `json:"raw_request"`
	ProjectID  string `json:"project_id"`
}

type getParams struct {
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
}

type listParams struct {
	ProjectID string `json:"project_id"`
	Status    string `json:"status,omitempty"`
}

type cancelParams struct {
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized\n"))
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/.well-known/agent.json":
		s.writeJSON(w, http.StatusOK, s.buildAgentCard())
		return
	case r.Method == http.MethodPost && r.URL.Path == "/a2a":
		s.handleRPC(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("invalid JSON-RPC request: %v", err))
		return
	}
	if req.JSONRPC != "2.0" {
		s.writeRPCError(w, req.ID, -32600, "jsonrpc must be 2.0")
		return
	}

	var (
		result any
		err    error
	)

	switch req.Method {
	case "tasks/send":
		var params sendParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("invalid params: %v", err))
			return
		}
		result, err = s.handleTaskSend(r.Context(), params)
	case "tasks/get":
		var params getParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("invalid params: %v", err))
			return
		}
		result, err = s.handleTaskGet(params)
	case "tasks/list":
		var params listParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("invalid params: %v", err))
			return
		}
		result, err = s.handleTaskList(params)
	case "tasks/cancel":
		var params cancelParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("invalid params: %v", err))
			return
		}
		result, err = s.handleTaskCancel(params)
	default:
		s.writeRPCError(w, req.ID, -32600, fmt.Sprintf("unsupported method: %s", req.Method))
		return
	}
	if err != nil {
		s.writeRPCError(w, req.ID, -32600, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (s *Server) handleTaskSend(ctx context.Context, params sendParams) (map[string]any, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("store is not configured")
	}
	projectID := strings.TrimSpace(params.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	title := strings.TrimSpace(params.Title)
	if title == "" {
		title = strings.TrimSpace(params.RawRequest)
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	raw := strings.TrimSpace(params.RawRequest)
	if raw == "" {
		raw = title
	}

	card, err := s.Store.CreateCard(projectID, title, raw)
	if err != nil {
		return nil, err
	}
	card, err = s.Store.ClarifyCard(projectID, card.ID, raw, "feature", projectID, "medium", "dispatch_execution", []string{title}, []string{"completed feature implementation"})
	if err != nil {
		return nil, err
	}
	card, err = s.Store.MarkReady(projectID, card.ID)
	if err != nil {
		return nil, err
	}
	card, err = s.Store.DispatchCard(projectID, card.ID)
	if err != nil {
		return nil, err
	}

	response := map[string]any{
		"task_id":                card.ID,
		"project_id":             card.ProjectID,
		"status":                 card.Status,
		"executor_runtime_state": card.ExecutorRuntimeState,
	}
	if s.Bridge != nil {
		result, err := s.Bridge.DispatchAndRun(ctx, projectID, card.ID)
		if err != nil {
			return nil, err
		}
		fresh, loadErr := s.Store.LoadCard(projectID, card.ID)
		if loadErr == nil {
			card = fresh
			response["status"] = card.Status
			response["executor_runtime_state"] = card.ExecutorRuntimeState
		}
		response["result_summary"] = result.Summary
		response["result_status"] = result.Status
	}
	return response, nil
}

func (s *Server) handleTaskGet(params getParams) (map[string]any, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("store is not configured")
	}
	if strings.TrimSpace(params.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(params.TaskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	card, err := s.Store.LoadCard(params.ProjectID, params.TaskID)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"task_id":                card.ID,
		"project_id":             card.ProjectID,
		"title":                  card.Title,
		"status":                 card.Status,
		"executor_runtime_state": card.ExecutorRuntimeState,
		"executor_session_id":    card.ExecutorSessionID,
		"result_summary":         "",
		"result_status":          "",
	}
	if card.ExecutorResult != nil {
		result["result_summary"] = card.ExecutorResult.Summary
		result["result_status"] = card.ExecutorResult.Status
		result["recommended_next_step"] = card.ExecutorResult.RecommendedNextStep
	}
	return result, nil
}

func (s *Server) handleTaskList(params listParams) (map[string]any, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("store is not configured")
	}
	if strings.TrimSpace(params.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	cards, err := s.Store.LoadCards(params.ProjectID)
	if err != nil {
		return nil, err
	}
	statusFilter := strings.TrimSpace(params.Status)
	items := make([]map[string]any, 0, len(cards))
	for _, card := range cards {
		if statusFilter != "" && card.Status != statusFilter {
			continue
		}
		item := map[string]any{
			"task_id":                card.ID,
			"title":                  card.Title,
			"status":                 card.Status,
			"executor_runtime_state": card.ExecutorRuntimeState,
		}
		if card.ExecutorResult != nil {
			item["result_status"] = card.ExecutorResult.Status
			item["result_summary"] = card.ExecutorResult.Summary
		}
		items = append(items, item)
	}
	return map[string]any{"tasks": items}, nil
}

func (s *Server) handleTaskCancel(params cancelParams) (map[string]any, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("store is not configured")
	}
	if strings.TrimSpace(params.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(params.TaskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	card, err := s.Store.ParkCard(params.ProjectID, params.TaskID, "Cancelled via A2A tasks/cancel")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"task_id":    card.ID,
		"project_id": card.ProjectID,
		"status":     card.Status,
	}, nil
}

func (s *Server) buildAgentCard() agentCard {
	return agentCard{
		Name:         "sdp-control-tower",
		Description:  "SDP Spec-Driven Pipeline control tower — orchestrates features from intent to deploy",
		URL:          fmt.Sprintf("http://localhost%s/a2a", normalizeAddr(s.Addr)),
		Capabilities: []string{"streaming"},
		Skills: []agentSkill{
			{ID: "feature-dispatch", Name: "Dispatch feature to executor", Description: "Accept feature request, shape, contract, dispatch, execute"},
			{ID: "status-query", Name: "Query task status", Description: "Get FeatureCard status and board state"},
		},
	}
}

func normalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	return addr
}

func (s *Server) authorized(r *http.Request) bool {
	if strings.TrimSpace(s.APIKey) == "" {
		return true
	}
	want := "Bearer " + s.APIKey
	return r.Header.Get("Authorization") == want
}

func (s *Server) writeRPCError(w http.ResponseWriter, id any, code int, message string) {
	s.writeJSON(w, http.StatusOK, rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
