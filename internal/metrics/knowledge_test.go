package metrics

import (
	"testing"
	"time"
)

func TestAnalyzeKnowledgeNil(t *testing.T) {
	if AnalyzeKnowledge(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeKnowledgeEmpty(t *testing.T) {
	if AnalyzeKnowledge(&GitData{}) != nil {
		t.Fatal("expected nil for empty commits")
	}
}

func TestAnalyzeKnowledgeBusFactor(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "core/main.go"}}},
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "core/util.go"}}},
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "core/handler.go"}}},
			{Author: "Bob", Date: now, Files: []FileChange{{Path: "core/server.go"}}},
			{Author: "Bob", Date: now, Files: []FileChange{{Path: "core/db.go"}}},
			{Author: "Charlie", Date: now, Files: []FileChange{{Path: "core/test.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	// Alice has 3, Bob has 2, Charlie has 1 = 6 total
	// Alice alone covers 3/6 = 50% → not > 50%
	// Alice + Bob covers 5/6 = 83% > 50% → bus factor = 2
	if kr.OverallBusFactor != 2 {
		t.Fatalf("expected bus factor 2 got %d", kr.OverallBusFactor)
	}
}

func TestAnalyzeKnowledgeBusFactorSingleAuthor(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "core/main.go"}}},
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "core/util.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	if kr.OverallBusFactor != 1 {
		t.Fatalf("expected bus factor 1 got %d", kr.OverallBusFactor)
	}
}

func TestAnalyzeKnowledgeGini(t *testing.T) {
	now := time.Now()
	// Equal distribution → gini = 0
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "f.go"}}},
			{Author: "Bob", Date: now, Files: []FileChange{{Path: "f.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	if kr.GiniCoefficient != 0 {
		t.Fatalf("expected gini=0 for equal distribution got %.4f", kr.GiniCoefficient)
	}
}

func TestAnalyzeKnowledgeGiniUnequal(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "f.go"}}},
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "f.go"}}},
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "f.go"}}},
			{Author: "Bob", Date: now, Files: []FileChange{{Path: "f.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	if kr.GiniCoefficient <= 0 {
		t.Fatalf("expected positive gini for unequal distribution got %.4f", kr.GiniCoefficient)
	}
}

func TestAnalyzeKnowledgeFormerContributors(t *testing.T) {
	now := time.Now()
	old := now.AddDate(0, -8, 0) // 8 months ago
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "f.go"}}},
			{Author: "Bob", Date: old, Files: []FileChange{{Path: "f.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	if len(kr.FormerContributors) != 1 {
		t.Fatalf("expected 1 former contributor got %d", len(kr.FormerContributors))
	}
	if kr.FormerContributors[0] != "Bob" {
		t.Fatalf("expected Bob as former got %s", kr.FormerContributors[0])
	}
	if kr.FormerContributorRatio != 0.5 {
		t.Fatalf("expected ratio 0.5 got %.2f", kr.FormerContributorRatio)
	}
}

func TestAnalyzeKnowledgeModuleBreakdown(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Commits: []RawCommit{
			{Author: "Alice", Date: now, Files: []FileChange{{Path: "api/server.go"}}},
			{Author: "Bob", Date: now, Files: []FileChange{{Path: "db/model.go"}}},
		},
	}
	kr := AnalyzeKnowledge(data)
	if len(kr.BusFactorByModule) < 2 {
		t.Fatalf("expected at least 2 modules got %d", len(kr.BusFactorByModule))
	}
}

func TestGiniEdgeCases(t *testing.T) {
	if gini(nil) != 0 {
		t.Fatal("expected 0 for nil")
	}
	if gini([]float64{}) != 0 {
		t.Fatal("expected 0 for empty")
	}
	if gini([]float64{0, 0}) != 0 {
		t.Fatal("expected 0 for all zeros")
	}
}

func TestBusFactorEdgeCases(t *testing.T) {
	if busFactor(nil, 0) != 0 {
		t.Fatal("expected 0 for nil")
	}
	if busFactor([]authorCommits{{"a", 5}}, 0) != 0 {
		t.Fatal("expected 0 for zero total")
	}
}
