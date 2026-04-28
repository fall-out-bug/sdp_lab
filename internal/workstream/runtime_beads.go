package workstream

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/executil"
)

type RuntimeIssueState struct {
	ID        string
	IsOpen    bool
	IsClaimed bool
	Priority  int
	CreatedAt time.Time
	Status    string
	Assignee  string
}

type RuntimeQueryError struct {
	Code     string   `json:"code"`
	LeafWSID string   `json:"leaf_ws_id"`
	IssueIDs []string `json:"issue_ids"`
	Reason   string   `json:"reason"`
}

func (e *RuntimeQueryError) Error() string {
	return fmt.Sprintf("%s: leaf=%s reason=%s issues=%v", e.Code, e.LeafWSID, e.Reason, e.IssueIDs)
}

type RuntimeAdapter interface {
	QueryBoundIssues(ctx context.Context, leaf WorkstreamLock) ([]RuntimeIssueState, error)
	ClaimIssue(ctx context.Context, issueID string) error
	ReleaseClaim(ctx context.Context, issueID, restoreStatus string) error
}

type ShellBeadsRuntimeAdapter struct {
	ProjectRoot string
	Runner      executil.CommandRunner
	BDPath      string
}

func NewShellBeadsRuntimeAdapter(projectRoot string) *ShellBeadsRuntimeAdapter {
	return &ShellBeadsRuntimeAdapter{
		ProjectRoot: projectRoot,
		Runner:      executil.GetDefaultRunner(),
		BDPath:      "bd",
	}
}

func BoundIssueIDs(leaf WorkstreamLock) []string {
	ids := make([]string, 0, 1+len(leaf.FindingIssueIDs)+len(leaf.HistoricalIssueIDs))
	if leaf.BoundPrimaryIssueID != "" {
		ids = append(ids, leaf.BoundPrimaryIssueID)
	}
	ids = append(ids, leaf.FindingIssueIDs...)
	ids = append(ids, leaf.HistoricalIssueIDs...)
	slices.Sort(ids)
	return ids
}

func (a *ShellBeadsRuntimeAdapter) QueryBoundIssues(ctx context.Context, leaf WorkstreamLock) ([]RuntimeIssueState, error) {
	ids := BoundIssueIDs(leaf)
	if len(ids) == 0 {
		return nil, nil
	}
	if a.Runner == nil {
		a.Runner = executil.GetDefaultRunner()
	}
	if a.BDPath == "" {
		a.BDPath = "bd"
	}

	args := append([]string{"show"}, ids...)
	args = append(args, "--json")
	out, err := a.Runner.Output(ctx, a.ProjectRoot, a.BDPath, args...)
	if err != nil {
		return nil, &RuntimeQueryError{
			Code:     "beads_query_failed",
			LeafWSID: leaf.WSID,
			IssueIDs: ids,
			Reason:   "transport",
		}
	}

	var payload []struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Priority  int    `json:"priority"`
		CreatedAt string `json:"created_at"`
		Assignee  string `json:"assignee"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, &RuntimeQueryError{
			Code:     "beads_query_failed",
			LeafWSID: leaf.WSID,
			IssueIDs: ids,
			Reason:   "invalid_payload",
		}
	}

	stateByID := make(map[string]RuntimeIssueState, len(payload))
	for _, item := range payload {
		createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil {
			return nil, &RuntimeQueryError{
				Code:     "beads_query_failed",
				LeafWSID: leaf.WSID,
				IssueIDs: ids,
				Reason:   "invalid_payload",
			}
		}
		stateByID[item.ID] = RuntimeIssueState{
			ID:        item.ID,
			IsOpen:    item.Status != "closed",
			IsClaimed: strings.TrimSpace(item.Assignee) != "",
			Priority:  item.Priority,
			CreatedAt: createdAt.UTC(),
			Status:    item.Status,
			Assignee:  item.Assignee,
		}
	}

	states := make([]RuntimeIssueState, 0, len(ids))
	for _, id := range ids {
		state, ok := stateByID[id]
		if !ok {
			return nil, &RuntimeQueryError{
				Code:     "beads_query_failed",
				LeafWSID: leaf.WSID,
				IssueIDs: ids,
				Reason:   "not_found",
			}
		}
		states = append(states, state)
	}
	return states, nil
}

func (a *ShellBeadsRuntimeAdapter) ClaimIssue(ctx context.Context, issueID string) error {
	if a.Runner == nil {
		a.Runner = executil.GetDefaultRunner()
	}
	if a.BDPath == "" {
		a.BDPath = "bd"
	}
	_, err := a.Runner.CombinedOutput(ctx, a.ProjectRoot, a.BDPath, "update", issueID, "--claim", "--json")
	if err != nil {
		return fmt.Errorf("claim issue %s: %w", issueID, err)
	}
	return nil
}

func (a *ShellBeadsRuntimeAdapter) ReleaseClaim(ctx context.Context, issueID, restoreStatus string) error {
	if a.Runner == nil {
		a.Runner = executil.GetDefaultRunner()
	}
	if a.BDPath == "" {
		a.BDPath = "bd"
	}
	args := []string{"update", issueID}
	if strings.TrimSpace(restoreStatus) != "" {
		args = append(args, "--status", restoreStatus)
	}
	args = append(args, "-a", "", "--json")
	_, err := a.Runner.CombinedOutput(ctx, a.ProjectRoot, a.BDPath, args...)
	if err != nil {
		return fmt.Errorf("release claim for issue %s: %w", issueID, err)
	}
	return nil
}

func ResolveActiveIssue(leaf WorkstreamLock, states []RuntimeIssueState) *RuntimeIssueState {
	if len(states) == 0 {
		return nil
	}
	stateByID := make(map[string]RuntimeIssueState, len(states))
	for _, state := range states {
		stateByID[state.ID] = state
	}

	blocking := rankIssueStates(filterOpenIssues(leaf.FindingIssueIDs, stateByID, func(state RuntimeIssueState) bool {
		return state.Priority <= 1
	}))
	if len(blocking) > 0 {
		state := blocking[0]
		return &state
	}

	if leaf.BoundPrimaryIssueID != "" {
		if state, ok := stateByID[leaf.BoundPrimaryIssueID]; ok && state.IsOpen {
			return &state
		}
	}

	nonBlocking := rankIssueStates(filterOpenIssues(leaf.FindingIssueIDs, stateByID, func(state RuntimeIssueState) bool {
		return state.Priority >= 2
	}))
	if len(nonBlocking) > 0 {
		state := nonBlocking[0]
		return &state
	}

	return nil
}

func CompetingClaimedIssues(leaf WorkstreamLock, states []RuntimeIssueState, allowedIssueID string) []string {
	bound := make(map[string]bool, len(BoundIssueIDs(leaf)))
	for _, id := range BoundIssueIDs(leaf) {
		bound[id] = true
	}
	var conflicts []string
	for _, state := range states {
		if !bound[state.ID] || !state.IsClaimed || state.ID == allowedIssueID {
			continue
		}
		conflicts = append(conflicts, state.ID)
	}
	sort.Strings(conflicts)
	return conflicts
}

func FindIssueState(states []RuntimeIssueState, issueID string) (RuntimeIssueState, bool) {
	for _, state := range states {
		if state.ID == issueID {
			return state, true
		}
	}
	return RuntimeIssueState{}, false
}

func filterOpenIssues(ids []string, states map[string]RuntimeIssueState, include func(RuntimeIssueState) bool) []RuntimeIssueState {
	result := make([]RuntimeIssueState, 0, len(ids))
	for _, id := range ids {
		state, ok := states[id]
		if !ok || !state.IsOpen || !include(state) {
			continue
		}
		result = append(result, state)
	}
	return result
}

func rankIssueStates(states []RuntimeIssueState) []RuntimeIssueState {
	ranked := append([]RuntimeIssueState(nil), states...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Priority != ranked[j].Priority {
			return ranked[i].Priority < ranked[j].Priority
		}
		if !ranked[i].CreatedAt.Equal(ranked[j].CreatedAt) {
			return ranked[i].CreatedAt.Before(ranked[j].CreatedAt)
		}
		return ranked[i].ID < ranked[j].ID
	})
	return ranked
}
