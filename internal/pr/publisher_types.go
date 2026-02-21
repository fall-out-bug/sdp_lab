package pr

import (
	"context"
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
