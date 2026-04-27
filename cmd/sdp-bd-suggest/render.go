//go:build sdp_experimental

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/bdseverity"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/bdtype"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/knn"
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
	Escalated  bool           `json:"escalated"`
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
func renderJSON(w io.Writer, title string, sev bdseverity.BdSeverityResult, typ bdtype.BdTypeResult, sevEscalated, typeEscalated bool) error {
	out := jsonOutput{
		Title: title,
		Type: classifierOutput{
			Value:      typ.Type,
			Confidence: typ.Confidence(),
			Status:     string(typ.ConfStatus()),
			Escalated:  typeEscalated,
			Neighbors:  toNeighborJSON(typ.Neighbors),
		},
		Priority: classifierOutput{
			Value:      sev.Priority,
			Confidence: sev.Confidence(),
			Status:     string(sev.ConfStatus()),
			Escalated:  sevEscalated,
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

// errWriter wraps io.Writer to capture the first write error so callers
// can check a single error value instead of each individual fmt.Fprintf call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	ew.err = err
	return n, err
}

// renderHuman writes a human-readable summary to w.
func renderHuman(w io.Writer, title string, sev bdseverity.BdSeverityResult, typ bdtype.BdTypeResult, sevEscalated, typeEscalated bool) error {
	ew := &errWriter{w: w}

	fmt.Fprintf(ew, "Title: %s\n\n", title)

	typeEscStr := ""
	if typeEscalated {
		typeEscStr = " (escalated)"
	}
	fmt.Fprintf(ew, "Type:     %-8s [%s, confidence: %.2f%s]\n",
		typ.Type, string(typ.ConfStatus()), typ.Confidence(), typeEscStr)

	sevEscStr := ""
	if sevEscalated {
		sevEscStr = " (escalated)"
	}
	fmt.Fprintf(ew, "Priority: %-8s [%s, confidence: %.2f%s]\n\n",
		sev.Priority, string(sev.ConfStatus()), sev.Confidence(), sevEscStr)

	// Show top neighbors for type classifier.
	if len(typ.Neighbors) > 0 {
		fmt.Fprintln(ew, "Top neighbors (type):")
		limit := 3
		if limit > len(typ.Neighbors) {
			limit = len(typ.Neighbors)
		}
		for i, n := range typ.Neighbors[:limit] {
			fmt.Fprintf(ew, "  %d. %-12s %-8s %.2f  %q\n",
				i+1, n.Metadata, n.Label, n.Score, n.Metadata)
		}
	}
	return ew.err
}
