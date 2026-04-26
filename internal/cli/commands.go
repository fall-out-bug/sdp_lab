// Package cli provides command migration and initialization for the CLI registry.
// F137-03: Command migration onto registry
package cli

import (
	"fmt"
	"os"
)

// InitRegistry initializes the global registry with all known SDP commands.
// This function should be called early in main() before any command processing.
func InitRegistry() error {
	registry := GetRegistry()

	// Register all card commands
	if err := registerCardCommands(registry); err != nil {
		return fmt.Errorf("register card commands: %w", err)
	}

	// Register board commands
	if err := registerBoardCommands(registry); err != nil {
		return fmt.Errorf("register board commands: %w", err)
	}

	// Register doctor commands
	if err := registerDoctorCommands(registry); err != nil {
		return fmt.Errorf("register doctor commands: %w", err)
	}

	// Register dispatch commands
	if err := registerDispatchCommands(registry); err != nil {
		return fmt.Errorf("register dispatch commands: %w", err)
	}

	// Register result commands
	if err := registerResultCommands(registry); err != nil {
		return fmt.Errorf("register result commands: %w", err)
	}

	// Register orchestrate commands
	if err := registerOrchestrateCommands(registry); err != nil {
		return fmt.Errorf("register orchestrate commands: %w", err)
	}

	// Register query commands
	if err := registerQueryCommands(registry); err != nil {
		return fmt.Errorf("register query commands: %w", err)
	}

	// Register deploy commands
	if err := registerDeployCommands(registry); err != nil {
		return fmt.Errorf("register deploy commands: %w", err)
	}

	// Register discovery commands
	if err := registerDiscoveryCommands(registry); err != nil {
		return fmt.Errorf("register discovery commands: %w", err)
	}

	// Register pipeline commands
	if err := registerPipelineCommands(registry); err != nil {
		return fmt.Errorf("register pipeline commands: %w", err)
	}

	// Register phase commands
	if err := registerPhaseCommands(registry); err != nil {
		return fmt.Errorf("register phase commands: %w", err)
	}

	// Register scout commands
	if err := registerScoutCommands(registry); err != nil {
		return fmt.Errorf("register scout commands: %w", err)
	}

	// Register spec commands
	if err := registerSpecCommands(registry); err != nil {
		return fmt.Errorf("register spec commands: %w", err)
	}

	// Register index commands
	if err := registerIndexCommands(registry); err != nil {
		return fmt.Errorf("register index commands: %w", err)
	}

	// Register bootstrap commands
	if err := registerBootstrapCommands(registry); err != nil {
		return fmt.Errorf("register bootstrap commands: %w", err)
	}

	// Register rules commands
	if err := registerRulesCommands(registry); err != nil {
		return fmt.Errorf("register rules commands: %w", err)
	}

	// Register build commands
	if err := registerBuildCommands(registry); err != nil {
		return fmt.Errorf("register build commands: %w", err)
	}

	// Register reset commands
	if err := registerResetCommands(registry); err != nil {
		return fmt.Errorf("register reset commands: %w", err)
	}

	// Register coverage commands
	if err := registerCoverageCommands(registry); err != nil {
		return fmt.Errorf("register coverage commands: %w", err)
	}

	// Register skills commands
	if err := registerSkillsCommands(registry); err != nil {
		return fmt.Errorf("register skills commands: %w", err)
	}

	return nil
}

func registerCardCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "card",
			Category:    "Card commands",
			Description: "Manage feature cards through their lifecycle",
			Usage:       "sdp card <create|show|clarify|needs-input|ready|park|execute|heartbeat|feedback|feedback-export|message-export|resume|resume-import|reply-ingest|deliver>",
			Subcommands: []string{
				"create", "show", "clarify", "needs-input", "ready", "park", "execute",
				"heartbeat", "feedback", "feedback-export", "message-export",
				"resume", "resume-import", "reply-ingest", "deliver",
			},
			Examples: []string{
				"sdp card create --project myproject --title 'Add feature' --raw 'description'",
				"sdp card show --project myproject --id card-123",
				"sdp card ready --project myproject --id card-123",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerBoardCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "board",
			Category:    "Board commands",
			Description: "Manage kanban boards for projects",
			Usage:       "sdp board <build|show>",
			Subcommands: []string{"build", "show"},
			Examples: []string{
				"sdp board build --project myproject",
				"sdp board show --project myproject",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerDoctorCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "doctor",
			Category:    "Doctor commands",
			Description: "Diagnose issues with SDP control state, adapters, and backlog",
			Usage:       "sdp doctor <control|adapters|backlog|all>",
			Subcommands: []string{"control", "adapters", "backlog", "all"},
			Examples: []string{
				"sdp doctor control",
				"sdp doctor adapters --strict",
				"sdp doctor backlog",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerDispatchCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "dispatch",
			Category:    "Dispatch commands",
			Description: "Dispatch cards for execution",
			Usage:       "sdp dispatch <card|next>",
			Subcommands: []string{"card", "next"},
			Examples: []string{
				"sdp dispatch card --project myproject --id card-123",
				"sdp dispatch next",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerResultCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "result",
			Category:    "Result commands",
			Description: "Ingest and manage execution results",
			Usage:       "sdp result ingest",
			Subcommands: []string{"ingest"},
			Examples: []string{
				"sdp result ingest --file result.json",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerOrchestrateCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "orchestrate",
			Category:    "Orchestrate commands",
			Description: "Run orchestration loop for result processing",
			Usage:       "sdp orchestrate once",
			Subcommands: []string{"once"},
			Examples: []string{
				"sdp orchestrate once",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerQueryCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "why",
			Category:    "Query commands (require beads/dual mode)",
			Description: "Show why a card is blocked",
			Usage:       "sdp why <card-id>",
			Examples:    []string{"sdp why card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "next",
			Category:    "Query commands (require beads/dual mode)",
			Description: "Show next actionable items",
			Usage:       "sdp next [--limit N]",
			Examples:    []string{"sdp next", "sdp next --limit 20"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "missing",
			Category:    "Query commands (require beads/dual mode)",
			Description: "Show items lacking evidence",
			Usage:       "sdp missing [project-id]",
			Examples:    []string{"sdp missing", "sdp missing myproject"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "approve",
			Category:    "Query commands (require beads/dual mode)",
			Description: "Resolve a human gate",
			Usage:       "sdp approve <card-id>",
			Examples:    []string{"sdp approve card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "trace",
			Category:    "Query commands (require beads/dual mode)",
			Description: "Show full feature trace",
			Usage:       "sdp trace <card-id>",
			Examples:    []string{"sdp trace card-123"},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerDeployCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "deploy",
			Category:    "Deploy commands",
			Description: "Manage deployments to staging and production",
			Usage:       "sdp deploy <staging|prod|rollback>",
			Subcommands: []string{"staging", "prod", "rollback"},
			Examples: []string{
				"sdp deploy staging /path/to/project",
				"sdp deploy prod v1.2.3 /path/to/project",
				"sdp deploy rollback v1.2.2 /path/to/project",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerDiscoveryCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "discover",
			Category:    "Discovery commands (Stage 0)",
			Description: "Run discovery pipeline (FRAME + SCAN + checkpoint)",
			Usage:       "sdp discover \"raw idea\"",
			Examples:    []string{"sdp discover \"Add user authentication\""},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerPipelineCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "intent",
			Category:    "Pipeline commands",
			Description: "Create intake card from raw intent",
			Usage:       "sdp intent \"description\"",
			Examples:    []string{"sdp intent \"I need to add OAuth support\""},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "status",
			Category:    "Pipeline commands",
			Description: "Show card status and phase",
			Usage:       "sdp status <card-id>",
			Examples:    []string{"sdp status card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "stuck",
			Category:    "Pipeline commands",
			Description: "Show stuck/long-running cards",
			Usage:       "sdp stuck",
			Examples:    []string{"sdp stuck"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "eval",
			Category:    "Pipeline commands",
			Description: "Run build evaluation manually",
			Usage:       "sdp eval <card-id>",
			Examples:    []string{"sdp eval card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "clarify",
			Category:    "Pipeline commands",
			Description: "Run clarification manually",
			Usage:       "sdp clarify <card-id>",
			Examples:    []string{"sdp clarify card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "plan",
			Category:    "Pipeline commands",
			Description: "Show plan for a card",
			Usage:       "sdp plan <card-id>",
			Examples:    []string{"sdp plan card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "approve-plan",
			Category:    "Pipeline commands",
			Description: "Approve a pending plan",
			Usage:       "sdp approve-plan <card-id>",
			Examples:    []string{"sdp approve-plan card-123"},
			IntroducedIn: "v1.0.0",
		},
		{
			Name:        "attention",
			Category:    "Pipeline commands",
			Description: "Show cards requiring attention",
			Usage:       "sdp attention",
			Examples:    []string{"sdp attention"},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerPhaseCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "phase",
			Category:    "Phase commands",
			Description: "Run phase-specific operations",
			Usage:       "sdp phase <plan|review|eval>",
			Subcommands: []string{"plan", "review", "eval"},
			Examples: []string{
				"sdp phase plan --feature-id F042",
				"sdp phase review --ws-id 00-042-01",
				"sdp phase eval --run-id run-123",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerScoutCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "scout",
			Category:    "Scout commands",
			Description: "Scout repository for code patterns",
			Usage:       "sdp scout [--format json|text|card] [--output DIR] <repo-path>",
			Examples: []string{
				"sdp scout /path/to/repo",
				"sdp scout --format json --output ./results /path/to/repo",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerSpecCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "spec",
			Category:    "Spec commands",
			Description: "Extract specifications from repository",
			Usage:       "sdp spec [--format json|text] [--category api|rules|invariants|sla] [--output DIR] [--enrich] [--diff] <repo-path>",
			Examples: []string{
				"sdp spec /path/to/repo",
				"sdp spec --category api --format json /path/to/repo",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerIndexCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "index",
			Category:    "Index commands",
			Description: "Build and query repository indexes",
			Usage:       "sdp index <build|stats|manifest>",
			Subcommands: []string{"build", "stats", "manifest"},
			Examples: []string{
				"sdp index build /path/to/repo",
				"sdp index stats /path/to/repo",
				"sdp index manifest --output ./docs /path/to/repo",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerBootstrapCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "bootstrap",
			Category:    "Bootstrap commands",
			Description: "Initialize repository with SDP workstreams",
			Usage:       "sdp bootstrap [--dry-run] [--force] [--beads] [--yes] [--auto-curate] [--only TYPES] <repo-path>",
			Examples: []string{
				"sdp bootstrap /path/to/repo",
				"sdp bootstrap --dry-run --only feature,epic /path/to/repo",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerRulesCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "rules",
			Category:    "Rules commands",
			Description: "Update rules from evidence sources",
			Usage:       "sdp rules update <repo-path> [--source-evidence=<dir>] [--manifest=<file>] [--format json|text]",
			Examples: []string{
				"sdp rules update /path/to/repo",
				"sdp rules update --source-evidence ./evidence /path/to/repo",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerBuildCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "build",
			Category:    "Build commands",
			Description: "Run build pipeline for a feature idea",
			Usage:       "sdp build \"<idea>\" [--strict] [--local] [--sandbox=<type>] [--dry-run] [--format json|text] [--output DIR] [--timeout DURATION]",
			Examples: []string{
				"sdp build \"Add user authentication\"",
				"sdp build --strict --format json \"Add OAuth support\"",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerResetCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "reset",
			Category:    "Reset commands",
			Description: "Reset checkpoint for a feature",
			Usage:       "sdp reset --feature F042 [--dry-run] [--yes]",
			Examples:    []string{"sdp reset --feature F042 --dry-run"},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerCoverageCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "coverage-scan",
			Category:    "Coverage commands",
			Description: "Scan code coverage",
			Usage:       "sdp coverage-scan [--path DIR] [--threshold PCT] [--format text|json] [--skip-test] [--package PATTERN] [--coverprofile FILE]",
			Examples: []string{
				"sdp coverage-scan --threshold 80",
				"sdp coverage-scan --format json --package ./internal/...",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func registerSkillsCommands(r *Registry) error {
	commands := []*CommandMetadata{
		{
			Name:        "skills",
			Category:    "Skills commands",
			Description: "Manage and augment skills",
			Usage:       "sdp skills <augment|update>",
			Subcommands: []string{"augment", "update"},
			Examples: []string{
				"sdp skills augment --stack config.json",
				"sdp skills update --project-root /path/to/project",
			},
			IntroducedIn: "v1.0.0",
		},
	}

	for _, cmd := range commands {
		if err := r.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

// CheckForDeprecatedCommands checks if the given command is deprecated and prints warnings.
// Returns true if the command is deprecated.
func CheckForDeprecatedCommands(cmd string) bool {
	registry := GetRegistry()
	metadata, exists := registry.Lookup(cmd)
	if !exists {
		return false
	}

	if metadata.Deprecated {
		fmt.Fprintf(os.Stderr, "WARNING: Command '%s' is deprecated.\n", cmd)
		if metadata.DeprecationMessage != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", metadata.DeprecationMessage)
		}
		return true
	}

	return false
}
