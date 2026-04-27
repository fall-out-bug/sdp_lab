package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	skills "github.com/fall-out-bug/sdp_lab/internal/skills"
)

func runSkillsUpdate(args []string) {
	fs := flag.NewFlagSet("skills update", flag.ExitOnError)
	projectRoot := fs.String("project-root", ".", "Project root directory (default: current directory)")
	dryRun := fs.Bool("dry-run", false, "Show what would change without writing files")
	verbose := fs.Bool("v", false, "Verbose output")

	_ = fs.Parse(args)

	// Detect stack
	stackConfigPath, err := skills.DetectStack(*projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to detect stack: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Detected stack config: %s\n", stackConfigPath)
	}

	// Load stack configuration
	config, err := skills.LoadStackConfig(stackConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load stack config: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Loaded stack config: %s (%s)\n", config.Stack, config.DisplayName)
		fmt.Printf("Sections: %d\n", len(config.Sections))
	}

	// Find all skill files
	skillsDir := filepath.Join(*projectRoot, ".agents", "skills")
	skillFiles, err := skills.FindSkillFiles(skillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to find skill files: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Found %d skill files in %s\n", len(skillFiles), skillsDir)
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

	// Validate all markers after augmentation
	if *verbose {
		fmt.Println("\nValidating markers...")
	}
	validationErrorCount := 0
	for _, skillFile := range skillFiles {
		if err := skills.ValidateMarkers(skillFile); err != nil {
			fmt.Printf("FAIL: %s\n", skillFile)
			fmt.Printf("  %v\n", err)
			validationErrorCount++
		} else if *verbose {
			fmt.Printf("OK: %s\n", skillFile)
		}
	}

	if validationErrorCount > 0 {
		fmt.Fprintf(os.Stderr, "\nValidation failed: %d files have errors\n", validationErrorCount)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("\nValidation passed: all %d files have valid markers\n", len(skillFiles))
	}

	// Print summary
	fmt.Printf("\nAugmented %d skills for %s stack\n", successCount, config.DisplayName)

	if errorCount > 0 {
		os.Exit(1)
	}
}
