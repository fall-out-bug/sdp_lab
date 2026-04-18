package c4

import (
	"fmt"
	"strings"

	"sdp_dev/internal/architect"
)

// GenerateLevel1 creates a Level 1 (System Context) model from infrastructure signals.
// It detects actors from ingress/exposed ports and external systems from dependencies
// and IaC resources.
func GenerateLevel1(profile *architect.CodebaseProfile, systemName string) ([]architect.Actor, []architect.ExternalSystem) {
	var actors []architect.Actor
	var externals []architect.ExternalSystem

	// Detect actors from ingress / exposed ports.
	if len(profile.Infra.Ingresses) > 0 {
		actors = append(actors, architect.Actor{
			ID:          "end-user",
			Description: "End User",
		})
	}
	if len(profile.Infra.ExposedPorts) > 0 {
		actors = append(actors, architect.Actor{
			ID:          "client",
			Description: "API Client",
		})
	}

	// Detect external systems from notable dependencies.
	seenExternal := make(map[string]bool)
	for _, dep := range profile.Dependencies.NotableDeps {
		name := dep.Name
		signal := dep.Signal
		switch signal {
		case "cloud_aws", "cloud_azure", "cloud_gcp":
			if !seenExternal[name] {
				seenExternal[name] = true
				externals = append(externals, architect.ExternalSystem{
					ID:          slug(name),
					Description: name,
					Technology:  signal,
					Evidence:    fmt.Sprintf("Found in %d manifest(s)", dep.FoundIn),
				})
			}
		}
	}

	// Detect external databases/services from Terraform resources.
	for _, res := range profile.Infra.Resources {
		switch {
		case strings.Contains(res.Type, "db_instance"),
			strings.Contains(res.Type, "rds_cluster"),
			strings.Contains(res.Type, "sql_database"):
			if !seenExternal[res.Name] {
				seenExternal[res.Name] = true
				externals = append(externals, architect.ExternalSystem{
					ID:          slug(res.Name),
					Description: "Managed Database: " + res.Name,
					Technology:  res.Provider,
					Evidence:    "Terraform resource: " + res.Type,
				})
			}
		}
	}

	// Always have at least one actor for L1.
	if len(actors) == 0 {
		actors = append(actors, architect.Actor{
			ID:          "developer",
			Description: "Developer",
		})
	}

	return actors, externals
}

// InferLevel1Relationships creates relationships between actors, the system, and external systems.
func InferLevel1Relationships(actors []architect.Actor, externals []architect.ExternalSystem, systemID string) []architect.C4Relationship {
	var rels []architect.C4Relationship

	// Actor -> System relationships
	for _, actor := range actors {
		rels = append(rels, architect.C4Relationship{
			From:        actor.ID,
			To:          systemID,
			Description: "uses",
			Type:        "sync",
		})
	}

	// System -> External System relationships
	for _, ext := range externals {
		rels = append(rels, architect.C4Relationship{
			From:        systemID,
			To:          ext.ID,
			Description: "calls",
			Type:        "sync",
		})
	}

	return rels
}
