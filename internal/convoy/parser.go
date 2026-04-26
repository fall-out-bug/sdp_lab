package convoy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Convoy represents a Gas Town convoy work unit
type Convoy struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      string            `json:"status"` // active, complete, blocked
	Priority    string            `json:"priority"`
	AssignedTo  string            `json:"assigned_to"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata"`
}

// ConvoyListResponse is the JSON output from `gt convoy list --json`
type ConvoyListResponse struct {
	Convoys []Convoy `json:"convoys"`
	Version string   `json:"version"`
}

// Parser parses Gas Town convoy output
type Parser struct {
	gtBinary string
}

// NewParser creates a new convoy parser
func NewParser() *Parser {
	gtBinary := "gt"
	if path := os.Getenv("GT_BINARY"); path != "" {
		gtBinary = path
	}
	return &Parser{gtBinary: gtBinary}
}

// ParseConvoyList runs `gt convoy list --json` and parses the output
func (p *Parser) ParseConvoyList() (*ConvoyListResponse, error) {
	cmd := exec.Command(p.gtBinary, "convoy", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run gt convoy list: %w", err)
	}

	var response ConvoyListResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse convoy list JSON: %w", err)
	}

	return &response, nil
}

// ParseConvoyListFromString parses convoy JSON from a string
func (p *Parser) ParseConvoyListFromString(jsonStr string) (*ConvoyListResponse, error) {
	var response ConvoyListResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse convoy list JSON: %w", err)
	}
	return &response, nil
}
