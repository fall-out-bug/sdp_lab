package discovery_test

import (
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestCheckpointRender_TwoSections(t *testing.T) {
	items := []discovery.ScanItem{
		{Name: "SettledTool", Disposition: discovery.DispositionInspire,
			CoverageScore: 0.8, PrimarySourceRead: true, ArchitectureReviewed: true,
			DescSentences: 12, MultiSource: true},
		{Name: "FlaggedTool", Disposition: discovery.DispositionExtract,
			CoverageScore: 0.18, PrimarySourceRead: false, Stars: 50000, DescSentences: 3,
			DepthFlag: &discovery.DepthFlag{Flagged: true, Reason: "no_primary_source",
				RecommendedAction: "deep_dive", Blocking: true}},
	}
	result := &discovery.ScanResult{Items: items, Whitespace: "nobody covers full pipeline"}
	out := discovery.RenderCheckpoint(result)

	if !strings.Contains(out, "Section A") {
		t.Error("missing Section A")
	}
	if !strings.Contains(out, "Section B") {
		t.Error("missing Section B")
	}
	if !strings.Contains(out, "FlaggedTool") {
		t.Error("FlaggedTool not in Section B")
	}
	if !strings.Contains(out, "[D]") {
		t.Error("missing deep-dive option")
	}
}
