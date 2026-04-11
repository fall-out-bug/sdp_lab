package c4

import (
	"fmt"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// inferRelationships creates C4Relationship edges between containers and
// components based on import dependencies, service deps, infra signals, etc.
func inferRelationships(profile *architect.CodebaseProfile, model *architect.ReferenceModel) []architect.C4Relationship {
	var rels []architect.C4Relationship
	seen := make(map[string]bool) // dedup key

	add := func(r architect.C4Relationship) {
		key := r.From + "|" + r.To + "|" + r.Type + "|" + r.Contract
		if seen[key] {
			return
		}
		seen[key] = true
		rels = append(rels, r)
	}

	// --- Container-to-Container relationships ---

	// From docker-compose depends_on.
	for _, sd := range profile.Infra.Services {
		fromID := containerID(sd.From)
		toID := containerID(sd.To)
		if fromID == toID {
			continue
		}
		add(architect.C4Relationship{
			From:        fromID,
			To:          toID,
			Description: "depends on",
			Type:        "sync",
		})
	}

	// From module boundaries (Maven/Gradle multi-module projects).
	// When a module boundary has children, create relationships between them.
	for _, mb := range profile.Infra.ModuleBoundaries {
		if len(mb.Children) < 2 {
			continue
		}
		// Each child module relates to sibling modules
		for i, childA := range mb.Children {
			nameA := filepath.Base(childA)
			idA := containerID(nameA)
			// Verify idA is actually a container
			foundA := false
			for _, c := range model.Containers {
				if c.ID == idA {
					foundA = true
					break
				}
			}
			if !foundA {
				continue
			}
			for j, childB := range mb.Children {
				if i == j {
					continue
				}
				nameB := filepath.Base(childB)
				idB := containerID(nameB)
				foundB := false
				for _, c := range model.Containers {
					if c.ID == idB {
						foundB = true
						break
					}
				}
				if !foundB {
					continue
				}
				add(architect.C4Relationship{
					From:        idA,
					To:          idB,
					Description: mb.BuildSystem + " module dependency",
					Type:        "sync",
				})
			}
		}
	}

		// From import graph clusters (cross-container imports).
	if profile.ImportGraph.Clusters != nil {
		containerForPackage := buildPackageToContainerMap(profile, model)

		for _, cluster := range profile.ImportGraph.Clusters {
			fromContainer := containerForPackage[cluster.ID]
			if fromContainer == "" {
				// Try to match by package prefix.
				for _, pkg := range cluster.Packages {
					if c := containerForPackage[pkg]; c != "" {
						fromContainer = c
						break
					}
				}
			}
			if fromContainer == "" {
				continue
			}

			// Infer cross-container relationships from external edges.
			if cluster.ExternalEdges > 0 {
				// For each external edge, try to determine the target container.
				// This is a heuristic: external edges likely go to other containers
				// or external systems.
				for _, otherCluster := range profile.ImportGraph.Clusters {
					if otherCluster.ID == cluster.ID {
						continue
					}
					toContainer := containerForPackage[otherCluster.ID]
					if toContainer == "" {
						continue
					}
					if toContainer == fromContainer {
						continue // intra-container
					}
					add(architect.C4Relationship{
						From:        fromContainer,
						To:          toContainer,
						Description: fmt.Sprintf("%d external imports", cluster.ExternalEdges),
						Type:        "sync",
					})
				}
			}
		}
	}

	// --- Actor-to-Container relationships (L1) ---
	for _, actor := range model.Actors {
		if len(model.Containers) == 0 {
			continue
		}
		// Connect actor to the first service-type container.
		for _, c := range model.Containers {
			if isServiceContainer(c) {
				add(architect.C4Relationship{
					From:        actor.ID,
					To:          c.ID,
					Description: "uses",
					Type:        "sync",
				})
				break
			}
		}
	}

	// --- Container-to-External relationships (L1) ---
	for _, ext := range model.ExternalSystems {
		// Connect to whichever container likely uses the external system.
		for _, c := range model.Containers {
			if isServiceContainer(c) {
				add(architect.C4Relationship{
					From:        c.ID,
					To:          ext.ID,
					Description: "calls",
					Type:        "sync",
				})
				break
			}
		}
	}

	// --- Component-to-Component relationships within containers (L3) ---
	for _, container := range model.Containers {
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
	}

	// --- Database persistence relationships ---
	// If a database container exists, connect service containers to it.
	for _, c := range model.Containers {
		if !isServiceContainer(c) {
			continue
		}
		for _, db := range model.Containers {
			if isDatabaseContainer(db) && db.ID != c.ID {
				add(architect.C4Relationship{
					From:        c.ID,
					To:          db.ID,
					Description: "persists data to",
					Type:        "data",
				})
			}
		}
	}

	// --- Spec-based relationships (Implements edges) ---
	for _, spec := range profile.Specs {
		// Find the container whose source path best matches the spec location.
		bestContainer := ""
		for _, c := range model.Containers {
			if c.Source != "" && spec.Path != "" {
				if pathsRelated(c.Source, spec.Path) {
					bestContainer = c.ID
					break
				}
			}
		}
		if bestContainer == "" && len(model.Containers) > 0 {
			bestContainer = model.Containers[0].ID
		}
		// Mark the spec as a contract reference on existing edges.
		for i := range rels {
			if rels[i].From == bestContainer && rels[i].Contract == "" {
				rels[i].Contract = spec.Path
			}
		}
	}

	return rels
}

// buildPackageToContainerMap maps package/cluster IDs to container IDs.
func buildPackageToContainerMap(profile *architect.CodebaseProfile, model *architect.ReferenceModel) map[string]string {
	result := make(map[string]string)

	for _, cluster := range profile.ImportGraph.Clusters {
		bestContainer := ""
		bestScore := 0
		for _, c := range model.Containers {
			score := matchScore(strings.ToLower(cluster.ID), strings.ToLower(c.Name))
			if score > bestScore {
				bestScore = score
				bestContainer = c.ID
			}
			// Also check source path.
			if c.Source != "" {
				score = matchScore(strings.ToLower(cluster.ID), strings.ToLower(c.Source))
				if score > bestScore {
					bestScore = score
					bestContainer = c.ID
				}
			}
		}
		if bestContainer != "" {
			result[cluster.ID] = bestContainer
		}
		for _, pkg := range cluster.Packages {
			if result[pkg] == "" {
				result[pkg] = bestContainer
			}
		}
	}

	return result
}

// isServiceContainer returns true if the container is a deployable service.
func isServiceContainer(c architect.C4Container) bool {
	tech := strings.ToLower(c.Technology)
	desc := strings.ToLower(c.Description)
	return !strings.Contains(desc, "database:") &&
		!strings.Contains(desc, "cache:") &&
		!strings.Contains(desc, "message broker:") &&
		!containsAny(tech, "postgres", "mysql", "mongo", "redis", "rabbitmq", "kafka")
}

// isDatabaseContainer returns true if the container represents a database.
func isDatabaseContainer(c architect.C4Container) bool {
	desc := strings.ToLower(c.Description)
	tech := strings.ToLower(c.Technology)
	return strings.Contains(desc, "database:") || strings.Contains(desc, "managed database") ||
		containsAny(tech, "postgres", "mysql", "mongo", "mariadb", "cockroach", "dynamodb")
}

// pathsRelated returns true if two paths share a common prefix directory.
func pathsRelated(a, b string) bool {
	aParts := strings.Split(a, "/")
	bParts := strings.Split(b, "/")
	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}
	shared := 0
	for i := 0; i < minLen-1; i++ {
		if aParts[i] == bParts[i] {
			shared++
		}
	}
	return shared > 0
}
