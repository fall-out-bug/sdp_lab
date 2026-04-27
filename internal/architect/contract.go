package architect

// ContractState represents the lifecycle state of a contract.
type ContractState string

const (
	ContractObserved  ContractState = "observed"  // auto-discovered from existing specs
	ContractProposed  ContractState = "proposed"  // suggested by AI or human draft
	ContractReference ContractState = "reference" // human-approved, enforced in CI
)

// ContractEndpoint identifies a provider or consumer of a contract.
type ContractEndpoint struct {
	Container string `json:"container"`
	Component string `json:"component,omitempty"`
}

// Contract represents a discovered or declared integration contract.
type Contract struct {
	ID               string           `json:"id"`
	Type             string           `json:"type"`              // "http_api", "async_event", "data", "grpc"
	Format           string           `json:"format"`            // "openapi", "asyncapi", "protobuf", "graphql", "sql_migration", "json_schema"
	SourcePath       string           `json:"source_path"`
	State            ContractState    `json:"state"`
	Provider         ContractEndpoint `json:"provider"`
	Consumers        []ContractEndpoint `json:"consumers,omitempty"`
	Confidence       float64          `json:"confidence"`
	ValidationStatus string           `json:"validation_status,omitempty"` // "pass", "fail", "pending"
	Note             string           `json:"note,omitempty"`
}

// ContractGap identifies a missing contract between components.
type ContractGap struct {
	Type     string           `json:"type"`     // "http_api", "async_event", "data"
	Between  ContractEndpoint `json:"between"`
	And      ContractEndpoint `json:"and"`
	Severity Severity         `json:"severity"`
	Note     string           `json:"note,omitempty"`
}

// ContractCatalog is the full inventory of contracts for a repository.
type ContractCatalog struct {
	Contracts []Contract    `json:"contracts,omitempty"`
	Gaps      []ContractGap `json:"gaps,omitempty"`
}
