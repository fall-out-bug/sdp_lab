package main

import (
	"encoding/json"
	"fmt"
	"io"

	"sdp_dev/internal/inference/microfirst/bdseverity"
	"sdp_dev/internal/inference/microfirst/bdtype"
	"sdp_dev/internal/inference/microfirst/knn"
)

// neighborJSON is the JSON shape for a single nearest-neighbour entry.
type neighborJSON struct {
	Label    string  `json:"label"`
	Score    float64 `json:"score"`
	Metadata string  `json:"metadata"`
}

// classifierOutput is the JSON shape for a single classifier result.
type classifierOutput struct {
	Value      string         `json:"value"`
	Confidence float64        `json:"confidence"`
	Status     string         `json:"status"`
	Neighbors  []neighborJSON `json:"neighbors,omitempty"`
}

// jsonOutput is the top-level JSON schema.
type jsonOutput struct {
	Title    string           `json:"title"`
	Type     classifierOutput `json:"type"`
	Priority classifierOutput `json:"priority"`
}

// toNeighborJSON converts a slice of knn.Match[string] to JSON-serialisable form.
func toNeighborJSON(matches []knn.Match[string]) []neighborJSON {
	out := make([]neighborJSON, 0, len(matches))
	for _, m := range matches {
		out = append(out, neighborJSON{
			Label:    m.Label,
			Score:    m.Score,
			Metadata: m.Metadata,
		})
	}
	return out
}

// renderJSON writes the JSON output to w.
func renderJSON(w io.Writer, title string, sev bdseverity.BdSeverityResult, typ bdtype.BdTypeResult) error {
	out := jsonOutput{
		Title: title,
		Type: classifierOutput{
			Value:      typ.Type,
			Confidence: typ.Confidence(),
			Status:     string(typ.ConfStatus()),
			Neighbors:  toNeighborJSON(typ.Neighbors),
		},
		Priority: classifierOutput{
			Value:      sev.Priority,
			Confidence: sev.Confidence(),
			Status:     string(sev.ConfStatus()),
			Neighbors:  toNeighborJSON(sev.Neighbors),
		},
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("render json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// renderHuman writes a human-readable summary to w.
func renderHuman(w io.Writer, title string, sev bdseverity.BdSeverityResult, typ bdtype.BdTypeResult) error {
	fmt.Fprintf(w, "Title: %s\n\n", title)
	fmt.Fprintf(w, "Type:     %-8s [%s, confidence: %.2f]\n",
		typ.Type, string(typ.ConfStatus()), typ.Confidence())
	fmt.Fprintf(w, "Priority: %-8s [%s, confidence: %.2f]\n\n",
		sev.Priority, string(sev.ConfStatus()), sev.Confidence())

	// Show top neighbors for type classifier.
	if len(typ.Neighbors) > 0 {
		fmt.Fprintln(w, "Top neighbors (type):")
		limit := 3
		if limit > len(typ.Neighbors) {
			limit = len(typ.Neighbors)
		}
		for i, n := range typ.Neighbors[:limit] {
			fmt.Fprintf(w, "  %d. %-12s %-8s %.2f  %q\n",
				i+1, n.Metadata, n.Label, n.Score, n.Metadata)
		}
	}
	return nil
}
