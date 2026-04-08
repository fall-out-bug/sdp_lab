package discovery_test

import (
	"testing"
	"sdp_dev/internal/discovery"
)

func TestDepthFlag_H3_NoVerdict_Without_PrimarySource(t *testing.T) {
	item := discovery.ScanItem{
		Name:              "DeerFlow",
		Disposition:       discovery.DispositionExtract,
		Stars:             50000,
		PrimarySourceRead: false,
		DescSentences:     3,
		SourceCount:       4,
	}
	flag := discovery.EvalDepth(item)
	if !flag.Flagged {
		t.Error("H3: EXTRACT verdict without primary source must be flagged")
	}
	if flag.Reason != "no_primary_source" {
		t.Errorf("expected reason no_primary_source, got %s", flag.Reason)
	}
}

func TestDepthFlag_H1_HighStarsLowDescription(t *testing.T) {
	item := discovery.ScanItem{
		Name:              "SomeTool",
		Disposition:       discovery.DispositionMonitor,
		Stars:             10000,
		PrimarySourceRead: true,
		DescSentences:     2, // < 5
		SourceCount:       2,
	}
	flag := discovery.EvalDepth(item)
	if !flag.Flagged {
		t.Error("H1: >5K stars + <5 desc sentences must be flagged")
	}
}

func TestDepthFlag_H2_SingleSourceThinDescription(t *testing.T) {
	item := discovery.ScanItem{
		Name:              "LesserKnownTool",
		Disposition:       discovery.DispositionInspire,
		Stars:             500,
		PrimarySourceRead: true, // primary read but only one source
		DescSentences:     5,    // < 8
		SourceCount:       1,    // single source
		MultiSource:       false,
	}
	flag := discovery.EvalDepth(item)
	if !flag.Flagged {
		t.Error("H2: primary source read but single source + thin description must be flagged")
	}
	if flag.Reason != "single_source_thin_description" {
		t.Errorf("expected reason single_source_thin_description, got %s", flag.Reason)
	}
}

func TestDepthFlag_Settled_NoFlag(t *testing.T) {
	item := discovery.ScanItem{
		Name:                 "WellResearched",
		Disposition:          discovery.DispositionInspire,
		Stars:                1000,
		PrimarySourceRead:    true,
		ArchitectureReviewed: true,
		DescSentences:        15,
		SourceCount:          3,
		MultiSource:          true,
	}
	flag := discovery.EvalDepth(item)
	if flag.Flagged {
		t.Errorf("well-researched item should not be flagged, got: %s", flag.Reason)
	}
}

func TestCoverageScore(t *testing.T) {
	item := discovery.ScanItem{
		PrimarySourceRead:    true,
		ArchitectureReviewed: true,
		DescSentences:        20,
		MultiSource:          true,
	}
	score := discovery.CoverageScore(item)
	if score < 0.9 {
		t.Errorf("fully covered item should score ≥0.9, got %.2f", score)
	}
}
