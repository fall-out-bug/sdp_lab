// Package main provides the sdp-harness CLI for running SDP phase turns
// and managing session lifecycle through the agentloop Harness.
//
// Usage:
//
//	sdp-harness new --session=<id>
//	  Creates a new session DB at $SDP_DATA_DIR/<id>.db (default: $HOME/.sdp/<id>.db).
//
//	sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]
//	  Runs one phase turn for the given session, streaming events to stdout.
//	  Restores an existing session or fails if session does not exist.
//
// Environment:
//
//	SDP_DATA_DIR  Directory for session DB files. Defaults to $HOME/.sdp.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/agentloop"
	"github.com/fall-out-bug/sdp_lab/internal/agentloop/livegw"
	"github.com/fall-out-bug/sdp_lab/internal/workstream"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("no subcommand given")
	}

	switch args[0] {
	case "compile-lock":
		return cmdCompileLock(args[1:])
	case "new":
		return cmdNew(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "release":
		return cmdRelease(args[1:])
	case "events":
		return cmdEvents(args[1:])
	case "--help", "-h", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand: %q", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `sdp-harness — SDP Mini-Harness CLI

Subcommands:
  compile-lock  --project-root=<path>                     Compile .sdp/workgraph.lock.json
  new           --session=<id> --project-root=<path> --feature=<FXXX> --ws=<id>
                                                          Create a new session bound to one executable leaf
  run           --session=<id> --prompt="<text>"         Run one phase turn for a bound leaf session
  release       --session=<id>                           Release the claim held by a bound session
  events        --session=<id>                           Print persisted structured events for one session

Environment:
  SDP_DATA_DIR         Directory for session DB files (default: $HOME/.sdp)
  SDP_HARNESS_BD_PATH  Optional path to bd executable for live dispatch`)
}

// dataDir returns the directory where session DB files are stored.
// Uses SDP_DATA_DIR env var; falls back to $HOME/.sdp.
func dataDir() (string, error) {
	if d := os.Getenv("SDP_DATA_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".sdp"), nil
}

// dbPath returns the path to the session DB file for the given sessionID.
func dbPath(sessionID string) (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", dir, err)
	}
	return filepath.Join(dir, sessionID+".db"), nil
}

func cmdCompileLock(args []string) error {
	fs := flag.NewFlagSet("compile-lock", flag.ContinueOnError)
	projectRoot := fs.String("project-root", ".", "Project root containing docs/workstreams")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := filepath.Abs(*projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}

	lock, report, err := workstream.CompileWorkgraphLock(root, workstream.DefaultCompileOptions())
	if err != nil {
		return err
	}
	if report.HasErrors() {
		return fmt.Errorf("workgraph compile failed: %s", summarizeIssues(report.Issues))
	}
	if err := workstream.WriteWorkgraphLock(root, lock); err != nil {
		return err
	}

	fmt.Printf("workgraph lock written at %s/.sdp/workgraph.lock.json (%d normalized features)\n", root, len(lock.Features))
	return nil
}

// cmdNew implements `sdp-harness new --session=<id> --project-root=<path> --feature=<FXXX> --ws=<id>`.
// It refuses to create a session unless the target is an executable leaf in a fresh workgraph lock.
func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	projectRoot := fs.String("project-root", ".", "Project root containing docs/workstreams (required)")
	featureID := fs.String("feature", "", "Feature ID (required)")
	wsID := fs.String("ws", "", "Leaf workstream ID (required)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}
	if *featureID == "" || *wsID == "" {
		fs.Usage()
		return fmt.Errorf("--feature and --ws are required")
	}

	root, err := filepath.Abs(*projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	observer := newDispatchEventObserver(*sessionID, store, os.Stderr)
	adapter := runtimeAdapterForRoot(root)
	lease, err := workstream.AcquireExecutionClaim(context.Background(), root, *featureID, *wsID, workstream.DefaultCompileOptions(), adapter, observer)
	if err != nil {
		return fmt.Errorf("acquire execution claim: %w\n(hint: run 'sdp-harness compile-lock --project-root=%s' after updating normalized workstreams)", err, root)
	}

	_, err = agentloop.NewBoundSession(*sessionID, agentloop.SessionBinding{
		FeatureID:      *featureID,
		WSID:           *wsID,
		ProjectRoot:    root,
		ClaimedIssueID: lease.ClaimedIssueID,
	}, store)
	if err != nil {
		if releaseErr := workstream.ReleaseExecutionClaim(context.Background(), adapter, lease, observer); releaseErr != nil {
			return fmt.Errorf("create session: %w (claim release failed: %v)", err, releaseErr)
		}
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("session %q created at %s for %s/%s (claimed %s)\n", *sessionID, path, *featureID, *wsID, lease.ClaimedIssueID)
	return nil
}

// cmdRun implements `sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]`.
// Restores a bound session, verifies the lock is still fresh for the same leaf, and runs one phase turn.
func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	prompt := fs.String("prompt", "", "User prompt for this phase turn (required)")
	token := fs.String("token", "", "Owner token (optional; required if session has ownerToken set)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}
	if *prompt == "" {
		fs.Usage()
		return fmt.Errorf("--prompt is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	session, err := agentloop.RecoverSession(*sessionID, store)
	if err != nil {
		return fmt.Errorf("recover session %q: %w\n(hint: use 'sdp-harness new --session=%s --project-root=<root> --feature=<FXXX> --ws=<id>' to create it)", *sessionID, err, *sessionID)
	}
	observer := newDispatchEventObserver(*sessionID, store, os.Stderr)
	if session.ProjectRoot == "" || session.FeatureID == "" || session.WSID == "" {
		return fmt.Errorf("session %q is not bound to a leaf workstream; recreate it with --project-root, --feature, and --ws", *sessionID)
	}
	if len(session.History) > 0 && session.Phase == "" {
		return agentloop.ErrHarnessTerminated
	}
	if session.ClaimedIssueID == "" {
		return fmt.Errorf("session %q has no claimed issue; create a fresh bound session", *sessionID)
	}
	if _, err := workstream.RevalidateExecutionClaim(context.Background(), session.ProjectRoot, session.FeatureID, session.WSID, session.ClaimedIssueID, workstream.DefaultCompileOptions(), runtimeAdapterForRoot(session.ProjectRoot), observer); err != nil {
		return fmt.Errorf("revalidate execution claim: %w", err)
	}

	// Build a minimal router with no real tools (MVP placeholder).
	// LiveGateway connects to OpenRouter; requires OPENROUTER_API_KEY.
	registry := agentloop.NewToolRegistry(nil)
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	gw, err := livegw.New(apiKey, "")
	if err != nil {
		return fmt.Errorf("create LiveGateway: %w\n(hint: set OPENROUTER_API_KEY env var)", err)
	}
	router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gw, nil)
	gate := agentloop.NewGateEngine(nil, 0) // 0 → 5s default

	// Try to restore an existing session; if not found, error (use `new` first).
	h, err := agentloop.RestoreHarness(*sessionID, *token, store, router, gate, nil)
	if err != nil {
		return fmt.Errorf("restore session %q: %w\n(hint: use 'sdp-harness new --session=%s' to create it)", *sessionID, err, *sessionID)
	}

	// Run the phase turn with a background context (no timeout for MVP CLI).
	ctx := context.Background()
	if err := h.RunPhase(ctx, *prompt, *token); err != nil {
		return fmt.Errorf("run phase: %w", err)
	}

	fmt.Printf("phase turn complete for session %q (%s/%s, claimed %s)\n", *sessionID, session.FeatureID, session.WSID, session.ClaimedIssueID)
	return nil
}

func cmdRelease(args []string) error {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}
	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	session, err := agentloop.RecoverSession(*sessionID, store)
	if err != nil {
		return fmt.Errorf("recover session %q: %w", *sessionID, err)
	}
	observer := newDispatchEventObserver(*sessionID, store, os.Stderr)
	if session.ProjectRoot == "" || session.FeatureID == "" || session.WSID == "" {
		return fmt.Errorf("session %q is not bound to a leaf workstream", *sessionID)
	}
	if session.ClaimedIssueID == "" {
		return fmt.Errorf("session %q has no claimed issue to release", *sessionID)
	}
	lease := workstream.DispatchLease{
		Target: workstream.ExecutionTarget{
			Feature:    workstream.FeatureLock{FeatureID: session.FeatureID},
			Workstream: workstream.WorkstreamLock{WSID: session.WSID},
		},
		ClaimedIssueID: session.ClaimedIssueID,
	}
	if err := workstream.ReleaseExecutionClaim(context.Background(), runtimeAdapterForRoot(session.ProjectRoot), lease, observer); err != nil {
		return fmt.Errorf("release execution claim: %w", err)
	}
	session.ClaimedIssueID = ""
	if err := store.Persist(session); err != nil {
		return fmt.Errorf("persist released session: %w", err)
	}

	fmt.Printf("released claim for session %q\n", *sessionID)
	return nil
}

func cmdEvents(args []string) error {
	fs := flag.NewFlagSet("events", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}
	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	events, err := store.LoadEvents(*sessionID)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	payload, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}
	fmt.Printf("%s\n", payload)
	return nil
}

func runtimeAdapterForRoot(projectRoot string) *workstream.ShellBeadsRuntimeAdapter {
	adapter := workstream.NewShellBeadsRuntimeAdapter(projectRoot)
	if bdPath := strings.TrimSpace(os.Getenv("SDP_HARNESS_BD_PATH")); bdPath != "" {
		adapter.BDPath = bdPath
	}
	return adapter
}

func summarizeIssues(issues []workstream.ValidationIssue) string {
	if len(issues) == 0 {
		return "no issues"
	}
	limit := 3
	if len(issues) < limit {
		limit = len(issues)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, issues[i].Message)
	}
	if len(issues) > limit {
		parts = append(parts, fmt.Sprintf("%d more", len(issues)-limit))
	}
	return strings.Join(parts, "; ")
}

type dispatchEventObserver struct {
	sessionID string
	store     agentloop.SessionStore
	stderr    *os.File
	counters  map[string]int
}

func newDispatchEventObserver(sessionID string, store agentloop.SessionStore, stderr *os.File) *dispatchEventObserver {
	return &dispatchEventObserver{
		sessionID: sessionID,
		store:     store,
		stderr:    stderr,
		counters:  map[string]int{},
	}
}

func (o *dispatchEventObserver) IncrementCounter(name string) {
	o.counters[name]++
	_ = o.store.PersistEvent(o.sessionID, agentloop.Event{
		Type:  "dispatch_metric",
		Code:  name,
		Count: 1,
		Fields: map[string]string{
			"total": strconv.Itoa(o.counters[name]),
		},
	})
}

func (o *dispatchEventObserver) RecordDiagnostic(diag workstream.DispatchDiagnostic) {
	fields := map[string]string{}
	for k, v := range diag.Fields {
		fields[k] = v
	}
	if diag.FeatureID != "" {
		fields["feature_id"] = diag.FeatureID
	}
	if diag.LeafWSID != "" {
		fields["leaf_ws_id"] = diag.LeafWSID
	}
	if diag.IssueID != "" {
		fields["issue_id"] = diag.IssueID
	}
	if diag.Reason != "" {
		fields["reason"] = diag.Reason
	}
	if len(diag.Conflicts) > 0 {
		fields["conflicts"] = strings.Join(diag.Conflicts, ",")
	}
	_ = o.store.PersistEvent(o.sessionID, agentloop.Event{
		Type:   "dispatch_diagnostic",
		Code:   diag.Code,
		Fields: fields,
	})

	if o.stderr == nil || diag.Code == "dispatch_success" || diag.Code == "dispatch_claim_released" {
		return
	}
	payload := map[string]any{
		"type":       "dispatch_diagnostic",
		"code":       diag.Code,
		"feature_id": diag.FeatureID,
		"leaf_ws_id": diag.LeafWSID,
	}
	if diag.IssueID != "" {
		payload["issue_id"] = diag.IssueID
	}
	if diag.Reason != "" {
		payload["reason"] = diag.Reason
	}
	if len(diag.Conflicts) > 0 {
		payload["conflicts"] = diag.Conflicts
	}
	if len(diag.Fields) > 0 {
		payload["fields"] = diag.Fields
	}
	if raw, err := json.Marshal(payload); err == nil {
		fmt.Fprintln(o.stderr, string(raw))
	}
}
