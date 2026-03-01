// sdp-beads-bridge provides CLI commands for Beads-SDP integration.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/workstream"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "formula":
		formulaCmd(os.Args[2:])
	case "query":
		queryCmd(os.Args[2:])
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`sdp-beads-bridge - Beads-SDP integration CLI

Commands:
  formula apply <formula.yaml> [--vars key=value]  Generate workstreams from formula
  formula list                                    List available formulas
  formula show <name>                             Show formula details
  query ready [--format json|text]                Query ready issues (no blockers)
  query deps [--type blocks|parent-child|discovered-from|related]
  query stats                                     Show dependency statistics

Examples:
  sdp-beads-bridge formula apply .beads/formulas/feature.yaml
  sdp-beads-bridge formula apply feature.yaml --vars name=auth priority=1
  sdp-beads-bridge formula list
  sdp-beads-bridge formula show my-feature
  sdp-beads-bridge query ready --format json
  sdp-beads-bridge query deps --type blocks
  sdp-beads-bridge query stats`)
}

func formulaCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sdp-beads-bridge formula <subcommand>")
		os.Exit(1)
	}

	subcmd := args[0]
	switch subcmd {
	case "apply":
		applyFormula(args[1:])
	case "list":
		listFormulas()
	case "show":
		showFormula(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown formula subcommand: %s\n", subcmd)
		os.Exit(1)
	}
}

func applyFormula(args []string) {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	varsFlag := fs.String("vars", "", "Variables as key=value,key=value")
	featureFlag := fs.String("feature", "", "Feature ID (e.g., F061)")
	projectFlag := fs.String("project", "00", "Project ID")
	outputFlag := fs.String("output", "docs/workstreams/backlog", "Output directory")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sdp-beads-bridge formula apply <formula.yaml> [--vars key=value]")
		os.Exit(1)
	}

	formulaPath := fs.Arg(0)

	// Parse variables
	vars := make(map[string]string)
	if *varsFlag != "" {
		for _, pair := range strings.Split(*varsFlag, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}
	}

	// Parse formula
	parser := beads.NewFormulaParser()
	formula, err := parser.ParseFile(formulaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing formula: %v\n", err)
		os.Exit(1)
	}

	// Generate workstreams
	wt := workstream.NewWorkstreamTemplate(workstream.TemplateConfig{
		ProjectID: *projectFlag,
		FeatureID: *featureFlag,
		OutputDir: *outputFlag,
	})

	files, err := wt.Generate(formula, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating workstreams: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d workstream files:\n", len(files))
	for _, f := range files {
		fmt.Printf("  - %s\n", f)
	}
}

func listFormulas() {
	parser := beads.NewFormulaParser()
	formulas, err := parser.ListFormulas()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing formulas: %v\n", err)
		os.Exit(1)
	}

	if len(formulas) == 0 {
		fmt.Println("No formulas found in search paths:")
		fmt.Println("  - .beads/formulas/")
		fmt.Println("  - ~/.beads/formulas/")
		return
	}

	fmt.Printf("Found %d formulas:\n\n", len(formulas))
	for _, f := range formulas {
		fmt.Printf("  %s", f.Name)
		if f.Version != "" {
			fmt.Printf(" (v%s)", f.Version)
		}
		if f.Description != "" {
			fmt.Printf(" - %s", f.Description)
		}
		fmt.Println()
		fmt.Printf("    %d steps, %d variables\n", len(f.Steps), len(f.Variables))
	}
}

func showFormula(args []string) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sdp-beads-bridge formula show <name>")
		os.Exit(1)
	}

	name := fs.Arg(0)

	parser := beads.NewFormulaParser()
	formula, err := parser.FindFormula(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonFlag {
		data, _ := json.MarshalIndent(formula, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Formula: %s", formula.Name)
	if formula.Version != "" {
		fmt.Printf(" (v%s)", formula.Version)
	}
	fmt.Println()
	
	if formula.Description != "" {
		fmt.Printf("\n%s\n", formula.Description)
	}
	
	fmt.Printf("\nSteps (%d):\n", len(formula.Steps))
	for i, step := range formula.Steps {
		fmt.Printf("  %d. %s", i+1, step.Name)
		if step.Title != "" {
			fmt.Printf(" - %s", step.Title)
		}
		fmt.Println()
	}
	
	if len(formula.Variables) > 0 {
		fmt.Printf("\nVariables (%d):\n", len(formula.Variables))
		for name, v := range formula.Variables {
			fmt.Printf("  - %s", name)
			if v.Type != "" {
				fmt.Printf(" (%s)", v.Type)
			}
			if v.Required {
				fmt.Print(" [required]")
			}
			if v.Default != nil {
				fmt.Printf(" default=%v", v.Default)
			}
			fmt.Println()
		}
	}
	
	if len(formula.Dependencies) > 0 {
		fmt.Printf("\nDependencies: %v\n", formula.Dependencies)
	}
	
	if formula.Extends != "" {
		fmt.Printf("\nExtends: %s\n", formula.Extends)
	}
}
