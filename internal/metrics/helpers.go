package metrics

import "strings"

// IsBot reports whether an author name matches known bot patterns.
func IsBot(author string) bool {
	bots := []string{"dependabot", "renovate", "github-actions", "mergify", "snyk", "semantic-release"}
	lower := strings.ToLower(author)
	for _, b := range bots {
		if strings.Contains(lower, b) {
			return true
		}
	}
	return false
}

// IsGeneratedFile reports whether a file path matches generated-file patterns.
func IsGeneratedFile(path string) bool {
	patterns := []string{".pb.go", ".generated.", ".min.js", ".min.css"}
	for _, p := range patterns {
		if strings.Contains(path, p) {
			return true
		}
	}
	suffixes := []string{".lock", ".sum", "-lock.json"}
	for _, s := range suffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}
	return false
}

// IsCIOnly reports whether all changed files in a commit are CI/infra config.
func IsCIOnly(files []FileChange) bool {
	if len(files) == 0 {
		return false
	}
	ciPrefixes := []string{".github/", ".gitlab-ci.yml", "Jenkinsfile", ".circleci/", ".travis.yml"}
	for _, f := range files {
		isCI := false
		for _, prefix := range ciPrefixes {
			if strings.HasPrefix(f.Path, prefix) {
				isCI = true
				break
			}
		}
		if !isCI {
			return false
		}
	}
	return true
}

// IsFormattingOnly reports whether a commit appears to be a mass reformatting.
func IsFormattingOnly(files []FileChange) bool {
	if len(files) < 3 {
		return false
	}
	formatCount := 0
	for _, f := range files {
		if f.Added == 0 && f.Deleted == 0 {
			continue
		}
		mn := min(f.Added, f.Deleted)
		mx := max(f.Added, f.Deleted)
		if mx > 0 && (mn >= mx*9/10) {
			formatCount++
		}
	}
	return formatCount*10 >= len(files)*9
}
