package artifact

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type IngestRequest struct {
	IssueID       string
	ArtifactID    string
	ArtifactClass string
	Phase         string
	Role          string
	CapturedAt    string
	Payload       any
}

type ArtifactEnvelope struct {
	IssueID       string
	ArtifactID    string
	ArtifactClass string
	Phase         string
	Role          string
	CapturedAt    string
	Payload       json.RawMessage
	Provenance    ProvenanceRecord
}

type ProvenanceIndexEntry struct {
	ContractVersion string
	HashAlgorithm   string
	IssueID         string
	Sequence        uint64
	ArtifactID      string
	ArtifactClass   string
	Phase           string
	Role            string
	CapturedAt      string
	Hash            string
	HashPrev        string
	PayloadDigest   string
}

type HashChainMetadata struct {
	IssueID      string
	RecordCount  int
	GenesisHash  string
	LastSequence uint64
	HeadHash     string
}

type BusService struct {
	mu            sync.RWMutex
	byIssue       map[string][]ArtifactEnvelope
	byIssueAndID  map[string]map[string]ArtifactEnvelope
	byHash        map[string]ArtifactEnvelope
	provenanceIdx map[string][]ProvenanceIndexEntry
}

func NewBusService() *BusService {
	return &BusService{
		byIssue:       map[string][]ArtifactEnvelope{},
		byIssueAndID:  map[string]map[string]ArtifactEnvelope{},
		byHash:        map[string]ArtifactEnvelope{},
		provenanceIdx: map[string][]ProvenanceIndexEntry{},
	}
}

func (s *BusService) Ingest(req IngestRequest) (ArtifactEnvelope, error) {
	if err := validateIngestRequest(req); err != nil {
		return ArtifactEnvelope{}, err
	}
	payload, err := CanonicalJSON(req.Payload)
	if err != nil {
		return ArtifactEnvelope{}, fmt.Errorf("canonical payload: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byIssueAndID[req.IssueID]; !ok {
		s.byIssueAndID[req.IssueID] = map[string]ArtifactEnvelope{}
	}
	if _, exists := s.byIssueAndID[req.IssueID][req.ArtifactID]; exists {
		return ArtifactEnvelope{}, fmt.Errorf("artifact %q already exists for issue %q", req.ArtifactID, req.IssueID)
	}

	sequence := uint64(len(s.byIssue[req.IssueID]))
	hashPrev := ""
	var previous *ProvenanceRecord
	if sequence > 0 {
		last := s.byIssue[req.IssueID][len(s.byIssue[req.IssueID])-1]
		hashPrev = last.Provenance.Hash
		prevCopy := last.Provenance
		previous = &prevCopy
	}

	provenance, err := BuildProvenanceRecord(ProvenanceInput{
		IssueID:       req.IssueID,
		ArtifactID:    req.ArtifactID,
		ArtifactClass: req.ArtifactClass,
		Phase:         req.Phase,
		Role:          req.Role,
		CapturedAt:    req.CapturedAt,
		Sequence:      sequence,
		HashPrev:      hashPrev,
	}, req.Payload)
	if err != nil {
		return ArtifactEnvelope{}, err
	}
	if err := ValidateAppend(previous, provenance); err != nil {
		return ArtifactEnvelope{}, err
	}

	envelope := ArtifactEnvelope{
		IssueID:       req.IssueID,
		ArtifactID:    req.ArtifactID,
		ArtifactClass: req.ArtifactClass,
		Phase:         req.Phase,
		Role:          req.Role,
		CapturedAt:    req.CapturedAt,
		Payload:       append([]byte(nil), payload...),
		Provenance:    provenance,
	}

	s.byIssue[req.IssueID] = append(s.byIssue[req.IssueID], envelope)
	s.byIssueAndID[req.IssueID][req.ArtifactID] = envelope
	s.byHash[provenance.Hash] = envelope
	s.provenanceIdx[req.IssueID] = append(s.provenanceIdx[req.IssueID], ProvenanceIndexEntry{
		ContractVersion: provenance.ContractVersion,
		HashAlgorithm:   provenance.HashAlgorithm,
		IssueID:         req.IssueID,
		Sequence:        provenance.Sequence,
		ArtifactID:      req.ArtifactID,
		ArtifactClass:   req.ArtifactClass,
		Phase:           req.Phase,
		Role:            req.Role,
		CapturedAt:      req.CapturedAt,
		Hash:            provenance.Hash,
		HashPrev:        provenance.HashPrev,
		PayloadDigest:   provenance.PayloadDigest,
	})

	return cloneEnvelope(envelope), nil
}

func (s *BusService) GetByIssueArtifactID(issueID, artifactID string) (ArtifactEnvelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byID, ok := s.byIssueAndID[issueID]
	if !ok {
		return ArtifactEnvelope{}, false
	}
	envelope, ok := byID[artifactID]
	if !ok {
		return ArtifactEnvelope{}, false
	}
	return cloneEnvelope(envelope), true
}

func (s *BusService) GetByIssueSequence(issueID string, sequence uint64) (ArtifactEnvelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream := s.byIssue[issueID]
	if len(stream) == 0 || sequence >= uint64(len(stream)) {
		return ArtifactEnvelope{}, false
	}
	return cloneEnvelope(stream[sequence]), true
}

func (s *BusService) LatestByIssue(issueID string) (ArtifactEnvelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream := s.byIssue[issueID]
	if len(stream) == 0 {
		return ArtifactEnvelope{}, false
	}
	return cloneEnvelope(stream[len(stream)-1]), true
}

func (s *BusService) GetByHash(hash string) (ArtifactEnvelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	envelope, ok := s.byHash[hash]
	if !ok {
		return ArtifactEnvelope{}, false
	}
	return cloneEnvelope(envelope), true
}

func (s *BusService) ListByIssue(issueID string) []ArtifactEnvelope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.byIssue[issueID]
	out := make([]ArtifactEnvelope, 0, len(records))
	for _, r := range records {
		out = append(out, cloneEnvelope(r))
	}
	return out
}

func (s *BusService) ProvenanceIndex(issueID string) []ProvenanceIndexEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := s.provenanceIdx[issueID]
	out := make([]ProvenanceIndexEntry, 0, len(rows))
	out = append(out, rows...)
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}

func (s *BusService) ChainMetadata(issueID string) (HashChainMetadata, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream := s.byIssue[issueID]
	if len(stream) == 0 {
		return HashChainMetadata{}, false
	}
	last := stream[len(stream)-1]
	genesis := stream[0]
	return HashChainMetadata{
		IssueID:      issueID,
		RecordCount:  len(stream),
		GenesisHash:  genesis.Provenance.Hash,
		LastSequence: last.Provenance.Sequence,
		HeadHash:     last.Provenance.Hash,
	}, true
}

func validateIngestRequest(req IngestRequest) error {
	for _, field := range []string{req.IssueID, req.ArtifactID, req.ArtifactClass, req.Phase, req.Role, req.CapturedAt} {
		if strings.TrimSpace(field) == "" {
			return errors.New("ingest request contains empty required field")
		}
	}
	if req.Payload == nil {
		return errors.New("ingest request payload is required")
	}
	return nil
}

func cloneEnvelope(in ArtifactEnvelope) ArtifactEnvelope {
	out := in
	out.Payload = append([]byte(nil), in.Payload...)
	return out
}
