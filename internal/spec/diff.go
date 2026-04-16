package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// DiffSpecs compares two SpecReport JSON snapshots and returns the drift diff.
func DiffSpecs(oldPath, newPath string) (*SpecDiff, error) {
	old, err := loadSnapshot(oldPath)
	if err != nil {
		return nil, fmt.Errorf("diff: load old snapshot: %w", err)
	}
	nw, err := loadSnapshot(newPath)
	if err != nil {
		return nil, fmt.Errorf("diff: load new snapshot: %w", err)
	}
	d := &SpecDiff{Version: "1.0.0", OldSnapshot: oldPath,
		NewSnapshot: newPath, GeneratedAt: time.Now().UTC()}
	d.APIChanges = diffAPI(old.APIContracts, nw.APIContracts)
	d.RuleChanges = diffRules(old.BusinessRules, nw.BusinessRules)
	d.InvChanges = diffInvariants(old.Invariants, nw.Invariants)
	d.SLAChanges = diffSLA(old.SLAParameters, nw.SLAParameters)
	for _, chs := range [][]Change{d.APIChanges, d.RuleChanges, d.InvChanges, d.SLAChanges} {
		for _, c := range chs {
			d.Summary.count(c.Category)
		}
	}
	return d, nil
}

func loadSnapshot(path string) (*SpecReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r SpecReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}

func (s *DiffSummary) count(cat string) {
	switch cat {
	case "added":
		s.Added++
	case "removed":
		s.Removed++
	case "modified":
		s.Modified++
	}
}

// diffMaps compares two key→value maps and returns added/removed/modified changes.
func diffMaps(prefix, detail string, old, nw map[string]string) []Change {
	var ch []Change
	for key, val := range old {
		if nv, ok := nw[key]; !ok {
			ch = append(ch, Change{Category: "removed", Key: prefix + key, Old: val})
		} else if val != nv {
			ch = append(ch, Change{Category: "modified", Key: prefix + key,
				Old: val, New: nv, Detail: detail})
		}
	}
	for key, val := range nw {
		if _, ok := old[key]; !ok {
			ch = append(ch, Change{Category: "added", Key: prefix + key, New: val})
		}
	}
	return ch
}

func diffAPI(old, nw APIContracts) []Change {
	var ch []Change
	oi := endpointIndex(old.HTTPEndpoints)
	ni := endpointIndex(nw.HTTPEndpoints)
	for key, ep := range oi {
		if nep, ok := ni[key]; !ok {
			ch = append(ch, Change{Category: "removed", Key: key,
				Old: fmt.Sprintf("%s %s → %s", ep.Method, ep.Path, ep.Handler)})
		} else if ep.Handler != nep.Handler {
			ch = append(ch, Change{Category: "modified", Key: key,
				Old: ep.Handler, New: nep.Handler, Detail: "handler changed"})
		}
	}
	for key, ep := range ni {
		if _, ok := oi[key]; !ok {
			ch = append(ch, Change{Category: "added", Key: key,
				New: fmt.Sprintf("%s %s → %s", ep.Method, ep.Path, ep.Handler)})
		}
	}
	return ch
}

func endpointIndex(eps []Endpoint) map[string]Endpoint {
	m := make(map[string]Endpoint, len(eps))
	for _, e := range eps {
		m[e.Method+" "+e.Path] = e
	}
	return m
}

func diffRules(old, nw BusinessRules) []Change {
	oi := make(map[string]string, len(old.Validations))
	for _, r := range old.Validations {
		key := r.Location + "#" + r.Field + "#" + r.Enforcement
		oi[key] = r.Description
	}
	ni := make(map[string]string, len(nw.Validations))
	for _, r := range nw.Validations {
		key := r.Location + "#" + r.Field + "#" + r.Enforcement
		ni[key] = r.Description
	}
	return diffMaps("", "description changed", oi, ni)
}

func diffInvariants(old, nw Invariants) []Change {
	var ch []Change
	oi := make(map[string]string)
	for _, v := range old.Database {
		oi[v.Table+"."+v.Column] = v.Detail
	}
	ni := make(map[string]string)
	for _, v := range nw.Database {
		ni[v.Table+"."+v.Column] = v.Detail
	}
	ch = append(ch, diffMaps("", "constraint changed", oi, ni)...)
	// Generic invariant slices
	ch = append(ch, diffInvMap("type:", old.TypeSystem, nw.TypeSystem,
		func(v TypeInvariant) string { return v.Category + "@" + v.Location },
		func(v TypeInvariant) string { return v.Detail })...)
	ch = append(ch, diffInvMap("concurrency:", old.Concurrency, nw.Concurrency,
		func(v ConcInvariant) string { return v.Category + "@" + v.Location },
		func(v ConcInvariant) string { return v.Detail })...)
	ch = append(ch, diffInvMap("architectural:", old.Architectural, nw.Architectural,
		func(v ArchInvariant) string { return v.Category + "@" + v.Location },
		func(v ArchInvariant) string { return v.Detail })...)
	return ch
}

func diffInvMap[T any](prefix string, old, nw []T,
	keyFn func(T) string, valFn func(T) string,
) []Change {
	oi := make(map[string]string, len(old))
	for _, v := range old {
		oi[keyFn(v)] = valFn(v)
	}
	ni := make(map[string]string, len(nw))
	for _, v := range nw {
		ni[keyFn(v)] = valFn(v)
	}
	return diffMaps(prefix, "invariant changed", oi, ni)
}

func diffSLA(old, nw SLAParameters) []Change {
	var ch []Change
	slices := []struct {
		cat string
		o, n []SLAParam
	}{
		{"timeout", old.Timeouts, nw.Timeouts},
		{"retry", old.Retries, nw.Retries},
		{"rate_limit", old.RateLimits, nw.RateLimits},
		{"circuit_breaker", old.CircuitBreakers, nw.CircuitBreakers},
		{"resource_pool", old.ResourcePools, nw.ResourcePools},
		{"health_check", old.HealthChecks, nw.HealthChecks},
	}
	for _, s := range slices {
		oi := make(map[string]string, len(s.o))
		for _, p := range s.o {
			oi[p.Category+":"+p.Component] = p.Value
		}
		ni := make(map[string]string, len(s.n))
		for _, p := range s.n {
			ni[p.Category+":"+p.Component] = p.Value
		}
		ch = append(ch, diffMaps(s.cat+":", "value changed", oi, ni)...)
	}
	return ch
}
