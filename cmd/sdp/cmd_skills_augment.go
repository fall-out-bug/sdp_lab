package main

import (
	"flag"
	"fmt"
	"os"

	skills "github.com/fall-out-bug/sdp_lab/internal/skills"
)

func runSkillsAugment(args []string) {
	fs := flag.NewFlagSet("skills augment", flag.ExitOnError)
	stackConfig := fs.String("stack", "", "Path to stack configuration JSON file (required)")
	skillsDir := fs.String("skills-dir", ".agents/skills", "Directory containing skill files")
	dryRun := fs.Bool("dry-run", false, "Print changes without writing files")
	validateOnly := fs.Bool("validate", false, "Only validate markers, don't augment")
	verbose := fs.Bool("v", false, "Verbose output")

	_ = fs.Parse(args)

	// Validate required flags
	if *stackConfig == "" {
		fmt.Fprintln(os.Stderr, "error: --stack flag is required")
		fmt.Fprintln(os.Stderr, "usage: sdp skills augment --stack <config.json> [--skills-dir DIR] [--dry-run] [--validate] [-v]")
		os.Exit(2)
	}

	// Load stack configuration
	config, err := skills.LoadStackConfig(*stackConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load stack config: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Loaded stack config: %s (%s)\n", config.Stack, config.DisplayName)
		fmt.Printf("Skills directory: %s\n", *skillsDir)
		fmt.Printf("Sections: %d\n", len(config.Sections))
	}

	// Find all skill files
	skillFiles, err := skills.FindSkillFiles(*skillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to find skill files: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Found %d skill files\n", len(skillFiles))
	}

	// Validation mode
	if *validateOnly {
		if *verbose {
			fmt.Println("\nValidating markers...")
		}
		errorCount := 0
		for _, skillFile := range skillFiles {
			if err := skills.ValidateMarkers(skillFile); err != nil {
				fmt.Printf("FAIL: %s\n", skillFile)
				fmt.Printf("  %v\n", err)
				errorCount++
			} else if *verbose {
				fmt.Printf("OK: %s\n", skillFile)
			}
		}
		if errorCount > 0 {
			fmt.Fprintf(os.Stderr, "\nValidation failed: %d files have errors\n", errorCount)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("\nValidation passed: all %d files have valid markers\n", len(skillFiles))
		}
		return
	}

	// Dry-run mode
	if *dryRun {
		if *verbose {
			fmt.Println("\nRunning dry-run augmentation...")
		}
		changeCount := 0
		for _, skillFile := range skillFiles {
			// Run dry-run for all files, including those without markers
			// The actual augmentation may add new markers to files without them
			output, err := skills.DryRunAugment(skillFile, config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to dry-run %s: %v\n", skillFile, err)
				os.Exit(1)
			}
			fmt.Println(output)
			if containsChanges(output) {
				changeCount++
			}
		}
		if *verbose {
			fmt.Printf("\nDry-run complete: %d files would be modified\n", changeCount)
		}
		return
	}

	// Augmentation mode
	if *verbose {
		fmt.Println("\nAugmenting skill files...")
	}
	successCount := 0
	errorCount := 0
	for _, skillFile := range skillFiles {
		if err := skills.AugmentSkill(skillFile, config); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to augment %s: %v\n", skillFile, err)
			errorCount++
		} else {
			if *verbose {
				fmt.Printf("OK: %s\n", skillFile)
			}
			successCount++
		}
	}

	if *verbose {
		fmt.Printf("\nAugmentation complete: %d succeeded, %d failed\n", successCount, errorCount)
	}

	if errorCount > 0 {
		os.Exit(1)
	}
}

// readFileContent reads the content of a file (helper for dry-run).
func readFileContent(filePath string) []byte {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	return content
}

// containsChanges checks if dry-run output indicates changes would be made.
func containsChanges(output string) bool {
	return hasSubstring(output, "content would be updated") || hasSubstring(output, "new marker would be added")
}

// hasSubstring checks if a string contains a substring.
func hasSubstring(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

// indexOfSubstring finds the index of a substring.
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// runSkills handles the 'skills' root command and delegates to subcommands.
func runSkills(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp skills <command> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Available commands:")
		fmt.Fprintln(os.Stderr, "  augment    Augment skill files with stack-specific content")
		fmt.Fprintln(os.Stderr, "  update     Auto-detect stack and augment all skills")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Use 'sdp skills <command> -h' for more information on a command.")
		os.Exit(2)
	}

	switch args[0] {
	case "augment":
		runSkillsAugment(args[1:])
	case "update":
		runSkillsUpdate(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown skills command '%s'\n", args[0])
		fmt.Fprintln(os.Stderr, "Available commands: augment, update")
		os.Exit(2)
	}
}
