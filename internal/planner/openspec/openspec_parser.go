package openspec

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/planner"
)

// OpenSpecChange represents a parsed OpenSpec change folder
type OpenSpecChange struct {
	SourcePath string
	SourceHash string
	Proposal   *OpenSpecProposal
	SpecDelta  *OpenSpecSpecDelta
	Design     *OpenSpecDesign
	Tasks      []*OpenSpecTask
	Unresolved []*UnresolvedMapping
	ParsedAt   time.Time
}

// OpenSpecProposal represents the proposal.md file
type OpenSpecProposal struct {
	Title       string
	Description string
	Priority    string
	Labels      []string
	Source      string
	RawContent  string
}

// OpenSpecSpecDelta represents spec changes (ADDED/MODIFIED/REMOVED)
type OpenSpecSpecDelta struct {
	Changes []*SpecChange
}

type SpecChange struct {
	Type     string // ADDED, MODIFIED, REMOVED
	Path     string
	Content  string
	Original string
}

// OpenSpecDesign represents design artifacts
type OpenSpecDesign struct {
	Architecture string
	Components   []string
	Decisions    []*DesignDecision
	RawContent   string
}

type DesignDecision struct {
	ID      string
	Title   string
	Context string
	Outcome string
}

// OpenSpecTask represents a task from tasks.md or individual task files
type OpenSpecTask struct {
	ID           string
	Title        string
	Description  string
	Status       string
	Priority     string
	Dependencies []string
	Estimate     string
	Source       string
}

// UnresolvedMapping represents ambiguous or unsupported mappings
type UnresolvedMapping struct {
	Source     string
	Reason     string
	Context    string
	Confidence float64
	Suggestion string
}

// ImportResult represents the normalized SDP planning payload
type ImportResult struct {
	ImportID     string
	SourcePath   string
	SourceHash   string
	Plan         *planner.Plan
	Unresolved   []*UnresolvedMapping
	MappingStats MappingStats
	ImportedAt   time.Time
}

type MappingStats struct {
	TotalTasks      int
	MappedTasks     int
	UnresolvedTasks int
	TotalDeps       int
	MappedDeps      int
	Confidence      float64
}

// ParserConfig holds configuration for parsing
type ParserConfig struct {
	StrictMode    bool
	MinConfidence float64
	IgnoreFiles   []string
}

// DefaultParserConfig returns default parser configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		StrictMode:    false,
		MinConfidence: 0.5,
		IgnoreFiles:   []string{".DS_Store", ".git"},
	}
}

// OpenSpecParser handles parsing of OpenSpec change folders
type OpenSpecParser struct {
	config *ParserConfig
}

// NewOpenSpecParser creates a new parser instance
func NewOpenSpecParser(config *ParserConfig) *OpenSpecParser {
	if config == nil {
		config = DefaultParserConfig()
	}
	return &OpenSpecParser{
		config: config,
	}
}

// ParseChange parses an OpenSpec change folder and returns normalized SDP planning payload
func (p *OpenSpecParser) ParseChange(changePath string) (*ImportResult, error) {
	// Validate path exists
	if _, err := os.Stat(changePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("change path does not exist: %s", changePath)
	}

	// Calculate source hash
	sourceHash, err := p.calculateSourceHash(changePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate source hash: %w", err)
	}

	// Parse OpenSpec artifacts
	change := &OpenSpecChange{
		SourcePath: changePath,
		SourceHash: sourceHash,
		ParsedAt:   time.Now(),
	}

	// Parse proposal (proposal.md)
	if err := p.parseProposal(change); err != nil {
		return nil, fmt.Errorf("failed to parse proposal: %w", err)
	}

	// Parse spec delta (spec.md)
	if err := p.parseSpecDelta(change); err != nil && p.config.StrictMode {
		return nil, fmt.Errorf("failed to parse spec delta: %w", err)
	}

	// Parse design (design.md or design folder)
	if err := p.parseDesign(change); err != nil && p.config.StrictMode {
		return nil, fmt.Errorf("failed to parse design: %w", err)
	}

	// Parse tasks (tasks.md or tasks folder)
	if err := p.parseTasks(change); err != nil {
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	// Convert to SDP plan
	result, err := p.convertToSDPPlan(change)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to SDP plan: %w", err)
	}

	return result, nil
}

// parseProposal parses the proposal.md file
func (p *OpenSpecParser) parseProposal(change *OpenSpecChange) error {
	proposalPath := filepath.Join(change.SourcePath, "proposal.md")
	content, err := os.ReadFile(proposalPath)
	if err != nil {
		return fmt.Errorf("proposal.md not found: %w", err)
	}

	proposal := &OpenSpecProposal{
		RawContent: string(content),
	}

	// Parse frontmatter and content
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			proposal.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		} else if strings.HasPrefix(line, "Priority: ") {
			proposal.Priority = strings.TrimSpace(strings.TrimPrefix(line, "Priority: "))
		}
	}

	// If no title found, use first heading or filename
	if proposal.Title == "" {
		proposal.Title = "Imported from OpenSpec proposal"
	}

	change.Proposal = proposal
	return nil
}

// parseSpecDelta parses spec changes with ADDED/MODIFIED/REMOVED markers
func (p *OpenSpecParser) parseSpecDelta(change *OpenSpecChange) error {
	specPath := filepath.Join(change.SourcePath, "spec.md")
	content, err := os.ReadFile(specPath)
	if err != nil {
		// spec.md is optional
		change.Unresolved = append(change.Unresolved, &UnresolvedMapping{
			Source:     "spec.md",
			Reason:     "file not found (optional)",
			Confidence: 0.0,
		})
		return nil
	}

	specDelta := &OpenSpecSpecDelta{
		Changes: []*SpecChange{},
	}

	// Parse ADDED/MODIFIED/REMOVED sections
	lines := strings.Split(string(content), "\n")
	var currentType string
	var currentPath string
	var currentContent strings.Builder

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "## ADDED"):
			currentType = "ADDED"
		case strings.HasPrefix(line, "## MODIFIED"):
			currentType = "MODIFIED"
		case strings.HasPrefix(line, "## REMOVED"):
			currentType = "REMOVED"
		case strings.HasPrefix(line, "### "):
			// New path entry
			if currentPath != "" {
				specDelta.Changes = append(specDelta.Changes, &SpecChange{
					Type:    currentType,
					Path:    currentPath,
					Content: currentContent.String(),
				})
				currentContent.Reset()
			}
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "### "))
		default:
			if currentPath != "" {
				currentContent.WriteString(line + "\n")
			}
		}
	}

	// Add last entry
	if currentPath != "" {
		specDelta.Changes = append(specDelta.Changes, &SpecChange{
			Type:    currentType,
			Path:    currentPath,
			Content: currentContent.String(),
		})
	}

	change.SpecDelta = specDelta
	return nil
}

// parseDesign parses design artifacts
func (p *OpenSpecParser) parseDesign(change *OpenSpecChange) error {
	designPath := filepath.Join(change.SourcePath, "design.md")
	content, err := os.ReadFile(designPath)
	if err != nil {
		// design.md is optional
		return nil
	}

	design := &OpenSpecDesign{
		RawContent: string(content),
		Decisions:  []*DesignDecision{},
	}

	// Parse architecture and decisions
	lines := strings.Split(string(content), "\n")
	var inArchitecture bool
	var inDecision bool
	var currentDecision *DesignDecision

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# Architecture"):
			inArchitecture = true
		case strings.HasPrefix(line, "# Decisions"):
			inArchitecture = false
		case strings.HasPrefix(line, "## ") && inArchitecture:
			design.Architecture += strings.TrimPrefix(line, "## ") + "\n"
		case strings.HasPrefix(line, "## ") && strings.Contains(line, "Decision"):
			inDecision = true
			if currentDecision != nil {
				design.Decisions = append(design.Decisions, currentDecision)
			}
			currentDecision = &DesignDecision{
				Title: strings.TrimSpace(strings.TrimPrefix(line, "## ")),
			}
		case inDecision && currentDecision != nil:
			if strings.HasPrefix(line, "### Context") {
				currentDecision.Context = strings.TrimPrefix(line, "### Context")
			} else if strings.HasPrefix(line, "### Outcome") {
				currentDecision.Outcome = strings.TrimPrefix(line, "### Outcome")
			}
		}
	}

	if currentDecision != nil {
		design.Decisions = append(design.Decisions, currentDecision)
	}

	change.Design = design
	return nil
}

// parseTasks parses tasks from tasks.md or tasks folder
func (p *OpenSpecParser) parseTasks(change *OpenSpecChange) error {
	tasksPath := filepath.Join(change.SourcePath, "tasks.md")
	content, err := os.ReadFile(tasksPath)

	if err != nil {
		// Try tasks folder as fallback
		tasksFolderPath := filepath.Join(change.SourcePath, "tasks")
		if entries, err := os.ReadDir(tasksFolderPath); err == nil {
			return p.parseTasksFromFolder(change, tasksFolderPath, entries)
		}
		return fmt.Errorf("no tasks.md or tasks/ folder found")
	}

	// Parse tasks from single file
	return p.parseTasksFromFile(change, string(content))
}

// parseTasksFromFile parses tasks from a single tasks.md file
func (p *OpenSpecParser) parseTasksFromFile(change *OpenSpecChange, content string) error {
	lines := strings.Split(content, "\n")
	var currentTask *OpenSpecTask
	taskID := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "## "):
			// New task
			if currentTask != nil {
				change.Tasks = append(change.Tasks, currentTask)
			}
			taskID++
			currentTask = &OpenSpecTask{
				ID:     fmt.Sprintf("openspec-task-%d", taskID),
				Title:  strings.TrimSpace(strings.TrimPrefix(line, "## ")),
				Source: "tasks.md",
			}
		case currentTask != nil && strings.HasPrefix(line, "- [x]"):
			currentTask.Status = "completed"
		case currentTask != nil && strings.HasPrefix(line, "- [ ]"):
			currentTask.Status = "pending"
		case currentTask != nil && strings.HasPrefix(line, "**Depends on:**"):
			deps := strings.Fields(strings.TrimPrefix(line, "**Depends on:**"))
			currentTask.Dependencies = deps
		case currentTask != nil && strings.HasPrefix(line, "**Priority:**"):
			currentTask.Priority = strings.TrimSpace(strings.TrimPrefix(line, "**Priority:**"))
		case currentTask != nil && line != "":
			currentTask.Description += line + "\n"
		}
	}

	if currentTask != nil {
		change.Tasks = append(change.Tasks, currentTask)
	}

	return nil
}

// parseTasksFromFolder parses tasks from individual files in tasks folder
func (p *OpenSpecParser) parseTasksFromFolder(change *OpenSpecChange, folderPath string, entries []os.DirEntry) error {
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		taskPath := filepath.Join(folderPath, entry.Name())
		content, err := os.ReadFile(taskPath)
		if err != nil {
			continue
		}

		task := &OpenSpecTask{
			ID:     fmt.Sprintf("openspec-task-%s", strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))),
			Title:  strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			Source: entry.Name(),
			Status: "pending",
		}

		// Parse task content
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				task.Title = strings.TrimPrefix(line, "# ")
			} else if line != "" {
				task.Description += line + "\n"
			}
		}

		change.Tasks = append(change.Tasks, task)
	}

	return nil
}

// convertToSDPPlan converts OpenSpec change to SDP plan
func (p *OpenSpecParser) convertToSDPPlan(change *OpenSpecChange) (*ImportResult, error) {
	// Generate deterministic import ID from source hash
	hashBytes := sha256.Sum256([]byte(change.SourceHash))
	importID := fmt.Sprintf("openspec-import-%x", hashBytes[:8])

	// Create plan
	plan := planner.NewPlan(
		importID,
		change.Proposal.Title,
		change.Proposal.Description,
	)

	// Create phases based on OpenSpec structure
	phaseID := fmt.Sprintf("phase-%s", importID)
	phase := &planner.Phase{
		ID:          phaseID,
		Name:        "OpenSpec Import",
		Description: fmt.Sprintf("Imported from %s", change.SourcePath),
		Status:      planner.TaskStatusPending,
	}

	if err := plan.AddPhase(phase); err != nil {
		return nil, fmt.Errorf("failed to add phase: %w", err)
	}

	// Convert tasks
	stats := MappingStats{}
	taskIDMap := make(map[string]string)

	for _, task := range change.Tasks {
		sdpTask, err := p.convertTask(task, change)
		if err != nil {
			change.Unresolved = append(change.Unresolved, &UnresolvedMapping{
				Source:     task.ID,
				Reason:     fmt.Sprintf("failed to convert task: %v", err),
				Confidence: 0.0,
			})
			stats.UnresolvedTasks++
			continue
		}

		if err := plan.AddTask(sdpTask); err != nil {
			change.Unresolved = append(change.Unresolved, &UnresolvedMapping{
				Source:     task.ID,
				Reason:     fmt.Sprintf("failed to add task: %v", err),
				Confidence: 0.0,
			})
			stats.UnresolvedTasks++
			continue
		}

		phase.Tasks = append(phase.Tasks, sdpTask.ID)
		taskIDMap[task.ID] = string(sdpTask.ID)
		stats.MappedTasks++
	}

	stats.TotalTasks = len(change.Tasks)

	// Map dependencies
	for _, task := range change.Tasks {
		if sdpTaskID, ok := taskIDMap[task.ID]; ok {
			sdpTask, _ := plan.GetTask(planner.TaskID(sdpTaskID))
			for _, depID := range task.Dependencies {
				if mappedDepID, ok := taskIDMap[depID]; ok {
					sdpTask.Dependencies = append(sdpTask.Dependencies, planner.TaskID(mappedDepID))
					stats.MappedDeps++
				} else {
					change.Unresolved = append(change.Unresolved, &UnresolvedMapping{
						Source:     task.ID,
						Reason:     fmt.Sprintf("dependency not found: %s", depID),
						Context:    "dependency mapping",
						Confidence: 0.0,
					})
				}
				stats.TotalDeps++
			}
		}
	}

	// Calculate confidence
	if stats.TotalTasks > 0 {
		stats.Confidence = float64(stats.MappedTasks) / float64(stats.TotalTasks)
	}

	// Validate plan
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	return &ImportResult{
		ImportID:     importID,
		SourcePath:   change.SourcePath,
		SourceHash:   change.SourceHash,
		Plan:         plan,
		Unresolved:   change.Unresolved,
		MappingStats: stats,
		ImportedAt:   time.Now(),
	}, nil
}

// convertTask converts OpenSpec task to SDP task
func (p *OpenSpecParser) convertTask(task *OpenSpecTask, change *OpenSpecChange) (*planner.Task, error) {
	status := planner.TaskStatusPending
	switch task.Status {
	case "completed":
		status = planner.TaskStatusCompleted
	case "in_progress", "in-progress":
		status = planner.TaskStatusInProgress
	case "blocked":
		status = planner.TaskStatusBlocked
	}

	sdpTask := &planner.Task{
		ID:          planner.TaskID(task.ID),
		Title:       task.Title,
		Description: task.Description,
		Status:      status,
		Priority:    p.parsePriority(task.Priority),
		Metadata: map[string]interface{}{
			"openspec_source": task.Source,
			"openspec_import": change.SourcePath,
		},
	}

	// Parse estimate
	if task.Estimate != "" {
		// Simple heuristic for estimate parsing
		if strings.Contains(task.Estimate, "h") {
			var hours float64
			if n, err := fmt.Sscanf(task.Estimate, "%f", &hours); err == nil && n == 1 {
				sdpTask.EstimatedHours = hours
			}
		}
	}

	return sdpTask, nil
}

// parsePriority converts OpenSpec priority to SDP priority
func (p *OpenSpecParser) parsePriority(priority string) int {
	switch strings.ToLower(priority) {
	case "critical", "p0":
		return 0
	case "high", "p1":
		return 1
	case "medium", "p2":
		return 2
	case "low", "p3":
		return 3
	default:
		return 2 // Default to medium
	}
}

// calculateSourceHash calculates a deterministic hash of the OpenSpec change folder
func (p *OpenSpecParser) calculateSourceHash(path string) (string, error) {
	hash := sha256.New()

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored files
		for _, ignore := range p.config.IgnoreFiles {
			if strings.Contains(path, ignore) {
				return nil
			}
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hash.Write(data)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ToJSON returns JSON representation of import result
func (r *ImportResult) ToJSON() (string, error) {
	data := map[string]interface{}{
		"import_id":   r.ImportID,
		"source_path": r.SourcePath,
		"source_hash": r.SourceHash,
		"plan": map[string]interface{}{
			"id":    r.Plan.ID(),
			"title": r.Plan.Title(),
			"goal":  r.Plan.Goal(),
		},
		"unresolved":    r.Unresolved,
		"mapping_stats": r.MappingStats,
		"imported_at":   r.ImportedAt.Format(time.RFC3339),
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// GetPlanGraph returns the plan graph for visualization
func (r *ImportResult) GetPlanGraph() *planner.PlanGraph {
	return r.Plan.ToGraph()
}

// ValidateImport checks if the import result meets quality thresholds
func (r *ImportResult) ValidateImport(minConfidence float64) error {
	if r.MappingStats.Confidence < minConfidence {
		return fmt.Errorf("import confidence %.2f below threshold %.2f", r.MappingStats.Confidence, minConfidence)
	}

	if r.MappingStats.MappedTasks == 0 {
		return fmt.Errorf("no tasks were successfully mapped")
	}

	if len(r.Unresolved) > 0 {
		// Return warning, not error, for unresolved items
		return fmt.Errorf("import has %d unresolved mappings (check Unresolved field for details)", len(r.Unresolved))
	}

	return nil
}
