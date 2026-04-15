package metrics

import (
	"bufio"
	"regexp"
	"strings"
	"time"
)

const gitDelim = "COMMIT_BOUNDARY_9F2A"
const gitLogFormat = "COMMIT_BOUNDARY_9F2A%H%nAUTHOR:%an%nDATE:%aI%nSUBJECT:%s%nBODY:%b%nNUMSTAT"

// parseCommits parses the raw git log output into RawCommit structs.
func parseCommits(raw string) []RawCommit {
	var commits []RawCommit
	blocks := strings.Split(raw, gitDelim)

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		c, ok := parseOneCommit(block)
		if !ok {
			continue
		}
		commits = append(commits, c)
	}
	return commits
}

func parseOneCommit(block string) (RawCommit, bool) {
	var c RawCommit
	scanner := bufio.NewScanner(strings.NewReader(block))
	inNumstat := false
	var numstatLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "NUMSTAT" {
			inNumstat = true
			continue
		}

		if inNumstat {
			if strings.HasPrefix(line, "COMMIT_BOUNDARY") || strings.HasPrefix(line, "AUTHOR:") {
				continue
			}
			numstatLines = append(numstatLines, line)
			continue
		}

		if strings.HasPrefix(line, "AUTHOR:") {
			c.Author = strings.TrimSpace(strings.TrimPrefix(line, "AUTHOR:"))
		} else if strings.HasPrefix(line, "DATE:") {
			ds := strings.TrimSpace(strings.TrimPrefix(line, "DATE:"))
			t, err := time.Parse(time.RFC3339, ds)
			if err == nil {
				c.Date = t
			}
		} else if strings.HasPrefix(line, "SUBJECT:") {
			c.Subject = strings.TrimSpace(strings.TrimPrefix(line, "SUBJECT:"))
		} else if strings.HasPrefix(line, "BODY:") {
			c.Body = strings.TrimSpace(strings.TrimPrefix(line, "BODY:"))
		} else if strings.HasPrefix(line, "COMMIT_BOUNDARY") {
			// Skip boundary markers
		} else if len(line) == 40 && isHex(line) && c.Hash == "" {
			c.Hash = line
		} else if c.Hash == "" && len(line) > 0 && len(line) <= 40 && isHex(line) {
			c.Hash = line
		}
	}

	for _, nl := range numstatLines {
		fc, ok := parseNumstatLine(nl)
		if ok {
			c.Files = append(c.Files, fc)
		}
	}

	if c.Hash == "" {
		return c, false
	}
	return c, true
}

func parseNumstatLine(line string) (FileChange, bool) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return FileChange{}, false
	}
	added := parseNumOrNeg(parts[0])
	deleted := parseNumOrNeg(parts[1])
	path := parts[2]
	if added < 0 || deleted < 0 {
		return FileChange{Path: path}, false
	}
	return FileChange{Added: added, Deleted: deleted, Path: path}, true
}

func parseNumOrNeg(s string) int {
	s = strings.TrimSpace(s)
	if s == "-" {
		return -1
	}
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return -1
		}
	}
	return n
}

var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+`)

func parseTags(raw string) []TagInfo {
	var tags []TagInfo
	for _, line := range strings.Split(raw, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		tags = append(tags, TagInfo{
			Tag:      tag,
			IsSemver: semverRe.MatchString(tag),
		})
	}
	return tags
}

// parseBranchesBatch parses output from git for-each-ref (single call).
func parseBranchesBatch(raw string) []BranchInfo {
	var branches []BranchInfo
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		idx := strings.LastIndex(line, " ")
		if idx < 0 {
			branches = append(branches, BranchInfo{Name: line})
			continue
		}
		name := line[:idx]
		dateStr := line[idx+1:]
		bi := BranchInfo{Name: name}
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			bi.LastCommit = &t
		}
		branches = append(branches, bi)
	}
	return branches
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
