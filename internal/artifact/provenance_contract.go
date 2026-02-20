package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	ProvenanceContractVersion = "artifact-provenance/v1"
	ProvenanceHashAlgorithm   = "sha256"
)

var deterministicSchemaFields = []string{
	"contract_version",
	"hash_algorithm",
	"issue_id",
	"artifact_id",
	"artifact_class",
	"phase",
	"role",
	"captured_at",
	"sequence",
	"hash_prev",
	"payload_digest",
	"hash",
}

type ProvenanceInput struct {
	IssueID       string
	ArtifactID    string
	ArtifactClass string
	Phase         string
	Role          string
	CapturedAt    string
	Sequence      uint64
	HashPrev      string
}

type ProvenanceRecord struct {
	ContractVersion string
	HashAlgorithm   string
	IssueID         string
	ArtifactID      string
	ArtifactClass   string
	Phase           string
	Role            string
	CapturedAt      string
	Sequence        uint64
	HashPrev        string
	PayloadDigest   string
	Hash            string
}

func DeterministicSchemaFields() []string {
	out := make([]string, len(deterministicSchemaFields))
	copy(out, deterministicSchemaFields)
	return out
}

func BuildProvenanceRecord(in ProvenanceInput, payload any) (ProvenanceRecord, error) {
	if err := validateInput(in); err != nil {
		return ProvenanceRecord{}, err
	}
	payloadDigest, err := DigestHex(payload)
	if err != nil {
		return ProvenanceRecord{}, err
	}

	record := ProvenanceRecord{
		ContractVersion: ProvenanceContractVersion,
		HashAlgorithm:   ProvenanceHashAlgorithm,
		IssueID:         in.IssueID,
		ArtifactID:      in.ArtifactID,
		ArtifactClass:   in.ArtifactClass,
		Phase:           in.Phase,
		Role:            in.Role,
		CapturedAt:      in.CapturedAt,
		Sequence:        in.Sequence,
		HashPrev:        in.HashPrev,
		PayloadDigest:   payloadDigest,
	}

	hash, err := computeRecordHash(record)
	if err != nil {
		return ProvenanceRecord{}, err
	}
	record.Hash = hash
	return record, nil
}

func ValidateAppend(previous *ProvenanceRecord, next ProvenanceRecord) error {
	if strings.TrimSpace(next.ContractVersion) != ProvenanceContractVersion {
		return fmt.Errorf("unsupported contract_version: %q", next.ContractVersion)
	}
	if strings.TrimSpace(next.HashAlgorithm) != ProvenanceHashAlgorithm {
		return fmt.Errorf("unsupported hash_algorithm: %q", next.HashAlgorithm)
	}
	for _, field := range []string{next.IssueID, next.ArtifactID, next.ArtifactClass, next.Phase, next.Role, next.CapturedAt, next.PayloadDigest, next.Hash} {
		if strings.TrimSpace(field) == "" {
			return errors.New("provenance record contains empty required field")
		}
	}
	if !isSHA256Hex(next.PayloadDigest) {
		return errors.New("invalid payload_digest")
	}
	if !isSHA256Hex(next.Hash) {
		return errors.New("invalid hash")
	}

	expected, err := computeRecordHash(next)
	if err != nil {
		return err
	}
	if next.Hash != expected {
		return errors.New("hash does not match deterministic schema")
	}

	if previous == nil {
		if next.Sequence != 0 {
			return errors.New("genesis record must have sequence 0")
		}
		if strings.TrimSpace(next.HashPrev) != "" {
			return errors.New("genesis record must have empty hash_prev")
		}
		return nil
	}

	if next.IssueID != previous.IssueID {
		return errors.New("append record issue_id mismatch")
	}
	if next.Sequence != previous.Sequence+1 {
		return errors.New("append record sequence must increment by one")
	}
	if next.HashPrev != previous.Hash {
		return errors.New("append record hash_prev must match previous hash")
	}
	return nil
}

func CanonicalJSON(v any) ([]byte, error) {
	return encodeCanonical(v)
}

func DigestHex(v any) (string, error) {
	b, err := CanonicalJSON(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validateInput(in ProvenanceInput) error {
	for _, field := range []string{in.IssueID, in.ArtifactID, in.ArtifactClass, in.Phase, in.Role, in.CapturedAt} {
		if strings.TrimSpace(field) == "" {
			return errors.New("provenance input contains empty required field")
		}
	}
	if in.Sequence == 0 && strings.TrimSpace(in.HashPrev) != "" {
		return errors.New("genesis input must have empty hash_prev")
	}
	if in.Sequence > 0 && !isSHA256Hex(in.HashPrev) {
		return errors.New("non-genesis input requires sha256 hash_prev")
	}
	return nil
}

func computeRecordHash(r ProvenanceRecord) (string, error) {
	b, err := CanonicalJSON(map[string]any{
		"contract_version": r.ContractVersion,
		"hash_algorithm":   r.HashAlgorithm,
		"issue_id":         r.IssueID,
		"artifact_id":      r.ArtifactID,
		"artifact_class":   r.ArtifactClass,
		"phase":            r.Phase,
		"role":             r.Role,
		"captured_at":      r.CapturedAt,
		"sequence":         r.Sequence,
		"hash_prev":        r.HashPrev,
		"payload_digest":   r.PayloadDigest,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func IsSHA256Hex(s string) bool {
	return isSHA256Hex(s)
}

func encodeCanonical(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte("null"), nil
	case bool, string, float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return json.Marshal(x)
	case json.Number:
		return []byte(x.String()), nil
	case []any:
		return encodeCanonicalArray(x)
	case []string:
		arr := make([]any, 0, len(x))
		for _, item := range x {
			arr = append(arr, item)
		}
		return encodeCanonicalArray(arr)
	case map[string]any:
		return encodeCanonicalMap(x)
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(b, &decoded); err != nil {
			return nil, err
		}
		return encodeCanonical(decoded)
	}
}

func encodeCanonicalArray(items []any) ([]byte, error) {
	buf := bytes.NewBufferString("[")
	for i, item := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, err := encodeCanonical(item)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func encodeCanonicalMap(m map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := bytes.NewBufferString("{")
	for i, key := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valueJSON, err := encodeCanonical(m[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valueJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
