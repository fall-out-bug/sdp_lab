// Package main implements query subcommands for sdp-beads-bridge.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/beads"
)

// queryCmd handles the query subcommand.
func queryCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sdp-beads-bridge query <subcommand>")
		fmt.Fprintln(os.Stderr, "Subcommands: ready, deps, stats, blockers")
		os.Exit(1)
	}

	subcmd := args[0]
	switch subcmd {
	case "ready":
		queryReadyCmd(args[1:])
	case "deps":
		queryDepsCmd(args[1:])
	case "stats":
		queryStatsCmd(args[1:])
	case "blockers":
		queryBlockersCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown query subcommand: %s\n", subcmd)
		os.Exit(1)
	}
}

// queryReadyCmd queries ready issues with no blockers.
func queryReadyCmd(args []string) {
	fs := flag.NewFlagSet("ready", flag.ExitOnError)
	formatFlag := fs.String("format", "text", "Output format: json or text")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}

	// Use bd ready command (fastest)
	issues, err := beads.ReadyCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying ready issues: %v\n", err)
		os.Exit(1)
	}

	switch *formatFlag {
	case "json":
		output := make([]map[string]interface{}, 0, len(issues))
		for _, issue := range issues {
			output = append(output, map[string]interface{}{
				"beads_id":   issue.ID,
				"title":      issue.Title,
				"priority":   issue.Priority,
				"status":     issue.Status,
				"labels":     issue.Labels,
				"blocked_by": issue.BlockedBy,
				"ready":      len(issue.BlockedBy) == 0,
				"ws_id":      issue.WSID,
			})
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	default:
		printReadyText(issues)
	}
}

// queryDepsCmd queries dependencies by type.
func queryDepsCmd(args []string) {
	fs := flag.NewFlagSet("deps", flag.ExitOnError)
	typeFlag := fs.String("type", "blocks", "Dependency type: blocks, parent-child, discovered-from, related")
	formatFlag := fs.String("format", "text", "Output format: json or text")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}

	issues, err := beads.ListIssuesCommand(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing issues: %v\n", err)
		os.Exit(1)
	}

	deps := make([]beads.Dependency, 0)
	for _, issue := range issues {
		if issue.DependencyCount == 0 && issue.DependentCount == 0 {
			continue
		}
		depItems, err := beads.DependencyListCommand(issue.ID, "down", *typeFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying dependencies for %s: %v\n", issue.ID, err)
			os.Exit(1)
		}
		for _, item := range depItems {
			deps = append(deps, beads.Dependency{
				FromIssueID:    issue.ID,
				ToIssueID:      item.ID,
				DependencyType: beads.DependencyType(item.DependencyType),
			})
		}
	}

	switch *formatFlag {
	case "json":
		data, _ := json.MarshalIndent(deps, "", "  ")
		fmt.Println(string(data))
	default:
		printDepsText(deps, *typeFlag)
	}
}

// queryStatsCmd shows dependency statistics.
func queryStatsCmd(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	formatFlag := fs.String("format", "text", "Output format: json or text")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}

	issues, err := beads.ListIssuesCommand(true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing issues: %v\n", err)
		os.Exit(1)
	}

	stats := make(map[beads.DependencyType]int)
	breakdown := make(map[int]int)
	for _, issue := range issues {
		if issue.Status == "open" || issue.Status == "in_progress" {
			breakdown[issue.Priority]++
		}
		if issue.DependencyCount == 0 {
			continue
		}
		depItems, err := beads.DependencyListCommand(issue.ID, "down", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying dependencies for %s: %v\n", issue.ID, err)
			os.Exit(1)
		}
		for _, item := range depItems {
			stats[beads.DependencyType(item.DependencyType)]++
		}
	}

	switch *formatFlag {
	case "json":
		output := map[string]interface{}{
			"dependencies_by_type": stats,
			"open_by_priority":     breakdown,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	default:
		printStatsText(stats, breakdown)
	}
}

// queryBlockersCmd queries blocking dependencies for an issue.
func queryBlockersCmd(args []string) {
	orderedArgs, issueID := splitBlockerArgs(args)
	fs := flag.NewFlagSet("blockers", flag.ExitOnError)
	formatFlag := fs.String("format", "text", "Output format: json or text")
	transitiveFlag := fs.Bool("transitive", false, "Show transitive blockers")
	if err := fs.Parse(orderedArgs); err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}

	if issueID == "" && fs.NArg() > 0 {
		issueID = fs.Arg(0)
	}

	if issueID == "" {
		fmt.Fprintln(os.Stderr, "Usage: sdp-beads-bridge query blockers <issue-id>")
		os.Exit(1)
	}

	var err error
	var blockers []beads.Issue
	if *transitiveFlag {
		blockers, err = beads.DependencyTreeCommand(issueID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying transitive blockers: %v\n", err)
			os.Exit(1)
		}
	} else {
		depItems, err := beads.DependencyListCommand(issueID, "down", "blocks")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying blockers: %v\n", err)
			os.Exit(1)
		}
		for _, item := range depItems {
			blockers = append(blockers, item.Issue)
		}
	}

	switch *formatFlag {
	case "json":
		data, _ := json.MarshalIndent(blockers, "", "  ")
		fmt.Println(string(data))
	default:
		printBlockersText(issueID, blockers, *transitiveFlag)
	}
}

func splitBlockerArgs(args []string) ([]string, string) {
	ordered := make([]string, 0, len(args))
	issueID := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			ordered = append(ordered, arg)
			if (arg == "--format" || arg == "-format") && i+1 < len(args) {
				i++
				ordered = append(ordered, args[i])
			}
			continue
		}
		if issueID == "" {
			issueID = arg
			continue
		}
		ordered = append(ordered, arg)
	}
	return ordered, issueID
}

func printReadyText(issues []beads.ReadyIssue) {
	ready := 0
	blocked := 0

	for _, issue := range issues {
		if len(issue.BlockedBy) == 0 {
			ready++
		} else {
			blocked++
		}
	}

	fmt.Printf("📋 Ready work (%d ready, %d blocked):\n\n", ready, blocked)

	for _, issue := range issues {
		status := "●"
		if len(issue.BlockedBy) > 0 {
			status = "○"
		}

		priority := fmt.Sprintf("P%d", issue.Priority)
		if issue.Priority == 1 {
			priority = "P1"
		}

		wsRef := ""
		if issue.WSID != "" {
			wsRef = fmt.Sprintf(" [%s]", issue.WSID)
		}

		fmt.Printf("  %s [%s] %s%s: %s\n", status, priority, issue.ID, wsRef, issue.Title)

		if len(issue.BlockedBy) > 0 {
			fmt.Printf("      blocked by: %s\n", strings.Join(issue.BlockedBy, ", "))
		}
	}
}

func printDepsText(deps []beads.Dependency, depType string) {
	fmt.Printf("📋 Dependencies (type: %s): %d total\n\n", depType, len(deps))

	for _, dep := range deps {
		fmt.Printf("  %s → %s [%s]\n", dep.FromIssueID, dep.ToIssueID, dep.DependencyType)
	}
}

func printStatsText(stats map[beads.DependencyType]int, breakdown map[int]int) {
	fmt.Println("📊 Beads Dependency Statistics")

	fmt.Println("Dependencies by type:")
	for depType, count := range stats {
		fmt.Printf("  %-20s %d\n", string(depType)+":", count)
	}

	if len(breakdown) > 0 {
		fmt.Println("\nOpen issues by priority:")
		for priority, count := range breakdown {
			fmt.Printf("  P%d: %d\n", priority, count)
		}
	}

	// Calculate totals
	total := 0
	for _, count := range stats {
		total += count
	}
	fmt.Printf("\nTotal dependencies: %d\n", total)
}

func printBlockersText(issueID string, blockers []beads.Issue, transitive bool) {
	if transitive {
		fmt.Printf("🔒 Transitive blockers for %s:\n\n", issueID)
	} else {
		fmt.Printf("🔒 Direct blockers for %s:\n\n", issueID)
	}

	if len(blockers) == 0 {
		fmt.Println("  ✓ No blockers found")
		return
	}

	for _, blocker := range blockers {
		status := blocker.Status
		switch status {
		case "done":
			status = "✓"
		case "open":
			status = "●"
		}
		fmt.Printf("  %s [%s] %s: %s\n", status, fmt.Sprintf("P%d", blocker.Priority), blocker.ID, blocker.Title)
	}
}
