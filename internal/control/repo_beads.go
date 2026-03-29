package control

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// BeadsCardRepository implements CardRepository using the `bd` CLI.
// This is a transitional shim (see R3 in BEADS_FIRST_CONTROL_TOWER_ROADMAP.md).
// Target: replace with Go SDK when Beads exposes one.
type BeadsCardRepository struct {
	bdPath string // path to bd binary (default: "bd")
	dbPath string // optional override for --db flag
	logger *log.Logger
}

// NewBeadsCardRepository creates a CLI-backed Beads repository.
func NewBeadsCardRepository(dbPath string, logger *log.Logger) *BeadsCardRepository {
	if logger == nil {
		logger = log.New(log.Writer(), "[beads-repo] ", log.LstdFlags)
	}
	return &BeadsCardRepository{
		bdPath: "bd",
		dbPath: dbPath,
		logger: logger,
	}
}

// bdIssue represents a Beads issue as returned by `bd show --json`.
type bdIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Type        string   `json:"issue_type"`
	Owner       string   `json:"owner"`
	CreatedAt   string   `json:"created_at"`
	CreatedBy   string   `json:"created_by"`
	UpdatedAt   string   `json:"updated_at"`
	Labels      []string `json:"labels"`
}

// runBDWrite executes a bd write command and returns raw output (no --json).
func (r *BeadsCardRepository) runBDWrite(args ...string) ([]byte, error) {
	var allArgs []string
	if r.dbPath != "" {
		allArgs = []string{"--db", r.dbPath}
	}
	allArgs = append(allArgs, args...)
	r.logger.Printf("exec: bd %s", strings.Join(args, " "))

	cmd := exec.CommandContext(context.Background(), r.bdPath, allArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bd %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// runBD executes a bd read command and returns JSON output.
func (r *BeadsCardRepository) runBD(args ...string) ([]byte, error) {
	var allArgs []string
	if r.dbPath != "" {
		allArgs = []string{"--db", r.dbPath}
	}
	allArgs = append(allArgs, "--json")
	allArgs = append(allArgs, args...)
	r.logger.Printf("exec: bd %s", strings.Join(args, " "))

	cmd := exec.CommandContext(context.Background(), r.bdPath, allArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bd %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// parseBdList parses JSON output from bd list/show (array of issues).
func parseBdList(data []byte) ([]bdIssue, error) {
	var issues []bdIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parse bd JSON: %w", err)
	}
	return issues, nil
}

// bdToCard converts a Beads issue to a FeatureCard.
func bdToCard(bd bdIssue) *FeatureCard {
	card := &FeatureCard{
		ID:        bd.ID,
		Title:     bd.Title,
		Status:    bd.Status,
		CreatedAt: bd.CreatedAt,
		UpdatedAt: bd.UpdatedAt,
	}

	// Map Beads priority to string
	card.ExecutionMode = strconv.Itoa(bd.Priority)

	// Map Beads description to normalized intent (objective)
	if bd.Description != "" {
		card.NormalizedIntent = bd.Description
	}

	// Extract SDP labels
	for _, label := range bd.Labels {
		switch {
		case strings.HasPrefix(label, "sdp:phase-"):
			card.TaskType = strings.TrimPrefix(label, "sdp:phase-")
		case strings.HasPrefix(label, "sdp:project-"):
			card.ProjectID = strings.TrimPrefix(label, "sdp:project-")
		}
	}

	return card
}

// extractSDPLabels builds Beads labels from FeatureCard fields.
func sdpLabelsFromCard(card *FeatureCard) []string {
	var labels []string
	if card.ProjectID != "" {
		labels = append(labels, fmt.Sprintf("sdp:project-%s", card.ProjectID))
	}
	if card.TaskType != "" {
		labels = append(labels, fmt.Sprintf("sdp:phase-%s", card.TaskType))
	}
	return labels
}

// CreateCard creates a Beads issue from a FeatureCard.
func (r *BeadsCardRepository) CreateCard(projectID string, card *FeatureCard) error {
	args := []string{"create", card.Title}

	if card.NormalizedIntent != "" {
		args = append(args, "--description", card.NormalizedIntent)
	}

	if card.ExecutionMode != "" {
		args = append(args, "--priority", card.ExecutionMode)
	}

	labels := sdpLabelsFromCard(card)
	if len(labels) > 0 {
		args = append(args, "--labels", strings.Join(labels, ","))
	}

	args = append(args, "--silent")

	data, err := r.runBDWrite(args...)
	if err != nil {
		return fmt.Errorf("create card: %w", err)
	}

	createdID := strings.TrimSpace(string(data))
	if createdID != "" {
		card.ID = createdID
	}

	return nil
}

// SaveCard updates an existing Beads issue.
func (r *BeadsCardRepository) SaveCard(card *FeatureCard) error {
	if card.ID == "" {
		return fmt.Errorf("cannot save card without ID")
	}

	

	switch card.Status {
	case "closed":
		_, err := r.runBDWrite("close", card.ID)
		return fmt.Errorf("close card %s: %w", card.ID, err)
	case "open":
		_, err := r.runBDWrite("reopen", card.ID)
		return fmt.Errorf("reopen card %s: %w", card.ID, err)
	default:
		a := []string{"update", card.ID}
		if card.Title != "" {
			a = append(a, "--title", card.Title)
		}
		if card.NormalizedIntent != "" {
			a = append(a, "--description", card.NormalizedIntent)
		}
		if card.ExecutionMode != "" {
			a = append(a, "--priority", card.ExecutionMode)
		}
		_, err := r.runBDWrite(a...)
		return fmt.Errorf("update card %s: %w", card.ID, err)
	}
}

// LoadCard loads a single card by project ID and card ID.
// In Beads, project is identified by label sdp:project-{name}.
func (r *BeadsCardRepository) LoadCard(projectID, cardID string) (*FeatureCard, error) {
	return r.LoadCardByID(cardID)
}

// LoadCardByID loads a single card by ID.
func (r *BeadsCardRepository) LoadCardByID(cardID string) (*FeatureCard, error) {
	data, err := r.runBD("show", cardID)
	if err != nil {
		return nil, fmt.Errorf("load card %s: %w", cardID, err)
	}

	issues, err := parseBdList(data)
	if err != nil {
		return nil, fmt.Errorf("parse card list for %s: %w", cardID, err)
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("card %s not found", cardID)
	}

	return bdToCard(issues[0]), nil
}

// LoadCards loads all cards for a project.
// Queries by label sdp:project-{projectID}.
func (r *BeadsCardRepository) LoadCards(projectID string) ([]FeatureCard, error) {
	label := fmt.Sprintf("sdp:project-%s", projectID)
	data, err := r.runBD("query", fmt.Sprintf("label=%s", label))
	if err != nil {
		return nil, fmt.Errorf("load cards for project %s: %w", projectID, err)
	}

	issues, err := parseBdList(data)
	if err != nil {
		return nil, fmt.Errorf("parse card list for project %s: %w", projectID, err)
	}

	cards := make([]FeatureCard, 0, len(issues))
	for _, issue := range issues {
		cards = append(cards, *bdToCard(issue))
	}
	return cards, nil
}

// QueryReady returns all issues that are ready (open, no blockers).
func (r *BeadsCardRepository) QueryReady() ([]FeatureCard, error) {
	data, err := r.runBD("ready")
	if err != nil {
		return nil, fmt.Errorf("query ready: %w", err)
	}

	issues, err := parseBdList(data)
	if err != nil {
		return nil, fmt.Errorf("parse ready card list: %w", err)
	}

	cards := make([]FeatureCard, 0, len(issues))
	for _, issue := range issues {
		cards = append(cards, *bdToCard(issue))
	}
	return cards, nil
}

// CreateGate creates a gate issue of the specified type.
func (r *BeadsCardRepository) CreateGate(parentID, gateType string) (string, error) {
	args := []string{
		"create", fmt.Sprintf("Gate: %s for %s", gateType, parentID),
		"--type", "chore",
		"--parent", parentID,
		"--labels", fmt.Sprintf("sdp:gate:%s", gateType),
		"--silent",
	}

	data, err := r.runBDWrite(args...)
	if err != nil {
		return "", fmt.Errorf("create gate: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ResolveGate closes a gate issue.
func (r *BeadsCardRepository) ResolveGate(gateID string) error {
	_, err := r.runBDWrite("close", gateID)
	return fmt.Errorf("resolve gate %s: %w", gateID, err)
}

// SetState sets an operational state dimension on an issue.
func (r *BeadsCardRepository) SetState(issueID, dimension, value, reason string) error {
	args := []string{
		"set-state", issueID,
		fmt.Sprintf("%s=%s", dimension, value),
	}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	_, err := r.runBDWrite(args...)
	return fmt.Errorf("set state %s=%s on %s: %w", dimension, value, issueID, err)
}

// UpdateMetadata merges SDP metadata into a Beads issue.
// Reads current metadata first, deep-merges, then writes back.
func (r *BeadsCardRepository) UpdateMetadata(issueID string, sdpMeta map[string]any) error {
	// Read current metadata from issue
	current := r.readMetadata(issueID)

	// Deep merge: new values override existing
	merged := deepMergeMetadata(current, sdpMeta)

	metaJSON, err := json.Marshal(map[string]any{"sdp": merged})
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.runBDWrite("update", issueID, "--metadata", string(metaJSON))
	return fmt.Errorf("write metadata for %s: %w", issueID, err)
}

// readMetadata reads current SDP metadata from a Beads issue.
func (r *BeadsCardRepository) readMetadata(issueID string) map[string]any {
	data, err := r.runBD("show", "--long", issueID)
	if err != nil {
		return nil
	}
	var issues []struct {
		Metadata string `json:"metadata"`
	}
	if err := json.Unmarshal(data, &issues); err != nil || len(issues) == 0 {
		return nil
	}
	if issues[0].Metadata == "" {
		return nil
	}
	var wrapped map[string]map[string]any
	if err := json.Unmarshal([]byte(issues[0].Metadata), &wrapped); err != nil {
		return nil
	}
	return wrapped["sdp"]
}

// deepMergeMetadata recursively merges src into dst.
func deepMergeMetadata(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := dst[k].(map[string]any); ok {
				dst[k] = deepMergeMetadata(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

// LinkContract associates a contract reference with an issue.
func (r *BeadsCardRepository) LinkContract(issueID, contractID, contractHash string) error {
	meta := map[string]any{
		"contract": map[string]string{
			"id":   contractID,
			"hash": contractHash,
		},
	}
	return r.UpdateMetadata(issueID, meta)
}

// LinkEvidence associates evidence artifact references with an issue.
func (r *BeadsCardRepository) LinkEvidence(issueID, phase string, artifactPaths []string) error {
	meta := map[string]any{
		fmt.Sprintf("evidence_%s", phase): map[string]any{
			"artifacts": artifactPaths,
		},
	}
	return r.UpdateMetadata(issueID, meta)
}

// SetProvenance records provenance hashes for an issue.
func (r *BeadsCardRepository) SetProvenance(issueID, packetHash, promptHash string) error {
	meta := map[string]any{
		"provenance": map[string]string{
			"packet_hash": packetHash,
			"prompt_hash": promptHash,
		},
	}
	return r.UpdateMetadata(issueID, meta)
}

// SetExecutorState records executor runtime state in metadata.
func (r *BeadsCardRepository) SetExecutorState(issueID, role, sessionID, state string) error {
	meta := map[string]any{
		"executor": map[string]string{
			"role":       role,
			"session_id": sessionID,
			"state":      state,
		},
	}
	return r.UpdateMetadata(issueID, meta)
}
