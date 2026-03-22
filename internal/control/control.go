package control

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultControlRoot = ".sdp/control"
	specVersion        = "v1.0"
)

var (
	ErrUnknownProject = errors.New("unknown project")
)

type FeatureCard struct {
	ID                   string   `yaml:"id" json:"id"`
	ProjectID            string   `yaml:"project_id" json:"project_id"`
	Title                string   `yaml:"title" json:"title"`
	Status               string   `yaml:"status" json:"status"`
	RawRequest           string   `yaml:"raw_request" json:"raw_request"`
	CreatedAt            string   `yaml:"created_at" json:"created_at"`
	UpdatedAt            string   `yaml:"updated_at" json:"updated_at"`
	NormalizedIntent     string   `yaml:"normalized_intent,omitempty" json:"normalized_intent,omitempty"`
	TaskType             string   `yaml:"task_type,omitempty" json:"task_type,omitempty"`
	ExecutionMode        string   `yaml:"execution_mode,omitempty" json:"execution_mode,omitempty"`
	TargetRepo           string   `yaml:"target_repo,omitempty" json:"target_repo,omitempty"`
	TargetArea           string   `yaml:"target_area,omitempty" json:"target_area,omitempty"`
	ScopeIn              []string `yaml:"scope_in,omitempty" json:"scope_in,omitempty"`
	ScopeOut             []string `yaml:"scope_out,omitempty" json:"scope_out,omitempty"`
	NonGoals             []string `yaml:"non_goals,omitempty" json:"non_goals,omitempty"`
	RiskLevel            string   `yaml:"risk_level,omitempty" json:"risk_level,omitempty"`
	WhyNow               string   `yaml:"why_now,omitempty" json:"why_now,omitempty"`
	Links                []string `yaml:"links,omitempty" json:"links,omitempty"`
	OpenQuestions        []string `yaml:"open_questions,omitempty" json:"open_questions,omitempty"`
	AcceptanceShape      []string `yaml:"acceptance_shape,omitempty" json:"acceptance_shape,omitempty"`
	RecommendedNext      string   `yaml:"recommended_next_step,omitempty" json:"recommended_next_step,omitempty"`
	IntakeArtifact       []string `yaml:"intake_artifact,omitempty" json:"intake_artifact,omitempty"`
	LinkedBeadsIDs       []string `yaml:"linked_beads_ids,omitempty" json:"linked_beads_ids,omitempty"`
	LinkedWorkstreams    []string `yaml:"linked_workstreams,omitempty" json:"linked_workstreams,omitempty"`
	RequiredArtifacts    []string `yaml:"required_artifacts,omitempty" json:"required_artifacts,omitempty"`
	RequiredChecks       []string `yaml:"required_checks,omitempty" json:"required_checks,omitempty"`
	LinkedArtifacts      []string `yaml:"linked_artifacts,omitempty" json:"linked_artifacts,omitempty"`
	ActiveAgents         []string `yaml:"active_agents,omitempty" json:"active_agents,omitempty"`
	BlockingReasons      []string `yaml:"blocking_reasons,omitempty" json:"blocking_reasons,omitempty"`
	WaitingOn            []string `yaml:"waiting_on,omitempty" json:"waiting_on,omitempty"`
	NeedsFeedbackFrom    []string `yaml:"needs_feedback_from,omitempty" json:"needs_feedback_from,omitempty"`
	FeedbackRequest      []string `yaml:"feedback_request,omitempty" json:"feedback_request,omitempty"`
	DecisionRequired     []string `yaml:"decision_required,omitempty" json:"decision_required,omitempty"`
	AuthorUpdate         []string `yaml:"author_update,omitempty" json:"author_update,omitempty"`
	AdminActionRequired  []string `yaml:"admin_action_required,omitempty" json:"admin_action_required,omitempty"`
	DispatchedAt         string   `yaml:"dispatched_at,omitempty" json:"dispatched_at,omitempty"`
	DispatchedTo         string   `yaml:"dispatched_to,omitempty" json:"dispatched_to,omitempty"`
	DispatchedPacketPath string   `yaml:"dispatched_packet_path,omitempty" json:"dispatched_packet_path,omitempty"`
}

type CardSummary struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Status              string   `json:"status"`
	RiskLevel           string   `json:"risk_level,omitempty"`
	RecommendedNextStep string   `json:"recommended_next_step,omitempty"`
	ActiveAgents        []string `json:"active_agents,omitempty"`
	WaitingOn           []string `json:"waiting_on,omitempty"`
	NeedsFeedbackFrom   []string `json:"needs_feedback_from,omitempty"`
	AuthorUpdate        []string `json:"author_update,omitempty"`
	AdminActionRequired []string `json:"admin_action_required,omitempty"`
	LinkedBeadsIDs      []string `json:"linked_beads_ids,omitempty"`
}

type ProjectBoardSnapshot struct {
	SpecVersion      string                   `json:"spec_version"`
	Timestamp        string                   `json:"timestamp"`
	Project          map[string]string        `json:"project"`
	Columns          map[string][]CardSummary `json:"columns"`
	Counts           map[string]int           `json:"counts"`
	ExecutionSummary map[string]any           `json:"execution_summary,omitempty"`
	NextAction       map[string]string        `json:"next_action"`
}

type PortfolioBoardSnapshot struct {
	SpecVersion string                 `json:"spec_version"`
	Timestamp   string                 `json:"timestamp"`
	Projects    []map[string]any       `json:"projects"`
	Totals      map[string]int         `json:"totals"`
	Queues      map[string][]QueueItem `json:"queues"`
	NextAction  map[string]string      `json:"next_action"`
}

type QueueItem struct {
	ProjectID           string   `json:"project_id"`
	CardID              string   `json:"card_id"`
	Title               string   `json:"title"`
	Status              string   `json:"status"`
	Reason              string   `json:"reason,omitempty"`
	RecommendedNextStep string   `json:"recommended_next_step,omitempty"`
	ActiveAgents        []string `json:"active_agents,omitempty"`
	NeedsFeedbackFrom   []string `json:"needs_feedback_from,omitempty"`
	AuthorUpdate        []string `json:"author_update,omitempty"`
	AdminActionRequired []string `json:"admin_action_required,omitempty"`
}

type ProjectRegistry struct {
	Projects []RegistryProject `yaml:"projects"`
}

type RegistryProject struct {
	ID          string `yaml:"id"`
	RepoURL     string `yaml:"repo_url"`
	BeadsPrefix string `yaml:"beads_prefix"`
}

type Store struct {
	ProjectRoot string
	ControlRoot string
	Registry    ProjectRegistry
}

func Open(projectRoot string) (*Store, error) {
	regPath := filepath.Join(projectRoot, "docs/specs/project-registry.yaml")
	data, err := os.ReadFile(regPath)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var reg ProjectRegistry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &Store{ProjectRoot: projectRoot, ControlRoot: filepath.Join(projectRoot, defaultControlRoot), Registry: reg}, nil
}

func (s *Store) CreateCard(projectID, title, rawRequest string) (*FeatureCard, error) {
	if !s.hasProject(projectID) {
		return nil, ErrUnknownProject
	}
	now := time.Now().UTC()
	id, err := s.nextCardID(projectID, now)
	if err != nil {
		return nil, err
	}
	card := &FeatureCard{
		ID:           id,
		ProjectID:    projectID,
		Title:        title,
		Status:       "inbox",
		RawRequest:   rawRequest,
		CreatedAt:    now.Format(time.RFC3339),
		UpdatedAt:    now.Format(time.RFC3339),
		ActiveAgents: []string{"orchestrator"},
	}
	card.IntakeArtifact = []string{filepath.ToSlash(filepath.Join(s.ControlRoot, "projects", projectID, "intake", id+".md"))}
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	if err := s.ensureIntakeArtifact(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Store) SaveCard(card *FeatureCard) error {
	if card == nil {
		return fmt.Errorf("nil card")
	}
	if err := os.MkdirAll(s.cardsDir(card.ProjectID), 0o755); err != nil {
		return err
	}
	card.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := yaml.Marshal(card)
	if err != nil {
		return err
	}
	return os.WriteFile(s.cardPath(card.ProjectID, card.ID), data, 0o644)
}

func (s *Store) LoadCards(projectID string) ([]FeatureCard, error) {
	pattern := filepath.Join(s.cardsDir(projectID), "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	cards := make([]FeatureCard, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var c FeatureCard
		if err := yaml.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		cards = append(cards, c)
	}
	return cards, nil
}

func (s *Store) BuildProjectSnapshot(projectID string) (*ProjectBoardSnapshot, error) {
	if !s.hasProject(projectID) {
		return nil, ErrUnknownProject
	}
	cards, err := s.LoadCards(projectID)
	if err != nil {
		return nil, err
	}
	columns := map[string][]CardSummary{}
	counts := map[string]int{}
	for _, status := range []string{"inbox", "clarifying", "ready", "executing", "reviewing", "blocked", "done", "parked", "needs_input"} {
		columns[status] = []CardSummary{}
		counts[status] = 0
	}
	for _, c := range cards {
		s := summarize(c)
		columns[c.Status] = append(columns[c.Status], s)
		counts[c.Status]++
	}
	proj := s.projectMeta(projectID)
	snap := &ProjectBoardSnapshot{
		SpecVersion: specVersion,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Project: map[string]string{
			"project_id":   projectID,
			"name":         projectID,
			"beads_prefix": proj.BeadsPrefix,
			"repo_url":     proj.RepoURL,
		},
		Columns:    columns,
		Counts:     counts,
		NextAction: deriveNextAction(cards),
	}
	return snap, s.writeJSON(filepath.Join(s.projectSnapshotsDir(projectID), "board.json"), snap)
}

func (s *Store) BuildPortfolioSnapshot() (*PortfolioBoardSnapshot, error) {
	projects := make([]map[string]any, 0, len(s.Registry.Projects))
	totals := map[string]int{"inbox": 0, "clarifying": 0, "ready": 0, "executing": 0, "reviewing": 0, "blocked": 0, "done": 0, "parked": 0, "needs_input": 0}
	queues := map[string][]QueueItem{"waiting_on_human": {}, "ready_to_execute": {}, "blocked": {}}
	for _, p := range s.Registry.Projects {
		snap, err := s.BuildProjectSnapshot(p.ID)
		if err != nil && !errors.Is(err, ErrUnknownProject) {
			return nil, err
		}
		if snap == nil {
			continue
		}
		projects = append(projects, map[string]any{"project_id": p.ID, "name": p.ID, "counts": snap.Counts, "next_action": snap.NextAction})
		for k, v := range snap.Counts {
			totals[k] += v
		}
		for _, c := range snap.Columns["needs_input"] {
			queues["waiting_on_human"] = append(queues["waiting_on_human"], QueueItem{ProjectID: p.ID, CardID: c.ID, Title: c.Title, Status: c.Status, RecommendedNextStep: c.RecommendedNextStep, ActiveAgents: c.ActiveAgents, NeedsFeedbackFrom: c.NeedsFeedbackFrom, AuthorUpdate: c.AuthorUpdate, AdminActionRequired: c.AdminActionRequired})
		}
		for _, c := range snap.Columns["ready"] {
			queues["ready_to_execute"] = append(queues["ready_to_execute"], QueueItem{ProjectID: p.ID, CardID: c.ID, Title: c.Title, Status: c.Status, RecommendedNextStep: c.RecommendedNextStep, ActiveAgents: c.ActiveAgents})
		}
		for _, c := range snap.Columns["blocked"] {
			queues["blocked"] = append(queues["blocked"], QueueItem{ProjectID: p.ID, CardID: c.ID, Title: c.Title, Status: c.Status, RecommendedNextStep: c.RecommendedNextStep, ActiveAgents: c.ActiveAgents, NeedsFeedbackFrom: c.NeedsFeedbackFrom, AuthorUpdate: c.AuthorUpdate, AdminActionRequired: c.AdminActionRequired})
		}
	}
	portfolio := &PortfolioBoardSnapshot{SpecVersion: specVersion, Timestamp: time.Now().UTC().Format(time.RFC3339), Projects: projects, Totals: totals, Queues: queues, NextAction: derivePortfolioAction(queues)}
	return portfolio, s.writeJSON(filepath.Join(s.ControlRoot, "portfolio", "snapshot.json"), portfolio)
}

func summarize(c FeatureCard) CardSummary {
	return CardSummary{ID: c.ID, Title: c.Title, Status: c.Status, RiskLevel: c.RiskLevel, RecommendedNextStep: c.RecommendedNext, ActiveAgents: c.ActiveAgents, WaitingOn: c.WaitingOn, NeedsFeedbackFrom: c.NeedsFeedbackFrom, AuthorUpdate: c.AuthorUpdate, AdminActionRequired: c.AdminActionRequired, LinkedBeadsIDs: c.LinkedBeadsIDs}
}

func deriveNextAction(cards []FeatureCard) map[string]string {
	for _, c := range cards {
		if c.Status == "needs_input" {
			return map[string]string{"recommended": "request_human_input", "reason": "A card is waiting on human/admin feedback", "target_card_id": c.ID}
		}
	}
	for _, c := range cards {
		if c.Status == "ready" {
			return map[string]string{"recommended": "spawn_execution", "reason": "A ready card can move into execution", "target_card_id": c.ID}
		}
	}
	for _, c := range cards {
		if c.Status == "inbox" || c.Status == "clarifying" {
			return map[string]string{"recommended": "continue_clarification", "reason": "A card can be shaped further", "target_card_id": c.ID}
		}
	}
	return map[string]string{"recommended": "idle", "reason": "No immediate project action needed"}
}

func derivePortfolioAction(queues map[string][]QueueItem) map[string]string {
	if items := queues["waiting_on_human"]; len(items) > 0 {
		return map[string]string{"recommended": "surface_feedback_request", "reason": "At least one card needs human/admin input", "target_project_id": items[0].ProjectID, "target_card_id": items[0].CardID}
	}
	if items := queues["ready_to_execute"]; len(items) > 0 {
		return map[string]string{"recommended": "start_execution", "reason": "At least one card is ready to execute", "target_project_id": items[0].ProjectID, "target_card_id": items[0].CardID}
	}
	return map[string]string{"recommended": "idle", "reason": "No immediate portfolio action needed"}
}

func (s *Store) hasProject(projectID string) bool {
	for _, p := range s.Registry.Projects {
		if p.ID == projectID {
			return true
		}
	}
	return false
}
func (s *Store) projectMeta(projectID string) RegistryProject {
	for _, p := range s.Registry.Projects {
		if p.ID == projectID {
			return p
		}
	}
	return RegistryProject{ID: projectID}
}
func (s *Store) projectDir(projectID string) string {
	return filepath.Join(s.ControlRoot, "projects", projectID)
}
func (s *Store) cardsDir(projectID string) string {
	return filepath.Join(s.projectDir(projectID), "cards")
}
func (s *Store) projectSnapshotsDir(projectID string) string {
	return filepath.Join(s.projectDir(projectID), "snapshots")
}
func (s *Store) intakeDir(projectID string) string {
	return filepath.Join(s.projectDir(projectID), "intake")
}
func (s *Store) cardPath(projectID, cardID string) string {
	return filepath.Join(s.cardsDir(projectID), cardID+".yaml")
}

func (s *Store) nextCardID(projectID string, now time.Time) (string, error) {
	files, err := filepath.Glob(filepath.Join(s.cardsDir(projectID), fmt.Sprintf("feature-%s-%s-*.yaml", projectID, now.Format("2006-01-02"))))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("feature-%s-%s-%03d", projectID, now.Format("2006-01-02"), len(files)+1), nil
}

func (s *Store) ensureIntakeArtifact(card *FeatureCard) error {
	if err := os.MkdirAll(s.intakeDir(card.ProjectID), 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.intakeDir(card.ProjectID), card.ID+".md")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	body := fmt.Sprintf("# Intake: %s\n\n## Raw request\n%s\n\n## Card\n- id: %s\n- project: %s\n- status: %s\n", card.Title, strings.TrimSpace(card.RawRequest), card.ID, card.ProjectID, card.Status)
	return os.WriteFile(path, []byte(body), 0o644)
}

func (s *Store) writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := jsonMarshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
