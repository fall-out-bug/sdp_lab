package pr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type GateSignal struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type PublishRequest struct {
	IssueID             string
	RunID               string
	PRURL               string
	PRTitle             string
	Repository          string
	BaseBranch          string
	HeadBranch          string
	CommitIDs           []string
	PRGatePassed        bool
	GateSignals         []GateSignal
	PublishedAt         time.Time
	RunContextLink      string
	EvidenceContextLink string
}

type CallbackRecipientTarget struct {
	ID          string
	Address     string
	Required    bool
	AckRequired bool
}

type CallbackSender interface {
	Send(ctx context.Context, recipient CallbackRecipientTarget, headers map[string]string, payload []byte) (int, error)
}

type CallbackAttempt struct {
	Attempt    int    `json:"attempt"`
	StatusCode int    `json:"status_code"`
	Retryable  bool   `json:"retryable"`
	Delivered  bool   `json:"delivered"`
	Error      string `json:"error,omitempty"`
}

type CallbackDispatchResult struct {
	RecipientID  string            `json:"recipient_id"`
	Address      string            `json:"address"`
	Delivered    bool              `json:"delivered"`
	DeadLettered bool              `json:"dead_lettered"`
	Reason       string            `json:"reason,omitempty"`
	Headers      map[string]string `json:"headers"`
	Attempts     []CallbackAttempt `json:"attempts"`
}

type CallbackDispatchReport struct {
	Channel        string                   `json:"channel"`
	IdempotencyKey string                   `json:"idempotency_key"`
	Payload        map[string]any           `json:"payload"`
	Results        []CallbackDispatchResult `json:"results"`
}

func BuildPublishPayload(req PublishRequest) (map[string]any, error) {
	req = normalizePublishRequest(req)
	if err := validatePublishRequest(req); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"event": "pr.published",
		"issue": map[string]any{
			"id": req.IssueID,
		},
		"trace": map[string]any{
			"run_id":                req.RunID,
			"pr_url":                req.PRURL,
			"commit_ids":            append([]string(nil), req.CommitIDs...),
			"run_context_link":      req.RunContextLink,
			"evidence_context_link": req.EvidenceContextLink,
		},
		"pr": map[string]any{
			"title":       req.PRTitle,
			"repository":  req.Repository,
			"base_branch": req.BaseBranch,
			"head_branch": req.HeadBranch,
		},
		"gates": map[string]any{
			"pr_gate_passed": req.PRGatePassed,
			"signals":        copySignals(req.GateSignals),
		},
		"published_at": req.PublishedAt.UTC().Format(time.RFC3339),
	}

	if err := validatePayloadContract(payload, DefaultPRPayloadContract()); err != nil {
		return nil, err
	}
	return payload, nil
}

func BuildCallbackHeaders(payload map[string]any) (map[string]string, string, error) {
	issueID, _ := getStringAtPath(payload, "issue.id")
	runID, _ := getStringAtPath(payload, "trace.run_id")
	prURL, _ := getStringAtPath(payload, "trace.pr_url")
	event, _ := getStringAtPath(payload, "event")
	publishedAt, _ := getStringAtPath(payload, "published_at")
	if issueID == "" || runID == "" || prURL == "" || event == "" || publishedAt == "" {
		return nil, "", fmt.Errorf("payload missing callback header source fields")
	}
	idempotencyKey := issueID + ":" + runID + ":" + prURL
	headers := map[string]string{
		"x-sdp-event":           event,
		"x-sdp-issue-id":        issueID,
		"x-sdp-run-id":          runID,
		"x-sdp-idempotency-key": idempotencyKey,
		"x-sdp-published-at":    publishedAt,
	}

	contract := DefaultCallbackChannelContract()
	for _, header := range contract.RequiredHeaders {
		if strings.TrimSpace(headers[header]) == "" {
			return nil, "", fmt.Errorf("missing required callback header %q", header)
		}
	}

	return headers, idempotencyKey, nil
}

func DispatchCallbacks(ctx context.Context, sender CallbackSender, payload map[string]any, recipients []CallbackRecipientTarget, routeMode string) (CallbackDispatchReport, error) {
	if sender == nil {
		return CallbackDispatchReport{}, errors.New("callback sender is required")
	}
	headers, idempotencyKey, err := BuildCallbackHeaders(payload)
	if err != nil {
		return CallbackDispatchReport{}, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return CallbackDispatchReport{}, fmt.Errorf("marshal callback payload: %w", err)
	}

	policy := DefaultCallbackRoutingReliabilityContract()
	orderedRecipients := orderRecipients(recipients, routeMode)
	results := make([]CallbackDispatchResult, 0, len(orderedRecipients))

	for _, recipient := range orderedRecipients {
		result := CallbackDispatchResult{
			RecipientID: recipient.ID,
			Address:     recipient.Address,
			Headers:     cloneHeaders(headers),
			Attempts:    make([]CallbackAttempt, 0, len(policy.RetryBudget)),
		}

		if strings.TrimSpace(recipient.Address) == "" {
			if recipient.Required {
				result.DeadLettered = true
				result.Reason = "missing-address"
			} else {
				result.Reason = "skipped-missing-address"
			}
			results = append(results, result)
			continue
		}

		for i, stage := range policy.RetryBudget {
			statusCode, sendErr := sender.Send(ctx, recipient, headers, body)
			retryable := sendErr != nil || IsRetryableCallbackStatus(statusCode)
			delivered := sendErr == nil && statusCode >= 200 && statusCode <= 299
			attempt := CallbackAttempt{
				Attempt:    stage.Attempt,
				StatusCode: statusCode,
				Retryable:  retryable,
				Delivered:  delivered,
			}
			if sendErr != nil {
				attempt.Error = sendErr.Error()
			}
			result.Attempts = append(result.Attempts, attempt)

			if delivered {
				result.Delivered = true
				result.Reason = "delivered"
				break
			}

			lastAttempt := i == len(policy.RetryBudget)-1
			if !retryable {
				if recipient.Required {
					result.DeadLettered = true
					result.Reason = "non-retryable-failure"
				} else {
					result.Reason = "optional-non-retryable-failure"
				}
				break
			}

			if lastAttempt {
				if recipient.Required {
					result.DeadLettered = true
					result.Reason = policy.DeadLetterReason
				} else {
					result.Reason = "optional-retry-budget-exhausted"
				}
			}
		}

		results = append(results, result)
	}

	return CallbackDispatchReport{
		Channel:        DefaultCallbackChannelContract().Channel,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
		Results:        results,
	}, nil
}

func normalizePublishRequest(req PublishRequest) PublishRequest {
	if req.PublishedAt.IsZero() {
		req.PublishedAt = time.Now().UTC()
	}
	req.IssueID = strings.TrimSpace(req.IssueID)
	req.RunID = strings.TrimSpace(req.RunID)
	req.PRURL = strings.TrimSpace(req.PRURL)
	req.PRTitle = strings.TrimSpace(req.PRTitle)
	req.Repository = strings.TrimSpace(req.Repository)
	req.BaseBranch = strings.TrimSpace(req.BaseBranch)
	req.HeadBranch = strings.TrimSpace(req.HeadBranch)
	req.RunContextLink = strings.TrimSpace(req.RunContextLink)
	req.EvidenceContextLink = strings.TrimSpace(req.EvidenceContextLink)
	commitIDs := make([]string, 0, len(req.CommitIDs))
	for _, commitID := range req.CommitIDs {
		trimmed := strings.TrimSpace(commitID)
		if trimmed != "" {
			commitIDs = append(commitIDs, trimmed)
		}
	}
	req.CommitIDs = commitIDs
	return req
}

func validatePublishRequest(req PublishRequest) error {
	if req.IssueID == "" {
		return errors.New("issue id is required")
	}
	if req.RunID == "" {
		return errors.New("run id is required")
	}
	if req.PRURL == "" {
		return errors.New("pr url is required")
	}
	if req.PRTitle == "" {
		return errors.New("pr title is required")
	}
	if req.Repository == "" {
		return errors.New("repository is required")
	}
	if req.BaseBranch == "" {
		return errors.New("base branch is required")
	}
	if req.HeadBranch == "" {
		return errors.New("head branch is required")
	}
	if len(req.CommitIDs) == 0 {
		return errors.New("at least one commit id is required")
	}
	if req.RunContextLink == "" {
		return errors.New("run context link is required")
	}
	if req.EvidenceContextLink == "" {
		return errors.New("evidence context link is required")
	}
	return nil
}

func validatePayloadContract(payload map[string]any, contract PRPayloadContract) error {
	missing := make([]string, 0)
	for _, field := range contract.RequiredFields {
		value, ok := getAtPath(payload, field.Path)
		if !ok || isZeroValue(value) {
			missing = append(missing, field.Path)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("payload missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func orderRecipients(recipients []CallbackRecipientTarget, routeMode string) []CallbackRecipientTarget {
	ordered := append([]CallbackRecipientTarget(nil), recipients...)
	if routeMode == "fanout-all" {
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
		return ordered
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Required != ordered[j].Required {
			return ordered[i].Required
		}
		return ordered[i].ID < ordered[j].ID
	})
	return ordered
}

func getAtPath(payload map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = payload
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := m[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func getStringAtPath(payload map[string]any, path string) (string, bool) {
	v, ok := getAtPath(payload, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok
}

func isZeroValue(v any) bool {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value) == ""
	case []string:
		return len(value) == 0
	case []any:
		return len(value) == 0
	case nil:
		return true
	}
	return false
}

func copySignals(in []GateSignal) []map[string]string {
	out := make([]map[string]string, 0, len(in))
	for _, signal := range in {
		out = append(out, map[string]string{
			"name":   strings.TrimSpace(signal.Name),
			"status": strings.TrimSpace(signal.Status),
		})
	}
	return out
}

func cloneHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		out[k] = v
	}
	return out
}
