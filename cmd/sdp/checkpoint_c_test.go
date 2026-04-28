// cmd/sdp/checkpoint_c_test.go
package main

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

// makeTestScanResult builds a ScanResult with one flagged + one settled item.
func makeTestScanResult() *discovery.ScanResult {
	flag := &discovery.DepthFlag{
		Flagged:  true,
		Blocking: true,
		Reason:   "no_primary_source",
	}
	return &discovery.ScanResult{
		Items: []discovery.ScanItem{
			{Name: "SettledTool", Disposition: discovery.DispositionInspire, CoverageScore: 0.7},
			{Name: "FlaggedTool", Disposition: discovery.DispositionAdopt, CoverageScore: 0.04, DepthFlag: flag},
		},
		Whitespace: "gap description",
	}
}

func TestResolveCheckpointC_NonInteractiveUsesDefaults(t *testing.T) {
	scan := makeTestScanResult()
	// Non-interactive: should apply default resolution (proceed provisional) without blocking.
	resolutions := resolveCheckpointC(scan, false, nil)
	// FlaggedTool should get a resolution.
	if _, ok := resolutions["FlaggedTool"]; !ok {
		t.Error("expected resolution for FlaggedTool in non-interactive mode")
	}
	// SettledTool should have no resolution (it is not flagged).
	if _, ok := resolutions["SettledTool"]; ok {
		t.Error("unexpected resolution for SettledTool (not flagged)")
	}
}

func TestResolveCheckpointC_DefaultResolutionIsProceedProvisional(t *testing.T) {
	scan := makeTestScanResult()
	resolutions := resolveCheckpointC(scan, false, nil)
	res := resolutions["FlaggedTool"]
	if !strings.Contains(res, "proceed") && !strings.Contains(res, "provisional") {
		t.Errorf("expected proceed_provisional default, got %q", res)
	}
}
