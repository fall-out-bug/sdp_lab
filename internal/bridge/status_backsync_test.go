package bridge

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeStatusClient struct {
	failSetLabelsUntil int
	failCommentUntil   int
	setLabelsCalls     int
	commentCalls       int
	lastRepo           string
	lastIssue          int
	lastLabels         []string
	lastComment        string
}

func (f *fakeStatusClient) SetLabels(_ context.Context, repo string, issueNumber int, labels []string) error {
	f.setLabelsCalls++
	f.lastRepo = repo
	f.lastIssue = issueNumber
	f.lastLabels = append([]string(nil), labels...)
	if f.setLabelsCalls <= f.failSetLabelsUntil {
		return errors.New("set labels transient failure")
	}
	return nil
}

func (f *fakeStatusClient) CreateComment(_ context.Context, repo string, issueNumber int, body string) error {
	f.commentCalls++
	f.lastRepo = repo
	f.lastIssue = issueNumber
	f.lastComment = body
	if f.commentCalls <= f.failCommentUntil {
		return errors.New("create comment transient failure")
	}
	return nil
}

type captureAuditor struct {
	entries []BackSyncAuditEntry
}

func (c *captureAuditor) Record(_ context.Context, entry BackSyncAuditEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestStatusBackSyncPayloadDeterministic(t *testing.T) {
	labelsA, commentA, err := StatusBackSyncPayload("sdplab-123", "in_progress")
	if err != nil {
		t.Fatalf("unexpected payload error: %v", err)
	}

	labelsB, commentB, err := StatusBackSyncPayload("sdplab-123", " in_progress ")
	if err != nil {
		t.Fatalf("unexpected payload error on normalized status: %v", err)
	}

	expected := []string{"sdp/beads-sync", "sdp/in-progress"}
	if !reflect.DeepEqual(labelsA, expected) {
		t.Fatalf("unexpected labels: %v", labelsA)
	}
	if !reflect.DeepEqual(labelsA, labelsB) {
		t.Fatalf("labels must be deterministic, got %v vs %v", labelsA, labelsB)
	}
	if commentA != commentB {
		t.Fatalf("comment must be deterministic, got %q vs %q", commentA, commentB)
	}
}

func TestParseGitHubIssueRef(t *testing.T) {
	ref, err := ParseGitHubIssueRef("gh:fall-out-bug/sdp_lab#42", "")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if ref.Repo != "fall-out-bug/sdp_lab" || ref.Number != 42 {
		t.Fatalf("unexpected parsed ref: %+v", ref)
	}

	ref, err = ParseGitHubIssueRef("gh-17", "fall-out-bug/sdp_lab")
	if err != nil {
		t.Fatalf("unexpected shorthand parse error: %v", err)
	}
	if ref.Repo != "fall-out-bug/sdp_lab" || ref.Number != 17 {
		t.Fatalf("unexpected shorthand ref: %+v", ref)
	}

	_, err = ParseGitHubIssueRef("gh-17", "")
	if err == nil {
		t.Fatalf("expected missing default repository error")
	}
}

func TestStatusBackSyncRetryAndAudit(t *testing.T) {
	client := &fakeStatusClient{failSetLabelsUntil: 2}
	auditor := &captureAuditor{}
	backsync := NewStatusBackSync(client, auditor, 4, time.Millisecond)

	result, err := backsync.SyncIssueStatus(context.Background(), BackSyncIssue{
		BeadsID:     "sdplab-123",
		Status:      "open",
		ExternalRef: "fall-out-bug/sdp_lab#77",
	}, "")
	if err != nil {
		t.Fatalf("expected eventual success, got error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success result, got %+v", result)
	}
	if result.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", result.Attempts)
	}
	if len(auditor.entries) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(auditor.entries))
	}
	if auditor.entries[0].Success || auditor.entries[1].Success || !auditor.entries[2].Success {
		t.Fatalf("unexpected audit success sequence: %+v", auditor.entries)
	}
}

func TestStatusBackSyncFailureAfterMaxRetries(t *testing.T) {
	client := &fakeStatusClient{failCommentUntil: 10}
	auditor := &captureAuditor{}
	backsync := NewStatusBackSync(client, auditor, 2, time.Millisecond)

	result, err := backsync.SyncIssueStatus(context.Background(), BackSyncIssue{
		BeadsID:     "sdplab-999",
		Status:      "blocked",
		ExternalRef: "fall-out-bug/sdp_lab#88",
	}, "")
	if err == nil {
		t.Fatalf("expected failure after max retries")
	}
	if result.Success {
		t.Fatalf("expected failed result, got %+v", result)
	}
	if result.Attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", result.Attempts)
	}
	if len(auditor.entries) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(auditor.entries))
	}
	if auditor.entries[0].Error == "" || auditor.entries[1].Error == "" {
		t.Fatalf("expected failure reasons in audit entries, got %+v", auditor.entries)
	}
}
