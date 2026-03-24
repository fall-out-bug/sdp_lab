package control

// CardRepository defines the persistence boundary for FeatureCards.
// It mirrors the storage-oriented subset of Store's API.
type CardRepository interface {
	CreateCard(projectID string, card *FeatureCard) error
	SaveCard(card *FeatureCard) error
	LoadCard(projectID, cardID string) (*FeatureCard, error)
	LoadCards(projectID string) ([]FeatureCard, error)
	LoadCardByID(cardID string) (*FeatureCard, error)
}
