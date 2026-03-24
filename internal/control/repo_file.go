package control

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// FileCardRepository persists FeatureCards as YAML files under the control root.
type FileCardRepository struct {
	projectRoot string
	controlRoot string
	registry    ProjectRegistry
}

func NewFileCardRepository(projectRoot, controlRoot string, registry ProjectRegistry) *FileCardRepository {
	return &FileCardRepository{
		projectRoot: projectRoot,
		controlRoot: controlRoot,
		registry:    registry,
	}
}

func (r *FileCardRepository) CreateCard(projectID string, card *FeatureCard) error {
	if card == nil {
		return fmt.Errorf("nil card")
	}
	if err := os.MkdirAll(r.cardsDir(projectID), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(card)
	if err != nil {
		return err
	}
	return os.WriteFile(r.cardPath(projectID, card.ID), data, 0o644)
}

func (r *FileCardRepository) SaveCard(card *FeatureCard) error {
	if card == nil {
		return fmt.Errorf("nil card")
	}
	if err := os.MkdirAll(r.cardsDir(card.ProjectID), 0o755); err != nil {
		return err
	}
	card.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := yaml.Marshal(card)
	if err != nil {
		return err
	}
	return os.WriteFile(r.cardPath(card.ProjectID, card.ID), data, 0o644)
}

func (r *FileCardRepository) LoadCard(projectID, cardID string) (*FeatureCard, error) {
	cards, err := r.LoadCards(projectID)
	if err != nil {
		return nil, err
	}
	for _, c := range cards {
		if c.ID == cardID {
			card := c
			return &card, nil
		}
	}
	return nil, fmt.Errorf("card not found: %s", cardID)
}

func (r *FileCardRepository) LoadCards(projectID string) ([]FeatureCard, error) {
	pattern := filepath.Join(r.cardsDir(projectID), "*.yaml")
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

func (r *FileCardRepository) LoadCardByID(cardID string) (*FeatureCard, error) {
	for _, project := range r.registry.Projects {
		cards, err := r.LoadCards(project.ID)
		if err != nil {
			continue
		}
		for _, c := range cards {
			if c.ID == cardID {
				card := c
				return &card, nil
			}
		}
	}
	return nil, fmt.Errorf("card not found: %s", cardID)
}

func (r *FileCardRepository) projectDir(projectID string) string {
	return filepath.Join(r.controlRoot, "projects", projectID)
}

func (r *FileCardRepository) cardsDir(projectID string) string {
	return filepath.Join(r.projectDir(projectID), "cards")
}

func (r *FileCardRepository) cardPath(projectID, cardID string) string {
	return filepath.Join(r.cardsDir(projectID), cardID+".yaml")
}
