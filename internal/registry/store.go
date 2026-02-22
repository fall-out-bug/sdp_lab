package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Store provides CRUD for projects backed by project-registry.yaml.
type Store struct {
	path    string
	mu      sync.RWMutex
	projects map[string]*Project
}

// StoreConfig holds options for the store.
type StoreConfig struct {
	RegistryPath string // path to project-registry.yaml
}

// NewStore creates a Store. If path is empty, uses specs/project-registry.yaml in cwd.
func NewStore(cfg StoreConfig) *Store {
	path := cfg.RegistryPath
	if path == "" {
		path = "specs/project-registry.yaml"
	}
	return &Store{
		path:     path,
		projects: make(map[string]*Project),
	}
}

// Load reads projects from the YAML file.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.projects = make(map[string]*Project)
			return nil
		}
		return fmt.Errorf("read registry: %w", err)
	}

	var doc struct {
		Projects []Project `yaml:"projects"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	s.projects = make(map[string]*Project)
	for i := range doc.Projects {
		p := &doc.Projects[i]
		p.EnsureBeadsPrefix()
		s.projects[p.ID] = p
	}
	return nil
}

// Save writes projects to the YAML file.
func (s *Store) Save() error {
	s.mu.RLock()
	projects := make([]*Project, 0, len(s.projects))
	for _, p := range s.projects {
		projects = append(projects, p)
	}
	s.mu.RUnlock()

	doc := struct {
		Projects []Project `yaml:"projects"`
	}{}
	for _, p := range projects {
		doc.Projects = append(doc.Projects, *p)
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}
	return nil
}

// FindByIssueID returns the project whose beads_prefix matches the issue ID prefix.
// e.g. "sdp_dev-5l9.2" -> project with beads_prefix "sdp_dev".
func (s *Store) FindByIssueID(issueID string) (*Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx := strings.Index(issueID, "-")
	if idx <= 0 {
		return nil, false
	}
	prefix := issueID[:idx]
	p, ok := s.projects[prefix]
	if ok && p != nil {
		cp := *p
		return &cp, true
	}
	for _, proj := range s.projects {
		if proj != nil && proj.BeadsPrefix == prefix {
			cp := *proj
			return &cp, true
		}
	}
	return nil, false
}

// Get returns a project by ID.
func (s *Store) Get(id string) (*Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok || p == nil {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// List returns all projects.
func (s *Store) List() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Project, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, *p)
	}
	return out
}

// Create adds a project. Returns error if ID exists.
func (s *Store) Create(p *Project) error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	p.EnsureBeadsPrefix()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[p.ID]; exists {
		return fmt.Errorf("project %q already exists", p.ID)
	}
	cp := *p
	s.projects[p.ID] = &cp
	return nil
}

// Update replaces a project. Returns error if ID does not exist.
func (s *Store) Update(p *Project) error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	p.EnsureBeadsPrefix()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[p.ID]; !exists {
		return fmt.Errorf("project %q not found", p.ID)
	}
	cp := *p
	s.projects[p.ID] = &cp
	return nil
}

// Delete removes a project.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[id]; !exists {
		return fmt.Errorf("project %q not found", id)
	}
	delete(s.projects, id)
	return nil
}
