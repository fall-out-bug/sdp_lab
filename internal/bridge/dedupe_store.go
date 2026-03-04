package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const (
	findingHashLabelPrefix = "finding-key-"
	payloadHashLabelPrefix = "payload-hash-"
	stableHashLength       = 16
)

type DedupeAction string

const (
	DedupeCreate       DedupeAction = "create"
	DedupeUpdate       DedupeAction = "update"
	DedupeReopenUpdate DedupeAction = "reopen_update"
	DedupeSkip         DedupeAction = "skip"
)

type DedupeDecision struct {
	Action      DedupeAction
	IssueID     string
	FindingHash string
	PayloadHash string
}

type DedupeRecord struct {
	IssueID     string
	Status      string
	FindingHash string
	PayloadHash string
}

type ExistingIssue struct {
	ID     string
	Status string
	Labels []string
}

type DedupeStore struct {
	mu      sync.RWMutex
	records map[string]DedupeRecord
}

func NewDedupeStore() *DedupeStore {
	return &DedupeStore{records: make(map[string]DedupeRecord)}
}

func (s *DedupeStore) ImportExisting(issues []ExistingIssue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, issue := range issues {
		findingHash, payloadHash := parseHashLabels(issue.Labels)
		if findingHash == "" {
			continue
		}

		record := DedupeRecord{
			IssueID:     issue.ID,
			Status:      normalizeStatus(issue.Status),
			FindingHash: findingHash,
			PayloadHash: payloadHash,
		}

		existing, ok := s.records[findingHash]
		if !ok || shouldReplaceRecord(existing, record) {
			s.records[findingHash] = record
		}
	}
}

func (s *DedupeStore) Decide(findingHash, payloadHash string) DedupeDecision {
	s.mu.RLock()
	record, ok := s.records[findingHash]
	s.mu.RUnlock()

	if !ok {
		return DedupeDecision{
			Action:      DedupeCreate,
			FindingHash: findingHash,
			PayloadHash: payloadHash,
		}
	}

	if record.PayloadHash == payloadHash {
		return DedupeDecision{
			Action:      DedupeSkip,
			IssueID:     record.IssueID,
			FindingHash: findingHash,
			PayloadHash: payloadHash,
		}
	}

	action := DedupeUpdate
	if isClosedStatus(record.Status) {
		action = DedupeReopenUpdate
	}

	return DedupeDecision{
		Action:      action,
		IssueID:     record.IssueID,
		FindingHash: findingHash,
		PayloadHash: payloadHash,
	}
}

func (s *DedupeStore) RecordCreated(findingHash, payloadHash, issueID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[findingHash] = DedupeRecord{
		IssueID:     issueID,
		Status:      "open",
		FindingHash: findingHash,
		PayloadHash: payloadHash,
	}
}

func (s *DedupeStore) RecordUpdated(findingHash, payloadHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.records[findingHash]
	record.Status = "open"
	record.PayloadHash = payloadHash
	record.FindingHash = findingHash
	s.records[findingHash] = record
}

func (s *DedupeStore) RecordClosed(findingHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.records[findingHash]
	record.Status = "closed"
	record.FindingHash = findingHash
	s.records[findingHash] = record
}

func ProtocolFindingHashes(source FindingsSource, finding ProtocolFinding) (string, string) {
	normSource := normalizedSourceFrom(source)
	findingKey := normalizeValue(finding.FindingKey)
	if findingKey == "" {
		findingKey = fallbackKey("protocol", normalizePath(finding.File), normalizeValue(finding.Category), normalizeValue(finding.Code), fmt.Sprintf("%d", finding.Line))
	}

	identity := normalizedProtocolIdentity{
		Source:     normSource,
		FindingKey: findingKey,
		Category:   normalizeValue(finding.Category),
		Code:       normalizeValue(finding.Code),
		File:       normalizePath(finding.File),
		Line:       finding.Line,
	}

	payload := normalizedProtocolPayload{
		Identity:        identity,
		Severity:        normalizeValue(finding.Severity),
		Message:         normalizeValue(finding.Message),
		RemediationHint: normalizeValue(findingHint(finding.Remediation)),
		FeatureID:       normalizeValue(finding.Context.FeatureID),
		WSID:            normalizeValue(finding.Context.WSID),
	}

	return stableHash(identity), stableHash(payload)
}

func DocsFindingHashes(source FindingsSource, finding DocsFinding) (string, string) {
	normSource := normalizedSourceFrom(source)
	findingKey := normalizeValue(finding.FindingKey)
	if findingKey == "" {
		findingKey = fallbackKey(
			"docs",
			normalizePath(finding.File),
			normalizeValue(finding.Category),
			normalizeValue(finding.Code),
			fmt.Sprintf("%d", finding.Line),
			fmt.Sprintf("%d", finding.Column),
		)
	}

	identity := normalizedDocsIdentity{
		Source:     normSource,
		FindingKey: findingKey,
		Category:   normalizeValue(finding.Category),
		Code:       normalizeValue(finding.Code),
		File:       normalizePath(finding.File),
		Line:       finding.Line,
		Column:     finding.Column,
		EndLine:    finding.EndLine,
	}

	payload := normalizedDocsPayload{
		Identity:        identity,
		Severity:        normalizeValue(finding.Severity),
		Message:         normalizeValue(finding.Message),
		RemediationHint: normalizeValue(findingHint(finding.Remediation)),
		LinkTarget:      normalizeValue(finding.Context.LinkTarget),
		LinkText:        normalizeValue(finding.Context.LinkText),
	}

	return stableHash(identity), stableHash(payload)
}

func findingHashLabel(findingHash string) string {
	return findingHashLabelPrefix + findingHash
}

func payloadHashLabel(payloadHash string) string {
	return payloadHashLabelPrefix + payloadHash
}

func parseHashLabels(labels []string) (string, string) {
	var findingHash string
	var payloadHash string

	for _, label := range labels {
		switch {
		case strings.HasPrefix(label, findingHashLabelPrefix):
			findingHash = strings.TrimPrefix(label, findingHashLabelPrefix)
		case strings.HasPrefix(label, payloadHashLabelPrefix):
			payloadHash = strings.TrimPrefix(label, payloadHashLabelPrefix)
		}
	}

	return findingHash, payloadHash
}

func fallbackKey(parts ...string) string {
	return stableHash(parts)
}

func findingHint(remediation *Remediation) string {
	if remediation == nil {
		return ""
	}

	if remediation.Hint != "" {
		return remediation.Hint
	}

	if remediation.SuggestedFix != "" {
		return remediation.SuggestedFix
	}

	if remediation.Action != "" {
		return remediation.Action
	}

	return ""
}

func shouldReplaceRecord(current, candidate DedupeRecord) bool {
	if current.IssueID == "" && candidate.IssueID != "" {
		return true
	}

	if isClosedStatus(current.Status) && !isClosedStatus(candidate.Status) {
		return true
	}

	if current.PayloadHash == "" && candidate.PayloadHash != "" {
		return true
	}

	return false
}

func isClosedStatus(status string) bool {
	return normalizeStatus(status) == "closed"
}

func normalizeStatus(status string) string {
	return normalizeValue(status)
}

func normalizePath(path string) string {
	cleaned := strings.ReplaceAll(path, "\\", "/")
	return normalizeValue(cleaned)
}

func normalizeValue(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(trimmed), " ")
}

func stableHash(input interface{}) string {
	raw, err := json.Marshal(input)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", input))
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])[:stableHashLength]
}

type normalizedSource struct {
	CheckName  string `json:"check_name"`
	Workflow   string `json:"workflow,omitempty"`
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
}

func normalizedSourceFrom(source FindingsSource) normalizedSource {
	return normalizedSource{
		CheckName:  normalizeValue(source.CheckName),
		Workflow:   normalizeValue(source.Workflow),
		Repository: normalizeValue(source.Repository),
		Branch:     normalizeValue(source.Branch),
	}
}

type normalizedProtocolIdentity struct {
	Source     normalizedSource `json:"source"`
	FindingKey string           `json:"finding_key"`
	Category   string           `json:"category"`
	Code       string           `json:"code,omitempty"`
	File       string           `json:"file"`
	Line       int              `json:"line,omitempty"`
}

type normalizedProtocolPayload struct {
	Identity        normalizedProtocolIdentity `json:"identity"`
	Severity        string                     `json:"severity"`
	Message         string                     `json:"message"`
	RemediationHint string                     `json:"remediation_hint,omitempty"`
	FeatureID       string                     `json:"feature_id,omitempty"`
	WSID            string                     `json:"ws_id,omitempty"`
}

type normalizedDocsIdentity struct {
	Source     normalizedSource `json:"source"`
	FindingKey string           `json:"finding_key"`
	Category   string           `json:"category"`
	Code       string           `json:"code,omitempty"`
	File       string           `json:"file"`
	Line       int              `json:"line,omitempty"`
	Column     int              `json:"column,omitempty"`
	EndLine    int              `json:"end_line,omitempty"`
}

type normalizedDocsPayload struct {
	Identity        normalizedDocsIdentity `json:"identity"`
	Severity        string                 `json:"severity"`
	Message         string                 `json:"message"`
	RemediationHint string                 `json:"remediation_hint,omitempty"`
	LinkTarget      string                 `json:"link_target,omitempty"`
	LinkText        string                 `json:"link_text,omitempty"`
}
