package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sdp_dev/internal/metrics"
	"time"
)

func runMetrics(args []string) {
	fs := flag.NewFlagSet("metrics", flag.ExitOnError)
	format := fs.String("format", "json", "output format: json, text, markdown")
	output := fs.String("output", "", "write report to this directory")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp metrics [--format json|text|markdown] [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	switch *format {
	case "json", "text", "markdown":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json, text, or markdown)\n", *format)
		os.Exit(2)
	}

	start := time.Now()
	data, err := metrics.Collect(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: metrics collection failed: %v\n", err)
		os.Exit(1)
	}
	data = metrics.Filter(data)

	report := metrics.MetricsReport{
		Version:         "1.0.0",
		GeneratedAt:     start.UTC(),
		RepoPath:        repoPath,
		DurationMs:      time.Since(start).Milliseconds(),
		CommitsAnalyzed: len(data.Commits),
	}

	if len(data.Commits) > 0 {
		report.Period = metrics.TimePeriod{
			From: data.Commits[len(data.Commits)-1].Date,
			To:   data.Commits[0].Date,
		}
	}

	// Run all 7 analyzers
	report.Hygiene = metrics.AnalyzeHygiene(data)
	report.Waste = metrics.AnalyzeWaste(data)
	report.GitFlow = metrics.AnalyzeGitFlow(data)
	report.ReleaseQuality = metrics.AnalyzeReleaseQuality(data)
	report.Stabilization = metrics.AnalyzeStabilization(data)
	report.KnowledgeRisk = metrics.AnalyzeKnowledge(data)
	report.Decay = metrics.AnalyzeDecay(data)

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(report, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		renderText(&report)
	case "markdown":
		renderMarkdown(&report)
	}

	if *output != "" {
		if err := os.MkdirAll(*output, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create output dir: %v\n", err)
		} else {
			b, _ := json.MarshalIndent(report, "", "  ")
			path := *output + "/report.json"
			if err := os.WriteFile(path, b, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not write report: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "artifact: %s\n", path)
			}
		}
	}
}

func renderText(r *metrics.MetricsReport) {
	fmt.Fprintf(os.Stdout, " %s — %d commits analyzed\n", r.RepoPath, r.CommitsAnalyzed)
	fmt.Fprintf(os.Stdout, " Period: %s to %s\n", r.Period.From.Format("2006-01-02"), r.Period.To.Format("2006-01-02"))
	fmt.Fprintf(os.Stdout, " Duration: %dms\n\n", r.DurationMs)

	if r.Hygiene != nil {
		fmt.Fprintf(os.Stdout, " [Hygiene]\n")
		fmt.Fprintf(os.Stdout, "  Ticket linked:     %.0f%%  %s\n", r.Hygiene.TicketLinkedRatio*100, metrics.RateTicketLinkedRatio(r.Hygiene.TicketLinkedRatio))
		fmt.Fprintf(os.Stdout, "  Conventional:      %.0f%%  %s\n", r.Hygiene.ConventionalCommitsRatio*100, metrics.RateTicketLinkedRatio(r.Hygiene.ConventionalCommitsRatio))
		fmt.Fprintf(os.Stdout, "  Fix/Feature ratio: %.2f   %s\n", r.Hygiene.FixToFeatureRatio, metrics.RateFixToFeature(r.Hygiene.FixToFeatureRatio))
	}
	if r.Waste != nil {
		fmt.Fprintf(os.Stdout, " [Waste]\n")
		fmt.Fprintf(os.Stdout, "  Churn ratio:       %.2f   %s\n", r.Waste.ChurnRatio, metrics.RateChurnRatio(r.Waste.ChurnRatio))
		fmt.Fprintf(os.Stdout, "  Revert rate:       %.3f  %s\n", r.Waste.RevertRate, metrics.RateRevertRate(r.Waste.RevertRate))
		fmt.Fprintf(os.Stdout, "  Abandoned branches: %d\n", r.Waste.AbandonedBranches)
	}
	if r.GitFlow != nil {
		fmt.Fprintf(os.Stdout, " [Git Flow]\n")
		fmt.Fprintf(os.Stdout, "  Model:             %s (confidence %.0f%%)\n", r.GitFlow.DetectedModel, r.GitFlow.Confidence*100)
		fmt.Fprintf(os.Stdout, "  Merge frequency:   %.1f/week\n", r.GitFlow.MergeFrequencyPerWeek)
	}
	if r.ReleaseQuality != nil {
		fmt.Fprintf(os.Stdout, " [Release Quality]\n")
		fmt.Fprintf(os.Stdout, "  Releases analyzed: %d\n", r.ReleaseQuality.ReleasesAnalyzed)
		fmt.Fprintf(os.Stdout, "  Avg TTFH:          %.1fh\n", r.ReleaseQuality.AvgTimeToFirstHotfixH)
	}
	if r.Stabilization != nil {
		fmt.Fprintf(os.Stdout, " [Stabilization]\n")
		fmt.Fprintf(os.Stdout, "  Avg patches:       %.1f\n", r.Stabilization.AvgPatchesToStable)
		fmt.Fprintf(os.Stdout, "  Trend:             %s\n", r.Stabilization.Trend)
	}
	if r.KnowledgeRisk != nil {
		fmt.Fprintf(os.Stdout, " [Knowledge Risk]\n")
		fmt.Fprintf(os.Stdout, "  Bus factor:        %d  %s\n", r.KnowledgeRisk.OverallBusFactor, metrics.RateBusFactor(r.KnowledgeRisk.OverallBusFactor))
		fmt.Fprintf(os.Stdout, "  Gini coefficient:  %.2f\n", r.KnowledgeRisk.GiniCoefficient)
		fmt.Fprintf(os.Stdout, "  Former contributors: %.0f%%\n", r.KnowledgeRisk.FormerContributorRatio*100)
	}
	if r.Decay != nil {
		fmt.Fprintf(os.Stdout, " [Code Decay]\n")
		fmt.Fprintf(os.Stdout, "  Shotgun ratio:     %.3f  %s\n", r.Decay.ShotgunSurgeryRatio, metrics.RateShotgunRatio(r.Decay.ShotgunSurgeryRatio))
		fmt.Fprintf(os.Stdout, "  Fix recurrence:    %d files\n", len(r.Decay.FixRecurrence))
		fmt.Fprintf(os.Stdout, "  Monotonic growth:  %d files\n", len(r.Decay.MonotonicGrowthFiles))
	}
}

func renderMarkdown(r *metrics.MetricsReport) {
	fmt.Fprintf(os.Stdout, "# SDP Metrics Report\n\n")
	fmt.Fprintf(os.Stdout, "**Repo:** %s | **Commits:** %d | **Period:** %s — %s\n\n",
		r.RepoPath, r.CommitsAnalyzed,
		r.Period.From.Format("2006-01-02"), r.Period.To.Format("2006-01-02"))

	fmt.Fprintf(os.Stdout, "| Category | Metric | Value | Rating |\n")
	fmt.Fprintf(os.Stdout, "|----------|--------|-------|--------|\n")

	if r.Hygiene != nil {
		fmt.Fprintf(os.Stdout, "| Hygiene | Ticket linked | %.0f%% | %s |\n", r.Hygiene.TicketLinkedRatio*100, metrics.RateTicketLinkedRatio(r.Hygiene.TicketLinkedRatio))
		fmt.Fprintf(os.Stdout, "| Hygiene | Conventional commits | %.0f%% | %s |\n", r.Hygiene.ConventionalCommitsRatio*100, metrics.RateTicketLinkedRatio(r.Hygiene.ConventionalCommitsRatio))
		fmt.Fprintf(os.Stdout, "| Hygiene | Fix/Feature ratio | %.2f | %s |\n", r.Hygiene.FixToFeatureRatio, metrics.RateFixToFeature(r.Hygiene.FixToFeatureRatio))
	}
	if r.Waste != nil {
		fmt.Fprintf(os.Stdout, "| Waste | Churn ratio | %.2f | %s |\n", r.Waste.ChurnRatio, metrics.RateChurnRatio(r.Waste.ChurnRatio))
		fmt.Fprintf(os.Stdout, "| Waste | Revert rate | %.3f | %s |\n", r.Waste.RevertRate, metrics.RateRevertRate(r.Waste.RevertRate))
	}
	if r.GitFlow != nil {
		fmt.Fprintf(os.Stdout, "| Git Flow | Model | %s | %.0f%% confidence |\n", r.GitFlow.DetectedModel, r.GitFlow.Confidence*100)
	}
	if r.ReleaseQuality != nil {
		fmt.Fprintf(os.Stdout, "| Release | Avg TTFH | %.1fh | - |\n", r.ReleaseQuality.AvgTimeToFirstHotfixH)
	}
	if r.Stabilization != nil {
		fmt.Fprintf(os.Stdout, "| Stabilization | Avg patches | %.1f | Trend: %s |\n", r.Stabilization.AvgPatchesToStable, r.Stabilization.Trend)
	}
	if r.KnowledgeRisk != nil {
		fmt.Fprintf(os.Stdout, "| Knowledge | Bus factor | %d | %s |\n", r.KnowledgeRisk.OverallBusFactor, metrics.RateBusFactor(r.KnowledgeRisk.OverallBusFactor))
		fmt.Fprintf(os.Stdout, "| Knowledge | Gini | %.2f | - |\n", r.KnowledgeRisk.GiniCoefficient)
	}
	if r.Decay != nil {
		fmt.Fprintf(os.Stdout, "| Decay | Shotgun ratio | %.3f | %s |\n", r.Decay.ShotgunSurgeryRatio, metrics.RateShotgunRatio(r.Decay.ShotgunSurgeryRatio))
		fmt.Fprintf(os.Stdout, "| Decay | Fix recurrence | %d files | - |\n", len(r.Decay.FixRecurrence))
	}
}
