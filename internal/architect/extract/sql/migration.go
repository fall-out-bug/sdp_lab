// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// detectMigrations scans well-known directories for migration files.
func detectMigrations(root string) *architect.MigrationInfo {
	for _, dir := range migrationRoots {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			names = append(names, e.Name())
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		return &architect.MigrationInfo{
			Dir:    dir,
			Count:  len(names),
			Latest: names[len(names)-1],
		}
	}
	return nil
}

// DetectMigrationFiles returns all migration files found in well-known directories.
func DetectMigrationFiles(root string) []architect.Migration {
	var migrations []architect.Migration

	for _, dir := range migrationRoots {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			migrations = append(migrations, architect.Migration{
				Path:      filepath.Join(dir, name),
				Version:   extractVersion(name),
				Direction: extractDirection(name),
			})
		}
	}

	// Sort migrations by path for deterministic output.
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Path < migrations[j].Path
	})

	return migrations
}

// extractVersion attempts to extract a version number from a migration filename.
// Supports common patterns: 20230101_name.sql, V2__name.sql, 001-name.sql, etc.
func extractVersion(filename string) string {
	// Try timestamp pattern (YYYYMMDD or YYYYMMDDHHMMSS)
	reTimestamp := regexp.MustCompile(`^(\d{8,14})`)
	if m := reTimestamp.FindStringSubmatch(filename); m != nil {
		return m[1]
	}

	// Try version prefix pattern: V2__, v1.0__, 001-, etc.
	reVersion := regexp.MustCompile(`^[vV]?(\d+(?:\.\d+)*)`)
	if m := reVersion.FindStringSubmatch(filename); m != nil {
		return m[1]
	}

	// Try numeric prefix: 001_name.sql
	reNumeric := regexp.MustCompile(`^(\d{3,})[_-]`)
	if m := reNumeric.FindStringSubmatch(filename); m != nil {
		return m[1]
	}

	return ""
}

// extractDirection determines if a migration is an "up" or "down" migration.
func extractDirection(filename string) string {
	lower := strings.ToLower(filename)

	// Check for down migration indicators
	if strings.Contains(lower, ".down.") ||
		strings.Contains(lower, "_down.") ||
		strings.Contains(lower, "-down.") ||
		strings.Contains(lower, ".rollback.") ||
		strings.Contains(lower, "_rollback.") {
		return "down"
	}

	// Check for up migration indicators
	if strings.Contains(lower, ".up.") ||
		strings.Contains(lower, "_up.") ||
		strings.Contains(lower, "-up.") ||
		strings.Contains(lower, ".forward.") ||
		strings.Contains(lower, "_forward.") {
		return "up"
	}

	// Default to "up" if no direction specified
	return "up"
}

// GetMigrationStats returns statistics about detected migrations.
func GetMigrationStats(migrations []architect.Migration) map[string]int {
	stats := map[string]int{
		"total": len(migrations),
		"up":    0,
		"down":  0,
	}

	for _, m := range migrations {
		stats[m.Direction]++
	}

	return stats
}

// SortMigrationsByVersion sorts migrations by version string.
func SortMigrationsByVersion(migrations []architect.Migration) {
	sort.Slice(migrations, func(i, j int) bool {
		// Try numeric comparison first
		vi := migrations[i].Version
		vj := migrations[j].Version

		// If both have versions, try to compare numerically
		if vi != "" && vj != "" {
			// Simple numeric comparison
			if vi != vj {
				return vi < vj
			}
		}

		// Fall back to path comparison
		return migrations[i].Path < migrations[j].Path
	})
}

// GroupMigrationsByDirection groups migrations by their direction.
func GroupMigrationsByDirection(migrations []architect.Migration) map[string][]architect.Migration {
	groups := make(map[string][]architect.Migration)

	for _, m := range migrations {
		dir := m.Direction
		if dir == "" {
			dir = "up"
		}
		groups[dir] = append(groups[dir], m)
	}

	return groups
}
