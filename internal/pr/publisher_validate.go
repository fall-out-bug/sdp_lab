package pr

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

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
