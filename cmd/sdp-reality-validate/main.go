// sdp-reality-validate validates .sdp/reality/*.json against schema/reality/*.schema.json.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type artifactSchema struct {
	ArtifactRel string
	SchemaRel   string
}

var requiredRealityArtifacts = []artifactSchema{
	{ArtifactRel: ".sdp/reality/reality-summary.json", SchemaRel: "schema/reality/reality-summary.schema.json"},
	{ArtifactRel: ".sdp/reality/feature-inventory.json", SchemaRel: "schema/reality/feature-inventory.schema.json"},
	{ArtifactRel: ".sdp/reality/architecture-map.json", SchemaRel: "schema/reality/architecture-map.schema.json"},
	{ArtifactRel: ".sdp/reality/integration-map.json", SchemaRel: "schema/reality/integration-map.schema.json"},
	{ArtifactRel: ".sdp/reality/quality-report.json", SchemaRel: "schema/reality/quality-report.schema.json"},
	{ArtifactRel: ".sdp/reality/drift-report.json", SchemaRel: "schema/reality/drift-report.schema.json"},
	{ArtifactRel: ".sdp/reality/readiness-report.json", SchemaRel: "schema/reality/readiness-report.schema.json"},
}

func main() {
	projectRoot := "."
	if len(os.Args) > 1 {
		projectRoot = os.Args[1]
	}

	issues, err := validateRealityArtifacts(projectRoot)
	if err != nil {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "reality-validate: %s\n", issue)
		}
		fmt.Fprintf(os.Stderr, "reality-validate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("reality-validate: validated %d artifact(s)\n", len(requiredRealityArtifacts))
}

func validateRealityArtifacts(projectRoot string) ([]string, error) {
	issues := make([]string, 0)

	for _, item := range requiredRealityArtifacts {
		artifactPath := filepath.Join(projectRoot, item.ArtifactRel)

		artifactData, err := os.ReadFile(artifactPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: read artifact: %v", item.ArtifactRel, err))
			continue
		}

		var payload any
		if err := json.Unmarshal(artifactData, &payload); err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid JSON: %v", item.ArtifactRel, err))
			continue
		}

		schema, err := compileSchema(projectRoot, item.SchemaRel)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: compile schema %s: %v", item.ArtifactRel, item.SchemaRel, err))
			continue
		}
		if err := schema.Validate(payload); err != nil {
			issues = append(issues, fmt.Sprintf("%s: schema validation failed: %v", item.ArtifactRel, err))
			continue
		}
	}

	if len(issues) > 0 {
		return issues, fmt.Errorf("%d artifact(s) failed validation", len(issues))
	}
	return nil, nil
}

func compileSchema(projectRoot, schemaRel string) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()

	schemaAbs := filepath.Join(projectRoot, schemaRel)
	schemaBaseName := strings.TrimSuffix(filepath.Base(schemaRel), ".schema.json")

	claimData, err := os.ReadFile(filepath.Join(projectRoot, "schema", "reality", "claim.schema.json"))
	if err != nil {
		return nil, err
	}
	sourceData, err := os.ReadFile(filepath.Join(projectRoot, "schema", "reality", "source.schema.json"))
	if err != nil {
		return nil, err
	}

	claimAliases := []string{
		"claim.schema.json",
		fmt.Sprintf("https://sdp.dev/reality/%s/claim.schema.json", schemaBaseName),
	}
	sourceAliases := []string{
		"source.schema.json",
		fmt.Sprintf("https://sdp.dev/reality/%s/source.schema.json", schemaBaseName),
	}

	for _, alias := range claimAliases {
		if err := compiler.AddResource(alias, bytes.NewReader(claimData)); err != nil {
			return nil, err
		}
	}
	for _, alias := range sourceAliases {
		if err := compiler.AddResource(alias, bytes.NewReader(sourceData)); err != nil {
			return nil, err
		}
	}

	return compiler.Compile(schemaAbs)
}
