package pr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
