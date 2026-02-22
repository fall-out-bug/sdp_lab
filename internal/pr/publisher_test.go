package pr

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubCallbackSender struct {
	attemptsByRecipient map[string]int
	responses           map[string][]stubSendResponse
}

type stubSendResponse struct {
	statusCode int
	err        error
}

func (s *stubCallbackSender) Send(_ context.Context, recipient CallbackRecipientTarget, _ map[string]string, _ []byte) (int, error) {
	if s.attemptsByRecipient == nil {
		s.attemptsByRecipient = map[string]int{}
	}
	idx := s.attemptsByRecipient[recipient.ID]
	s.attemptsByRecipient[recipient.ID] = idx + 1
	queue := s.responses[recipient.ID]
	if len(queue) == 0 {
		return 202, nil
	}
	if idx >= len(queue) {
		last := queue[len(queue)-1]
		return last.statusCode, last.err
	}
	resp := queue[idx]
	return resp.statusCode, resp.err
}

func TestBuildPublishPayloadIncludesRunAndEvidenceLinks(t *testing.T) {
	req := PublishRequest{
		IssueID:             "sdp_dev-2aq.17.2",
		RunID:               "run-123",
		PRURL:               "https://github.com/org/repo/pull/10",
		PRTitle:             "BUILD: publish callbacks",
		Repository:          "org/repo",
		BaseBranch:          "master",
		HeadBranch:          "feature/pr-callback",
		CommitIDs:           []string{"abc123"},
		PRGatePassed:        true,
		GateSignals:         []GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}},
		PublishedAt:         time.Date(2026, 2, 20, 20, 0, 0, 0, time.UTC),
		RunContextLink:      ".sdp/runs/sdp_dev-2aq.17.2.json",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.17.2.json",
	}

	payload, err := BuildPublishPayload(req)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	runLink, _ := getStringAtPath(payload, "trace.run_context_link")
	if runLink != req.RunContextLink {
		t.Fatalf("unexpected run context link: got=%q want=%q", runLink, req.RunContextLink)
	}
	evidenceLink, _ := getStringAtPath(payload, "trace.evidence_context_link")
	if evidenceLink != req.EvidenceContextLink {
		t.Fatalf("unexpected evidence context link: got=%q want=%q", evidenceLink, req.EvidenceContextLink)
	}

	headers, key, err := BuildCallbackHeaders(payload)
	if err != nil {
		t.Fatalf("build headers: %v", err)
	}
	if key != "sdp_dev-2aq.17.2:run-123:https://github.com/org/repo/pull/10" {
		t.Fatalf("unexpected idempotency key: %s", key)
	}
	if got := headers["x-sdp-run-id"]; got != req.RunID {
		t.Fatalf("unexpected run id header: %s", got)
	}
}

func TestDispatchCallbacksRetriesRetryableThenDelivers(t *testing.T) {
	payload, err := BuildPublishPayload(PublishRequest{
		IssueID:             "sdp_dev-2aq.17.2",
		RunID:               "run-1",
		PRURL:               "https://example/pull/1",
		PRTitle:             "title",
		Repository:          "org/repo",
		BaseBranch:          "master",
		HeadBranch:          "feat",
		CommitIDs:           []string{"abc"},
		PRGatePassed:        true,
		GateSignals:         []GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}},
		PublishedAt:         time.Date(2026, 2, 20, 20, 0, 0, 0, time.UTC),
		RunContextLink:      ".sdp/runs/sdp_dev-2aq.17.2.json",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.17.2.json",
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	sender := &stubCallbackSender{
		responses: map[string][]stubSendResponse{
			"issue-owner": {
				{statusCode: 503},
				{statusCode: 202},
			},
		},
	}
	report, err := DispatchCallbacks(context.Background(), sender, payload, []CallbackRecipientTarget{{ID: "issue-owner", Address: "owner@example.com", Required: true, AckRequired: true}}, "required-first")
	if err != nil {
		t.Fatalf("dispatch callbacks: %v", err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("unexpected result count: %d", len(report.Results))
	}
	result := report.Results[0]
	if !result.Delivered {
		t.Fatalf("expected delivered result: %+v", result)
	}
	if result.DeadLettered {
		t.Fatalf("did not expect dead letter: %+v", result)
	}
	if len(result.Attempts) != 2 {
		t.Fatalf("expected two attempts, got %d", len(result.Attempts))
	}
	if !result.Attempts[0].Retryable || result.Attempts[0].Delivered {
		t.Fatalf("expected first attempt retryable and not delivered: %+v", result.Attempts[0])
	}
}

func TestDispatchCallbacksDeadLettersAfterRetryBudget(t *testing.T) {
	payload, err := BuildPublishPayload(PublishRequest{
		IssueID:             "sdp_dev-2aq.17.2",
		RunID:               "run-2",
		PRURL:               "https://example/pull/2",
		PRTitle:             "title",
		Repository:          "org/repo",
		BaseBranch:          "master",
		HeadBranch:          "feat",
		CommitIDs:           []string{"abc"},
		PRGatePassed:        true,
		GateSignals:         []GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}},
		PublishedAt:         time.Date(2026, 2, 20, 21, 0, 0, 0, time.UTC),
		RunContextLink:      ".sdp/runs/sdp_dev-2aq.17.2.json",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.17.2.json",
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	sender := &stubCallbackSender{
		responses: map[string][]stubSendResponse{
			"issue-owner": {
				{statusCode: 503},
				{statusCode: 503},
				{statusCode: 503},
				{statusCode: 503},
				{statusCode: 503},
				{statusCode: 503},
				{statusCode: 503},
			},
		},
	}
	report, err := DispatchCallbacks(context.Background(), sender, payload, []CallbackRecipientTarget{{ID: "issue-owner", Address: "owner@example.com", Required: true, AckRequired: true}}, "required-first")
	if err != nil {
		t.Fatalf("dispatch callbacks: %v", err)
	}
	result := report.Results[0]
	if result.Delivered {
		t.Fatalf("unexpected delivery result: %+v", result)
	}
	if !result.DeadLettered {
		t.Fatalf("expected dead letter result: %+v", result)
	}
	if result.Reason != "retry-window-exhausted" {
		t.Fatalf("unexpected dead letter reason: %s", result.Reason)
	}
	if len(result.Attempts) != 7 {
		t.Fatalf("unexpected attempts count: %d", len(result.Attempts))
	}
}

func TestDispatchCallbacksDoesNotRetryNonRetryableFailure(t *testing.T) {
	payload, err := BuildPublishPayload(PublishRequest{
		IssueID:             "sdp_dev-2aq.17.2",
		RunID:               "run-3",
		PRURL:               "https://example/pull/3",
		PRTitle:             "title",
		Repository:          "org/repo",
		BaseBranch:          "master",
		HeadBranch:          "feat",
		CommitIDs:           []string{"abc"},
		PRGatePassed:        true,
		GateSignals:         []GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}},
		PublishedAt:         time.Date(2026, 2, 20, 22, 0, 0, 0, time.UTC),
		RunContextLink:      ".sdp/runs/sdp_dev-2aq.17.2.json",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.17.2.json",
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	sender := &stubCallbackSender{
		responses: map[string][]stubSendResponse{
			"issue-owner": {
				{statusCode: 400},
			},
			"orchestrator-audit": {
				{err: errors.New("transport down")},
				{statusCode: 202},
			},
		},
	}
	report, err := DispatchCallbacks(context.Background(), sender, payload, []CallbackRecipientTarget{
		{ID: "issue-owner", Address: "owner@example.com", Required: true, AckRequired: true},
		{ID: "orchestrator-audit", Address: "audit://callbacks", Required: true, AckRequired: true},
	}, "required-first")
	if err != nil {
		t.Fatalf("dispatch callbacks: %v", err)
	}
	if len(report.Results) != 2 {
		t.Fatalf("unexpected result count: %d", len(report.Results))
	}
	owner := report.Results[0]
	if len(owner.Attempts) != 1 {
		t.Fatalf("expected one attempt for non-retryable failure, got %d", len(owner.Attempts))
	}
	if owner.Attempts[0].Retryable {
		t.Fatalf("expected non-retryable owner failure: %+v", owner.Attempts[0])
	}
	audit := report.Results[1]
	if !audit.Delivered {
		t.Fatalf("expected audit recipient eventually delivered: %+v", audit)
	}
	if len(audit.Attempts) != 2 {
		t.Fatalf("expected two attempts for audit recipient, got %d", len(audit.Attempts))
	}
}

func TestDispatchCallbacksRequiredFirstAndMissingRequiredAddress(t *testing.T) {
	payload, err := BuildPublishPayload(PublishRequest{
		IssueID:             "sdp_dev-2aq.17.2",
		RunID:               "run-4",
		PRURL:               "https://example/pull/4",
		PRTitle:             "title",
		Repository:          "org/repo",
		BaseBranch:          "master",
		HeadBranch:          "feat",
		CommitIDs:           []string{"abc"},
		PRGatePassed:        true,
		GateSignals:         []GateSignal{{Name: "publish:pr-gate-pass", Status: "pass"}},
		PublishedAt:         time.Date(2026, 2, 20, 23, 0, 0, 0, time.UTC),
		RunContextLink:      ".sdp/runs/sdp_dev-2aq.17.2.json",
		EvidenceContextLink: ".sdp/evidence/sdp_dev-2aq.17.2.json",
	})
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	sender := &stubCallbackSender{responses: map[string][]stubSendResponse{}}
	report, err := DispatchCallbacks(context.Background(), sender, payload, []CallbackRecipientTarget{
		{ID: "watchers", Address: "watchers@example.com", Required: false, AckRequired: false},
		{ID: "issue-owner", Address: "", Required: true, AckRequired: true},
		{ID: "orchestrator-audit", Address: "audit://callbacks", Required: true, AckRequired: true},
	}, "required-first")
	if err != nil {
		t.Fatalf("dispatch callbacks: %v", err)
	}
	if len(report.Results) != 3 {
		t.Fatalf("unexpected result count: %d", len(report.Results))
	}
	if report.Results[0].RecipientID != "issue-owner" {
		t.Fatalf("expected required recipients first, got %s", report.Results[0].RecipientID)
	}
	if !report.Results[0].DeadLettered || report.Results[0].Reason != "missing-address" {
		t.Fatalf("expected missing required address to dead-letter, got %+v", report.Results[0])
	}
	if report.Results[2].RecipientID != "watchers" {
		t.Fatalf("expected optional recipient last, got %s", report.Results[2].RecipientID)
	}
}
