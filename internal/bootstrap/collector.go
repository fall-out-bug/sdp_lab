package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
)

const sdpDir = ".sdp"

// Collector reads .sdp/ analysis artifacts and repo metadata to assemble
// a DataSourceInfo for the bootstrap planner.
type Collector struct {
	// RepoPath is the absolute path to the target repository.
	RepoPath string
}

// NewCollector creates a Collector for the given repo path.
func NewCollector(repoPath string) *Collector {
	return &Collector{RepoPath: repoPath}
}

// Collect reads all available .sdp/ analysis data and returns a DataSourceInfo.
// It degrades gracefully: only scout.json is required; everything else is optional.
func (c *Collector) Collect() (*DataSourceInfo, error) {
	ds := &DataSourceInfo{}
	sdpPath := filepath.Join(c.RepoPath, sdpDir)

	// Scout is required — fail if missing.
	scoutData, err := c.readScout(sdpPath)
	if err != nil {
		return nil, fmt.Errorf("collector: scout.json is required: %w", err)
	}
	ds.Scout = scoutData

	// Optional sources — degrade gracefully.
	ds.Architect, _ = c.readArchitect(sdpPath)
	ds.Metrics, _ = c.readMetrics(sdpPath)
	ds.Spec, _ = c.readSpec(sdpPath)
	ds.Index, _ = c.readIndex(sdpPath)

	return ds, nil
}

// CollectOptional reads all available .sdp/ data but does not fail when
// scout.json is absent. Returns an empty DataSourceInfo instead.
func (c *Collector) CollectOptional() *DataSourceInfo {
	ds := &DataSourceInfo{}
	sdpPath := filepath.Join(c.RepoPath, sdpDir)

	ds.Scout, _ = c.readScout(sdpPath)
	ds.Architect, _ = c.readArchitect(sdpPath)
	ds.Metrics, _ = c.readMetrics(sdpPath)
	ds.Spec, _ = c.readSpec(sdpPath)
	ds.Index, _ = c.readIndex(sdpPath)

	return ds
}

// DataSourcesAvailable returns a map indicating which data sources were found.
func (c *Collector) DataSourcesAvailable() map[string]bool {
	sdpPath := filepath.Join(c.RepoPath, sdpDir)
	return map[string]bool{
		"scout":     fileExists(filepath.Join(sdpPath, "scout.json")),
		"architect": fileExists(filepath.Join(sdpPath, "architect", "report.json")),
		"metrics":   fileExists(filepath.Join(sdpPath, "metrics", "report.json")),
		"spec":      dirHasFiles(filepath.Join(sdpPath, "specs")),
		"index":     fileExists(filepath.Join(sdpPath, "index.db")),
	}
}

// ExistingConfig detects which SDP configuration files already exist in the repo.
func (c *Collector) ExistingConfig() []string {
	var existing []string
	checks := []struct {
		path string
		name string
	}{
		{"CLAUDE.md", "claude_md"},
		{"AGENTS.md", "agents_md"},
		{".claude/settings.json", "claude_settings"},
		{".claude/hooks", "hooks_dir"},
		{".beads", "beads_dir"},
		{".sdp", "sdp_dir"},
	}
	for _, ck := range checks {
		if pathExists(filepath.Join(c.RepoPath, ck.path)) {
			existing = append(existing, ck.name)
		}
	}
	return existing
}

// readScout reads and parses .sdp/scout.json.
func (c *Collector) readScout(sdpPath string) (*ScoutData, error) {
	data, err := os.ReadFile(filepath.Join(sdpPath, "scout.json"))
	if err != nil {
		return nil, err
	}
	return ParseScoutData(data)
}

// readArchitect reads and parses .sdp/architect/report.json.
func (c *Collector) readArchitect(sdpPath string) (*ArchitectData, error) {
	p := filepath.Join(sdpPath, "architect", "report.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return ParseArchitectData(data)
}

// readMetrics reads and parses .sdp/metrics/report.json.
func (c *Collector) readMetrics(sdpPath string) (*MetricsData, error) {
	p := filepath.Join(sdpPath, "metrics", "report.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return ParseMetricsData(data)
}

// readSpec reads specification files from .sdp/specs/.
func (c *Collector) readSpec(sdpPath string) (*SpecData, error) {
	specDir := filepath.Join(sdpPath, "specs")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, err
	}
	sd := &SpecData{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		sd.Files = append(sd.Files, SpecFile{
			Name:        e.Name(),
			Description: describeSpecFile(e.Name()),
		})
	}
	if len(sd.Files) == 0 {
		return nil, fmt.Errorf("no spec files found")
	}
	return sd, nil
}

// readIndex reads index metadata from .sdp/index.db.
func (c *Collector) readIndex(sdpPath string) (*IndexData, error) {
	p := filepath.Join(sdpPath, "index.db")
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("index.db is empty")
	}
	// Read minimal metadata — we don't need the full SQLite contents.
	// For bootstrap planning, just knowing it exists and has data is enough.
	return &IndexData{
		Files:   -1, // unknown without parsing SQLite
		Symbols: -1,
	}, nil
}

// fileExists reports whether the path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// pathExists reports whether the path exists (file or directory).
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirHasFiles reports whether the directory exists and contains at least one file.
func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// describeSpecFile returns a short description for a spec file name.
func describeSpecFile(name string) string {
	ext := filepath.Ext(name)
	switch ext {
	case ".md":
		return "Markdown specification"
	case ".json":
		return "JSON specification"
	case ".yaml", ".yml":
		return "YAML specification"
	default:
		return "Specification file"
	}
}
