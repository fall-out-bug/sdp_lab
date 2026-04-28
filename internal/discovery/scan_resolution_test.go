package discovery_test

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestApplyResolutions_DowngradeChangesDisposition(t *testing.T) {
	item := discovery.ScanItem{
		Name:        "FlaggedTool",
		Disposition: discovery.DispositionAdopt,
		DepthFlag:   &discovery.DepthFlag{Flagged: true, Reason: "no_primary_source"},
	}
	result := &discovery.ScanResult{Items: []discovery.ScanItem{item}}

	resolutions := map[string]string{
		"FlaggedTool": "downgrade",
	}
	updated := discovery.ApplyResolutions(result, resolutions)
	if updated.Items[0].Disposition != discovery.DispositionMonitor {
		t.Errorf("expected MONITOR after downgrade, got %s", updated.Items[0].Disposition)
	}
	if updated.Items[0].DepthFlag != nil {
		t.Error("downgrade should clear DepthFlag to nil")
	}
}

func TestApplyResolutions_ProceedProvisionalPreservesDisposition(t *testing.T) {
	item := discovery.ScanItem{
		Name:        "FlaggedTool",
		Disposition: discovery.DispositionAdopt,
		DepthFlag:   &discovery.DepthFlag{Flagged: true},
	}
	result := &discovery.ScanResult{Items: []discovery.ScanItem{item}}
	resolutions := map[string]string{"FlaggedTool": "proceed_provisional"}
	updated := discovery.ApplyResolutions(result, resolutions)
	if updated.Items[0].Disposition != discovery.DispositionAdopt {
		t.Errorf("expected ADOPT preserved, got %s", updated.Items[0].Disposition)
	}
}
