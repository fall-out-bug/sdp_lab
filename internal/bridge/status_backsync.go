package bridge

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrEmptyGitHubReference      = errors.New("empty github issue reference")
	ErrUnsupportedGitHubRef      = errors.New("unsupported github issue reference")
	ErrMissingDefaultRepository  = errors.New("missing default repository")
	ErrUnsupportedBeadsStatus    = errors.New("unsupported beads status")
	ErrInvalidGitHubIssueNumber  = errors.New("invalid github issue number")
	ErrInvalidGitHubRepoIdentity = errors.New("invalid github repository identity")
)

type GitHubIssueRef struct {
	Repo   string
	Number int
}

type BackSyncIssue struct {
	BeadsID     string
	Status      string
	ExternalRef string
}

type BackSyncResult struct {
	Success   bool
	Attempts  int
	Repo      string
	Issue     int
	Labels    []string
	Comment   string
	LastError string
}

type BackSyncAuditEntry struct {
	Timestamp  string
	BeadsID    string
	GitHubRepo string
	GitHubID   int
	Status     string
	Attempt    int
	Success    bool
	Error      string
}

type BackSyncAuditor interface {
	Record(ctx context.Context, entry BackSyncAuditEntry) error
}

type NoopBackSyncAuditor struct{}

func (NoopBackSyncAuditor) Record(_ context.Context, _ BackSyncAuditEntry) error {
	return nil
}

type GitHubStatusClient interface {
	SetLabels(ctx context.Context, repo string, issueNumber int, labels []string) error
	CreateComment(ctx context.Context, repo string, issueNumber int, body string) error
}

type StatusBackSync struct {
	client     GitHubStatusClient
	auditor    BackSyncAuditor
	maxRetries int
	backoff    time.Duration
	now        func() time.Time
}

func NewStatusBackSync(client GitHubStatusClient, auditor BackSyncAuditor, maxRetries int, backoff time.Duration) *StatusBackSync {
	if maxRetries < 1 {
		maxRetries = 1
	}
	if backoff <= 0 {
		backoff = time.Second
	}
	if auditor == nil {
		auditor = NoopBackSyncAuditor{}
	}

	return &StatusBackSync{
		client:     client,
		auditor:    auditor,
		maxRetries: maxRetries,
		backoff:    backoff,
		now:        time.Now,
	}
}

func (s *StatusBackSync) SyncIssueStatus(ctx context.Context, issue BackSyncIssue, defaultRepo string) (BackSyncResult, error) {
	ref, err := ParseGitHubIssueRef(issue.ExternalRef, defaultRepo)
	if err != nil {
		return BackSyncResult{Success: false, LastError: err.Error()}, err
	}

	labels, comment, err := StatusBackSyncPayload(issue.BeadsID, issue.Status)
	if err != nil {
		return BackSyncResult{Success: false, Repo: ref.Repo, Issue: ref.Number, LastError: err.Error()}, err
	}

	result := BackSyncResult{Repo: ref.Repo, Issue: ref.Number, Labels: labels, Comment: comment}
	var lastErr error

	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		err = s.client.SetLabels(ctx, ref.Repo, ref.Number, labels)
		if err == nil {
			err = s.client.CreateComment(ctx, ref.Repo, ref.Number, comment)
		}

		auditEntry := BackSyncAuditEntry{
			Timestamp:  s.now().UTC().Format(time.RFC3339),
			BeadsID:    issue.BeadsID,
			GitHubRepo: ref.Repo,
			GitHubID:   ref.Number,
			Status:     normalizeStatus(issue.Status),
			Attempt:    attempt,
			Success:    err == nil,
		}
		if err != nil {
			auditEntry.Error = err.Error()
		}
		_ = s.auditor.Record(ctx, auditEntry)

		result.Attempts = attempt
		if err == nil {
			result.Success = true
			result.LastError = ""
			return result, nil
		}

		lastErr = err
		result.LastError = err.Error()
		if attempt == s.maxRetries {
			break
		}

		if waitErr := waitWithContext(ctx, s.backoff*time.Duration(attempt)); waitErr != nil {
			return result, waitErr
		}
	}

	return result, fmt.Errorf("status back-sync failed after %d attempts: %w", result.Attempts, lastErr)
}

func ParseGitHubIssueRef(externalRef, defaultRepo string) (GitHubIssueRef, error) {
	ref := strings.TrimSpace(externalRef)
	if ref == "" {
		return GitHubIssueRef{}, ErrEmptyGitHubReference
	}

	if strings.HasPrefix(ref, "gh:") {
		ref = strings.TrimPrefix(ref, "gh:")
	}

	if strings.HasPrefix(ref, "gh-") {
		if strings.TrimSpace(defaultRepo) == "" {
			return GitHubIssueRef{}, ErrMissingDefaultRepository
		}
		number, err := strconv.Atoi(strings.TrimPrefix(ref, "gh-"))
		if err != nil || number <= 0 {
			return GitHubIssueRef{}, ErrInvalidGitHubIssueNumber
		}
		return GitHubIssueRef{Repo: strings.TrimSpace(defaultRepo), Number: number}, nil
	}

	parts := strings.Split(ref, "#")
	if len(parts) != 2 {
		return GitHubIssueRef{}, ErrUnsupportedGitHubRef
	}

	repo := strings.TrimSpace(parts[0])
	if repo == "" || !strings.Contains(repo, "/") {
		return GitHubIssueRef{}, ErrInvalidGitHubRepoIdentity
	}

	number, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || number <= 0 {
		return GitHubIssueRef{}, ErrInvalidGitHubIssueNumber
	}

	return GitHubIssueRef{Repo: repo, Number: number}, nil
}

func StatusBackSyncPayload(beadsID, status string) ([]string, string, error) {
	normalized := normalizeStatus(status)
	label, err := statusLabel(normalized)
	if err != nil {
		return nil, "", err
	}

	labels := []string{"sdp/beads-sync", label}
	comment := fmt.Sprintf("SDP Beads issue `%s` status synchronized to `%s`.", strings.TrimSpace(beadsID), normalized)
	return labels, comment, nil
}

func statusLabel(status string) (string, error) {
	switch status {
	case "open":
		return "sdp/open", nil
	case "in_progress":
		return "sdp/in-progress", nil
	case "blocked":
		return "sdp/blocked", nil
	case "deferred":
		return "sdp/deferred", nil
	case "closed":
		return "sdp/done", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedBeadsStatus, status)
	}
}

func waitWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
