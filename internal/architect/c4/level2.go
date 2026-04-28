package c4

import (
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// GenerateLevel2 creates Level 2 (Container) nodes from deploy unit signals.
// It applies priority-ordered heuristics to discover deployable units from:
// 1. Docker containers (Dockerfile, docker-compose)
// 2. Kubernetes workloads
// 3. docker-compose service dependencies
// 4. Module boundaries (cmd/, maven, gradle, npm workspaces)
// 5. Import graph clusters (fallback)
func GenerateLevel2(profile *architect.CodebaseProfile) []architect.C4Container {
	return detectContainers(profile)
}

// InferLevel2Relationships creates container-to-container relationships from:
// - docker-compose depends_on
// - Module boundary dependencies (Maven/Gradle multi-module)
// - Import graph cross-cluster edges
// - Database persistence relationships
func InferLevel2Relationships(profile *architect.CodebaseProfile, containers []architect.C4Container) []architect.C4Relationship {
	var rels []architect.C4Relationship
	seen := make(map[string]bool)

	add := func(r architect.C4Relationship) {
		key := r.From + "|" + r.To + "|" + r.Type + "|" + r.Contract
		if seen[key] {
			return
		}
		seen[key] = true
		rels = append(rels, r)
	}

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
	for _, mb := range profile.Infra.ModuleBoundaries {
		if len(mb.Children) < 2 {
			continue
		}
		for i, childA := range mb.Children {
			nameA := filepath.Base(childA)
			idA := containerID(nameA)
			foundA := false
			for _, c := range containers {
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
				for _, c := range containers {
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
		containerForPackage := buildPackageToContainerMap(profile, &architect.ReferenceModel{Containers: containers})

		for _, cluster := range profile.ImportGraph.Clusters {
			fromContainer := containerForPackage[cluster.ID]
			if fromContainer == "" {
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

			if cluster.ExternalEdges > 0 {
				for _, otherCluster := range profile.ImportGraph.Clusters {
					if otherCluster.ID == cluster.ID {
						continue
					}
					toContainer := containerForPackage[otherCluster.ID]
					if toContainer == "" {
						continue
					}
					if toContainer == fromContainer {
						continue
					}
					add(architect.C4Relationship{
						From:        fromContainer,
						To:          toContainer,
						Description: "import dependency",
						Type:        "sync",
					})
				}
			}
		}
	}

	// Database persistence relationships.
	for _, c := range containers {
		if !isServiceContainer(c) {
			continue
		}
		for _, db := range containers {
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

	return rels
}
