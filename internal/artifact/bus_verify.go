package artifact

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type IssueVerificationReport struct {
	IssueID           string
	RecordsChecked    int
	IndexRowsChecked  int
	TamperFindings    []string
	RetentionFindings []string
}

func (r IssueVerificationReport) IntegrityOK() bool {
	return len(r.TamperFindings) == 0
}

func (r IssueVerificationReport) RetentionOK() bool {
	return len(r.RetentionFindings) == 0
}

func (r IssueVerificationReport) OK() bool {
	return r.IntegrityOK() && r.RetentionOK()
}

func (s *BusService) VerifyIssue(issueID string, now time.Time) IssueVerificationReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := IssueVerificationReport{IssueID: issueID}
	stream := s.byIssue[issueID]
	report.RecordsChecked = len(stream)
	if len(stream) == 0 {
		return report
	}

	retentionByClass := map[string]int{}
	for _, class := range Classes() {
		retentionByClass[class.ID] = class.RetentionDays
	}

	var previous *ProvenanceRecord
	for i, envelope := range stream {
		if err := ValidateAppend(previous, envelope.Provenance); err != nil {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("sequence %d append validation failed: %v", i, err))
		}
		digest, err := DigestHex(envelope.Payload)
		if err != nil {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("sequence %d payload digest failed: %v", i, err))
		} else if digest != envelope.Provenance.PayloadDigest {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("sequence %d payload digest mismatch", i))
		}
		if mapped, ok := s.byHash[envelope.Provenance.Hash]; !ok {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("sequence %d missing by-hash entry", i))
		} else if mapped.Provenance.Hash != envelope.Provenance.Hash || mapped.ArtifactID != envelope.ArtifactID {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("sequence %d by-hash entry mismatch", i))
		}

		retentionDays, ok := retentionByClass[envelope.ArtifactClass]
		if !ok {
			report.RetentionFindings = append(report.RetentionFindings, fmt.Sprintf("sequence %d unknown artifact class %q", i, envelope.ArtifactClass))
		} else {
			capturedAt, err := time.Parse(time.RFC3339, envelope.CapturedAt)
			if err != nil {
				report.RetentionFindings = append(report.RetentionFindings, fmt.Sprintf("sequence %d invalid captured_at %q", i, envelope.CapturedAt))
			} else if now.After(capturedAt.AddDate(0, 0, retentionDays)) {
				report.RetentionFindings = append(report.RetentionFindings, fmt.Sprintf("sequence %d class %q exceeded retention (%d days)", i, envelope.ArtifactClass, retentionDays))
			}
		}

		prev := envelope.Provenance
		previous = &prev
	}

	rows := append([]ProvenanceIndexEntry(nil), s.provenanceIdx[issueID]...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Sequence < rows[j].Sequence })
	report.IndexRowsChecked = len(rows)
	if len(rows) != len(stream) {
		report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("provenance index row count %d does not match stream %d", len(rows), len(stream)))
	}
	for i := 0; i < len(rows) && i < len(stream); i++ {
		envelope := stream[i]
		row := rows[i]
		if row.Sequence != envelope.Provenance.Sequence {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("index sequence mismatch at position %d", i))
		}
		if row.Hash != envelope.Provenance.Hash || row.HashPrev != envelope.Provenance.HashPrev {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("index hash linkage mismatch at sequence %d", row.Sequence))
		}
		if row.PayloadDigest != envelope.Provenance.PayloadDigest {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("index payload digest mismatch at sequence %d", row.Sequence))
		}
		if row.ContractVersion != envelope.Provenance.ContractVersion || row.HashAlgorithm != envelope.Provenance.HashAlgorithm {
			report.TamperFindings = append(report.TamperFindings, fmt.Sprintf("index contract metadata mismatch at sequence %d", row.Sequence))
		}
	}

	report.TamperFindings = uniqStrings(report.TamperFindings)
	report.RetentionFindings = uniqStrings(report.RetentionFindings)
	return report
}

func uniqStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		norm := strings.TrimSpace(item)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out
}
