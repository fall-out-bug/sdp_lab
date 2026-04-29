package main

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/backlog"
)

func TestAddParentChildCountsUsesChildParentDependencies(t *testing.T) {
	features := []backlog.Feature{
		{BeadID: "sdplab-parent", FID: "F162", Title: "F162: Parent", Status: "open", IssueType: "epic"},
	}
	index := map[string]int{"sdplab-parent": 0}
	entries := []beadEntry{
		{
			ID:     "sdplab-child",
			Parent: "sdplab-parent",
			Dependencies: []struct {
				DependsOnID string `json:"depends_on_id"`
				Type        string `json:"type"`
			}{
				{DependsOnID: "sdplab-parent", Type: "parent-child"},
			},
		},
	}

	got := addParentChildCounts(features, index, entries)
	if got[0].DepCount != 1 {
		t.Fatalf("DepCount = %d, want 1", got[0].DepCount)
	}
}

func TestAddParentChildCountsIgnoresNonParentDependencies(t *testing.T) {
	features := []backlog.Feature{
		{BeadID: "sdplab-parent", FID: "F162", Title: "F162: Parent", Status: "open", IssueType: "epic"},
	}
	index := map[string]int{"sdplab-parent": 0}
	entries := []beadEntry{
		{
			ID: "sdplab-child",
			Dependencies: []struct {
				DependsOnID string `json:"depends_on_id"`
				Type        string `json:"type"`
			}{
				{DependsOnID: "sdplab-parent", Type: "blocks"},
			},
		},
	}

	got := addParentChildCounts(features, index, entries)
	if got[0].DepCount != 0 {
		t.Fatalf("DepCount = %d, want 0", got[0].DepCount)
	}
}
