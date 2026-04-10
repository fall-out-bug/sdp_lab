package extract

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/architect"
)

// GitHistoryExtractor analyzes git history for co-change coupling,
// ownership patterns, hotspots, and CODEOWNERS.
type GitHistoryExtractor struct {
	MaxYears   int // default: 2
	MaxCommits int // sampling threshold, default: 50000
}

// Name implements architect.Extractor.
func (GitHistoryExtractor) Name() string { return "git_history" }

// Extract implements architect.Extractor.
func (e GitHistoryExtractor) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if .git directory exists
	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Not a git repo, return empty fragment
		return &architect.ProfileFragment{}, nil
	}

	maxYears := e.MaxYears
	if maxYears == 0 {
		maxYears = 2
	}

	maxCommits := e.MaxCommits
	if maxCommits == 0 {
		maxCommits = 50000
	}

	// Calculate period
	since := time.Now().AddDate(-maxYears, 0, 0).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")
	period := fmt.Sprintf("%s to %s", since, to)

	// Run git analyses
	commitCount := e.countCommits(ctx, repoRoot, maxYears)
	if commitCount < 0 {
		// Check if this is an empty repo (no commits) by checking if HEAD exists
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
		cmd.Dir = repoRoot
		err := cmd.Run()
		if err != nil {
			// Empty repo - return valid GitAnalysis with 0 commits
			return &architect.ProfileFragment{
				GitAnalysis: &architect.GitAnalysis{
					AnalyzedCommits: 0,
					AnalyzedPeriod:  period,
				},
			}, nil
		}
		// Git failed for another reason, return empty fragment
		return &architect.ProfileFragment{}, nil
	}

	topContributors := e.getTopContributors(ctx, repoRoot, maxYears)
	hotspots := e.detectHotspots(ctx, repoRoot, maxYears)
	coChangeClusters := e.detectCoChanges(ctx, repoRoot, maxYears)
	ownership := e.detectOwnership(ctx, repoRoot, maxYears)

	// Parse CODEOWNERS
	codeowners := e.parseCODEOWNERS(repoRoot)
	if codeowners != nil {
		if ownership == nil {
			ownership = make(map[string][]string)
		}
		for pattern, owners := range codeowners {
			ownership["@codeowners:"+pattern] = owners
		}
	}

	return &architect.ProfileFragment{
		GitAnalysis: &architect.GitAnalysis{
			AnalyzedCommits:  commitCount,
			AnalyzedPeriod:   period,
			TopContributors:  topContributors,
			Hotspots:         hotspots,
			CoChangeClusters: coChangeClusters,
			Ownership:        ownership,
		},
	}, nil
}

// countCommits returns the number of commits in the given period.
// Returns -1 if git command fails.
func (e GitHistoryExtractor) countCommits(ctx context.Context, repoRoot string, years int) int {
	since := fmt.Sprintf("%d years ago", years)
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "--since="+since)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// getTopContributors returns the top contributors by commit count.
func (e GitHistoryExtractor) getTopContributors(ctx context.Context, repoRoot string, years int) []string {
	since := fmt.Sprintf("%d years ago", years)
	cmd := exec.CommandContext(ctx, "git", "shortlog", "-sn", "--all", "--since="+since)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var contributors []string
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Format: "count name"
			name := strings.Join(parts[1:], " ")
			contributors = append(contributors, name)
		}
	}

	// Limit to top 20
	if len(contributors) > 20 {
		contributors = contributors[:20]
	}

	return contributors
}

// detectHotspots identifies files with high churn and multiple contributors.
func (e GitHistoryExtractor) detectHotspots(ctx context.Context, repoRoot string, years int) []architect.Hotspot {
	since := fmt.Sprintf("%d years ago", years)
	// Get file change history with authors
	cmd := exec.CommandContext(ctx, "git", "log", "--format=%an", "--name-only", "--diff-filter=AMDR", "--since="+since)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Track file changes and authors
	fileChanges := make(map[string]int)    // file -> change count
	fileAuthors := make(map[string]map[string]bool) // file -> authors set

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentAuthor string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// Empty line - skip but don't reset author (files follow author)
			continue
		}

		// Check if line looks like a file path (has extension or path separator)
		isFilePath := strings.Contains(line, ".") || strings.Contains(line, "/") || strings.Contains(line, "\\")

		if isFilePath {
			// This is a file path
			if currentAuthor != "" {
				fileChanges[line]++
				if fileAuthors[line] == nil {
					fileAuthors[line] = make(map[string]bool)
				}
				fileAuthors[line][currentAuthor] = true
			}
		} else {
			// This is an author name
			currentAuthor = line
		}
	}

	// Convert to hotspots
	var hotspots []architect.Hotspot
	for file, changes := range fileChanges {
		// Filter out very short-lived files (< 2 changes for testing)
		if changes < 2 {
			continue
		}
		authors := len(fileAuthors[file])
		hotspots = append(hotspots, architect.Hotspot{
			Path:    file,
			Changes: changes,
			Authors: authors,
		})
	}

	// Sort by changes descending
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Changes > hotspots[j].Changes
	})

	// Limit to top 20
	if len(hotspots) > 20 {
		hotspots = hotspots[:20]
	}

	return hotspots
}

// detectCoChanges identifies files that frequently change together.
func (e GitHistoryExtractor) detectCoChanges(ctx context.Context, repoRoot string, years int) []architect.CoChangeCluster {
	since := fmt.Sprintf("%d years ago", years)
	cmd := exec.CommandContext(ctx, "git", "log", "--format=%H", "--name-only", "--since="+since)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Track file changes and co-occurrences
	fileChanges := make(map[string]int)              // file -> change count
	coOccurrences := make(map[string]map[string]int) // file -> file -> count

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var filesInCommit []string
	var inCommit bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// End of commit, process co-occurrences
			if len(filesInCommit) > 1 {
				for i, f1 := range filesInCommit {
					fileChanges[f1]++
					for j := i + 1; j < len(filesInCommit); j++ {
						f2 := filesInCommit[j]
						if coOccurrences[f1] == nil {
							coOccurrences[f1] = make(map[string]int)
						}
						if coOccurrences[f2] == nil {
							coOccurrences[f2] = make(map[string]int)
						}
						coOccurrences[f1][f2]++
						coOccurrences[f2][f1]++
					}
				}
			}
			filesInCommit = nil
			inCommit = false
			continue
		}

		// Check if this looks like a commit hash (40 hex chars)
		if len(line) == 40 {
			isHash := true
			for _, c := range line {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					isHash = false
					break
				}
			}
			if isHash {
				filesInCommit = nil
				inCommit = true
				continue
			}
		}

		// Only collect files after we've seen a commit hash
		if inCommit {
			filesInCommit = append(filesInCommit, line)
		}
	}

	// Compute co-change ratios and find strong pairs
	type pair struct {
		f1, f2 string
		ratio  float64
	}
	var strongPairs []pair

	for f1, coMap := range coOccurrences {
		for f2, count := range coMap {
			// Jaccard-like ratio: co_changes / min(changes_A, changes_B)
			minChanges := fileChanges[f1]
			if fileChanges[f2] < minChanges {
				minChanges = fileChanges[f2]
			}
			if minChanges == 0 {
				continue
			}

			ratio := float64(count) / float64(minChanges)
			if ratio > 0.3 {
				strongPairs = append(strongPairs, pair{f1, f2, ratio})
			}
		}
	}

	// Group into clusters using transitive closure
	visited := make(map[string]bool)
	var clusters []architect.CoChangeCluster

	for _, p := range strongPairs {
		if visited[p.f1] && visited[p.f2] {
			continue
		}

		// Start a new cluster
		cluster := make(map[string]bool)
		var queue []string

		if !visited[p.f1] {
			queue = append(queue, p.f1)
		}
		if !visited[p.f2] {
			queue = append(queue, p.f2)
		}

		for len(queue) > 0 {
			file := queue[0]
			queue = queue[1:]

			if visited[file] {
				continue
			}
			visited[file] = true
			cluster[file] = true

			// Find all files that co-change with this file
			for coFile, coCount := range coOccurrences[file] {
				// Check if ratio meets threshold
				minChanges := fileChanges[file]
				if fileChanges[coFile] < minChanges {
					minChanges = fileChanges[coFile]
				}
				if minChanges > 0 {
					ratio := float64(coCount) / float64(minChanges)
					if ratio > 0.3 && !visited[coFile] {
						queue = append(queue, coFile)
					}
				}
			}
		}

		if len(cluster) > 1 {
			files := make([]string, 0, len(cluster))
			for f := range cluster {
				files = append(files, f)
			}
			sort.Strings(files)

			// Calculate cluster ratio (average of all pairs)
			var totalRatio float64
			var pairCount int
			for i, f1 := range files {
				for j := i + 1; j < len(files); j++ {
					f2 := files[j]
					if coOccurrences[f1][f2] > 0 {
						minChanges := fileChanges[f1]
						if fileChanges[f2] < minChanges {
							minChanges = fileChanges[f2]
						}
						if minChanges > 0 {
							totalRatio += float64(coOccurrences[f1][f2]) / float64(minChanges)
							pairCount++
						}
					}
				}
			}

			var avgRatio float64
			if pairCount > 0 {
				avgRatio = totalRatio / float64(pairCount)
			}

			clusters = append(clusters, architect.CoChangeCluster{
				Files:         files,
				CoChangeRatio: avgRatio,
				Signal:        generateClusterSignal(files),
			})
		}
	}

	// Sort by cluster size (descending) then by ratio (descending)
	sort.Slice(clusters, func(i, j int) bool {
		if len(clusters[i].Files) != len(clusters[j].Files) {
			return len(clusters[i].Files) > len(clusters[j].Files)
		}
		return clusters[i].CoChangeRatio > clusters[j].CoChangeRatio
	})

	// Limit to top 10
	if len(clusters) > 10 {
		clusters = clusters[:10]
	}

	return clusters
}

// generateClusterSignal creates a human-readable description of the cluster.
func generateClusterSignal(files []string) string {
	if len(files) == 2 {
		return fmt.Sprintf("%s and %s frequently change together", filepath.Base(files[0]), filepath.Base(files[1]))
	}
	return fmt.Sprintf("%d files frequently change together", len(files))
}

// detectOwnership maps directories to their contributors.
func (e GitHistoryExtractor) detectOwnership(ctx context.Context, repoRoot string, years int) map[string][]string {
	since := fmt.Sprintf("%d years ago", years)
	cmd := exec.CommandContext(ctx, "git", "log", "--format=%an", "--name-only", "--since="+since)
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Track authors per directory
	dirAuthors := make(map[string]map[string]bool) // directory -> authors set

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentAuthor string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			// Empty line - skip but don't reset author (files follow author)
			continue
		}

		// Check if line looks like a file path (has extension or path separator)
		isFilePath := strings.Contains(line, ".") || strings.Contains(line, "/") || strings.Contains(line, "\\")

		if isFilePath {
			// This is a file path
			if currentAuthor != "" {
				// Get top-level directory
				parts := strings.Split(line, string(filepath.Separator))
				if len(parts) > 0 {
					dir := parts[0]
					if dirAuthors[dir] == nil {
						dirAuthors[dir] = make(map[string]bool)
					}
					dirAuthors[dir][currentAuthor] = true
				}
			}
		} else {
			// This is an author name
			currentAuthor = line
		}
	}

	// Convert to result format
	ownership := make(map[string][]string)
	for dir, authors := range dirAuthors {
		var authorList []string
		for author := range authors {
			authorList = append(authorList, author)
		}
		sort.Strings(authorList)
		ownership[dir+"/"] = authorList
	}

	return ownership
}

// parseCODEOWNERS parses CODEOWNERS file if it exists.
func (e GitHistoryExtractor) parseCODEOWNERS(repoRoot string) map[string][]string {
	// Try both locations
	locations := []string{
		filepath.Join(repoRoot, ".github", "CODEOWNERS"),
		filepath.Join(repoRoot, "CODEOWNERS"),
		filepath.Join(repoRoot, "docs", "CODEOWNERS"),
	}

	var content []byte
	var path string
	for _, loc := range locations {
		data, err := os.ReadFile(loc)
		if err == nil {
			content = data
			path = loc
			break
		}
	}

	if len(content) == 0 {
		return nil
	}

	// Parse CODEOWNERS format
	// Format: pattern @owner1 @owner2
	owners := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pattern := parts[0]
		var ownerList []string
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "@") {
				ownerList = append(ownerList, part)
			}
		}

		if len(ownerList) > 0 {
			owners[path+":"+pattern] = ownerList
		}
	}

	return owners
}
