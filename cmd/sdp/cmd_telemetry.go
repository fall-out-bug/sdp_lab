package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/trace"
	"sdp_dev/internal/trace/bead"
	"sdp_dev/internal/trace/client"
	"sdp_dev/internal/trace/consent"
	traceDaemon "sdp_dev/internal/trace/daemon"
)

func runTelemetry(args []string) {
	if len(args) == 0 {
		runTelemetryHelp()
		os.Exit(2)
	}

	switch args[0] {
	case "init":
		runTelemetryInit(args[1:])
	case "span-start":
		runTelemetrySpanStart(args[1:])
	case "span-end":
		runTelemetrySpanEnd(args[1:])
	case "event":
		runTelemetryEvent(args[1:])
	case "daemon":
		runTelemetryDaemon(args[1:])
	case "shutdown":
		runTelemetryShutdown(args[1:])
	case "consent":
		runTelemetryConsent(args[1:])
	default:
		runTelemetryHelp()
		os.Exit(2)
	}
}

func runTelemetryHelp() {
	fmt.Fprintln(os.Stderr, "usage: sdp telemetry <command> [flags]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  init --feature <bead_id>          Initialize trace session")
	fmt.Fprintln(os.Stderr, "  span-start --kind <kind>          Start a new span")
	fmt.Fprintln(os.Stderr, "  span-end --span-id <id>           End a span")
	fmt.Fprintln(os.Stderr, "  event --span-id <id> --name <nm>  Add event to span")
	fmt.Fprintln(os.Stderr, "  daemon                            Start trace daemon")
	fmt.Fprintln(os.Stderr, "  shutdown                          Shutdown trace daemon")
	fmt.Fprintln(os.Stderr, "  consent                           Show/set consent level")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Span kinds: tool, agent, phase, bead")
}

func runTelemetryInit(args []string) {
	var featureID string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--feature", "-f":
			if i+1 < len(args) {
				featureID = args[i+1]
				i++
			}
		}
	}

	if featureID == "" {
		fmt.Fprintln(os.Stderr, "error: --feature is required")
		fmt.Fprintln(os.Stderr, "usage: sdp telemetry init --feature <bead_id>")
		os.Exit(2)
	}

	// Validate bead ID
	if err := bead.ValidateBeadID(featureID); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid feature ID: %v\n", err)
		os.Exit(2)
	}

	// Set current feature
	projectRoot := bead.FindProjectRoot(".")
	resolver := bead.NewResolver(projectRoot)

	if err := resolver.SetCurrentFeatureID(featureID); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to set current feature: %v\n", err)
		os.Exit(2)
	}

	// Get session ID
	sessionID, err := bead.GetCurrentSessionID(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get session ID: %v\n", err)
		os.Exit(2)
	}

	// Generate trace ID
	traceGen := bead.NewTraceIDGenerator()
	traceID := traceGen.Generate()

	// Write session metadata
	tracesDir := filepath.Join(projectRoot, ".sdp", "traces")
	if err := os.MkdirAll(tracesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create traces directory: %v\n", err)
		os.Exit(2)
	}

	metadata := &trace.SessionMetadata{
		SessionID:  sessionID,
		TraceID:    traceID,
		StartTime:  time.Now().Format(time.RFC3339),
		EpicBeadID: featureID,
		Harness:    getHarness(),
	}

	metadataPath := filepath.Join(tracesDir, "current.env")
	metadataData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to marshal metadata: %v\n", err)
		os.Exit(2)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to write metadata: %v\n", err)
		os.Exit(2)
	}

	// Output TRACEPARENT for shell consumption
	fmt.Printf("export TRACEPARENT=00-%s-0000000000000001-01\n", traceID)
	fmt.Printf("export SDP_SESSION_ID=%s\n", sessionID)
	fmt.Printf("export SDP_EPIC_BEAD_ID=%s\n", featureID)
	fmt.Fprintf(os.Stderr, "Trace initialized: trace_id=%s session_id=%s feature=%s\n", traceID, sessionID, featureID)
}

func runTelemetrySpanStart(args []string) {
	var kind, name, toolCallID, phase, verdict string
	var cycle int

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--kind", "-k":
			if i+1 < len(args) {
				kind = args[i+1]
				i++
			}
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--tool-call-id":
			if i+1 < len(args) {
				toolCallID = args[i+1]
				i++
			}
		case "--phase":
			if i+1 < len(args) {
				phase = args[i+1]
				i++
			}
		case "--cycle":
			if i+1 < len(args) {
				if _, err := fmt.Sscanf(args[i+1], "%d", &cycle); err != nil {
					fmt.Fprintf(os.Stderr, "error: invalid --cycle %q\n", args[i+1])
					os.Exit(2)
				}
				i++
			}
		case "--verdict":
			if i+1 < len(args) {
				verdict = args[i+1]
				i++
			}
		}
	}

	if kind == "" || name == "" {
		fmt.Fprintln(os.Stderr, "error: --kind and --name are required")
		fmt.Fprintln(os.Stderr, "usage: sdp telemetry span-start --kind <kind> --name <name>")
		os.Exit(2)
	}

	// Map kind to SpanKind
	var spanKind trace.SpanKind
	switch kind {
	case "tool":
		spanKind = trace.SpanKindExecuteTool
	case "agent":
		spanKind = trace.SpanKindInvokeAgent
	case "phase":
		spanKind = trace.SpanKindDeliveryLoopPhase
	case "bead":
		spanKind = trace.SpanKindSDPBeadEvent
	default:
		fmt.Fprintf(os.Stderr, "error: invalid span kind: %s\n", kind)
		os.Exit(2)
	}

	// Build attributes
	attrs := make(map[string]string)
	attrs["gen_ai.tool.name"] = name

	if phase != "" {
		attrs["sdp.phase.name"] = phase
	}
	if cycle > 0 {
		attrs["sdp.phase.cycle_number"] = fmt.Sprintf("%d", cycle)
	}
	if verdict != "" {
		attrs["sdp.review.verdict"] = verdict
	}

	// Get current feature
	projectRoot := bead.FindProjectRoot(".")
	resolver := bead.NewResolver(projectRoot)
	featureID, err := resolver.GetCurrentFeatureID()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get current feature: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'sdp telemetry init --feature <bead_id>' first")
		os.Exit(2)
	}
	attrs["sdp.epic.bead_id"] = featureID

	// Get session ID
	sessionID, err := bead.GetCurrentSessionID(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get session ID: %v\n", err)
		os.Exit(2)
	}
	attrs["sdp.session.id"] = sessionID

	// Get harness
	harness := getHarness()
	attrs["sdp.harness"] = harness

	// Send to daemon
	socketPath := getSocketPath(sessionID)
	cli := client.NewClient(socketPath)
	traceID, spanID, err := cli.Span(nil, spanKind, toolCallID, attrs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to start span: %v\n", err)
		fmt.Fprintln(os.Stderr, "Is the trace daemon running? Start it with: sdp telemetry daemon")
		os.Exit(2)
	}

	// Output for shell consumption
	fmt.Printf("export SDP_TRACE_ID=%s\n", traceID)
	fmt.Printf("export SDP_SPAN_ID=%s\n", spanID)
}

func runTelemetrySpanEnd(args []string) {
	var spanID, toolCallID, errorMsg string
	var exitCode int
	var durationMs int64

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--span-id":
			if i+1 < len(args) {
				spanID = args[i+1]
				i++
			}
		case "--tool-call-id":
			if i+1 < len(args) {
				toolCallID = args[i+1]
				i++
			}
		case "--exit-code":
			if i+1 < len(args) {
				if _, err := fmt.Sscanf(args[i+1], "%d", &exitCode); err != nil {
					fmt.Fprintf(os.Stderr, "error: invalid --exit-code %q\n", args[i+1])
					os.Exit(2)
				}
				i++
			}
		case "--duration-ms":
			if i+1 < len(args) {
				if _, err := fmt.Sscanf(args[i+1], "%d", &durationMs); err != nil {
					fmt.Fprintf(os.Stderr, "error: invalid --duration-ms %q\n", args[i+1])
					os.Exit(2)
				}
				i++
			}
		case "--error":
			if i+1 < len(args) {
				errorMsg = args[i+1]
				i++
			}
		}
	}

	if spanID == "" {
		fmt.Fprintln(os.Stderr, "error: --span-id is required")
		fmt.Fprintln(os.Stderr, "usage: sdp telemetry span-end --span-id <id>")
		os.Exit(2)
	}

	// Build attributes
	attrs := make(map[string]string)
	attrs["sdp.tool.exit_code"] = fmt.Sprintf("%d", exitCode)
	if durationMs > 0 {
		attrs["sdp.tool.duration_ms"] = fmt.Sprintf("%d", durationMs)
	}
	if errorMsg != "" {
		attrs["sdp.tool.error"] = errorMsg
	}

	// Get session ID
	projectRoot := bead.FindProjectRoot(".")
	sessionID, err := bead.GetCurrentSessionID(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get session ID: %v\n", err)
		os.Exit(2)
	}

	// Send to daemon
	socketPath := getSocketPath(sessionID)
	cli := client.NewClient(socketPath)
	if err := cli.End(nil, spanID, toolCallID, attrs); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to end span: %v\n", err)
		os.Exit(2)
	}
}

func runTelemetryEvent(args []string) {
	fmt.Fprintln(os.Stderr, "error: event command not yet implemented")
	os.Exit(2)
}

func runTelemetryDaemon(args []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--foreground" {
			// For MVP, always run in foreground
		}
	}

	// Get session ID
	projectRoot := bead.FindProjectRoot(".")
	sessionID, err := bead.GetCurrentSessionID(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get session ID: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'sdp telemetry init' first")
		os.Exit(2)
	}

	// Build socket path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get home directory: %v\n", err)
		os.Exit(2)
	}
	socketPath := fmt.Sprintf("%s/.sdp/sockets/trace-%s.sock", homeDir, sessionID)

	// Build traces directory
	tracesDir := filepath.Join(projectRoot, ".sdp", "traces")

	// Load schema (for MVP, use empty schema)
	schema := &trace.Schema{}

	// Get consent level
	consentLevel := consent.GetConsentLevelFromFileOrEnv("")

	// Create daemon config
	config := &traceDaemon.Config{
		SocketPath:   socketPath,
		TracesDir:    tracesDir,
		SessionID:    sessionID,
		Schema:       schema,
		ConsentLevel: consentLevel,
	}

	daemon := traceDaemon.NewDaemon(config)

	fmt.Printf("Starting trace daemon...\n")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Socket: %s\n", socketPath)
	fmt.Printf("Traces: %s\n", tracesDir)
	fmt.Printf("Consent level: %s\n", consentLevel)

	// Start daemon
	if err := daemon.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to start daemon: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("Trace daemon started\n")

	// Run until interrupted
	fmt.Printf("Press Ctrl+C to stop\n")
	<-make(chan struct{})

	// Shutdown
	if err := daemon.Shutdown(); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to stop daemon: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("Trace daemon stopped\n")
}

func runTelemetryShutdown(args []string) {
	// Get session ID
	projectRoot := bead.FindProjectRoot(".")
	sessionID, err := bead.GetCurrentSessionID(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get session ID: %v\n", err)
		os.Exit(2)
	}

	socketPath := getSocketPath(sessionID)
	cli := client.NewClient(socketPath)
	if err := cli.Shutdown(nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to shutdown daemon: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("Trace daemon shutdown requested\n")
}

func runTelemetryConsent(args []string) {
	if len(args) == 0 {
		// Show current consent level
		level := consent.GetConsentLevelFromFileOrEnv("")
		fmt.Printf("Current consent level: %s\n", level)
		fmt.Println()
		fmt.Println(consent.FormatConsentBanner())
		return
	}

	// Set consent level
	newLevel := args[0]
	switch newLevel {
	case "metadata", "findings", "content":
		os.Setenv("SDP_TRACE_CONSENT", newLevel)
		fmt.Printf("Consent level set to: %s\n", newLevel)
		fmt.Printf("To make permanent, add to ~/.bashrc:\n")
		fmt.Printf("  export SDP_TRACE_CONSENT=%s\n", newLevel)
	default:
		fmt.Fprintf(os.Stderr, "error: invalid consent level: %s\n", newLevel)
		fmt.Fprintf(os.Stderr, "Valid levels: metadata, findings, content\n")
		os.Exit(2)
	}
}

func getHarness() string {
	harness := os.Getenv("SDP_HARNESS")
	if harness == "" {
		harness = "claude-code"
	}
	return harness
}

func getSocketPath(sessionID string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.sdp/sockets/trace-%s.sock", homeDir, sessionID)
}
