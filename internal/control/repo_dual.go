package control

import (
	"context"
	"fmt"
	"log"
	"time"
)

// DualWriteRepository implements CardRepository with dual-write and shadow-read.
// Primary: FileCardRepository (current source of truth).
// Shadow: BeadsCardRepository (writes are mirrored, reads are compared).
//
// During migration (R4), this detects drift between the two stores.
// When confidence is high enough, we flip primary to Beads (R5).
type DualWriteRepository struct {
	primary *FileCardRepository
	shadow  *BeadsCardRepository
	logger  *log.Logger
}

// NewDualWriteRepository creates a dual-write repository.
func NewDualWriteRepository(primary *FileCardRepository, shadow *BeadsCardRepository, logger *log.Logger) *DualWriteRepository {
	if logger == nil {
		logger = log.New(log.Writer(), "[dual-write] ", log.LstdFlags)
	}
	return &DualWriteRepository{
		primary: primary,
		shadow:  shadow,
		logger:  logger,
	}
}

// DriftReport captures differences between primary and shadow stores.
type DriftReport struct {
	GeneratedAt   time.Time   `json:"generated_at"`
	TotalPrimary  int         `json:"total_primary"`
	TotalShadow   int         `json:"total_shadow"`
	MissingShadow []string    `json:"missing_in_shadow"`  // IDs in primary but not shadow
	MissingPrimary []string   `json:"missing_in_primary"` // IDs in shadow but not primary
	StatusMismatches []StatusMismatch `json:"status_mismatches"`
	ShadowErrors   []string    `json:"shadow_errors"` // IDs that failed shadow read
}

// StatusMismatch records a status difference for a single card.
type StatusMismatch struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Primary     string `json:"primary_status"`
	Shadow      string `json:"shadow_status"`
}

// CreateCard writes to primary, then shadows to Beads.
func (d *DualWriteRepository) CreateCard(projectID string, card *FeatureCard) error {
	// Primary write first
	if err := d.primary.CreateCard(projectID, card); err != nil {
		return fmt.Errorf("primary create: %w", err)
	}

	// Shadow write (best-effort, log errors but don't fail)
	if err := d.shadow.CreateCard(projectID, card); err != nil {
		d.logger.Printf("WARN: shadow create failed for %s: %v", card.ID, err)
	}

	return nil
}

// SaveCard writes to primary, then shadows to Beads.
func (d *DualWriteRepository) SaveCard(card *FeatureCard) error {
	if err := d.primary.SaveCard(card); err != nil {
		return fmt.Errorf("primary save: %w", err)
	}

	if err := d.shadow.SaveCard(card); err != nil {
		d.logger.Printf("WARN: shadow save failed for %s: %v", card.ID, err)
	}

	return nil
}

// LoadCard reads from primary (shadow not consulted for single reads).
func (d *DualWriteRepository) LoadCard(projectID, cardID string) (*FeatureCard, error) {
	return d.primary.LoadCard(projectID, cardID)
}

// LoadCardByID reads from primary.
func (d *DualWriteRepository) LoadCardByID(cardID string) (*FeatureCard, error) {
	return d.primary.LoadCardByID(cardID)
}

// LoadCards reads from primary.
func (d *DualWriteRepository) LoadCards(projectID string) ([]FeatureCard, error) {
	return d.primary.LoadCards(projectID)
}

// Compare generates a drift report between primary and shadow stores.
// This is the core R4 operation — call it periodically to build migration confidence.
func (d *DualWriteRepository) Compare(ctx context.Context, projectID string) (*DriftReport, error) {
	report := &DriftReport{
		GeneratedAt: time.Now().UTC(),
	}

	// Load from primary
	primaryCards, err := d.primary.LoadCards(projectID)
	if err != nil {
		return nil, fmt.Errorf("primary load: %w", err)
	}
	report.TotalPrimary = len(primaryCards)

	// Build primary index
	primaryIndex := make(map[string]*FeatureCard, len(primaryCards))
	for i := range primaryCards {
		primaryIndex[primaryCards[i].ID] = &primaryCards[i]
	}

	// Load from shadow
	var shadowCards []FeatureCard
	if d.shadow != nil {
		var err error
		shadowCards, err = d.shadow.LoadCards(projectID)
		if err != nil {
			d.logger.Printf("WARN: shadow load failed for project %s: %v", projectID, err)
		}
	}
	report.TotalShadow = len(shadowCards)

	// Build shadow index
	shadowIndex := make(map[string]FeatureCard, len(shadowCards))
	for _, c := range shadowCards {
		shadowIndex[c.ID] = c
	}

	// Find missing in shadow
	for id := range primaryIndex {
		if _, ok := shadowIndex[id]; !ok {
			report.MissingShadow = append(report.MissingShadow, id)
		}
	}

	// Find missing in primary
	for id := range shadowIndex {
		if _, ok := primaryIndex[id]; !ok {
			report.MissingPrimary = append(report.MissingPrimary, id)
		}
	}

	// Find status mismatches
	for id, primaryCard := range primaryIndex {
		if shadowCard, ok := shadowIndex[id]; ok {
			if primaryCard.Status != shadowCard.Status {
				report.StatusMismatches = append(report.StatusMismatches, StatusMismatch{
					ID:      id,
					Title:   primaryCard.Title,
					Primary: primaryCard.Status,
					Shadow:  shadowCard.Status,
				})
			}
		}
	}

	return report, nil
}

// QueryReady queries from shadow (Beads is the authority on readiness).
func (d *DualWriteRepository) QueryReady() ([]FeatureCard, error) {
	return d.shadow.QueryReady()
}

// SetState sets operational state on shadow (Beads owns state).
func (d *DualWriteRepository) SetState(issueID, dimension, value, reason string) error {
	return d.shadow.SetState(issueID, dimension, value, reason)
}

// CreateGate creates a gate on shadow.
func (d *DualWriteRepository) CreateGate(parentID, gateType string) (string, error) {
	return d.shadow.CreateGate(parentID, gateType)
}

// ResolveGate resolves a gate on shadow.
func (d *DualWriteRepository) ResolveGate(gateID string) error {
	return d.shadow.ResolveGate(gateID)
}
