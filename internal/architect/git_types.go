package architect

// Hotspot is a file with high churn and multiple contributors.
type Hotspot struct {
	Path    string `json:"path"`
	Changes int    `json:"changes"`
	Authors int    `json:"authors"`
}

// CoChangeCluster groups files that frequently change together.
type CoChangeCluster struct {
	Files         []string `json:"files"`
	CoChangeRatio float64  `json:"co_change_ratio"` // 0.0-1.0
	Signal        string   `json:"signal,omitempty"` // human-readable insight
}

// GitAnalysis holds results from git history analysis.
type GitAnalysis struct {
	AnalyzedCommits  int                `json:"analyzed_commits"`
	AnalyzedPeriod   string             `json:"analyzed_period,omitempty"` // "2024-01-01 to 2026-04-10"
	TopContributors  []string           `json:"top_contributors,omitempty"`
	Hotspots         []Hotspot          `json:"hotspots,omitempty"`
	CoChangeClusters []CoChangeCluster  `json:"co_change_clusters,omitempty"`
	Ownership        map[string][]string `json:"ownership,omitempty"` // directory -> contributors
}
