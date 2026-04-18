package c4

import (
	"strings"

	"sdp_dev/internal/architect"
)

// GenerateLevel3 creates Level 3 (Component) nodes within each container from
// import graph clusters across all 5 language extractors.
// It assigns clusters to containers based on path matching.
func GenerateLevel3(profile *architect.CodebaseProfile, containers []architect.C4Container) {
	assignComponents(profile, &architect.ReferenceModel{Containers: containers})
}

// InferLevel3Relationships creates component-to-component relationships within
// a container based on import graph edges and path hierarchy.
func InferLevel3Relationships(container *architect.C4Container) []architect.C4Relationship {
	var rels []architect.C4Relationship
	seen := make(map[string]bool)

	add := func(r architect.C4Relationship) {
		key := r.From + "|" + r.To + "|" + r.Type
		if seen[key] {
			return
		}
		seen[key] = true
		rels = append(rels, r)
	}

	for i, compA := range container.Components {
		for j, compB := range container.Components {
			if i >= j {
				continue
			}
			// If component paths are related by prefix, add a relationship.
			if compA.Path != "" && compB.Path != "" {
				if strings.HasPrefix(compA.Path, compB.Path) || strings.HasPrefix(compB.Path, compA.Path) {
					add(architect.C4Relationship{
						From:        compA.ID,
						To:          compB.ID,
						Description: "internal dependency",
						Type:        "sync",
					})
				}
			}
		}
	}

	return rels
}
