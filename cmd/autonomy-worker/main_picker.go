package main

import (
	"encoding/json"
	"fmt"
	"sort"
)

func listIssues() (map[string]issue, error) {
	out, err := bdRunner("list", "--json")
	if err != nil {
		return nil, err
	}
	var items []issue
	if err := json.Unmarshal(extractJSON(out), &items); err != nil {
		return nil, err
	}
	byID := make(map[string]issue, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	return byID, nil
}

func loadIssueDetail(issueID string) (issue, error) {
	out, err := bdRunner("show", issueID, "--json")
	if err != nil {
		return issue{}, err
	}
	var list []issue
	jsonOut := extractJSON(out)
	if err := json.Unmarshal(jsonOut, &list); err == nil && len(list) > 0 {
		return list[0], nil
	}
	var it issue
	if err := json.Unmarshal(jsonOut, &it); err != nil {
		return issue{}, err
	}
	return it, nil
}

func hasLabel(labels []string, name string) bool {
	for _, v := range labels {
		if v == name {
			return true
		}
	}
	return false
}

func hasWorkstreamLabel(labels []string) bool {
	for _, l := range labels {
		for _, w := range supportedWorkstreams {
			if l == w {
				return true
			}
		}
	}
	return false
}

func depsSatisfied(it issue, byID map[string]issue) bool {
	for _, d := range it.Dependencies {
		if d.IssueID != "" && d.IssueID != it.ID {
			continue
		}
		if d.kind() == "parent-child" {
			continue
		}
		if d.Status != "" {
			if d.Status == "closed" || d.Status == "done" {
				continue
			}
			return false
		}
		depIssue, ok := byID[d.refID()]
		if !ok {
			return false
		}
		if depIssue.Status != "closed" && depIssue.Status != "done" {
			return false
		}
	}
	return true
}

func pickCandidate(byID map[string]issue, debug bool) (*issue, error) {
	items := make([]issue, 0)
	for _, it := range byID {
		if it.IssueType != "task" {
			if debug {
				fmt.Printf("skip %s: issue_type=%s\n", it.ID, it.IssueType)
			}
			continue
		}
		if it.Status != "open" {
			if debug {
				fmt.Printf("skip %s: status=%s\n", it.ID, it.Status)
			}
			continue
		}
		if !hasLabel(it.Labels, "autonomy") {
			if debug {
				fmt.Printf("skip %s: missing label autonomy\n", it.ID)
			}
			continue
		}
		if !hasLabel(it.Labels, "strict-evidence") {
			if debug {
				fmt.Printf("skip %s: missing label strict-evidence\n", it.ID)
			}
			continue
		}
		if !hasWorkstreamLabel(it.Labels) {
			if debug {
				fmt.Printf("skip %s: no supported workstream label\n", it.ID)
			}
			continue
		}
		detail, err := loadIssueDetail(it.ID)
		if err != nil {
			if debug {
				fmt.Printf("skip %s: load issue detail failed: %v\n", it.ID, err)
			}
			continue
		}
		if depsSatisfied(detail, byID) {
			items = append(items, detail)
		} else if debug {
			fmt.Printf("skip %s: dependencies not satisfied\n", it.ID)
		}
	}
	if len(items) == 0 {
		return nil, nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return &items[0], nil
}
